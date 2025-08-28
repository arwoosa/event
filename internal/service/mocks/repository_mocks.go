//nolint:errcheck // Mock files use type assertions that are safe in test context
package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// MockEventRepository is a mock implementation of EventRepository
type MockEventRepository struct {
	mock.Mock
}

// Create implements EventRepository interface
func (m *MockEventRepository) Create(ctx context.Context, event *models.Event) (*models.Event, error) {
	args := m.Called(ctx, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

// FindByID implements EventRepository interface
func (m *MockEventRepository) FindByID(ctx context.Context, id string) (*models.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

// Update implements EventRepository interface
func (m *MockEventRepository) Update(ctx context.Context, id string, event *models.Event) (*models.Event, error) {
	args := m.Called(ctx, id, event)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

// Delete implements EventRepository interface
func (m *MockEventRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Find implements EventRepository interface
func (m *MockEventRepository) Find(ctx context.Context, filter *repository.EventFilter) (*repository.EventListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.EventListResult), args.Error(1)
}

// FindPublic implements EventRepository interface
func (m *MockEventRepository) FindPublic(ctx context.Context, filter *repository.PublicEventFilter) (*repository.EventListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.EventListResult), args.Error(1)
}

// FindPublicByID implements EventRepository interface
func (m *MockEventRepository) FindPublicByID(ctx context.Context, id string) (*models.Event, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Event), args.Error(1)
}

// CountByStatus implements EventRepository interface
func (m *MockEventRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

// ExistsByID implements EventRepository interface
func (m *MockEventRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// Helper methods for setting up common mock scenarios

// SetupCreateSuccess sets up the mock to return a successful create
func (m *MockEventRepository) SetupCreateSuccess(event *models.Event) {
	// Set a new ID for the created event
	createdEvent := *event
	createdEvent.ID = primitive.NewObjectID()
	m.On("Create", mock.Anything, event).Return(&createdEvent, nil)
}

// SetupFindByIDSuccess sets up the mock to return an event for FindByID
func (m *MockEventRepository) SetupFindByIDSuccess(id string, event *models.Event) {
	m.On("FindByID", mock.Anything, id).Return(event, nil)
}

// SetupFindByIDNotFound sets up the mock to return not found error
func (m *MockEventRepository) SetupFindByIDNotFound(id string) {
	m.On("FindByID", mock.Anything, id).Return(nil, errors.ErrEventNotFound)
}

// MockSessionRepository is a mock implementation of SessionRepository
type MockSessionRepository struct {
	mock.Mock
}

// Create implements SessionRepository interface
func (m *MockSessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

// CreateBatch implements SessionRepository interface
func (m *MockSessionRepository) CreateBatch(ctx context.Context, sessions []*models.Session) ([]*models.Session, error) {
	args := m.Called(ctx, sessions)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Session), args.Error(1)
}

// FindByID implements SessionRepository interface
func (m *MockSessionRepository) FindByID(ctx context.Context, id string) (*models.Session, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

// FindByEventID implements SessionRepository interface
func (m *MockSessionRepository) FindByEventID(ctx context.Context, eventID string) ([]*models.Session, error) {
	args := m.Called(ctx, eventID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Session), args.Error(1)
}

// FindByEventIDs implements SessionRepository interface
func (m *MockSessionRepository) FindByEventIDs(ctx context.Context, eventIDs []string) (map[string][]*models.Session, error) {
	args := m.Called(ctx, eventIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string][]*models.Session), args.Error(1)
}

// Update implements SessionRepository interface
func (m *MockSessionRepository) Update(ctx context.Context, id string, session *models.Session) (*models.Session, error) {
	args := m.Called(ctx, id, session)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Session), args.Error(1)
}

// Delete implements SessionRepository interface
func (m *MockSessionRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// DeleteByEventID implements SessionRepository interface
func (m *MockSessionRepository) DeleteByEventID(ctx context.Context, eventID string) error {
	args := m.Called(ctx, eventID)
	return args.Error(0)
}

// CountByEventID implements SessionRepository interface
func (m *MockSessionRepository) CountByEventID(ctx context.Context, eventID string) (int64, error) {
	args := m.Called(ctx, eventID)
	return args.Get(0).(int64), args.Error(1)
}

// BulkUpdateSessions implements SessionRepository interface
func (m *MockSessionRepository) BulkUpdateSessions(ctx context.Context, create []*models.Session, update []*models.Session, deleteIDs []string) error {
	args := m.Called(ctx, create, update, deleteIDs)
	return args.Error(0)
}

// DeleteByEventIDs implements SessionRepository interface
func (m *MockSessionRepository) DeleteByEventIDs(ctx context.Context, eventIDs []string) error {
	args := m.Called(ctx, eventIDs)
	return args.Error(0)
}

// ExistsByID implements SessionRepository interface
func (m *MockSessionRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	args := m.Called(ctx, id)
	return args.Bool(0), args.Error(1)
}

// Helper methods for setting up common mock scenarios

// SetupCreateBatchSuccess sets up the mock to return successful batch creation
func (m *MockSessionRepository) SetupCreateBatchSuccess(sessions []*models.Session) {
	// Assign IDs to sessions
	createdSessions := make([]*models.Session, len(sessions))
	for i, session := range sessions {
		created := *session
		created.ID = primitive.NewObjectID()
		createdSessions[i] = &created
	}
	m.On("CreateBatch", mock.Anything, sessions).Return(createdSessions, nil)
}

// SetupFindByEventIDSuccess sets up the mock to return sessions for an event
func (m *MockSessionRepository) SetupFindByEventIDSuccess(eventID string, sessions []*models.Session) {
	m.On("FindByEventID", mock.Anything, eventID).Return(sessions, nil)
}

// SetupCountByEventIDSuccess sets up the mock to return session count
func (m *MockSessionRepository) SetupCountByEventIDSuccess(eventID string, count int64) {
	m.On("CountByEventID", mock.Anything, eventID).Return(count, nil)
}

// SetupBulkUpdateSuccess sets up the mock for successful bulk update
func (m *MockSessionRepository) SetupBulkUpdateSuccess() {
	m.On("BulkUpdateSessions", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
}
