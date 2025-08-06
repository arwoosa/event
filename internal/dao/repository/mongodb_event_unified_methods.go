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

	// Step 1: Match base query conditions
	if len(baseQuery) > 0 {
		pipeline = append(pipeline, bson.M{"$match": baseQuery})
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

	// Step 4: Handle pagination with cursor or offset
	if pageToken != nil && *pageToken != "" {
		cursor, err := r.decodeCursor(*pageToken)
		if err == nil && cursor.LastID != "" {
			lastObjectID, err := primitive.ObjectIDFromHex(cursor.LastID)
			if err == nil {
				pipeline = append(pipeline, bson.M{
					"$match": bson.M{"_id": bson.M{"$gt": lastObjectID}},
				})
			}
		}
	} else if offset > 0 {
		pipeline = append(pipeline, bson.M{"$skip": offset})
	}

	// Step 5: Handle sorting
	sortStage := bson.M{}
	if sortBy != nil && sortOrder != nil {
		sortDirection := 1
		if *sortOrder == "desc" {
			sortDirection = -1
		}
		sortStage[*sortBy] = sortDirection
	} else {
		sortStage["created_at"] = -1
	}
	pipeline = append(pipeline, bson.M{"$sort": sortStage})

	// Step 6: Limit results
	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
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
	pagination := &Pagination{
		HasNext: len(events) == limit,
		HasPrev: offset > 0 || (pageToken != nil && *pageToken != ""),
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

