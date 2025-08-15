package errors

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name           string
		field          string
		message        string
		index          *int
		expectedOutput string
	}{
		{
			name:           "Simple validation error",
			field:          "title",
			message:        "is required",
			index:          nil,
			expectedOutput: "validation error in title: is required",
		},
		{
			name:           "Validation error with index",
			field:          "sessions",
			message:        "start time is invalid",
			index:          &[]int{2}[0],
			expectedOutput: "validation error in sessions[2]: start time is invalid",
		},
		{
			name:           "Empty field name",
			field:          "",
			message:        "unknown error",
			index:          nil,
			expectedOutput: "validation error in : unknown error",
		},
		{
			name:           "Zero index",
			field:          "faq",
			message:        "question is empty",
			index:          &[]int{0}[0],
			expectedOutput: "validation error in faq[0]: question is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ValidationError{
				Field:   tt.field,
				Message: tt.message,
				Index:   tt.index,
			}

			assert.Equal(t, tt.expectedOutput, err.Error())
		})
	}
}

func TestBusinessError_Error(t *testing.T) {
	tests := []struct {
		name           string
		code           string
		message        string
		cause          error
		expectedOutput string
	}{
		{
			name:           "Simple business error",
			code:           "INVALID_STATUS",
			message:        "cannot transition to this status",
			cause:          nil,
			expectedOutput: "INVALID_STATUS: cannot transition to this status",
		},
		{
			name:           "Business error with cause",
			code:           "DATABASE_ERROR",
			message:        "failed to save event",
			cause:          errors.New("connection timeout"),
			expectedOutput: "DATABASE_ERROR: failed to save event (connection timeout)",
		},
		{
			name:           "Empty code",
			code:           "",
			message:        "something went wrong",
			cause:          nil,
			expectedOutput: ": something went wrong",
		},
		{
			name:           "Empty message",
			code:           "ERROR",
			message:        "",
			cause:          nil,
			expectedOutput: "ERROR: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &BusinessError{
				Code:    tt.code,
				Message: tt.message,
				Cause:   tt.cause,
			}

			assert.Equal(t, tt.expectedOutput, err.Error())
		})
	}
}

func TestBusinessError_Unwrap(t *testing.T) {
	causeErr := errors.New("root cause error")

	tests := []struct {
		name          string
		businessErr   *BusinessError
		expectedCause error
	}{
		{
			name: "Business error with cause",
			businessErr: &BusinessError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Cause:   causeErr,
			},
			expectedCause: causeErr,
		},
		{
			name: "Business error without cause",
			businessErr: &BusinessError{
				Code:    "TEST_ERROR",
				Message: "test message",
				Cause:   nil,
			},
			expectedCause: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.businessErr.Unwrap()
			assert.Equal(t, tt.expectedCause, result)
		})
	}
}

func TestNewValidationError(t *testing.T) {
	field := "email"
	message := "invalid format"

	err := NewValidationError(field, message)

	require.NotNil(t, err)
	assert.Equal(t, field, err.Field)
	assert.Equal(t, message, err.Message)
	assert.Nil(t, err.Index)
	assert.Equal(t, "validation error in email: invalid format", err.Error())
}

func TestNewValidationErrorWithIndex(t *testing.T) {
	field := "items"
	message := "missing required field"
	index := 5

	err := NewValidationErrorWithIndex(field, message, index)

	require.NotNil(t, err)
	assert.Equal(t, field, err.Field)
	assert.Equal(t, message, err.Message)
	require.NotNil(t, err.Index)
	assert.Equal(t, index, *err.Index)
	assert.Equal(t, "validation error in items[5]: missing required field", err.Error())
}

func TestNewBusinessError(t *testing.T) {
	code := "PERMISSION_DENIED"
	message := "user does not have access"
	cause := errors.New("unauthorized")

	err := NewBusinessError(code, message, cause)

	require.NotNil(t, err)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.Equal(t, "PERMISSION_DENIED: user does not have access (unauthorized)", err.Error())
}

func TestNewBusinessErrorWithoutCause(t *testing.T) {
	code := "INVALID_INPUT"
	message := "input validation failed"

	err := NewBusinessError(code, message, nil)

	require.NotNil(t, err)
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Nil(t, err.Cause)
	assert.Equal(t, "INVALID_INPUT: input validation failed", err.Error())
}

func TestCommonErrors(t *testing.T) {
	// Test that common error variables are properly defined
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"ErrEventNotFound", ErrEventNotFound, "event not found"},
		{"ErrSessionNotFound", ErrSessionNotFound, "session not found"},
		{"ErrInvalidStatus", ErrInvalidStatus, "invalid status"},
		{"ErrInvalidVisibility", ErrInvalidVisibility, "invalid visibility"},
		{"ErrInvalidTransition", ErrInvalidTransition, "invalid status transition"},
		{"ErrNoSessions", ErrNoSessions, "event must have at least one session"},
		{"ErrHasOrders", ErrHasOrders, "event has existing orders"},
		{"ErrUnauthorized", ErrUnauthorized, "unauthorized access"},
		{"ErrInvalidMerchantID", ErrInvalidMerchantID, "invalid merchant_id"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NotNil(t, tt.err)
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestErrorTypeAssertions(t *testing.T) {
	// Test that we can properly type assert our custom errors

	validationErr := NewValidationError("field", "message")
	businessErr := NewBusinessError("CODE", "message", nil)

	// Test ValidationError type assertion
	var err error = validationErr
	if ve, ok := err.(*ValidationError); ok {
		assert.Equal(t, "field", ve.Field)
		assert.Equal(t, "message", ve.Message)
	} else {
		t.Fatal("Failed to type assert ValidationError")
	}

	// Test BusinessError type assertion
	err = businessErr
	if be, ok := err.(*BusinessError); ok {
		assert.Equal(t, "CODE", be.Code)
		assert.Equal(t, "message", be.Message)
	} else {
		t.Fatal("Failed to type assert BusinessError")
	}
}

func TestErrorWrapping(t *testing.T) {
	// Test error wrapping functionality
	rootCause := errors.New("database connection failed")
	businessErr := NewBusinessError("DB_ERROR", "failed to save", rootCause)

	// Test that errors.Is works with wrapped errors
	assert.True(t, errors.Is(businessErr, rootCause))

	// Test that errors.Unwrap works
	unwrapped := errors.Unwrap(businessErr)
	assert.Equal(t, rootCause, unwrapped)

	// Test with multiple levels of wrapping
	wrapperErr := NewBusinessError("WRAPPER_ERROR", "operation failed", businessErr)
	assert.True(t, errors.Is(wrapperErr, businessErr))
	assert.True(t, errors.Is(wrapperErr, rootCause))
}

func TestValidationErrorChaining(t *testing.T) {
	// Test creating multiple validation errors for complex validation scenarios
	errors := []*ValidationError{
		NewValidationError("title", "is required"),
		NewValidationErrorWithIndex("sessions", "invalid time format", 0),
		NewValidationErrorWithIndex("faq", "question too long", 2),
	}

	assert.Len(t, errors, 3)

	// Verify each error has the correct format
	assert.Equal(t, "validation error in title: is required", errors[0].Error())
	assert.Equal(t, "validation error in sessions[0]: invalid time format", errors[1].Error())
	assert.Equal(t, "validation error in faq[2]: question too long", errors[2].Error())
}

func TestBusinessErrorCodes(t *testing.T) {
	// Test common business error codes that might be used in the application
	commonCodes := []string{
		"INVALID_STATUS_TRANSITION",
		"PUBLISHED_IMMUTABLE",
		"UNAUTHORIZED_ACCESS",
		"EVENT_NOT_FOUND",
		"SESSION_NOT_FOUND",
		"HAS_ORDERS",
		"NO_SESSIONS",
		"INVALID_TIME_RANGE",
		"DUPLICATE_SESSION",
	}

	for _, code := range commonCodes {
		err := NewBusinessError(code, "test message", nil)
		assert.Contains(t, err.Error(), code)
		assert.Equal(t, code, err.Code)
	}
}

func TestErrorMessageConsistency(t *testing.T) {
	// Ensure error messages are consistent and useful for debugging

	// ValidationError should always include field name
	validationErr := NewValidationError("test_field", "test message")
	assert.Contains(t, validationErr.Error(), "test_field")
	assert.Contains(t, validationErr.Error(), "validation error")

	// ValidationError with index should include both field and index
	indexErr := NewValidationErrorWithIndex("test_array", "test message", 3)
	assert.Contains(t, indexErr.Error(), "test_array[3]")
	assert.Contains(t, indexErr.Error(), "validation error")

	// BusinessError should always include code
	businessErr := NewBusinessError("TEST_CODE", "test message", nil)
	assert.Contains(t, businessErr.Error(), "TEST_CODE")

	// BusinessError with cause should include both message and cause
	causeErr := errors.New("cause message")
	businessErrWithCause := NewBusinessError("TEST_CODE", "test message", causeErr)
	assert.Contains(t, businessErrWithCause.Error(), "TEST_CODE")
	assert.Contains(t, businessErrWithCause.Error(), "test message")
	assert.Contains(t, businessErrWithCause.Error(), "cause message")
}
