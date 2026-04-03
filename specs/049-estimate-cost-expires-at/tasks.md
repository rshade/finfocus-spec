# Tasks: EstimateCost expires_at Cache-Hint Parity

**Input**: Design documents from `/specs/049-estimate-cost-expires-at/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: Included per Constitution V (Test-First Protocol is NON-NEGOTIABLE).
TDD approach: write tests first, verify they fail, then implement.

**Organization**: Tasks grouped by user story for independent implementation
and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to
- Include exact file paths in descriptions

## Phase 1: Setup

**Purpose**: No new project structure needed. Extends existing files.

<!-- markdownlint-disable MD013 -->

- [x] T001 Verify `EstimateCostResponse` field numbers
  in `proto/finfocus/v1/costsource.proto` (confirm field 5 available)
- [x] T002 Review existing `expires_at` patterns in
  `sdk/go/pluginsdk/expires_at.go` and
  `sdk/go/testing/expires_at_conformance_test.go`

<!-- markdownlint-enable MD013 -->

---

## Phase 2: Foundational (Proto Contract Change)

**Purpose**: Add the `expires_at` field to `EstimateCostResponse` in the
proto definition and regenerate all code. MUST complete before any SDK or
test work begins (Constitution I: Contract First).

**CRITICAL**: No user story work can begin until this phase is complete.

<!-- markdownlint-disable MD013 -->

- [x] T003 Add `google.protobuf.Timestamp expires_at = 5` field with
  documentation comment to `EstimateCostResponse` in
  `proto/finfocus/v1/costsource.proto`
  (see `contracts/proto-changes.md` for exact comment text)
- [x] T004 Run `make generate` to regenerate Go and TypeScript protobuf
  bindings
- [x] T005 Run `buf breaking` to verify backward compatibility
- [x] T006 Verify generated Go code has `ExpiresAt *timestamppb.Timestamp`
  on `EstimateCostResponse` in
  `sdk/go/proto/finfocus/v1/costsource.pb.go`

<!-- markdownlint-enable MD013 -->

**Checkpoint**: Proto contract updated. Generated code available.

---

## Phase 3: US1 - Plugin Author Sets Cache Hint (Priority: P1)

**Goal**: Plugin author sets `expires_at` on `EstimateCostResponse` and
caller receives the timestamp via gRPC.

**Independent Test**: Mock plugin sets `expires_at` with positive duration,
caller reads it as future timestamp. Zero duration produces nil.

### Tests for User Story 1

> **NOTE: Write tests FIRST, ensure they FAIL before implementation**

<!-- markdownlint-disable MD013 -->

- [x] T007 [P] [US1] Write `TestExpiresAtEstimateCost_RoundTrip` in
  `sdk/go/testing/expires_at_conformance_test.go` — mock with
  `EstimateCostExpiresAtDuration = 1h`, assert non-nil future timestamp
- [x] T008 [P] [US1] Write `TestExpiresAtEstimateCost_NilBackwardCompat` in
  `sdk/go/testing/expires_at_conformance_test.go` — mock with zero
  duration, assert `resp.ExpiresAt` is nil (also covers US1 scenario 3:
  older plugins without `expires_at` produce nil for newer callers)
- [x] T009 [P] [US1] Write `TestExpiresAtEstimateCost_PastTimestamp` in
  `sdk/go/testing/expires_at_conformance_test.go` — mock with
  `EstimateCostExpiresAtDuration = -1h`, assert past timestamp

### Implementation for User Story 1

- [x] T010 [US1] Add `EstimateCostExpiresAtDuration time.Duration` field
  to `MockPlugin` struct in `sdk/go/testing/mock_plugin.go`
  (after `ProjectedCostExpiresAtDuration`, same pattern)
- [x] T011 [US1] Update `MockPlugin.EstimateCost()` in
  `sdk/go/testing/mock_plugin.go` — set `resp.ExpiresAt` when duration
  is non-zero (same pattern as `GetProjectedCost` lines 1227-1230)
- [x] T012 [US1] Run `go test -v -run TestExpiresAtEstimateCost
  ./sdk/go/testing/` — verify T007-T009 pass

<!-- markdownlint-enable MD013 -->

**Checkpoint**: Plugin authors can set `expires_at` on estimates.

---

## Phase 4: US2 - Caller Applies Uniform Caching Policy (Priority: P2)

**Goal**: Callers use `EstimateCostExpiresAt()` and
`IsEstimateCostExpired()` helpers for uniform caching across all cost RPCs.

**Independent Test**: Helpers return correct results for set, unset, and
past timestamps. Nil response handling (no panics).

### Tests for User Story 2

> **NOTE: Write tests FIRST, ensure they FAIL before implementation**

<!-- markdownlint-disable MD013 -->

- [x] T013 [P] [US2] Write `TestEstimateCostExpiresAt_Set` in
  `sdk/go/pluginsdk/expires_at_test.go` — response with future
  `expires_at`, assert returns time and `true`
- [x] T014 [P] [US2] Write `TestEstimateCostExpiresAt_Unset` in
  `sdk/go/pluginsdk/expires_at_test.go` — response with nil
  `expires_at`, assert returns zero time and `false`
- [x] T015 [P] [US2] Write `TestEstimateCostExpiresAt_NilResponse` in
  `sdk/go/pluginsdk/expires_at_test.go` — nil response, assert
  returns zero time and `false` (no panic)
- [x] T016 [P] [US2] Write `TestIsEstimateCostExpired_Future` in
  `sdk/go/pluginsdk/expires_at_test.go` — future `expires_at`,
  assert returns `false`
- [x] T017 [P] [US2] Write `TestIsEstimateCostExpired_Past` in
  `sdk/go/pluginsdk/expires_at_test.go` — past `expires_at`,
  assert returns `true`
- [x] T018 [P] [US2] Write `TestIsEstimateCostExpired_NilResponse` in
  `sdk/go/pluginsdk/expires_at_test.go` — nil response,
  assert returns `false` (no panic)
- [x] T019 [P] [US2] Write `TestIsEstimateCostExpired_NilExpiresAt` in
  `sdk/go/pluginsdk/expires_at_test.go` — nil `expires_at`,
  assert returns `false`
- [x] T020 [P] [US2] Write `TestIsEstimateCostExpired_UnixEpoch` in
  `sdk/go/pluginsdk/expires_at_test.go` — `expires_at` set to
  Unix epoch (1970-01-01), assert returns `true` (always in past)

### Implementation for User Story 2

- [x] T021 [P] [US2] Implement `EstimateCostExpiresAt()` in
  `sdk/go/pluginsdk/expires_at.go` — nil-safe reader returning
  `(time.Time, bool)` (mirror `ProjectedCostExpiresAt`)
- [x] T022 [P] [US2] Implement `IsEstimateCostExpired()` in
  `sdk/go/pluginsdk/expires_at.go` — nil-safe checker returning
  bool (mirror `IsProjectedCostExpired`)
- [x] T023 [US2] Run `go test -v -run "TestEstimateCostExpiresAt|
  TestIsEstimateCostExpired" ./sdk/go/pluginsdk/` — verify T013-T020

<!-- markdownlint-enable MD013 -->

**Checkpoint**: Callers check estimate expiration like actual/projected.

---

## Phase 5: US3 - Plugin Author Uses Functional Option (Priority: P3)

**Goal**: Plugin authors use `WithEstimateCostExpiresAt()` for type-safe
`expires_at` setting via functional option pattern.

**Independent Test**: Apply option to response, verify field is set.
Apply with zero time, verify field is nil.

### Tests for User Story 3

> **NOTE: Write tests FIRST, ensure they FAIL before implementation**

<!-- markdownlint-disable MD013 -->

- [x] T024 [P] [US3] Write `TestWithEstimateCostExpiresAt_Set` in
  `sdk/go/pluginsdk/expires_at_test.go` — apply option via
  `NewEstimateCostResponse()`, assert `ExpiresAt` is non-nil
- [x] T025 [P] [US3] Write `TestWithEstimateCostExpiresAt_ZeroTime` in
  `sdk/go/pluginsdk/expires_at_test.go` — apply with `time.Time{}`,
  assert `ExpiresAt` is nil

### Implementation for User Story 3

- [x] T026 [US3] Implement `WithEstimateCostExpiresAt()` in
  `sdk/go/pluginsdk/expires_at.go` — returns
  `EstimateCostResponseOption`, zero time clears to nil
  (mirror `WithProjectedCostExpiresAt` from `helpers.go`)
- [x] T027 [US3] Run `go test -v -run TestWithEstimateCostExpiresAt
  ./sdk/go/pluginsdk/` — verify T024-T025 pass

<!-- markdownlint-enable MD013 -->

**Checkpoint**: All user stories complete. Full `expires_at` parity.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Benchmarks, full validation, and documentation verification.

<!-- markdownlint-disable MD013 -->

- [x] T028 [P] Add `BenchmarkEstimateCostExpiresAt` and
  `BenchmarkIsEstimateCostExpired` in
  `sdk/go/testing/benchmark_test.go` — verify < 10 ns/op, 0 allocs
- [x] T029 [P] Add `BenchmarkWithEstimateCostExpiresAt` in
  `sdk/go/testing/benchmark_test.go` — verify option performance
- [x] T030 Run `make validate` — all tests, linting, npm validations
- [x] T031 Run `golangci-lint run ./...` — no lint errors on new code
- [x] T032 Verify godoc comments on all new exported functions in
  `sdk/go/pluginsdk/expires_at.go`
- [x] T033 Run `go test -bench=. -benchmem ./sdk/go/testing/` — confirm
  benchmarks meet < 10 ns/op target
- [x] T034 Validate `quickstart.md` examples match implemented signatures

<!-- markdownlint-enable MD013 -->

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — verification only
- **Foundational (Phase 2)**: Depends on Setup — BLOCKS all user stories
- **US1 (Phase 3)**: Depends on Foundational (proto field must exist)
- **US2 (Phase 4)**: Depends on Foundational — independent of US1
- **US3 (Phase 5)**: Depends on Foundational — independent of US1/US2
- **Polish (Phase 6)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: After Foundational — no dependencies on other stories
- **US2 (P2)**: After Foundational — independent of US1
- **US3 (P3)**: After Foundational — independent of US1/US2

### Within Each User Story

- Tests MUST be written and FAIL before implementation (Constitution V)
- Implementation makes failing tests pass
- Verification run confirms all tests green

### Parallel Opportunities

- T007, T008, T009 can run in parallel (separate test functions)
- T013-T020 can all run in parallel (separate test functions)
- T024, T025 can run in parallel (separate test functions)
- T021, T022 can run in parallel (independent functions)
- T028, T029 can run in parallel (independent benchmarks)
- US2 and US3 can start in parallel after Foundational phase

---

## Parallel Example: User Story 2

```text
# Launch all unit tests for US2 together (they all fail initially):
Task: "Write TestEstimateCostExpiresAt_Set"
Task: "Write TestEstimateCostExpiresAt_Unset"
Task: "Write TestEstimateCostExpiresAt_NilResponse"
Task: "Write TestIsEstimateCostExpired_Future"
Task: "Write TestIsEstimateCostExpired_Past"
Task: "Write TestIsEstimateCostExpired_NilResponse"
Task: "Write TestIsEstimateCostExpired_NilExpiresAt"
Task: "Write TestIsEstimateCostExpired_UnixEpoch"

# Then launch both implementations in parallel:
Task: "Implement EstimateCostExpiresAt"
Task: "Implement IsEstimateCostExpired"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (verification)
2. Complete Phase 2: Foundational (proto change + regenerate)
3. Complete Phase 3: User Story 1 (mock plugin + conformance tests)
4. **STOP and VALIDATE**: `go test -v -run TestExpiresAtEstimateCost`
5. Proto field is usable — callers read `resp.ExpiresAt` directly

### Incremental Delivery

1. Setup + Foundational -> Proto field available
2. User Story 1 -> Mock plugin supports expires_at -> Test
3. User Story 2 -> Reader/checker helpers -> Test
4. User Story 3 -> Functional option -> Test
5. Polish -> Benchmarks, full validation, docs check
6. Each story adds SDK ergonomics without breaking previous

### Parallel Team Strategy

After Foundational phase:

- Developer A: US1 (mock plugin in `sdk/go/testing/`)
- Developer B: US2 + US3 (SDK helpers in `sdk/go/pluginsdk/`)
- Stories complete and validate independently

---

## Notes

- [P] tasks = different files or independent functions
- [Story] label maps task to user story for traceability
- Each user story is independently completable and testable
- Verify tests fail before implementing (Constitution V)
- Commit after each phase completion
- Stop at any checkpoint to validate independently
