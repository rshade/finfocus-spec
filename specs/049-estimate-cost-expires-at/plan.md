# Implementation Plan: EstimateCost expires_at Cache-Hint Parity

**Branch**: `049-estimate-cost-expires-at` | **Date**: 2026-04-02 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/049-estimate-cost-expires-at/spec.md`

## Summary

Add an optional `expires_at` (`google.protobuf.Timestamp`) field to `EstimateCostResponse`
(field 5) to complete cache-hint parity across all three cost response types. This is a
contract-first, additive proto change with Go SDK helpers, mock plugin support, conformance
tests, and benchmarks. TypeScript types are auto-generated from proto definitions.

This feature was explicitly deferred from spec 045 (which added `expires_at` to
`ActualCostResult` and `GetProjectedCostResponse`) and follows identical patterns.

## Technical Context

**Language/Version**: Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
**Storage**: N/A (stateless proto field addition, no persistence)
**Testing**: `go test` (unit, integration, conformance, benchmarks), vitest (TypeScript)
**Target Platform**: Cross-platform (gRPC library)
**Project Type**: SDK/specification library
**Performance Goals**: Zero overhead when field is not populated; helper functions < 10 ns/op
**Constraints**: Full backward compatibility; no breaking proto changes
**Scale/Scope**: 1 proto field, 4 SDK helper functions, 1 functional option, mock plugin
update, conformance tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

- [x] **I. Contract First**: Proto definition updated first (`costsource.proto`), then SDK code
- [x] **II. Multi-Provider**: Field is provider-agnostic (any plugin can set it regardless of provider)
- [x] **III. Spec Consumes**: No pricing logic; `expires_at` is a passthrough caching hint from plugins
- [x] **IV. Separation of Concerns**: Change scoped to spec repo (proto + SDK); no application logic
- [x] **V. Test-First Protocol**: Conformance tests written before implementation (TDD)
- [x] **VI. Backward Compatibility**: Additive field change; nil default preserves existing behavior
- [x] **VII. Documentation**: Proto field comments, godoc on helpers, quickstart examples
- [x] **VIII. Performance**: Benchmark tests verify zero-allocation for helper functions
- [x] **IX. Observability**: No new logging/metrics needed (stateless field)
- [x] **X. Follow Established Patterns**: Mirrors exact pattern from 045 (`expires_at.go`, helpers.go)
- [x] **XI. Copyright Headers**: New files include Apache 2.0 header
- [x] **XII. Capability Declaration**: No new capabilities (field addition, not new RPC)
- [x] **XIII. SDK Synchronization**: Go SDK helpers added; TypeScript types auto-generated from proto
- [x] **XIV. Documentation Integrity**: Plan includes godoc coverage verification and README updates

## Project Structure

### Documentation (this feature)

```text
specs/049-estimate-cost-expires-at/
├── spec.md              # Feature specification
├── plan.md              # This file
├── research.md          # Phase 0 research decisions
├── data-model.md        # Entity changes and field mappings
├── quickstart.md        # Usage examples for plugin developers and callers
├── contracts/
│   ├── proto-changes.md # Exact proto diff with field comments
│   └── sdk-helpers.md   # Go SDK function signatures and contracts
├── checklists/
│   └── requirements.md  # Spec quality validation
└── tasks.md             # Implementation tasks (created by /speckit.tasks)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
└── costsource.proto              # Add expires_at field 5 to EstimateCostResponse

sdk/go/proto/finfocus/v1/
└── costsource.pb.go              # Regenerated from proto (make generate)

sdk/go/pluginsdk/
├── expires_at.go                 # Add EstimateCost helpers (reader + option functions)
└── expires_at_test.go            # Add unit tests for EstimateCost expiration helpers

sdk/go/testing/
├── mock_plugin.go                # Add EstimateCostExpiresAtDuration configuration field
├── expires_at_conformance_test.go # Add EstimateCost conformance tests
└── benchmark_test.go             # Add EstimateCost expires_at benchmark cases

sdk/typescript/packages/client/src/generated/
└── finfocus/v1/costsource_pb.ts  # Auto-regenerated from proto
```

**Structure Decision**: Existing monorepo structure. Proto changes flow through `make generate`
to update Go and TypeScript generated code. New SDK helpers extend existing `expires_at.go`
file following the symmetric pattern established in spec 045. Conformance tests extend the
existing `expires_at_conformance_test.go` file.

## Complexity Tracking

> No constitution violations. All gates pass.

No entries needed.
