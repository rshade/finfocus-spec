# Tasks: Batch Cost RPC

**Input**: Design documents from `/specs/046-batch-cost-rpc/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/proto-diff.md,
quickstart.md

**Tests**: Conformance tests are required per Constitution V (Test-First Protocol).

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Proto-First)

**Purpose**: Define the protocol contract before any implementation (Constitution I).

- [x] T001 Add `CostQueryType` enum (UNSPECIFIED=0, ESTIMATE=1, ACTUAL=2, PROJECTED=3)
  to `proto/finfocus/v1/enums.proto` after the `UsageProfile` enum block
- [x] T002 Add `PLUGIN_CAPABILITY_BATCH_COST = 12` to the `PluginCapability` enum in
  `proto/finfocus/v1/enums.proto` after `DISMISS_RECOMMENDATIONS = 11`
- [x] T003 Add `BatchCost` RPC to `CostSourceService` in
  `proto/finfocus/v1/costsource.proto` after the `DryRun` RPC (see
  `contracts/proto-diff.md` for exact definition)
- [x] T004 Add new messages (`BatchCostRequest`, `BatchCostResponse`,
  `ResourceCostResult`, `CostData`, `ActualCostData`, `ResourceError`) to
  `proto/finfocus/v1/costsource.proto` (see `contracts/proto-diff.md` for field
  definitions)
- [x] T005 Run `make generate` to regenerate Go protobuf code in `sdk/go/proto/`
- [x] T006 Run `buf breaking` to verify all changes are backward-compatible
- [x] T007 Run `go build ./...` to verify generated code compiles

**Checkpoint**: Proto contract defined. All changes are additive. Generated code compiles.

---

## Phase 2: Foundational (SDK Infrastructure)

**Purpose**: Core SDK infrastructure that MUST be complete before any user story
implementation.

- [x] T008 Create `sdk/go/pluginsdk/batch.go` with Apache 2.0 header, package
  declaration, and batch constants: `DefaultMaxBatchSize = 100`, `MaxBatchSize = 1000`,
  `DefaultBatchWorkers = 10`, `MinBatchWorkers = 1`, `MaxBatchWorkers = 50`
- [x] T009 Add `CostQueryType` validation helpers to `sdk/go/pluginsdk/batch.go`:
  `IsValidCostQueryType()` using zero-allocation package-level slice (follow
  `sdk/go/registry/` pattern), `NormalizeCostQueryType()` that treats UNSPECIFIED as
  ESTIMATE
- [x] T010 Add `BatchCostHandler` optional interface to `sdk/go/pluginsdk/sdk.go` after
  the `DryRunHandler` interface: single method
  `BatchCost(ctx, *pbc.BatchCostRequest) (*pbc.BatchCostResponse, error)`
- [x] T011 Add `MaxBatchSize int` and `BatchWorkers int` fields to `ServeConfig` struct
  in `sdk/go/pluginsdk/sdk.go`
- [x] T012 Add `ValidateBatchCostRequest()` function to `sdk/go/pluginsdk/batch.go`:
  validate batch size <= MaxBatchSize, validate start/end required when query_type=ACTUAL,
  validate start < end, return empty response for empty resources list
- [x] T013 [P] Add `NewBatchCostResponse()` builder with functional options
  (`WithBatchResults()`, `WithMaxBatchSize()`) to `sdk/go/pluginsdk/batch.go`
- [x] T014 [P] Add `NewResourceError()` helper function to
  `sdk/go/pluginsdk/batch.go` for constructing `ResourceError` messages with gRPC
  status codes

**Checkpoint**: SDK infrastructure ready. Interfaces, constants, validation, and helpers
defined. User story implementation can begin.

---

## Phase 3: User Story 1 - Batch Cost Query for Resource Stacks (P1)

**Goal**: Enable querying cost data for multiple resources in a single request-response
cycle across all three query types (estimate, actual, projected).

**Independent Test**: Send a batch request with 5-10 resources of mixed types and verify
cost results are returned for each resource in a single response.

### Implementation for User Story 1

- [x] T015 [US1] Wire `BatchCost` RPC to handler dispatch in
  `sdk/go/pluginsdk/connect.go`: check if plugin implements `BatchCostHandler`, if yes
  call it directly, if no invoke fallback (placeholder for US3). Include request
  validation via `ValidateBatchCostRequest()`.
- [x] T016 [US1] Add batch cost mock behavior to `MockPlugin` in
  `sdk/go/testing/mock_plugin.go`: add `BatchCostHandler` implementation that processes
  each resource using existing mock logic, add `ShouldErrorOnBatchCost` and
  `BatchCostDelay` configuration fields, add `UnsupportedBatchResourceTypes` map
- [x] T017 [US1] Add `BatchCost` integration tests to
  `sdk/go/testing/integration_test.go`: test batch estimate with 5 mixed-provider
  resources, test batch actual cost with time range and verify results contain
  `ActualCostData`, test batch projected cost and verify results contain `CostEstimate`
- [x] T018 [US1] Add batch conformance tests for basic batch in
  `sdk/go/testing/batch_conformance_test.go`: test batch with all three query types
  (ESTIMATE, ACTUAL, PROJECTED), test positional ordering (results[i] == resources[i]),
  test empty batch returns empty response, test UNSPECIFIED defaults to ESTIMATE
- [x] T019 [US1] Add validation tests for time range in
  `sdk/go/pluginsdk/batch_test.go`: test ACTUAL query without start/end returns error,
  test ACTUAL query with start >= end returns error, test ESTIMATE/PROJECTED queries
  ignore start/end fields

**Checkpoint**: Batch cost queries work for all three query types via custom handler.
Results are positionally ordered. Empty batches handled gracefully.

---

## Phase 4: User Story 2 - Partial Failure Handling (P1)

**Goal**: Per-resource error handling ensures individual failures do not prevent
successful resources from returning results.

**Independent Test**: Send a batch with a mix of supported and unsupported resource types
and verify supported resources return cost data while unsupported ones return structured
`ResourceError` entries.

### Implementation for User Story 2

- [x] T020 [US2] Implement per-resource error conversion in
  `sdk/go/pluginsdk/batch.go`: add `resourceErrorFromGRPCError()` that converts gRPC
  status errors to `ResourceError` messages preserving error codes, add
  `resourceErrorUnsupported()` for unsupported resource types
- [x] T021 [US2] Update `MockPlugin.BatchCost()` in `sdk/go/testing/mock_plugin.go`:
  add configurable per-resource error injection via `UnsupportedBatchResourceTypes` map,
  return `ResourceError` with `resource_type_unsupported=true` for unsupported types
- [x] T022 [US2] Add partial failure conformance tests in
  `sdk/go/testing/batch_conformance_test.go`: test 5 resources (3 supported, 2
  unsupported) returns 3 cost results and 2 errors, test all resources fail returns all
  errors (not top-level gRPC error), test transient error includes meaningful error code
  and message, test error codes match gRPC status codes (NOT_FOUND=5,
  UNIMPLEMENTED=12, INTERNAL=13, UNAVAILABLE=14)
- [x] T023 [US2] Add `ResourceError` field validation tests in
  `sdk/go/pluginsdk/batch_test.go`: test `NewResourceError()` sets code and message,
  test `resourceErrorUnsupported()` sets `resource_type_unsupported=true`,
  test `resourceErrorFromGRPCError()` preserves original status code

**Checkpoint**: Partial failure semantics work correctly. A batch with 80% supported
resources returns cost data for 80% and errors for 20%.

---

## Phase 5: User Story 3 - Plugin Fallback to Sequential Processing (P2)

**Goal**: Plugins without a custom `BatchCostHandler` can serve batch requests via
the SDK's automatic concurrent fallback.

**Independent Test**: Implement a plugin without `BatchCostHandler` and verify batch
requests are served correctly via automatic sequential fallback.

### Implementation for User Story 3

- [x] T024 [US3] Implement bounded parallelism worker pool in
  `sdk/go/pluginsdk/batch.go`: `batchCostFallback()` function that fans out to
  existing individual RPCs (`EstimateCost`, `GetActualCost`, `GetProjectedCost`) per
  resource using a `chan struct{}` semaphore with configurable concurrency, collect
  results in pre-allocated slice preserving request order (index-based), convert
  individual RPC errors to `ResourceError` entries (never fail the entire batch)
- [x] T025 [US3] Wire fallback into `BatchCost` RPC dispatch in
  `sdk/go/pluginsdk/connect.go`: when plugin does NOT implement `BatchCostHandler`,
  call `batchCostFallback()` instead of returning `Unimplemented`
- [x] T026 [US3] Add fallback conformance tests in
  `sdk/go/testing/batch_conformance_test.go`: test batch via fallback with plugin that
  does NOT implement `BatchCostHandler` returns correct results, test fallback preserves
  positional ordering, test fallback handles per-resource errors without failing batch,
  test fallback with configurable worker count (1 = sequential, 10 = concurrent)
- [x] T027 [US3] Add fallback vs custom handler conformance test in
  `sdk/go/testing/batch_conformance_test.go`: test that when plugin implements
  `BatchCostHandler`, the custom handler is called (not fallback), verify by checking
  custom handler side-effects

**Checkpoint**: Plugins without `BatchCostHandler` automatically support batch requests.
Fallback uses bounded parallelism and preserves all partial failure semantics.

---

## Phase 6: User Story 4 - Batch Size Limits and Capability Discovery (P2)

**Goal**: Hosts can discover batch support and maximum batch size before sending requests.
Oversized batches are rejected with clear errors.

**Independent Test**: Query plugin capabilities and verify batch support and size limit
are reported, then send an oversized batch and verify the error.

### Implementation for User Story 4

- [x] T028 [US4] Add `BatchCostHandler` detection to `inferCapabilities()` in
  `sdk/go/pluginsdk/plugin_info.go`: add type assertion for `BatchCostHandler` appending
  `PLUGIN_CAPABILITY_BATCH_COST`, update `maxCapabilities` from 8 to 9, update
  `maxValidCapability` from 11 to 12
- [x] T029 [US4] Add `max_batch_size` to `GetPluginInfo` metadata in
  `sdk/go/pluginsdk/plugin_info.go`: populate
  `"max_batch_size": strconv.Itoa(maxBatchSize)` in legacy metadata map when
  `BATCH_COST` capability is present, add `"supports_batch_cost": "true"` to
  `CapabilitiesToLegacyMetadataWithWarnings()` mapping
- [x] T030 [US4] Add batch size limit enforcement to `BatchCost` RPC dispatch in
  `sdk/go/pluginsdk/connect.go`: reject requests exceeding configured `MaxBatchSize`
  with `codes.InvalidArgument` and message including the limit value
- [x] T031 [US4] Add capability discovery conformance tests in
  `sdk/go/testing/batch_conformance_test.go`: test plugin with `BatchCostHandler`
  reports `PLUGIN_CAPABILITY_BATCH_COST` in capabilities, test plugin without
  `BatchCostHandler` does NOT report batch capability, test `GetPluginInfo` metadata
  includes `max_batch_size` and `supports_batch_cost` keys, test default max batch
  size is 100
- [x] T032 [US4] Add batch size limit conformance tests in
  `sdk/go/testing/batch_conformance_test.go`: test batch request with 101 resources
  (default limit 100) returns `InvalidArgument` error, test custom max batch size
  (e.g., 200) allows 150 resources, test batch at exactly the limit succeeds

**Checkpoint**: Hosts can discover batch support and limits. Oversized batches are
rejected with clear error messages.

---

## Phase 7: User Story 5 - Batch DryRun Support (P3)

**Goal**: Batch requests with `dry_run=true` return field mapping information per
resource type without querying external cost APIs.

**Independent Test**: Send a batch request with `dry_run=true` and verify field mapping
results are returned for each resource type.

### Implementation for User Story 5

- [x] T033 [US5] Implement dry_run handling in batch flow in
  `sdk/go/pluginsdk/batch.go`: when `req.DryRun` is true, invoke DryRun logic per
  resource type (reuse `HandleDryRun` from existing `DryRunHandler`), populate
  `CostData.DryRunResult` in `ResourceCostResult`, handle plugins without
  `DryRunHandler` by returning field mappings with UNSUPPORTED status
- [x] T034 [US5] Add dry_run batch support to `MockPlugin` in
  `sdk/go/testing/mock_plugin.go`: when batch `dry_run=true`, return
  `DryRunResult` per resource using existing mock field mapping configuration
- [x] T035 [US5] Add batch dry_run conformance tests in
  `sdk/go/testing/batch_conformance_test.go`: test batch with `dry_run=true`
  returns `DryRunResult` per resource, test dry_run results contain field mappings
  (not cost data), test dry_run with unsupported resource returns error or empty
  mappings, test dry_run with multiple resource types returns different mappings

**Checkpoint**: Batch dry_run works for field mapping discovery across multiple
resource types.

---

## Phase 8: TypeScript SDK Synchronization

**Purpose**: Keep TypeScript SDK in sync per Constitution XIII.

- [x] T036 [P] Run `buf generate` to regenerate TypeScript protobuf bindings in
  `sdk/typescript/`
- [x] T037 Add `batchCost()` method to `CostSourceClient` in
  `sdk/typescript/packages/client/src/clients/cost-source.ts`: wrap
  `this.client.batchCost(request)` with optional resource count validation
- [x] T038 [P] Create batch helper utilities in
  `sdk/typescript/packages/client/src/utils/batch.ts`: export `DEFAULT_MAX_BATCH_SIZE`,
  `MAX_BATCH_SIZE` constants, add `isBatchSupported()` helper that checks
  capabilities for `PLUGIN_CAPABILITY_BATCH_COST`
- [x] T039 Add TypeScript unit tests for `batchCost()` in
  `sdk/typescript/packages/client/src/__tests__/batch.test.ts`: test basic batch call,
  test empty batch, test partial failure response parsing
- [x] T040 Verify TypeScript build passes: `cd sdk/typescript && npm run build`

**Checkpoint**: TypeScript SDK exposes `batchCost()` with matching constants and helpers.

---

## Phase 9: Polish and Cross-Cutting Concerns

**Purpose**: Benchmarks, documentation, and full validation across all stories.

- [x] T041 [P] Add batch benchmarks to `sdk/go/testing/benchmark_test.go`:
  `BenchmarkBatchCostEstimate` (10, 50, 100 resources),
  `BenchmarkBatchCostActual` (with time range),
  `BenchmarkBatchCostFallback` (fallback vs custom handler comparison),
  `BenchmarkBatchCostConcurrent` (parallel batch requests)
- [x] T042 [P] Add `CostQueryType` validation benchmarks to
  `sdk/go/pluginsdk/batch_test.go`: verify zero-allocation (0 allocs/op) for
  `IsValidCostQueryType()`
- [x] T043 [P] Update `sdk/go/pluginsdk/README.md` with batch cost documentation:
  `BatchCostHandler` interface, `ServeConfig.MaxBatchSize` and `BatchWorkers` fields,
  fallback behavior description, code examples from quickstart.md
- [x] T044 [P] Update `sdk/go/testing/README.md` with batch testing documentation:
  batch conformance test descriptions, mock plugin batch configuration
- [x] T045 Run `make lint` to verify all Go linting passes (extended timeout)
- [x] T046 Run `make test` to verify all tests pass including new batch tests
- [x] T047 Run `make validate` to verify tests, linting, and npm validations pass
- [x] T048 Verify backward compatibility: run existing test suite without modifications
  to confirm all pre-existing tests still pass (SC-005)

**Checkpoint**: All benchmarks pass, documentation updated, full validation green.

---

## Dependencies and Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies - proto-first
- **Phase 2 (Foundational)**: Depends on Phase 1 (needs generated code)
- **Phase 3 (US1)**: Depends on Phase 2 (needs interfaces and helpers)
- **Phase 4 (US2)**: Depends on Phase 2 (uses error helpers); can run parallel to US1
- **Phase 5 (US3)**: Depends on Phase 3 (needs handler dispatch wired)
- **Phase 6 (US4)**: Depends on Phase 2 (uses constants); can run parallel to US1-US3
- **Phase 7 (US5)**: Depends on Phase 3 (needs batch flow working)
- **Phase 8 (TypeScript)**: Depends on Phase 1 (needs proto); can run parallel to US phases
- **Phase 9 (Polish)**: Depends on all user stories complete

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational only. No other story dependencies.
- **US2 (P1)**: Depends on Foundational only. Can run parallel with US1.
- **US3 (P2)**: Depends on US1 (needs handler dispatch). Cannot start until US1 complete.
- **US4 (P2)**: Depends on Foundational only. Can run parallel with US1-US3.
- **US5 (P3)**: Depends on US1 (needs batch flow). Cannot start until US1 complete.

### Parallel Opportunities

```text
Phase 1 (Setup) ──────────────────► sequential (proto changes must compile)
Phase 2 (Foundational) ───────────► T013, T014 can run parallel
Phase 3 (US1) + Phase 4 (US2) ───► can run in parallel (different concerns)
Phase 5 (US3) + Phase 6 (US4) ───► US4 can start parallel with US3
Phase 7 (US5) ────────────────────► after US1 complete
Phase 8 (TypeScript) ────────────► T036, T038 parallel; can start after Phase 1
Phase 9 (Polish) ─────────────────► T041, T042, T043, T044 all parallel
```

---

## Parallel Example: US1 + US2

```text
# After Phase 2 (Foundational) completes, launch US1 and US2 in parallel:

Agent A (US1 - Happy Path):
  T015: Wire BatchCost RPC dispatch in connect.go
  T016: Add BatchCost to MockPlugin in mock_plugin.go
  T017: Add integration tests in integration_test.go
  T018: Add basic conformance tests in batch_conformance_test.go
  T019: Add validation tests in batch_test.go

Agent B (US2 - Partial Failure):
  T020: Implement per-resource error conversion in batch.go
  T021: Update MockPlugin with error injection in mock_plugin.go
  T022: Add partial failure conformance tests in batch_conformance_test.go
  T023: Add ResourceError field validation tests in batch_test.go
```

---

## Implementation Strategy

### MVP First (US1 + US2 Only)

1. Complete Phase 1: Setup (proto definitions)
2. Complete Phase 2: Foundational (SDK infrastructure)
3. Complete Phase 3: US1 (batch cost query - happy path)
4. Complete Phase 4: US2 (partial failure handling)
5. **STOP and VALIDATE**: `make test && make lint`
6. MVP delivers core batch value with proper error handling

### Incremental Delivery

1. Setup + Foundational → Proto contract locked
2. US1 + US2 → Core batch with partial failure (MVP)
3. US3 → Fallback makes batch zero-effort for existing plugins
4. US4 → Capability discovery enables intelligent host behavior
5. US5 → DryRun extends batch to field mapping introspection
6. TypeScript + Polish → Full multi-language support with benchmarks

### Suggested MVP Scope

**US1 + US2 (both P1)** form the natural MVP:

- US1 provides the core batch RPC with all three query types
- US2 provides production-ready partial failure semantics
- Together they deliver immediate value to hosts querying multi-resource stacks
- Total: ~14 tasks (T001-T014 + T015-T023)

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story is independently completable and testable
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- `make lint` requires extended timeout (5+ minutes)
- Conformance tests sharing a `TestHarness` must NOT use `t.Parallel()` in subtests
