package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"event/internal/errors"
	"event/internal/models"
	"event/internal/service/mocks"
	"event/internal/testutils"
)

func TestEventService_CreateEvent_WithoutSessions_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}
	orderService := &mocks.MockOrderService{}

	sessionService := NewSessionService(sessionRepo, eventRepo)
	eventService := NewEventService(eventRepo, sessionService, orderService)

	ctx := context.Background()

	// Create test request without sessions to avoid complexity
	merchantID := testutils.ValidObjectIDString()
	userID := testutils.ValidObjectIDString()

	req := &CreateEventRequest{
		Title:      "Test Event",
		Summary:    "Test Summary",
		MerchantID: merchantID,
		UserID:     userID,
		Visibility: models.VisibilityPrivate,
		// No sessions
	}

	// Mock successful event creation
	createdEvent := testutils.TestEvent()
	eventRepo.On("Create", ctx, testutils.MatchAnyEvent()).Return(createdEvent, nil)

	// Execute
	result, err := eventService.CreateEvent(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusDraft, result.Status)

	// Verify mocks
	eventRepo.AssertExpectations(t)
}

func TestEventService_GetEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{} // Minimal setup for this test
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, sessionService, orderService)

	ctx := context.Background()
	merchantID := testutils.ValidObjectIDString()
	eventID := testutils.ValidObjectIDString()

	event := testutils.TestEvent()

	// Mock existence check
	eventRepo.On("ExistsByMerchantAndID", ctx, merchantID, eventID).Return(true, nil)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	result, err := eventService.GetEvent(ctx, merchantID, eventID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, event.Title, result.Title)

	eventRepo.AssertExpectations(t)
}

func TestEventService_GetEvent_NotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, sessionService, orderService)

	ctx := context.Background()
	merchantID := testutils.ValidObjectIDString()
	eventID := testutils.ValidObjectIDString()

	// Mock existence check returns false
	eventRepo.On("ExistsByMerchantAndID", ctx, merchantID, eventID).Return(false, nil)

	// Execute
	result, err := eventService.GetEvent(ctx, merchantID, eventID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestEventService_CreateEvent_InvalidMerchantID(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}
	orderService := &mocks.MockOrderService{}

	sessionService := NewSessionService(sessionRepo, eventRepo)
	eventService := NewEventService(eventRepo, sessionService, orderService)

	ctx := context.Background()

	req := &CreateEventRequest{
		Title:      "Test Event",
		MerchantID: "invalid_id", // Invalid ObjectID
		UserID:     testutils.ValidObjectIDString(),
	}

	// Execute
	result, err := eventService.CreateEvent(ctx, req)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	testutils.AssertError(t, err, errors.ErrorCodeValidationError, "invalid merchant_id")
}
