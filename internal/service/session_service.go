package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/dao/repository"
	"event/internal/models"
)

// SessionService implements the business logic for session management
type SessionService struct {
	sessionRepo repository.SessionRepository
	eventRepo   repository.EventRepository
}

// NewSessionService creates a new session service
func NewSessionService(
	sessionRepo repository.SessionRepository,
	eventRepo repository.EventRepository,
) *SessionService {
	return &SessionService{
		sessionRepo: sessionRepo,
		eventRepo:   eventRepo,
	}
}

// CreateSessionsForEvent creates sessions for an event
func (s *SessionService) CreateSessionsForEvent(ctx context.Context, eventID, brandID string, sessionReqs []*SessionRequest) ([]*models.Session, error) {
	// Validate event exists and belongs to brand
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.BrandID.Hex() != brandID {
		return nil, models.ErrUnauthorized
	}

	// Convert session requests to models
	sessions, err := s.convertSessionRequestsToModels(sessionReqs, eventID, brandID)
	if err != nil {
		return nil, err
	}

	// Validate sessions for duplicates
	if err := models.ValidateSessions(sessions); err != nil {
		return nil, models.NewValidationError("sessions", err.Error())
	}

	// Create sessions in batch
	return s.sessionRepo.CreateBatch(ctx, sessions)
}

// GetSessionsForEvent retrieves all sessions for an event
func (s *SessionService) GetSessionsForEvent(ctx context.Context, eventID, brandID string) ([]*models.Session, error) {
	// Verify event belongs to brand
	exists, err := s.eventRepo.ExistsByBrandAndID(ctx, brandID, eventID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.ErrEventNotFound
	}

	return s.sessionRepo.FindByEventID(ctx, eventID)
}

// GetSessionsForEvents retrieves sessions for multiple events (batch operation)
func (s *SessionService) GetSessionsForEvents(ctx context.Context, eventIDs []string) (map[string][]*models.Session, error) {
	return s.sessionRepo.FindByEventIDs(ctx, eventIDs)
}

// UpdateSessionsForEvent replaces all sessions for an event
func (s *SessionService) UpdateSessionsForEvent(ctx context.Context, eventID, brandID string, sessionReqs []*SessionRequest) ([]*models.Session, error) {
	// Validate event exists and belongs to brand
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.BrandID.Hex() != brandID {
		return nil, models.ErrUnauthorized
	}

	// Check if event can be modified
	if err := s.validateEventModification(event); err != nil {
		return nil, err
	}

	// Convert session requests to models
	newSessions, err := s.convertSessionRequestsToModels(sessionReqs, eventID, brandID)
	if err != nil {
		return nil, err
	}

	// Validate sessions for duplicates
	if err := models.ValidateSessions(newSessions); err != nil {
		return nil, models.NewValidationError("sessions", err.Error())
	}

	// Delete existing sessions and create new ones in transaction-like operation
	if err := s.sessionRepo.DeleteByEventID(ctx, eventID); err != nil {
		return nil, fmt.Errorf("failed to delete existing sessions: %w", err)
	}

	return s.sessionRepo.CreateBatch(ctx, newSessions)
}

// DeleteSessionsForEvent removes all sessions for an event
func (s *SessionService) DeleteSessionsForEvent(ctx context.Context, eventID, brandID string) error {
	// Verify event belongs to brand
	exists, err := s.eventRepo.ExistsByBrandAndID(ctx, brandID, eventID)
	if err != nil {
		return err
	}
	if !exists {
		return models.ErrEventNotFound
	}

	return s.sessionRepo.DeleteByEventID(ctx, eventID)
}

// GetSession retrieves a single session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID, brandID string) (*models.Session, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verify session belongs to brand
	if session.BrandID.Hex() != brandID {
		return nil, models.ErrUnauthorized
	}

	return session, nil
}

// UpdateSession updates a single session
func (s *SessionService) UpdateSession(ctx context.Context, sessionID, brandID string, sessionReq *SessionRequest) (*models.Session, error) {
	// Get existing session
	existingSession, err := s.GetSession(ctx, sessionID, brandID)
	if err != nil {
		return nil, err
	}

	// Validate event can be modified
	event, err := s.eventRepo.FindByID(ctx, existingSession.EventID.Hex())
	if err != nil {
		return nil, err
	}
	if err := s.validateEventModification(event); err != nil {
		return nil, err
	}

	// Convert request to model
	updatedSession, err := s.convertSessionRequestToModel(sessionReq, existingSession.EventID.Hex(), brandID)
	if err != nil {
		return nil, err
	}
	updatedSession.ID = existingSession.ID
	updatedSession.CreatedAt = existingSession.CreatedAt

	// Check for duplicates with other sessions for the same event
	if err := s.validateSessionUpdate(ctx, updatedSession); err != nil {
		return nil, err
	}

	return s.sessionRepo.Update(ctx, sessionID, updatedSession)
}

// DeleteSession removes a single session
func (s *SessionService) DeleteSession(ctx context.Context, sessionID, brandID string) error {
	// Get existing session
	session, err := s.GetSession(ctx, sessionID, brandID)
	if err != nil {
		return err
	}

	// Validate event can be modified
	event, err := s.eventRepo.FindByID(ctx, session.EventID.Hex())
	if err != nil {
		return err
	}
	if err := s.validateEventModification(event); err != nil {
		return err
	}

	// Check if this is the last session for the event
	count, err := s.sessionRepo.CountByEventID(ctx, session.EventID.Hex())
	if err != nil {
		return err
	}
	if count <= 1 {
		return models.NewBusinessError("LAST_SESSION", "cannot delete the last session of an event", models.ErrNoSessions)
	}

	return s.sessionRepo.Delete(ctx, sessionID)
}

// GetSessionsByBrand retrieves sessions for a brand with filtering
func (s *SessionService) GetSessionsByBrand(ctx context.Context, brandID string, filter *repository.SessionFilter) ([]*models.Session, error) {
	return s.sessionRepo.FindByBrandID(ctx, brandID, filter)
}

// ValidateSessionsForEvent validates sessions without creating them
func (s *SessionService) ValidateSessionsForEvent(ctx context.Context, eventID, brandID string, sessionReqs []*SessionRequest) error {
	// Validate event exists and belongs to brand
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.BrandID.Hex() != brandID {
		return models.ErrUnauthorized
	}

	// Convert and validate sessions
	sessions, err := s.convertSessionRequestsToModels(sessionReqs, eventID, brandID)
	if err != nil {
		return err
	}

	return models.ValidateSessions(sessions)
}

// Helper methods

func (s *SessionService) convertSessionRequestsToModels(sessionReqs []*SessionRequest, eventID, brandID string) ([]*models.Session, error) {
	sessions := make([]*models.Session, len(sessionReqs))
	for i, sessionReq := range sessionReqs {
		session, err := s.convertSessionRequestToModel(sessionReq, eventID, brandID)
		if err != nil {
			return nil, fmt.Errorf("invalid session at index %d: %w", i, err)
		}
		sessions[i] = session
	}
	return sessions, nil
}

func (s *SessionService) convertSessionRequestToModel(sessionReq *SessionRequest, eventID, brandID string) (*models.Session, error) {
	eventObjectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, models.NewValidationError("event_id", "invalid event_id")
	}

	brandObjectID, err := primitive.ObjectIDFromHex(brandID)
	if err != nil {
		return nil, models.NewValidationError("brand_id", "invalid brand_id")
	}

	startTime, err := time.Parse(time.RFC3339, sessionReq.StartTime)
	if err != nil {
		return nil, models.NewValidationError("start_time", "invalid start_time format, must be RFC3339")
	}

	endTime, err := time.Parse(time.RFC3339, sessionReq.EndTime)
	if err != nil {
		return nil, models.NewValidationError("end_time", "invalid end_time format, must be RFC3339")
	}

	if !startTime.Before(endTime) {
		return nil, models.NewValidationError("time", "start_time must be before end_time")
	}

	return &models.Session{
		ID:        primitive.NewObjectID(),
		EventID:   eventObjectID,
		BrandID:   brandObjectID,
		StartTime: startTime,
		EndTime:   endTime,
	}, nil
}

func (s *SessionService) validateEventModification(event *models.Event) error {
	// Published events cannot be modified
	if event.Status == models.StatusPublished {
		return models.NewBusinessError("PUBLISHED_IMMUTABLE", "published events cannot be modified", nil)
	}
	return nil
}

func (s *SessionService) validateSessionUpdate(ctx context.Context, updatedSession *models.Session) error {
	// Get all sessions for the event
	allSessions, err := s.sessionRepo.FindByEventID(ctx, updatedSession.EventID.Hex())
	if err != nil {
		return err
	}

	// Create unique key for the updated session
	updatedKey := fmt.Sprintf("%d-%d", updatedSession.StartTime.Unix(), updatedSession.EndTime.Unix())
	
	// Build hash set of existing sessions (excluding the one being updated) - O(n) complexity
	existingKeys := make(map[string]bool)
	for _, session := range allSessions {
		if session.ID != updatedSession.ID {
			sessionKey := fmt.Sprintf("%d-%d", session.StartTime.Unix(), session.EndTime.Unix())
			existingKeys[sessionKey] = true
		}
	}
	
	// Check if the updated session conflicts with existing ones - O(1) lookup
	if existingKeys[updatedKey] {
		return models.NewValidationError("sessions", "session with identical start and end times already exists")
	}

	return nil
}