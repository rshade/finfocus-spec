# Implementation Plan: Scorer Advertised Limits

**Branch**: `591-scorer-advertised-limits` | **Spec**: [spec.md](spec.md)

## Summary

Publish a scorer's batch limit and signals in `GetPluginInfo` metadata and mark the oversize-batch
error with a `google.rpc.ErrorInfo` detail. No proto field changes; only comments in `scoring.proto`.

## Technical Context

Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript SDK. `google.golang.org/genproto/googleapis/rpc`
(`errdetails`) is already in `go.sum` as an indirect dependency and becomes a direct one. Stateless.

## Constitution Check

- Proto first: no wire change; the proto comment documents the keys and the detail. `make generate`
  must leave `sdk/` free of semantic diffs.
- SDK parity: Go (`pluginsdk`, `testing`) and TypeScript client both updated.
- Test first: tests precede each implementation task.
- Additive: `buf breaking` against `main`.
- Capability declaration (XII): the keys are metadata, not capabilities; discovery is unchanged and
  `WithCapabilities` still works alongside `WithScorerLimits`.
- Performance (VIII): `ParseScorerLimits` and `IsBatchTooLarge` run at startup and on error paths, not per
  record; benchmarks record their cost (T010). No allocation goal is claimed for them.
- Headers and docs (XI, XIV): new Go files carry the Apache header; exported symbols have godoc; docs and
  README change in the same PR.

## Design

- `sdk/go/testing/scorer_limits.go` (new, because `testing` cannot import `pluginsdk`): key constants,
  `FormatScorerLimits`, `ParseScorerLimits`, `BatchTooLargeReason`, `IsBatchTooLarge`.
- `scoring.go`: the oversize failure returns a `batchTooLargeError` whose `GRPCStatus()` is
  `InvalidArgument` plus an `ErrorInfo` detail. It still unwraps to `ErrInvalidScoreRequest`.
- `pluginsdk/scorer.go`: `WithScorerLimits`, `ParseScorerLimits`, `IsBatchTooLarge` delegating.
  `plugin_info.go`: `PluginInfo.Validate` rejects malformed scorer metadata.
- `connect_errors.go`: copy status details onto the `connect.Error`.
- `scorer_mock.go`: `AdvertisedScorerMetadata()`. `scorer_conformance.go`: `advertised_limits`
  scenario (optional interface) and tightened `oversize_batch`.
- TypeScript: `parseScorerLimits`, `isBatchTooLarge` in the scorer client module, exported from index.
- Docs: `scoring.proto` comments, `docs/recommendation-scoring.md`, `sdk/go/testing/README.md`.

## Risks

- Issue 580 and open PR 600 touch the same files; keep hunks small, rebase before merge.
- Adding a direct dependency edits `go.mod`; run `go mod tidy`.
