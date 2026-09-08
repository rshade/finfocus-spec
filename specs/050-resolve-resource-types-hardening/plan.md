# Implementation Plan: ResolveResourceTypes Hardening

**Branch**: `050-resolve-resource-types-hardening` | **Date**: 2026-09-07 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/050-resolve-resource-types-hardening/spec.md`

## Summary

Harden the `ResolveResourceTypes` RPC and `TypeRegistry` SDK helper shipped in spec
049 with six additive, backward-compatible improvements: (1) a request-size limit on
`source_types` mirroring the existing `BatchCost` DoS guard, (2) an `expires_at`
caching hint on the response plus a `TypeRegistry` default-TTL option, (3) conformance
test and mock-plugin coverage plus resolved/unresolved Prometheus metrics, (4) a batch
registration API for property-name overrides, (5) a JSON mapping-file loader so plugin
authors can maintain mapping data as versioned files instead of hardcoded Go calls, and
(6) a documentation fix adding a CloudFormation example alongside the existing
Terraform one. Only item 2 touches the wire protocol (one new response field); every
other item is Go/TypeScript SDK or documentation only.

## Technical Context

**Language/Version**: Go 1.27.1 (per go.mod) + Protocol Buffers v3, TypeScript (SDK)
**Primary Dependencies**: google.golang.org/protobuf, google.golang.org/grpc,
buf v1.32.1, github.com/prometheus/client_golang (all existing, unchanged)
**Storage**: N/A (stateless in-memory type mappings and metrics counters)
**Testing**: `go test` + testify, bufconn-based `TestHarness`, vitest (TypeScript)
**Target Platform**: Linux/macOS/Windows (cross-platform gRPC library)
**Project Type**: Library (gRPC specification + multi-language SDK)
**Performance Goals**: `ValidateResolveResourceTypesRequest` and the `expires_at` stamp
add O(1)/negligible overhead to `Resolve()`; existing <100ns/op, 0 allocs/op lookup
target for `TypeRegistry.Resolve()` is preserved for the no-TTL, under-limit path.
**Constraints**: All changes additive (no breaking proto changes); no production
consumer of this RPC exists yet, so the new request-size limit carries no realistic
compatibility risk.
**Scale/Scope**: `DefaultMaxSourceTypes = 200`, hard ceiling `MaxSourceTypes = 2000`
per request (deliberately above `BatchCost`'s 100/1000, since `source_types` is a
de-duplicated type list, not a per-resource list).

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| # | Principle | Status | Evidence |
|---|-----------|--------|----------|
| I | gRPC Proto Specification-First | PASS | The one wire change (`expires_at` field) is defined in proto first, `make generate` run before any Go/TS code is written |
| II | Multi-Provider gRPC Consistency | PASS | No provider-specific data added anywhere; the JSON mapping-file loader is a generic deserializer, not provider content |
| III | Spec Consumes, Not Calculates | PASS | No pricing/calculation logic touched; this feature only affects type-string resolution and caching metadata |
| IV | Strict Separation of Concerns | PASS | JSON loader ships zero mapping data (mechanism only, same split spec 049 already established); mapping data remains plugin-repo responsibility |
| V | Test-First Protocol | PASS | Each of the six items ships with new/expanded tests (validator tests, `expires_at` tests, conformance tests, metrics tests, loader tests) |
| VI | Protobuf Backward Compatibility | PASS | Single additive field (`expires_at = 2`) on `ResolveResourceTypesResponse`; no field removal/renumbering/type change |
| VII | Documentation & Identity | PASS | README updated for limits, caching hint, property overrides, and the loader; data-model.md gets the CloudFormation example, all in the same PRs as the code |
| VIII | Performance as gRPC Requirement | PASS | Validator and TTL-stamp are O(1); benchmarks re-run to confirm `TypeRegistry.Resolve()` stays at its existing performance target for the common case |
| IX | Observability & Validation | PASS | New resolved/unresolved Prometheus counters (with test coverage, unlike the `GetRecommendations` precedent) plus the existing `zerolog` debug logging this reuses |
| X | Follow Established Patterns | PASS | Mirrors `ValidateBatchCostRequest`/`resolveBatchSize` (batch.go), the `expires_at.go` per-message triple pattern, and the `GetRecommendations` conformance/metrics precedent exactly |
| XI | Mandatory Copyright Headers | PASS | Apache 2.0 header on the one new file (`type_registry_loader.go`) |
| XII | Automatic Capability Declaration | PASS | Unchanged — no new optional interface introduced; `ResolveResourceTypesProvider` capability detection from spec 049 is untouched |
| XIII | Multi-Language SDK Sync | PASS | TypeScript client mirrors the request-size limit (`resolveResourceTypes()` gets the same pre-flight check as `batchCost()`); `expires_at` reaches TS via regenerated bindings automatically, consistent with how spec 045 handled it |
| XIV | Documentation Integrity | PASS | Godoc required on all new exported symbols; README code examples for the new APIs must compile; data-model.md updated in the same change |

**Gate Result**: ALL PASS — no violations to justify.

## Project Structure

### Documentation (this feature)

```text
specs/050-resolve-resource-types-hardening/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   └── resolve_resource_types_response.proto  # Proto contract delta (expires_at field)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
proto/finfocus/v1/
└── costsource.proto              # + expires_at field on ResolveResourceTypesResponse

sdk/go/
├── proto/finfocus/v1/            # Regenerated protobuf Go code (make generate)
└── pluginsdk/
    ├── batch.go                  # + DefaultMaxSourceTypes/MaxSourceTypes consts,
    │                             #   ValidateResolveResourceTypesRequest, resolveSourceTypesLimit
    ├── sdk.go                    # + maxSourceTypes on Server/ServeConfig; validation call
    │                             #   wired before ResolveResourceTypes dispatch tiers
    ├── expires_at.go             # + IsResolveResourceTypesExpired, ResolveResourceTypesExpiresAt,
    │                             #   WithResolveResourceTypesExpiresAt, ResolveResourceTypesResponseOption
    ├── expires_at_test.go        # + tests for the new triple
    ├── type_registry.go          # + TypeMapping struct, RegisterMappingsWithProperties,
    │                             #   TypeRegistryOption, NewTypeRegistry(opts...), WithDefaultTTL,
    │                             #   NewResolveResourceTypesResponse constructor
    ├── type_registry_test.go     # + tests for batch-with-properties and WithDefaultTTL
    ├── type_registry_loader.go   # NEW: LoadMappingsFromJSON / LoadMappingsFromFile
    ├── type_registry_loader_test.go  # NEW: loader tests
    ├── resolve_resource_types_test.go  # + validator tests, handler-level limit-enforcement test
    ├── metrics.go                # + ResolveResourceTypesResolved/Unresolved counters,
    │                             #   interceptor branch mirroring GetRecommendations
    ├── metrics_test.go           # + tests for the new counters
    └── README.md                 # + Limits, Caching hint, Property overrides,
                                   #   Loading mappings from a data file sections

sdk/go/testing/
├── mock_plugin.go                    # + ResolveResourceTypesConfig, SetResolveResourceTypesConfig,
│                                     #   ResolveResourceTypes method, seeded default mapping
├── conformance_test.go               # + ResolveResourceTypes_EmptyPlugin/_Basic in Basic tier
└── resolve_resource_types_conformance_test.go  # + one-line cross-reference comment only

sdk/typescript/packages/client/src/
├── utils/batch.ts                # + DEFAULT_MAX_SOURCE_TYPES / MAX_SOURCE_TYPES
└── clients/cost-source.ts        # + pre-flight length check in resolveResourceTypes()

specs/049-resolve-resource-types/
└── data-model.md                 # + CloudFormation example alongside Terraform example
```

**Structure Decision**: Extends the existing `pluginsdk`/`testing` package layout from
spec 049. The only new source file is `type_registry_loader.go` (+ test) in
`sdk/go/pluginsdk/`; every other change is an addition to a file spec 049 or an earlier
spec already introduced, following each file's established pattern (`batch.go` for
limits, `expires_at.go` for caching hints, `metrics.go` for the `GetRecommendations`
precedent).

## Complexity Tracking

> No Constitution Check violations to justify.
