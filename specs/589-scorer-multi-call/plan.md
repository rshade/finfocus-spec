# Implementation Plan: Multi-Call Scorer Identity and Per-Item Error Semantics

**Branch**: `589-scorer-multi-call` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/589-scorer-multi-call/spec.md`

## Summary

`ScorerInfo` gains `provider_request_ids` (5) and `models` (6), and the scalar `provider_request_id`
(4) is deprecated. The response validator rejects empty list entries, a primary model that differs from
`models[0]`, and `resource_type_unsupported` on per-item errors. Conformance inherits the rules through
the validator. The mock gains two options, and TypeScript and the docs, including the cache key, follow.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
buf v1.32.1; no new dependencies

**Storage**: N/A

**Testing**: `go test`, `RunScorerConformance` broken-scorer table, vitest + msw

**Target Platform**: Scorer plugins and hosts (Go, TypeScript)

**Project Type**: Protocol specification and SDK library

**Performance Goals**: No zero-allocation path touched

**Constraints**: Additive; do not touch request or response fields used by open PRs 602-604

**Scale/Scope**: Two fields plus one deprecation, three validator rules, two mock options, docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Proto and `make generate` first |
| II. Multi-provider consistency | PASS | Provider-neutral lists |
| III. Spec consumes, does not calculate | PASS | No score math |
| IV. Separation of concerns | PASS | Host display guidance only |
| V. Test first | PASS | Validator, mock, conformance, and TS tests fail first |
| VI. Backward compatibility | PASS | Deprecated field kept (survives one MAJOR); new fields additive |
| VII. Documentation | PASS | Proto comments, scoring doc, READMEs, CLAUDE.md |
| VIII. Performance | PASS | Not a hot path |
| IX. Observability and validation | PASS | Errors name list and index |
| X. Established patterns | PASS | Validator in `sdk/go/testing/scoring.go`; broken-scorer table |
| XI. Copyright headers | PASS | No new source files expected |
| XII. Capability declaration | PASS | Same capability |
| XIII. Multi-language SDK sync | PASS | TS regenerated and tested |
| XIV. Documentation integrity | PASS | Godoc on new mock options and validator |

Post-design re-check: PASS.

## Project Structure

### Documentation (this feature)

```text
specs/589-scorer-multi-call/ (plan, research, data-model, quickstart, contracts/, checklists/, tasks)
```

### Source Code (repository root)

```text
proto/finfocus/v1/scoring.proto
sdk/go/proto/..., sdk/typescript/.../generated/             # regenerated
sdk/go/testing/scoring.go, scoring_test.go                  # rules and tests
sdk/go/testing/scorer_mock.go, scorer_mock_test.go          # two options
sdk/go/testing/scorer_conformance_test.go                   # three broken scorers
sdk/typescript/packages/client/test/recommendation-scorer.test.ts
docs/recommendation-scoring.md, sdk/go/testing/README.md, sdk/go/pluginsdk/README.md, CLAUDE.md
```

**Structure Decision**: Existing layout.

## Complexity Tracking

No constitution violations.
