# Feature Specification: Integrate ValidationError with Validation Implementation

**Feature Branch**: `047-validation-error-integration`
**Created**: 2026-03-07
**Status**: Draft
**Input**: GitHub Issue #210 - refactor(pluginsdk): Integrate ValidationError type with validation implementation

## Clarifications

### Session 2026-03-07

- Q: When validation returns `ValidationError` instead of bare sentinels, should `Error()` preserve
  the exact original string or use the richer `ValidationError` format?
  A: Use the richer `ValidationError` format. String-based error matching is an anti-pattern;
  callers should use `errors.Is` for identity checks.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Extract Structured Error Details from Validation Failures (Priority: P1)

As a plugin developer, I want to extract structured error details (field name, constraint,
actual/expected values) from validation failures so that I can build programmatic error handling
without parsing error message strings.

**Why this priority**: This is the core value proposition. The `ValidationError` type exists
specifically to enable `errors.As`-based programmatic inspection, but currently no validation
function returns it, making the type useless in practice.

**Independent Test**: Can be fully tested by calling `ValidateFocusRecord` with a record that
violates a cost hierarchy rule and using `errors.As(err, &valErr)` to extract field-level details.

**Acceptance Scenarios**:

1. **Given** a FocusCostRecord where effective_cost exceeds billed_cost,
   **When** I call `ValidateFocusRecord`,
   **Then** the returned error supports `errors.As(err, &valErr)` and `valErr.FieldName` is
   `"effective_cost"`.
2. **Given** a FocusCostRecord where effective_cost exceeds billed_cost,
   **When** I call `ValidateFocusRecord` and check `errors.Is(err, ErrEffectiveCostExceedsBilledCost)`,
   **Then** the check returns `true` (backward compatibility preserved).
3. **Given** a FocusCostRecord with a missing mandatory field,
   **When** I call `ValidateFocusRecord`,
   **Then** the returned error supports `errors.As(err, &valErr)` with the missing field name
   and constraint details populated.

---

### User Story 2 - Aggregate Mode Rich Error Collection (Priority: P2)

As a data quality engineer, I want aggregate validation mode to return `ValidationError` instances
for every failure so that I can programmatically categorize and report on all issues in a batch of
cost records without string parsing.

**Why this priority**: Aggregate mode is the primary use case for rich error context since users
process multiple errors and need to group/filter by field name or constraint type.

**Independent Test**: Can be tested by calling `ValidateFocusRecordWithOptions` in aggregate mode
with a record containing multiple violations and verifying each returned error unwraps to a
`ValidationError` with correct field details.

**Acceptance Scenarios**:

1. **Given** a FocusCostRecord with three distinct validation failures,
   **When** I call `ValidateFocusRecordWithOptions` in aggregate mode,
   **Then** all three returned errors support `errors.As` extraction with distinct field names.
2. **Given** a FocusCostRecord with both a cost hierarchy violation and a missing commitment status,
   **When** I validate in aggregate mode and iterate over errors,
   **Then** I can programmatically separate errors by `FieldName` without string parsing.

---

### User Story 3 - Error Chain Unwrapping (Priority: P2)

As a plugin developer, I want `ValidationError` to support `Unwrap()` so that I can use both
`errors.Is` (for sentinel error matching) and `errors.As` (for structured detail extraction)
on the same error.

**Why this priority**: Go's error wrapping convention requires `Unwrap()` for proper error chain
support. Without it, `errors.Is` cannot traverse to the wrapped sentinel error.

**Independent Test**: Can be tested by wrapping a sentinel error in a `ValidationError`, then
verifying both `errors.Is(err, sentinel)` and `errors.As(err, &valErr)` work on the same error.

**Acceptance Scenarios**:

1. **Given** a `ValidationError` wrapping `ErrEffectiveCostExceedsBilledCost`,
   **When** I call `errors.Is(err, ErrEffectiveCostExceedsBilledCost)`,
   **Then** it returns `true`.
2. **Given** the same wrapped error,
   **When** I call `errors.As(err, &valErr)`,
   **Then** it returns `true` and `valErr.FieldName` is populated.

---

### Edge Cases

- What happens when `ValidationError` wraps a nil inner error (e.g., for ad-hoc errors like
  "record is nil" that have no sentinel)? The `Unwrap()` method returns nil, and `errors.Is`
  checks against specific sentinels return false, but `errors.As` still extracts the
  structured fields.
- What happens when a cost field is NaN or Infinity? These validation errors also return
  `ValidationError` with the field name and constraint, maintaining consistent error typing
  across all validation paths.
- How does this affect existing code that uses `errors.Is` against sentinel errors? Full backward
  compatibility is maintained because `ValidationError.Unwrap()` exposes the wrapped sentinel,
  so `errors.Is(err, ErrEffectiveCostExceedsBilledCost)` continues to work.
- The `Error()` output format changes from the sentinel's message to the richer `ValidationError`
  format. Code that uses string-based error matching should migrate to `errors.Is`. This is an
  intentional breaking change for string matchers, as `errors.Is` is the idiomatic approach.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: `ValidationError` MUST include an `Unwrap()` method that returns the wrapped
  sentinel error, enabling Go's `errors.Is` to traverse the error chain.
- **FR-002**: All validation functions in `focus_conformance.go` that currently return sentinel
  errors MUST return `*ValidationError` wrapping those sentinels, with `FieldName`, `Constraint`,
  `ActualValue`, and `ExpectedValue` populated.
- **FR-003**: All validation functions that currently return inline string errors MUST return
  `*ValidationError` with appropriate field details populated.
- **FR-004**: Both `ValidateFocusRecord` (fail-fast) and `ValidateFocusRecordWithOptions`
  (aggregate) MUST return errors that satisfy `errors.As(err, &valErr)` for `*ValidationError`.
- **FR-005**: `errors.Is(err, <sentinel>)` MUST continue to work for all 7 existing sentinel
  errors to preserve backward compatibility.
- **FR-006**: A constructor or mechanism MUST exist to create a `ValidationError` with a wrapped
  inner error for error chaining support.
- **FR-007**: The `Error()` method MUST use the `ValidationError` structured format
  (`"field: constraint (actual: X, expected: Y)"`) for all validation errors, providing richer
  diagnostic output than the previous sentinel message strings.

### Key Entities

- **ValidationError**: Structured error type with FieldName, Constraint, ActualValue,
  ExpectedValue, and a wrapped inner error for chain support.
- **Sentinel Errors**: Package-level error variables (7 total, e.g.,
  `ErrEffectiveCostExceedsBilledCost`) that identify specific validation rule violations.
- **ValidationOptions**: Configuration controlling fail-fast vs. aggregate mode behavior
  (unchanged by this feature).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of validation errors returned by `ValidateFocusRecord` and
  `ValidateFocusRecordWithOptions` are extractable via `errors.As(err, &valErr)`.
- **SC-002**: 100% backward compatibility with existing `errors.Is(err, sentinel)` checks
  across all 7 sentinel errors.
- **SC-003**: All existing tests in `focus_conformance_test.go` and `validation_error_test.go`
  continue to pass (with adaptation for the new `Error()` format and `Unwrap` capability).
- **SC-004**: Plugin developers can categorize validation failures by field name
  programmatically, eliminating string parsing for error classification.
- **SC-005**: No regression in validation performance: fail-fast mode adds at most one
  struct allocation per validation error beyond the current baseline.

## Assumptions

- **Option B selected**: This specification follows Option B from the issue (wrap sentinel errors
  with `ValidationError`). This provides the best balance of structured error access and backward
  compatibility via Go's `errors.Is`/`errors.As` dual pattern.
- **Both modes get rich errors**: Both fail-fast and aggregate modes return `ValidationError`.
  Option C (aggregate-only) was considered but wrapping in fail-fast mode adds only one struct
  allocation per error and provides a consistent, predictable API surface.
- **Ad-hoc errors wrapped too**: Validation errors created via inline string construction
  (e.g., mandatory field checks, currency validation, NaN/Inf checks) will also be wrapped in
  `ValidationError` to provide a uniform error type across all validation paths.
- **Constructor change is additive**: Adding wrapped error support to `ValidationError` can use
  a new constructor or functional options, preserving the existing `NewValidationError` signature
  for backward compatibility.
- **Error() format change is intentional**: The `Error()` output changes from bare sentinel
  messages to the richer `ValidationError` format. Callers relying on string matching should
  migrate to `errors.Is`, which is the idiomatic Go error identity check.
