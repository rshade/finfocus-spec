# Implementation Plan: Resource Descriptor on Actual Cost Requests

**Branch**: `597-actual-cost-resource-descriptor` | **Date**: 2026-10-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/597-actual-cost-resource-descriptor/spec.md`

## Summary

Add `ResourceDescriptor resource = 11` to `GetActualCostRequest` so hosts can send list-price plugins
the same provider, type, SKU, region, and declared inputs on the actual path that they send on the
projected path. The field is optional and additive: unset keeps today's behavior, and `tags` keeps its
meaning as cloud tags. Both actual cost request validators check the descriptor when it is set, the
mock copies its type, region, and SKU into the FOCUS record it attaches, a Standard conformance check
passes plugins that ignore it, and the docs state the fallback and precedence rules. Design decisions
are in [research.md](research.md).

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1; no new
dependencies

**Storage**: N/A (one optional message field on GetActualCostRequest)

**Testing**: `go test` (table tests, contract suite, bufconn conformance harness, benchmarks), vitest +
msw for the TypeScript client

**Target Platform**: Library SDKs and wire protocol

**Project Type**: Protocol specification with Go and TypeScript SDKs

**Performance Goals**: `ValidateActualCostRequest` stays at 0 allocs/op for a valid request without a
descriptor, and for one with a descriptor that has no attributes

**Constraints**: Additive only (`buf breaking` clean); field 10 stays free; generated code is never
hand-edited

**Scale/Scope**: One proto field, two validator call sites, one mock change, one conformance check,
one contract-suite pair, one TS test, five docs

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto first | PASS | Proto field and `make generate` precede SDK code |
| II. Multi-provider consistency | PASS | Reuses the provider-neutral `ResourceDescriptor` |
| III. Spec consumes, does not calculate | PASS | Mock copies descriptor fields; no cost math depends on them (R4) |
| IV. Separation of concerns | PASS | Host-side descriptor construction is out of scope (core) |
| V. Test-first | PASS | Validator, contract, conformance, mock, and TS tests are written before the code they cover |
| VI. Backward compatibility | PASS | New field number, optional, `buf breaking` clean; `tags` unchanged |
| VII. Documentation | PASS | Proto comment plus PROPERTY_MAPPING, developer guide, Go and TS READMEs |
| VIII. Performance | PASS | Nil guard at call site keeps the `_Valid` benchmark at 0 allocs/op (R2) |
| IX. Observability and validation | PASS | Both validator layers cover the field |
| X. Follow established patterns | PASS | Mirrors 585 (actual cost field, Standard check) and 596 (attributes fixture) |
| XI. Copyright headers | PASS | The one new test file carries the Apache 2.0 header |
| XII. Automatic capability declaration | PASS | No capability change |
| XIII. SDK synchronization | PASS | Go and TS bindings regenerated; TS iterator test |
| XIV. Documentation integrity | PASS | README examples touched are checked to compile |

Post-design re-check: unchanged, all PASS. No complexity tracking needed.

## Project Structure

### Documentation (this feature)

```text
specs/597-actual-cost-resource-descriptor/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── proto.md
│   └── go-sdk.md
├── checklists/
│   └── requirements.md
└── tasks.md             # /speckit-tasks
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto            # field 11 + comment
sdk/go/proto/finfocus/v1/                      # regenerated
sdk/typescript/packages/client/src/generated/  # regenerated
sdk/go/pluginsdk/
├── validation.go                              # ValidateActualCostRequest step 6
└── validation_test.go                         # descriptor cases + benchmark
sdk/go/testing/
├── contract.go                                # ValidateGetActualCostRequest + 2 suite cases
├── contract_test.go
├── mock_plugin.go                             # FOCUS record reads descriptor
├── mock_actual_cost_resource_test.go          # new: mock reads descriptor
├── rpc_correctness.go                         # RPCCorrectness_GetActualCostWithResource
├── rpc_correctness_test.go
└── README.md
sdk/typescript/packages/client/test/pagination.test.ts
docs/PROPERTY_MAPPING.md
PLUGIN_DEVELOPER_GUIDE.md
sdk/go/pluginsdk/README.md
sdk/typescript/README.md
CLAUDE.md / AGENTS.md / GEMINI.md               # agent context (by hand)
```

**Structure Decision**: Existing layout; all changes extend files that already own the actual cost
request, its validators, the mock, and the conformance suite.

## Complexity Tracking

None.
