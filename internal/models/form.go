package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventForm represents the form entity for events
type EventForm struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	EventID   primitive.ObjectID `json:"event_id" bson:"event_id"`
	Schema    interface{}        `json:"schema" bson:"schema"`     // JSON Schema defining data structure and validation rules
	UISchema  interface{}        `json:"uischema" bson:"uischema"` // UI Schema defining form layout and appearance
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	CreatedBy string             `json:"created_by" bson:"created_by"`
	UpdatedAt time.Time          `json:"updated_at" bson:"updated_at"`
	UpdatedBy string             `json:"updated_by" bson:"updated_by"`
}

// SetUpdateInfo updates the UpdatedAt and UpdatedBy fields
func (f *EventForm) SetUpdateInfo(updatedBy string) {
	f.UpdatedAt = time.Now()
	f.UpdatedBy = updatedBy
}

// SetCreateInfo sets the creation timestamp and creator
func (f *EventForm) SetCreateInfo(createdBy string) {
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now
	f.CreatedBy = createdBy
	f.UpdatedBy = createdBy
}

// HasValidEvent checks if the form has a valid event ID
func (f *EventForm) HasValidEvent() bool {
	return f.EventID != primitive.NilObjectID
}
