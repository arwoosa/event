package errors

import "fmt"

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string
	Message string
	Index   *int // for array field validation
}

func (e *ValidationError) Error() string {
	if e.Index != nil {
		return fmt.Sprintf("validation error in %s[%d]: %s", e.Field, *e.Index, e.Message)
	}
	return fmt.Sprintf("validation error in %s: %s", e.Field, e.Message)
}

// BusinessError represents a business logic error
type BusinessError struct {
	Code    string
	Message string
	Cause   error
}

func (e *BusinessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *BusinessError) Unwrap() error {
	return e.Cause
}

// NewValidationError creates a new validation error
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorWithIndex creates a new validation error with array index
func NewValidationErrorWithIndex(field, message string, index int) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Index:   &index,
	}
}

// NewBusinessError creates a new business error
func NewBusinessError(code, message string, cause error) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}
