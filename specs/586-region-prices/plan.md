# Implementation Plan: Per-Region Retail Prices on Cost Responses

**Branch**: `586-region-prices` | **Date**: 2026-10-02 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/586-region-prices/spec.md`

## Summary

Add a `RegionPrice` message and an advisory `region_prices` list to `GetProjectedCostResponse` (field
17) and `EstimateCostResponse` (field 7), leaving 16 and 6 for issue 588. Row rules live in
`sdk/go/testing` so the `pluginsdk` validators and the conformance harness share them. `pluginsdk`
gains two response options, the mock plugin can return rows, and the TypeScript client test reads
them. No sum rule: the primary cost stays the requested region's price.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; existing
`sdk/go/currency`; no new dependencies

**Storage**: N/A (response fields only)

**Testing**: `go test` with benchmarks, bufconn `TestHarness`, vitest + msw

**Target Platform**: Plugin SDKs (Go, TypeScript) and the gRPC wire contract

**Project Type**: Protocol specification and SDK library

**Performance Goals**: `ValidateGetProjectedCostResponse` and `ValidateEstimateCostResponse` stay at
0 allocs/op with and without rows. The no-rows `_Valid` benchmarks stay within noise of `main`.

**Constraints**: Additive wire change; fields 16 and 6 untouched

**Scale/Scope**: One message, two fields, one shared validator, two options, mock field, docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Proto and `make generate` precede SDK code |
| II. Multi-provider consistency | PASS | Region is the provider's own name; no provider-specific rules |
| III. Spec consumes, does not calculate | PASS | Rows are passed through; no conversion, no sums |
| IV. Separation of concerns | PASS | No host comparison logic in the SDK |
| V. Test first | PASS | Validator, option, mock, and TS tests fail before implementation |
| VI. Backward compatibility | PASS | New message and new field numbers only |
| VII. Documentation | PASS | Proto comments, developer guide, both READMEs |
| VIII. Performance | PASS | Call-site `len() > 0` guard; A/B benchmarks against `main` |
| IX. Observability and validation | PASS | Errors name `region_prices[i].<field>` |
| X. Established patterns | PASS | 053 option/mock pattern; 052 testing-holds-rules pattern |
| XI. Copyright headers | PASS | New files get the Apache 2.0 header |
| XII. Capability declaration | PASS | Optional response data; no capability needed |
| XIII. Multi-language SDK sync | PASS | TS regenerated and tested |
| XIV. Documentation integrity | PASS | Godoc on all new exports |

Post-design re-check: PASS.

## Project Structure

### Documentation (this feature)

```text
specs/586-region-prices/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/proto-diff.md
├── checklists/
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto                  # RegionPrice + fields 17 and 7
sdk/go/proto/finfocus/v1/costsource.pb.go           # regenerated
sdk/typescript/packages/client/src/generated/       # regenerated
sdk/go/testing/region_price.go                      # ValidateRegionPrices + sentinels (new)
sdk/go/testing/region_price_test.go                 # rule table + mock tests (new)
sdk/go/testing/harness.go                           # harness response validators call the rule
sdk/go/testing/mock_plugin.go                       # MockPlugin.RegionPrices
sdk/go/pluginsdk/validation.go                      # aliases + guarded calls in both validators
sdk/go/pluginsdk/helpers.go                         # WithProjectedCostRegionPrices, WithEstimateCostRegionPrices
sdk/go/pluginsdk/region_price_test.go               # validator, option, benchmark tests (new)
sdk/typescript/packages/client/test/mocks/handlers.ts
sdk/typescript/packages/client/test/integration.test.ts
PLUGIN_DEVELOPER_GUIDE.md, sdk/go/pluginsdk/README.md, sdk/go/testing/README.md, CLAUDE.md
```

**Structure Decision**: Existing layout. No new packages.

## Complexity Tracking

No constitution violations.
