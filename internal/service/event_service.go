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
	eventRepo    repository.EventRepository
	orderService OrderServiceClient
}

// NewEventService creates a new event service
func NewEventService(eventRepo repository.EventRepository, orderService OrderServiceClient) *EventService {
	return &EventService{
		eventRepo:    eventRepo,
		orderService: orderService,
	}
}

// CreateEventRequest represents the request to create an event
type CreateEventRequest struct {
	Title         string
	Summary       string
	Status        string
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
	// Validate request
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// Convert request to model
	event, err := s.convertCreateRequestToModel(req)
	if err != nil {
		return nil, err
	}

	// Validate business rules
	if err := s.validateEventBusinessRules(event); err != nil {
		return nil, err
	}

	// Create event
	return s.eventRepo.Create(ctx, event)
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

	// Convert request to model
	event, err := s.convertUpdateRequestToModel(req, existingEvent)
	if err != nil {
		return nil, err
	}

	// Validate business rules
	if err := s.validateEventBusinessRules(event); err != nil {
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

	// Apply partial updates
	updatedEvent := s.applyPatchToEvent(existingEvent, req)

	// Validate business rules
	if err := s.validateEventBusinessRules(updatedEvent); err != nil {
		return nil, err
	}

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

func (s *EventService) validateCreateRequest(req *CreateEventRequest) error {
	if req.Title == "" {
		return models.NewValidationError("title", "title is required")
	}
	if len(req.Title) > 60 {
		return models.NewValidationError("title", "title must be 60 characters or less")
	}
	if len(req.Summary) > 160 {
		return models.NewValidationError("summary", "summary must be 160 characters or less")
	}
	if req.CoverImageURL == "" {
		return models.NewValidationError("cover_image_url", "cover image URL is required")
	}
	if req.Location == nil {
		return models.NewValidationError("location", "location is required")
	}
	if len(req.Sessions) == 0 {
		return models.NewValidationError("sessions", "at least one session is required")
	}
	if req.Detail == nil || req.Detail.Content == "" {
		return models.NewValidationError("detail.content", "detail content is required")
	}
	if len(req.Detail.Content) > 65536 { // 64KB
		return models.NewValidationError("detail.content", "detail content must be 64KB or less")
	}

	// Validate status and visibility
	if req.Status != "" && !models.IsValidStatus(req.Status) {
		return models.NewValidationError("status", "invalid status")
	}
	if req.Visibility != "" && !models.IsValidVisibility(req.Visibility) {
		return models.NewValidationError("visibility", "invalid visibility")
	}

	// Validate location
	if err := s.validateLocation(req.Location); err != nil {
		return err
	}

	// Validate sessions
	if err := s.validateSessions(req.Sessions); err != nil {
		return err
	}

	// Validate FAQ
	if err := s.validateFAQ(req.FAQ); err != nil {
		return err
	}

	return nil
}

func (s *EventService) validateUpdateRequest(req *UpdateEventRequest) error {
	return s.validateCreateRequest(&CreateEventRequest{
		Title:         req.Title,
		Summary:       req.Summary,
		Status:        req.Status,
		Visibility:    req.Visibility,
		CoverImageURL: req.CoverImageURL,
		Location:      req.Location,
		Sessions:      req.Sessions,
		Detail:        req.Detail,
		FAQ:           req.FAQ,
	})
}

func (s *EventService) validateLocation(loc *LocationRequest) error {
	if loc.Name == "" {
		return models.NewValidationError("location.name", "location name is required")
	}
	if loc.Address == "" {
		return models.NewValidationError("location.address", "location address is required")
	}
	if loc.PlaceID == "" {
		return models.NewValidationError("location.place_id", "location place_id is required")
	}
	if loc.Coordinates != nil {
		if loc.Coordinates.Type != "Point" {
			return models.NewValidationError("location.coordinates.type", "coordinates type must be 'Point'")
		}
		lng, lat := loc.Coordinates.Coordinates[0], loc.Coordinates.Coordinates[1]
		if lng < -180 || lng > 180 {
			return models.NewValidationError("location.coordinates", "longitude must be between -180 and 180")
		}
		if lat < -90 || lat > 90 {
			return models.NewValidationError("location.coordinates", "latitude must be between -90 and 90")
		}
	}
	return nil
}

func (s *EventService) validateSessions(sessions []*SessionRequest) error {
	if len(sessions) == 0 {
		return models.NewValidationError("sessions", "at least one session is required")
	}

	parsedSessions := make([]models.Session, len(sessions))
	for i, session := range sessions {
		startTime, err := time.Parse(time.RFC3339, session.StartTime)
		if err != nil {
			return models.NewValidationErrorWithIndex("sessions", "invalid start_time format, must be RFC3339", i)
		}
		endTime, err := time.Parse(time.RFC3339, session.EndTime)
		if err != nil {
			return models.NewValidationErrorWithIndex("sessions", "invalid end_time format, must be RFC3339", i)
		}
		if !startTime.Before(endTime) {
			return models.NewValidationErrorWithIndex("sessions", "start_time must be before end_time", i)
		}
		parsedSessions[i] = models.Session{
			StartTime: startTime,
			EndTime:   endTime,
		}
	}

	// Check for overlaps
	for i := 0; i < len(parsedSessions); i++ {
		for j := i + 1; j < len(parsedSessions); j++ {
			if parsedSessions[i].OverlapsWith(parsedSessions[j]) {
				return models.NewValidationError("sessions", fmt.Sprintf("sessions %d and %d overlap", i, j))
			}
		}
	}

	return nil
}

func (s *EventService) validateFAQ(faqs []*FAQRequest) error {
	for i, faq := range faqs {
		if faq.Question == "" {
			return models.NewValidationErrorWithIndex("faq", "question is required", i)
		}
		if faq.Answer == "" {
			return models.NewValidationErrorWithIndex("faq", "answer is required", i)
		}
		if len(faq.Question) > 100 {
			return models.NewValidationErrorWithIndex("faq", "question must be 100 characters or less", i)
		}
		if len(faq.Answer) > 300 {
			return models.NewValidationErrorWithIndex("faq", "answer must be 300 characters or less", i)
		}
	}
	return nil
}

func (s *EventService) validateEventBusinessRules(event *models.Event) error {
	return event.ValidateSessions()
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
		if err := s.validatePublishRequirements(event); err != nil {
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

func (s *EventService) validatePublishRequirements(event *models.Event) error {
	if event.Title == "" {
		return models.NewValidationError("title", "title is required for publishing")
	}
	if event.CoverImageURL == "" {
		return models.NewValidationError("cover_image_url", "cover image is required for publishing")
	}
	if event.Detail.Content == "" {
		return models.NewValidationError("detail.content", "detail content is required for publishing")
	}
	if !event.HasSessions() {
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

	// Default values
	status := req.Status
	if status == "" {
		status = models.StatusDraft
	}
	visibility := req.Visibility
	if visibility == "" {
		visibility = models.VisibilityPrivate
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

	// Convert sessions
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

	return &models.Event{
		Title:         req.Title,
		BrandID:       brandID,
		Summary:       req.Summary,
		Status:        status,
		Visibility:    visibility,
		CoverImageURL: req.CoverImageURL,
		Location:      location,
		Sessions:      sessions,
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

	// Convert sessions
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
	existing.Sessions = sessions
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
		existing.Sessions = sessions
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
