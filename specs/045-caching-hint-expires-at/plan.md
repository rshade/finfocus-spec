# Implementation Plan: Caching Hint (expires_at) for Cost Results

**Branch**: `045-caching-hint-expires-at` | **Date**: 2026-02-16 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/045-caching-hint-expires-at/spec.md`

## Summary

Add an optional `expires_at` (`google.protobuf.Timestamp`) field to `ActualCostResult` (field 8)
and `GetProjectedCostResponse` (field 13) to enable plugins to signal data freshness to callers.
This is a contract-first, additive proto change with Go SDK helpers, mock plugin support,
conformance tests, and benchmarks. TypeScript types are auto-generated from proto definitions.

## Technical Context

**Language/Version**: Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
**Storage**: N/A (stateless proto field addition, no persistence)
**Testing**: `go test` (unit, integration, conformance, benchmarks), vitest (TypeScript)
**Target Platform**: Cross-platform (gRPC library)
**Project Type**: SDK/specification library
**Performance Goals**: Zero overhead when field is not populated; helper functions < 10 ns/op
**Constraints**: Full backward compatibility; no breaking proto changes
**Scale/Scope**: 2 proto fields, 5 SDK helper functions, mock plugin updates, conformance tests

## Constitution Check

_GATE: Must pass before Phase 0 research. Re-check after Phase 1 design._

- [x] **Contract First**: Proto definitions are updated first (`costsource.proto`), then SDK code
- [x] **Spec Consumes**: No pricing logic; `expires_at` is a passthrough caching hint from plugins
- [x] **Multi-Provider**: Field is provider-agnostic (any plugin can set it regardless of provider)
- [x] **FinFocus Alignment**: Uses `finfocus.v1` package namespace; no legacy naming
- [x] **SDK Synchronization**: Go SDK helpers added; TypeScript types auto-generated from proto
- [x] **Documentation Integrity**: Plan includes godoc coverage verification and README updates
- [x] **Test-First Protocol**: Conformance tests written before implementation (TDD)
- [x] **Backward Compatibility**: Additive field change; nil default preserves existing behavior
- [x] **Performance as Requirement**: Benchmark tests verify zero-allocation for helper functions

## Project Structure

### Documentation (this feature)

```text
specs/045-caching-hint-expires-at/
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
└── costsource.proto              # Add expires_at to ActualCostResult + GetProjectedCostResponse

sdk/go/proto/finfocus/v1/
└── costsource.pb.go              # Regenerated from proto (make generate)

sdk/go/pluginsdk/
├── helpers.go                    # Add WithProjectedCostExpiresAt option function
├── expires_at.go                 # New file: expiration check helpers
└── expires_at_test.go            # New file: unit tests for expiration helpers

sdk/go/testing/
├── mock_plugin.go                # Add ExpiresAtDuration configuration field
├── expires_at_conformance_test.go # New file: conformance tests for expires_at
└── benchmark_test.go             # Add expires_at benchmark cases

sdk/typescript/packages/client/src/generated/
└── finfocus/v1/costsource_pb.ts  # Auto-regenerated from proto
```

**Structure Decision**: Existing monorepo structure. Proto changes flow through `make generate`
to update Go and TypeScript generated code. New SDK helpers follow existing patterns in
`sdk/go/pluginsdk/helpers.go`. New test file follows `*_conformance_test.go` naming convention.

## Complexity Tracking

> No constitution violations. All gates pass.

No entries needed.
