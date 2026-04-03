# Tasks: Integrate ValidationError with Validation Implementation

**Input**: Design documents from `/specs/047-validation-error-integration/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Included per Constitution Principle V (Test-First Protocol).

**Organization**: Tasks are grouped by user story to enable independent implementation
and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Phase 1: Foundational (ValidationError Type Changes)

**Purpose**: Extend `ValidationError` with error chain support. MUST complete before
any user story work begins.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [x] T001 Add failing tests for `Unwrap()`, `errors.Is` chain traversal, and
  `NewValidationErrorWithCause` in `sdk/go/pluginsdk/validation_error_test.go`
- [x] T002 Add unexported `err` field and `Unwrap()` method to `ValidationError`
  struct in `sdk/go/pluginsdk/validation_error.go`
- [x] T003 Add `NewValidationErrorWithCause` constructor in
  `sdk/go/pluginsdk/validation_error.go`

**Checkpoint**: `ValidationError` supports error wrapping and chain traversal.
Run `go test ./sdk/go/pluginsdk/ -run TestValidationError` to verify.

---

## Phase 2: User Story 1 - Extract Structured Error Details (Priority: P1) 🎯 MVP

**Goal**: All 24 validation errors returned by `ValidateFocusRecord` and
`ValidateFocusRecordWithOptions` are `*ValidationError` instances extractable
via `errors.As`.

**Independent Test**: Call `ValidateFocusRecord` with a record violating cost
hierarchy rules, then use `errors.As(err, &valErr)` to extract field-level details.

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [x] T004 [P] [US1] Write test in `sdk/go/pluginsdk/focus_conformance_test.go` verifying
  `errors.As(err, &valErr)` extracts `FieldName` from `ValidateFocusRecord` output
  for sentinel error cases (cost hierarchy, commitment discount, pricing unit)
- [x] T005 [P] [US1] Write test in `sdk/go/pluginsdk/focus_conformance_test.go` verifying
  `errors.As(err, &valErr)` extracts `FieldName` from `ValidateFocusRecord` output
  for ad-hoc error cases (mandatory fields, currency, NaN/Inf, record-is-nil)

### Implementation for User Story 1

- [x] T006 [US1] Convert 7 sentinel error return sites to `*ValidationError` wrapping
  sentinels via `NewValidationErrorWithCause` in
  `sdk/go/pluginsdk/focus_conformance.go` (see data-model.md Sentinel Error Wrapping
  table for field mappings)
- [x] T007 [US1] Convert 12 mandatory field error sites to `*ValidationError` via
  `NewValidationError` in `sdk/go/pluginsdk/focus_conformance.go` function
  `validateMandatoryFields`
- [x] T008 [US1] Convert 2 cost value error sites (NaN/Inf) to `*ValidationError` in
  `sdk/go/pluginsdk/focus_conformance.go` function `checkCostValue`
- [x] T009 [US1] Convert 5 remaining error sites to `*ValidationError` in
  `sdk/go/pluginsdk/focus_conformance.go`: `ValidateFocusRecordWithOptions` nil guard,
  `validateContractedCostRule`, `validateAllocationRule`, `validateCurrency`, and
  consumed quantity check in `validateBusinessRulesWithOptions`
- [x] T010 [US1] Update existing test assertions in
  `sdk/go/pluginsdk/focus_conformance_test.go` for new `Error()` format
  (ValidationError structured format replaces bare sentinel/string messages)

**Checkpoint**: All `ValidateFocusRecord` errors support `errors.As` extraction.
Run `go test ./sdk/go/pluginsdk/ -run TestValidateFocusRecord` to verify.

---

## Phase 3: User Story 2 - Aggregate Mode Rich Error Collection (Priority: P2)

**Goal**: Aggregate validation mode returns `*ValidationError` for every failure,
enabling programmatic categorization by field name without string parsing.

**Independent Test**: Call `ValidateFocusRecordWithOptions` in aggregate mode with
a record containing multiple violations; verify each error unwraps to `ValidationError`.

### Tests for User Story 2

- [x] T011 [US2] Write test in `sdk/go/pluginsdk/focus_conformance_test.go` verifying
  aggregate mode returns multiple `*ValidationError` instances with distinct `FieldName`
  values when a record has 3+ violations

### Implementation for User Story 2

- [x] T012 [US2] Verify aggregate mode paths in
  `sdk/go/pluginsdk/focus_conformance.go` propagate `*ValidationError` correctly
  through `ValidateFocusRecordWithOptions` (no additional code changes expected
  if US1 conversion is complete; fix any gaps found)

**Checkpoint**: Aggregate mode errors all support `errors.As` extraction.
Run `go test ./sdk/go/pluginsdk/ -run TestValidateFocusRecordWithOptions` to verify.

---

## Phase 4: User Story 3 - Error Chain Unwrapping (Priority: P2)

**Goal**: `errors.Is(err, sentinel)` works on `*ValidationError` returned by
validation functions, preserving backward compatibility for all 7 sentinel errors.

**Independent Test**: Validate a record with a cost hierarchy violation, then verify
both `errors.Is(err, ErrEffectiveCostExceedsBilledCost)` and `errors.As(err, &valErr)`
work on the same returned error.

### Tests for User Story 3

- [x] T013 [US3] Write test in `sdk/go/pluginsdk/focus_conformance_test.go` verifying
  `errors.Is(err, sentinel)` returns `true` for all 7 sentinel errors when returned
  by `ValidateFocusRecord`

### Implementation for User Story 3

- [x] T014 [US3] Verify all 7 sentinel wrapping sites use `NewValidationErrorWithCause`
  with the correct sentinel as `cause` parameter in
  `sdk/go/pluginsdk/focus_conformance.go` (no additional code changes expected
  if US1 T006 is complete; fix any gaps found)

**Checkpoint**: All 7 sentinel errors reachable via `errors.Is` through `ValidationError`.
Run `go test ./sdk/go/pluginsdk/ -run TestValidationError` to verify.

---

## Phase 5: Polish and Cross-Cutting Concerns

**Purpose**: Performance validation, documentation, and final quality checks.

- [x] T015 [P] Add benchmark for `ValidationError` creation and `Unwrap()` in
  `sdk/go/pluginsdk/validation_error_test.go` (target: < 100 ns/op, 1 alloc/op)
- [x] T016 [P] Update godoc comments on `ValidationError` struct and methods in
  `sdk/go/pluginsdk/validation_error.go` to document `Unwrap()` and error chain usage;
  verify Apache 2.0 copyright headers are preserved on all modified `.go` files
- [x] T017 [P] Update `sdk/go/pluginsdk/README.md` with validation error extraction
  examples (see quickstart.md for before/after patterns)
- [x] T018 Run `make lint` to verify no linting regressions
- [x] T019 Run `make test` to verify all tests pass across the entire project
- [x] T020 Run `make lint-markdown` to verify markdown files pass linting

---

## Dependencies and Execution Order

### Phase Dependencies

- **Foundational (Phase 1)**: No dependencies — start immediately
- **US1 (Phase 2)**: Depends on Phase 1 completion (needs `Unwrap()` and new constructor)
- **US2 (Phase 3)**: Depends on Phase 2 completion (needs converted error sites)
- **US3 (Phase 4)**: Depends on Phase 2 T006 (needs sentinel wrapping)
- **Polish (Phase 5)**: Depends on all user stories being complete

### User Story Dependencies

- **US1 (P1)**: Depends on Foundational only. Core implementation story.
- **US2 (P2)**: Depends on US1 (aggregate mode uses same converted error sites).
- **US3 (P2)**: Depends on US1 T006 specifically (sentinel wrapping).
  Can run in parallel with US2 once T006 is complete.

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Sentinel conversion (T006) before ad-hoc conversion (T007–T009)
- Implementation before test assertion updates (T010)

### Parallel Opportunities

- T001 can start immediately (tests for foundational type)
- T004 and T005 (US1 tests) can be written in parallel (different test functions)
- T006–T009 are sequential (same file, different functions within focus_conformance.go)
- T011 (US2 test) and T013 (US3 test) can start once T006 is complete
- T015, T016, T017 (Polish) can all run in parallel (different files)

---

## Parallel Example: Phase 2 (User Story 1)

```text
# Step 1: Write failing tests in parallel
Task T004: "errors.As test for sentinel errors in focus_conformance_test.go"
Task T005: "errors.As test for ad-hoc errors in focus_conformance_test.go"

# Step 2: Convert error sites sequentially (same file)
Task T006: "Convert 7 sentinel error sites"
Task T007: "Convert 10 mandatory field sites"
Task T008: "Convert 2 NaN/Inf sites"
Task T009: "Convert 5 remaining sites (nil guard + business rules)"

# Step 3: Update existing tests
Task T010: "Update test assertions for new Error() format"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Foundational (T001–T003)
2. Complete Phase 2: User Story 1 (T004–T010)
3. **STOP and VALIDATE**: `errors.As` works on all validation errors
4. This alone delivers the core value proposition

### Incremental Delivery

1. Phase 1: Foundational (T001–T003) → `ValidationError` has `Unwrap()`
2. Phase 2: US1 (T004–T010) → All errors are `*ValidationError` (MVP)
3. Phase 3: US2 (T011–T012) → Aggregate mode verified
4. Phase 4: US3 (T013–T014) → `errors.Is` backward compat verified
5. Phase 5: Polish (T015–T020) → Benchmarks, docs, final validation

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Constitution V (Test-First) requires tests before implementation
- 24 total error return sites to convert (7 sentinel + 17 ad-hoc, see data-model.md)
- `manifest.go` uses `pbc.ValidationError` (protobuf type) — NOT affected by these changes
- Commit after each phase completion
- Stop at any checkpoint to validate independently
