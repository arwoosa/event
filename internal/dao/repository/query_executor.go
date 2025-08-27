package repository

import (
	"context"
	"fmt"

	"github.com/arwoosa/vulpes/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/arwoosa/event/internal/models"
)

// QueryExecutor handles MongoDB query execution
type QueryExecutor struct {
	collection *mongo.Collection
}

// NewQueryExecutor creates a new query executor
func NewQueryExecutor(collection *mongo.Collection) *QueryExecutor {
	return &QueryExecutor{
		collection: collection,
	}
}

// ExecuteEventQuery executes an aggregation pipeline and returns events
func (e *QueryExecutor) ExecuteEventQuery(ctx context.Context, pipeline []bson.M) ([]*models.Event, error) {
	cursor, err := e.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute unified aggregation: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in executeUnifiedQuery", log.Err(err))
		}
	}()

	// Directly decode to Event models - MongoDB driver handles aggregation results automatically
	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode events with sessions: %w", err)
	}

	return events, nil
}

// ExecuteCountQuery executes a count query and returns the total count
func (e *QueryExecutor) ExecuteCountQuery(ctx context.Context, pipeline []bson.M) (*int64, error) {
	// Create count pipeline by removing skip/limit stages
	countPipeline := []bson.M{}
	for _, stage := range pipeline {
		if _, hasSkip := stage["$skip"]; hasSkip {
			continue
		}
		if _, hasLimit := stage["$limit"]; hasLimit {
			continue
		}
		countPipeline = append(countPipeline, stage)
	}

	// Add count stage
	countPipeline = append(countPipeline, bson.M{"$count": "total"})

	countCursor, err := e.collection.Aggregate(ctx, countPipeline)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := countCursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in count query", log.Err(err))
		}
	}()

	var countResult []bson.M
	if err := countCursor.All(ctx, &countResult); err != nil {
		return nil, err
	}

	if len(countResult) == 0 {
		return nil, nil
	}

	if total, ok := countResult[0]["total"].(int32); ok {
		totalCount := int64(total)
		return &totalCount, nil
	}

	return nil, nil
}
