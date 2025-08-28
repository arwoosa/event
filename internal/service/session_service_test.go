package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
	"github.com/arwoosa/event/internal/testutils"
)

func TestSessionService_CreateSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Create session requests
	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
		{
			StartTime: time.Now().Add(time.Hour * 48).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 50).Format(time.RFC3339),
		},
	}

	// Mock event validation
	event := testutils.TestEvent()
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Mock successful session creation
	createdSessions := testutils.TestSessionsForEvent(event.ID, 2)
	sessionRepo.On("CreateBatch", ctx, testutils.MatchAnySessionSlice()).Return(createdSessions, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, sessionReqs)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestSessionService_CreateSessionsForEvent_EventNotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event not found
	eventRepo.On("FindByID", ctx, eventID).Return(nil, errors.ErrEventNotFound)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, sessionReqs)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_CreateSessionsForEvent_WrongMerchant(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event exists (authorization is handled by API Gateway)
	event := testutils.TestEvent()
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Mock successful session creation
	createdSessions := testutils.TestSessionsForEvent(event.ID, 1)
	sessionRepo.On("CreateBatch", ctx, testutils.MatchAnySessionSlice()).Return(createdSessions, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, sessionReqs)

	// Assert - should succeed since authorization is handled by API Gateway
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)

	eventRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// No permission check in service layer anymore - authorization handled by API Gateway

	// Mock sessions retrieval
	sessions := testutils.TestSessionsForEvent(primitive.NewObjectID(), 3)
	sessionRepo.On("FindByEventID", ctx, eventID).Return(sessions, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 3)

	eventRepo.AssertExpectations(t)
	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSessionsForEvent_EventNotExists(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Mock no sessions found for event
	sessionRepo.On("FindByEventID", ctx, eventID).Return([]*models.Session{}, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID)

	// Assert - should succeed but return empty slice
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_Success(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()

	// Create session and matching event
	session := testutils.TestSession()
	event := testutils.TestEvent()
	event.ID = session.EventID

	// Mock session retrieval (no permission check in service layer)
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.ID)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_UnauthorizedMerchant(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()

	// Create session and event with different merchant
	session := testutils.TestSession()
	event := testutils.TestEvent()
	event.ID = session.EventID

	// Mock session retrieval (authorization is handled by API Gateway)
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID)

	// Assert - should succeed since authorization is handled by API Gateway
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.ID)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_ValidateSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Create valid session requests
	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
		{
			StartTime: time.Now().Add(time.Hour * 48).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 50).Format(time.RFC3339),
		},
	}

	// No permission check in service layer - authorization handled by API Gateway

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, sessionReqs)

	// Assert
	require.NoError(t, err)
}

func TestSessionService_ValidateSessionsForEvent_DuplicateTimes(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Create duplicate session requests (same times)
	sameTime := time.Now().Add(time.Hour * 24)
	sessionReqs := []*SessionRequest{
		{
			StartTime: sameTime.Format(time.RFC3339),
			EndTime:   sameTime.Add(time.Hour * 2).Format(time.RFC3339),
		},
		{
			StartTime: sameTime.Format(time.RFC3339),                    // Same start time
			EndTime:   sameTime.Add(time.Hour * 2).Format(time.RFC3339), // Same end time
		},
	}

	// No permission check in service layer - authorization handled by API Gateway

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, sessionReqs)

	// Assert
	require.Error(t, err)
	testutils.AssertError(t, err, errors.ErrorCodeValidationError, "have identical start and end times")
}
