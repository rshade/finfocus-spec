# Tasks: Terraform Type Resolution RPC

**Input**: Design documents from `/specs/049-resolve-resource-types/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Included per Constitution V (Test-First Protocol - NON-NEGOTIABLE for gRPC spec changes).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Exact file paths included in all descriptions

## Phase 1: Setup (Proto Definitions)

**Purpose**: Define the wire protocol. All user stories depend on these proto changes.

- [x] T001 Add SourceFormat enum to `proto/finfocus/v1/enums.proto` with values UNSPECIFIED (0), TERRAFORM (1), CLOUDFORMATION (2) per contracts/resolve_resource_types.proto
- [x] T002 Add `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES = 13` to PluginCapability enum in `proto/finfocus/v1/enums.proto` after BATCH_COST (12)
- [x] T003 Add ResourceTypeMapping message to `proto/finfocus/v1/costsource.proto` with fields pulumi_token (string, 1), supported (bool, 2), property_mappings (map<string,string>, 3)
- [x] T004 Add ResolveResourceTypesRequest and ResolveResourceTypesResponse messages to `proto/finfocus/v1/costsource.proto` per contracts/resolve_resource_types.proto
- [x] T005 Add ResolveResourceTypes RPC to CostSourceService in `proto/finfocus/v1/costsource.proto` with documentation following the DryRun/BatchCost RPC comment pattern
- [x] T006 Run `make generate` to regenerate Go protobuf code in `sdk/go/proto/finfocus/v1/`
- [x] T007 Verify generated code compiles with `go build ./...`

---

## Phase 2: Foundational (SDK Core Infrastructure)

**Purpose**: Interface definition and server handler skeleton. MUST complete before user story implementation.

**CRITICAL**: No user story work can begin until this phase is complete.

- [x] T008 Add `ResolveResourceTypesProvider` interface to `sdk/go/pluginsdk/sdk.go` following the RecommendationsProvider/BudgetsProvider pattern (single method: ResolveResourceTypes)
- [x] T009 Add `ResolveResourceTypes` server handler method to the server struct in `sdk/go/pluginsdk/sdk.go` implementing empty-response fallback when plugin does not implement ResolveResourceTypesProvider (following GetRecommendations pattern at sdk.go:652-666, NOT GetBudgets Unimplemented pattern)
- [x] T010 Add nil response guard in the server handler (return codes.Internal if plugin returns nil response, following pattern at sdk.go:681-687)

**Checkpoint**: Foundation ready -- `ResolveResourceTypes` RPC returns empty response for all plugins. User story implementation can now begin.

---

## Phase 3: User Story 1 - Core Resolves Terraform Types via Plugin (Priority: P1) MVP

**Goal**: A plugin implementing `ResolveResourceTypesProvider` returns correct Pulumi token mappings. Plugins without the interface return empty responses for graceful fallback.

**Independent Test**: Send ResolveResourceTypesRequest with known Terraform types to a mock plugin; verify response contains correct Pulumi tokens. Send to a plugin without the interface; verify empty response.

### Tests for User Story 1

> **NOTE: Write tests FIRST per Constitution V. Ensure they FAIL before implementation.**

- [x] T011 [P] [US1] Create mock plugin implementing ResolveResourceTypesProvider in `sdk/go/pluginsdk/resolve_resource_types_test.go` with 3 test Terraform->Pulumi mappings (aws_instance, aws_s3_bucket, aws_lambda_function)
- [x] T012 [P] [US1] Write test `TestResolveResourceTypes_WithProvider` in `sdk/go/pluginsdk/resolve_resource_types_test.go` verifying: (a) request with 3 known types returns 3 mappings with correct tokens and supported=true, (b) request with 1 known + 1 unknown type returns only the known mapping
- [x] T013 [P] [US1] Write test `TestResolveResourceTypes_WithoutProvider` in `sdk/go/pluginsdk/resolve_resource_types_test.go` verifying a plugin without the interface returns empty ResolveResourceTypesResponse (not error)
- [x] T014 [US1] Write test `TestResolveResourceTypes_EdgeCases` in `sdk/go/pluginsdk/resolve_resource_types_test.go` covering: (a) empty source_types list returns empty response, (b) SOURCE_FORMAT_UNSPECIFIED returns empty response, (c) nil response from plugin returns codes.Internal

### Implementation for User Story 1

- [x] T015 [US1] Implement the server handler delegate path in `sdk/go/pluginsdk/sdk.go` -- when plugin implements ResolveResourceTypesProvider, delegate to plugin.ResolveResourceTypes() with error handling (following GetRecommendations delegate pattern at sdk.go:670-677)
- [x] T015a [US1] Add zerolog debug-level log in the server handler `ResolveResourceTypes` in `sdk/go/pluginsdk/sdk.go` logging: (a) count of requested types, (b) count of resolved types, (c) list of unresolved type names -- following the logging pattern in GetRecommendations handler (sdk.go:645-649)
- [x] T016 [US1] Verify all US1 tests pass with `go test -v -run TestResolveResourceTypes ./sdk/go/pluginsdk/`

**Checkpoint**: ResolveResourceTypes RPC works end-to-end with mock plugins. Both provider and non-provider paths verified.

---

## Phase 4: User Story 2 - Plugin Developer Registers Type Mappings Declaratively (Priority: P2)

**Goal**: Plugin developers register Terraform->Pulumi mappings via TypeRegistry and the SDK auto-handles the RPC. Fewer than 10 lines of configuration code.

**Independent Test**: Create a TypeRegistry with test mappings, call Resolve(), verify correct results. Verify cross-format isolation (Terraform mappings don't leak to CloudFormation queries).

### Tests for User Story 2

- [x] T017 [P] [US2] Write test `TestTypeRegistry_Resolve_KnownTypes` in `sdk/go/pluginsdk/type_registry_test.go` verifying: registry with 3 Terraform mappings resolves all 3 correctly with supported=true
- [x] T018 [P] [US2] Write test `TestTypeRegistry_Resolve_UnknownTypes` in `sdk/go/pluginsdk/type_registry_test.go` verifying: request with 3 known + 1 unknown type returns exactly 3 mappings (unknown omitted)
- [x] T019 [P] [US2] Write test `TestTypeRegistry_Resolve_CrossFormatIsolation` in `sdk/go/pluginsdk/type_registry_test.go` verifying: Terraform mappings are NOT returned for CloudFormation source format queries
- [x] T020 [P] [US2] Write test `TestTypeRegistry_Resolve_EmptyAndUnspecified` in `sdk/go/pluginsdk/type_registry_test.go` verifying: (a) empty source_types returns empty response, (b) SOURCE_FORMAT_UNSPECIFIED returns empty response
- [x] T021 [P] [US2] Write test `TestTypeRegistry_RegisterMapping_Single` in `sdk/go/pluginsdk/type_registry_test.go` verifying: single mapping registration with supported=false resolves correctly with supported=false

### Implementation for User Story 2

- [x] T022 [US2] Create `sdk/go/pluginsdk/type_registry.go` with TypeRegistry struct, NewTypeRegistry() constructor, RegisterMapping(), RegisterMappings(), Resolve(), and Len() methods per data-model.md entity 6
- [x] T023 [US2] Add Apache 2.0 copyright header to `sdk/go/pluginsdk/type_registry.go`
- [x] T024 [US2] Add `TypeRegistry *TypeRegistry` field to ServeConfig struct in `sdk/go/pluginsdk/sdk.go`
- [x] T025 [US2] Update server handler in `sdk/go/pluginsdk/sdk.go` to check ServeConfig.TypeRegistry as second fallback (after interface check, before empty response) -- delegate to registry.Resolve() when TypeRegistry is non-nil
- [x] T026 [US2] Write test `TestResolveResourceTypes_WithTypeRegistry` in `sdk/go/pluginsdk/resolve_resource_types_test.go` verifying: server with TypeRegistry (no interface) delegates to registry and returns correct mappings
- [x] T027 [US2] Write test `TestResolveResourceTypes_InterfacePrecedence` in `sdk/go/pluginsdk/resolve_resource_types_test.go` verifying: when both interface and TypeRegistry are configured, interface result is used (not registry)
- [x] T028 [US2] Verify all US2 tests pass with `go test -v -run "TestTypeRegistry|TestResolveResourceTypes_With" ./sdk/go/pluginsdk/`
- [x] T028a [US2] Write test `TestTypeRegistry_ConcurrentReads` in `sdk/go/pluginsdk/type_registry_test.go` launching 10 goroutines that concurrently call Resolve() on a shared TypeRegistry with 100 mappings; run with `go test -race` to verify no data races (FR-012)

**Checkpoint**: TypeRegistry handles RPC requests automatically. Plugin developers can register mappings declaratively with <10 lines.

---

## Phase 5: User Story 3 - Capability Auto-Detection for Type Resolution (Priority: P3)

**Goal**: GetPluginInfo automatically reports RESOLVE_RESOURCE_TYPES capability when plugin implements the interface or when TypeRegistry is configured.

**Independent Test**: Create a mock plugin implementing ResolveResourceTypesProvider, call GetPluginInfo, verify capabilities list includes PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES and legacy metadata includes "supports_resolve_resource_types".

### Tests for User Story 3

- [x] T029 [P] [US3] Write test `TestInferCapabilities_ResolveResourceTypes` in `sdk/go/pluginsdk/capabilities_test.go` verifying: mock plugin implementing ResolveResourceTypesProvider triggers PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES in inferred capabilities
- [x] T030 [P] [US3] Write test `TestInferCapabilities_NoResolveResourceTypes` in `sdk/go/pluginsdk/capabilities_test.go` verifying: mock plugin NOT implementing the interface does NOT include the new capability
- [x] T031 [P] [US3] Write test `TestLegacyMetadata_ResolveResourceTypes` in `sdk/go/pluginsdk/capability_compat_test.go` verifying: PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES maps to legacy metadata key "supports_resolve_resource_types"

### Implementation for User Story 3

- [x] T032 [US3] Update `optionalCapabilities` constant from 5 to 6 in `sdk/go/pluginsdk/plugin_info.go` (line ~197) and update the comment listing interfaces (line ~195-196) to include ResolveResourceTypesProvider
- [x] T033 [US3] Update `maxValidCapability` constant from BATCH_COST (12) to RESOLVE_RESOURCE_TYPES (13) in `sdk/go/pluginsdk/plugin_info.go` (line ~217)
- [x] T034 [US3] Add ResolveResourceTypesProvider type assertion to `inferCapabilities()` in `sdk/go/pluginsdk/plugin_info.go` following the DryRunHandler/BatchCostHandler pattern (line ~267-269)
- [x] T035 [US3] Add `PLUGIN_CAPABILITY_RESOLVE_RESOURCE_TYPES: "supports_resolve_resource_types"` to `legacyCapabilityNames` map in `sdk/go/pluginsdk/capability_compat.go` (line ~37) and update the exhaustiveness comment (line ~16) from "values (1-12)" to "values (1-13)"
- [x] T036 [US3] Add ResolveResourceTypesProvider method to `mockCapabilityPlugin` in `sdk/go/pluginsdk/capabilities_test.go` and verify `TestGetPluginInfo_AutoDiscovery` still passes with the new capability included
- [x] T037 [US3] Verify all US3 tests pass with `go test -v -run "TestInferCapabilities|TestLegacyMetadata|TestGetPluginInfo" ./sdk/go/pluginsdk/`

**Checkpoint**: Capability auto-detection works. GetPluginInfo reports the new capability via both enum and legacy metadata.

---

## Phase 6: User Story 4 - Future-Proof Property Mapping Overrides (Priority: P4)

**Goal**: The property_mappings field on ResourceTypeMapping is present in the proto, serializes/deserializes correctly, and is empty by default.

**Independent Test**: Create a ResourceTypeMapping with property_mappings populated, serialize to wire format and back, verify data integrity.

### Tests for User Story 4

- [x] T038 [P] [US4] Write test `TestResourceTypeMapping_PropertyMappings_RoundTrip` in `sdk/go/pluginsdk/resolve_resource_types_test.go` verifying: ResourceTypeMapping with property_mappings populated serializes and deserializes correctly via proto.Marshal/Unmarshal

### Implementation for User Story 4

- [x] T039 [US4] Add `WithPropertyMappings` option or direct field support in TypeRegistry.RegisterMapping() in `sdk/go/pluginsdk/type_registry.go` to allow setting property_mappings on individual mappings
- [x] T040 [US4] Write test `TestTypeRegistry_RegisterMapping_WithPropertyMappings` in `sdk/go/pluginsdk/type_registry_test.go` verifying: mapping registered with property_mappings resolves with those mappings preserved
- [x] T041 [US4] Verify property_mappings is empty by default in TypeRegistry responses when not explicitly set

**Checkpoint**: Property mappings field works end-to-end. Empty by default, preserved when populated.

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: TypeScript SDK sync, documentation, benchmarks, and final validation.

- [x] T042 [P] Add `resolveResourceTypes()` async method to CostSourceClient in `sdk/typescript/packages/client/src/clients/cost-source.ts` following the dryRun()/batchCost() pattern
- [x] T043 [P] Add TypeScript client test for resolveResourceTypes() in `sdk/typescript/packages/client/src/__tests__/` following existing RPC test patterns
- [x] T044 [P] Write benchmark `BenchmarkTypeRegistry_Resolve` in `sdk/go/pluginsdk/type_registry_test.go` measuring: (a) single lookup, (b) batch of 10, (c) batch of 1000; verify <100ns/op and 0 allocs/op per lookup
- [x] T045 [P] Add conformance test for ResolveResourceTypes in `sdk/go/testing/` integration test suite following existing RPC conformance patterns
- [x] T046 [P] Add godoc comments to all exported types and functions in `sdk/go/pluginsdk/type_registry.go`
- [x] T046a [P] Update `sdk/go/pluginsdk/README.md` capability auto-detection table (line ~657-660) to add ResolveResourceTypesProvider row, and add TypeRegistry usage section following the existing DryRun/BatchCost documentation pattern
- [x] T047 Run `make lint` to verify no linting errors across all changed files
- [x] T048 Run `make test` to verify all existing tests still pass (zero regressions per SC-001)
- [x] T049 Run `make validate` to run full validation suite (tests + linting + npm validations)
- [x] T050 Run `go test -bench=. -benchmem ./sdk/go/pluginsdk/` and verify TypeRegistry performance meets targets
- [x] T051 Validate quickstart.md code examples compile by cross-referencing exported function signatures

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies -- can start immediately
- **Foundational (Phase 2)**: Depends on Phase 1 completion (needs generated proto code) -- BLOCKS all user stories
- **User Stories (Phase 3-6)**: All depend on Phase 2 completion
  - US1 (Phase 3) and US2 (Phase 4) can proceed **in parallel** (different files)
  - US3 (Phase 5) depends on the interface from Phase 2 only (no dependency on US1 or US2)
  - US4 (Phase 6) depends on Phase 1 only (proto field existence)
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Phase 2 (server handler + interface). No dependency on other stories.
- **US2 (P2)**: Depends on Phase 1 (proto types) + Phase 2 (ServeConfig integration). No dependency on US1.
- **US3 (P3)**: Depends on Phase 2 (interface definition). No dependency on US1 or US2.
- **US4 (P4)**: Depends on Phase 1 (proto field). No dependency on other stories.

### Within Each User Story

- Tests MUST be written and FAIL before implementation (Constitution V)
- Proto types before Go SDK code
- Interface before implementation
- Unit tests before integration tests

### Parallel Opportunities

- T001 and T002 can run in parallel (different enum additions in same file)
- T003, T004, T005 are sequential (messages then RPC in costsource.proto)
- T011-T014 (US1 tests) can all run in parallel
- T017-T021 (US2 tests) can all run in parallel
- T029-T031 (US3 tests) can all run in parallel
- US1, US2, US3, US4 can start in parallel after Phase 2 (if team capacity allows)
- T042-T046 (Polish tasks) can all run in parallel

---

## Parallel Example: User Story 2

```text
# Launch all TypeRegistry tests together (write-first, expect FAIL):
Task T017: "TestTypeRegistry_Resolve_KnownTypes in type_registry_test.go"
Task T018: "TestTypeRegistry_Resolve_UnknownTypes in type_registry_test.go"
Task T019: "TestTypeRegistry_Resolve_CrossFormatIsolation in type_registry_test.go"
Task T020: "TestTypeRegistry_Resolve_EmptyAndUnspecified in type_registry_test.go"
Task T021: "TestTypeRegistry_RegisterMapping_Single in type_registry_test.go"

# Then implement (sequential, same file):
Task T022: "Create type_registry.go with TypeRegistry struct and methods"
Task T023: "Add copyright header"
Task T024: "Add TypeRegistry field to ServeConfig"
Task T025: "Update server handler for TypeRegistry fallback"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Proto definitions + `make generate`
2. Complete Phase 2: Interface + server handler skeleton
3. Complete Phase 3: US1 tests + server handler delegation
4. **STOP and VALIDATE**: `go test -v ./sdk/go/pluginsdk/ -run TestResolveResourceTypes`
5. The RPC works end-to-end with mock plugins

### Incremental Delivery

1. Phase 1 + 2 -> Proto + SDK foundation ready
2. Add US1 (Phase 3) -> RPC works with interface implementation -> MVP
3. Add US2 (Phase 4) -> TypeRegistry provides declarative ergonomics
4. Add US3 (Phase 5) -> Capability auto-detection completes the plugin lifecycle
5. Add US4 (Phase 6) -> Property mappings field verified for future use
6. Polish (Phase 7) -> TypeScript SDK, benchmarks, docs, full validation

### Parallel Team Strategy

With multiple developers after Phase 2 completes:

- Developer A: US1 (RPC handler tests + implementation)
- Developer B: US2 (TypeRegistry + ServeConfig integration)
- Developer C: US3 (capability inference + legacy metadata)
- Developer D: US4 (property mappings round-trip)

All stories integrate independently -- no cross-story blocking.

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Constitution V: ALL tests must be written and fail before implementation
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total tasks: 54 (T001-T051 + T015a, T028a, T046a)
