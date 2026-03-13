# Tasks: Extract Test Helper for ResourceDescriptor Creation

**Input**: Design documents from `/specs/048-test-descriptor-helper/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, quickstart.md

**Tests**: Not explicitly requested. Validation is via existing test suite
(`go test ./sdk/go/pluginsdk/...`) — all tests must pass identically before
and after changes.

**Organization**: Tasks are grouped by user story. US1 (helper creation)
must complete before US2 (refactoring).

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Baseline Verification)

**Purpose**: Capture baseline test state before any changes

- [x] T001 Run `go test ./sdk/go/pluginsdk/...` and confirm all tests
  pass (baseline)
- [x] T002 Record current line count of
  `sdk/go/pluginsdk/helpers_test.go` for SC-002 measurement

**Checkpoint**: Baseline captured — implementation can begin

---

## Phase 2: User Story 1 — SDK Developer Writes New Tests (Priority: P1) MVP

**Goal**: Create the `newTestDescriptor` helper function so SDK developers
can quickly create `ResourceDescriptor` instances with sensible defaults
for new tests.

**Independent Test**: Call `newTestDescriptor("test-id", "test-arn")` and
assert the result matches an equivalent verbose `NewResourceDescriptor`
call with the same defaults.

### Implementation for User Story 1

- [x] T003 [US1] Add `newTestDescriptor` helper function to
  `sdk/go/pluginsdk/helpers_test.go` with signature
  `func newTestDescriptor(id, arn string, opts ...ResourceDescriptorOption)`
- [x] T004 [US1] Implement base defaults in `newTestDescriptor`:
  provider=`"aws"`, resourceType=`"ec2"`, sku=`"t3.micro"` via `WithSKU`,
  region=`"us-east-1"` via `WithRegion` in
  `sdk/go/pluginsdk/helpers_test.go`
- [x] T005 [US1] Implement conditional ID/ARN application: only call
  `WithID(id)` when `id != ""` and `WithARN(arn)` when `arn != ""` in
  `sdk/go/pluginsdk/helpers_test.go`
- [x] T006 [US1] Implement option ordering: append caller-provided `opts`
  after base options so overrides work correctly in
  `sdk/go/pluginsdk/helpers_test.go`
- [x] T007 [US1] Run `go test ./sdk/go/pluginsdk/...` and confirm all
  tests still pass after adding the helper

**Checkpoint**: Helper function exists and is usable. All existing tests
pass. User Story 1 is independently complete — a developer can now use
`newTestDescriptor` in new tests.

---

## Phase 3: User Story 2 — Existing Tests Refactored (Priority: P2)

**Goal**: Refactor existing `NewResourceDescriptor` calls that use the
common `"aws", "ec2"` pattern to use `newTestDescriptor`, reducing
boilerplate and improving consistency.

**Independent Test**: Run the full test suite before and after
refactoring — all tests must pass with identical behavior (SC-001).

### Implementation for User Story 2

- [x] T008 [US2] Identify all `NewResourceDescriptor` callsites in
  `sdk/go/pluginsdk/helpers_test.go` that use `"aws", "ec2"` pattern
  and are NOT testing construction behavior per FR-008 and R-004
- [x] T009 [US2] Refactor eligible callsites to use `newTestDescriptor`
  in `sdk/go/pluginsdk/helpers_test.go` — preserve exact test
  semantics (FR-007)
- [x] T010 [US2] Verify NO refactoring was applied to tests listed in
  R-004 exclusions (Basic, WithIDOption, WithARNOption, WithAllOptions,
  WithID table tests, WithARN table tests, Composability,
  FuzzResourceDescriptorID) in `sdk/go/pluginsdk/helpers_test.go`
- [x] T011 [US2] Run `go test ./sdk/go/pluginsdk/...` and confirm all
  tests pass identically after refactoring

**Checkpoint**: Refactoring complete. All tests pass. Line count reduced
in refactored sections (SC-002, SC-003).

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and quality checks

- [x] T012 Run `golangci-lint run ./sdk/go/pluginsdk/...` and fix any
  lint issues in `sdk/go/pluginsdk/helpers_test.go`
- [x] T013 Verify no new exported symbols were introduced (SC-004) —
  `newTestDescriptor` must remain unexported in
  `sdk/go/pluginsdk/helpers_test.go`
- [x] T014 Compare line count of `sdk/go/pluginsdk/helpers_test.go`
  against T002 baseline to confirm reduction (SC-002)
- [x] T015 Run quickstart.md validation: verify all code examples from
  `specs/048-test-descriptor-helper/quickstart.md` are consistent with
  the implemented helper

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies — start immediately
- **User Story 1 (Phase 2)**: Depends on Phase 1 baseline capture
- **User Story 2 (Phase 3)**: Depends on Phase 2 — helper must exist
  before refactoring
- **Polish (Phase 4)**: Depends on Phase 3 completion

### User Story Dependencies

- **User Story 1 (P1)**: Can start after baseline (Phase 1) — No
  dependencies on other stories
- **User Story 2 (P2)**: Depends on User Story 1 — requires
  `newTestDescriptor` to exist before refactoring callsites

### Within Each User Story

- T003–T006 are logically sequential (building up the helper function)
- T008–T010 are logically sequential (identify, refactor, verify)
- T007 and T011 are validation gates at the end of each story

### Parallel Opportunities

- T003–T006 could be collapsed into a single implementation step since
  they all modify the same function in the same file
- T012 and T013 can run in parallel (different checks, no file conflicts)
- Phase 4 tasks T012, T013, T014 are all independent validation checks

---

## Parallel Example: Phase 4 (Polish)

```bash
# Launch all validation checks together:
Task: "Run golangci-lint on sdk/go/pluginsdk/"
Task: "Verify no new exported symbols"
Task: "Compare line count against baseline"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Baseline verification
2. Complete Phase 2: User Story 1 — create `newTestDescriptor` helper
3. **STOP and VALIDATE**: All tests pass, helper is usable for new tests
4. This alone delivers value for future test development

### Incremental Delivery

1. Baseline → captured
2. Add `newTestDescriptor` helper → Test suite passes → MVP complete!
3. Refactor existing callsites → Test suite still passes → Full delivery
4. Polish → Lint clean, line count confirmed reduced

### Practical Note

Given the small scope (single file, single function), all phases can
realistically be completed in a single implementation pass. The phased
structure exists for traceability and checkpoint validation.

---

## Notes

- Single file modified: `sdk/go/pluginsdk/helpers_test.go`
- Zero behavior changes — this is a pure refactoring
- Per R-004: most existing tests are intentionally verbose (testing
  construction API) and should NOT be refactored
- The primary value is for **future test development** (US1), not mass
  refactoring (US2)
- Avoid: refactoring tests that use non-standard providers, test
  construction behavior directly, or use struct literals
