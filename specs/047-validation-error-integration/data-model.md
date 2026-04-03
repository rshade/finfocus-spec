# Data Model: ValidationError Integration

**Feature**: 047-validation-error-integration
**Date**: 2026-03-07

## Entities

### ValidationError (modified)

Structured validation error with error chain support.

| Field | Type | Description | Change |
|-------|------|-------------|--------|
| FieldName | string | Name of the field that failed validation | Existing |
| Constraint | string | Description of the violated rule | Existing |
| ActualValue | string | String representation of actual value | Existing |
| ExpectedValue | string | String representation of expected value | Existing |
| err | error | Unexported field; wrapped inner error accessible only via `Unwrap()` | **New** |

**Methods**:

| Method | Signature | Description | Change |
|--------|-----------|-------------|--------|
| Error | `() string` | Returns formatted error message | Existing (format unchanged) |
| Unwrap | `() error` | Returns wrapped inner error | **New** |

**Validation Rules**:

- `FieldName` MUST be non-empty for all validation errors
- `Constraint` MUST be non-empty for all validation errors
- `err` MAY be nil (for ad-hoc errors without a sentinel)
- `Unwrap()` returns the unexported `err` field directly (nil-safe)

### Sentinel Errors (unchanged)

Package-level error variables used for identity matching via `errors.Is`.

| Variable | Message |
|----------|---------|
| ErrEffectiveCostExceedsBilledCost | "effective_cost must not exceed billed_cost" |
| ErrListCostLessThanEffectiveCost | "list_cost must be >= effective_cost" |
| ErrCommitmentStatusMissing | "commitment_discount_status required when..." |
| ErrCommitmentIDMissingForStatus | "commitment_discount_id required when..." |
| ErrCapacityReservationStatusMissing | "capacity_reservation_status required when..." |
| ErrCapacityReservationIDMissing | "capacity_reservation_id required when..." |
| ErrPricingUnitMissing | "pricing_unit required when pricing_quantity > 0" |

### Constructors

| Function | Signature | Change |
|----------|-----------|--------|
| NewValidationError | `(fieldName, constraint, actual, expected string) *ValidationError` | Existing (unchanged) |
| NewValidationErrorWithCause | `(fieldName, constraint, actual, expected string, cause error) *ValidationError` | **New** |

## Relationships

```text
ValidateFocusRecord() ──returns──▶ error
                                      │
                                      ▼
                              *ValidationError
                                      │
                           Unwrap() ──▶ sentinel error (or nil)
                                      │
                       errors.Is() ──▶ ErrEffectiveCostExceedsBilledCost
                       errors.As() ──▶ *ValidationError (FieldName, Constraint, etc.)
```

## Error Site Mapping

Mapping of each error return site to its `ValidationError` field values:

### Sentinel Error Wrapping (7 sites)

| Sentinel | FieldName | Constraint |
|----------|-----------|------------|
| ErrEffectiveCostExceedsBilledCost | effective_cost | must not exceed billed_cost |
| ErrListCostLessThanEffectiveCost | list_cost | must be >= effective_cost |
| ErrCommitmentStatusMissing | commitment_discount_status | required when commitment_discount_id set |
| ErrCommitmentIDMissingForStatus | commitment_discount_id | required when commitment_discount_status set |
| ErrCapacityReservationStatusMissing | capacity_reservation_status | required when capacity_reservation_id set |
| ErrCapacityReservationIDMissing | capacity_reservation_id | required when capacity_reservation_status set |
| ErrPricingUnitMissing | pricing_unit | required when pricing_quantity > 0 |

### Ad-Hoc Error Conversion (19 sites)

| Current Error | FieldName | Constraint | Err (wrapped) |
|---------------|-----------|------------|---------------|
| "record is nil" | record | must not be nil | nil |
| "{name} cannot be infinity" | {name} | must be finite | nil |
| "{name} cannot be NaN" | {name} | must be a number | nil |
| "provider_name is required" | provider_name | required | nil |
| "billing_account_id is required" | billing_account_id | required | nil |
| "billing_period_start is required" | billing_period.start | required | nil |
| "billing_period_end is required" | billing_period.end | required | nil |
| "billing_currency is required" | billing_currency | required | nil |
| "charge_period_start is required" | charge_period.start | required | nil |
| "charge_period_end is required" | charge_period.end | required | nil |
| "charge_category is required" | charge_category | required | nil |
| "charge_class is required" | charge_class | required | nil |
| "charge_description is required" | charge_description | required | nil |
| "service_category is required" | service_category | required | nil |
| "service_name is required" | service_name | required | nil |
| "allocated_resource_id required..." | allocated_resource_id | required when allocated_method_id set | nil |
| "{field} must be valid ISO 4217..." | {field} | must be valid ISO 4217 currency code | nil |
| "contracted_cost must equal..." | contracted_cost | must equal unit_price x quantity | nil |
| "consumed_quantity must be positive..." | consumed_quantity | must be positive for usage charges | nil |
