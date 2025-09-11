package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
	"github.com/arwoosa/event/internal/testutils"
)

func TestEventService_CreateEvent_WithoutSessions_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}
	formRepo := &mocks.MockFormRepository{}
	orderService := &mocks.MockOrderService{}

	sessionService := NewSessionService(sessionRepo, eventRepo)
	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()

	// Create test request without sessions to avoid complexity
	userID := testutils.ValidObjectIDString()

	req := &CreateEventRequest{
		Title:      "Test Event",
		MerchantID: "test-merchant-123",
		Summary:    "Test Summary",
		UserID:     userID,
		Visibility: models.VisibilityPrivate,
		// No sessions
	}

	// Mock successful event creation
	createdEvent := testutils.TestEvent()
	eventRepo.On("Create", ctx, testutils.MatchAnyEvent()).Return(createdEvent, nil)

	// Execute
	result, err := eventService.CreateEvent(ctx, req)

	// Assert - should succeed since Keto code is now commented out
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, createdEvent.ID, result.ID)

	// Verify mocks
	eventRepo.AssertExpectations(t)
}

func TestEventService_GetEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{} // Minimal setup for this test
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	event := testutils.TestEvent()

	// Convert string ID to ObjectID for the mock
	objectID, err := primitive.ObjectIDFromHex(eventID)
	require.NoError(t, err)
	
	// Mock direct findByID (no permission check in service layer anymore)
	eventRepo.On("FindByID", ctx, objectID).Return(event, nil)

	// Execute
	result, err := eventService.GetEvent(ctx, eventID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, event.Title, result.Title)

	eventRepo.AssertExpectations(t)
}

func TestEventService_GetEvent_NotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Convert string ID to ObjectID for the mock
	objectID, err := primitive.ObjectIDFromHex(eventID)
	require.NoError(t, err)
	
	// Mock FindByID returns not found error
	eventRepo.On("FindByID", ctx, objectID).Return(nil, errors.ErrEventNotFound)

	// Execute
	result, err := eventService.GetEvent(ctx, eventID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}
