# Quick Start: ValidationError Integration

**Feature**: 047-validation-error-integration

```go
// Validate a FOCUS cost record
err := pluginsdk.ValidateFocusRecord(record)
if err != nil {
    // Can only check identity via errors.Is
    if errors.Is(err, pluginsdk.ErrEffectiveCostExceedsBilledCost) {
        // Handle cost hierarchy violation
    }
    // No way to extract field name or constraint programmatically
    // Must parse error string: "effective_cost must not exceed billed_cost"
    log.Error().Err(err).Msg("validation failed")
}
```

## After (New Behavior)

```go
// Validate a FOCUS cost record
err := pluginsdk.ValidateFocusRecord(record)
if err != nil {
    // errors.Is still works (backward compatible)
    if errors.Is(err, pluginsdk.ErrEffectiveCostExceedsBilledCost) {
        // Handle cost hierarchy violation
    }

    // NEW: Extract structured error details via errors.As
    var valErr *pluginsdk.ValidationError
    if errors.As(err, &valErr) {
        log.Error().
            Str("field", valErr.FieldName).
            Str("constraint", valErr.Constraint).
            Str("actual", valErr.ActualValue).
            Str("expected", valErr.ExpectedValue).
            Msg("validation failed")
    }
}
```

## Aggregate Mode with Rich Errors

```go
// Collect all validation errors
opts := pluginsdk.ValidationOptions{Mode: pluginsdk.ValidationModeAggregate}
errs := pluginsdk.ValidateFocusRecordWithOptions(record, opts)

// Categorize errors by field name programmatically
fieldErrors := make(map[string][]*pluginsdk.ValidationError)
for _, err := range errs {
    var valErr *pluginsdk.ValidationError
    if errors.As(err, &valErr) {
        fieldErrors[valErr.FieldName] = append(fieldErrors[valErr.FieldName], valErr)
    }
}

// Report errors grouped by field
for field, errors := range fieldErrors {
    fmt.Printf("Field %s has %d error(s)\n", field, len(errors))
}
```

## Migration Guide

### For code using errors.Is (no changes needed)

```go
// This continues to work unchanged
if errors.Is(err, pluginsdk.ErrEffectiveCostExceedsBilledCost) {
    // ...
}
```

### For code using string matching (must migrate)

```go
// BEFORE (fragile, will break)
if err.Error() == "effective_cost must not exceed billed_cost" {
    // ...
}

// AFTER (idiomatic)
if errors.Is(err, pluginsdk.ErrEffectiveCostExceedsBilledCost) {
    // ...
}
```
