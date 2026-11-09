# Implementation Plan: Capability-Scoped Supports Requests

**Branch**: `606-supports-request-capability` | **Date**: 2026-11-08 | **Spec**: [spec.md](./spec.md)

**Input**: Feature specification from `specs/606-supports-request-capability/spec.md`

## Summary

Add `PluginCapability capability = 2` to `SupportsRequest`, so a host can say which operation it is about to
call and a plugin can answer per capability. `UNSPECIFIED` keeps today's "any capability" meaning, so the
change is additive. The Go SDK gains three things:

- A zero-allocation read helper, `RequestedCapability`, that treats unknown values as unspecified.
- A host-side `Client.SupportsFor`.
- A capability field in the Supports handler-error log.

`sdk/go/testing` gains a mock option (`UnsupportedCapabilities`) and a Standard conformance check
(`RPCCorrectness_SupportsCapability`). The TypeScript bindings regenerate, and the client passes the field
through. The docs explain the unspecified rule and per-capability answers. See [research.md](./research.md)
for the decisions.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod), Protocol Buffers v3, TypeScript (SDK)

**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, connectrpc.com/connect, zerolog,
buf v1.32.1. No new dependencies.

**Storage**: N/A (one enum field on a request message)

**Testing**: `go test` (unit, bufconn conformance, gRPC and Connect `Serve` tests), Go benchmarks, vitest for
the TS client

**Target Platform**: Plugin SDK library (Linux, macOS, and Windows plugin binaries); TS client for Node and
browsers

**Project Type**: Protocol specification plus multi-language SDK library

**Performance Goals**: `RequestedCapability` costs 0 allocs/op on known and unspecified values. The cost-RPC
hot paths are unchanged.

**Constraints**: Additive only (`buf breaking` clean). No exported signature changes. The Supports success
path is unchanged.

**Scale/Scope**: 1 proto field, 1 helper, 1 client method, 1 log field, 1 mock field, and 1 conformance
check, plus docs in about 6 files

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-checked after Phase 1 design.*

| Principle | Status | How |
|-----------|--------|-----|
| I. Proto first | Pass | The proto field and regeneration come first; generated code is never hand-edited. |
| II. Multi-provider consistency | Pass | The field is provider-agnostic. No billing modes, so no cross-provider examples are needed. Tests use the neutral `custom` provider and `harness.SampleResource()`. |
| III. Consumes, does not calculate | Pass | No pricing logic. |
| IV. Separation of concerns | Pass | Host routing logic stays in finfocus core, as a follow-up. |
| V. Test first | Pass | T002, a request→response contract test, is written and fails before the proto change (T003), which then makes it pass. Each story's tests (helper, client, mock, conformance, TS) fail before its implementation. |
| VI. Backward compatibility | Pass | Optional field 2. `UNSPECIFIED` is today's behavior, and the change is minor. |
| VII. Documentation | Pass | The proto comment, guide, SDK READMEs, testing README, and TS README change in the same PR. |
| VIII. Performance | Pass | Zero-alloc helper with a benchmark. The success path is unchanged. |
| IX. Observability | Pass | The handler-error log gains `capability`; no metrics change. |
| X. Established patterns | Pass | The helper follows `NormalizeUsageProfile`. The check follows the spot check's registration and uses `harness.SampleResource()`. |
| XI. Copyright headers | Pass | New Go and TS files carry the Apache 2.0 header. |
| XII. Automatic capability declaration | Pass | No new provider interface and no change to inference. |
| XIII. SDK synchronization | Pass | TS bindings regenerate, and a TS test shows the field reaches the server. |
| XIV. Documentation integrity | Pass | New exported symbols get godoc. README snippets are compile-checked. |

There are no violations, so Complexity Tracking is empty. Re-checked after Phase 1: the design adds no
interface, dependency, or signature change, and the gates still pass.

## Project Structure

### Documentation (this feature)

```text
specs/606-supports-request-capability/
├── spec.md
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/supports-capability.md
├── checklists/requirements.md
└── tasks.md                # /speckit-tasks
```

### Source Code (repository root)

```text
proto/finfocus/v1/costsource.proto          # SupportsRequest.capability = 2
sdk/go/proto/finfocus/v1/                   # regenerated (make generate)
sdk/go/pluginsdk/
├── supports_capability.go                  # NEW: RequestedCapability
├── supports_capability_test.go             # NEW: helper tests, benchmark, end-to-end, client, log
├── client.go                               # SupportsFor; Supports delegates
├── sdk.go                                  # handler-error log gains FieldCapability
├── logging.go                              # FieldCapability constant
└── README.md                               # helper + SupportsFor docs
sdk/go/testing/
├── supports_capability.go                  # NEW: RPCCorrectness_SupportsCapability
├── supports_capability_test.go             # NEW: check passes ignoring/per-capability, fails erroring
├── rpc_correctness.go                      # one registration line in RPCCorrectnessTests
├── mock_plugin.go                          # UnsupportedCapabilities
├── contract_test.go                        # ValidateSupportsRequest accepts unknown values
└── README.md                               # check + mock field
sdk/typescript/packages/client/
├── src/generated/...                       # regenerated
└── test/supports-capability.test.ts        # NEW
sdk/typescript/README.md                    # snippet (the client package has no README)
PLUGIN_DEVELOPER_GUIDE.md                   # Supports sections
docs/conformance.md                         # if it lists checks
CLAUDE.md, AGENTS.md, GEMINI.md             # Active Technologies, Recent Changes, pattern note (by hand)
```

**Structure Decision**: Use the existing SDK layout. The new behavior goes in new
`supports_capability.go` files in `pluginsdk` and `testing`, and existing files get only minimal edits.
That keeps the merge surface with other in-flight work small. `rpc_correctness.go` gets only the
registration line.

## Complexity Tracking

No violations.
