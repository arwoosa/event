package repository

import (
	"context"
	"time"

	"github.com/arwoosa/event/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventFilter represents filtering options for event queries
type EventFilter struct {
	MerchantID           *string
	Status               *string
	Visibility           *string
	SessionStartTimeFrom *time.Time
	SessionStartTimeTo   *time.Time
	TitleSearch          *string
	SortBy               *string // created_at, updated_at, session_start_time
	SortOrder            *string // asc, desc
	Limit                int
	Offset               int
	PageToken            *string
}

// PublicEventFilter represents filtering options for public event queries
type PublicEventFilter struct {
	MerchantID           *string
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
// All methods now return events with sessions populated
type EventRepository interface {
	// CRUD operations
	Create(ctx context.Context, event *models.Event) (*models.Event, error)
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error)
	Update(ctx context.Context, id primitive.ObjectID, event *models.Event) (*models.Event, error)
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Console API queries (with sessions populated)
	Find(ctx context.Context, filter *EventFilter) (*EventListResult, error)

	// Public API queries (with sessions populated)
	FindPublic(ctx context.Context, filter *PublicEventFilter) (*EventListResult, error)
	FindPublicByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error)

	// Specialized queries (with sessions populated)
	CountByStatus(ctx context.Context, status string) (int64, error)

	// Existence checks
	ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error)
}
