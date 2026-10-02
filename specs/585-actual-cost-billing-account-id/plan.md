# Implementation Plan: Caller-Supplied Billing Account ID on Actual Cost Requests

**Branch**: `585-actual-cost-billing-account-id` | **Date**: 2026-10-01 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/585-actual-cost-billing-account-id/spec.md`

## Summary

Add `string billing_account_id = 9` to `GetActualCostRequest` so a host can supply the FOCUS billing
account id that a plugin needs for a valid FOCUS record. Empty means "not supplied" and plugins must
not invent one. Non-empty is copied into any attached FOCUS record and never filters or changes cost.
The mock plugin becomes the reference producer, a Standard-level conformance test checks the echo
rule, and the TypeScript SDK gets the field through regeneration plus a pagination test.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new
dependencies

**Storage**: N/A (stateless request field)

**Testing**: `go test` (testify), bufconn `TestHarness`, vitest + msw for TypeScript

**Target Platform**: Plugin SDKs (Go, TypeScript) and the gRPC wire contract

**Project Type**: Protocol specification and SDK library

**Performance Goals**: No change to any zero-allocation path. The mock allocates a FOCUS record only
when the request carries an id.

**Constraints**: Additive wire change only; `buf breaking` against `main` must pass

**Scale/Scope**: One proto field, one mock behavior, one validator, one conformance test, docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Proto edit and `make generate` precede SDK code |
| II. Multi-provider consistency | PASS | Field is provider-neutral; no provider-specific semantics |
| III. Spec consumes, does not calculate | PASS | Id is passed through; no cost math changes |
| IV. Separation of concerns | PASS | No host logic; plugin contract and SDK test helpers only |
| V. Test first | PASS | Mock, validator, conformance, and TS tests are written failing first |
| VI. Backward compatibility | PASS | New field number; nothing renamed, renumbered, or retyped |
| VII. Documentation | PASS | Proto comment, developer guide, pluginsdk and testing READMEs |
| VIII. Performance | PASS | No hot-path change; validator is not on a benchmarked path |
| IX. Observability and validation | PASS | Contract error names the offending result index and field |
| X. Established patterns | PASS | Follows `contract.go` `ContractError` and `rpc_correctness.go` patterns |
| XI. Copyright headers | PASS | Any new source file gets the Apache 2.0 header |
| XII. Capability declaration | PASS | No new capability; the field is optional for plugins |
| XIII. Multi-language SDK sync | PASS | TS bindings regenerated; TS test added; client wrapper needs no change |
| XIV. Documentation integrity | PASS | Exported symbols get godoc; README examples stay compilable |

Post-design re-check: PASS. The design adds no exported surface beyond one sentinel, one validator,
and one conformance test entry.

## Project Structure

### Documentation (this feature)

```text
specs/585-actual-cost-billing-account-id/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   └── proto-diff.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks output
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto              # add field 9 + comment
sdk/go/proto/finfocus/v1/costsource.pb.go       # regenerated
sdk/typescript/packages/client/src/generated/   # regenerated
sdk/go/testing/
├── contract.go                                 # ErrBillingAccountIDMismatch, ValidateActualCostBillingAccount
├── contract_test.go                            # validator table tests
├── mock_plugin.go                              # attach FOCUS record when id is set
├── mock_billing_account_test.go                # mock echo / omission / cost-unchanged tests
├── rpc_correctness.go                          # RPCCorrectness_GetActualCostBillingAccount
├── rpc_correctness_test.go                     # registration + pass/fail tests
└── README.md                                   # document the new test and validator
sdk/go/pluginsdk/
├── actual_cost_billing_account_test.go         # pluginsdk_test: mock + ValidateFocusRecord
└── README.md                                   # field contract for plugin authors
sdk/typescript/packages/client/test/pagination.test.ts   # billingAccountId on every page
PLUGIN_DEVELOPER_GUIDE.md                       # request block + implementation notes
CLAUDE.md                                       # Active Technologies / Recent Changes (by hand)
```

**Structure Decision**: Existing repository layout. No new packages.

## Complexity Tracking

No constitution violations.
