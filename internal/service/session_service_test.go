package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/models"
	"event/internal/service/mocks"
	"event/internal/testutils"
)

func TestSessionService_CreateSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

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
	event.BrandID, _ = primitive.ObjectIDFromHex(brandID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Mock successful session creation
	createdSessions := testutils.TestSessionsForEvent(event.ID, 2)
	sessionRepo.On("CreateBatch", ctx, testutils.MatchAnySessionSlice()).Return(createdSessions, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, brandID, sessionReqs)

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

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event not found
	eventRepo.On("FindByID", ctx, eventID).Return(nil, models.ErrEventNotFound)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, brandID, sessionReqs)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, models.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_CreateSessionsForEvent_WrongBrand(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()
	wrongBrandID := testutils.ValidObjectIDString()

	sessionReqs := []*SessionRequest{
		{
			StartTime: time.Now().Add(time.Hour * 24).Format(time.RFC3339),
			EndTime:   time.Now().Add(time.Hour * 26).Format(time.RFC3339),
		},
	}

	// Mock event with different brand
	event := testutils.TestEvent()
	event.BrandID, _ = primitive.ObjectIDFromHex(wrongBrandID) // Different brand
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	result, err := sessionService.CreateSessionsForEvent(ctx, eventID, brandID, sessionReqs)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, models.ErrUnauthorized, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	// Mock existence check
	eventRepo.On("ExistsByBrandAndID", ctx, brandID, eventID).Return(true, nil)

	// Mock sessions retrieval
	sessions := testutils.TestSessionsForEvent(primitive.NewObjectID(), 3)
	sessionRepo.On("FindByEventID", ctx, eventID).Return(sessions, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID, brandID)

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

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	// Mock existence check returns false
	eventRepo.On("ExistsByBrandAndID", ctx, brandID, eventID).Return(false, nil)

	// Execute
	result, err := sessionService.GetSessionsForEvent(ctx, eventID, brandID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, models.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_Success(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	// Create session with matching brand
	session := testutils.TestSession()
	session.BrandID, _ = primitive.ObjectIDFromHex(brandID)

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID, brandID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, session.ID, result.ID)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_GetSession_UnauthorizedBrand(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()
	wrongBrandID := testutils.ValidObjectIDString()

	// Create session with different brand
	session := testutils.TestSession()
	session.BrandID, _ = primitive.ObjectIDFromHex(wrongBrandID) // Different brand

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Execute
	result, err := sessionService.GetSession(ctx, sessionID, brandID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, models.ErrUnauthorized, err)

	sessionRepo.AssertExpectations(t)
}

func TestSessionService_DeleteSession_Success(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	// Create session
	session := testutils.TestSession()
	session.BrandID, _ = primitive.ObjectIDFromHex(brandID)

	// Create event (draft, so modifiable)
	event := testutils.TestEvent()
	event.Status = models.StatusDraft

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Mock event retrieval for validation
	eventRepo.On("FindByID", ctx, session.EventID.Hex()).Return(event, nil)

	// Mock session count check (more than 1, so can delete)
	sessionRepo.On("CountByEventID", ctx, session.EventID.Hex()).Return(int64(2), nil)

	// Mock deletion
	sessionRepo.On("Delete", ctx, sessionID).Return(nil)

	// Execute
	err := sessionService.DeleteSession(ctx, sessionID, brandID)

	// Assert
	require.NoError(t, err)

	sessionRepo.AssertExpectations(t)
	eventRepo.AssertExpectations(t)
}

func TestSessionService_DeleteSession_LastSession(t *testing.T) {
	// Setup
	sessionRepo := &mocks.MockSessionRepository{}
	eventRepo := &mocks.MockEventRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	sessionID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

	// Create session
	session := testutils.TestSession()
	session.BrandID, _ = primitive.ObjectIDFromHex(brandID)

	// Create event (draft, so modifiable)
	event := testutils.TestEvent()
	event.Status = models.StatusDraft

	// Mock session retrieval
	sessionRepo.On("FindByID", ctx, sessionID).Return(session, nil)

	// Mock event retrieval for validation
	eventRepo.On("FindByID", ctx, session.EventID.Hex()).Return(event, nil)

	// Mock session count check (only 1 session, so cannot delete)
	sessionRepo.On("CountByEventID", ctx, session.EventID.Hex()).Return(int64(1), nil)

	// Execute
	err := sessionService.DeleteSession(ctx, sessionID, brandID)

	// Assert
	require.Error(t, err)
	testutils.AssertError(t, err, "LAST_SESSION", "cannot delete the last session of an event")

	sessionRepo.AssertExpectations(t)
	eventRepo.AssertExpectations(t)
}

func TestSessionService_ValidateSessionsForEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

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
	event.BrandID, _ = primitive.ObjectIDFromHex(brandID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, brandID, sessionReqs)

	// Assert
	require.NoError(t, err)

	eventRepo.AssertExpectations(t)
}

func TestSessionService_ValidateSessionsForEvent_DuplicateTimes(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionRepo := &mocks.MockSessionRepository{}

	sessionService := NewSessionService(sessionRepo, eventRepo, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()
	brandID := testutils.ValidObjectIDString()

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
	event.BrandID, _ = primitive.ObjectIDFromHex(brandID)
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)

	// Execute
	err := sessionService.ValidateSessionsForEvent(ctx, eventID, brandID, sessionReqs)

	// Assert
	require.Error(t, err)
	testutils.AssertError(t, err, "VALIDATION_ERROR", "have identical start and end times")

	eventRepo.AssertExpectations(t)
}
