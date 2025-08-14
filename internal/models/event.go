package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	
	"event/internal/errors"
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
	Sessions      []Session          `json:"sessions" bson:"sessions,omitempty"` // Populated from aggregation, not stored in events collection
	Detail        []DetailBlock      `json:"detail" bson:"detail"`
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

// DetailBlock represents a single content block
type DetailBlock struct {
	Type string      `json:"type" bson:"type"` // text, image
	Data interface{} `json:"data" bson:"data"`
}

// TextData represents text block data
type TextData struct {
	Content string `json:"content" bson:"content"`
}

// ImageData represents image block data
type ImageData struct {
	URL     string `json:"url" bson:"url"`
	Alt     string `json:"alt" bson:"alt"`
	Caption string `json:"caption" bson:"caption"`
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

// Detail block type constants
const (
	BlockTypeText  = "text"
	BlockTypeImage = "image"
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

// IsValidBlockType checks if the block type is valid
func IsValidBlockType(blockType string) bool {
	return blockType == BlockTypeText || blockType == BlockTypeImage
}

// CanTransitionTo checks if the event can transition to the new status
// Implements unidirectional state flow: Draft → Published → Archived
func (e *Event) CanTransitionTo(newStatus string) bool {
	// Same status transitions are allowed (no-op)
	if e.Status == newStatus {
		return true
	}

	switch e.Status {
	case StatusDraft:
		// Draft can go to Published (unidirectional flow)
		return newStatus == StatusPublished
	case StatusPublished:
		// Published can only go to Archived (unidirectional flow)
		return newStatus == StatusArchived
	case StatusArchived:
		// Archived is final state - no transitions allowed (unidirectional flow)
		return false
	}
	return false
}

func (e *Event) IsValidStatusForUpdate() error {
	// Archived events cannot be updated
	if e.Status == StatusArchived {
		return errors.NewBusinessError("ARCHIVED_IMMUTABLE", "archived events cannot be updated", nil)
	}
	return nil
}

func (e *Event) IsValidStatusForDelete() error {
	switch e.Status {
	case StatusDraft:
		// Draft events can be deleted
		return nil
	case StatusPublished:
		return errors.NewBusinessError("PUBLISHED_IMMUTABLE", "published events cannot be deleted", nil)
	case StatusArchived:
		return errors.NewBusinessError("ARCHIVED_IMMUTABLE", "archived events cannot be deleted", nil)
	}
	return nil
}

// IsPublic checks if the event is published and public (visible in search)
func (e *Event) IsPublic() bool {
	return e.Status == StatusPublished && e.Visibility == VisibilityPublic
}

// IsShareable checks if the event can be shared via direct link
func (e *Event) IsShareable() bool {
	return e.Status == StatusPublished
}
