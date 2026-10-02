# Implementation Plan: Alternative Retail Price Options

**Branch**: `557-price-options` | **Date**: 2026-10-01 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/557-price-options/spec.md`

## Summary

Add a `PriceOption` message and an advisory `repeated PriceOption price_options` field to
`GetProjectedCostResponse` (field 16) and `EstimateCostResponse` (field 6) so plugins can report
on-demand, reservation, and savings-plan prices beside the one selected price (issue #588). The list
is never summed into `cost_per_month` or `cost_monthly`. Fields 17 and 7 are held by comment for a
later per-region list.

The Go SDK validators reject a nil entry, NaN or infinity on any of the four doubles, and a negative
`unit_price`, `monthly_cost`, or `upfront_cost`. They reject the list on a dry-run projected
response. They do not recompute `savings_fraction` or relate the list to the primary cost. Validation
stays at 0 allocs/op. Two copying options (`WithProjectedCostPriceOptions`,
`WithEstimatePriceOptions`), two mock plugin fields, regenerated TypeScript bindings, and README
sections complete the change.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (client SDK)

**Primary Dependencies**: google.golang.org/protobuf v1.36.12 (`proto.CloneOf`),
google.golang.org/grpc, connectrpc.com/connect, buf v1.32.1 (all existing, unchanged)

**Storage**: N/A (stateless proto message, fields, and validation)

**Testing**: `go test` with testify (`require.ErrorIs`), bufconn `TestHarness`, Go benchmarks with
`testing.AllocsPerRun`, and vitest with msw for TypeScript

**Target Platform**: Go plugin SDK and TypeScript client (Node.js and browser)

**Project Type**: Library. The wire-protocol specification and SDKs.

**Performance Goals**: Both validators stay at 0 allocs/op on valid input, with no options and with
four options. The existing `_Valid` benchmarks stay within 10% (A/B against a `main` worktree with
prebuilt test binaries, per the 053 note in CLAUDE.md).

**Constraints**: Wire and `buf breaking` compatible. No `reserved` statement for fields 17 and 7.
Generated code is regenerated, never edited by hand. No recomputation of `savings_fraction`.

**Scale/Scope**: One message, two fields, two comment updates, three sentinel errors, one validator
helper, two options, two mock fields, and tests, benchmarks, and docs. About 12 files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | Proto-first | PASS | The message and fields (contracts/proto-changes.md) come first, then `make generate`. No PricingSpec change, so no JSON schema update |
| II | Multi-provider | PASS | `model` and `term` are free provider strings; quickstart and README examples cover Azure (Consumption/Reservation/SavingsPlan) and AWS (On-Demand/Reserved/Savings Plans). GCP CUDs fit the same shape |
| III | Consume, don't calculate | PASS | Plugins supply every number, including `savings_fraction`. The SDK checks only that values are finite and non-negative (research R4) |
| IV | Separation of concerns | PASS | Side-by-side display belongs to Core. The Azure plugin work is plugin issue #45 |
| V | Test-first | PASS | tasks.md will start with Go and TS tests that reference `PriceOption`, so they fail to compile before the proto change, then fail on behavior until the validator lands |
| VI | Backward compatibility | PASS | Additive fields 16 and 6, empty-list default, `buf breaking` in quickstart step 1, MINOR bump. Comment-only hold for 17/7 (research R2) |
| VII | Documentation | PASS | Proto comments, pluginsdk README section and TOC, testing README, and PLUGIN_DEVELOPER_GUIDE bullet in the same PR (research R10) |
| VIII | Performance | PASS | Index loop over the slice, guarded by `len() > 0` at the call site; new benchmarks require 0 allocs/op (research R5) |
| IX | Validation layers | PASS | Proto comment documents the rules; the SDK validator enforces them |
| X | Established patterns | PASS | Follows 053 `validateCostBreakdown`, sentinel errors, `WithProjectedCost*` options, and mock fields |
| XI | Copyright headers | PASS | The one new test file (`sdk/go/testing/price_options_test.go`) gets the Apache 2.0 header |
| XII | Capability declaration | N/A | No new RPC or capability. A non-empty list signals support (research R8) |
| XIII | SDK sync | PASS | TypeScript bindings regenerated, plus an integration round-trip test (research R9) |
| XIV | Doc integrity | PASS | New exported symbols get godoc; README examples use exact symbol names |

**Result**: All gates pass. One justified exception to Principle V step 1 is recorded under
Complexity Tracking.

**Post-design re-check (after Phase 1)**: No new violations. The design adds no dependencies, no new
packages, no new RPCs, and no new capability. The only rules beyond the issue text (negative prices,
dry-run, nil entry) tighten a brand-new field, so no existing plugin can break.

## Project Structure

### Documentation (this feature)

```text
specs/557-price-options/
├── plan.md              # This file
├── research.md          # Phase 0: decisions R1–R10
├── data-model.md        # Phase 1: PriceOption fields, validation order, mock fields
├── quickstart.md        # Phase 1: validation run guide
├── contracts/
│   ├── proto-changes.md # PriceOption, fields 16/6, comment holds for 17/7
│   └── sdk-helpers.md   # Go sentinels, validator, options, mock fields; TS exposure
├── checklists/
│   └── requirements.md  # Spec quality checklist (16/16)
└── tasks.md             # Phase 2 (/speckit-tasks, not created here)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
└── costsource.proto                    # + PriceOption; price_options = 16 / = 6; comment updates

sdk/go/proto/finfocus/v1/
└── costsource.pb.go                    # regenerated

sdk/go/pluginsdk/
├── validation.go                       # sentinels, validatePriceOptions, both validators + godoc
├── validation_pricing_test.go          # table tests + _WithPriceOptions benchmarks + AllocsPerRun
├── helpers.go                          # WithProjectedCostPriceOptions, WithEstimatePriceOptions
├── helpers_test.go                     # deep-copy semantics, nil/empty
└── README.md                           # "Price Option Helpers" section + TOC

sdk/go/testing/
├── mock_plugin.go                      # ProjectedCostPriceOptions, EstimateCostPriceOptions
├── price_options_test.go               # harness round-trip for both RPCs, pluginsdk validator pass
└── README.md                           # mock field docs

PLUGIN_DEVELOPER_GUIDE.md               # bullet next to the cost_breakdown bullet

sdk/typescript/packages/client/
├── src/generated/finfocus/v1/costsource_pb.ts   # regenerated
└── test/{mocks/handlers.ts,integration.test.ts} # priceOptions round-trip
```

**Structure Decision**: This follows the existing single-repository SDK layout and the 053
`cost_breakdown` file set. The only new source file is `sdk/go/testing/price_options_test.go`,
following `cost_breakdown_test.go`.

**Conformance scope**: `pluginsdk.ValidateGetProjectedCostResponse` and
`pluginsdk.ValidateEstimateCostResponse` alone enforce the price option rules. The conformance
suite's `plugintesting.ValidateProjectedCostResponse` and `ValidateEstimateCostResponse`
(`sdk/go/testing/harness.go`) are unchanged, for the same import-cycle reason recorded in 053. A
conformance-level check is a possible follow-up.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| Principle V step 1 ("write conformance tests"): `plugintesting.Validate*` and `Run*Conformance` do not check `price_options` | `sdk/go/testing` cannot import `pluginsdk` (import cycle), and a private copy of the rules would drift (the reason 053 gives for `cost_breakdown`) | Harness tests in `sdk/go/testing/price_options_test.go` run the real `pluginsdk` validators against the mock over bufconn, which covers the RPC contract. A conformance-level check is follow-up work using the 051 private-copy-plus-drift-test pattern |
