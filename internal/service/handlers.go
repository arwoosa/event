package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"event/internal/dao/repository"
	"event/internal/models"
)

// EventHandler handles all event-related gRPC operations
type EventHandler struct {
	eventService  *EventService
	publicService *PublicService
}

// NewEventHandler creates a new event handler
func NewEventHandler(eventService *EventService, publicService *PublicService) *EventHandler {
	return &EventHandler{
		eventService:  eventService,
		publicService: publicService,
	}
}

// Console API Handler Methods

// CreateEvent handles event creation
func (h *EventHandler) CreateEvent(ctx context.Context, req *CreateEventRequest) (*models.Event, error) {
	// Extract user and brand information from context
	userID, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.BrandID = brandID
	req.UserID = userID

	return h.eventService.CreateEvent(ctx, req)
}

// GetEventList handles event list retrieval
func (h *EventHandler) GetEventList(ctx context.Context, filter *repository.EventFilter) (*repository.EventListResult, error) {
	// Extract brand information from context
	_, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return h.eventService.GetEventList(ctx, brandID, filter)
}

// GetEvent handles single event retrieval
func (h *EventHandler) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	// Extract brand information from context
	_, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return h.eventService.GetEvent(ctx, brandID, eventID)
}

// UpdateEvent handles full event update
func (h *EventHandler) UpdateEvent(ctx context.Context, req *UpdateEventRequest) (*models.Event, error) {
	// Extract user and brand information from context
	userID, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.UserID = userID

	return h.eventService.UpdateEvent(ctx, brandID, req)
}

// PatchEvent handles partial event update
func (h *EventHandler) PatchEvent(ctx context.Context, req *PatchEventRequest) (*models.Event, error) {
	// Extract user and brand information from context
	userID, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	req.UserID = userID

	return h.eventService.PatchEvent(ctx, brandID, req)
}

// DeleteEvent handles event deletion
func (h *EventHandler) DeleteEvent(ctx context.Context, eventID string) error {
	// Extract user and brand information from context
	userID, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return err
	}

	return h.eventService.DeleteEvent(ctx, brandID, eventID, userID)
}

// UpdateEventStatus handles event status update
func (h *EventHandler) UpdateEventStatus(ctx context.Context, eventID, newStatus string) (*models.Event, error) {
	// Extract user and brand information from context
	userID, brandID, err := h.extractUserAndBrandFromContext(ctx)
	if err != nil {
		return nil, err
	}

	return h.eventService.UpdateEventStatus(ctx, brandID, eventID, newStatus, userID)
}

// Public API Handler Methods

// SearchPublicEvents handles public event search
func (h *EventHandler) SearchPublicEvents(ctx context.Context, req *SearchEventsRequest) (*repository.EventListResult, error) {
	return h.publicService.SearchEvents(ctx, req)
}

// GetPublicEvent handles public event retrieval (for sharing)
func (h *EventHandler) GetPublicEvent(ctx context.Context, eventID string) (*models.Event, error) {
	return h.publicService.GetEvent(ctx, eventID)
}

// Helper methods

func (h *EventHandler) extractUserAndBrandFromContext(ctx context.Context) (userID, brandID string, err error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	// Extract user ID
	userIDValues := md.Get("user-id")
	if len(userIDValues) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing user-id header")
	}
	userID = userIDValues[0]

	// Extract brand ID
	brandIDValues := md.Get("brand-id")
	if len(brandIDValues) == 0 {
		return "", "", status.Error(codes.Unauthenticated, "missing brand-id header")
	}
	brandID = brandIDValues[0]

	return userID, brandID, nil
}

// HandleError converts service errors to appropriate gRPC errors
func (h *EventHandler) HandleError(err error) error {
	switch e := err.(type) {
	case *models.ValidationError:
		return status.Error(codes.InvalidArgument, e.Error())
	case *models.BusinessError:
		switch e.Code {
		case "PUBLISHED_IMMUTABLE":
			return status.Error(codes.FailedPrecondition, e.Error())
		case "HAS_ORDERS":
			return status.Error(codes.FailedPrecondition, e.Error())
		case "INVALID_TRANSITION":
			return status.Error(codes.FailedPrecondition, e.Error())
		default:
			return status.Error(codes.InvalidArgument, e.Error())
		}
	default:
		if err == models.ErrEventNotFound {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, fmt.Sprintf("internal error: %v", err))
	}
}

// Utility functions

// ParseTimeString parses RFC3339 time string
func ParseTimeString(timeStr string) (time.Time, error) {
	return time.Parse(time.RFC3339, timeStr)
}

// FormatTimeString formats time to RFC3339 string
func FormatTimeString(t time.Time) string {
	return t.Format(time.RFC3339)
}
