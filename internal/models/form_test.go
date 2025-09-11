package models

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	testCreatorID = "creator123"
	testUpdaterID = "updater456"
)

func TestEventForm_SetCreateInfo(t *testing.T) {
	form := &EventForm{
		ID:      primitive.NewObjectID(),
		EventID: primitive.NewObjectID(),
	}

	createdBy := "user123"
	beforeTime := time.Now().Add(-1 * time.Second)

	form.SetCreateInfo(createdBy)

	afterTime := time.Now().Add(1 * time.Second)

	if form.CreatedBy != createdBy {
		t.Errorf("Expected CreatedBy = %s, got %s", createdBy, form.CreatedBy)
	}

	if form.UpdatedBy != createdBy {
		t.Errorf("Expected UpdatedBy = %s, got %s", createdBy, form.UpdatedBy)
	}

	if form.CreatedAt.Before(beforeTime) || form.CreatedAt.After(afterTime) {
		t.Errorf("CreatedAt timestamp should be between %v and %v, got %v", beforeTime, afterTime, form.CreatedAt)
	}

	if form.UpdatedAt.Before(beforeTime) || form.UpdatedAt.After(afterTime) {
		t.Errorf("UpdatedAt timestamp should be between %v and %v, got %v", beforeTime, afterTime, form.UpdatedAt)
	}

	// CreatedAt and UpdatedAt should be equal for new forms
	if !form.CreatedAt.Equal(form.UpdatedAt) {
		t.Errorf("CreatedAt and UpdatedAt should be equal for new forms, got CreatedAt: %v, UpdatedAt: %v", form.CreatedAt, form.UpdatedAt)
	}
}

func TestEventForm_SetUpdateInfo(t *testing.T) {
	form := &EventForm{
		ID:        primitive.NewObjectID(),
		EventID:   primitive.NewObjectID(),
		CreatedAt: time.Now().Add(-1 * time.Hour),
		CreatedBy: testCreatorID,
		UpdatedAt: time.Now().Add(-1 * time.Hour),
		UpdatedBy: testCreatorID,
	}

	updatedBy := testUpdaterID
	beforeUpdate := time.Now().Add(-1 * time.Second)

	form.SetUpdateInfo(updatedBy)

	afterUpdate := time.Now().Add(1 * time.Second)

	if form.UpdatedBy != updatedBy {
		t.Errorf("Expected UpdatedBy = %s, got %s", updatedBy, form.UpdatedBy)
	}

	if form.UpdatedAt.Before(beforeUpdate) || form.UpdatedAt.After(afterUpdate) {
		t.Errorf("UpdatedAt timestamp should be between %v and %v, got %v", beforeUpdate, afterUpdate, form.UpdatedAt)
	}

	// CreatedBy and CreatedAt should remain unchanged
	if form.CreatedBy != testCreatorID {
		t.Errorf("CreatedBy should remain unchanged, got %s", form.CreatedBy)
	}
}

func TestEventForm_HasValidEvent(t *testing.T) {
	tests := []struct {
		name     string
		form     *EventForm
		expected bool
	}{
		{
			name: "valid event ID",
			form: &EventForm{
				EventID: primitive.NewObjectID(),
			},
			expected: true,
		},
		{
			name: "nil event ID",
			form: &EventForm{
				EventID: primitive.NilObjectID,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.form.HasValidEvent(); got != tt.expected {
				t.Errorf("EventForm.HasValidEvent() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestEventForm_CompleteWorkflow(t *testing.T) {
	// Create new form
	form := &EventForm{
		EventID: primitive.NewObjectID(),
		Schema: map[string]interface{}{
			"type": "object",
		},
		UISchema: map[string]interface{}{
			"type": "VerticalLayout",
		},
	}

	// Set creation info
	form.SetCreateInfo(testCreatorID)

	// Validate initial state
	if !form.HasValidEvent() {
		t.Error("Form should have valid event ID")
	}

	// Simulate update
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	form.SetUpdateInfo(testUpdaterID)

	// Validate after update
	if form.CreatedBy != testCreatorID {
		t.Errorf("CreatedBy should remain 'creator123', got %s", form.CreatedBy)
	}

	if form.UpdatedBy != testUpdaterID {
		t.Errorf("UpdatedBy should be 'updater456', got %s", form.UpdatedBy)
	}

	if !form.UpdatedAt.After(form.CreatedAt) {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}
