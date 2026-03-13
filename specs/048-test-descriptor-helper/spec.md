# Feature Specification: Extract Test Helper for ResourceDescriptor Creation

**Feature Branch**: `048-test-descriptor-helper`
**Created**: 2026-03-12
**Status**: Draft
**Input**: GitHub Issue #204 - refactor(pluginsdk): Extract test helper for ResourceDescriptor creation

## User Scenarios & Testing *(mandatory)*

### User Story 1 - SDK Developer Writes New Tests (Priority: P1)

An SDK developer adding new tests to `helpers_test.go` needs to create `ResourceDescriptor`
instances quickly with sensible defaults, without copying boilerplate from existing tests.

**Why this priority**: This is the primary use case — reducing friction for ongoing test
development and ensuring consistent test defaults across the file.

**Independent Test**: Can be verified by adding a new test that uses the helper function
and confirming it produces the same `ResourceDescriptor` as the equivalent verbose call.

**Acceptance Scenarios**:

1. **Given** the `newTestDescriptor` helper exists in `helpers_test.go`,
   **When** a developer calls it with an ID and ARN,
   **Then** a `ResourceDescriptor` is returned with standard test defaults
   plus the specified ID and ARN.

2. **Given** the `newTestDescriptor` helper exists,
   **When** a developer calls it with additional options,
   **Then** those options are applied on top of the defaults.

3. **Given** the `newTestDescriptor` helper exists,
   **When** a developer passes empty strings for ID or ARN,
   **Then** those fields are not set on the descriptor (only non-empty values applied).

---

### User Story 2 - Existing Tests Refactored to Use Helper (Priority: P2)

All existing `NewResourceDescriptor` calls in `helpers_test.go` that use the common
pattern with standard test defaults are refactored to use the new helper,
reducing line count and improving consistency.

**Why this priority**: The refactoring delivers the maintainability improvement promised
by the feature, but depends on the helper existing first.

**Independent Test**: Can be verified by running the full test suite before and after
refactoring — all tests must pass with identical behavior.

**Acceptance Scenarios**:

1. **Given** existing tests use verbose `ResourceDescriptor` construction with standard
   defaults,
   **When** they are refactored to use `newTestDescriptor`,
   **Then** all tests continue to pass with no behavior changes.

2. **Given** some tests use non-standard providers or resource types,
   **When** the refactoring is applied,
   **Then** those tests are left unchanged (helper only applies to the common pattern).

---

### Edge Cases

- What happens when both `id` and `arn` are empty strings? The helper creates a
  descriptor with only the base defaults (provider, resource type, SKU, region).
- What happens when additional options override a base default (e.g., passing a
  different region)? The override takes precedence since additional options are
  appended after base options.
- Tests that intentionally test `NewResourceDescriptor` construction behavior
  (e.g., basic construction, option ordering) are NOT refactored, as their
  verbosity is intentional for test clarity.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The helper function MUST be unexported and exist only in `helpers_test.go`.
- **FR-002**: The helper MUST accept an ID string, ARN string, and variadic descriptor
  options as parameters.
- **FR-003**: The helper MUST apply sensible test defaults for provider, resource type,
  SKU, and region.
- **FR-004**: The helper MUST only set ID when the ID parameter is non-empty.
- **FR-005**: The helper MUST only set ARN when the ARN parameter is non-empty.
- **FR-006**: Additional options MUST be appended after base options, allowing callers
  to override defaults.
- **FR-007**: The refactoring MUST NOT change any test behavior — all existing tests
  MUST pass identically before and after.
- **FR-008**: Tests that directly test `NewResourceDescriptor` construction behavior
  MUST NOT be refactored, as their explicit construction is part of the test intent.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: All existing tests pass after refactoring with zero behavior changes.
- **SC-002**: Test file line count is reduced in refactored sections.
- **SC-003**: Every refactored test call uses fewer lines than the original verbose call.
- **SC-004**: No new exported symbols are introduced — the helper remains test-internal.
- **SC-005**: All linting checks pass cleanly with no new warnings.

## Assumptions

- The most common test pattern uses a specific provider and resource type — other
  providers appear infrequently and don't warrant separate helpers initially.
- The chosen default SKU and region values are acceptable standard test values and
  won't conflict with test assertions.
- Provider-specific helper variants are out of scope for this iteration but may be
  added later if patterns emerge.
- The `sdk/go/testing/` package may have similar patterns but is out of scope for
  this issue; it can be addressed in a follow-up.
