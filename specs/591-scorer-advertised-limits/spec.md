# Feature Specification: Scorer Advertised Limits

**Feature Branch**: `591-scorer-advertised-limits`

**Created**: 2026-10-01

**Status**: Implemented

**Input**: GitHub issue 573, "Scorer batch limit and supported signals are only discoverable from a
response". Follows `556-recommendation-scoring`.

## Summary

A scorer's `max_batch_size` and `supported_signals` are fields of `ScoreRecommendationsResponse`, so a
host learns them only after a first call. That call can itself fail: the oversize-batch error shares
`INVALID_ARGUMENT` with empty requests, duplicate ids, and unsupported signals, so a host that halves
the batch and retries can hide a genuine input error. This feature lets a scorer publish both values
through `GetPluginInfo` before the first scoring call, and lets a host tell "batch too large" apart from
the other `INVALID_ARGUMENT` causes. The response fields stay authoritative for each call. The change is
additive: no proto field, message, enum value, or RPC is added or renumbered.

## Clarifications

- Q: Advertise through new proto fields or `GetPluginInfo` metadata? -> A: Metadata keys. The issue
  names them, and `GetPluginInfoResponse.metadata` already exists, so `buf breaking` is untouched and
  hosts that predate this change ignore the keys.
- Q: A new gRPC code for "batch too large"? -> A: No. The code stays `INVALID_ARGUMENT` so existing
  hosts and scorers keep working. The error gains a `google.rpc.ErrorInfo` detail with reason
  `BATCH_TOO_LARGE` (documented in the proto comment). A new code would silently change behavior for
  every existing scorer and host.
- Q: Does an advertised value replace the response value? -> A: No. The response is authoritative per
  call. A host uses the advertised values to plan, and treats a mismatch as a scorer defect.
- Q: Is advertising required? -> A: No. A scorer that does not advertise keeps working; hosts fall back
  to the response fields.
- Q: Overlap with issue 580 (`ScorerInfo` scalars, per-item errors)? -> A: This feature adds no
  `ScorerInfo` field and no per-item error semantics; it touches only the batch-limit error and the
  discovery path.

## User Stories

- **US1, Advertise (P1)**: A scorer author calls one `PluginInfo` option with the batch limit and the
  signal list. `GetPluginInfo` then returns both before any scoring call, and a host-side helper parses
  them.
- **US2, Distinguish (P1)**: A host receiving the oversize-batch error can identify it from the error
  detail alone, over gRPC and Connect, without matching message text.
- **US3, Conformance (P2)**: `RunScorerConformance` fails a scorer whose advertised values differ from
  its response, and fails an oversize-batch error that lacks the `BATCH_TOO_LARGE` detail.
- **US4, Documentation (P2)**: `docs/recommendation-scoring.md`, the proto comments, and the TypeScript
  client explain the keys, the detail, and the fallback.

## Requirements

- **FR-001**: Two reserved metadata keys, `scorer_max_batch_size` (decimal integer, 1 through 2147483647; the
  TypeScript parser also accepts a leading plus sign) and `scorer_supported_signals` (comma-separated lowercase signal names
  with no spaces, for example `risk,priority`).
- **FR-002**: `pluginsdk.WithScorerLimits(maxBatchSize, signals...)` sets both keys. Invalid input
  (limit below 1, no signal, unspecified or repeated signal) is rejected by `PluginInfo.Validate`, not
  silently dropped.
- **FR-003**: `pluginsdk.ParseScorerLimits(metadata)` returns the limit and signals, or a nil result with
  a nil error when the plugin advertised neither key. Malformed input is an error wrapping
  `ErrInvalidScorerLimits`: one key without the other, a limit below 1 or not an integer, and an empty,
  unknown, non-lowercase, unspecified, or repeated signal name.
- **FR-004**: The oversize error returned by `ValidateScoreRecommendationsRequest` keeps code
  `INVALID_ARGUMENT`, wraps `ErrInvalidScoreRequest`, and carries an `ErrorInfo` detail (reason
  `BATCH_TOO_LARGE`, domain `finfocus.v1.RecommendationScorerService`). A helper `IsBatchTooLarge(err)`
  reports it. Other failures carry no such detail.
- **FR-005**: The Connect adapter preserves every status detail, including types not registered in the
  process, by passing the `*anypb.Any` through unchanged.
- **FR-006**: `MockRecommendationScorer` exposes its advertised metadata. `RunScorerConformance` adds an
  `advertised_limits` scenario that runs when the implementation serves `GetPluginInfo` (the served metadata is
  compared; a scorer that advertises neither key passes) or exposes advertised metadata, and a tightened
  `oversize_batch` scenario that requires the detail.
- **FR-007**: The proto comments, `docs/recommendation-scoring.md`, and the TypeScript client document
  and parse the keys and the detail.
- **FR-008**: Additive only; `buf breaking` passes.

## Edge Cases

- Neither key present: not an error; hosts fall back to the response fields.
- Only one key present, a limit of 0 or a non-integer, a repeated, unknown, unspecified, or uppercase
  signal name: `ParseScorerLimits` errors and `PluginInfo.Validate` rejects the plugin info.
- Advertised values differ from the response: the response wins; `advertised_limits` fails the scorer.
- An `INVALID_ARGUMENT` without the `BATCH_TOO_LARGE` detail (empty, duplicate id, unsupported signal)
  is not a batch-size error, and `IsBatchTooLarge` is false for it.
- A Connect caller sees the same detail as a gRPC caller.
- Metadata is advisory and unauthenticated: a host must not size security-relevant limits from it.

## Success Criteria

- A scorer built with the option reports its limit and signals through `GetPluginInfo` in a test over
  gRPC.
- A host separates an oversize error from a duplicate-id error using `IsBatchTooLarge` alone.
- A scorer whose advertised limit differs from its response fails `advertised_limits`.
- `make buf-lint`, `buf breaking`, `make test`, `make lint-go`, and the TypeScript tests pass.

## Assumptions

- Signal names in the metadata value are the `ScoreSignal` enum names without the `SCORE_SIGNAL_`
  prefix, lowercased.
- A plugin that sets `WithCapabilities` still advertises limits through the same option.

## Out of Scope

Distinct gRPC codes, new proto fields, per-item error changes, `ScorerInfo` changes (issue 580), and
calibration advertising.
