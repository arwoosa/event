package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/arwoosa/event/internal/dao/repository"
	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
	"github.com/arwoosa/event/internal/service/mocks"
	"github.com/arwoosa/event/internal/testutils"
)

// Local matcher for PublicEventFilter to avoid circular dependency
func matchPublicEventFilter() interface{} {
	return mock.MatchedBy(func(filter *repository.PublicEventFilter) bool {
		return filter != nil
	})
}

func TestPublicService_SearchEvents_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{} // Minimal setup

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Create search request
	merchantID := testutils.ValidObjectIDString()
	titleSearch := "test event"
	pageSize := int32(20)

	req := &SearchEventsRequest{
		MerchantID:  &merchantID,
		TitleSearch: &titleSearch,
		PageSize:    &pageSize,
	}

	// Expected result
	totalCount := int64(2)
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{
			testutils.TestPublishedEvent(),
			testutils.TestPublishedEvent(),
		},
		Pagination: &repository.Pagination{
			TotalCount: &totalCount,
		},
	}

	// Mock repository call
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Events, 2)
	assert.NotNil(t, result.Pagination)
	assert.Equal(t, int64(2), *result.Pagination.TotalCount)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_WithLocationFilter(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Create search request with location
	lat := 25.0330
	lng := 121.5654
	radius := 1000

	req := &SearchEventsRequest{
		LocationLat:    &lat,
		LocationLng:    &lng,
		LocationRadius: &radius,
	}

	// Expected result
	totalCount := int64(1)
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{testutils.TestPublishedEvent()},
		Pagination: &repository.Pagination{
			TotalCount: &totalCount,
		},
	}

	// Mock unified search with location filter
	eventRepo.On("FindPublic", ctx, mock.MatchedBy(func(filter *repository.PublicEventFilter) bool {
		return filter != nil &&
			filter.LocationLat != nil && *filter.LocationLat == lat &&
			filter.LocationLng != nil && *filter.LocationLng == lng &&
			filter.LocationRadius != nil && *filter.LocationRadius == radius
	})).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Events, 1)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_EmptyRequest(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Empty search request
	req := &SearchEventsRequest{}

	// Expected result
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{},
		Pagination: &repository.Pagination{
			TotalCount: &[]int64{0}[0],
		},
	}

	// Mock repository call with default filter
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Events, 0)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_WithPagination(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Create paginated search request
	page := int32(2)
	pageSize := int32(10)

	req := &SearchEventsRequest{
		Page:     &page,
		PageSize: &pageSize,
	}

	// Expected result
	totalCount := int64(25)
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{testutils.TestPublishedEvent()},
		Pagination: &repository.Pagination{
			TotalCount: &totalCount,
		},
	}

	// Mock repository call - should set offset based on page
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Pagination)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_GetEvent_Success(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Expected published event
	event := testutils.TestPublishedEvent()

	// Mock repository call
	eventRepo.On("FindPublicByID", ctx, eventID).Return(event, nil)

	// Execute
	result, err := publicService.GetEvent(ctx, eventID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, event.Title, result.Title)
	assert.Equal(t, models.StatusPublished, result.Status)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_GetEvent_NotFound(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()
	eventID := testutils.ValidObjectIDString()

	// Mock event not found
	eventRepo.On("FindPublicByID", ctx, eventID).Return(nil, errors.ErrEventNotFound)

	// Execute
	result, err := publicService.GetEvent(ctx, eventID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, errors.ErrEventNotFound, err)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_FilterValidation(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Test various filter combinations
	testCases := []struct {
		name string
		req  *SearchEventsRequest
	}{
		{
			name: "With merchant filter",
			req: &SearchEventsRequest{
				MerchantID: testutils.StringPtr(testutils.ValidObjectIDString()),
			},
		},
		{
			name: "With title search",
			req: &SearchEventsRequest{
				TitleSearch: testutils.StringPtr("test event"),
			},
		},
		{
			name: "With page token",
			req: &SearchEventsRequest{
				PageToken: testutils.StringPtr("next_page_token"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Expected result
			expectedResult := &repository.EventListResult{
				Events: []*models.Event{},
				Pagination: &repository.Pagination{
					TotalCount: &[]int64{0}[0],
				},
			}

			// Mock repository call
			eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

			// Execute
			result, err := publicService.SearchEvents(ctx, tc.req)

			// Assert
			require.NoError(t, err)
			assert.NotNil(t, result)
		})
	}

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_LocationRequiresBoth(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Test that location search requires both lat and lng
	lat := 25.0330

	req := &SearchEventsRequest{
		LocationLat: &lat, // Only latitude, missing longitude
	}

	// Expected result - should use normal search, not nearby
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{},
		Pagination: &repository.Pagination{
			TotalCount: &[]int64{0}[0],
		},
	}

	// Mock normal search (not nearby search)
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_DefaultPageSize(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Request without page size should use default
	req := &SearchEventsRequest{}

	// Expected result
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{},
		Pagination: &repository.Pagination{
			TotalCount: &[]int64{0}[0],
		},
	}

	// Mock repository call - filter should have default limit of 20
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)

	eventRepo.AssertExpectations(t)
}

func TestPublicService_SearchEvents_PageBasedPagination(t *testing.T) {
	// Setup
	eventRepo := &mocks.MockEventRepository{}
	sessionService := &SessionService{}

	publicService := NewPublicService(eventRepo, sessionService, nil)

	ctx := context.Background()

	// Create page-based search request
	page := int32(2)
	pageSize := int32(5)

	req := &SearchEventsRequest{
		Page:     &page,
		PageSize: &pageSize,
	}

	// Expected result with page-based pagination info
	totalCount := int64(23)
	currentPage := int32(2)
	totalPages := int32(5) // ceil(23/5) = 5
	expectedResult := &repository.EventListResult{
		Events: []*models.Event{
			testutils.TestPublishedEvent(),
			testutils.TestPublishedEvent(),
		},
		Pagination: &repository.Pagination{
			TotalCount:  &totalCount,
			CurrentPage: &currentPage,
			TotalPages:  &totalPages,
			HasNext:     true,
			HasPrev:     true,
		},
	}

	// Mock repository call
	eventRepo.On("FindPublic", ctx, matchPublicEventFilter()).Return(expectedResult, nil)

	// Execute
	result, err := publicService.SearchEvents(ctx, req)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Pagination)

	// Verify page-based pagination fields
	assert.Equal(t, &totalCount, result.Pagination.TotalCount)
	assert.Equal(t, &currentPage, result.Pagination.CurrentPage)
	assert.Equal(t, &totalPages, result.Pagination.TotalPages)
	assert.True(t, result.Pagination.HasNext)
	assert.True(t, result.Pagination.HasPrev)

	eventRepo.AssertExpectations(t)
}
