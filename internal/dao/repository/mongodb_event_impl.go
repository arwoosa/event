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

	// Generate IDs for sessions if not provided
	for i := range event.Sessions {
		if event.Sessions[i].ID.IsZero() {
			event.Sessions[i].ID = primitive.NewObjectID()
		}
	}

	_, err := r.collection.InsertOne(ctx, event)
	if err != nil {
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	return event, nil
}

// FindByID finds an event by its ID
func (r *MongoEventRepository) FindByID(ctx context.Context, id string) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	var event models.Event
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to find event: %w", err)
	}

	return &event, nil
}

// Update updates an existing event
func (r *MongoEventRepository) Update(ctx context.Context, id string, event *models.Event) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	event.UpdatedAt = time.Now()

	// Generate IDs for new sessions
	for i := range event.Sessions {
		if event.Sessions[i].ID.IsZero() {
			event.Sessions[i].ID = primitive.NewObjectID()
		}
	}

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

// FindByBrandID finds events by brand ID with filtering
func (r *MongoEventRepository) FindByBrandID(ctx context.Context, brandID string, filter *EventFilter) (*EventListResult, error) {
	brandObjectID, err := primitive.ObjectIDFromHex(brandID)
	if err != nil {
		return nil, fmt.Errorf("invalid brand ID: %w", err)
	}

	query := bson.M{"brand_id": brandObjectID}

	// Apply filters
	if filter.Status != nil {
		query["status"] = *filter.Status
	}
	if filter.Visibility != nil {
		query["visibility"] = *filter.Visibility
	}
	if filter.SessionStartTimeFrom != nil || filter.SessionStartTimeTo != nil {
		sessionFilter := bson.M{}
		if filter.SessionStartTimeFrom != nil {
			sessionFilter["$gte"] = *filter.SessionStartTimeFrom
		}
		if filter.SessionStartTimeTo != nil {
			sessionFilter["$lte"] = *filter.SessionStartTimeTo
		}
		query["sessions.start_time"] = sessionFilter
	}
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		query["$text"] = bson.M{"$search": *filter.TitleSearch}
	}

	return r.executeQuery(ctx, query, filter.SortBy, filter.SortOrder, filter.Limit, filter.Offset, filter.PageToken)
}

// FindPublic finds public events
func (r *MongoEventRepository) FindPublic(ctx context.Context, filter *PublicEventFilter) (*EventListResult, error) {
	query := bson.M{
		"status":     models.StatusPublished,
		"visibility": models.VisibilityPublic,
	}

	// Apply filters
	if filter.BrandID != nil {
		brandObjectID, err := primitive.ObjectIDFromHex(*filter.BrandID)
		if err != nil {
			return nil, fmt.Errorf("invalid brand ID: %w", err)
		}
		query["brand_id"] = brandObjectID
	}
	if filter.SessionStartTimeFrom != nil || filter.SessionStartTimeTo != nil {
		sessionFilter := bson.M{}
		if filter.SessionStartTimeFrom != nil {
			sessionFilter["$gte"] = *filter.SessionStartTimeFrom
		}
		if filter.SessionStartTimeTo != nil {
			sessionFilter["$lte"] = *filter.SessionStartTimeTo
		}
		query["sessions.start_time"] = sessionFilter
	}
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		query["$text"] = bson.M{"$search": *filter.TitleSearch}
	}

	// Handle geospatial queries separately if provided
	if filter.LocationLat != nil && filter.LocationLng != nil {
		return r.findNearbyInternal(ctx, *filter.LocationLat, *filter.LocationLng,
			getLocationRadius(filter.LocationRadius), query, filter.SortBy, filter.SortOrder,
			filter.Limit, filter.Offset, filter.PageToken)
	}

	return r.executeQuery(ctx, query, filter.SortBy, filter.SortOrder, filter.Limit, filter.Offset, filter.PageToken)
}

// FindPublicByID finds a public event by ID (for sharing)
func (r *MongoEventRepository) FindPublicByID(ctx context.Context, id string) (*models.Event, error) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid event ID: %w", err)
	}

	var event models.Event
	query := bson.M{
		"_id":    objectID,
		"status": models.StatusPublished,
	}

	err = r.collection.FindOne(ctx, query).Decode(&event)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrEventNotFound
		}
		return nil, fmt.Errorf("failed to find event: %w", err)
	}

	return &event, nil
}

// FindNearby finds events near a location
func (r *MongoEventRepository) FindNearby(ctx context.Context, lat, lng float64, radius int, filter *PublicEventFilter) (*EventListResult, error) {
	query := bson.M{
		"status":     models.StatusPublished,
		"visibility": models.VisibilityPublic,
	}

	if filter.BrandID != nil {
		brandObjectID, err := primitive.ObjectIDFromHex(*filter.BrandID)
		if err != nil {
			return nil, fmt.Errorf("invalid brand ID: %w", err)
		}
		query["brand_id"] = brandObjectID
	}

	return r.findNearbyInternal(ctx, lat, lng, radius, query, filter.SortBy, filter.SortOrder,
		filter.Limit, filter.Offset, filter.PageToken)
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

func (r *MongoEventRepository) findNearbyInternal(ctx context.Context, lat, lng float64, radius int,
	baseQuery bson.M, sortBy, sortOrder *string, limit, offset int, pageToken *string) (*EventListResult, error) {

	// Add geospatial query
	geoQuery := bson.M{
		"location.coordinates": bson.M{
			"$geoWithin": bson.M{
				"$centerSphere": []interface{}{
					[]float64{lng, lat},
					float64(radius) / 6378100.0, // Convert meters to radians
				},
			},
		},
	}

	// Merge with base query
	for k, v := range geoQuery {
		baseQuery[k] = v
	}

	return r.executeQuery(ctx, baseQuery, sortBy, sortOrder, limit, offset, pageToken)
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
