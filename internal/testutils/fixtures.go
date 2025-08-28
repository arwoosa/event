package testutils

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/models"
)

const (
	TestMerchantIDValue = "test-merchant-id"
)

// TestEvent creates a test event with default values
func TestEvent() *models.Event {
	return &models.Event{
		ID:            primitive.NewObjectID(),
		Title:         "Test Event",
		Summary:       "Test event summary",
		Status:        models.StatusDraft,
		Visibility:    models.VisibilityPrivate,
		CoverImageURL: "https://example.com/image.jpg",
		Location: models.Location{
			Name:    "Test Location",
			Address: "123 Test St, Test City",
			PlaceID: "test_place_id",
			Coordinates: models.GeoJSONPoint{
				Type:        models.GeoJSONTypePoint,
				Coordinates: [2]float64{121.5654, 25.0330}, // Taipei coordinates
			},
		},
		Detail: []models.DetailBlock{
			{
				Type: models.BlockTypeText,
				Data: models.TextData{Content: "Test event content"},
			},
			{
				Type: models.BlockTypeImage,
				Data: models.ImageData{
					URL:     "https://example.com/test-image.jpg",
					Alt:     "Test image",
					Caption: "This is a test image",
				},
			},
		},
		FAQ: []models.FAQ{
			{
				Question: "What is this event about?",
				Answer:   "This is a test event for testing purposes.",
			},
		},
		CreatedAt: time.Now(),
		CreatedBy: primitive.NewObjectID(),
		UpdatedAt: time.Now(),
		UpdatedBy: primitive.NewObjectID(),
	}
}

// TestEventWithStatus creates a test event with specified status
func TestEventWithStatus(status string) *models.Event {
	event := TestEvent()
	event.Status = status
	return event
}

// TestPublishedEvent creates a published test event
func TestPublishedEvent() *models.Event {
	return TestEventWithStatus(models.StatusPublished)
}

// TestArchivedEvent creates an archived test event
func TestArchivedEvent() *models.Event {
	return TestEventWithStatus(models.StatusArchived)
}

// TestSession creates a test session with default values
func TestSession() *models.Session {
	now := time.Now()
	capacity := 50 // Default capacity for testing
	return &models.Session{
		ID:        primitive.NewObjectID(),
		EventID:   primitive.NewObjectID(),
		Name:      "Test Session",          // New field
		Capacity:  &capacity,               // New field (pointer to int)
		StartTime: now.Add(time.Hour * 24), // Tomorrow
		EndTime:   now.Add(time.Hour * 26), // Tomorrow + 2 hours
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// TestSessionWithTimes creates a test session with specified times
func TestSessionWithTimes(startTime, endTime time.Time) *models.Session {
	session := TestSession()
	session.StartTime = startTime
	session.EndTime = endTime
	return session
}

// TestSessionsForEvent creates multiple test sessions for an event
func TestSessionsForEvent(eventID primitive.ObjectID, count int) []*models.Session {
	sessions := make([]*models.Session, count)
	baseTime := time.Now().Add(time.Hour * 24) // Start tomorrow

	for i := 0; i < count; i++ {
		startTime := baseTime.Add(time.Duration(i*3) * time.Hour) // 3 hours apart
		endTime := startTime.Add(time.Hour * 2)                   // 2 hours duration
		capacity := 30 + (i * 10)                                 // Different capacities for testing

		sessions[i] = &models.Session{
			ID:        primitive.NewObjectID(),
			EventID:   eventID,
			Name:      fmt.Sprintf("Session %d", i+1), // New field
			Capacity:  &capacity,                      // New field
			StartTime: startTime,
			EndTime:   endTime,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
	}

	return sessions
}

// Service request data for testing (avoiding circular imports)
type TestLocationRequest struct {
	Name        string
	Address     string
	PlaceID     string
	Coordinates *TestGeoJSONPointRequest
}

type TestGeoJSONPointRequest struct {
	Type        string
	Coordinates [2]float64
}

type TestSessionRequest struct {
	ID        string
	Name      string
	Capacity  *int
	StartTime string
	EndTime   string
}

type TestDetailBlockRequest struct {
	Type string
	Data interface{}
}

type TestFAQRequest struct {
	Question string
	Answer   string
}

type TestCreateEventRequest struct {
	Title         string
	Summary       string
	Visibility    string
	CoverImageURL string
	Location      *TestLocationRequest
	Sessions      []*TestSessionRequest
	Detail        []TestDetailBlockRequest
	FAQ           []*TestFAQRequest
	UserID        string
}

type TestPatchEventRequest struct {
	ID       string
	Title    *string
	Summary  *string
	Sessions []*TestSessionRequest
	UserID   string
}

type TestSearchEventsRequest struct {
	TitleSearch *string
	PageSize    *int32
}

// CreateTestCreateEventRequest creates a test create event request
func CreateTestCreateEventRequest() *TestCreateEventRequest {
	return &TestCreateEventRequest{
		Title:         "Test Event",
		Summary:       "Test event summary",
		Visibility:    models.VisibilityPrivate,
		CoverImageURL: "https://example.com/image.jpg",
		Location: &TestLocationRequest{
			Name:    "Test Location",
			Address: "123 Test St, Test City",
			PlaceID: "test_place_id",
			Coordinates: &TestGeoJSONPointRequest{
				Type:        models.GeoJSONTypePoint,
				Coordinates: [2]float64{121.5654, 25.0330},
			},
		},
		Sessions: []*TestSessionRequest{
			{
				StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
				EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
			},
		},
		Detail: []TestDetailBlockRequest{
			{
				Type: models.BlockTypeText,
				Data: models.TextData{Content: "Test event content"},
			},
			{
				Type: models.BlockTypeImage,
				Data: models.ImageData{
					URL:     "https://example.com/test-image.jpg",
					Alt:     "Test image",
					Caption: "This is a test image",
				},
			},
		},
		FAQ: []*TestFAQRequest{
			{
				Question: "What is this event about?",
				Answer:   "This is a test event for testing purposes.",
			},
		},
		UserID: primitive.NewObjectID().Hex(),
	}
}

// CreateTestPatchEventRequest creates a test patch event request
func CreateTestPatchEventRequest(eventID string) *TestPatchEventRequest {
	title := "Patched Test Event"
	summary := "Patched test event summary"

	return &TestPatchEventRequest{
		ID:      eventID,
		Title:   &title,
		Summary: &summary,
		UserID:  primitive.NewObjectID().Hex(),
	}
}

// CreateTestSearchEventsRequest creates a test search events request
func CreateTestSearchEventsRequest() *TestSearchEventsRequest {
	titleSearch := "test"
	pageSize := int32(20)

	return &TestSearchEventsRequest{
		TitleSearch: &titleSearch,
		PageSize:    &pageSize,
	}
}

// TestUser creates a test user ID
func TestUserID() primitive.ObjectID {
	return primitive.NewObjectID()
}

// TestMerchantID creates a test merchant ID
func TestMerchantID() string {
	return TestMerchantIDValue
}

// InvalidObjectID returns an invalid ObjectID string for testing
func InvalidObjectID() string {
	return "invalid_object_id"
}

// ValidObjectIDString returns a valid ObjectID string for testing
func ValidObjectIDString() string {
	return primitive.NewObjectID().Hex()
}
