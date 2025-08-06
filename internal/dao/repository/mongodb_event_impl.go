package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"event/internal/models"
)

// MongoEventRepository implements EventRepository using MongoDB
type MongoEventRepository struct {
	client     *mongo.Client
	database   string
	collection *mongo.Collection
}

// NewMongoEventRepository creates a new MongoDB-based event repository
func NewMongoEventRepository(client *mongo.Client, database string) EventRepository {
	return &MongoEventRepository{
		client:     client,
		database:   database,
		collection: client.Database(database).Collection("events"),
	}
}

// Create inserts a new event
func (r *MongoEventRepository) Create(ctx context.Context, event *models.Event) (*models.Event, error) {
	if event.ID.IsZero() {
		event.ID = primitive.NewObjectID()
	}

	now := time.Now()
	event.CreatedAt = now
	event.UpdatedAt = now

	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// FindByID finds an event by ID with sessions populated
func (r *MongoEventRepository) FindByID(ctx context.Context, id string) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	pipeline := []bson.M{
		// Match the specific event
		{"$match": bson.M{"_id": objectID}},
		// Lookup sessions
		{"$lookup": bson.M{
			"from":         "sessions",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "sessions",
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode event with sessions: %w", err)
	}

	if len(events) == 0 {
		return nil, models.ErrEventNotFound
	}

	return events[0], nil
}

// Update updates an existing event
func (r *MongoEventRepository) Update(ctx context.Context, id string, event *models.Event) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	event.UpdatedAt = time.Now()

	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": objectID}, event)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, models.ErrEventNotFound
	}

	return event, nil
}

// Delete removes an event
func (r *MongoEventRepository) Delete(ctx context.Context, id string) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return fmt.Errorf("invalid event ID: %w", err)
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	if result.DeletedCount == 0 {
		return models.ErrEventNotFound
	}

	return nil
}

// FindByBrandID finds events by brand ID with sessions populated and filtering
func (r *MongoEventRepository) FindByBrandID(ctx context.Context, brandID string, filter *EventFilter) (*EventListResult, error) {
	brandObjectID, err := primitive.ObjectIDFromHex(brandID)
	if err != nil {
		return nil, fmt.Errorf("invalid brand ID: %w", err)
	}

	baseQuery := bson.M{"brand_id": brandObjectID}

	// Apply filters
	if filter.Status != nil {
		baseQuery["status"] = *filter.Status
	}
	if filter.Visibility != nil {
		baseQuery["visibility"] = *filter.Visibility
	}
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		baseQuery["$text"] = bson.M{"$search": *filter.TitleSearch}
	}

	return r.buildUnifiedPipeline(ctx, baseQuery, filter.SessionStartTimeFrom, filter.SessionStartTimeTo,
		filter.SortBy, filter.SortOrder, filter.Limit, filter.Offset, filter.PageToken)
}

// FindPublic finds public events with sessions populated and filtering
func (r *MongoEventRepository) FindPublic(ctx context.Context, filter *PublicEventFilter) (*EventListResult, error) {
	baseQuery := bson.M{
		"status":     models.StatusPublished,
		"visibility": models.VisibilityPublic,
	}

	// Apply filters
	if filter.BrandID != nil {
		brandObjectID, err := primitive.ObjectIDFromHex(*filter.BrandID)
		if err != nil {
			return nil, fmt.Errorf("invalid brand ID: %w", err)
		}
		baseQuery["brand_id"] = brandObjectID
	}
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		baseQuery["$text"] = bson.M{"$search": *filter.TitleSearch}
	}

	// Handle geospatial queries
	if filter.LocationLat != nil && filter.LocationLng != nil {
		geoQuery := bson.M{
			"location.coordinates": bson.M{
				"$geoWithin": bson.M{
					"$centerSphere": []interface{}{
						[]float64{*filter.LocationLng, *filter.LocationLat},
						float64(getLocationRadius(filter.LocationRadius)) / 6378100.0, // Convert meters to earth radius in radians
					},
				},
			},
		}
		for k, v := range geoQuery {
			baseQuery[k] = v
		}
	}

	return r.buildUnifiedPipeline(ctx, baseQuery, filter.SessionStartTimeFrom, filter.SessionStartTimeTo,
		filter.SortBy, filter.SortOrder, filter.Limit, filter.Offset, filter.PageToken)
}

// FindPublicByID finds a public event by ID with sessions populated
func (r *MongoEventRepository) FindPublicByID(ctx context.Context, id string) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	pipeline := []bson.M{
		// Match public event
		{"$match": bson.M{
			"_id":    objectID,
			"status": models.StatusPublished,
		}},
		// Lookup sessions
		{"$lookup": bson.M{
			"from":         "sessions",
			"localField":   "_id",
			"foreignField": "event_id",
			"as":           "sessions",
		}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to execute aggregation: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode event with sessions: %w", err)
	}

	if len(events) == 0 {
		return nil, models.ErrEventNotFound
	}

	return events[0], nil
}

// SearchByTitle performs text search on event titles
func (r *MongoEventRepository) SearchByTitle(ctx context.Context, query string, filter *EventFilter) (*EventListResult, error) {
	searchQuery := bson.M{"$text": bson.M{"$search": query}}

	if filter.BrandID != nil {
		brandObjectID, err := primitive.ObjectIDFromHex(*filter.BrandID)
		if err != nil {
			return nil, fmt.Errorf("invalid brand ID: %w", err)
		}
		searchQuery["brand_id"] = brandObjectID
	}

	return r.executeQuery(ctx, searchQuery, filter.SortBy, filter.SortOrder, filter.Limit, filter.Offset, filter.PageToken)
}

// CountByBrandAndStatus counts events by brand and status
func (r *MongoEventRepository) CountByBrandAndStatus(ctx context.Context, brandID, status string) (int64, error) {
	brandObjectID, err := primitive.ObjectIDFromHex(brandID)
	if err != nil {
		return 0, fmt.Errorf("invalid brand ID: %w", err)
	}

	query := bson.M{
		"brand_id": brandObjectID,
		"status":   status,
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

// ExistsByID checks if an event exists by ID
func (r *MongoEventRepository) ExistsByID(ctx context.Context, id string) (bool, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, fmt.Errorf("invalid event ID: %w", err)
	}

	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": objectID})
	if err != nil {
		return false, fmt.Errorf("failed to check event existence: %w", err)
	}

	return count > 0, nil
}

// ExistsByBrandAndID checks if an event exists for a specific brand
func (r *MongoEventRepository) ExistsByBrandAndID(ctx context.Context, brandID, id string) (bool, error) {
	brandObjectID, err := primitive.ObjectIDFromHex(brandID)
	if err != nil {
		return false, fmt.Errorf("invalid brand ID: %w", err)
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return false, fmt.Errorf("invalid event ID: %w", err)
	}

	count, err := r.collection.CountDocuments(ctx, bson.M{
		"_id":      objectID,
		"brand_id": brandObjectID,
	})
	if err != nil {
		return false, fmt.Errorf("failed to check event existence: %w", err)
	}

	return count > 0, nil
}

// Helper methods

func (r *MongoEventRepository) executeQuery(ctx context.Context, query bson.M, sortBy, sortOrder *string,
	limit, offset int, pageToken *string) (*EventListResult, error) {

	opts := options.Find()

	// Handle sorting
	if sortBy != nil && sortOrder != nil {
		sortDirection := 1
		if *sortOrder == "desc" {
			sortDirection = -1
		}
		opts.SetSort(bson.D{{Key: *sortBy, Value: sortDirection}})
	} else {
		// Default sort by created_at desc
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

	// Handle pagination
	if pageToken != nil && *pageToken != "" {
		cursor, err := r.decodeCursor(*pageToken)
		if err == nil && cursor.LastID != "" {
			lastObjectID, err := primitive.ObjectIDFromHex(cursor.LastID)
			if err == nil {
				query["_id"] = bson.M{"$gt": lastObjectID}
			}
		}
	} else if offset > 0 {
		opts.SetSkip(int64(offset))
	}

	if limit > 0 {
		opts.SetLimit(int64(limit))
	}

	cursor, err := r.collection.Find(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer cursor.Close(ctx)

	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode events: %w", err)
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

// Cursor represents a pagination cursor
type Cursor struct {
	LastID    string    `json:"last_id"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *MongoEventRepository) encodeCursor(cursor *Cursor) string {
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

func (r *MongoEventRepository) decodeCursor(token string) (*Cursor, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, err
	}

	return &cursor, nil
}

func getLocationRadius(radius *int) int {
	if radius != nil {
		return *radius
	}
	return 1000 // Default 1km
}
