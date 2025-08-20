package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/errors"
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
	merchantID := testutils.ValidObjectIDString()

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
	event.MerchantID, _ = primitive.ObjectIDFromHex(merchantID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Mock successful session creation
	createdSessions := testutils.TestSessionsForEvent(event.ID, 2)
	sessionRepo.On("CreateBatch", ctx, testutils.MatchAnySessionSlice()).Return(createdSessions, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, merchantID, sessionReqs)

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
	merchantID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event not found
	eventRepo.On("FindByID", ctx, eventID).Return(nil, errors.ErrEventNotFound)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, merchantID, sessionReqs)

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
	merchantID := testutils.ValidObjectIDString()
	wrongMerchantID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event with different merchant
	event := testutils.TestEvent()
	event.MerchantID, _ = primitive.ObjectIDFromHex(wrongMerchantID) // Different merchant
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, merchantID, sessionReqs)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrUnauthorized, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	merchantID := testutils.ValidObjectIDString()

	// Mock existence check
	eventRepo.On("ExistsByMerchantAndID", ctx, merchantID, eventID).Return(true, nil)

	// Mock sessions retrieval
	sessions := testutils.TestSessionsForEvent(primitive.NewObjectID(), 3)
	sessionRepo.On("FindByEventID", ctx, eventID).Return(sessions, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID, merchantID)

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
	merchantID := testutils.ValidObjectIDString()

	// Mock existence check returns false
	eventRepo.On("ExistsByMerchantAndID", ctx, merchantID, eventID).Return(false, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID, merchantID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_Success(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	merchantID := testutils.ValidObjectIDString()

	// Create session and matching event
	session := testutils.TestSession()
	event := testutils.TestEvent()
	event.ID = session.EventID
	event.MerchantID, _ = primitive.ObjectIDFromHex(merchantID) // Set matching merchant

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)
	// Mock event retrieval for merchant validation (new requirement)
	eventRepo.On("FindByID", ctx, session.EventID.Hex()).Return(event, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID, merchantID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.ID)

	sessionRepo.AssertExpectations(t)
	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_UnauthorizedMerchant(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	merchantID := testutils.ValidObjectIDString()
	differentMerchantID := testutils.ValidObjectIDString()

	// Create session and event with different merchant
	session := testutils.TestSession()
	event := testutils.TestEvent()
	event.ID = session.EventID
	event.MerchantID, _ = primitive.ObjectIDFromHex(differentMerchantID) // Different merchant

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)
	// Mock event retrieval for merchant validation
	eventRepo.On("FindByID", ctx, session.EventID.Hex()).Return(event, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID, merchantID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrUnauthorized, err)

	sessionRepo.AssertExpectations(t)
	eventRepo.AssertExpectations(t)
}

func TestSessionService_ValidateSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	merchantID := testutils.ValidObjectIDString()

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

	// Mock event validation
	event := testutils.TestEvent()
	event.MerchantID, _ = primitive.ObjectIDFromHex(merchantID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, merchantID, sessionReqs)

	// Assert
	require.NoError(t, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_ValidateSessionsForEvent_DuplicateTimes(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	merchantID := testutils.ValidObjectIDString()

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

	// Mock event validation
	event := testutils.TestEvent()
	event.MerchantID, _ = primitive.ObjectIDFromHex(merchantID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, merchantID, sessionReqs)

	// Assert
	require.Error(t, err)
	testutils.AssertError(t, err, errors.ErrorCodeValidationError, "have identical start and end times")

	eventRepo.AssertExpectations(t)
}
