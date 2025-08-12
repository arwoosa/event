package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	vulpeslog "github.com/arwoosa/vulpes/log"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/dao/repository"
	"event/internal/models"
)

// EventService implements the business logic for event management
type EventService struct {
	eventRepo      repository.EventRepository
	sessionService *SessionService
	orderService   OrderServiceClient
}

// NewEventService creates a new event service
func NewEventService(
	eventRepo repository.EventRepository,
	sessionService *SessionService,
	orderService OrderServiceClient,
) *EventService {
	return &EventService{
		eventRepo:      eventRepo,
		sessionService: sessionService,
		orderService:   orderService,
	}
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	Title   string
	Summary string
	// Status field removed - events are always created as draft
	Visibility    string
	CoverImageURL string
	Location      *LocationRequest
	Sessions      []*SessionRequest
	Detail        []DetailBlockRequest
	FAQ           []*FAQRequest
	BrandID       string
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
	HasOrdersForSession(ctx context.Context, sessionID string) (bool, error)
}

// CreateEvent creates a new event
func (s *EventService) CreateEvent(ctx context.Context, req *CreateEventRequest) (*models.Event, error) {
	// Validate draft requirements (minimal validation)
	if err := s.validateDraftRequest(req); err != nil {
		return nil, err
	}

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

	// Create sessions for the event if provided
	if len(req.Sessions) > 0 {
		_, err = s.sessionService.CreateSessionsForEvent(ctx, createdEvent.ID.Hex(), req.BrandID, req.Sessions)
		if err != nil {
			// If session creation fails, we should delete the event to maintain consistency
			if deleteErr := s.eventRepo.Delete(ctx, createdEvent.ID.Hex()); deleteErr != nil {
				// Log the rollback error but return the original session creation error
				vulpeslog.Error("Failed to rollback event creation",
					vulpeslog.String("eventID", createdEvent.ID.Hex()),
					vulpeslog.Err(deleteErr))
			}
			return nil, fmt.Errorf("failed to create sessions: %w", err)
		}
	}

	return createdEvent, nil
}

// GetEvent retrieves an event by ID for the specified brand
func (s *EventService) GetEvent(ctx context.Context, brandID, eventID string) (*models.Event, error) {
	// Check if event exists for this brand
	exists, err := s.eventRepo.ExistsByBrandAndID(ctx, brandID, eventID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.ErrEventNotFound
	}

	return s.eventRepo.FindByID(ctx, eventID)
}

// GetEventList retrieves a list of events for the specified brand with filtering
func (s *EventService) GetEventList(ctx context.Context, brandID string, filter *repository.EventFilter) (*repository.EventListResult, error) {
	// Ensure brand ID is set in filter
	filter.BrandID = &brandID

	return s.eventRepo.FindByBrandID(ctx, brandID, filter)
}

// PatchEvent partially updates an event
func (s *EventService) PatchEvent(ctx context.Context, brandID string, req *PatchEventRequest) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, brandID, req.ID)
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

		_, err = s.sessionService.UpdateSessionsForEvent(ctx, req.ID, brandID, req.Sessions, existingEvent, existingSessionPtrs)
		if err != nil {
			return nil, fmt.Errorf("failed to update sessions: %w", err)
		}
	}

	// Apply partial updates
	updatedEvent := s.applyPatchToEvent(existingEvent, req)

	return s.eventRepo.Update(ctx, req.ID, updatedEvent)
}

// DeleteEvent deletes an event
func (s *EventService) DeleteEvent(ctx context.Context, brandID, eventID, userID string) error {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, brandID, eventID)
	if err != nil {
		return err
	}

	// Check if deletion is allowed
	if err := existingEvent.IsValidStatusForDelete(); err != nil {
		return err
	}

	// Delete sessions first
	if err := s.sessionService.DeleteSessionsForEvent(ctx, eventID, brandID); err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	return s.eventRepo.Delete(ctx, eventID)
}

// UpdateEventStatus updates the status of an event
func (s *EventService) UpdateEventStatus(ctx context.Context, brandID, eventID, newStatus, userID string) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, brandID, eventID)
	if err != nil {
		return nil, err
	}

	// Validate status transition
	if err := s.validateStatusTransition(ctx, existingEvent, newStatus); err != nil {
		return nil, err
	}

	// Update status
	existingEvent.Status = newStatus
	existingEvent.UpdatedBy, _ = primitive.ObjectIDFromHex(userID)
	existingEvent.UpdatedAt = time.Now()

	return s.eventRepo.Update(ctx, eventID, existingEvent)
}

// Validation methods

// validateDraftRequest validates business logic requirements for draft event creation
// Basic field validation is now handled by proto-gen-validate
func (s *EventService) validateDraftRequest(req *CreateEventRequest) error {
	// All basic field validation (title, summary, visibility, etc.) is now handled by PGV
	// Only business logic validation remains here
	return nil
}

// validateEventChanges validates field-level restrictions for published events
func (s *EventService) validateEventChanges(existing *models.Event, req *PatchEventRequest) error {
	// Archived events cannot be updated
	if err := existing.IsValidStatusForUpdate(); err != nil {
		return err
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
		return models.NewBusinessError(
			"PUBLISHED_FIELD_RESTRICTED",
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
		return models.NewValidationError("status", "invalid status")
	}

	if event.Status == newStatus {
		return nil // No change
	}

	if !event.CanTransitionTo(newStatus) {
		return models.NewBusinessError("INVALID_TRANSITION",
			fmt.Sprintf("cannot transition from %s to %s", event.Status, newStatus), models.ErrInvalidTransition)
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
		// TODO: change method to CanArchived
		hasOrders, err := s.orderService.HasOrders(ctx, event.ID.Hex())
		if err != nil {
			return fmt.Errorf("failed to check orders: %w", err)
		}
		if hasOrders {
			return models.NewBusinessError("HAS_ORDERS", "cannot change status of event with existing orders", models.ErrHasOrders)
		}
	}

	return nil
}

func (s *EventService) validatePublishRequirements(ctx context.Context, event *models.Event) error {
	if event.Title == "" {
		return models.NewValidationError("title", "title is required for publishing")
	}
	if event.CoverImageURL == "" {
		return models.NewValidationError("cover_image_url", "cover image is required for publishing")
	}
	if len(event.Detail) == 0 {
		return models.NewValidationError("detail", "detail blocks are required for publishing")
	}

	// Check actual session count from database instead of cached count
	sessionCount, err := s.sessionService.sessionRepo.CountByEventID(ctx, event.ID.Hex())
	if err != nil {
		return fmt.Errorf("failed to check session count: %w", err)
	}
	if sessionCount == 0 {
		return models.NewValidationError("sessions", "at least one session is required for publishing")
	}

	if event.Location.Name == "" || event.Location.Address == "" {
		return models.NewValidationError("location", "complete location information is required for publishing")
	}
	return nil
}

// Conversion methods

func (s *EventService) convertCreateRequestToModel(req *CreateEventRequest) (*models.Event, error) {
	brandID, err := primitive.ObjectIDFromHex(req.BrandID)
	if err != nil {
		return nil, models.NewValidationError("brand_id", "invalid brand_id")
	}

	userID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		return nil, models.NewValidationError("user_id", "invalid user_id")
	}

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
		BrandID:       brandID,
		Summary:       req.Summary,
		Status:        status,
		Visibility:    visibility,
		CoverImageURL: req.CoverImageURL,
		Location:      location,
		Detail:        detail,
		FAQ:           faq,
		CreatedBy:     userID,
		UpdatedBy:     userID,
	}, nil
}

func (s *EventService) applyPatchToEvent(existing *models.Event, req *PatchEventRequest) *models.Event {
	userID, _ := primitive.ObjectIDFromHex(req.UserID)
	existing.UpdatedBy = userID
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
		return models.NewValidationError("detail", "failed to serialize detail blocks")
	}
	
	const maxSize = 64 * 1024 // 64KB
	if len(data) > maxSize {
		return models.NewValidationError("detail", 
			fmt.Sprintf("detail size exceeds limit: %d bytes (max: %d bytes)", len(data), maxSize))
	}
	
	return nil
}
