# Implementation Plan: Include Dismissed Recommendations

**Branch**: `599-include-dismissed` | **Date**: 2026-10-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/599-include-dismissed/spec.md`

## Summary

Add `bool include_dismissed = 8` to `GetRecommendationsRequest`. Plugins that store dismissals omit
those IDs unless the field is true. `excluded_recommendation_ids` still omits its IDs in both cases.
The Go SDK exposes `ApplyRecommendationVisibility`, the mock plugin applies the same rules before
pagination, and the developer guide describes the field. Generated Go and TypeScript bindings come
from `make generate`. Design decisions are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf; no new dependencies

**Storage**: N/A (one bool on an existing request message)

**Testing**: `go test` for the helper and the mock; one TypeScript round-trip; `buf lint` and
`buf breaking`

**Target Platform**: Library SDKs and the wire protocol

**Project Type**: Protocol specification with Go and TypeScript SDKs

**Performance Goals**: The helper returns the input slice unchanged when both ID lists are empty

**Constraints**: Additive only (`buf breaking` clean); field numbers 1-7 stay put; generated code is
never hand-edited; the mock must not import `pluginsdk` (import cycle)

**Scale/Scope**: One proto field, one helper, one mock config field, one developer-guide section,
one TypeScript test

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Field and `make generate` precede helper and mock code |
| II. Multi-provider consistency | PASS | One rule for every plugin that stores dismissals |
| III. Spec consumes, does not calculate | PASS | The helper filters IDs; it does not price anything |
| IV. Separation of concerns | PASS | Host wiring is rshade/finfocus#545, specified in core as `622-include-dismissed` |
| V. Test-first | PASS | Helper and mock tests are written before the filter |
| VI. Backward compatibility | PASS | New field number, default false, `buf breaking` clean |
| VII. Documentation | PASS | Proto comment and the plugin developer guide |
| VIII. Performance | PASS | Empty ID lists do not copy the slice |
| IX. Observability and validation | PASS | No new validation; a bool has no illegal value |
| X. Follow established patterns | PASS | Same shape as `excluded_recommendation_ids` |
| XI. Copyright headers | PASS | New test files carry the Apache 2.0 header |
| XII. Automatic capability declaration | PASS | No capability change |
| XIII. SDK synchronization | PASS | Go and TypeScript regenerated |
| XIV. Documentation integrity | PASS | Developer guide snippet matches the proto field |

Post-design re-check: unchanged, all PASS. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/599-include-dismissed/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── proto.md
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto
sdk/go/pluginsdk/helpers.go
sdk/go/pluginsdk/recommendation_visibility_test.go
sdk/go/testing/mock_plugin.go
sdk/go/testing/recommendations_visibility_test.go
sdk/typescript/packages/client/test/integration.test.ts
PLUGIN_DEVELOPER_GUIDE.md
```

## Complexity Tracking

No constitution violations.
