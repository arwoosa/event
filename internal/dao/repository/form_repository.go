package repository

import (
	"context"

	"github.com/arwoosa/event/internal/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FormRepository defines the interface for event form data access
// Since each event has only one form, the interface is simpler than event repository
type FormRepository interface {
	// CRUD operations
	// Create creates a new form for an event
	Create(ctx context.Context, form *models.EventForm) (*models.EventForm, error)

	// FindByEventID finds the form for a specific event
	FindByEventID(ctx context.Context, eventID primitive.ObjectID) (*models.EventForm, error)

	// Update updates an existing form
	Update(ctx context.Context, id primitive.ObjectID, form *models.EventForm) (*models.EventForm, error)

	// Delete deletes a form by event ID
	DeleteByEventID(ctx context.Context, eventID primitive.ObjectID) error

	// Delete deletes a form by ID
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Existence checks
	ExistsByEventID(ctx context.Context, eventID primitive.ObjectID) (bool, error)
	ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error)

	// Find form by ID (for internal use)
	FindByID(ctx context.Context, id primitive.ObjectID) (*models.EventForm, error)
}
