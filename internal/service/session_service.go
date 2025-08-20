package service

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
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
func (s *SessionService) CreateSessionsForEvent(ctx context.Context, eventID, merchantID string, sessionReqs []*SessionRequest) ([]*models.Session, error) {
	// Validate event exists and belongs to merchant
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.MerchantID.Hex() != merchantID {
		return nil, errors.ErrUnauthorized
	}

	// Convert session requests to models
	sessions, err := s.convertSessionRequestsToModels(sessionReqs, eventID)
	if err != nil {
		return nil, err
	}

	// Validate sessions for duplicates
	if err := models.ValidateSessions(sessions); err != nil {
		return nil, errors.NewValidationError("sessions", err.Error())
	}

	// Create sessions in batch
	return s.sessionRepo.CreateBatch(ctx, sessions)
}

// GetSessionsForEvent retrieves all sessions for an event
func (s *SessionService) GetSessionsForEvent(ctx context.Context, eventID, merchantID string) ([]*models.Session, error) {
	// Verify event belongs to merchant
	exists, err := s.eventRepo.ExistsByMerchantAndID(ctx, merchantID, eventID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.ErrEventNotFound
	}

	return s.sessionRepo.FindByEventID(ctx, eventID)
}

// GetSessionsForEvents retrieves sessions for multiple events (batch operation)
func (s *SessionService) GetSessionsForEvents(ctx context.Context, eventIDs []string) (map[string][]*models.Session, error) {
	return s.sessionRepo.FindByEventIDs(ctx, eventIDs)
}

// UpdateSessionsForEvent updates sessions for an event with smart diff-based approach
// Handles create, update operations based on session IDs in the request
// existingEvent and existingSessions are optional - if provided, skips database queries for better performance
func (s *SessionService) UpdateSessionsForEvent(ctx context.Context, eventID, merchantID string, sessionReqs []*SessionRequest, existingEvent *models.Event, existingSessions []*models.Session) ([]*models.Session, error) {
	// Use provided data or fetch from database
	var event *models.Event
	var sessions []*models.Session
	var err error

	if existingEvent != nil {
		event = existingEvent
	} else {
		// Validate event exists and belongs to merchant
		event, err = s.eventRepo.FindByID(ctx, eventID)
		if err != nil {
			return nil, err
		}
	}

	// Validate merchant ownership
	if event.MerchantID.Hex() != merchantID {
		return nil, errors.ErrUnauthorized
	}

	if existingSessions != nil {
		sessions = existingSessions
	} else {
		// Get existing sessions from database
		sessions, err = s.sessionRepo.FindByEventID(ctx, eventID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch existing sessions: %w", err)
		}
	}

	// Build existing sessions map for quick lookup
	existingSessionsMap := make(map[string]*models.Session)
	for _, session := range sessions {
		existingSessionsMap[session.ID.Hex()] = session
	}

	// Process requests and build final session list in one pass
	var sessionsToCreate []*models.Session
	var sessionsToUpdate []*models.Session
	updatedSessionIDs := make(map[string]bool)

	for _, sessionReq := range sessionReqs {
		if sessionReq.ID == "" {
			// Create new session
			newSession, err := s.convertSessionRequestToModel(sessionReq, eventID)
			if err != nil {
				return nil, fmt.Errorf("invalid new session: %w", err)
			}
			sessionsToCreate = append(sessionsToCreate, newSession)
		} else {
			// Update existing session
			updatedSessionIDs[sessionReq.ID] = true
			existingSession, exists := existingSessionsMap[sessionReq.ID]
			if !exists {
				return nil, errors.NewValidationError("session_id", fmt.Sprintf("session with ID %s not found", sessionReq.ID))
			}

			updatedSession, err := s.convertSessionRequestToModel(sessionReq, eventID)
			if err != nil {
				return nil, fmt.Errorf("invalid update for session %s: %w", sessionReq.ID, err)
			}

			// Preserve original ID, created time
			updatedSession.ID = existingSession.ID
			updatedSession.CreatedAt = existingSession.CreatedAt

			sessionsToUpdate = append(sessionsToUpdate, updatedSession)
		}
	}

	// Build complete final session list: existing unchanged + new + updated
	allFinalSessions := make([]*models.Session, 0, len(sessions)+len(sessionsToCreate))

	// Add existing unchanged sessions and new/updated sessions
	for _, existing := range sessions {
		if !updatedSessionIDs[existing.ID.Hex()] {
			// append unchanged existing session
			allFinalSessions = append(allFinalSessions, existing)
		}
	}
	allFinalSessions = append(allFinalSessions, sessionsToCreate...)
	allFinalSessions = append(allFinalSessions, sessionsToUpdate...)

	// Validate complete final session collection for duplicates
	if err := models.ValidateSessions(allFinalSessions); err != nil {
		return nil, errors.NewValidationError("sessions", err.Error())
	}

	// Execute all operations in a single bulk write
	if err := s.sessionRepo.BulkUpdateSessions(ctx, sessionsToCreate, sessionsToUpdate, nil); err != nil {
		return nil, fmt.Errorf("failed to bulk update sessions: %w", err)
	}

	// Return final session list
	return s.sessionRepo.FindByEventID(ctx, eventID)
}

// DeleteSessionsForEvent removes all sessions for an event
func (s *SessionService) DeleteSessionsForEvent(ctx context.Context, eventID, merchantID string) error {
	// Verify event belongs to merchant
	exists, err := s.eventRepo.ExistsByMerchantAndID(ctx, merchantID, eventID)
	if err != nil {
		return err
	}
	if !exists {
		return errors.ErrEventNotFound
	}

	return s.sessionRepo.DeleteByEventID(ctx, eventID)
}

// GetSession retrieves a single session by ID
func (s *SessionService) GetSession(ctx context.Context, sessionID, merchantID string) (*models.Session, error) {
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Verify session belongs to merchant through parent event
	event, err := s.eventRepo.FindByID(ctx, session.EventID.Hex())
	if err != nil {
		return nil, err
	}
	if event.MerchantID.Hex() != merchantID {
		return nil, errors.ErrUnauthorized
	}

	return session, nil
}

// DeleteSessionById removes a session by session ID for a specific event
func (s *SessionService) DeleteSessionById(ctx context.Context, eventID, sessionID, merchantID string) error {
	// Validate event exists and belongs to merchant
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.MerchantID.Hex() != merchantID {
		return errors.ErrUnauthorized
	}

	// Get the specific session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}

	// Verify session belongs to the specified event
	if session.EventID.Hex() != eventID {
		return errors.NewBusinessError(errors.ErrorCodeSessionNotFound, "session does not belong to this event", errors.ErrSessionNotFound)
	}

	if event.Status != models.StatusDraft {
		return errors.NewBusinessError(errors.ErrorCodePublishedImmutable, "cannot delete sessions for published or archived events", nil)
	}

	return s.sessionRepo.Delete(ctx, sessionID)
}

// GetSessionsByMerchant retrieves sessions for a merchant with filtering
func (s *SessionService) GetSessionsByMerchant(ctx context.Context, merchantID string, filter *repository.SessionFilter) ([]*models.Session, error) {
	return s.sessionRepo.FindByMerchantID(ctx, merchantID, filter)
}

// ValidateSessionsForEvent validates sessions without creating them
func (s *SessionService) ValidateSessionsForEvent(ctx context.Context, eventID, merchantID string, sessionReqs []*SessionRequest) error {
	// Validate event exists and belongs to merchant
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.MerchantID.Hex() != merchantID {
		return errors.ErrUnauthorized
	}

	// Convert and validate sessions
	sessions, err := s.convertSessionRequestsToModels(sessionReqs, eventID)
	if err != nil {
		return err
	}

	return models.ValidateSessions(sessions)
}

// Helper methods

func (s *SessionService) convertSessionRequestsToModels(sessionReqs []*SessionRequest, eventID string) ([]*models.Session, error) {
	sessions := make([]*models.Session, len(sessionReqs))
	for i, sessionReq := range sessionReqs {
		session, err := s.convertSessionRequestToModel(sessionReq, eventID)
		if err != nil {
			return nil, fmt.Errorf("invalid session at index %d: %w", i, err)
		}
		sessions[i] = session
	}
	return sessions, nil
}

func (s *SessionService) convertSessionRequestToModel(sessionReq *SessionRequest, eventID string) (*models.Session, error) {
	eventObjectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, errors.NewValidationError("event_id", "invalid event_id")
	}

	startTime, err := time.Parse(time.RFC3339, sessionReq.StartTime)
	if err != nil {
		return nil, errors.NewValidationError("start_time", "invalid start_time format, must be RFC3339")
	}

	endTime, err := time.Parse(time.RFC3339, sessionReq.EndTime)
	if err != nil {
		return nil, errors.NewValidationError("end_time", "invalid end_time format, must be RFC3339")
	}

	if !startTime.Before(endTime) {
		return nil, errors.NewValidationError("time", "start_time must be before end_time")
	}

	session := &models.Session{
		EventID:   eventObjectID,
		Name:      sessionReq.Name,
		Capacity:  sessionReq.Capacity,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Set ID if provided (for updates), otherwise generate new one (for creates)
	if sessionReq.ID != "" {
		sessionObjectID, err := primitive.ObjectIDFromHex(sessionReq.ID)
		if err != nil {
			return nil, errors.NewValidationError("session_id", "invalid session_id")
		}
		session.ID = sessionObjectID
	} else {
		session.ID = primitive.NewObjectID()
	}

	return session, nil
}
