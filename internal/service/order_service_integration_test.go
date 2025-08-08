package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"event/internal/models"
	"event/internal/service/mocks"
	"event/internal/testutils"
)

func TestPublicService_IsPublished_Integration(t *testing.T) {
	tests := []struct {
		name           string
		event          *models.Event
		eventRepoError error
		expectError    bool
		expectedResult bool
	}{
		{
			name:           "Published event",
			event:          testutils.TestPublishedEvent(),
			expectError:    false,
			expectedResult: true,
		},
		{
			name:           "Draft event",
			event:          testutils.TestEvent(), // Default is draft
			expectError:    false,
			expectedResult: false,
		},
		{
			name:           "Archived event",
			event:          testutils.TestArchivedEvent(),
			expectError:    false,
			expectedResult: false,
		},
		{
			name:           "Event not found",
			event:          nil,
			eventRepoError: models.ErrEventNotFound,
			expectError:    true,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup fresh mocks for each test
			eventRepo := &mocks.MockEventRepository{}
			sessionService := &SessionService{}
			publicService := NewPublicService(eventRepo, sessionService)

			ctx := context.Background()
			eventID := testutils.ValidObjectIDString()

			// Mock repository behavior
			if tt.eventRepoError != nil {
				eventRepo.On("FindByID", ctx, eventID).Return(nil, tt.eventRepoError)
			} else {
				eventRepo.On("FindByID", ctx, eventID).Return(tt.event, nil)
			}

			// Execute
			result, err := publicService.IsPublished(ctx, eventID)

			// Assert
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}

			eventRepo.AssertExpectations(t)
		})
	}
}
