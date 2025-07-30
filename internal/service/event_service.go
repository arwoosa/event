package service

import (
	"context"
	"fmt"
	"time"

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
	Detail        *DetailRequest
	FAQ           []*FAQRequest
	BrandID       string
	UserID        string
}

// UpdateEventRequest represents the request to update an event
type UpdateEventRequest struct {
	ID            string
	Title         string
	Summary       string
	Status        string
	Visibility    string
	CoverImageURL string
	Location      *LocationRequest
	Sessions      []*SessionRequest
	Detail        *DetailRequest
	FAQ           []*FAQRequest
	UserID        string
}

// PatchEventRequest represents the request to partially update an event
type PatchEventRequest struct {
	ID            string
	Title         *string
	Summary       *string
	Status        *string
	Visibility    *string
	CoverImageURL *string
	Location      *LocationRequest
	Sessions      []*SessionRequest
	Detail        *DetailRequest
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
	StartTime string // RFC3339 format
	EndTime   string // RFC3339 format
}

// DetailRequest represents detail content in requests
type DetailRequest struct {
	Content     string
	ContentType string
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
			s.eventRepo.Delete(ctx, createdEvent.ID.Hex())
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

// UpdateEvent fully updates an event
func (s *EventService) UpdateEvent(ctx context.Context, brandID string, req *UpdateEventRequest) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, brandID, req.ID)
	if err != nil {
		return nil, err
	}

	// Check if update is allowed based on status
	if err := s.validateUpdatePermissions(existingEvent, req.UserID); err != nil {
		return nil, err
	}

	// Validate request
	if err := s.validateUpdateRequest(req); err != nil {
		return nil, err
	}

	// Update sessions using SessionService
	_, err = s.sessionService.UpdateSessionsForEvent(ctx, req.ID, brandID, req.Sessions)
	if err != nil {
		return nil, fmt.Errorf("failed to update sessions: %w", err)
	}

	// Convert request to model (without sessions)
	event, err := s.convertUpdateRequestToModel(req, existingEvent)
	if err != nil {
		return nil, err
	}

	return s.eventRepo.Update(ctx, req.ID, event)
}

// PatchEvent partially updates an event
func (s *EventService) PatchEvent(ctx context.Context, brandID string, req *PatchEventRequest) (*models.Event, error) {
	// Get existing event
	existingEvent, err := s.GetEvent(ctx, brandID, req.ID)
	if err != nil {
		return nil, err
	}

	// Check if update is allowed based on status
	if err := s.validateUpdatePermissions(existingEvent, req.UserID); err != nil {
		return nil, err
	}

	// Update sessions if provided
	if len(req.Sessions) > 0 {
		_, err = s.sessionService.UpdateSessionsForEvent(ctx, req.ID, brandID, req.Sessions)
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
	if err := s.validateDeletePermissions(existingEvent); err != nil {
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

func (s *EventService) validateUpdateRequest(req *UpdateEventRequest) error {
	// All field validation (title, summary, status, visibility, etc.) is now handled by PGV
	// Only business logic validation remains here
	return nil
}

func (s *EventService) validateUpdatePermissions(event *models.Event, userID string) error {
	// If event is published, it cannot be modified
	if event.Status == models.StatusPublished {
		return models.NewBusinessError("PUBLISHED_IMMUTABLE", "published events cannot be modified", nil)
	}

	// If event is archived, check for orders
	if event.Status == models.StatusArchived {
		hasOrders, err := s.orderService.HasOrders(context.Background(), event.ID.Hex())
		if err != nil {
			return fmt.Errorf("failed to check orders: %w", err)
		}
		if hasOrders {
			return models.NewBusinessError("HAS_ORDERS", "cannot modify event with existing orders", models.ErrHasOrders)
		}
	}

	return nil
}

func (s *EventService) validateDeletePermissions(event *models.Event) error {
	// Published events cannot be deleted
	if event.Status == models.StatusPublished {
		return models.NewBusinessError("PUBLISHED_IMMUTABLE", "published events cannot be deleted", nil)
	}

	// If archived, check for orders
	if event.Status == models.StatusArchived {
		hasOrders, err := s.orderService.HasOrders(context.Background(), event.ID.Hex())
		if err != nil {
			return fmt.Errorf("failed to check orders: %w", err)
		}
		if hasOrders {
			return models.NewBusinessError("HAS_ORDERS", "cannot delete event with existing orders", models.ErrHasOrders)
		}
	}

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
	case models.StatusDraft:
		// Can only transition to draft from archived
		if event.Status == models.StatusArchived {
			hasOrders, err := s.orderService.HasOrders(ctx, event.ID.Hex())
			if err != nil {
				return fmt.Errorf("failed to check orders: %w", err)
			}
			if hasOrders {
				return models.NewBusinessError("HAS_ORDERS", "cannot change status of event with existing orders", models.ErrHasOrders)
			}
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
	if event.Detail.Content == "" {
		return models.NewValidationError("detail.content", "detail content is required for publishing")
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
	var detail models.Detail
	if req.Detail != nil {
		detail = models.Detail{
			Content:     req.Detail.Content,
			ContentType: req.Detail.ContentType,
		}
		if detail.ContentType == "" {
			detail.ContentType = models.ContentTypeHTML
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

func (s *EventService) convertUpdateRequestToModel(req *UpdateEventRequest, existing *models.Event) (*models.Event, error) {
	userID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		return nil, models.NewValidationError("user_id", "invalid user_id")
	}

	// Convert location
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

	// Sessions are now handled by SessionService

	// Convert detail
	detail := models.Detail{
		Content:     req.Detail.Content,
		ContentType: req.Detail.ContentType,
	}
	if detail.ContentType == "" {
		detail.ContentType = models.ContentTypeHTML
	}

	// Convert FAQ
	faq := make([]models.FAQ, len(req.FAQ))
	for i, faqReq := range req.FAQ {
		faq[i] = models.FAQ{
			Question: faqReq.Question,
			Answer:   faqReq.Answer,
		}
	}

	// Update the existing event
	existing.Title = req.Title
	existing.Summary = req.Summary
	existing.Status = req.Status
	existing.Visibility = req.Visibility
	existing.CoverImageURL = req.CoverImageURL
	existing.Location = location
	// Sessions are handled separately by SessionService
	existing.Detail = detail
	existing.FAQ = faq
	existing.UpdatedBy = userID
	existing.UpdatedAt = time.Now()

	return existing, nil
}

func (s *EventService) applyPatchToEvent(existing *models.Event, req *PatchEventRequest) *models.Event {
	userID, _ := primitive.ObjectIDFromHex(req.UserID)
	existing.UpdatedBy = userID
	existing.UpdatedAt = time.Now()

	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Summary != nil {
		existing.Summary = *req.Summary
	}
	if req.Status != nil {
		existing.Status = *req.Status
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

	if len(req.Sessions) > 0 {
		sessions := make([]models.Session, len(req.Sessions))
		for i, sessionReq := range req.Sessions {
			startTime, _ := time.Parse(time.RFC3339, sessionReq.StartTime)
			endTime, _ := time.Parse(time.RFC3339, sessionReq.EndTime)
			sessions[i] = models.Session{
				ID:        primitive.NewObjectID(),
				StartTime: startTime,
				EndTime:   endTime,
			}
		}
		// Sessions are handled separately by SessionService
	}

	if req.Detail != nil {
		detail := models.Detail{
			Content:     req.Detail.Content,
			ContentType: req.Detail.ContentType,
		}
		if detail.ContentType == "" {
			detail.ContentType = models.ContentTypeHTML
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
