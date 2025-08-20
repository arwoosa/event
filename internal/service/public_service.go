package service

import (
	"context"
	"time"

	"github.com/arwoosa/event/conf"
	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/models"
)

// PublicService implements the business logic for public event access
type PublicService struct {
	eventRepo        repository.EventRepository
	sessionService   *SessionService
	paginationConfig *conf.PaginationConfig
}

// NewPublicService creates a new public service
func NewPublicService(eventRepo repository.EventRepository, sessionService *SessionService, paginationConfig *conf.PaginationConfig) *PublicService {
	return &PublicService{
		eventRepo:        eventRepo,
		sessionService:   sessionService,
		paginationConfig: paginationConfig,
	}
}

// SearchEventsRequest represents the request to search public events
type SearchEventsRequest struct {
	MerchantID           *string
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
		Limit:  20, // Default
		Offset: 0,
	}

	// Only set non-nil and non-empty values
	if req.MerchantID != nil && *req.MerchantID != "" {
		filter.MerchantID = req.MerchantID
	}
	if req.TitleSearch != nil && *req.TitleSearch != "" {
		filter.TitleSearch = req.TitleSearch
	}
	if req.LocationLat != nil {
		filter.LocationLat = req.LocationLat
	}
	if req.LocationLng != nil {
		filter.LocationLng = req.LocationLng
	}
	if req.LocationRadius != nil {
		filter.LocationRadius = req.LocationRadius
	}
	if req.SortBy != nil && *req.SortBy != "" {
		filter.SortBy = req.SortBy
	}
	if req.SortOrder != nil && *req.SortOrder != "" {
		filter.SortOrder = req.SortOrder
	}
	if req.PageToken != nil && *req.PageToken != "" {
		filter.PageToken = req.PageToken
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

	// Handle pagination with fallback to hardcoded defaults if config is not available
	defaultPageSize := 20 // Default fallback
	maxPageSize := 100    // Default fallback
	if s.paginationConfig != nil {
		if s.paginationConfig.DefaultPageSize > 0 {
			defaultPageSize = s.paginationConfig.DefaultPageSize
		}
		if s.paginationConfig.MaxPageSize > 0 {
			maxPageSize = s.paginationConfig.MaxPageSize
		}
	}

	filter.Limit = defaultPageSize
	if req.PageSize != nil {
		if *req.PageSize > 0 && int(*req.PageSize) <= maxPageSize {
			filter.Limit = int(*req.PageSize)
		}
	}
	if req.Page != nil && *req.Page > 0 {
		filter.Offset = int((*req.Page - 1) * int32(filter.Limit))
		filter.PageToken = nil // Don't use cursor pagination if page is specified
	}

	// All searches now use the unified FindPublic method
	// Geospatial parameters are already set in the filter above
	return s.eventRepo.FindPublic(ctx, filter)
}

// GetEvent retrieves a public event by ID (for sharing links)
func (s *PublicService) GetEvent(ctx context.Context, eventID string) (*models.Event, error) {
	return s.eventRepo.FindPublicByID(ctx, eventID)
}

// IsPublished checks if an event is published (for OrderService)
func (s *PublicService) IsPublished(ctx context.Context, eventID string) (bool, error) {
	// Find event by ID without merchant filtering (for internal service use)
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return false, err
	}

	// Check if event is published
	isPublished := event.Status == models.StatusPublished
	return isPublished, nil
}
