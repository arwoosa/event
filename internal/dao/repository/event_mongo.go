package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/arwoosa/vulpes/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/arwoosa/event/conf"
	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// MongoEventRepository implements EventRepository using MongoDB
type MongoEventRepository struct {
	client           *mongo.Client
	database         string
	collection       *mongo.Collection
	paginationConfig *conf.PaginationConfig
}

// NewMongoEventRepository creates a new MongoDB-based event repository
func NewMongoEventRepository(client *mongo.Client, database string, paginationConfig *conf.PaginationConfig) EventRepository {
	return &MongoEventRepository{
		client:           client,
		database:         database,
		collection:       client.Database(database).Collection("events"),
		paginationConfig: paginationConfig,
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
func (r *MongoEventRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
	pipeline := []bson.M{
		// Match the specific event
		{"$match": bson.M{"_id": id}},
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
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in FindByID", log.Err(err))
		}
	}()

	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode event with sessions: %w", err)
	}

	if len(events) == 0 {
		return nil, errors.ErrEventNotFound
	}

	return events[0], nil
}

// Update updates an existing event
func (r *MongoEventRepository) Update(ctx context.Context, id primitive.ObjectID, event *models.Event) (*models.Event, error) {
	event.UpdatedAt = time.Now()

	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": id}, event)
	if err != nil {
		return nil, fmt.Errorf("failed to update event: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, errors.ErrEventNotFound
	}

	return event, nil
}

// Delete removes an event
func (r *MongoEventRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.ErrEventNotFound
	}

	return nil
}

// Find finds events with sessions populated and filtering
func (r *MongoEventRepository) Find(ctx context.Context, filter *EventFilter) (*EventListResult, error) {
	baseQuery := bson.M{}

	// Apply merchant_id filter for multi-tenant isolation
	if filter.MerchantID != nil {
		baseQuery["merchant_id"] = *filter.MerchantID
	}

	// Apply filters
	if filter.Status != nil {
		baseQuery["status"] = *filter.Status
	}
	if filter.Visibility != nil {
		baseQuery["visibility"] = *filter.Visibility
	}
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		baseQuery["title"] = bson.M{"$regex": *filter.TitleSearch, "$options": "i"} // ignore case
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

	// Apply merchant_id filter for multi-tenant isolation
	if filter.MerchantID != nil && *filter.MerchantID != "" {
		baseQuery["merchant_id"] = *filter.MerchantID
	}

	// Apply filters
	if filter.TitleSearch != nil && *filter.TitleSearch != "" {
		baseQuery["title"] = bson.M{"$regex": *filter.TitleSearch, "$options": "i"} // ignore case
	}

	// Handle geospatial queries
	if filter.LocationLat != nil && filter.LocationLng != nil {
		geoQuery := bson.M{
			"location.coordinates": bson.M{
				"$geoWithin": bson.M{
					"$centerSphere": []interface{}{
						[]float64{*filter.LocationLng, *filter.LocationLat},
						float64(r.getLocationRadius(filter.LocationRadius)) / 6378100.0, // Convert meters to earth radius in radians
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
func (r *MongoEventRepository) FindPublicByID(ctx context.Context, id primitive.ObjectID) (*models.Event, error) {
	pipeline := []bson.M{
		// Match public event
		{"$match": bson.M{
			"_id":    id,
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
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in FindPublicByID", log.Err(err))
		}
	}()

	var events []*models.Event
	if err = cursor.All(ctx, &events); err != nil {
		return nil, fmt.Errorf("failed to decode event with sessions: %w", err)
	}

	if len(events) == 0 {
		return nil, errors.ErrEventNotFound
	}

	return events[0], nil
}

// CountByStatus counts events by status
func (r *MongoEventRepository) CountByStatus(ctx context.Context, status string) (int64, error) {
	query := bson.M{
		"status": status,
	}

	count, err := r.collection.CountDocuments(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to count events: %w", err)
	}

	return count, nil
}

// ExistsByID checks if an event exists by ID
func (r *MongoEventRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		return false, fmt.Errorf("failed to check event existence: %w", err)
	}

	return count > 0, nil
}

// Helper methods

// Cursor represents a pagination cursor
type Cursor struct {
	LastID    string    `json:"last_id"`
	Timestamp time.Time `json:"timestamp"`
}

func (r *MongoEventRepository) encodeCursor(cursor *Cursor) string {
	data, err := json.Marshal(cursor)
	if err != nil {
		// This should never happen with well-formed Cursor struct, but handle gracefully
		return ""
	}
	return base64.URLEncoding.EncodeToString(data)
}

func (r *MongoEventRepository) decodeCursor(token string) (*Cursor, error) {
	data, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.NewValidationError("page_token", "invalid base64 encoding")
	}

	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return nil, fmt.Errorf("malformed cursor: %w", err)
	}

	// Validate ObjectID format
	if cursor.LastID != "" {
		if _, err := primitive.ObjectIDFromHex(cursor.LastID); err != nil {
			return nil, fmt.Errorf("invalid cursor ID: %w", err)
		}
	}

	return &cursor, nil
}

func (r *MongoEventRepository) getLocationRadius(radius *int) int {
	if radius != nil {
		return *radius
	}
	// Use config default if available, otherwise fallback to hardcoded default
	if r.paginationConfig != nil && r.paginationConfig.DefaultLocationRadius > 0 {
		return r.paginationConfig.DefaultLocationRadius
	}
	return 1000 // Final fallback default radius in meters
}
