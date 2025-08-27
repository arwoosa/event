package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PipelineBuilder builds MongoDB aggregation pipelines
type PipelineBuilder struct {
	pipeline []bson.M
	repo     *MongoEventRepository
}

// NewPipelineBuilder creates a new pipeline builder
func NewPipelineBuilder(repo *MongoEventRepository) *PipelineBuilder {
	return &PipelineBuilder{
		pipeline: []bson.M{},
		repo:     repo,
	}
}

// AddMatchStage adds match conditions to the pipeline
func (b *PipelineBuilder) AddMatchStage(baseQuery bson.M, pageToken *string, sortOrder *string) error {
	matchConditions := bson.M{}

	// Add base query conditions
	for key, value := range baseQuery {
		matchConditions[key] = value
	}

	// Add cursor pagination condition if provided
	if pageToken != nil && *pageToken != "" {
		cursor, err := b.repo.decodeCursor(*pageToken)
		if err != nil {
			return fmt.Errorf("cursor validation failed: %w", err)
		}
		if cursor.LastID != "" {
			lastObjectID, err := primitive.ObjectIDFromHex(cursor.LastID)
			if err != nil {
				return fmt.Errorf("invalid cursor ID: %w", err)
			}
			// Determine sort direction (default is desc for created_at)
			isDescending := sortOrder == nil || *sortOrder != sortOrderAsc
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
		b.pipeline = append(b.pipeline, bson.M{"$match": matchConditions})
	}

	return nil
}

// AddSessionLookupStage adds session lookup to the pipeline
func (b *PipelineBuilder) AddSessionLookupStage(sessionStartTimeFrom, sessionStartTimeTo *time.Time) {
	if hasSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo) {
		// Build session time filter conditions
		sessionTimeFilter := b.buildSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo)

		// Lookup only matching sessions (filter during lookup)
		lookupStage := bson.M{
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
		b.pipeline = append(b.pipeline, lookupStage)

		// Only keep events that have at least one matching session
		b.pipeline = append(b.pipeline, bson.M{
			"$match": bson.M{
				"sessions": bson.M{"$ne": []interface{}{}},
			},
		})
	} else {
		// Standard session lookup without filtering
		b.pipeline = append(b.pipeline, bson.M{
			"$lookup": bson.M{
				"from":         "sessions",
				"localField":   "_id",
				"foreignField": "event_id",
				"as":           "sessions",
			},
		})
	}
}

// buildSessionTimeFilter builds session time filter conditions
func (b *PipelineBuilder) buildSessionTimeFilter(sessionStartTimeFrom, sessionStartTimeTo *time.Time) bson.M {
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
	return sessionTimeFilter
}

// AddSkipStage adds skip stage for offset pagination
func (b *PipelineBuilder) AddSkipStage(offset int) {
	if offset > 0 {
		b.pipeline = append(b.pipeline, bson.M{"$skip": offset})
	}
}

// AddSortStage adds sort stage to the pipeline
func (b *PipelineBuilder) AddSortStage(sortBy, sortOrder *string) {
	sortStage := bson.M{}
	sortField := "created_at" // Default sort field
	sortDirection := -1       // Default descending

	if sortBy != nil && *sortBy != "" {
		sortField = *sortBy
	}

	if sortOrder != nil && *sortOrder == sortOrderAsc {
		sortDirection = 1
	}

	sortStage[sortField] = sortDirection
	b.pipeline = append(b.pipeline, bson.M{"$sort": sortStage})
}

// AddLimitStage adds limit stage to the pipeline
func (b *PipelineBuilder) AddLimitStage(limit int, pageToken *string) {
	if limit > 0 {
		actualLimit := limit
		isCursorPagination := pageToken != nil && *pageToken != ""
		if isCursorPagination {
			// For cursor-based pagination, fetch limit+1 to determine if there are more results
			actualLimit = limit + 1
		}
		b.pipeline = append(b.pipeline, bson.M{"$limit": actualLimit})
	}
}

// Build returns the constructed pipeline
func (b *PipelineBuilder) Build() []bson.M {
	return b.pipeline
}

// BuildUnifiedPipeline builds a complete unified aggregation pipeline
func (b *PipelineBuilder) BuildUnifiedPipeline(
	ctx context.Context, baseQuery bson.M,
	sessionStartTimeFrom, sessionStartTimeTo *time.Time, sortBy, sortOrder *string,
	limit, offset int, pageToken *string,
) ([]bson.M, error) {
	// Step 1: Match base query conditions and cursor pagination
	if err := b.AddMatchStage(baseQuery, pageToken, sortOrder); err != nil {
		return nil, err
	}

	// Step 2: Lookup sessions from sessions collection
	b.AddSessionLookupStage(sessionStartTimeFrom, sessionStartTimeTo)

	// Step 3: Handle offset pagination (cursor is already handled in Step 1)
	b.AddSkipStage(offset)

	// Step 4: Handle sorting
	b.AddSortStage(sortBy, sortOrder)

	// Step 5: Limit results (use limit+1 for cursor-based pagination to check if there are more results)
	b.AddLimitStage(limit, pageToken)

	return b.Build(), nil
}
