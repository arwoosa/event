package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	vulpeslog "github.com/arwoosa/vulpes/log"

	"github.com/arwoosa/vulpes/relation"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// EventService implements the business logic for event management
type EventService struct {
	eventRepo      repository.EventRepository
	formRepo       repository.FormRepository
	sessionService *SessionService
	orderService   OrderServiceClient
}

// NewEventService creates a new event service
func NewEventService(
	eventRepo repository.EventRepository,
	formRepo repository.FormRepository,
	sessionService *SessionService,
	orderService OrderServiceClient,
) *EventService {
	return &EventService{
		eventRepo:      eventRepo,
		formRepo:       formRepo,
		sessionService: sessionService,
		orderService:   orderService,
	}
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	Title      string
	MerchantID string
	Summary    string
	// Status field removed - events are always created as draft
	Visibility    string
	CoverImageURL string
	Location      *LocationRequest
	Sessions      []*SessionRequest
	Detail        []DetailBlockRequest
	FAQ           []*FAQRequest
	UserID        string
}

// PatchEventRequest represents the request to partially update an event
type PatchEventRequest struct {
	ID            string
	Title         *string
	Summary       *string
	Visibility    *string
	CoverImageURL *string
	Location      *LocationRequest
	Sessions      []*SessionRequest
	Detail        []DetailBlockRequest
	FAQ           []*FAQRequest
	UserID        string
}

// LocationRequest represents location data in requests
type LocationRequest struct {
	Name        string
	Address     string
	PlaceID     string
	Coordinates *GeoJSONPointRequest
}

// GeoJSONPointRequest represents geospatial coordinates
type GeoJSONPointRequest struct {
	Type        string
	Coordinates [2]float64
}

// SessionRequest represents session data in requests
type SessionRequest struct {
	ID        string `json:"id,omitempty"` // Empty = create new, Non-empty = update existing
	Name      string `json:"name"`         // Session name (optional)
	Capacity  *int   `json:"capacity"`     // Capacity limit (optional, nil means unlimited)
	StartTime string `json:"start_time"`   // RFC3339 format
	EndTime   string `json:"end_time"`     // RFC3339 format
}

// DetailBlockRequest represents a single content block in requests
type DetailBlockRequest struct {
	Type string
	Data interface{}
}

// FAQRequest represents FAQ data in requests
type FAQRequest struct {
	Question string
	Answer   string
}

// OrderServiceClient interface for external order service
type OrderServiceClient interface {
	HasOrders(ctx context.Context, eventID string) (bool, error)
}

// CreateEvent creates a new event
func (s *EventService) CreateEvent(ctx context.Context, req *CreateEventRequest) (*models.Event, error) {
	// Convert request to model - will force draft status
	event, err := s.convertCreateRequestToModel(req)
	if err != nil {
		return nil, err
	}

	// Create event first
	createdEvent, err := s.eventRepo.Create(ctx, event)
	if err != nil {
		return nil, err
	}

	// Write Keto tuple
	if err := relation.AddUserResourceRole(ctx, req.UserID, "Event", createdEvent.ID.Hex(), relation.RoleOwner); err != nil {
		vulpeslog.Error("Failed to write Keto tuple for event ownership",
			vulpeslog.String("eventID", createdEvent.ID.Hex()),
			vulpeslog.String("userID", req.UserID),
			vulpeslog.Err(err))
		// Rollback event creation - without Keto tuple, the event cannot be operated on
		if deleteErr := s.eventRepo.Delete(ctx, createdEvent.ID); deleteErr != nil {
			vulpeslog.Error("Failed to rollback event creation after Keto failure",
				vulpeslog.String("eventID", createdEvent.ID.Hex()),
				vulpeslog.Err(deleteErr))
		}
		return nil, fmt.Errorf("failed to establish event ownership in authorization system: %w", err)
	}

	// Create sessions for the event if provided
	if len(req.Sessions) > 0 {
		_, err = s.sessionService.CreateSessionsForEvent(ctx, createdEvent.ID.Hex(), req.Sessions)
		if err != nil {
			// If session creation fails, rollback both event and Keto tuple
			vulpeslog.Error("Session creation failed, rolling back event creation",
				vulpeslog.String("eventID", createdEvent.ID.Hex()),
				vulpeslog.Err(err))

			// Delete the event from database
			if deleteErr := s.eventRepo.Delete(ctx, createdEvent.ID); deleteErr != nil {
				vulpeslog.Error("Failed to rollback event creation",
					vulpeslog.String("eventID", createdEvent.ID.Hex()),
					vulpeslog.Err(deleteErr))
			}

			// Delete the Keto tuple to avoid orphaned authorization data
			if err := relation.DeleteObjectId(ctx, "Event", createdEvent.ID.Hex()); err != nil {
				vulpeslog.Error("Failed to delete Keto tuple for event ownership",
					vulpeslog.String("eventID", createdEvent.ID.Hex()),
					vulpeslog.String("userID", req.UserID),
					vulpeslog.Err(err))
			}

			return nil, fmt.Errorf("failed to create sessions: %w", err)
		}
	}

	return createdEvent, nil
}

// GetEvent retrieves an event by ID
// Authorization is handled by API Gateway before reaching this service
func (s *EventService) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, errors.NewValidationError("event_id", "invalid event ID")
	}

	return s.eventRepo.FindByID(ctx, objectID)
}

// GetEventList retrieves a list of events with filtering
// Authorization/filtering is handled by API Gateway before reaching this service
func (s *EventService) GetEventList(ctx context.Context, filter *repository.EventFilter) (*repository.EventListResult, error) {
	return s.eventRepo.Find(ctx, filter)
}

// PatchEvent partially updates an event
func (s *EventService) PatchEvent(ctx context.Context, req *PatchEventRequest) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	if err := s.validateEventChanges(existingEvent, req); err != nil {
		return nil, err
	}

	// Validate detail size if provided
	if len(req.Detail) > 0 {
		detail := make([]models.DetailBlock, len(req.Detail))
		for i, blockReq := range req.Detail {
			detail[i] = models.DetailBlock{
				Type: blockReq.Type,
				Data: blockReq.Data,
			}
		}
		if err := validateDetailSize(detail); err != nil {
			return nil, err
		}
	}

	// Update sessions if provided
	if len(req.Sessions) > 0 {
		// Convert []Session to []*Session for compatibility
		existingSessionPtrs := make([]*models.Session, len(existingEvent.Sessions))
		for i := range existingEvent.Sessions {
			existingSessionPtrs[i] = &existingEvent.Sessions[i]
		}

		updatedSessions, err := s.sessionService.UpdateSessionsForEvent(ctx, req.ID, req.Sessions, existingEvent, existingSessionPtrs)
		if err != nil {
			return nil, err
		}

		// Apply updated sessions to existingEvent to ensure consistency
		existingEvent.Sessions = make([]models.Session, len(updatedSessions))
		for i, session := range updatedSessions {
			existingEvent.Sessions[i] = *session
		}
	}

	// Apply partial updates
	updatedEvent := s.applyPatchToEvent(existingEvent, req)

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(req.ID)
	if err != nil {
		return nil, errors.NewValidationError("event_id", "invalid event ID")
	}

	return s.eventRepo.Update(ctx, objectID, updatedEvent)
}

// DeleteEvent deletes an event
func (s *EventService) DeleteEvent(ctx context.Context, eventID, userID string) error {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return err
	}

	// Check if deletion is allowed
	if err := existingEvent.IsValidStatusForDelete(); err != nil {
		return err
	}

	// Delete Keto tuple
	if err := relation.DeleteObjectId(ctx, "Event", eventID); err != nil {
		vulpeslog.Error("Failed to delete Keto tuple for event ownership",
			vulpeslog.String("eventID", eventID),
			vulpeslog.String("userID", userID),
			vulpeslog.Err(err))
		return fmt.Errorf("failed to delete access control: %w", err)
	}

	// Delete sessions first
	if err := s.sessionService.DeleteSessionsForEvent(ctx, eventID); err != nil {
		return err
	}

	// Delete event
	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return errors.NewValidationError("event_id", "invalid event ID")
	}

	// Delete form if exists
	if err := s.formRepo.DeleteByEventID(ctx, objectID); err != nil && err != errors.ErrFormNotFound {
		return fmt.Errorf("failed to delete form: %w", err)
	}

	// Delete event
	if err := s.eventRepo.Delete(ctx, objectID); err != nil {
		return err
	}

	return nil
}

// UpdateEventStatus updates the status of an event
func (s *EventService) UpdateEventStatus(ctx context.Context, eventID, newStatus, userID string) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if err := s.validateStatusTransition(ctx, existingEvent, newStatus); err != nil {
		return nil, err
	}

	// Update status
	existingEvent.Status = newStatus
	existingEvent.UpdatedBy = userID
	existingEvent.UpdatedAt = time.Now()

	// Convert string ID to ObjectID
	objectID, err := primitive.ObjectIDFromHex(eventID)
	if err != nil {
		return nil, errors.NewValidationError("event_id", "invalid event ID")
	}

	return s.eventRepo.Update(ctx, objectID, existingEvent)
}

// Validation methods

// validateEventChanges validates field-level restrictions for published events
func (s *EventService) validateEventChanges(existing *models.Event, req *PatchEventRequest) error {
	// Archived events cannot be updated
	if existing.Status == models.StatusArchived {
		return errors.NewBusinessError("ARCHIVED_IMMUTABLE", "archived events cannot be updated", nil)
	}
	if existing.Status == models.StatusDraft {
		return nil // No restrictions for draft events
	}

	// For published events, only allow editing of specific safe fields:
	// - FAQ (additional Q&A)
	// - Visibility

	// Restricted fields for published events:
	// - Title
	// - Summary
	// - CoverImageURL
	// - Detail content
	// - Location
	// - Sessions
	// - Status transitions (handled by separate UpdateEventStatus method)

	restrictedFields := []string{}

	// Check for restricted field changes
	if req.Title != nil && *req.Title != existing.Title {
		restrictedFields = append(restrictedFields, "title")
	}
	if req.Summary != nil && *req.Summary != existing.Summary {
		restrictedFields = append(restrictedFields, "summary")
	}
	if req.CoverImageURL != nil && *req.CoverImageURL != existing.CoverImageURL {
		restrictedFields = append(restrictedFields, "cover_image_url")
	}
	if len(req.Detail) > 0 {
		// For the new blocks structure, any detail change is restricted for published events
		restrictedFields = append(restrictedFields, "detail")
	}
	if req.Location != nil {
		restrictedFields = append(restrictedFields, "location")
	}
	if len(req.Sessions) > 0 {
		for _, sessionReq := range req.Sessions {
			// If any session ID is provided, it indicates an update to existing sessions, which is restricted
			// If session ID is empty, it indicates a new session creation, which is allowed
			if sessionReq.ID != "" {
				restrictedFields = append(restrictedFields, "sessions")
				break
			}
		}
	}

	if len(restrictedFields) > 0 {
		return errors.NewBusinessError(
			errors.ErrorCodePublishedFieldRestricted,
			fmt.Sprintf("cannot modify restricted fields for published events: %v", restrictedFields),
			nil,
		)
	}

	// Allow changes to:
	// - FAQ (req.FAQ)
	// - Visibility (req.Visibility)
	return nil
}

func (s *EventService) validateStatusTransition(ctx context.Context, event *models.Event, newStatus string) error {
	if !models.IsValidStatus(newStatus) {
		return errors.NewValidationError("status", "invalid status")
	}

	if event.Status == newStatus {
		return nil // No change
	}

	if !event.CanTransitionTo(newStatus) {
		return errors.NewBusinessError(errors.ErrorCodeInvalidTransition,
			fmt.Sprintf("cannot transition from %s to %s", event.Status, newStatus), errors.ErrInvalidTransition)
	}

	// Special validations for transitions
	switch newStatus {
	case models.StatusPublished:
		// Validate all required fields for publishing
		if err := s.validatePublishRequirements(ctx, event); err != nil {
			return err
		}
	case models.StatusArchived:
		// Validate all required fields for publishing
		hasOrders, err := s.orderService.HasOrders(ctx, event.ID.Hex())
		if err != nil {
			return fmt.Errorf("failed to check orders: %w", err)
		}
		if hasOrders {
			return errors.NewBusinessError(errors.ErrorCodeHasOrders, "cannot change status of event with existing orders", errors.ErrHasOrders)
		}
	}

	return nil
}

func (s *EventService) validatePublishRequirements(ctx context.Context, event *models.Event) error {
	var missingFields []string

	if event.Title == "" {
		missingFields = append(missingFields, "title")
	}
	if event.CoverImageURL == "" {
		missingFields = append(missingFields, "cover_image_url")
	}
	if len(event.Detail) == 0 {
		missingFields = append(missingFields, "detail")
	}

	// Check actual session count from database instead of cached count
	sessionCount, err := s.sessionService.sessionRepo.CountByEventID(ctx, event.ID)
	if err != nil {
		return fmt.Errorf("failed to check session count: %w", err)
	}
	if sessionCount == 0 {
		missingFields = append(missingFields, "sessions")
	}

	if event.Location.Name == "" || event.Location.Address == "" {
		missingFields = append(missingFields, "location")
	}

	if len(missingFields) > 0 {
		return errors.NewBusinessError(
			errors.ErrorCodePublishRequirementsMissing,
			fmt.Sprintf("missing required fields for publishing: %v", missingFields),
			nil,
		)
	}

	return nil
}

// Conversion methods

func (s *EventService) convertCreateRequestToModel(req *CreateEventRequest) (*models.Event, error) {
	// Force draft status for all created events
	status := models.StatusDraft

	// Default visibility
	visibility := req.Visibility
	if visibility == "" {
		visibility = models.VisibilityPrivate
	}

	// Convert location (optional for drafts)
	var location models.Location
	if req.Location != nil {
		location = models.Location{
			Name:    req.Location.Name,
			Address: req.Location.Address,
			PlaceID: req.Location.PlaceID,
		}
		if req.Location.Coordinates != nil {
			location.Coordinates = models.GeoJSONPoint{
				Type:        models.GeoJSONTypePoint,
				Coordinates: req.Location.Coordinates.Coordinates,
			}
		} else {
			// For drafts without coordinates, set minimal valid GeoJSON
			location.Coordinates = models.GeoJSONPoint{
				Type:        models.GeoJSONTypePoint,
				Coordinates: [2]float64{0.0, 0.0},
			}
		}
	} else {
		// For drafts without location, set minimal valid GeoJSON to avoid MongoDB error
		location = models.Location{
			Coordinates: models.GeoJSONPoint{
				Type:        models.GeoJSONTypePoint,
				Coordinates: [2]float64{0.0, 0.0},
			},
		}
	}

	// Sessions are now handled by SessionService

	// Convert detail (optional for drafts)
	var detail []models.DetailBlock
	if len(req.Detail) > 0 {
		detail = make([]models.DetailBlock, len(req.Detail))
		for i, blockReq := range req.Detail {
			detail[i] = models.DetailBlock{
				Type: blockReq.Type,
				Data: blockReq.Data,
			}
		}

		// Validate detail size
		if err := validateDetailSize(detail); err != nil {
			return nil, err
		}
	}

	// Convert FAQ
	faq := make([]models.FAQ, len(req.FAQ))
	for i, faqReq := range req.FAQ {
		faq[i] = models.FAQ{
			Question: faqReq.Question,
			Answer:   faqReq.Answer,
		}
	}

	return &models.Event{
		Title:         req.Title,
		MerchantID:    req.MerchantID,
		Summary:       req.Summary,
		Status:        status,
		Visibility:    visibility,
		CoverImageURL: req.CoverImageURL,
		Location:      location,
		Detail:        detail,
		FAQ:           faq,
		CreatedBy:     req.UserID,
		UpdatedBy:     req.UserID,
	}, nil
}

func (s *EventService) applyPatchToEvent(existing *models.Event, req *PatchEventRequest) *models.Event {
	existing.UpdatedBy = req.UserID
	existing.UpdatedAt = time.Now()

	// Sessions are handled separately by SessionService
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Summary != nil {
		existing.Summary = *req.Summary
	}
	if req.Visibility != nil {
		existing.Visibility = *req.Visibility
	}
	if req.CoverImageURL != nil {
		existing.CoverImageURL = *req.CoverImageURL
	}

	if req.Location != nil {
		location := models.Location{
			Name:    req.Location.Name,
			Address: req.Location.Address,
			PlaceID: req.Location.PlaceID,
		}
		if req.Location.Coordinates != nil {
			location.Coordinates = models.GeoJSONPoint{
				Type:        models.GeoJSONTypePoint,
				Coordinates: req.Location.Coordinates.Coordinates,
			}
		}
		existing.Location = location
	}

	if len(req.Detail) > 0 {
		detail := make([]models.DetailBlock, len(req.Detail))
		for i, blockReq := range req.Detail {
			detail[i] = models.DetailBlock{
				Type: blockReq.Type,
				Data: blockReq.Data,
			}
		}
		existing.Detail = detail
	}

	if len(req.FAQ) > 0 {
		faq := make([]models.FAQ, len(req.FAQ))
		for i, faqReq := range req.FAQ {
			faq[i] = models.FAQ{
				Question: faqReq.Question,
				Answer:   faqReq.Answer,
			}
		}
		existing.FAQ = faq
	}

	return existing
}

// validateDetailSize validates that the detail blocks don't exceed the size limit
func validateDetailSize(detail []models.DetailBlock) error {
	// Serialize detail to calculate size
	data, err := json.Marshal(detail)
	if err != nil {
		return errors.NewValidationError("detail", "failed to serialize detail blocks")
	}

	const maxSize = 64 * 1024 // 64KB
	if len(data) > maxSize {
		return errors.NewValidationError("detail",
			fmt.Sprintf("detail size exceeds limit: %d bytes (max: %d bytes)", len(data), maxSize))
	}

	return nil
}

// === Form Management Business Logic ===

// SetEventFormRequest represents the request to create or update event form
type SetEventFormRequest struct {
	EventID  primitive.ObjectID
	Schema   interface{}
	UISchema interface{}
	UserID   string
}

// SetEventForm creates or updates a form for an event
func (s *EventService) SetEventForm(ctx context.Context, req *SetEventFormRequest) (*models.EventForm, error) {
	// Verify event exists
	event, err := s.eventRepo.FindByID(ctx, req.EventID)
	if err != nil {
		if err == errors.ErrEventNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to verify event: %w", err)
	}
	// check if event status allows form update
	if event.Status != models.StatusDraft {
		return nil, errors.NewBusinessError("FORM_UPDATE_IMMUTABLE", "only draft events can create or update forms", nil)
	}

	// Create form model
	form := &models.EventForm{
		EventID:  req.EventID,
		Schema:   req.Schema,
		UISchema: req.UISchema,
	}

	// Check if form already exists to determine create vs update
	existingForm, err := s.formRepo.FindByEventID(ctx, req.EventID)
	if err != nil && err != errors.ErrFormNotFound {
		return nil, fmt.Errorf("failed to check existing form: %w", err)
	}

	var result *models.EventForm
	if err == errors.ErrFormNotFound {
		// Form doesn't exist, create new form
		form.SetCreateInfo(req.UserID)
		result, err = s.formRepo.Create(ctx, form)
		if err != nil {
			return nil, fmt.Errorf("failed to create event form: %w", err)
		}
	} else {
		// Form exists, update existing form
		form.SetUpdateInfo(req.UserID)
		result, err = s.formRepo.Update(ctx, existingForm.ID, form)
		if err != nil {
			return nil, fmt.Errorf("failed to update event form: %w", err)
		}
	}

	vulpeslog.Info("Event form set successfully",
		vulpeslog.String("event_id", req.EventID.Hex()),
		vulpeslog.String("form_id", result.ID.Hex()),
		vulpeslog.String("user_id", req.UserID))

	return result, nil
}

// GetEventForm gets the form for an event
func (s *EventService) GetEventForm(ctx context.Context, eventID primitive.ObjectID) (*models.EventForm, error) {
	// Get form
	form, err := s.formRepo.FindByEventID(ctx, eventID)
	if err != nil {
		if err == errors.ErrFormNotFound {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get event form: %w", err)
	}

	return form, nil
}

// DeleteEventForm deletes the form for an event
func (s *EventService) DeleteEventForm(ctx context.Context, eventID primitive.ObjectID, userID string) error {
	// Verify event exists
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		if err == errors.ErrEventNotFound {
			return err
		}
		return fmt.Errorf("failed to verify event: %w", err)
	}

	// check if event status allows form deletion
	if err := event.IsValidStatusForDelete(); err != nil {
		return err
	}

	// Delete form
	err = s.formRepo.DeleteByEventID(ctx, eventID)
	if err != nil {
		if err == errors.ErrFormNotFound {
			return err
		}
		return fmt.Errorf("failed to delete event form: %w", err)
	}

	vulpeslog.Info("Event form deleted successfully",
		vulpeslog.String("event_id", eventID.Hex()),
		vulpeslog.String("user_id", userID))

	return nil
}
