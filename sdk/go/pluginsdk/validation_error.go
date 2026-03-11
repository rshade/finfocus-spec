package pluginsdk

import "fmt"

// ValidationError represents a structured validation error with discrete fields
// for programmatic error inspection. This enables callers to parse and categorize
// validation failures without string parsing.
//
// ValidationError implements Unwrap() for Go's error chain traversal. When created
// with [NewValidationErrorWithCause], both [errors.Is] (sentinel matching) and
// [errors.As] (type extraction) work on the same error value:
//
//	err := ValidateFocusRecord(record)
//
//	// Extract structured details
//	var valErr *ValidationError
//	if errors.As(err, &valErr) {
//	    fmt.Printf("Field %s failed: %s\n", valErr.FieldName, valErr.Constraint)
//	}
//
//	// Match sentinel identity through error chain
//	if errors.Is(err, ErrEffectiveCostExceedsBilledCost) {
//	    // handle cost hierarchy violation
//	}
type ValidationError struct {
	// FieldName is the name of the field that failed validation.
	FieldName string

	// Constraint describes the validation rule that was violated.
	Constraint string

	// ActualValue is a string representation of the actual value found.
	ActualValue string

	// ExpectedValue is a string representation of what was expected.
	ExpectedValue string

	// err is the wrapped inner error for error chain support.
	// When set, Unwrap() returns this error, enabling errors.Is() chain traversal.
	// Unexported to prevent callers from mutating the error chain directly.
	err error
}

// NewValidationError creates a new ValidationError with the specified fields.
// This constructor ensures consistent field population and makes it easier
// to evolve the type in the future without breaking existing code.
func NewValidationError(fieldName, constraint, actualValue, expectedValue string) *ValidationError {
	return &ValidationError{
		FieldName:     fieldName,
		Constraint:    constraint,
		ActualValue:   actualValue,
		ExpectedValue: expectedValue,
	}
}

// NewValidationErrorWithCause creates a new ValidationError that wraps a cause error.
// The cause is accessible via Unwrap(), enabling errors.Is() chain traversal
// to match sentinel errors through the ValidationError wrapper.
func NewValidationErrorWithCause(
	fieldName, constraint, actualValue, expectedValue string,
	cause error,
) *ValidationError {
	return &ValidationError{
		FieldName:     fieldName,
		Constraint:    constraint,
		ActualValue:   actualValue,
		ExpectedValue: expectedValue,
		err:           cause,
	}
}

// Error implements the error interface.
// Format: "{FieldName}: {Constraint} (actual: {ActualValue}, expected: {ExpectedValue})".
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s (actual: %s, expected: %s)",
		e.FieldName, e.Constraint, e.ActualValue, e.ExpectedValue)
}

// Unwrap returns the wrapped inner error, enabling errors.Is() and errors.As()
// chain traversal through Go's standard error wrapping convention.
func (e *ValidationError) Unwrap() error {
	return e.err
}
