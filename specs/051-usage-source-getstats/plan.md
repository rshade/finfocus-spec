# Implementation Plan: Usage Source Service (GetStats)

**Branch**: `051-usage-source-getstats` | **Date**: 2026-09-25 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/051-usage-source-getstats/spec.md`

## Summary

Add a new `UsageSourceService` with a single `GetStats` RPC in a new `proto/finfocus/v1/usage.proto`.
It returns per-workload usage rows keyed by a string subject map, plus the priceable resources those
workloads run on, as ordinary `ResourceDescriptor`s. Add `PLUGIN_CAPABILITY_USAGE_STATS = 14`.

In the Go SDK:

- An optional `UsageSourceProvider` interface. When a plugin implements it, `pluginsdk.Serve` registers
  the service in both gRPC and Connect modes, adds it to the Connect health checker, and infers the
  capability.
- A startup warning when a usage source relies on inferred capabilities.
- An unexported `toConnectError` helper so Connect clients receive the same status codes as gRPC
  clients (connect-go reports gRPC `status` errors as `Unknown` otherwise; research R8).
- Exported `Subject*`/`Kind*`/`Metric*` constants.

In `sdk/go/testing`: `ValidateGetStatsRequest` (in `contract.go`, research R11),
`ValidateStatsResponse`, and a bufconn `UsageSourceHarness`. In TypeScript:
regenerated bindings plus a `UsageSourceClient` wrapper and mirrored constants. The change is purely
additive.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod); TypeScript 5.x (SDK); Protocol Buffers v3

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect,
connectrpc.com/grpchealth, zerolog, buf v1.32.1; TS: @bufbuild/protobuf v2, @connectrpc/connect,
vitest + msw (tests)

**Storage**: N/A (stateless RPC contract and SDK helpers)

**Testing**: `go test` (testify, bufconn harness, `Serve` with injected `net.Listener`), vitest + msw
for TypeScript, `buf lint` / `buf breaking`

**Target Platform**: Plugin binaries (Linux/macOS/Windows) served over gRPC or Connect; TS clients in
browser and Node.js

**Project Type**: Protocol specification plus multi-language SDK library

**Performance Goals**: Capability inference and `IsValidCapability` stay zero-allocation. The
validation helper is linear in rows + priceable entries. It runs in tests only and has no hot-path
budget, but it has a benchmark per Constitution VIII.

**Constraints**: Additive only (`buf breaking` FILE rules must pass). `pluginsdk` → `testing` import
direction is fixed (`pluginsdk/conformance.go` imports `sdk/go/testing`), so `testing` must not
import `pluginsdk`. Names from #505 are fixed: `UsageSourceProvider`, `ValidateStatsResponse`, and
the `Subject*`/`Kind*`/`Metric*` constants.

**Scale/Scope**: 1 new proto file (1 service, 1 RPC, 3 messages, 1 enum) + 1 enum value; ~8 new/changed
Go files in `pluginsdk`, 4 in `testing` (2 new, `contract.go` and `contract_test.go` extended); 1 TS client +
1 TS util; docs in 6 places

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | How this plan complies |
|---|-----------|--------|------------------------|
| I | Proto-first | PASS | `usage.proto` + enum value land first; all Go/TS code is generated or wraps generated types. No PricingSpec change, so no JSON schema change. Both new messages get validators: `ValidateGetStatsRequest` (R11) and `ValidateStatsResponse` (R5). |
| II | Multi-provider consistency | PASS | Subjects are a provider-agnostic string map; priceable entries reuse `ResourceDescriptor`. Docs show AWS, Azure, and GCP node descriptors plus an unpriceable (Fargate-style) node. |
| III | Spec consumes, doesn't calculate | PASS | Sources return pre-integrated amounts; the SDK does no integration or allocation math (allocation is #506). |
| IV | Separation of concerns | PASS | No concrete usage source or allocator here; `pluginsdk` gains no new external dependencies. |
| V | Test-first | PASS | Tasks order: serving/capability/validator/TS tests written failing before implementation (see quickstart scenarios). |
| VI | Backward compatibility | PASS | New file + new enum value only; `buf breaking` must pass; existing inference unchanged (FR-013). |
| VII | Documentation | PASS | Inline proto comments; `docs/usage-source.md`; pluginsdk README capability table; `PLUGIN_DEVELOPER_GUIDE.md`; testing README; root `README.md` service list; `sdk/typescript/README.md` client section (R10). |
| VIII | Performance | PASS | Inference remains type assertions into a pre-sized slice (`optionalCapabilities` 6→7); known-subject-key lookup uses a package-level slice (registry pattern); benchmarks for `ValidateStatsResponse` and `toConnectError`. |
| IX | Observability | PASS | Startup warning via the configured zerolog logger; no new metrics. Error codes survive both transports (R8), so hosts can tell permission, auth, and argument failures apart. |
| X | Established patterns | PASS | Mirrors `ResolveResourceTypesProvider` (optional interface), `usage_profile.go` (constants + TS mirror), `TestHarness` (bufconn). |
| XI | Copyright headers | PASS | All new `.go`, `.proto`, `.ts` files carry the Apache 2.0 header. |
| XII | Automatic capability declaration | PASS (with documented exception) | Inferred automatically from `UsageSourceProvider`. Usage-only plugins are *advised* to override explicitly, which XII allows ("Override Support"). |
| XIII | Multi-language SDK sync | PASS | TS bindings regenerated; `UsageSourceClient` + constants mirror Go; vitest coverage. |
| XIV | Documentation integrity | PASS | All exported symbols get godoc; README snippets compile (added to `example_test.go`); testing package README updated. |

No violations; Complexity Tracking is empty.

**Post-design re-check (after Phase 1)**: PASS. The design adds two non-obvious choices:

- A private duplicate of the subject vocabulary in `sdk/go/testing`, forced by the import
  direction. It is guarded by a drift test in `pluginsdk` (research R4).
- An explicit gRPC-status-to-Connect error conversion in the usage adapter, because connect-go
  reports unrecognized errors as `Unknown` (research R8). The existing `ConnectHandler` has the
  same defect for cost RPCs; that fix is tracked separately so 051 does not change existing RPC
  behavior (VI).

## Project Structure

### Documentation (this feature)

```text
specs/051-usage-source-getstats/
├── plan.md              # This file
├── research.md          # Phase 0 decisions (R1–R11)
├── data-model.md        # Phase 1 entities, validation rules
├── quickstart.md        # Phase 1 validation scenarios
├── contracts/
│   ├── usage.proto          # Proto contract (target content for proto/finfocus/v1/usage.proto)
│   ├── go-sdk-api.md        # pluginsdk + testing public Go API
│   └── typescript-api.md    # TS client wrapper + constants
├── checklists/requirements.md
└── tasks.md             # Phase 2 (/speckit-tasks; not created here)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── usage.proto                       # NEW: UsageSourceService, GetStats*, UsageRow, StatsMode
└── enums.proto                       # + PLUGIN_CAPABILITY_USAGE_STATS = 14

sdk/go/proto/finfocus/v1/             # GENERATED (make generate)
├── usage.pb.go, usage_grpc.pb.go
└── pbcconnect/usage.connect.go

sdk/go/pluginsdk/
├── sdk.go                            # + UsageSourceProvider; register in serveGRPC/serveConnect; health; warning
├── usage_source.go                   # NEW: gRPC + Connect adapters, warnUsageSourceCapabilities
├── usage_source_test.go              # NEW: serve over gRPC & Connect (responses + error codes), not-registered case, warning
├── connect_errors.go                 # NEW: toConnectError (gRPC status → connect.Error, same code/message)
├── connect_errors_test.go            # NEW: nil, *connect.Error, codes 1–16, plain error
├── subjects.go                       # NEW: Subject*/Kind*/Metric*/Unit* constants
├── subjects_test.go                  # NEW: vocabulary drift guard vs sdk/go/testing
├── plugin_info.go                    # inferCapabilities + optionalCapabilities=7 + maxValidCapability=14
├── capability_compat.go              # + "supports_usage_stats"
├── capability_compat_test.go         # + TestLegacyMetadata_UsageStats
├── conformance_test.go               # IsValidCapability bounds (14 valid, 15 invalid)
├── capabilities_test.go / plugin_info_test.go  # inferred + explicit capability cases
├── example_test.go                   # compiled usage-source example embedding BasePlugin (README sync)
└── README.md                         # capability table row + usage-only rule + BasePlugin embedding

sdk/go/testing/
├── usage_source.go                   # NEW: ValidateStatsResponse, UsageSourceHarness
├── usage_source_test.go              # NEW: table-driven rejection/acceptance + harness + benchmark
├── contract.go                       # + ValidateGetStatsRequest, ErrInvertedStatsWindow (R11)
├── contract_test.go                  # + TestValidateGetStatsRequest
└── README.md                         # document helpers + harness

sdk/typescript/packages/client/
├── src/generated/finfocus/v1/usage_pb.ts  # GENERATED
├── src/clients/usage-source.ts       # NEW: UsageSourceClient
├── src/utils/usage-subjects.ts       # NEW: mirrored constants
├── src/index.ts                      # exports
└── test/usage-source.test.ts         # NEW: msw round-trip

sdk/typescript/README.md              # UsageSourceClient section

docs/usage-source.md                  # NEW: semantics (subjects, metrics, units, priceable, errors)
PLUGIN_DEVELOPER_GUIDE.md             # usage-source section + explicit-capabilities rule
README.md                             # list UsageSourceService beside CostSourceService
```

**Structure Decision**: Existing layout: protos in `proto/finfocus/v1/`, generated Go in
`sdk/go/proto`, SDK helpers in `sdk/go/pluginsdk`, test tooling in `sdk/go/testing`, TS client in
`sdk/typescript/packages/client`. No new packages. The one new doc page goes in `docs/`.

## Complexity Tracking

No constitution violations to justify.
