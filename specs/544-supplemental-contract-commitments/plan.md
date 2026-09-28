# Implementation Plan: Supplemental Dataset Service (Contract Commitments)

**Branch**: `544-supplemental-contract-commitments` | **Date**: 2026-09-28 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/544-supplemental-contract-commitments/spec.md`

## Summary

Add `SupplementalDatasetService.GetContractCommitments` in a new
`proto/finfocus/v1/supplemental.proto`, returning the existing FOCUS 1.3 `ContractCommitment`
message (imported from `focus.proto`, which does not change), plus
`PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16`.

In the Go SDK, `pluginsdk` gains `ContractCommitmentProvider`, served exactly like
`UsageSourceProvider` and `AllocatorProvider`: a new `optionalServices.commitments` field,
registration over gRPC and Connect, a Connect health entry, `toConnectError`, capability inference,
and the legacy flag `supports_contract_commitments`. Delegating wrappers expose the rules to plugins
and hosts, and `ContractCommitmentBuilder.Build` uses the same commitment validator.

`sdk/go/testing` owns the rules (the import-cycle constraint from spec 051): commitment, request,
and response validation, the window-matching rule, and pagination. It also gets
`MockContractCommitmentSource` (the reference producer), `ContractCommitmentHarness`, and
`RunContractCommitmentConformance` with seven source-agnostic scenarios.

TypeScript gets regenerated bindings and a `SupplementalDatasetClient` with `getContractCommitments`
and a paging iterator `contractCommitments`. Docs gain `docs/supplemental-datasets.md` and SDK README
sections. Every change is additive.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod); TypeScript 5.x (SDK); Protocol Buffers v3

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc,
connectrpc.com/connect, connectrpc.com/grpchealth, buf v1.32.1 (all existing). No new Go or npm
dependencies.

**Storage**: N/A (stateless RPC; the reference producer holds an in-memory list)

**Testing**: `go test` (table tests, bufconn harness, Serve tests over a real listener for both
transports, benchmarks with `b.ReportAllocs()` and `testing.AllocsPerRun`); vitest + msw for TS

**Target Platform**: Plugin binaries and hosts on Linux, macOS, and Windows; Node.js and browsers
for the TS client

**Project Type**: Protocol specification plus Go and TypeScript SDK libraries

**Performance Goals**: `ValidateContractCommitment`, `ValidateGetContractCommitmentsRequest`, and
`ContractCommitmentMatchesWindow` make zero allocations. `ValidateGetContractCommitmentsResponse`
makes zero allocations for pages of up to 64 records (pairwise duplicate check) and uses a map
above that (research R4).

**Constraints**: `sdk/go/testing` cannot import `pluginsdk`; `focus.proto`, `buf.yaml`, and
`GetActualCost` do not change; no `unsafe`; the gRPC and Connect behavior of plugins that do not
implement the provider must not change.

**Scale/Scope**: Pages of at most 1000 commitments; typical sources hold tens to hundreds of
commitments.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
| --- | --- | --- |
| I. Proto-first | Pass | The new proto is written first; generated code is never hand-edited. PricingSpec is not touched, so there is no JSON schema change. |
| II. Multi-provider | Pass | Commitments are provider-agnostic FOCUS records; docs show AWS, Azure, and GCP commitment types. |
| III. Consumes, not calculates | Pass | Pass-through of records the source holds; no discount or amortization math. |
| IV. Separation of concerns | Pass | Only protocol, SDK helpers, and a test producer. |
| V. Test-first | Pass | Conformance and SDK tests are written before the proto change and fail to compile until it lands (tasks test-first gate). |
| VI. Backward compatibility | Pass | New file, new enum value; `buf breaking` against `main`. |
| VII. Documentation | Pass | Proto comments, `docs/supplemental-datasets.md`, pluginsdk/testing/TS READMEs, CLAUDE.md. |
| VIII. Performance | Pass | Benchmarks for every new exported function; zero-allocation validators proven with `AllocsPerRun`. |
| IX. Observability | Pass | Serving reuses the existing interceptor chain; no new logging. |
| X. Established patterns | Pass | Mirrors specs 044 (pagination), 051 (optional service), 052 (invalid-argument errors, delegation). |
| XI. Copyright headers | Pass | Every new Go, proto, and TS file gets the Apache 2.0 header. |
| XII. Automatic capability | Pass | Inferred from `ContractCommitmentProvider`; legacy key added; explicit override unchanged. |
| XIII. SDK sync | Pass | TS bindings and `SupplementalDatasetClient` ship in the same change. |
| XIV. Doc integrity | Pass | Godoc on all exports; README examples use exact symbols; compiled `Example` functions. |

No violations; Complexity Tracking is empty.

## Project Structure

### Documentation (this feature)

```text
specs/544-supplemental-contract-commitments/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── supplemental.proto
│   ├── go-sdk-api.md
│   └── typescript-api.md
├── checklists/requirements.md
└── tasks.md
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── supplemental.proto                 # NEW: service, request, response
└── enums.proto                        # + PLUGIN_CAPABILITY_CONTRACT_COMMITMENTS = 16

sdk/go/proto/finfocus/v1/              # regenerated (supplemental*.go, enums.pb.go, pbcconnect/)

sdk/go/testing/
├── supplemental.go                    # NEW: validators, window rule, pagination, sentinels
├── supplemental_mock.go               # NEW: MockContractCommitmentSource
├── contract_commitment_conformance.go # NEW: harness + RunContractCommitmentConformance
├── export_test.go                     # + RunContractCommitmentScenariosForTest
└── *_test.go                          # NEW tests and benchmarks

sdk/go/pluginsdk/
├── sdk.go                             # + ContractCommitmentProvider, optionalServices.commitments,
│                                      #   gRPC/Connect registration, health entry
├── supplemental.go                    # NEW: adapters + delegating wrappers
├── plugin_info.go                     # inference, optionalCapabilities, maxValidCapability
├── capability_compat.go               # supports_contract_commitments
├── contract_commitment_builder.go     # validate() delegates to the shared validator
└── *_test.go                          # serve, parity, capability, wrapper, example, benchmark tests

sdk/typescript/packages/client/
├── src/generated/finfocus/v1/         # regenerated supplemental_pb.ts, enums_pb.ts
├── src/clients/supplemental-dataset.ts
├── src/index.ts
└── test/supplemental-dataset.test.ts

docs/supplemental-datasets.md          # NEW canonical semantics
```

**Structure Decision**: Follow the existing single-repo layout; the new service mirrors the files
added for `usage.proto` (051) and `allocation.proto` (052).

## Complexity Tracking

None.
