package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
	"github.com/arwoosa/event/internal/testutils"
)

func TestEventService_ValidatePublishedEventChanges(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	// Create a published event
	publishedEvent := testutils.TestPublishedEvent()

	tests := []struct {
		name           string
		req            *PatchEventRequest
		expectError    bool
		expectedErrMsg string
	}{
		{
			name: "Allow FAQ changes",
			req: &PatchEventRequest{
				ID: publishedEvent.ID.Hex(),
				FAQ: []*FAQRequest{
					{Question: "New question", Answer: "New answer"},
				},
				UserID: testutils.ValidObjectIDString(),
			},
			expectError: false,
		},
		{
			name: "Restrict cover image URL changes",
			req: &PatchEventRequest{
				ID:            publishedEvent.ID.Hex(),
				CoverImageURL: &[]string{"https://new-image.com/photo.jpg"}[0],
				UserID:        testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "cover_image_url",
		},
		{
			name: "Restrict detail content changes",
			req: &PatchEventRequest{
				ID: publishedEvent.ID.Hex(),
				Detail: []DetailBlockRequest{
					{
						Type: models.BlockTypeText,
						Data: models.TextData{Content: "Updated content"},
					},
				},
				UserID: testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "detail",
		},
		{
			name: "Restrict title changes",
			req: &PatchEventRequest{
				ID:     publishedEvent.ID.Hex(),
				Title:  &[]string{"New Title"}[0],
				UserID: testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "cannot modify restricted fields for published events: [title]",
		},
		{
			name: "Restrict summary changes",
			req: &PatchEventRequest{
				ID:      publishedEvent.ID.Hex(),
				Summary: &[]string{"New Summary"}[0],
				UserID:  testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "cannot modify restricted fields for published events: [summary]",
		},
		{
			name: "Restrict location changes",
			req: &PatchEventRequest{
				ID: publishedEvent.ID.Hex(),
				Location: &LocationRequest{
					Name:        "New Location",
					Address:     "New Address",
					PlaceID:     "new-place-id",
					Coordinates: &GeoJSONPointRequest{Type: "Point", Coordinates: [2]float64{1.0, 2.0}},
				},
				UserID: testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "cannot modify restricted fields for published events: [location]",
		},
		{
			name: "Restrict multiple fields",
			req: &PatchEventRequest{
				ID:      publishedEvent.ID.Hex(),
				Title:   &[]string{"New Title"}[0],
				Summary: &[]string{"New Summary"}[0],
				Location: &LocationRequest{
					Name: "New Location",
				},
				UserID: testutils.ValidObjectIDString(),
			},
			expectError:    true,
			expectedErrMsg: "cannot modify restricted fields for published events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := eventService.validateEventChanges(publishedEvent, tt.req)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEventService_PatchPublishedEvent_Integration(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	formRepo := &mocks.MockFormRepository{}
	sessionService := &SessionService{}
	orderService := &mocks.MockOrderService{}

	eventService := NewEventService(eventRepo, formRepo, sessionService, orderService)

	ctx := context.Background()

	// Create a published event
	publishedEvent := testutils.TestPublishedEvent()
	eventID := publishedEvent.ID.Hex()

	tests := []struct {
		name        string
		req         *PatchEventRequest
		expectError bool
	}{
		{
			name: "Successfully update allowed fields",
			req: &PatchEventRequest{
				ID: eventID,
				FAQ: []*FAQRequest{
					{Question: "How to register?", Answer: "Visit our website"},
				},
				UserID: testutils.ValidObjectIDString(),
			},
			expectError: false,
		},
		{
			name: "Fail to update restricted fields",
			req: &PatchEventRequest{
				ID:     eventID,
				Title:  &[]string{"Updated Title"}[0],
				UserID: testutils.ValidObjectIDString(),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the repository calls
			// Convert string ID to ObjectID for the mock
			objectID, err := primitive.ObjectIDFromHex(eventID)
			require.NoError(t, err)

			// No permission check in service layer - authorization handled by API Gateway
			eventRepo.On("FindByID", ctx, objectID).Return(publishedEvent, nil)

			if !tt.expectError {
				// Mock successful update - use MatchedBy to allow for field modifications
				eventRepo.On("Update", ctx, objectID, mock.MatchedBy(func(event *models.Event) bool {
					return event.ID == publishedEvent.ID
				})).Return(publishedEvent, nil)
			}

			// Execute
			result, err := eventService.PatchEvent(ctx, tt.req)

			// Assert
			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
			}

			eventRepo.AssertExpectations(t)
		})
	}
}
