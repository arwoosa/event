package repository

import (
	"context"
	"event/internal/models"
	"time"
)

// EventFilter represents filtering options for event queries
type EventFilter struct {
	BrandID                *string
	Status                 *string
	Visibility             *string
	SessionStartTimeFrom   *time.Time
	SessionStartTimeTo     *time.Time
	TitleSearch           *string
	SortBy                *string // created_at, updated_at, session_start_time
	SortOrder             *string // asc, desc
	Limit                 int
	Offset                int
	PageToken             *string
}

// PublicEventFilter represents filtering options for public event queries
type PublicEventFilter struct {
	BrandID              *string
	TitleSearch          *string
	SessionStartTimeFrom *time.Time
	SessionStartTimeTo   *time.Time
	LocationLat          *float64
	LocationLng          *float64
	LocationRadius       *int // in meters
	SortBy               *string
	SortOrder            *string
	Limit                int
	Offset               int
	PageToken            *string
}

// Pagination represents pagination information
type Pagination struct {
	NextPageToken *string
	PrevPageToken *string
	HasNext       bool
	HasPrev       bool
	TotalCount    *int64
	CurrentPage   *int32
	TotalPages    *int32
}

// EventListResult represents the result of a paginated event query
type EventListResult struct {
	Events     []*models.Event
	Pagination *Pagination
}

// EventRepository defines the interface for event data access
type EventRepository interface {
	// CRUD operations
	Create(ctx context.Context, event *models.Event) (*models.Event, error)
	FindByID(ctx context.Context, id string) (*models.Event, error)
	Update(ctx context.Context, id string, event *models.Event) (*models.Event, error)
	Delete(ctx context.Context, id string) error
	
	// Console API queries
	FindByBrandID(ctx context.Context, brandID string, filter *EventFilter) (*EventListResult, error)
	
	// Public API queries
	FindPublic(ctx context.Context, filter *PublicEventFilter) (*EventListResult, error)
	FindPublicByID(ctx context.Context, id string) (*models.Event, error)
	
	// Specialized queries
	FindNearby(ctx context.Context, lat, lng float64, radius int, filter *PublicEventFilter) (*EventListResult, error)
	SearchByTitle(ctx context.Context, query string, filter *EventFilter) (*EventListResult, error)
	CountByBrandAndStatus(ctx context.Context, brandID, status string) (int64, error)
	
	// Existence checks
	ExistsByID(ctx context.Context, id string) (bool, error)
	ExistsByBrandAndID(ctx context.Context, brandID, id string) (bool, error)
}