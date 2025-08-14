package testutils

import (
	"github.com/stretchr/testify/mock"

	"event/internal/models"
)

// Custom matchers for testify mock

// MatchAnyEvent matches any Event pointer
func MatchAnyEvent() interface{} {
	return mock.MatchedBy(func(e *models.Event) bool {
		return e != nil
	})
}

// MatchAnySession matches any Session pointer
func MatchAnySession() interface{} {
	return mock.MatchedBy(func(s *models.Session) bool {
		return s != nil
	})
}

// MatchAnySessionSlice matches any slice of Session pointers
func MatchAnySessionSlice() interface{} {
	return mock.MatchedBy(func(sessions []*models.Session) bool {
		return sessions != nil
	})
}

// MatchAnyObjectIDString matches any valid ObjectID string
func MatchAnyObjectIDString() interface{} {
	return mock.MatchedBy(func(id string) bool {
		return len(id) == 24 // MongoDB ObjectID hex string length
	})
}

// Repository filter matchers are defined in service tests to avoid circular imports

// Generic request matchers (avoiding service package dependency)

// MatchAnyRequestWithTitle matches any request struct with a Title field
func MatchAnyRequestWithTitle() interface{} {
	return mock.MatchedBy(func(req interface{}) bool {
		return req != nil
	})
}

// MatchAnyRequestWithID matches any request struct with an ID field
func MatchAnyRequestWithID() interface{} {
	return mock.MatchedBy(func(req interface{}) bool {
		return req != nil
	})
}

// MatchEventWithTitle matches an Event with specific title
func MatchEventWithTitle(title string) interface{} {
	return mock.MatchedBy(func(e *models.Event) bool {
		return e != nil && e.Title == title
	})
}

// MatchEventWithStatus matches an Event with specific status
func MatchEventWithStatus(status string) interface{} {
	return mock.MatchedBy(func(e *models.Event) bool {
		return e != nil && e.Status == status
	})
}

// MatchSessionWithEventID matches a Session with specific event ID
func MatchSessionWithEventID(eventID string) interface{} {
	return mock.MatchedBy(func(s *models.Session) bool {
		return s != nil && s.EventID.Hex() == eventID
	})
}

// MatchStringSlice matches any string slice
func MatchStringSlice() interface{} {
	return mock.MatchedBy(func(slice []string) bool {
		return slice != nil
	})
}

// MatchEmptyStringSlice matches empty string slice
func MatchEmptyStringSlice() interface{} {
	return mock.MatchedBy(func(slice []string) bool {
		return slice != nil && len(slice) == 0
	})
}

// MatchNonEmptyStringSlice matches non-empty string slice
func MatchNonEmptyStringSlice() interface{} {
	return mock.MatchedBy(func(slice []string) bool {
		return len(slice) > 0
	})
}

// Filter matchers with brand ID are defined in service tests to avoid circular imports

// MatchAnyContext matches any context
func MatchAnyContext() interface{} {
	return mock.AnythingOfType("*context.emptyCtx")
}

// Helper functions for creating specific matchers

// CreateTitleMatcher creates a matcher for events with specific title
func CreateTitleMatcher(title string) func(*models.Event) bool {
	return func(e *models.Event) bool {
		return e != nil && e.Title == title
	}
}

// CreateStatusMatcher creates a matcher for events with specific status
func CreateStatusMatcher(status string) func(*models.Event) bool {
	return func(e *models.Event) bool {
		return e != nil && e.Status == status
	}
}

// CreateBrandMatcher creates a matcher for events with specific brand ID
func CreateBrandMatcher(brandID string) func(*models.Event) bool {
	return func(e *models.Event) bool {
		return e != nil && e.BrandID.Hex() == brandID
	}
}

// CreateSessionEventMatcher creates a matcher for sessions with specific event ID
func CreateSessionEventMatcher(eventID string) func(*models.Session) bool {
	return func(s *models.Session) bool {
		return s != nil && s.EventID.Hex() == eventID
	}
}

// CreateSessionCountMatcher creates a matcher for session slices with specific count
func CreateSessionCountMatcher(count int) func([]*models.Session) bool {
	return func(sessions []*models.Session) bool {
		return sessions != nil && len(sessions) == count
	}
}
