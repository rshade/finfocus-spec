# Feature Specification: Recommendation Scoring

**Feature Branch**: `556-recommendation-scoring`

**Created**: 2026-09-29

**Status**: Complete

**Input**: GitHub issue 556, `RecommendationScorerService.ScoreRecommendations` for optional
scorer plugins. The companion host work is `finfocus` issue 1569.

## Summary

Add an optional `RecommendationScorerService` with one RPC, `ScoreRecommendations`. The host sends
recommendations that cost-source plugins already returned and receives per-recommendation signals:
risk, likelihood the resource is in that state on purpose, worth acting now, priority, thin
evidence, and duplicate grouping. A scorer plugin owns any model, API key, and network access, so
the host keeps its "no external API clients in core" boundary. The contract is model-neutral.

## Clarifications

- Q: New service or a method on `CostSourceService`? → A: A new service in `scoring.proto`, the
  `AllocatorService` precedent. A scorer is not a cost source.
- Q: One `risk` signal or two? → A: One. It covers downtime, data loss, and financial lock-in.
- Q: Who groups duplicates? → A: The scorer, through `duplicate_group_id`.
- Q: Does the SDK cap `max_batch_size`? → A: No. The scorer chooses it. Validators only require at
  least 1 and that the request fits.
- Q: Which capability number? → A: 18. The issue text says 17, but
  `PLUGIN_CAPABILITY_INVOICE_DATA = 17` already exists, so 18 is the next free value. Legacy
  metadata is `supports_recommendation_scoring`.
- Q: Are dataset ownership and evaluation in scope? → A: No. The evaluation dataset stays an open
  question outside this repository.

## User Stories

- **US1, Discovery (P1)**: A plugin implementing `RecommendationScorerProvider` advertises
  `PLUGIN_CAPABILITY_RECOMMENDATION_SCORING`. A plugin without it is never registered for the
  service and gets `UNIMPLEMENTED`.
- **US2, Scoring (P1)**: A host sends a batch and gets index-aligned results, each either scores or
  a `ResourceError`, over gRPC and Connect with equal responses.
- **US3, Validation (P1)**: Scorers and hosts share validators for index alignment, unique ids,
  `[0, 1]` and `[0, 3]` ranges, `max_batch_size`, `ResourceError` codes other than OK, and
  duplicate groups. An empty request is invalid.
- **US4, Conformance (P2)**: `RunScorerConformance` rejects misaligned results, duplicate ids,
  out-of-range signals, oversize batches, and an OK `ResourceError` code.
- **US5, Documentation (P2)**: `docs/recommendation-scoring.md` states the trust rules and
  calibration semantics verbatim, plus data handling and non-normative threshold guidance.

## Requirements

- **FR-001**: `scoring.proto` defines the service, `ScoreSignal`, `ScoreCalibration`,
  `IdentifierMode`, and the request, response, result, scores, and `ScorerInfo` messages. No
  existing message changes.
- **FR-002**: `PLUGIN_CAPABILITY_RECOMMENDATION_SCORING = 18` in `enums.proto`.
- **FR-003**: `Serve` registers the service in gRPC and Connect modes and in the Connect health
  checker only when the plugin implements the provider.
- **FR-004**: Numeric signals are `optional double`. `risk`, `false_positive`, `worth_acting`, and
  `insufficient_evidence` are finite and in `[0, 1]`. `priority` is finite and in `[0, 3]`.
- **FR-005**: A signal may be set only if it is in `supported_signals` and, when the request named
  signals, in the request.
- **FR-006**: A non-empty `duplicate_group_id` is shared by at least two results.
- **FR-007**: The docs state the four normative trust rules.

## Success Criteria

- `buf lint` and `buf breaking` against `origin/main` pass.
- `go test -race ./...` passes, and the mock scorer round-trips a batch over gRPC and Connect.
- Each broken-scorer case fails its named conformance scenario.

## Out of Scope

Producing, editing, or applying recommendations, generating text, calibrated probabilities,
streaming, and persistence.
