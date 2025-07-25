package service

import (
	"context"
	"time"

	"event/internal/dao/repository"
	"event/internal/models"
)

// PublicService implements the business logic for public event access
type PublicService struct {
	eventRepo repository.EventRepository
}

// NewPublicService creates a new public service
func NewPublicService(eventRepo repository.EventRepository) *PublicService {
	return &PublicService{
		eventRepo: eventRepo,
	}
}

// SearchEventsRequest represents the request to search public events
type SearchEventsRequest struct {
	BrandID              *string
	TitleSearch          *string
	SessionStartTimeFrom *string
	SessionStartTimeTo   *string
	LocationLat          *float64
	LocationLng          *float64
	LocationRadius       *int
	SortBy               *string
	SortOrder            *string
	PageToken            *string
	Page                 *int32
	PageSize             *int32
}

// SearchEvents searches for public events
func (s *PublicService) SearchEvents(ctx context.Context, req *SearchEventsRequest) (*repository.EventListResult, error) {
	filter := &repository.PublicEventFilter{
		BrandID:        req.BrandID,
		TitleSearch:    req.TitleSearch,
		LocationLat:    req.LocationLat,
		LocationLng:    req.LocationLng,
		LocationRadius: req.LocationRadius,
		SortBy:         req.SortBy,
		SortOrder:      req.SortOrder,
		PageToken:      req.PageToken,
		Limit:          20, // Default
		Offset:         0,
	}

	// Handle time filters
	if req.SessionStartTimeFrom != nil {
		if t, err := time.Parse(time.RFC3339, *req.SessionStartTimeFrom); err == nil {
			filter.SessionStartTimeFrom = &t
		}
	}
	if req.SessionStartTimeTo != nil {
		if t, err := time.Parse(time.RFC3339, *req.SessionStartTimeTo); err == nil {
			filter.SessionStartTimeTo = &t
		}
	}

	// Handle pagination
	if req.PageSize != nil {
		if *req.PageSize > 0 && *req.PageSize <= 100 {
			filter.Limit = int(*req.PageSize)
		}
	}
	if req.Page != nil && *req.Page > 0 {
		filter.Offset = int((*req.Page - 1) * int32(filter.Limit))
		filter.PageToken = nil // Don't use cursor pagination if page is specified
	}

	// Handle geospatial search
	if req.LocationLat != nil && req.LocationLng != nil {
		radius := 1000 // Default 1km
		if req.LocationRadius != nil {
			radius = *req.LocationRadius
		}
		return s.eventRepo.FindNearby(ctx, *req.LocationLat, *req.LocationLng, radius, filter)
	}

	return s.eventRepo.FindPublic(ctx, filter)
}

// GetEvent retrieves a public event by ID (for sharing links)
func (s *PublicService) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	return s.eventRepo.FindPublicByID(ctx, eventID)
}
