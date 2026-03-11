# Research: ValidationError Integration

**Feature**: 047-validation-error-integration
**Date**: 2026-03-07

## Research Topics

### 1. Go Error Wrapping Best Practices

**Decision**: Add an `Err` field to `ValidationError` and implement `Unwrap() error` method.

**Rationale**: Go's `errors` package (since 1.13) relies on `Unwrap()` for chain traversal.
The `errors.Is` function calls `Unwrap()` recursively to find a matching sentinel. The
`errors.As` function checks each error in the chain for type matching. By adding `Unwrap()`,
a single `*ValidationError` supports both:

- `errors.Is(err, ErrEffectiveCostExceedsBilledCost)` — identity check via chain traversal
- `errors.As(err, &valErr)` — type extraction for structured field access

**Alternatives considered**:

- **fmt.Errorf with %w**: Would wrap the sentinel but lose structured fields. Rejected because
  the whole point is providing `FieldName`, `Constraint`, etc.
- **Custom Is() method**: Could override `Is()` on `ValidationError` instead of `Unwrap()`.
  Rejected because `Unwrap()` is the standard convention and works with `errors.As` too.
- **Separate wrapper type**: A new `WrappedValidationError` type. Rejected as unnecessary
  complexity; adding `Err` field to existing type is simpler.

### 2. Constructor Design for Wrapped Errors

**Decision**: Add a `NewValidationErrorWithCause` constructor alongside the existing
`NewValidationError`. Signature: `(fieldName, constraint, actual, expected string, cause error)`
returning `*ValidationError`.

**Rationale**: The existing `NewValidationError(fieldName, constraint, actual, expected)` has
4 parameters and is already in use. Adding a 5th parameter would break existing callers.
A separate constructor with the `WithCause` suffix follows Go naming conventions.

**Alternatives considered**:

- **Functional options**: `NewValidationError("field", "constraint", WithCause(err))`. Rejected
  as over-engineered for a simple struct with 5 fields.
- **Add optional parameter to existing**: `NewValidationError(field, constraint, actual, expected, cause...)`.
  Rejected because variadic error parameters are confusing and the existing signature is stable.
- **Builder pattern**: `NewValidationError().WithField("x").WithCause(err)`. Rejected as
  over-engineered for this use case.

### 3. Error Return Site Inventory

**Decision**: Convert all 24 error return sites in `focus_conformance.go`.

**Inventory**:

| Category | Count | Examples |
|----------|-------|---------|
| Sentinel errors | 7 | `ErrEffectiveCostExceedsBilledCost`, `ErrPricingUnitMissing` |
| Mandatory field missing | 11 | `"provider_name is required"`, `"billing_currency is required"` |
| Invalid cost values | 2 | `"{name} cannot be infinity"`, `"{name} cannot be NaN"` |
| Business rule violations | 3 | contracted cost mismatch, consumed quantity, allocation rule |
| Currency validation | 1 | `"{field} must be a valid ISO 4217 currency code"` |
| **Total** | **24** | |

**Rationale**: Converting all sites provides a uniform API. Partial conversion would leave
callers unable to rely on `errors.As` working consistently.

### 4. Performance Impact Assessment

**Decision**: Accept one heap allocation per validation error (struct creation).

**Rationale**: Current sentinel errors are zero-allocation (package-level variables). Wrapping
them in `*ValidationError` adds one allocation (~96 bytes on 64-bit: 4 string headers + 1
error interface = 5 * 16 bytes + padding). This is negligible because:

- Validation errors are on the error path (not hot path)
- A single gRPC response involves orders-of-magnitude more allocations
- The sentinel variables themselves remain zero-allocation for `errors.Is` comparison

**Benchmark requirement**: Add benchmark for `ValidationError` creation to confirm < 100 ns/op.

### 5. Naming Collision: pluginsdk.ValidationError vs pbc.ValidationError

**Decision**: No action needed. The two types are distinct and used in different contexts.

**Details**:

- `pluginsdk.ValidationError` — Go struct for SDK-level validation (this feature)
- `pbc.ValidationError` — Protobuf-generated message used in `manifest.go`
- `pluginsdk.ValidationErrors` — Slice type wrapping `[]*pbc.ValidationError`

These types serve different purposes and are never confused in practice because they're
in different packages.
