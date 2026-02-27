# Implementation Plan: Batch Cost RPC

**Branch**: `046-batch-cost-rpc` | **Date**: 2026-02-24 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/046-batch-cost-rpc/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See
`.specify/templates/plan-template.md` for the execution workflow.

## Summary

Add a dedicated `BatchCost` gRPC RPC to the CostSource service that accepts multiple
`ResourceDescriptor` entries with a shared cost query type (estimate/actual/projected) and
time range, returning per-resource results with partial failure semantics. The SDK provides
automatic sequential fallback with bounded parallelism for plugins that do not implement a
custom batch handler, and a new `PLUGIN_CAPABILITY_BATCH_COST` enum value (12) for
capability discovery.

## Technical Context

**Language/Version**: Go 1.25.7 (per go.mod) + Protocol Buffers v3, TypeScript (SDK)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc, buf v1.32.1,
zerolog
**Storage**: N/A (stateless batch RPC, no data persistence)
**Testing**: go test (unit + integration), conformance suite, vitest + msw (TypeScript)
**Target Platform**: gRPC server/client (Linux, macOS, Windows)
**Project Type**: Library (gRPC spec + multi-language SDK)
**Performance Goals**: Batch of 50 resources < sum of 50 individual requests; <100ms p99 for
SDK overhead per resource; zero-allocation enum validation for CostQueryType
**Constraints**: Backward compatible (plugins without batch support continue unmodified);
default max batch size 100; unary request-response (no streaming)
**Scale/Scope**: Typical batch 10-200 resources; max 100 per request (configurable)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Notes |
|---|-----------|--------|-------|
| I | Proto Spec-First | PASS | New proto messages and RPC defined before SDK implementation |
| II | Multi-Provider Consistency | PASS | ResourceDescriptor is provider-agnostic; batch operates on any provider's resources |
| III | Spec Consumes, Not Calculates | PASS | Batch aggregates existing cost data; no new pricing logic |
| IV | Strict Separation of Concerns | PASS | Batch RPC is plugin SDK scope; orchestration is host/core responsibility |
| V | Test-First Protocol | PASS | Conformance tests define expected batch behavior before implementation |
| VI | Protobuf Backward Compatibility | PASS | Additive only: new RPC, new messages, new enum value (12); no field removals or renumbering |
| VII | Documentation & Identity | PASS | README, SDK docs, and quickstart updated in same PR |
| VIII | Performance as Requirement | PASS | Benchmarks required; zero-alloc CostQueryType validation; bounded parallelism in fallback |
| IX | Observability & Validation | PASS | Structured logging for batch operations; validation on batch size limits |
| X | Follow Established Patterns | PASS | Follows DryRun/Recommendations optional interface pattern for capability discovery |
| XI | Copyright Headers | PASS | All new files include Apache 2.0 headers |
| XII | Automatic Capability Declaration | PASS | BatchCostHandler interface auto-detected via type assertion; PLUGIN_CAPABILITY_BATCH_COST (12) |
| XIII | Multi-Language SDK Sync | PASS | Go SDK and TypeScript SDK updated together; buf regenerates TS bindings |
| XIV | Documentation Integrity | PASS | Godoc on all exported symbols; README code examples compile |

**Gate Result**: ALL PASS - proceed to Phase 0.

## Project Structure

### Documentation (this feature)

```text
specs/046-batch-cost-rpc/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── proto-diff.md    # Detailed proto message changes
└── tasks.md             # Phase 2 output (/speckit.tasks command)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
├── costsource.proto     # New BatchCost RPC + messages
└── enums.proto          # New PLUGIN_CAPABILITY_BATCH_COST (12), CostQueryType enum

sdk/go/
├── pluginsdk/
│   ├── sdk.go           # BatchCostHandler interface, Plugin interface unchanged
│   ├── batch.go         # NEW: batch helpers, fallback logic, worker pool
│   ├── batch_test.go    # NEW: unit tests for batch helpers
│   ├── connect.go       # Wire BatchCost RPC to handler or fallback
│   └── plugin_info.go   # Add BATCH_COST capability detection
├── proto/               # Regenerated from buf
└── testing/
    ├── mock_plugin.go              # Add batch mock behavior
    ├── integration_test.go         # BatchCost integration tests
    ├── benchmark_test.go           # BatchCost benchmarks
    └── batch_conformance_test.go   # NEW: batch conformance tests

sdk/typescript/packages/client/src/
├── clients/cost-source.ts  # Add batchCost() method
└── utils/batch.ts          # NEW: batch helper utilities
```

**Structure Decision**: Follows existing single-project library structure. New batch logic
in dedicated `batch.go` file (matching `dry_run.go` pattern). No new packages needed.

## Complexity Tracking

> No constitution violations to justify.

No violations detected. The batch RPC follows established patterns (optional interface,
capability auto-discovery, fallback to sequential) without introducing new architectural
concepts.
