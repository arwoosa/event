package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/arwoosa/vulpes/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// MongoSessionRepository implements SessionRepository using MongoDB
type MongoSessionRepository struct {
	client     *mongo.Client
	database   string
	collection *mongo.Collection
}

// NewMongoSessionRepository creates a new MongoDB-based session repository
func NewMongoSessionRepository(client *mongo.Client, database string) SessionRepository {
	collection := client.Database(database).Collection("sessions")
	return &MongoSessionRepository{
		client:     client,
		database:   database,
		collection: collection,
	}
}

// Create inserts a new session
func (r *MongoSessionRepository) Create(ctx context.Context, session *models.Session) (*models.Session, error) {
	if session.ID.IsZero() {
		session.ID = primitive.NewObjectID()
	}

	now := time.Now()
	session.CreatedAt = now
	session.UpdatedAt = now

	if err := session.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	_, err := r.collection.InsertOne(ctx, session)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	return session, nil
}

// CreateBatch inserts multiple sessions in a single operation
func (r *MongoSessionRepository) CreateBatch(ctx context.Context, sessions []*models.Session) ([]*models.Session, error) {
	if len(sessions) == 0 {
		return []*models.Session{}, nil
	}

	now := time.Now()
	documents := make([]interface{}, len(sessions))

	for i, session := range sessions {
		if session.ID.IsZero() {
			session.ID = primitive.NewObjectID()
		}
		session.CreatedAt = now
		session.UpdatedAt = now

		if err := session.IsValid(); err != nil {
			return nil, fmt.Errorf("invalid session at index %d: %w", i, err)
		}

		documents[i] = session
	}

	// Validate for duplicates
	if err := models.ValidateSessions(sessions); err != nil {
		return nil, fmt.Errorf("session validation failed: %w", err)
	}

	_, err := r.collection.InsertMany(ctx, documents)
	if err != nil {
		return nil, fmt.Errorf("failed to create sessions: %w", err)
	}

	return sessions, nil
}

// FindByID finds a session by its ID
func (r *MongoSessionRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Session, error) {
	var session models.Session
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&session)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}

	return &session, nil
}

// FindByEventID finds all sessions for a specific event
func (r *MongoSessionRepository) FindByEventID(ctx context.Context, eventID primitive.ObjectID) ([]*models.Session, error) {
	opts := options.Find().SetSort(bson.D{{Key: "start_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"event_id": eventID}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in FindByEventID", log.Err(err))
		}
	}()

	var sessions []*models.Session
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	return sessions, nil
}

// FindByEventIDs finds sessions for multiple events
func (r *MongoSessionRepository) FindByEventIDs(ctx context.Context, eventIDs []primitive.ObjectID) (map[primitive.ObjectID][]*models.Session, error) {
	if len(eventIDs) == 0 {
		return make(map[primitive.ObjectID][]*models.Session), nil
	}

	opts := options.Find().SetSort(bson.D{{Key: "event_id", Value: 1}, {Key: "start_time", Value: 1}})
	cursor, err := r.collection.Find(ctx, bson.M{"event_id": bson.M{"$in": eventIDs}}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to find sessions: %w", err)
	}
	defer func() {
		if err := cursor.Close(ctx); err != nil {
			log.Error("Failed to close cursor in FindByEventIDs", log.Err(err))
		}
	}()

	result := make(map[primitive.ObjectID][]*models.Session)
	for _, eventID := range eventIDs {
		result[eventID] = []*models.Session{}
	}

	var sessions []*models.Session
	if err = cursor.All(ctx, &sessions); err != nil {
		return nil, fmt.Errorf("failed to decode sessions: %w", err)
	}

	for _, session := range sessions {
		result[session.EventID] = append(result[session.EventID], session)
	}

	return result, nil
}

// Update updates an existing session
func (r *MongoSessionRepository) Update(ctx context.Context, id primitive.ObjectID, session *models.Session) (*models.Session, error) {

	session.UpdatedAt = time.Now()

	if err := session.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": id}, session)
	if err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	if result.MatchedCount == 0 {
		return nil, errors.ErrSessionNotFound
	}

	return session, nil
}

// Delete removes a session
func (r *MongoSessionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.ErrSessionNotFound
	}

	return nil
}

// DeleteByEventID removes all sessions for an event
func (r *MongoSessionRepository) DeleteByEventID(ctx context.Context, eventID primitive.ObjectID) error {

	_, err := r.collection.DeleteMany(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	return nil
}

// DeleteByEventIDs removes sessions for multiple events
func (r *MongoSessionRepository) DeleteByEventIDs(ctx context.Context, eventIDs []primitive.ObjectID) error {
	if len(eventIDs) == 0 {
		return nil
	}

	_, err := r.collection.DeleteMany(ctx, bson.M{"event_id": bson.M{"$in": eventIDs}})
	if err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	return nil
}

// CountByEventID counts sessions for an event
func (r *MongoSessionRepository) CountByEventID(ctx context.Context, eventID primitive.ObjectID) (int64, error) {

	count, err := r.collection.CountDocuments(ctx, bson.M{"event_id": eventID})
	if err != nil {
		return 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	return count, nil
}

// ExistsByID checks if a session exists by ID
func (r *MongoSessionRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {

	count, err := r.collection.CountDocuments(ctx, bson.M{"_id": id})
	if err != nil {
		return false, fmt.Errorf("failed to check session existence: %w", err)
	}

	return count > 0, nil
}

// BulkUpdateSessions performs bulk operations (create, update, delete) in a single request
func (r *MongoSessionRepository) BulkUpdateSessions(ctx context.Context, creates []*models.Session, updates []*models.Session, deleteIDs []primitive.ObjectID) error {
	if len(creates) == 0 && len(updates) == 0 && len(deleteIDs) == 0 {
		return nil // No operations to perform
	}

	writeModels := make([]mongo.WriteModel, 0, len(creates)+len(updates)+len(deleteIDs))
	now := time.Now()

	// Add delete operations first
	for _, objectID := range deleteIDs {
		deleteModel := mongo.NewDeleteOneModel().SetFilter(bson.M{"_id": objectID})
		writeModels = append(writeModels, deleteModel)
	}

	// Add update operations
	for _, session := range updates {
		if err := session.IsValid(); err != nil {
			return fmt.Errorf("invalid session for update: %w", err)
		}
		session.UpdatedAt = now

		updateModel := mongo.NewUpdateOneModel().
			SetFilter(bson.M{"_id": session.ID}).
			SetUpdate(bson.M{"$set": session})
		writeModels = append(writeModels, updateModel)
	}

	// Add create operations
	for _, session := range creates {
		if err := session.IsValid(); err != nil {
			return fmt.Errorf("invalid session for create: %w", err)
		}
		if session.ID.IsZero() {
			session.ID = primitive.NewObjectID()
		}
		session.CreatedAt = now
		session.UpdatedAt = now

		insertModel := mongo.NewInsertOneModel().SetDocument(session)
		writeModels = append(writeModels, insertModel)
	}

	// Execute bulk write operation
	opts := options.BulkWrite().SetOrdered(false) // Allow parallel execution for better performance
	_, err := r.collection.BulkWrite(ctx, writeModels, opts)
	if err != nil {
		return fmt.Errorf("bulk write operation failed: %w", err)
	}

	return nil
}
