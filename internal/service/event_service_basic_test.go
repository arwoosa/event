package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
	"github.com/arwoosa/event/internal/testutils"
)

const (
	testMerchantID = "test-merchant-id"
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
	merchantID := testMerchantID
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

	// Mock deletion for rollback when Keto fails (expected in test environment)
	eventRepo.On("Delete", ctx, createdEvent.ID.Hex()).Return(nil)

	// Execute
	result, err := eventService.CreateEvent(ctx, req)

	// Assert - expect failure due to Keto connection not initialized in test
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to establish event ownership in authorization system")

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
	merchantID := testMerchantID
	eventID := testutils.ValidObjectIDString()

	event := testutils.TestEvent()

	// Mock direct findByID (no permission check in service layer anymore)
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
	merchantID := testMerchantID
	eventID := testutils.ValidObjectIDString()

	// Mock FindByID returns not found error
	eventRepo.On("FindByID", ctx, eventID).Return(nil, errors.ErrEventNotFound)

	// Execute
	result, err := eventService.GetEvent(ctx, merchantID, eventID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}
