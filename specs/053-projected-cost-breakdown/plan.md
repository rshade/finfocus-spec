# Implementation Plan: Projected Cost Breakdown

**Branch**: `053-projected-cost-breakdown` | **Date**: 2026-09-27 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/053-projected-cost-breakdown/spec.md`

## Summary

Add `map<string, double> cost_breakdown = 15` to `GetProjectedCostResponse` so plugins can report
named monthly cost components (for example `compute` and `root_volume`) alongside `cost_per_month`,
without free-text parsing (issue #433). The Go SDK validator enforces the clarified rules:

- lowercase snake_case keys
- at most 32 entries
- finite, non-negative values
- a sum within `max(0.01, 0.1% × total)` of the total, as a hard error
- empty on dry-run responses

It does so with zero allocations on the happy path. A copying `WithProjectedCostBreakdown` option,
mock plugin support, regenerated TypeScript bindings and README documentation complete the change.
`EstimateCostResponse` is unchanged except for a comment that points at this field's rules.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (client SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
buf v1.32.1 (all existing, unchanged)

**Storage**: N/A (stateless proto field and validation)

**Testing**: `go test` with testify (`require.ErrorIs`), bufconn `TestHarness`, Go benchmarks, and
vitest with msw for TypeScript

**Target Platform**: Go plugin SDK and TypeScript client (Node.js and browser)

**Project Type**: Library. The wire-protocol specification and SDKs.

**Performance Goals**: `ValidateGetProjectedCostResponse` keeps 0 allocs/op on valid input with or
without a 32-entry breakdown. The existing `_Valid` benchmark stays within 10%.

**Constraints**: Wire and `buf breaking` compatible. No regular expressions or sorting on the
validation path. Generated code is regenerated, never edited by hand.

**Scale/Scope**: One proto field, one comment update, about 5 sentinel errors, one validator helper,
one option helper, one mock field, and tests, benchmarks and docs. About 10 files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | Proto-first | PASS | The proto field (contracts/proto-changes.md) comes first, then `make generate`. No PricingSpec change, so no JSON schema update is needed |
| II | Multi-provider | PASS | Keys are provider-agnostic, and the recommended vocabulary covers AWS, Azure, GCP and Kubernetes components alike |
| III | Consume, don't calculate | PASS | The SDK only checks a sum. Plugins supply pre-adjusted component costs, and negative credits are excluded |
| IV | Separation of concerns | PASS | Display and grouping belong to Core, and aws-public changes are follow-ups |
| V | Test-first | PASS | tasks.md has a test-first gate: the Go and TS tests are written before the proto change (T002) and fail to compile, then fail on behavior until each story's implementation lands |
| VI | Backward compatibility | PASS | Additive field 15, empty-map default, `buf breaking` in quickstart step 1, MINOR bump |
| VII | Documentation | PASS | Proto comment, README section and TOC in the same PR (research R11) |
| VIII | Performance | PASS | Byte-loop key check and map range; new benchmarks require 0 allocs/op (research R3, R4) |
| IX | Validation layers | PASS | Proto comment documents the rules, and the SDK validator enforces them |
| X | Established patterns | PASS | Follows `validateMetadataMap`, sentinel errors, `WithProjectedCost*` options and #427's file set |
| XI | Copyright headers | PASS | No new source files are needed; any new test file gets the Apache 2.0 header |
| XII | Capability declaration | N/A | No new RPC or capability; an empty map signals "no breakdown" |
| XIII | SDK sync | PASS | TypeScript bindings regenerated, plus an integration test (research R9) |
| XIV | Doc integrity | PASS | New exported symbols get godoc, and README examples use exact symbol names |

**Result**: All gates pass. No violations, so Complexity Tracking is empty.

**Post-design re-check (after Phase 1)**: No new violations. The design adds no dependencies, no
new packages and no new RPCs.

## Project Structure

### Documentation (this feature)

```text
specs/053-projected-cost-breakdown/
├── plan.md              # This file
├── research.md          # Phase 0: decisions R1–R12
├── data-model.md        # Phase 1: field rules, validation order, mock field
├── quickstart.md        # Phase 1: validation run guide
├── contracts/
│   ├── proto-changes.md # Field 15 comment + EstimateCostResponse comment
│   └── sdk-helpers.md   # Go sentinels, validator, option; TS exposure
├── checklists/
│   └── requirements.md  # Spec quality checklist (16/16)
└── tasks.md             # Phase 2 (/speckit-tasks, not created here)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
└── costsource.proto                    # + cost_breakdown = 15; EstimateCostResponse comment

sdk/go/proto/finfocus/v1/
└── costsource.pb.go                    # regenerated

sdk/go/pluginsdk/
├── validation.go                       # sentinels, constants, validateCostBreakdown, godoc
├── validation_pricing_test.go          # table tests + _WithCostBreakdown(32) benchmarks
├── helpers.go                          # WithProjectedCostBreakdown
├── helpers_test.go                     # copy semantics, nil/empty
└── README.md                           # "Cost Breakdown Helpers" section + TOC

sdk/go/testing/
├── mock_plugin.go                      # ProjectedCostBreakdown (weights scaled to total)
└── cost_breakdown_test.go              # harness round-trip, batch path, validator pass

sdk/typescript/packages/client/
├── src/generated/finfocus/v1/costsource_pb.ts   # regenerated
└── test/{mocks/handlers.ts,integration.test.ts} # costBreakdown round-trip
```

**Structure Decision**: This follows the existing single-repository SDK layout. Everything goes in
the same packages and files as the #427 `metadata` field, plus the helper and mock additions the spec
requires. The only new source file is `sdk/go/testing/cost_breakdown_test.go`, following
`metadata_test.go`.

**Conformance scope**: `pluginsdk.ValidateGetProjectedCostResponse` alone enforces the breakdown rules.
The conformance suite's `plugintesting.ValidateProjectedCostResponse` (`sdk/go/testing/harness.go`)
is unchanged. It cannot import `pluginsdk` (import cycle), and it doesn't check `metadata` from #427
either. Copying the rules there would mean a second copy that can drift out of sync. Plugins get the
checks by calling the SDK validator, and `cost_breakdown_test.go` runs that validator against the mock
through the harness. A conformance-level breakdown check is a possible follow-up. If it is added, it
should use the private-copy-plus-drift-test pattern from 051 (`KnownSubjectKeys`).

## Complexity Tracking

No Constitution Check violations. This section is intentionally empty.
