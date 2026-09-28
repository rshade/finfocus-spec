# Implementation Plan: Allocator Service (Allocate)

**Branch**: `052-allocator-allocate` | **Date**: 2026-09-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/052-allocator-allocate/spec.md`

## Summary

Add `AllocatorService.Allocate` in a new `proto/finfocus/v1/allocation.proto`, with the #506 shape
unchanged, because downstream plans already code against it. Add `PLUGIN_CAPABILITY_ALLOCATION = 15`.
An allocator divides successfully priced infrastructure across workloads and returns workload, idle,
and cluster rows plus the effective policy and its digest.

In the Go SDK, `pluginsdk` gains:

- `AllocatorProvider`, served over gRPC and Connect exactly like #505's `UsageSourceProvider`: same
  interceptors, the health checker, `toConnectError`, capability inference, the legacy flag, and
  the startup warning.
- `DecodePolicy`, a strict decoder that names the JSON path of unknown fields.
- Delegating wrappers `CheckConservation`, `ValidateAllocateRequest`, and `ResolveCurrency`, so hosts
  call production code.

`sdk/go/testing` owns the rules: conservation with a NaN-safe tolerance, request and response
validation, and currency resolution. It also gets an `AllocatorHarness` and
`RunAllocatorConformance`, with 12 policy-agnostic subtests. A reference allocator in
`sdk/go/internal/refalloc` proves the suite, and deliberately broken wrappers prove that the suite
rejects them.

On the TypeScript side: regenerated bindings plus an `AllocatorClient`. The docs gain
`docs/allocator.md` and a "Writing an allocator" section. The change is additive only.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod); TypeScript 5.x (SDK); Protocol Buffers v3

**Primary Dependencies**:

- Go: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
  connectrpc.com/grpchealth, zerolog, buf v1.32.1. No new Go dependencies (the strict decoder uses
  stdlib `encoding/json` and `reflect`).
- TypeScript: @bufbuild/protobuf v2, @connectrpc/connect, vitest + msw.

**Storage**: N/A (stateless RPC contract and SDK helpers)

**Testing**:

- `go test`: testify, bufconn `AllocatorHarness`, and `Serve` with an injected `net.Listener`. Tests
  that import `refalloc` are external (`pluginsdk_test`, `testing_test`) packages.
- `export_test.go` exposes the conformance scenario table to the package's external tests.
- vitest + msw for TypeScript; `buf lint` / `buf breaking`.

**Target Platform**: Plugin binaries (Linux/macOS/Windows) over gRPC or Connect; TS clients in
browser and Node.js

**Project Type**: Protocol specification plus multi-language SDK library

**Performance Goals**:

- Capability inference and `IsValidCapability` stay zero-allocation.
- `CheckConservation` and `ValidateAllocateResponse` are O(rows + priced). They run once per
  allocation, and each has benchmarks at 1k and 10k rows (constitution VIII).
- `DecodePolicy` has a benchmark for a small nested policy.

**Constraints**:

- Additive only (buf `FILE` breaking rules).
- Import direction `pluginsdk → testing` is fixed, so the rules live in `testing` and `pluginsdk`
  delegates (FR-032).
- Names are fixed by downstream plans: `AllocatorProvider`, `DecodePolicy`, `CheckConservation`,
  `ValidateAllocateRequest`, `ResolveCurrency`, `RunAllocatorConformance`, and the proto field
  names.
- Existing usage-source warning text is unchanged (SC-006).

**Scale/Scope**:

- Proto: 1 new file (1 service, 1 RPC, 4 messages) + 1 enum value.
- Go: about 6 new and 6 changed files in `pluginsdk`, 3 new files in `testing`, and 1 internal
  reference package.
- TypeScript: 1 client and 1 test.
- Docs: 1 new page and 5 updated.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | How this plan complies |
|---|-----------|--------|------------------------|
| I | Proto-first | PASS | `allocation.proto` + enum value land first; Go/TS code is generated or wraps generated types. No PricingSpec change, so no JSON schema change. Both new messages get validators: `ValidateAllocateRequest`, `ValidateAllocateResponse`. |
| II | Multi-provider consistency | PASS | Priced resources reuse the provider-agnostic `ResourceDescriptor`; subjects are a string map. Docs show AWS, Azure, and GCP nodes plus an unpriced node. |
| III | Spec consumes, doesn't calculate | PASS (justified in spec) | The contract standardizes the *shape* of allocated cost and its invariants. The only math in the repository is `internal/refalloc`, which Go's `internal` rule makes unimportable outside `sdk/go`, so it cannot ship as a product. The SDK helpers verify sums; they do not allocate. `DecodePolicy` is JSON plumbing, not pricing logic. |
| IV | Separation of concerns | PASS | No production allocator, CLI, or rendering here (SP2/SP3). No new Go dependencies. |
| V | Test-first | PASS | Ordering for tasks: conservation, validation, and decoder tables, then serving and parity tests, then conformance against broken allocators (failing first), then implementation, then the reference allocator. |
| VI | Backward compatibility | PASS | New file + new enum value only; `buf breaking` must pass. Inference is otherwise unchanged, and the usage-source warning is unchanged. The generalized serve plumbing is internal (unexported). |
| VII | Documentation | PASS | Inline proto comments state every invariant and error (FR-030). New `docs/allocator.md`. Updated: pluginsdk README capability table, root README service list, TS README, `PLUGIN_DEVELOPER_GUIDE.md` "Writing an allocator", testing README. |
| VIII | Performance | PASS | Inference is one more type assertion into a pre-sized slice (`optionalCapabilities` 7→8); the row-kind lookup uses a package-level slice; benchmarks per R13. |
| IX | Observability | PASS | The startup warning uses the configured zerolog logger. Error codes survive both transports through `GRPCStatus()` errors (R3) and `toConnectError`. |
| X | Established patterns | PASS | Mirrors 051 point for point (provider interface, adapters, harness, TS client). The design doc is this spec folder. |
| XI | Copyright headers | PASS | Every new `.go`, `.proto`, and `.ts` file carries the Apache 2.0 header. |
| XII | Automatic capability declaration | PASS (documented exception) | Inferred from `AllocatorProvider`. Allocation-only plugins are *advised* to override explicitly, and are warned when they do not, as XII's "Override Support" allows (same as 051). |
| XIII | Multi-language SDK sync | PASS | Bindings regenerated; `AllocatorClient` with vitest coverage; ships in the same release as Go (SC-007). |
| XIV | Documentation integrity | PASS | Godoc on every export. README snippets for `AllocatorProvider` and `DecodePolicy` compile in `example_test.go`. The testing README documents the new helpers. |

No violations; Complexity Tracking is empty.

**Post-design re-check (after Phase 1)**: PASS. The design adds four non-obvious choices, each
recorded in research.md:

- **Status-carrying plain errors (R3)**: validation errors implement `GRPCStatus()`. Allocators can
  return them directly, and SP2 can re-wrap them, with no "rpc error:" double prefix. Existing
  `ContractError` is untouched (VI).
- **NaN-safe conservation (R6)**: non-finite values and epsilons fail closed, because IEEE
  comparisons would otherwise let a NaN total pass.
- **Policy-agnostic conformance (R8)**: rejection fixtures are derived from the allocator's own
  effective policy, so the suite works against any schema. `RunAllocatorConformance` accepts a
  one-method `AllocateServer` interface, which all #506-style callers already satisfy.
- **Reference allocator placement (R9)**: `sdk/go/internal/refalloc` imports the public
  `pluginsdk` helpers, so only external test packages can import it. That is intentional: it
  exercises the same path production allocators take.

## Project Structure

### Documentation (this feature)

```text
specs/052-allocator-allocate/
├── plan.md              # This file
├── research.md          # Phase 0 decisions (R1–R14)
├── data-model.md        # Phase 1 entities and validation rules (Q, P, C, D tables)
├── quickstart.md        # Phase 1 validation scenarios
├── contracts/
│   ├── allocation.proto     # Target content for proto/finfocus/v1/allocation.proto
│   ├── go-sdk-api.md        # pluginsdk + testing public Go API, conformance subtests
│   └── typescript-api.md    # AllocatorClient
├── checklists/requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks; not created here)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── allocation.proto                  # NEW: AllocatorService, AllocateRequest/Response, PricedResource, AllocationRow
└── enums.proto                       # + PLUGIN_CAPABILITY_ALLOCATION = 15

sdk/go/proto/finfocus/v1/             # GENERATED (make generate)
├── allocation.pb.go, allocation_grpc.pb.go
└── pbcconnect/allocation.connect.go

sdk/go/pluginsdk/
├── sdk.go                            # + AllocatorProvider; optionalServices struct replaces usage param
│                                     #   in serveGRPC/serveConnect; register + health
├── allocator.go                      # NEW: gRPC + Connect adapters; delegating CheckConservation,
│                                     #   ValidateAllocateRequest, ResolveCurrency, DefaultConservationEpsilon
├── policy.go                         # NEW: DecodePolicy, ErrInvalidPolicy (reflection path walk + merge)
├── policy_test.go                    # NEW: path, nested merge, array replace, null/empty, trailing, types
├── policy_benchmark_test.go          # NEW
├── usage_source.go                   # warnUsageSourceCapabilities → warnInferredOnlyCapabilities (same text)
├── plugin_info.go                    # inferCapabilities + optionalCapabilities=8 + maxValidCapability=15
├── capability_compat.go              # + "supports_allocation"
├── capability_compat_test.go         # + allocation legacy flag
├── conformance_test.go               # IsValidCapability bounds (15 valid, 16 invalid)
├── plugin_info_test.go               # inferred + explicit allocation capability; warning cases
├── allocator_serve_test.go           # NEW (package pluginsdk_test): refalloc over gRPC & Connect,
│                                     #   error parity, health, not-registered, usage+allocator together
├── example_test.go                   # compiled AllocatorProvider + DecodePolicy examples
└── README.md                         # capability table row, allocator-only rule, helpers

sdk/go/testing/
├── allocation.go                     # NEW: CheckConservation, ConservationError, ResolveCurrency,
│                                     #   ValidateAllocateRequest, ValidateAllocateResponse, sentinels, constants
├── allocation_test.go                # NEW: table-driven Q/P/C rules (pass + fail each) + benchmarks
├── allocator_conformance.go          # NEW: AllocateServer, AllocatorHarness, RunAllocatorConformance, scenarios
├── allocator_conformance_test.go     # NEW (package testing_test): refalloc passes; broken allocators fail
├── export_test.go                    # NEW: exposes the scenario runner to external tests
└── README.md                         # helpers, harness, conformance

sdk/go/internal/refalloc/
├── refalloc.go                       # NEW: reference allocator (test fixture only; see R9)
└── refalloc_test.go                  # NEW (package refalloc_test): policy decode/version, digest, math sanity

sdk/typescript/packages/client/
├── src/generated/finfocus/v1/allocation_pb.ts  # GENERATED
├── src/clients/allocator.ts          # NEW: AllocatorClient
├── src/index.ts                      # exports
└── test/allocator.test.ts            # NEW: msw round-trip + error code

sdk/typescript/README.md              # AllocatorClient section
docs/allocator.md                     # NEW: contract semantics, invariants, policy, currency, errors
PLUGIN_DEVELOPER_GUIDE.md             # "Writing an allocator"
README.md                             # list AllocatorService beside CostSource/UsageSource
```

**Structure Decision**: The existing layout is unchanged. Protos live in `proto/finfocus/v1/`,
generated Go in `sdk/go/proto`, SDK helpers in `sdk/go/pluginsdk`, test tooling and the rule
implementations in `sdk/go/testing`, and the TS client in `sdk/typescript/packages/client`. The only
new directory is `sdk/go/internal/refalloc`, placed under `internal` so the reference allocator stays
a non-product test fixture that both `pluginsdk` and `testing` external tests can import.

## Complexity Tracking

No constitution violations to justify.
