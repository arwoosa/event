package repository

import (
	"context"
	"time"

	"github.com/arwoosa/event/internal/models"
)

// SessionRepository defines the interface for session data access
type SessionRepository interface {
	// Create inserts a new session
	Create(ctx context.Context, session *models.Session) (*models.Session, error)

	// CreateBatch inserts multiple sessions in a single operation
	CreateBatch(ctx context.Context, sessions []*models.Session) ([]*models.Session, error)

	// FindByID finds a session by its ID
	FindByID(ctx context.Context, id string) (*models.Session, error)

	// FindByEventID finds all sessions for a specific event
	FindByEventID(ctx context.Context, eventID string) ([]*models.Session, error)

	// FindByEventIDs finds sessions for multiple events (for batch operations)
	FindByEventIDs(ctx context.Context, eventIDs []string) (map[string][]*models.Session, error)

	// FindByMerchantID finds sessions by merchant ID with filtering
	FindByMerchantID(ctx context.Context, merchantID string, filter *SessionFilter) ([]*models.Session, error)

	// Update updates an existing session
	Update(ctx context.Context, id string, session *models.Session) (*models.Session, error)

	// Delete removes a session
	Delete(ctx context.Context, id string) error

	// DeleteByEventID removes all sessions for an event
	DeleteByEventID(ctx context.Context, eventID string) error

	// DeleteByEventIDs removes sessions for multiple events
	DeleteByEventIDs(ctx context.Context, eventIDs []string) error

	// CountByEventID counts sessions for an event
	CountByEventID(ctx context.Context, eventID string) (int64, error)

	// ExistsByID checks if a session exists by ID
	ExistsByID(ctx context.Context, id string) (bool, error)

	// ExistsByEventAndMerchant checks if sessions exist for an event within a merchant
	ExistsByEventAndMerchant(ctx context.Context, eventID, merchantID string) (bool, error)

	// BulkUpdateSessions performs bulk operations (create, update, delete) in a single request
	BulkUpdateSessions(ctx context.Context, creates []*models.Session, updates []*models.Session, deleteIDs []string) error
}

// SessionFilter represents filtering options for session queries
type SessionFilter struct {
	EventID       *string
	MerchantID    *string
	StartTimeFrom *time.Time
	StartTimeTo   *time.Time
	EndTimeFrom   *time.Time
	EndTimeTo     *time.Time
	SortBy        *string // start_time, end_time, created_at
	SortOrder     *string // asc, desc
	Limit         int
	Offset        int
}
