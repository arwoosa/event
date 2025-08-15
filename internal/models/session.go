package models

import (
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Session represents an event session as a separate collection
// Merchant isolation is now handled through the parent Event entity
type Session struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID   primitive.ObjectID `json:"event_id" bson:"event_id"`
	Name      string             `json:"name" bson:"name"`         // Session name (optional)
	Capacity  *int               `json:"capacity" bson:"capacity"` // Capacity limit (optional, nil means unlimited)
	StartTime time.Time          `json:"start_time" bson:"start_time"`
	EndTime   time.Time          `json:"end_time" bson:"end_time"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
}

// IsDuplicateOf checks if two sessions have identical start and end times
func (s *Session) IsDuplicateOf(other *Session) bool {
	return s.StartTime.Equal(other.StartTime) && s.EndTime.Equal(other.EndTime)
}

// IsValid validates session time constraints
// Merchant validation is no longer required as sessions inherit merchant from event
func (s *Session) IsValid() error {
	if s.EventID.IsZero() {
		return errors.New("event_id is required")
	}
	// MerchantID validation removed - merchant isolation handled through Event
	if s.StartTime.IsZero() {
		return errors.New("start_time is required")
	}
	if s.EndTime.IsZero() {
		return errors.New("end_time is required")
	}
	if !s.StartTime.Before(s.EndTime) {
		return errors.New("start_time must be before end_time")
	}
	// Validate capacity if provided
	if s.Capacity != nil && *s.Capacity < 0 {
		return errors.New("capacity must be non-negative")
	}
	return nil
}

// Duration returns the duration of the session
func (s *Session) Duration() time.Duration {
	return s.EndTime.Sub(s.StartTime)
}

// ValidateSessions checks for duplicate sessions within a collection
func ValidateSessions(sessions []*Session) error {
	// Check each session's validity
	for i, session := range sessions {
		if err := session.IsValid(); err != nil {
			return fmt.Errorf("session %d: %w", i, err)
		}
	}

	// Check for duplicate sessions using hash set - O(n) complexity
	seen := make(map[string]int) // map[timeKey]sessionIndex
	for i, session := range sessions {
		// Create unique key from start and end times
		key := fmt.Sprintf("%d-%d", session.StartTime.Unix(), session.EndTime.Unix())

		if existingIndex, exists := seen[key]; exists {
			return fmt.Errorf("sessions %d and %d have identical start and end times", existingIndex, i)
		}
		seen[key] = i
	}

	return nil
}

// GetEarliestStartTime returns the earliest start time among sessions
func GetEarliestStartTime(sessions []*Session) *time.Time {
	if len(sessions) == 0 {
		return nil
	}

	earliest := sessions[0].StartTime
	for _, session := range sessions[1:] {
		if session.StartTime.Before(earliest) {
			earliest = session.StartTime
		}
	}
	return &earliest
}

// GetLatestEndTime returns the latest end time among sessions
func GetLatestEndTime(sessions []*Session) *time.Time {
	if len(sessions) == 0 {
		return nil
	}

	latest := sessions[0].EndTime
	for _, session := range sessions[1:] {
		if session.EndTime.After(latest) {
			latest = session.EndTime
		}
	}
	return &latest
}
