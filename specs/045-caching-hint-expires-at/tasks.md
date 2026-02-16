# Tasks: Caching Hint (expires_at) for Cost Results

**Input**: Design documents from `/specs/045-caching-hint-expires-at/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Organization**: Tasks are grouped by user story to enable independent implementation and testing
of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Setup (Proto Contract Changes)

**Purpose**: Add `expires_at` fields to proto definitions and regenerate SDK code.
This is the "Contract First" foundation that all user stories depend on.

- [x] T001 Add `expires_at` field (field 8) to `ActualCostResult` message with inline
  documentation comments per contracts/proto-changes.md in proto/finfocus/v1/costsource.proto
- [x] T002 Add `expires_at` field (field 13) to `GetProjectedCostResponse` message with inline
  documentation comments per contracts/proto-changes.md in proto/finfocus/v1/costsource.proto
- [x] T003 Run `make generate` to regenerate Go protobuf code in sdk/go/proto/finfocus/v1/
  and TypeScript bindings in sdk/typescript/packages/client/src/generated/
- [x] T004 Validate proto changes pass `buf lint` and `buf breaking` checks by running
  `make lint-go` (includes buf lint)

**Checkpoint**: Proto contract established. Generated code available for SDK implementation.

---

## Phase 2: User Story 1 - Plugin Signals Data Freshness Duration (Priority: P1)

**Goal**: Enable plugins to set `expires_at` on `ActualCostResult` records to signal data
freshness to callers. Includes mock plugin support and conformance testing.

**Independent Test**: Mock plugin returns cost results with `expires_at` set; conformance tests
verify the field round-trips correctly through gRPC transport.

### Implementation for User Story 1

**Test ordering note**: Mock plugin changes (T005-T006) precede conformance tests (T007-T010)
because the proto contract is already established in Phase 1 (Constitution V satisfied), and the
mock must populate the field for conformance tests to exercise the gRPC round-trip meaningfully.

- [x] T005 [US1] Add `ExpiresAtDuration` field (`time.Duration`) to `MockPlugin` struct in
  sdk/go/testing/mock_plugin.go with zero-value default (no expires_at set)
- [x] T006 [US1] Update `MockPlugin.GetActualCost` method to set `ExpiresAt` on each
  `ActualCostResult` when `ExpiresAtDuration` is non-zero using
  `timestamppb.New(time.Now().Add(m.ExpiresAtDuration))` in sdk/go/testing/mock_plugin.go
- [x] T007 [US1] Create conformance test `TestExpiresAtActualCost_RoundTrip` with copyright
  header that verifies `expires_at` set by mock plugin is readable by client after gRPC
  transport in sdk/go/testing/expires_at_conformance_test.go
- [x] T008 [US1] Create conformance test `TestExpiresAtActualCost_NilBackwardCompat` that
  verifies a mock plugin with zero `ExpiresAtDuration` returns nil `expires_at` on all results
  (backward compatibility) in sdk/go/testing/expires_at_conformance_test.go
- [x] T009 [US1] Create conformance test `TestExpiresAtActualCost_PastTimestamp` that verifies
  a mock plugin can set `expires_at` to a past timestamp and the field is preserved through
  gRPC transport in sdk/go/testing/expires_at_conformance_test.go
- [x] T010 [US1] Create conformance test `TestExpiresAtActualCost_PerResultIndependence` that
  verifies each `ActualCostResult` in a paginated response carries its own independent
  `expires_at` value in sdk/go/testing/expires_at_conformance_test.go
- [x] T011 [US1] Run `go test -v -run TestExpiresAtActualCost ./sdk/go/testing/` to verify
  all US1 conformance tests pass

**Checkpoint**: Plugins can set `expires_at` on actual cost results. Mock plugin supports
configurable expiration. Conformance tests verify gRPC round-trip.

---

## Phase 3: User Story 2 - Caller Expiration Check Helpers (Priority: P2)

**Goal**: Provide SDK helper functions for callers to check whether cost data has expired,
enabling local cache management decisions.

**Independent Test**: Unit tests verify expiration check helpers return correct results for
nil, past, and future `expires_at` values.

### Implementation for User Story 2

- [x] T012 [P] [US2] Create `sdk/go/pluginsdk/expires_at.go` with copyright header and package
  declaration, implementing `IsActualCostExpired(result *pbc.ActualCostResult, now time.Time) bool`
  per contracts/sdk-helpers.md. Handle nil result (return false) and nil expires_at (return false).
  A zero `time.Time` in expires_at should be treated as nil/unset (no caching guidance).
- [x] T013 [P] [US2] Add `ActualCostExpiresAt(result *pbc.ActualCostResult) (time.Time, bool)`
  function to sdk/go/pluginsdk/expires_at.go per contracts/sdk-helpers.md
- [x] T014 [P] [US2] Add `IsProjectedCostExpired(resp *pbc.GetProjectedCostResponse, now time.Time) bool`
  function to sdk/go/pluginsdk/expires_at.go per contracts/sdk-helpers.md
- [x] T015 [P] [US2] Add `ProjectedCostExpiresAt(resp *pbc.GetProjectedCostResponse) (time.Time, bool)`
  function to sdk/go/pluginsdk/expires_at.go per contracts/sdk-helpers.md
- [x] T016 [US2] Create `sdk/go/pluginsdk/expires_at_test.go` with copyright header and
  table-driven unit tests for all four helper functions covering: nil result, nil expires_at,
  past timestamp, future timestamp, zero-value timestamp, and exact boundary time
- [x] T017 [US2] Run `go test -v -run TestExpires ./sdk/go/pluginsdk/` to verify all US2 unit
  tests pass
- [x] T018 [US2] Run `goimports -w sdk/go/pluginsdk/expires_at.go sdk/go/pluginsdk/expires_at_test.go`
  to ensure import formatting

**Checkpoint**: Callers have helper functions to check expiration status. All edge cases tested.

---

## Phase 4: User Story 3 - Plugin Signals Pricing Cycle Boundary (Priority: P3)

**Goal**: Enable plugins to set `expires_at` on projected cost responses to signal when
pricing data should be refreshed (e.g., at billing cycle boundaries).

**Independent Test**: Mock plugin returns projected cost response with `expires_at` set;
conformance tests verify the field round-trips correctly.

### Implementation for User Story 3

- [x] T019 [US3] Add `WithProjectedCostExpiresAt(expiresAt time.Time) GetProjectedCostResponseOption`
  functional option to sdk/go/pluginsdk/helpers.go following existing pattern
  (e.g., `WithPredictionInterval`)
- [x] T020 [US3] Add `ProjectedCostExpiresAtDuration` field (`time.Duration`) to `MockPlugin`
  struct in sdk/go/testing/mock_plugin.go
- [x] T021 [US3] Update `MockPlugin.GetProjectedCost` method to set `ExpiresAt` on the response
  when `ProjectedCostExpiresAtDuration` is non-zero in sdk/go/testing/mock_plugin.go
- [x] T022 [US3] Create conformance test `TestExpiresAtProjectedCost_RoundTrip` that verifies
  `expires_at` set on projected cost response is readable by client in
  sdk/go/testing/expires_at_conformance_test.go
- [x] T023 [US3] Create conformance test `TestExpiresAtProjectedCost_NilBackwardCompat` that
  verifies a mock plugin with zero `ProjectedCostExpiresAtDuration` returns nil `expires_at`
  in sdk/go/testing/expires_at_conformance_test.go
- [x] T024 [US3] Create conformance test `TestExpiresAtProjectedCost_WithOptionBuilder` that
  verifies `WithProjectedCostExpiresAt` option correctly sets the field on
  `NewGetProjectedCostResponse` in sdk/go/testing/expires_at_conformance_test.go
- [x] T025 [US3] Run `go test -v -run TestExpiresAtProjectedCost ./sdk/go/testing/` to verify
  all US3 conformance tests pass

**Checkpoint**: Plugins can signal pricing cycle boundaries. Both actual and projected cost
types now support caching hints.

---

## Phase 5: Polish and Cross-Cutting Concerns

**Purpose**: Benchmarks, documentation, and full validation across all user stories

- [x] T026 [P] Add benchmark `BenchmarkExpiresAtHelpers` with sub-benchmarks for all four
  expiration check helpers (target: < 10 ns/op, 0 allocs/op) in
  sdk/go/testing/benchmark_test.go
- [x] T027 [P] Add benchmark `BenchmarkActualCostWithExpiresAt` measuring GetActualCost RPC
  overhead when `expires_at` is set vs not set in sdk/go/testing/benchmark_test.go
- [x] T028 [P] Verify godoc coverage is >80% for all new exported functions in
  sdk/go/pluginsdk/expires_at.go and sdk/go/pluginsdk/helpers.go
- [x] T029 [P] Update sdk/go/pluginsdk/README.md to document `expires_at` helpers with usage
  examples from quickstart.md
- [x] T030 Run full test suite with `make test` to verify no regressions across all packages
- [x] T031 Run full lint suite with `make lint` to verify no linting violations (extended
  timeout, 5+ minutes)
- [x] T032 Run `npx markdownlint-cli2` on all new and modified markdown files to verify
  markdown formatting

---

## Dependencies and Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - must start here (proto contract first)
- **US1 (Phase 2)**: Depends on Phase 1 completion (needs generated proto types)
- **US2 (Phase 3)**: Depends on Phase 1 completion (needs generated proto types).
  Can run in parallel with Phase 2 (different files).
- **US3 (Phase 4)**: Depends on Phase 1 completion (needs generated proto types).
  Can run in parallel with Phases 2 and 3 (different file areas).
- **Polish (Phase 5)**: Depends on Phases 2, 3, and 4 completion

### User Story Dependencies

- **US1 (P1)**: Proto types only. No dependency on US2 or US3.
- **US2 (P2)**: Proto types only. Helpers work independently of mock plugin (US1)
  and projected cost option (US3). Can be implemented in parallel with US1.
- **US3 (P3)**: Proto types only. No dependency on US1 or US2.

### Within Each User Story

- Mock plugin changes before conformance tests (US1, US3)
- Helper function implementation before unit tests (US2)
- All implementation before verification step (run tests)

### Parallel Opportunities

```text
Phase 1: T001 → T002 → T003 → T004 (sequential - same file, then generate)

Phase 2+3+4 can run in parallel after Phase 1:
  Phase 2: T005 → T006 → T007,T008,T009,T010 (parallel conformance tests) → T011
  Phase 3: T012,T013,T014,T015 (all parallel - same new file) → T016 → T017 → T018
  Phase 4: T019 → T020 → T021 → T022,T023,T024 (parallel conformance tests) → T025

Phase 5: T026,T027,T028,T029 (all parallel) → T030 → T031 → T032
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Proto changes + code generation
2. Complete Phase 2: US1 - ActualCostResult expires_at with mock + conformance
3. **STOP and VALIDATE**: Run `go test -v ./sdk/go/testing/ -run TestExpiresAtActualCost`
4. Feature is usable for the primary use case (rate-limit management)

### Incremental Delivery

1. Phase 1 (Proto) → Contract established
2. Phase 2 (US1) → Plugins can set expires_at on actual costs
3. Phase 3 (US2) → Callers have helper functions for cache management
4. Phase 4 (US3) → Projected cost responses also support expires_at
5. Phase 5 (Polish) → Benchmarks, docs, full validation

### Parallel Execution (3 developers)

1. All complete Phase 1 together (4 sequential tasks)
2. After Phase 1:
   - Developer A: Phase 2 (US1 - mock plugin + conformance)
   - Developer B: Phase 3 (US2 - expiration check helpers)
   - Developer C: Phase 4 (US3 - projected cost option + conformance)
3. All complete Phase 5 (Polish) together

---

## Notes

- [P] tasks = different files, no dependencies on incomplete tasks
- [Story] label maps task to specific user story for traceability
- Conformance tests in sdk/go/testing/ must NOT use `t.Parallel()` on subtests sharing a TestHarness
- Run `goimports -w` after creating new Go files
- `make lint` can take 5+ minutes; use extended timeout
- Proto field 8 (ActualCostResult) and field 13 (GetProjectedCostResponse) per research.md
