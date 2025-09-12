package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
)

func TestEventService_SetEventForm_Create(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()
	userID := "test-user-123"

	req := &SetEventFormRequest{
		EventID: eventID,
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type": "string",
				},
			},
		},
		UISchema: map[string]interface{}{
			"type": "VerticalLayout",
		},
		UserID: userID,
	}

	// Mock - event exists and is in draft status (required for form creation)
	event := &models.Event{
		ID:     eventID,
		Status: models.StatusDraft,
	}
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)
	
	// Mock - form doesn't exist yet
	formRepo.On("FindByEventID", ctx, eventID).Return(nil, errors.ErrFormNotFound)
	
	// Mock successful creation
	createdForm := &models.EventForm{
		ID:      primitive.NewObjectID(),
		EventID: eventID,
		Schema:  req.Schema,
		UISchema: req.UISchema,
	}
	createdForm.SetCreateInfo(userID)
	
	formRepo.On("Create", ctx, mock.MatchedBy(func(form *models.EventForm) bool {
		return form.EventID == eventID && form.CreatedBy == userID
	})).Return(createdForm, nil)

	// Execute
	result, err := eventService.SetEventForm(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, eventID, result.EventID)
	assert.Equal(t, userID, result.CreatedBy)
	assert.Equal(t, req.Schema, result.Schema)
	assert.Equal(t, req.UISchema, result.UISchema)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}

func TestEventService_SetEventForm_Update(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()
	userID := "test-user-123"

	req := &SetEventFormRequest{
		EventID: eventID,
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"email": map[string]interface{}{
					"type": "string",
				},
			},
		},
		UISchema: map[string]interface{}{
			"type": "HorizontalLayout",
		},
		UserID: userID,
	}

	// Mock - event exists and is in draft status (required for form update)
	event := &models.Event{
		ID:     eventID,
		Status: models.StatusDraft,
	}
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)
	
	// Mock - existing form
	existingForm := &models.EventForm{
		ID:      primitive.NewObjectID(),
		EventID: eventID,
		Schema:  map[string]interface{}{"type": "object"},
		UISchema: map[string]interface{}{"type": "VerticalLayout"},
	}
	existingForm.SetCreateInfo("original-user")
	
	formRepo.On("FindByEventID", ctx, eventID).Return(existingForm, nil)
	
	// Mock successful update
	updatedForm := &models.EventForm{
		ID:       existingForm.ID,
		EventID:  eventID,
		Schema:   req.Schema,
		UISchema: req.UISchema,
	}
	updatedForm.SetCreateInfo("original-user")
	updatedForm.SetUpdateInfo(userID)
	
	formRepo.On("Update", ctx, existingForm.ID, mock.MatchedBy(func(form *models.EventForm) bool {
		return form.EventID == eventID && form.UpdatedBy == userID
	})).Return(updatedForm, nil)

	// Execute
	result, err := eventService.SetEventForm(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, eventID, result.EventID)
	assert.Equal(t, userID, result.UpdatedBy)
	assert.Equal(t, req.Schema, result.Schema)
	assert.Equal(t, req.UISchema, result.UISchema)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}

func TestEventService_GetEventForm_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()

	// Mock - existing form
	existingForm := &models.EventForm{
		ID:      primitive.NewObjectID(),
		EventID: eventID,
		Schema: map[string]interface{}{
			"type": "object",
		},
		UISchema: map[string]interface{}{
			"type": "VerticalLayout",
		},
	}
	existingForm.SetCreateInfo("test-user")
	
	formRepo.On("FindByEventID", ctx, eventID).Return(existingForm, nil)

	// Execute
	result, err := eventService.GetEventForm(ctx, eventID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, eventID, result.EventID)
	assert.Equal(t, existingForm.ID, result.ID)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}

func TestEventService_GetEventForm_NotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()
	
	formRepo.On("FindByEventID", ctx, eventID).Return(nil, errors.ErrFormNotFound)

	// Execute
	result, err := eventService.GetEventForm(ctx, eventID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrFormNotFound, err)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}

func TestEventService_DeleteEventForm_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()
	userID := "test-user-123"
	
	// Mock - event exists
	event := &models.Event{ID: eventID}
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)
	
	formRepo.On("DeleteByEventID", ctx, eventID).Return(nil)

	// Execute
	err := eventService.DeleteEventForm(ctx, eventID, userID)

	// Assert
	require.NoError(t, err)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}

func TestEventService_DeleteEventForm_NotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()
	eventID := primitive.NewObjectID()
	userID := "test-user-123"
	
	// Mock - event exists
	event := &models.Event{ID: eventID}
	eventRepo.On("FindByID", ctx, eventID).Return(event, nil)
	
	formRepo.On("DeleteByEventID", ctx, eventID).Return(errors.ErrFormNotFound)

	// Execute
	err := eventService.DeleteEventForm(ctx, eventID, userID)

	// Assert
	require.Error(t, err)
	assert.Equal(t, errors.ErrFormNotFound, err)

	// Verify mocks
	eventRepo.AssertExpectations(t)
	formRepo.AssertExpectations(t)
}