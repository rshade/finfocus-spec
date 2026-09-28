# Implementation Plan: FOCUS 1.4 Cost and Usage Columns

**Branch**: `055-focus-14-cost-usage-columns` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/055-focus-14-cost-usage-columns/spec.md`

## Summary

Phase 1 of the FOCUS 1.4 umbrella (issue 540), delivering issue #541. Two additive fields join
`FocusCostRecord`: `invoice_detail_id = 67` (FOCUS 1.4 `InvoiceDetailId`) and
`commitment_program_eligibility_details = 68` (FOCUS 1.4 `CommitmentProgramEligibilityDetails`, a
JSON object carried as a string). A comment reserves 69 to 80 for later cost-row columns, and
`invoice_issuer = 40` is documented as the 1.4 `InvoiceIssuerName` column without a wire change.

The record validator is brought in line with 1.4:

- The mandatory provider check accepts `service_provider_name` or the deprecated `provider_name`
  (loosening; the error keeps the `provider_name` field name).
- A new `validateFocus14Rules` rejects eligibility details that are not a well-formed JSON object and
  an `invoice_detail_id` without an `invoice_id`. Both checks are allocation-free.

Two allocation-free builder setters, dry-run field names, JSON-LD terms, the TypeScript builder,
regenerated bindings, docs, and a `focus14_conformance_test.go` complete the change.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (client SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1, Go stdlib
`encoding/json` (`json.Valid`). No new modules.

**Storage**: N/A (stateless proto fields and validation)

**Testing**: `go test` with testify, `testing.AllocsPerRun`, Go benchmarks with `b.ReportAllocs()`,
runnable `Example` tests, and vitest with msw for TypeScript

**Target Platform**: Go plugin SDK and TypeScript client (Node.js and browser)

**Project Type**: Library. The wire-protocol specification and SDKs.

**Performance Goals**: New setters < 1 ns/op and 0 allocs/op. `ValidateFocusRecord` stays 0
allocs/op on a valid record without eligibility details, and at most 1 with them (research R2).

**Constraints**: `buf breaking` against `main` passes (additive only). Every previously valid record
stays valid. Generated code is regenerated, never edited by hand.

**Scale/Scope**: 2 proto fields plus comments, 2 builder setters, 1 validation function with 2 rules
and 2 sentinels, 1 relaxed rule, 2 dry-run names, 2 JSON-LD terms, 2 TS builder methods, tests,
benchmarks, and docs. About 20 files.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | Proto-first | PASS | Fields and comments in `focus.proto` (contracts/proto-changes.md) come before SDK code; `make generate` produces bindings. No PricingSpec change, so no JSON schema update |
| II | Multi-provider | PASS | Both columns are provider-agnostic FOCUS columns; examples cover AWS Savings Plans, Azure reservations and GCP committed use |
| III | Consume, don't calculate | PASS | The SDK checks shape only (JSON object, ID pairing); no pricing math |
| IV | Separation of concerns | PASS | No host logic; invoice reconciliation stays in Core |
| V | Test-first | PASS | tasks.md test-first gate: Go and TS tests are written before the proto change and fail to compile (missing fields/setters/sentinels) |
| VI | Backward compatibility | PASS | Additive fields 67 and 68; fields 1 and 55 stay; the only behavior change loosens a rule; `buf breaking` in quickstart step 1 |
| VII | Documentation | PASS | Proto comments, `docs/focus-columns.md` 1.4 section, pluginsdk README section, CLAUDE.md, same PR |
| VIII | Performance | PASS | Setter and validator benchmarks with `b.ReportAllocs()`, plus `AllocsPerRun` guards (research R2, R9) |
| IX | Validation layers | PASS | Proto comments state the rules; the SDK validator enforces the per-record ones |
| X | Established patterns | PASS | Follows `validateFocus13Rules`, sentinel errors with `NewValidationErrorWithCause`, and 1.3 setter style |
| XI | Copyright headers | PASS | The only new source file, `sdk/go/testing/focus14_conformance_test.go`, carries the Apache 2.0 header |
| XII | Capability declaration | N/A | No new RPC or capability |
| XIII | SDK sync | PASS | TS bindings regenerated; TS builder gains matching setters; vitest covers the wire round trip |
| XIV | Doc integrity | PASS | Godoc on new exported symbols, README uses exact names, runnable `Example` with `// Output:` |

**Result**: All gates pass, so Complexity Tracking is empty.

**Post-design re-check (after Phase 1)**: No new violations. No new packages, RPCs, or modules.

## Project Structure

### Documentation (this feature)

```text
specs/055-focus-14-cost-usage-columns/
├── plan.md              # This file
├── research.md          # Phase 0 decisions R1-R11
├── data-model.md        # Field and rule definitions
├── quickstart.md        # Validation guide
├── contracts/
│   ├── proto-changes.md # focus.proto edits, word for word
│   └── sdk-api.md       # Go and TS public API additions
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks output
```

### Source Code (repository root)

```text
proto/finfocus/v1/focus.proto                       # fields 67, 68; comments on 1, 19, 40, 51, 53, 55, 62
sdk/go/proto/finfocus/v1/focus.pb.go                # regenerated
sdk/go/pluginsdk/
├── focus_builder.go                                # WithInvoiceDetailID, WithCommitmentProgramEligibilityDetails
├── focus_conformance.go                            # relaxed provider rule, validateFocus14Rules, 2 sentinels
├── dry_run.go                                      # +2 names, FOCUS 1.2-1.4 wording
├── focus_builder_test.go / focus_conformance_test.go / focus_benchmark_test.go / dry_run_test.go
├── example_test.go                                 # ExampleFocusRecordBuilder_WithCommitmentProgramEligibilityDetails
└── README.md                                       # FOCUS 1.4 section + TOC
sdk/go/testing/
├── focus14_conformance_test.go                     # NEW
└── mock_plugin.go                                  # default dry-run mappings +2 names
sdk/go/jsonld/vocabulary.go, serializer.go, serializer_test.go
sdk/typescript/packages/client/
├── src/generated/finfocus/v1/focus_pb.ts           # regenerated
├── src/builders/focus-record.ts                    # 2 setters
└── test/integration.test.ts, test/mocks/handlers.ts
docs/focus-columns.md                               # FOCUS 1.4 section
CLAUDE.md                                           # Active Technologies, Recent Changes, pattern note
```

**Structure Decision**: Single-repository SDK layout. Every change lands in an existing package; the
only new source file is the conformance test in `sdk/go/testing/`.

## Complexity Tracking

No violations to justify.
