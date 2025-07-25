package models

import (
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// Event represents the main event entity
type Event struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Title         string             `json:"title" bson:"title"`
	BrandID       primitive.ObjectID `json:"brand_id" bson:"brand_id"`
	Summary       string             `json:"summary" bson:"summary"`
	Status        string             `json:"status" bson:"status"`         // draft, published, archived
	Visibility    string             `json:"visibility" bson:"visibility"` // public, private
	CoverImageURL string             `json:"cover_image_url" bson:"cover_image_url"`
	Location      Location           `json:"location" bson:"location"`
	Sessions      []Session          `json:"sessions" bson:"sessions"`
	Detail        Detail             `json:"detail" bson:"detail"`
	FAQ           []FAQ              `json:"faq" bson:"faq"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	CreatedBy     primitive.ObjectID `json:"created_by" bson:"created_by"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
	UpdatedBy     primitive.ObjectID `json:"updated_by" bson:"updated_by"`
}

// Location represents the event location with geospatial data
type Location struct {
	Name        string       `json:"name" bson:"name"`
	Address     string       `json:"address" bson:"address"`
	PlaceID     string       `json:"place_id" bson:"place_id"`
	Coordinates GeoJSONPoint `json:"coordinates" bson:"coordinates"`
}

// GeoJSONPoint represents a GeoJSON Point for MongoDB geospatial indexing
type GeoJSONPoint struct {
	Type        string     `json:"type" bson:"type"`               // Always "Point"
	Coordinates [2]float64 `json:"coordinates" bson:"coordinates"` // [longitude, latitude]
}

// Session represents a time-based event session
type Session struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	StartTime time.Time          `json:"start_time" bson:"start_time"`
	EndTime   time.Time          `json:"end_time" bson:"end_time"`
}

// Detail represents the event detail content
type Detail struct {
	Content     string `json:"content" bson:"content"`
	ContentType string `json:"content_type" bson:"content_type"` // html, json, markdown
}

// FAQ represents a frequently asked question entry
type FAQ struct {
	Question string `json:"question" bson:"question"`
	Answer   string `json:"answer" bson:"answer"`
}

// Event status constants
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

// Event visibility constants
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// Content type constants
const (
	ContentTypeHTML     = "html"
	ContentTypeJSON     = "json"
	ContentTypeMarkdown = "markdown"
)

// GeoJSON type constants
const (
	GeoJSONTypePoint = "Point"
)

// IsValidStatus checks if the status is valid
func IsValidStatus(status string) bool {
	return status == StatusDraft || status == StatusPublished || status == StatusArchived
}

// IsValidVisibility checks if the visibility is valid
func IsValidVisibility(visibility string) bool {
	return visibility == VisibilityPublic || visibility == VisibilityPrivate
}

// IsValidContentType checks if the content type is valid
func IsValidContentType(contentType string) bool {
	return contentType == ContentTypeHTML || contentType == ContentTypeJSON || contentType == ContentTypeMarkdown
}

// CanTransitionTo checks if the event can transition to the new status
func (e *Event) CanTransitionTo(newStatus string) bool {
	switch e.Status {
	case StatusDraft:
		return newStatus == StatusPublished
	case StatusPublished:
		return newStatus == StatusArchived
	case StatusArchived:
		return newStatus == StatusPublished || newStatus == StatusDraft
	}
	return false
}

// IsPublic checks if the event is published and public (visible in search)
func (e *Event) IsPublic() bool {
	return e.Status == StatusPublished && e.Visibility == VisibilityPublic
}

// IsShareable checks if the event can be shared via direct link
func (e *Event) IsShareable() bool {
	return e.Status == StatusPublished
}

// HasSessions validates that the event has at least one session
func (e *Event) HasSessions() bool {
	return len(e.Sessions) > 0
}

// ValidateSessions checks for session overlaps and time validity
func (e *Event) ValidateSessions() error {
	if !e.HasSessions() {
		return errors.New("event must have at least one session")
	}

	// Check each session's time validity
	for i, session := range e.Sessions {
		if !session.StartTime.Before(session.EndTime) {
			return fmt.Errorf("session %d: start_time must be before end_time", i)
		}
	}

	// Check for overlapping sessions
	for i := 0; i < len(e.Sessions); i++ {
		for j := i + 1; j < len(e.Sessions); j++ {
			if e.Sessions[i].OverlapsWith(e.Sessions[j]) {
				return fmt.Errorf("sessions %d and %d overlap", i, j)
			}
		}
	}

	return nil
}

// OverlapsWith checks if two sessions overlap in time
func (s *Session) OverlapsWith(other Session) bool {
	return s.StartTime.Before(other.EndTime) && other.StartTime.Before(s.EndTime)
}

// GetEarliestSessionTime returns the earliest start time among all sessions
func (e *Event) GetEarliestSessionTime() *time.Time {
	if !e.HasSessions() {
		return nil
	}

	earliest := e.Sessions[0].StartTime
	for _, session := range e.Sessions[1:] {
		if session.StartTime.Before(earliest) {
			earliest = session.StartTime
		}
	}
	return &earliest
}

// GetLatestSessionTime returns the latest end time among all sessions
func (e *Event) GetLatestSessionTime() *time.Time {
	if !e.HasSessions() {
		return nil
	}

	latest := e.Sessions[0].EndTime
	for _, session := range e.Sessions[1:] {
		if session.EndTime.After(latest) {
			latest = session.EndTime
		}
	}
	return &latest
}
