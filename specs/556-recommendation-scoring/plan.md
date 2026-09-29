# Implementation Plan: Recommendation Scoring

**Branch**: `556-recommendation-scoring` | **Date**: 2026-09-29 | **Spec**: [spec.md](spec.md)

## Summary

Add `RecommendationScorerService` in `scoring.proto` with capability 18. Follow the allocator
pattern: an optional provider interface, unexported adapters, `optionalServices` wiring, validators
in `sdk/go/testing` that `pluginsdk` delegates to, a mock, and a conformance runner.

## Decisions

- Validators live in `sdk/go/testing/scoring.go` because `testing` cannot import `pluginsdk`.
  `pluginsdk/scorer.go` delegates so hosts use one rule from production code.
- Request errors reuse `newInvalidArgument`, so both transports report `InvalidArgument` without an
  `rpc error:` prefix. Response errors wrap `ErrInvalidScoreResponse`.
- The request validator takes `maxBatchSize`; zero means no limit. The response validator reads
  `max_batch_size` from the response.
- `MockRecommendationScorer` uses fixed rules and is its own type, so `MockPlugin` capabilities do
  not change.
- `RunScorerConformance` is model-agnostic: it checks structure, never score values. It discovers
  `max_batch_size` and `supported_signals` with a one-recommendation probe.
- Adding the capability bumps `maxValidCapability`, `optionalCapabilities` (11),
  `legacyCapabilityNames`, and the `IsValidCapability` bounds test.
- `Serve` warns once for a scorer-only plugin that relies on inferred capabilities.

## Files

- `proto/finfocus/v1/scoring.proto`, `proto/finfocus/v1/enums.proto`, generated Go, Connect, and
  TypeScript bindings
- `sdk/go/testing/scoring.go`, `scorer_mock.go`, `scorer_conformance.go`
- `sdk/go/pluginsdk/scorer.go`, `sdk.go`, `plugin_info.go`, `capability_compat.go`,
  `usage_source.go`
- `sdk/typescript/packages/client/src/clients/recommendation-scorer.ts`
- `docs/recommendation-scoring.md`
