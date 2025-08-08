package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/models"
)

// hasSessionTimeFilter checks if session time filtering is needed
func hasSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo *time.Time) bool {
	return sessionStartTimeFrom != nil || sessionStartTimeTo != nil
}

// buildUnifiedPipeline builds a unified aggregation pipeline that combines events and sessions
// This method properly handles session time filtering and returns only matching sessions
func (r *MongoEventRepository) buildUnifiedPipeline(ctx context.Context, baseQuery bson.M,
	sessionStartTimeFrom, sessionStartTimeTo *time.Time, sortBy, sortOrder *string,
	limit, offset int, pageToken *string) (*EventListResult, error) {

	pipeline := []bson.M{}

	// Determine sort direction (default is desc for created_at)
	isDescending := true
	if sortOrder != nil && *sortOrder == "asc" {
		isDescending = false
	}

	// Step 1: Match base query conditions and cursor pagination
	matchConditions := bson.M{}

	// Add base query conditions
	for key, value := range baseQuery {
		matchConditions[key] = value
	}

	// Add cursor pagination condition if provided
	if pageToken != nil && *pageToken != "" {
		cursor, err := r.decodeCursor(*pageToken)
		if err != nil {
			return nil, fmt.Errorf("cursor validation failed: %w", err)
		}
		if cursor.LastID != "" {
			lastObjectID, err := primitive.ObjectIDFromHex(cursor.LastID)
			if err != nil {
				return nil, fmt.Errorf("invalid cursor ID: %w", err)
			}
			// For descending sort, use $lt (less than) - get records after this cursor
			// For ascending sort, use $gt (greater than) - get records after this cursor
			if isDescending {
				matchConditions["_id"] = bson.M{"$lt": lastObjectID}
			} else {
				matchConditions["_id"] = bson.M{"$gt": lastObjectID}
			}
		}
	}

	// Add coordinate validity filter
	matchConditions["location.coordinates.coordinates"] = bson.M{"$exists": true, "$type": "array"}

	if len(matchConditions) > 0 {
		pipeline = append(pipeline, bson.M{"$match": matchConditions})
	}

	// Step 2: Lookup sessions from sessions collection
	pipeline = append(pipeline, bson.M{
		"$lookup": bson.M{
			"from":         "sessions",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "sessions",
		},
	})

	// Step 3: Apply session time filtering if needed - simplified approach
	if hasSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo) {
		// Build session time filter conditions
		sessionTimeFilter := bson.M{}
		if sessionStartTimeFrom != nil {
			sessionTimeFilter["start_time"] = bson.M{"$gte": *sessionStartTimeFrom}
		}
		if sessionStartTimeTo != nil {
			if existing, ok := sessionTimeFilter["start_time"].(bson.M); ok {
				existing["$lte"] = *sessionStartTimeTo
			} else {
				sessionTimeFilter["start_time"] = bson.M{"$lte": *sessionStartTimeTo}
			}
		}

		// Lookup only matching sessions (filter during lookup)
		pipeline[len(pipeline)-1] = bson.M{
			"$lookup": bson.M{
				"from": "sessions",
				"let":  bson.M{"event_id": "$_id"},
				"pipeline": []bson.M{
					{
						"$match": bson.M{
							"$expr": bson.M{"$eq": []string{"$event_id", "$$event_id"}},
						},
					},
					{
						"$match": sessionTimeFilter,
					},
				},
				"as": "sessions",
			},
		}

		// Only keep events that have at least one matching session
		pipeline = append(pipeline, bson.M{
			"$match": bson.M{
				"sessions": bson.M{"$ne": []interface{}{}},
			},
		})
	}

	// Step 4: Handle offset pagination (cursor is already handled in Step 1)
	if offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": offset})
	}

	// Step 5: Handle sorting
	sortStage := bson.M{}
	sortField := "created_at" // 預設排序欄位
	sortDirection := -1       // 預設降序

	if sortBy != nil && *sortBy != "" {
		sortField = *sortBy
	}

	if sortOrder != nil && *sortOrder == "asc" {
		sortDirection = 1
	}

	sortStage[sortField] = sortDirection
	pipeline = append(pipeline, bson.M{"$sort": sortStage})

	// Step 6: Limit results (use limit+1 for cursor-based pagination to check if there are more results)
	if limit > 0 {
		actualLimit := limit
		isCursorPagination := pageToken != nil && *pageToken != ""
		if isCursorPagination {
			// For cursor-based pagination, fetch limit+1 to determine if there are more results
			actualLimit = limit + 1
		}
		pipeline = append(pipeline, bson.M{"$limit": actualLimit})
	}

	return r.executeUnifiedQuery(ctx, pipeline, limit, offset, pageToken)
}

// executeUnifiedQuery executes the unified aggregation pipeline and returns EventListResult
func (r *MongoEventRepository) executeUnifiedQuery(ctx context.Context, pipeline []bson.M,
	limit, offset int, pageToken *string) (*EventListResult, error) {

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute unified aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	// Directly decode to Event models - MongoDB driver handles aggregation results automatically
	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode events with sessions: %w", err)
	}

	// Build pagination info
	hasNext := false
	isCursorPagination := pageToken != nil && *pageToken != ""
	if isCursorPagination {
		// For cursor-based pagination, check if we got limit+1 results
		hasNext = len(events) > limit
		if hasNext {
			// Remove the extra result used for pagination check
			events = events[:limit]
		}
	} else {
		// For offset-based pagination, assume there might be more (traditional behavior)
		hasNext = len(events) == limit
	}

	pagination := &Pagination{
		HasNext: hasNext,
		HasPrev: offset > 0, // For offset pagination
	}

	// For cursor pagination, any valid cursor means we're not on first page
	if pageToken != nil && *pageToken != "" {
		pagination.HasPrev = true
	}

	if len(events) > 0 && pagination.HasNext {
		lastEvent := events[len(events)-1]
		nextToken := r.encodeCursor(&Cursor{
			LastID:    lastEvent.ID.Hex(),
			Timestamp: lastEvent.CreatedAt,
		})
		pagination.NextPageToken = &nextToken
	}

	return &EventListResult{
		Events:     events,
		Pagination: pagination,
	}, nil
}
