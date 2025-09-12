package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// TestMongoFormRepository tests the MongoDB implementation of FormRepository
// These tests use a mock/in-memory approach for unit testing
func TestFormRepository_Interface(t *testing.T) {
	// This test ensures our mock repository correctly implements the interface
	var _ FormRepository = (*MockFormRepository)(nil)
}

// MockFormRepository is a simple in-memory implementation for testing
type MockFormRepository struct {
	forms map[primitive.ObjectID]*models.EventForm
	byEventID map[primitive.ObjectID]*models.EventForm
}

func NewMockFormRepository() *MockFormRepository {
	return &MockFormRepository{
		forms: make(map[primitive.ObjectID]*models.EventForm),
		byEventID: make(map[primitive.ObjectID]*models.EventForm),
	}
}

func (r *MockFormRepository) Create(ctx context.Context, form *models.EventForm) (*models.EventForm, error) {
	if form.ID.IsZero() {
		form.ID = primitive.NewObjectID()
	}
	
	// Check if form already exists for this event
	if _, exists := r.byEventID[form.EventID]; exists {
		return nil, errors.ErrFormAlreadyExists
	}
	
	r.forms[form.ID] = form
	r.byEventID[form.EventID] = form
	return form, nil
}

func (r *MockFormRepository) FindByEventID(ctx context.Context, eventID primitive.ObjectID) (*models.EventForm, error) {
	form, exists := r.byEventID[eventID]
	if !exists {
		return nil, errors.ErrFormNotFound
	}
	return form, nil
}

func (r *MockFormRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.EventForm, error) {
	form, exists := r.forms[id]
	if !exists {
		return nil, errors.ErrFormNotFound
	}
	return form, nil
}

func (r *MockFormRepository) Update(ctx context.Context, id primitive.ObjectID, form *models.EventForm) (*models.EventForm, error) {
	existing, exists := r.forms[id]
	if !exists {
		return nil, errors.ErrFormNotFound
	}
	
	// Update the form
	form.ID = id
	r.forms[id] = form
	r.byEventID[existing.EventID] = form
	return form, nil
}

func (r *MockFormRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	form, exists := r.forms[id]
	if !exists {
		return errors.ErrFormNotFound
	}
	
	delete(r.forms, id)
	delete(r.byEventID, form.EventID)
	return nil
}

func (r *MockFormRepository) DeleteByEventID(ctx context.Context, eventID primitive.ObjectID) error {
	form, exists := r.byEventID[eventID]
	if !exists {
		return errors.ErrFormNotFound
	}
	
	delete(r.forms, form.ID)
	delete(r.byEventID, eventID)
	return nil
}

func (r *MockFormRepository) ExistsByEventID(ctx context.Context, eventID primitive.ObjectID) (bool, error) {
	_, exists := r.byEventID[eventID]
	return exists, nil
}

func (r *MockFormRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	_, exists := r.forms[id]
	return exists, nil
}

// Unit tests using the mock repository
func TestFormRepository_Create_Success(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{
			"type": "object",
		},
		UISchema: map[string]interface{}{
			"type": "VerticalLayout",
		},
	}
	form.SetCreateInfo("test-user")
	
	result, err := repo.Create(ctx, form)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.ID.IsZero())
	assert.Equal(t, eventID, result.EventID)
	assert.Equal(t, "test-user", result.CreatedBy)
}

func TestFormRepository_Create_DuplicateEventID(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form1 := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form1.SetCreateInfo("test-user")
	
	form2 := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "HorizontalLayout"},
	}
	form2.SetCreateInfo("test-user")
	
	// First creation should succeed
	_, err := repo.Create(ctx, form1)
	require.NoError(t, err)
	
	// Second creation with same eventID should fail
	_, err = repo.Create(ctx, form2)
	require.Error(t, err)
	assert.Equal(t, errors.ErrFormAlreadyExists, err)
}

func TestFormRepository_FindByEventID_Success(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID:  eventID,
		Schema:   map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	// Create form first
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Find by event ID
	result, err := repo.FindByEventID(ctx, eventID)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, eventID, result.EventID)
}

func TestFormRepository_FindByEventID_NotFound(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	
	result, err := repo.FindByEventID(ctx, eventID)
	
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrFormNotFound, err)
}

func TestFormRepository_Update_Success(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	// Create form first
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Update the form
	updatedForm := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		},
		UISchema: map[string]interface{}{"type": "HorizontalLayout"},
	}
	updatedForm.SetCreateInfo("test-user")
	updatedForm.SetUpdateInfo("updater-user")
	
	result, err := repo.Update(ctx, created.ID, updatedForm)
	
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, created.ID, result.ID)
	assert.Equal(t, eventID, result.EventID)
	assert.Equal(t, "updater-user", result.UpdatedBy)
}

func TestFormRepository_Update_NotFound(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	formID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID: primitive.NewObjectID(),
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	
	result, err := repo.Update(ctx, formID, form)
	
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrFormNotFound, err)
}

func TestFormRepository_Delete_Success(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	// Create form first
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Delete the form
	err = repo.Delete(ctx, created.ID)
	
	require.NoError(t, err)
	
	// Verify it's gone
	_, err = repo.FindByID(ctx, created.ID)
	assert.Equal(t, errors.ErrFormNotFound, err)
}

func TestFormRepository_DeleteByEventID_Success(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	form := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	// Create form first
	created, err := repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Delete by event ID
	err = repo.DeleteByEventID(ctx, eventID)
	
	require.NoError(t, err)
	
	// Verify it's gone
	_, err = repo.FindByEventID(ctx, eventID)
	assert.Equal(t, errors.ErrFormNotFound, err)
	
	_, err = repo.FindByID(ctx, created.ID)
	assert.Equal(t, errors.ErrFormNotFound, err)
}

func TestFormRepository_ExistsByEventID(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	eventID := primitive.NewObjectID()
	
	// Check non-existing
	exists, err := repo.ExistsByEventID(ctx, eventID)
	require.NoError(t, err)
	assert.False(t, exists)
	
	// Create form
	form := &models.EventForm{
		EventID: eventID,
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	_, err = repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Check existing
	exists, err = repo.ExistsByEventID(ctx, eventID)
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestFormRepository_ExistsByID(t *testing.T) {
	repo := NewMockFormRepository()
	ctx := context.Background()
	
	formID := primitive.NewObjectID()
	
	// Check non-existing
	exists, err := repo.ExistsByID(ctx, formID)
	require.NoError(t, err)
	assert.False(t, exists)
	
	// Create form
	form := &models.EventForm{
		ID: formID,
		EventID: primitive.NewObjectID(),
		Schema: map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	form.SetCreateInfo("test-user")
	
	_, err = repo.Create(ctx, form)
	require.NoError(t, err)
	
	// Check existing
	exists, err = repo.ExistsByID(ctx, formID)
	require.NoError(t, err)
	assert.True(t, exists)
}