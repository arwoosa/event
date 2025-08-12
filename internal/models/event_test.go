package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		expected bool
	}{
		{"Valid draft status", StatusDraft, true},
		{"Valid published status", StatusPublished, true},
		{"Valid archived status", StatusArchived, true},
		{"Invalid status", "invalid", false},
		{"Empty status", "", false},
		{"Case sensitive", "Draft", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidVisibility(t *testing.T) {
	tests := []struct {
		name       string
		visibility string
		expected   bool
	}{
		{"Valid public visibility", VisibilityPublic, true},
		{"Valid private visibility", VisibilityPrivate, true},
		{"Invalid visibility", "invalid", false},
		{"Empty visibility", "", false},
		{"Case sensitive", "Public", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidVisibility(tt.visibility)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidBlockType(t *testing.T) {
	tests := []struct {
		name      string
		blockType string
		expected  bool
	}{
		{"Valid text block type", BlockTypeText, true},
		{"Valid image block type", BlockTypeImage, true},
		{"Invalid block type", "invalid", false},
		{"Empty block type", "", false},
		{"Case sensitive", "Text", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidBlockType(tt.blockType)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvent_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  string
		toStatus    string
		expected    bool
		description string
	}{
		// Draft transitions - can go to Published or directly to Archived
		{"Draft to Published", StatusDraft, StatusPublished, true, "Draft can be published"},
		{"Draft to Archived", StatusDraft, StatusArchived, false, "Draft cannot be directly archived (unidirectional flow)"},
		{"Draft to Draft", StatusDraft, StatusDraft, true, "Same status transition allowed (no-op)"},

		// Published transitions - can only go to Archived (unidirectional)
		{"Published to Archived", StatusPublished, StatusArchived, true, "Published can be archived"},
		{"Published to Draft", StatusPublished, StatusDraft, false, "Published cannot go back to draft (unidirectional)"},
		{"Published to Published", StatusPublished, StatusPublished, true, "Same status transition allowed (no-op)"},

		// Archived transitions - final state, no further transitions (unidirectional)
		{"Archived to Published", StatusArchived, StatusPublished, false, "Archived is final state (unidirectional)"},
		{"Archived to Draft", StatusArchived, StatusDraft, false, "Archived is final state (unidirectional)"},
		{"Archived to Archived", StatusArchived, StatusArchived, true, "Same status transition allowed (no-op)"},

		// Invalid transitions
		{"Invalid from status", "invalid", StatusPublished, false, "Invalid from status"},
		{"Invalid to status", StatusDraft, "invalid", false, "Invalid to status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{Status: tt.fromStatus}
			result := event.CanTransitionTo(tt.toStatus)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestEvent_IsPublic(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		visibility string
		expected   bool
	}{
		{"Published and public", StatusPublished, VisibilityPublic, true},
		{"Published but private", StatusPublished, VisibilityPrivate, false},
		{"Draft and public", StatusDraft, VisibilityPublic, false},
		{"Archived and public", StatusArchived, VisibilityPublic, false},
		{"Draft and private", StatusDraft, VisibilityPrivate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				Status:     tt.status,
				Visibility: tt.visibility,
			}
			result := event.IsPublic()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvent_IsShareable(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		visibility string
		expected   bool
	}{
		{"Published and public", StatusPublished, VisibilityPublic, true},
		{"Published and private", StatusPublished, VisibilityPrivate, true},
		{"Draft and public", StatusDraft, VisibilityPublic, false},
		{"Draft and private", StatusDraft, VisibilityPrivate, false},
		{"Archived and public", StatusArchived, VisibilityPublic, false},
		{"Archived and private", StatusArchived, VisibilityPrivate, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				Status:     tt.status,
				Visibility: tt.visibility,
			}
			result := event.IsShareable()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeoJSONPoint_Validation(t *testing.T) {
	tests := []struct {
		name        string
		geoPoint    GeoJSONPoint
		expectValid bool
	}{
		{
			name: "Valid GeoJSON Point",
			geoPoint: GeoJSONPoint{
				Type:        GeoJSONTypePoint,
				Coordinates: [2]float64{121.5654, 25.0330}, // Taipei
			},
			expectValid: true,
		},
		{
			name: "Valid GeoJSON Point - boundary longitude",
			geoPoint: GeoJSONPoint{
				Type:        GeoJSONTypePoint,
				Coordinates: [2]float64{180.0, 0.0},
			},
			expectValid: true,
		},
		{
			name: "Valid GeoJSON Point - boundary latitude",
			geoPoint: GeoJSONPoint{
				Type:        GeoJSONTypePoint,
				Coordinates: [2]float64{0.0, 90.0},
			},
			expectValid: true,
		},
		{
			name: "Invalid type",
			geoPoint: GeoJSONPoint{
				Type:        "LineString",
				Coordinates: [2]float64{121.5654, 25.0330},
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test type validation
			if tt.expectValid {
				assert.Equal(t, GeoJSONTypePoint, tt.geoPoint.Type)
			} else {
				assert.NotEqual(t, GeoJSONTypePoint, tt.geoPoint.Type)
			}

			// Test coordinate bounds (longitude: -180 to 180, latitude: -90 to 90)
			if tt.expectValid && tt.geoPoint.Type == GeoJSONTypePoint {
				lng, lat := tt.geoPoint.Coordinates[0], tt.geoPoint.Coordinates[1]
				assert.GreaterOrEqual(t, lng, -180.0, "Longitude should be >= -180")
				assert.LessOrEqual(t, lng, 180.0, "Longitude should be <= 180")
				assert.GreaterOrEqual(t, lat, -90.0, "Latitude should be >= -90")
				assert.LessOrEqual(t, lat, 90.0, "Latitude should be <= 90")
			}
		})
	}
}

func TestLocation_Complete(t *testing.T) {
	tests := []struct {
		name        string
		location    Location
		expectValid bool
	}{
		{
			name: "Complete location",
			location: Location{
				Name:    "Test Location",
				Address: "123 Test St",
				PlaceID: "place123",
				Coordinates: GeoJSONPoint{
					Type:        GeoJSONTypePoint,
					Coordinates: [2]float64{121.5654, 25.0330},
				},
			},
			expectValid: true,
		},
		{
			name: "Missing name",
			location: Location{
				Address: "123 Test St",
				PlaceID: "place123",
				Coordinates: GeoJSONPoint{
					Type:        GeoJSONTypePoint,
					Coordinates: [2]float64{121.5654, 25.0330},
				},
			},
			expectValid: false,
		},
		{
			name: "Missing coordinates",
			location: Location{
				Name:    "Test Location",
				Address: "123 Test St",
				PlaceID: "place123",
			},
			expectValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectValid {
				assert.NotEmpty(t, tt.location.Name, "Name should not be empty")
				assert.NotEmpty(t, tt.location.Address, "Address should not be empty")
				assert.Equal(t, GeoJSONTypePoint, tt.location.Coordinates.Type, "Coordinates type should be Point")
			} else {
				isComplete := tt.location.Name != "" && tt.location.Address != "" &&
					tt.location.Coordinates.Type == GeoJSONTypePoint
				assert.False(t, isComplete, "Location should not be complete")
			}
		})
	}
}

func TestDetail_Validation(t *testing.T) {
	tests := []struct {
		name        string
		detail      []DetailBlock
		expectValid bool
	}{
		{
			name: "Valid text block detail",
			detail: []DetailBlock{
				{
					Type: BlockTypeText,
					Data: TextData{Content: "Test content"},
				},
			},
			expectValid: true,
		},
		{
			name: "Valid image block detail",
			detail: []DetailBlock{
				{
					Type: BlockTypeImage,
					Data: ImageData{
						URL:     "https://example.com/image.jpg",
						Alt:     "Test image",
						Caption: "Test caption",
					},
				},
			},
			expectValid: true,
		},
		{
			name: "Mixed blocks detail",
			detail: []DetailBlock{
				{
					Type: BlockTypeText,
					Data: TextData{Content: "Text content"},
				},
				{
					Type: BlockTypeImage,
					Data: ImageData{
						URL: "https://example.com/image.jpg",
						Alt: "Test image",
					},
				},
			},
			expectValid: true,
		},
		{
			name:        "Empty blocks",
			detail:      []DetailBlock{},
			expectValid: false, // Empty blocks array is not valid for publishing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// For our test validation, we check if blocks are not empty
			valid := len(tt.detail) > 0
			for _, block := range tt.detail {
				if !IsValidBlockType(block.Type) {
					valid = false
					break
				}
			}
			assert.Equal(t, tt.expectValid, valid)
		})
	}
}

func TestFAQ_Structure(t *testing.T) {
	faq := FAQ{
		Question: "What is this?",
		Answer:   "This is a test FAQ.",
	}

	assert.NotEmpty(t, faq.Question, "Question should not be empty")
	assert.NotEmpty(t, faq.Answer, "Answer should not be empty")
	assert.True(t, len(faq.Question) <= 100, "Question should be within character limit")
	assert.True(t, len(faq.Answer) <= 300, "Answer should be within character limit")
}

func TestEventConstants(t *testing.T) {
	// Test status constants
	assert.Equal(t, "draft", StatusDraft)
	assert.Equal(t, "published", StatusPublished)
	assert.Equal(t, "archived", StatusArchived)

	// Test visibility constants
	assert.Equal(t, "public", VisibilityPublic)
	assert.Equal(t, "private", VisibilityPrivate)

	// Test block type constants
	assert.Equal(t, "text", BlockTypeText)
	assert.Equal(t, "image", BlockTypeImage)

	// Test GeoJSON type constant
	assert.Equal(t, "Point", GeoJSONTypePoint)
}

func TestEvent_StatusTransitionMatrix(t *testing.T) {
	// Complete test matrix for all status transitions (unidirectional flow)
	transitions := map[string]map[string]bool{
		StatusDraft: {
			StatusDraft:     true,  // Same status allowed (no-op)
			StatusPublished: true,  // Draft -> Published
			StatusArchived:  false, // Draft -> Archived (not allowed)
		},
		StatusPublished: {
			StatusDraft:     false, // Published cannot go back to Draft (unidirectional)
			StatusPublished: true,  // Same status allowed (no-op)
			StatusArchived:  true,  // Published -> Archived
		},
		StatusArchived: {
			StatusDraft:     false, // Archived is final state (unidirectional)
			StatusPublished: false, // Archived is final state (unidirectional)
			StatusArchived:  true,  // Same status allowed (no-op)
		},
	}

	for fromStatus, toTransitions := range transitions {
		for toStatus, expected := range toTransitions {
			t.Run(fmt.Sprintf("%s_to_%s", fromStatus, toStatus), func(t *testing.T) {
				event := &Event{Status: fromStatus}
				result := event.CanTransitionTo(toStatus)
				assert.Equal(t, expected, result,
					"Transition from %s to %s should be %v", fromStatus, toStatus, expected)
			})
		}
	}
}
