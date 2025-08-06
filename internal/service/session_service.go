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
	sessionRepo  repository.SessionRepository
	eventRepo    repository.EventRepository
	orderService OrderServiceClient
}

// NewSessionService creates a new session service
func NewSessionService(
	sessionRepo repository.SessionRepository,
	eventRepo repository.EventRepository,
	orderService OrderServiceClient,
) *SessionService {
	return &SessionService{
		sessionRepo:  sessionRepo,
		eventRepo:    eventRepo,
		orderService: orderService,
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

// UpdateSessionsForEvent updates sessions for an event with smart diff-based approach
// Handles create, update, delete operations based on session IDs in the request
func (s *SessionService) UpdateSessionsForEvent(ctx context.Context, eventID, brandID string, sessionReqs []*SessionRequest) ([]*models.Session, error) {
	// Validate event exists and belongs to brand
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.BrandID.Hex() != brandID {
		return nil, models.ErrUnauthorized
	}

	// Check if event can be modified (basic event-level check)
	if err := s.validateEventModification(event); err != nil {
		return nil, err
	}

	// TODO: Add session-level order checking here
	// Currently we only check event-level orders, but we should:
	// 1. Check individual sessions for existing orders before allowing delete/time changes
	// 2. Allow adding new sessions even if some sessions have orders
	// 3. Allow updating session times only if no orders exist for that specific session
	// 4. Prevent deletion of sessions that have existing orders
	// Implementation: call orderService.HasOrdersForSession(ctx, sessionID) for each affected session

	// Get existing sessions
	existingSessions, err := s.sessionRepo.FindByEventID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing sessions: %w", err)
	}

	// Build existing sessions map for quick lookup
	existingSessionsMap := make(map[string]*models.Session)
	for _, session := range existingSessions {
		existingSessionsMap[session.ID.Hex()] = session
	}

	// Classify operations: create, update, delete
	var sessionsToCreate []*models.Session
	var sessionsToUpdate []*models.Session
	var sessionIDsToDelete []string

	// Build request sessions map to track which existing sessions are in the request
	requestedSessionIDs := make(map[string]bool)

	for _, sessionReq := range sessionReqs {
		if sessionReq.ID == "" {
			// Create new session
			newSession, err := s.convertSessionRequestToModel(sessionReq, eventID, brandID)
			if err != nil {
				return nil, fmt.Errorf("invalid new session: %w", err)
			}
			sessionsToCreate = append(sessionsToCreate, newSession)
		} else {
			// Update existing session
			requestedSessionIDs[sessionReq.ID] = true

			existingSession, exists := existingSessionsMap[sessionReq.ID]
			if !exists {
				return nil, models.NewValidationError("session_id", fmt.Sprintf("session with ID %s not found", sessionReq.ID))
			}

			updatedSession, err := s.convertSessionRequestToModel(sessionReq, eventID, brandID)
			if err != nil {
				return nil, fmt.Errorf("invalid update for session %s: %w", sessionReq.ID, err)
			}

			// Preserve original ID, created time
			updatedSession.ID = existingSession.ID
			updatedSession.CreatedAt = existingSession.CreatedAt

			sessionsToUpdate = append(sessionsToUpdate, updatedSession)
		}
	}

	// Find sessions to delete (existing sessions not in request)
	for sessionID := range existingSessionsMap {
		if !requestedSessionIDs[sessionID] {
			sessionIDsToDelete = append(sessionIDsToDelete, sessionID)
		}
	}

	// Validate final session collection for duplicates
	allFinalSessions := make([]*models.Session, 0, len(sessionsToCreate)+len(sessionsToUpdate))
	allFinalSessions = append(allFinalSessions, sessionsToCreate...)
	allFinalSessions = append(allFinalSessions, sessionsToUpdate...)

	if err := models.ValidateSessions(allFinalSessions); err != nil {
		return nil, models.NewValidationError("sessions", err.Error())
	}

	// Ensure at least one session remains
	if len(allFinalSessions) == 0 {
		return nil, models.NewBusinessError("NO_SESSIONS", "at least one session is required", models.ErrNoSessions)
	}

	// Execute all operations in a single bulk write
	if err := s.sessionRepo.BulkUpdateSessions(ctx, sessionsToCreate, sessionsToUpdate, sessionIDsToDelete); err != nil {
		return nil, fmt.Errorf("failed to bulk update sessions: %w", err)
	}

	// Return final session list
	return s.sessionRepo.FindByEventID(ctx, eventID)
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

// DeleteSessionById removes a session by session ID for a specific event
func (s *SessionService) DeleteSessionById(ctx context.Context, eventID, sessionID, brandID string) error {
	// Validate event exists and belongs to brand
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.BrandID.Hex() != brandID {
		return models.ErrUnauthorized
	}

	// Get the specific session
	session, err := s.sessionRepo.FindByID(ctx, sessionID)
	if err != nil {
		return err
	}

	// Verify session belongs to the specified event
	if session.EventID.Hex() != eventID {
		return models.NewBusinessError("SESSION_NOT_FOUND", "session does not belong to this event", models.ErrSessionNotFound)
	}

	// TODO: brandID will be remove in the future
	// Verify session belongs to the brand
	if session.BrandID.Hex() != brandID {
		return models.ErrUnauthorized
	}

	// Check if this is the last session for the event
	count, err := s.sessionRepo.CountByEventID(ctx, eventID)
	if err != nil {
		return err
	}

	// Only prevent deletion of last session for published events
	// Draft and archived events can have their last session deleted
	if event.Status == models.StatusPublished && count <= 1 {
		return models.NewBusinessError("LAST_SESSION", "cannot delete the last session of a published event", models.ErrNoSessions)
	}

	// Check if session has any existing orders
	if s.orderService != nil {
		hasOrders, err := s.orderService.HasOrdersForSession(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("failed to check orders for session: %w", err)
		}
		if hasOrders {
			return models.NewBusinessError("SESSION_HAS_ORDERS", "cannot delete session with existing orders", nil)
		}
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

	session := &models.Session{
		EventID:   eventObjectID,
		BrandID:   brandObjectID,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// Set ID if provided (for updates), otherwise generate new one (for creates)
	if sessionReq.ID != "" {
		sessionObjectID, err := primitive.ObjectIDFromHex(sessionReq.ID)
		if err != nil {
			return nil, models.NewValidationError("session_id", "invalid session_id")
		}
		session.ID = sessionObjectID
	} else {
		session.ID = primitive.NewObjectID()
	}

	return session, nil
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
