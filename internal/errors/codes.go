package errors

// Business error codes - centralized constants for all error codes used across the application
const (
	ErrorCodeInvalidTransition        = "INVALID_TRANSITION"
	ErrorCodePublishedImmutable       = "PUBLISHED_IMMUTABLE"
	ErrorCodePublishedFieldRestricted = "PUBLISHED_FIELD_RESTRICTED"
	ErrorCodeHasOrders                = "HAS_ORDERS"
	ErrorCodeSessionHasOrders         = "SESSION_HAS_ORDERS"
	ErrorCodeLastSession              = "LAST_SESSION"
	ErrorCodeSessionNotFound          = "SESSION_NOT_FOUND"
	ErrorCodeValidationError          = "VALIDATION_ERROR"
)
