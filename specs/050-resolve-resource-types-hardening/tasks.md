# Tasks: ResolveResourceTypes Hardening

**Input**: Design documents from `/specs/050-resolve-resource-types-hardening/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: Included per Constitution V (Test-First Protocol - NON-NEGOTIABLE for gRPC spec changes).

**Organization**: Tasks are grouped by user story to enable independent implementation and testing.
Each user story maps 1:1 to one PR from the approved implementation plan (US1=PR B, US2=PR A,
US3=PR D, US4=PR C, US5=PR E, US6=PR F) — stories are ordered here by spec.md priority (P1→P4),
which happens to already match a sensible dependency-free build order.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4, US5, US6)
- Exact file paths included in all descriptions

## Phase 1: Setup

**Purpose**: Confirm a clean baseline before any changes.

- [x] T001 Verify baseline: `go build ./...` and `go test ./sdk/go/pluginsdk/... ./sdk/go/testing/...` pass cleanly before starting

---

## Phase 2: Foundational

**Purpose**: Blocking prerequisites shared by all user stories.

**None required.** Unlike spec 049 (which needed a shared proto layer and interface
skeleton before any story could start), every story in this feature extends an
already-shipped file (`batch.go`, `expires_at.go`, `type_registry.go`, `mock_plugin.go`,
`conformance_test.go`, `metrics.go`) independently, and only User Story 2 touches the
proto layer — scoped to that story's own phase below, not shared infrastructure.
Proceed directly to Phase 3.

---

## Phase 3: User Story 1 - Core Is Protected From Oversized Type-Resolution Requests (Priority: P1) 🎯 MVP

**Goal**: Reject a `ResolveResourceTypesRequest` whose `source_types` count exceeds a
configurable limit, before any of the three dispatch tiers run. Mirror client-side in
TypeScript.

**Independent Test**: Send an oversized request and verify `InvalidArgument` with no
provider/registry invocation; send a well-formed request and verify unchanged behavior.

### Tests for User Story 1

> **NOTE: Write these tests FIRST per Constitution V. Ensure they FAIL before implementation.**

- [x] T002 [P] [US1] Write table-driven test `TestValidateResolveResourceTypesRequest` in `sdk/go/pluginsdk/resolve_resource_types_test.go` covering: nil request, empty `source_types`, `SOURCE_FORMAT_UNSPECIFIED`, under limit, at limit, over limit, default applied when `maxSourceTypes<=0`, clamp applied when configured above `MaxSourceTypes`
- [x] T003 [P] [US1] Write test `TestResolveResourceTypes_SizeLimitEnforcedBeforeDispatch` in `sdk/go/pluginsdk/resolve_resource_types_test.go` using a spy `ResolveResourceTypesProvider` that records invocation; assert it is NOT invoked when the request exceeds the configured limit and IS invoked when at/under it
- [x] T004 [P] [US1] Create `sdk/typescript/packages/client/test/resolve-resource-types.test.ts` (following the `batch.test.ts` pattern: msw `setupServer`, imports from `../src/...`) verifying `resolveResourceTypes()` throws a `ValidationError` locally (no RPC call made) when `sourceTypes.length` exceeds `MAX_SOURCE_TYPES`, mirroring the existing `batchCost()` size-limit test in `batch.test.ts`

### Implementation for User Story 1

- [x] T005 [US1] Add `DefaultMaxSourceTypes = 200` and `MaxSourceTypes = 2000` constants to `sdk/go/pluginsdk/batch.go`, co-located with `DefaultMaxBatchSize`/`MaxBatchSize`
- [x] T006 [US1] Add `resolveSourceTypesLimit(configured int) int32` to `sdk/go/pluginsdk/batch.go`, mirroring `resolveBatchSize` (depends on T005)
- [x] T007 [US1] Add `ValidateResolveResourceTypesRequest(req *pbc.ResolveResourceTypesRequest, maxSourceTypes int32) error` to `sdk/go/pluginsdk/batch.go`: nil for nil req/empty list/UNSPECIFIED format; internally applies `if maxSourceTypes <= 0 { maxSourceTypes = DefaultMaxSourceTypes }` before comparing (mirroring `ValidateBatchCostRequest`'s identical defensive fallback at `batch.go:265-267`); `status.Error(codes.InvalidArgument, ...)` when `len(source_types) > maxSourceTypes` (depends on T005)
- [x] T008 [US1] Add `maxSourceTypes int32` field to the `Server` struct and `MaxSourceTypes int` field to `ServeConfig` in `sdk/go/pluginsdk/sdk.go`, set via `resolveSourceTypesLimit` in `Serve()` (depends on T006)
- [x] T009 [US1] Call `ValidateResolveResourceTypesRequest` at the top of `Server.ResolveResourceTypes` in `sdk/go/pluginsdk/sdk.go`, before the interface/TypeRegistry/empty-fallback dispatch tiers, returning its error immediately if non-nil (depends on T007, T008)
- [x] T010 [US1] Add `DEFAULT_MAX_SOURCE_TYPES = 200` / `MAX_SOURCE_TYPES = 2000` constants to `sdk/typescript/packages/client/src/utils/batch.ts`
- [x] T011 [US1] Add the pre-flight length check to `resolveResourceTypes()` in `sdk/typescript/packages/client/src/clients/cost-source.ts`, mirroring `batchCost()`'s existing check (depends on T010)
- [x] T012 [US1] Add a short "Limits" paragraph to the ResolveResourceTypes section of `sdk/go/pluginsdk/README.md` documenting the default/max
- [x] T013 [US1] Verify US1 tests pass: `go test -v -run "TestValidateResolveResourceTypesRequest|TestResolveResourceTypes_SizeLimit" ./sdk/go/pluginsdk/` and the new TypeScript test suite

**Checkpoint**: Oversized requests are rejected before any dispatch tier runs, in both Go and TypeScript.

---

## Phase 4: User Story 2 - Core Can Cache Type Resolutions Instead of Re-Querying Every Time (Priority: P2)

**Goal**: Add an advisory `expires_at` caching hint to `ResolveResourceTypesResponse`,
plus a `TypeRegistry` default-TTL option so plugin authors don't hand-set it per call.

**Independent Test**: Configure a `TypeRegistry` with a default TTL and verify
`Resolve()` stamps `expires_at`; verify a registry without one leaves it nil.

**Note**: This is the only story in this feature that touches the proto layer. The
proto edit and regeneration are scoped here (not in Setup/Foundational) since no other
story depends on them.

### Proto Change for User Story 2

- [x] T014 [US2] Add `google.protobuf.Timestamp expires_at = 2;` field to `ResolveResourceTypesResponse` in `proto/finfocus/v1/costsource.proto` per `contracts/resolve_resource_types_response.proto`
- [x] T015 [US2] Run `make generate` to regenerate Go and TypeScript protobuf bindings (depends on T014)
- [x] T016 [US2] Verify generated code compiles: `go build ./...` (depends on T015)

### Tests for User Story 2

- [x] T017 [P] [US2] Write tests for `IsResolveResourceTypesExpired`, `ResolveResourceTypesExpiresAt`, `WithResolveResourceTypesExpiresAt` in `sdk/go/pluginsdk/expires_at_test.go`, mirroring the existing `EstimateCostResponse` triple's test shape (nil resp, nil `expires_at`, past, future) (depends on T016)
- [x] T018 [P] [US2] Write test `TestNewTypeRegistry_WithDefaultTTL` in `sdk/go/pluginsdk/type_registry_test.go` verifying `Resolve()` stamps `expires_at` at approximately now+TTL when configured via `WithDefaultTTL`, and leaves it nil when not configured (depends on T016)

### Implementation for User Story 2

- [x] T019 [US2] Add `ResolveResourceTypesResponseOption` type and `NewResolveResourceTypesResponse(opts ...ResolveResourceTypesResponseOption) *pbc.ResolveResourceTypesResponse` constructor to `sdk/go/pluginsdk/type_registry.go` (depends on T016)
- [x] T020 [US2] Add `IsResolveResourceTypesExpired`, `ResolveResourceTypesExpiresAt`, `WithResolveResourceTypesExpiresAt` to `sdk/go/pluginsdk/expires_at.go`, copied from the `EstimateCostResponse` triple with types swapped (depends on T019)
- [x] T021 [US2] Add `TypeRegistryOption` type; change `NewTypeRegistry()` to `NewTypeRegistry(opts ...TypeRegistryOption) *TypeRegistry` (variadic — existing zero-arg call sites remain valid); add `WithDefaultTTL(ttl time.Duration) TypeRegistryOption` and a `defaultTTL time.Duration` field on `TypeRegistry`, in `sdk/go/pluginsdk/type_registry.go` (depends on T016)
- [x] T022 [US2] Update `Resolve()` in `sdk/go/pluginsdk/type_registry.go` to stamp `resp.ExpiresAt = timestamppb.New(time.Now().Add(r.defaultTTL))` when `r.defaultTTL > 0`, before returning (depends on T021)
- [x] T023 [US2] Add a "Caching hint" subsection with a `WithDefaultTTL` example to the ResolveResourceTypes section of `sdk/go/pluginsdk/README.md`
- [x] T024 [US2] Verify US2 tests pass: `go test -v -run "TestIsResolveResourceTypesExpired|TestResolveResourceTypesExpiresAt|TestWithResolveResourceTypesExpiresAt|TestNewTypeRegistry_WithDefaultTTL" ./sdk/go/pluginsdk/`

**Checkpoint**: `ResolveResourceTypesResponse` carries an advisory caching hint; `TypeRegistry` can apply one automatically.

---

## Phase 5: User Story 3 - Plugin Authors Can Verify Type-Resolution Support and Operators Can Observe It (Priority: P2)

**Goal**: Add conformance coverage (Basic tier) and MockPlugin support for
`ResolveResourceTypes`, plus resolved/unresolved Prometheus counters with test coverage.

**Independent Test**: Basic conformance passes for both a non-implementing plugin and
the mock plugin; metrics counters change by the expected amounts for a mixed
resolved/unresolved call and don't change on error.

### Tests for User Story 3

- [x] T025 [P] [US3] Add `ResolveResourceTypesConfig` struct (`Mappings`, `ShouldError`, `ErrorMessage`), `SetResolveResourceTypesConfig`, and a `ResolveResourceTypes` method implementing `pluginsdk.ResolveResourceTypesProvider` to `sdk/go/testing/mock_plugin.go`, mirroring `RecommendationsConfig`; seed `NewMockPlugin()` with one default mapping (`aws_instance` → `aws:ec2/instance:Instance`)
- [x] T026 [P] [US3] Write `createResolveResourceTypesEmptyPluginTest`/`createResolveResourceTypesBasicTest` helper functions and register `ResolveResourceTypes_EmptyPlugin`/`ResolveResourceTypes_Basic` `plugintesting.ConformanceTest` entries into `addBasicConformanceTests` in `sdk/go/testing/conformance_test.go`, mirroring the `GetRecommendations_EmptyPlugin`/`GetRecommendations_Basic` pattern exactly (depends on T025 for the `_Basic` test's mock data)
- [x] T027 [US3] Add a one-line comment at the top of `sdk/go/testing/resolve_resource_types_conformance_test.go` cross-referencing the new suite-registered tests, clarifying this file validates a different (`Server`-fallback) layer and is intentionally left otherwise unmodified
- [x] T028 [P] [US3] Write test(s) in `sdk/go/pluginsdk/metrics_test.go` asserting `ResolveResourceTypesResolved`/`ResolveResourceTypesUnresolved` counters increment correctly for a mixed resolved/unresolved response and do NOT increment on an error response

### Implementation for User Story 3

- [x] T029 [US3] Add `ResolveResourceTypesResolved`/`ResolveResourceTypesUnresolved` `*prometheus.CounterVec` fields (labeled `plugin_name`) to `PluginMetrics` in `sdk/go/pluginsdk/metrics.go`, registered in `NewPluginMetrics` alongside `RecommendationsTotal`/`RecommendationsPerResponse`
- [x] T030 [US3] Add a `recordResolveResourceTypesMetrics(metrics *PluginMetrics, req *pbc.ResolveResourceTypesRequest, resp *pbc.ResolveResourceTypesResponse)` function and a matching `if method == "finfocus.v1.CostSource/ResolveResourceTypes" && err == nil` branch in `MetricsInterceptorWithRegistry` in `sdk/go/pluginsdk/metrics.go`, mirroring the existing `GetRecommendations` branch (depends on T029)
- [x] T031 [US3] Verify US3 tests pass: `go test -v -run "TestConformance|TestResolveResourceTypes|TestMetrics" ./sdk/go/testing/... ./sdk/go/pluginsdk/...`

**Checkpoint**: `ResolveResourceTypes` has Basic-tier conformance coverage and observable hit/miss metrics, both with tests.

---

## Phase 6: User Story 4 - Plugin Authors Register Property Overrides in Bulk (Priority: P3)

**Goal**: A batch registration method that includes property-name overrides, plus
README documentation for the (already-shipped, previously undocumented)
`property_mappings` mechanism.

**Independent Test**: Register a batch with mixed property-override presence and verify
each resolved mapping's overrides (or absence) match what was registered.

### Tests for User Story 4

- [x] T032 [P] [US4] Write test `TestTypeRegistry_RegisterMappingsWithProperties` in `sdk/go/pluginsdk/type_registry_test.go` covering a batch with mixed property-override presence (some entries with overrides, one with an empty/nil map)

### Implementation for User Story 4

- [x] T033 [US4] Add `TypeMapping{PulumiToken string; PropertyMappings map[string]string}` struct and `RegisterMappingsWithProperties(format pbc.SourceFormat, mappings map[string]TypeMapping)` method (always `Supported: true`, mirroring `RegisterMappings`'s existing convention) to `sdk/go/pluginsdk/type_registry.go`
- [x] T034 [US4] Add a "Property name overrides" subsection to the ResolveResourceTypes section of `sdk/go/pluginsdk/README.md`: one example distinguishing a mechanical rename (no override needed) from a non-mechanical one, and a note that finfocus-spec only stores this data — applying it during translation remains core's responsibility
- [x] T035 [US4] Verify US4 tests pass: `go test -v -run TestTypeRegistry_RegisterMappingsWithProperties ./sdk/go/pluginsdk/`

**Checkpoint**: Plugin authors can batch-register property overrides; the mechanism is documented.

---

## Phase 7: User Story 5 - Plugin Communities Maintain Mapping Data as Files, Not Hardcoded Registrations (Priority: P3)

**Goal**: `TypeRegistry.LoadMappingsFromJSON`/`LoadMappingsFromFile`, shipping zero
provider-specific data — mechanism only.

**Independent Test**: Load a well-formed JSON file and verify identical resolution to
hand-coded registration; verify malformed/invalid input fails loudly.

### Tests for User Story 5

- [x] T036 [P] [US5] Write test `TestLoadMappingsFromJSON_ValidRoundTrip` in `sdk/go/pluginsdk/type_registry_loader_test.go` verifying a well-formed JSON document (per `contracts/type_registry_mapping_file.schema.json`) resolves identically to the equivalent `RegisterMapping`/`RegisterMappingWithProperties` calls
- [x] T037 [P] [US5] Write test `TestLoadMappingsFromJSON_MalformedJSON` verifying malformed JSON returns a descriptive error and registers nothing
- [x] T038 [P] [US5] Write test `TestLoadMappingsFromJSON_UnknownSourceFormat` verifying an unrecognized/missing `source_format` string returns an error and registers nothing
- [x] T039 [P] [US5] Write test `TestLoadMappingsFromJSON_MissingPulumiToken` verifying an entry missing `pulumi_token` returns an error
- [x] T040 [P] [US5] Write test `TestLoadMappingsFromFile_FileNotFound` verifying a nonexistent file path returns an error
- [x] T041 [P] [US5] Write test `TestLoadMappingsFromJSON_MergeWithExisting` verifying a file loaded after existing registrations for the same format overwrites overlapping keys (last-write-wins) and preserves non-overlapping ones

### Implementation for User Story 5

- [x] T042 [US5] Create `sdk/go/pluginsdk/type_registry_loader.go` with Apache 2.0 header, `typeRegistryFile`/`typeRegistryFileEntry` struct types (JSON tags per `contracts/type_registry_mapping_file.schema.json`), and `LoadMappingsFromJSON(reader io.Reader) error` performing case-insensitive `source_format` matching against the `SourceFormat` enum names and per-entry `pulumi_token` validation
- [x] T043 [US5] Add `LoadMappingsFromFile(path string) error` to `sdk/go/pluginsdk/type_registry_loader.go`, opening the file and delegating to `LoadMappingsFromJSON` (depends on T042)
- [x] T044 [US5] Add a "Loading mappings from a data file" subsection (Option 1b) to the ResolveResourceTypes section of `sdk/go/pluginsdk/README.md`, explicitly stating finfocus-spec does not ship or maintain any mapping data file itself
- [x] T045 [US5] Verify US5 tests pass: `go test -v -run TestLoadMappingsFrom ./sdk/go/pluginsdk/`

**Checkpoint**: Plugin authors can load mapping data from external JSON files instead of hardcoded Go calls.

---

## Phase 8: User Story 6 - CloudFormation-Supporting Plugin Authors Have a Concrete Example (Priority: P4)

**Goal**: Data-model documentation shows a CloudFormation example alongside the
existing Terraform one.

**Independent Test**: Read `specs/049-resolve-resource-types/data-model.md` and confirm
both examples are present.

- [x] T046 [US6] Confirm `specs/049-resolve-resource-types/data-model.md`'s `source_types` field row includes both the Terraform (`aws_instance`) and CloudFormation (`AWS::EC2::Instance`) examples — **already applied during the planning phase of this feature**; this task is a verification checkpoint, not new work

**Checkpoint**: Documentation asymmetry resolved.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Full-suite validation and consistency checks across all six stories.

- [x] T047 [P] Run `make lint-markdown-fix` after all README/data-model.md edits across US1-US5
- [x] T047a [P] Add godoc comments to all new exported symbols introduced across US1-US5 (`TypeMapping`, `TypeRegistryOption`, `WithDefaultTTL`, `RegisterMappingsWithProperties`, `LoadMappingsFromJSON`, `LoadMappingsFromFile`, `ResolveResourceTypesResponseOption`, `NewResolveResourceTypesResponse`, `IsResolveResourceTypesExpired`, `ResolveResourceTypesExpiresAt`, `WithResolveResourceTypesExpiresAt`), per Constitution XIV's 80%+ coverage requirement — mirrors spec 049's T046
- [x] T048 Run `golangci-lint run ./...` (or `make lint-go` for speed) to verify zero new lint issues across all changed files
- [x] T049 Run `make test` to verify all existing tests still pass (zero regressions per SC-001)
- [x] T050 Run `go test -bench=. -benchmem ./sdk/go/pluginsdk/` and confirm `TypeRegistry.Resolve()` stays within its existing <100ns/op, 0 allocs/op target for the no-TTL, under-limit path
- [x] T051 Run `npx tsc --noEmit` and the vitest suite in `sdk/typescript/packages/client/` to confirm the TypeScript changes compile and pass
- [x] T052 Run `make validate` for full validation (tests + linting + npm validations)
- [x] T053 Validate quickstart.md code examples compile by cross-referencing exported function signatures
- [x] T054 Run `speckit.analyze` and remediate any cross-artifact findings before considering the feature ready for implementation review

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — can start immediately
- **Foundational (Phase 2)**: Empty — no blocking work, proceed directly to Phase 3
- **User Stories (Phase 3-8)**: All depend only on Phase 1 (baseline check)
  - US1 (Phase 3), US4 (Phase 6), US5 (Phase 7), US6 (Phase 8) have no proto dependency and no cross-story dependency
  - US2 (Phase 4) is the only story with an internal proto→generate→code sequence (T014→T015→T016), scoped entirely within its own phase
  - US3 (Phase 5) depends on nothing but existing `sdk/go/pluginsdk`/`sdk/go/testing` code
- **Polish (Phase 9)**: Depends on all six user stories being complete

### User Story Dependencies

- **US1 (P1)**: Independent. No dependency on other stories.
- **US2 (P2)**: Independent, but is the sole owner of the proto change — no other story reads or depends on `expires_at`.
- **US3 (P2)**: Independent. No dependency on other stories.
- **US4 (P3)**: Independent. No dependency on other stories.
- **US5 (P3)**: Independent. No dependency on other stories.
- **US6 (P4)**: Independent, documentation-only.

### File-Overlap Coordination (not a logical dependency, but affects merge order)

Three files are touched by more than one story. These stories remain logically
independent and individually testable, but if worked as separate branches/PRs, merge
one before starting the next story's edits to that file to avoid rebase conflicts:

- `sdk/go/pluginsdk/type_registry.go`: US2 (T019, T021, T022), US4 (T033)
- `sdk/go/pluginsdk/README.md` (ResolveResourceTypes section): US1 (T012), US2 (T023), US4 (T034), US5 (T044)
- `sdk/go/pluginsdk/resolve_resource_types_test.go`: US1 only (T002, T003) — no actual overlap, listed for completeness since spec 049 also used this file

### Within Each User Story

- Tests MUST be written and FAIL before implementation (Constitution V)
- Proto changes (US2 only) before any Go/TS code referencing the new field
- Struct/type additions before the methods that use them
- Unit tests before the story's final verification task

### Parallel Opportunities

- T002-T004 (US1 tests) can run in parallel
- T017-T018 (US2 tests) can run in parallel once T016 completes
- T025, T028 (US3 tests) can run in parallel; T026 depends on T025
- T036-T041 (US5 tests) can all run in parallel
- US1, US3, US4, US5, US6 can start in parallel immediately after Phase 1
- US2 can start in parallel too, but its own T014→T015→T016 sequence gates the rest of that story
- T047, T048, T050, T051 (Polish) can run in parallel; T049 and T052 should run after them

---

## Parallel Example: User Story 5

```text
# Launch all loader tests together (write-first, expect FAIL):
Task T036: "TestLoadMappingsFromJSON_ValidRoundTrip in type_registry_loader_test.go"
Task T037: "TestLoadMappingsFromJSON_MalformedJSON in type_registry_loader_test.go"
Task T038: "TestLoadMappingsFromJSON_UnknownSourceFormat in type_registry_loader_test.go"
Task T039: "TestLoadMappingsFromJSON_MissingPulumiToken in type_registry_loader_test.go"
Task T040: "TestLoadMappingsFromFile_FileNotFound in type_registry_loader_test.go"
Task T041: "TestLoadMappingsFromJSON_MergeWithExisting in type_registry_loader_test.go"

# Then implement (sequential, same new file):
Task T042: "Create type_registry_loader.go with LoadMappingsFromJSON"
Task T043: "Add LoadMappingsFromFile"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup baseline check
2. Complete Phase 3: US1 tests + size-limit implementation (Go + TypeScript)
3. **STOP and VALIDATE**: `go test -v ./sdk/go/pluginsdk/ -run TestValidateResolveResourceTypesRequest`
4. The DoS-guard gap (the only P1 item) is closed

### Incremental Delivery

1. Phase 1 → baseline confirmed
2. Add US1 (Phase 3) → request-size limit closes the security gap → MVP
3. Add US2 (Phase 4) → caching hint reduces redundant RPC traffic
4. Add US3 (Phase 5) → conformance + observability parity with other optional RPCs
5. Add US4 (Phase 6) → property-override batch API + docs
6. Add US5 (Phase 7) → JSON mapping-file loader
7. Add US6 (Phase 8) → CloudFormation doc example (verification only)
8. Polish (Phase 9) → full validation, benchmarks, `speckit.analyze` remediation

### Parallel Team Strategy

With multiple developers after Phase 1 completes:

- Developer A: US1 (size limit, Go + TypeScript)
- Developer B: US2 (proto + expires_at + TypeRegistry default TTL)
- Developer C: US3 (conformance + MockPlugin + metrics)
- Developer D: US4 (property-mappings batch API) — coordinate `type_registry.go`/README merge order with Developer B
- Developer E: US5 (JSON loader) — coordinate README merge order with B/D

All stories integrate independently at the code level — no cross-story logical
blocking, only the shared-file merge-order coordination noted above.

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Constitution V: ALL tests must be written and fail before implementation
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Total tasks: 55 (T001-T054 + T047a)
- `speckit.analyze` remediation applied: T007/data-model.md corrected to include `ValidateResolveResourceTypesRequest`'s defensive `maxSourceTypes<=0` fallback (matching the `ValidateBatchCostRequest` precedent); T004 corrected to the real TS test directory (`test/`, not `src/__tests__/`); T047a added for godoc coverage (Constitution XIV), mirroring spec 049's T046
