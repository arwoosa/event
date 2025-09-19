package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	sortOrderAsc = "asc"
)

// hasSessionTimeFilter checks if session time filtering is needed
func hasSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo *time.Time) bool {
	return sessionStartTimeFrom != nil || sessionStartTimeTo != nil
}

// buildUnifiedPipeline builds a unified aggregation pipeline that combines events and sessions
// This method properly handles session time filtering and returns only matching sessions
func (r *MongoEventRepository) buildUnifiedPipeline(ctx context.Context, baseQuery bson.M,
	sessionStartTimeFrom, sessionStartTimeTo *time.Time, sortBy, sortOrder *string,
	limit, offset int, pageToken *string,
) (*EventListResult, error) {
	// Use pipeline builder to construct the aggregation pipeline
	builder := NewPipelineBuilder(r)
	pipeline, err := builder.BuildUnifiedPipeline(ctx, baseQuery, sessionStartTimeFrom, sessionStartTimeTo, sortBy, sortOrder, limit, offset, pageToken)
	if err != nil {
		return nil, err
	}

	return r.executeUnifiedQuery(ctx, pipeline, limit, offset, pageToken)
}

// executeUnifiedQuery executes the unified aggregation pipeline and returns EventListResult
func (r *MongoEventRepository) executeUnifiedQuery(ctx context.Context, pipeline []bson.M,
	limit, offset int, pageToken *string,
) (*EventListResult, error) {
	// Execute main query
	executor := NewQueryExecutor(r.collection)
	events, err := executor.ExecuteEventQuery(ctx, pipeline)
	if err != nil {
		return nil, err
	}

	// Get total count for offset-based pagination
	var totalCount *int64
	isCursorPagination := pageToken != nil && *pageToken != ""
	if !isCursorPagination && offset >= 0 {
		if count, err := executor.ExecuteCountQuery(ctx, pipeline); err == nil {
			totalCount = count
		}
		// If count fails, just proceed without page info (totalCount stays nil)
	}

	// Process pagination
	processor := NewPaginationProcessor(r)
	paginationInput := &PaginationInput{
		Events:     events,
		Limit:      limit,
		Offset:     offset,
		PageToken:  pageToken,
		TotalCount: totalCount,
	}
	pagination := processor.ProcessPagination(paginationInput)

	return &EventListResult{
		Events:     paginationInput.Events, // events may have been modified by pagination processor
		Pagination: pagination,
	}, nil
}
