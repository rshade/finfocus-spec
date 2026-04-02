# Implementation Plan: Terraform Type Resolution RPC

**Branch**: `049-resolve-resource-types` | **Date**: 2026-04-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/049-resolve-resource-types/spec.md`

## Summary

Add a `ResolveResourceTypes` RPC to the CostSourceService that translates IaC-format resource type
strings (Terraform, CloudFormation) to Pulumi type tokens. The implementation includes proto
definitions (new RPC, SourceFormat enum, ResourceTypeMapping message, PluginCapability value 13),
a Go SDK `TypeRegistry` helper for declarative mapping registration, interface-based capability
auto-discovery, and TypeScript SDK client updates. All changes are additive and
backward-compatible.

## Technical Context

**Language/Version**: Go 1.25.8 (per go.mod) + Protocol Buffers v3, TypeScript (SDK)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1
(existing, unchanged)
**Storage**: N/A (stateless type mappings held in-memory after initialization)
**Testing**: `go test` + testify, bufconn-based TestHarness, vitest + msw (TypeScript)
**Target Platform**: Linux/macOS/Windows (cross-platform gRPC library)
**Project Type**: Library (gRPC specification + multi-language SDK)
**Performance Goals**: TypeRegistry.Resolve() < 100ns/op, 0 allocs/op per lookup after init
**Constraints**: Thread-safe concurrent reads; all changes additive (no breaking proto changes)
**Scale/Scope**: TypeRegistry supports 1000+ registered type mappings per source format

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | gRPC Proto Specification-First | PASS | Proto changes (RPC, messages, enums) are the first implementation step |
| II | Multi-Provider gRPC Consistency | PASS | SourceFormat enum is provider-agnostic; TypeRegistry holds no provider data |
| III | Spec Consumes, Not Calculates | PASS | Type mapping is translation/lookup, not pricing calculation |
| IV | Strict Separation of Concerns | PASS | Spec provides mechanism (TypeRegistry); mapping data lives in plugin repos |
| V | Test-First Protocol | PASS | Conformance tests define expected behavior before implementation |
| VI | Protobuf Backward Compatibility | PASS | All changes are additive: new RPC, new enum values, new messages |
| VII | Documentation & Identity | PASS | README, godoc, and SDK docs updated in same PR |
| VIII | Performance as gRPC Requirement | PASS | Zero-allocation lookup target; benchmarks required |
| IX | Observability & Validation | PASS | Structured zerolog logging for unresolved types |
| X | Follow Established Patterns | PASS | Follows DryRunHandler/RecommendationsProvider interface pattern exactly |
| XI | Mandatory Copyright Headers | PASS | Apache 2.0 headers on all new source files |
| XII | Automatic Capability Declaration | PASS | Interface-driven auto-discovery for RESOLVE_RESOURCE_TYPES |
| XIII | Multi-Language SDK Sync | PASS | TypeScript CostSourceClient updated with resolveResourceTypes() |
| XIV | Documentation Integrity | PASS | Godoc on all exports; README examples compile |

**Gate Result**: ALL PASS - no violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/049-resolve-resource-types/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── resolve_resource_types.proto  # Proto contract for the new RPC
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── costsource.proto          # + ResolveResourceTypes RPC definition
└── enums.proto               # + SourceFormat enum, PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES

sdk/go/
├── proto/finfocus/v1/        # Regenerated protobuf Go code (make generate)
└── pluginsdk/
    ├── sdk.go                # + ResolveResourceTypesProvider interface
    │                         # + server handler for ResolveResourceTypes RPC
    ├── plugin_info.go        # + inferCapabilities() updated for new interface
    ├── capability_compat.go  # + legacyCapabilityNames entry for new capability
    ├── type_registry.go      # NEW: TypeRegistry helper (declarative mapping registration)
    ├── type_registry_test.go # NEW: Unit tests for TypeRegistry
    ├── capabilities_test.go  # + test for new capability auto-detection
    └── conformance_test.go   # + conformance test for ResolveResourceTypes

sdk/typescript/packages/client/src/
└── clients/cost-source.ts    # + resolveResourceTypes() async method
```

**Structure Decision**: Extends existing package layout. The only new file is
`type_registry.go` (+ test) in `sdk/go/pluginsdk/`. All other changes are additions to
existing files, following the established pattern for optional RPCs.

## Complexity Tracking

> No Constitution Check violations to justify.
