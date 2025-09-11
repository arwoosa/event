package errors

import "errors"

// Common domain errors - these represent business rule violations or entity states
var (
	ErrEventNotFound     = errors.New("event not found")
	ErrSessionNotFound   = errors.New("session not found")
	ErrFormNotFound      = errors.New("form not found")
	ErrFormAlreadyExists = errors.New("form already exists")
	ErrInvalidFormID     = errors.New("invalid form ID")
	ErrInvalidStatus     = errors.New("invalid status")
	ErrInvalidVisibility = errors.New("invalid visibility")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrNoSessions        = errors.New("event must have at least one session")
	ErrHasOrders         = errors.New("event has existing orders")
	ErrUnauthorized      = errors.New("unauthorized access")
	ErrInvalidMerchantID = errors.New("invalid merchant_id")
)
