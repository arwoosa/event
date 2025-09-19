package repository

import (
	"context"
	"fmt"

	"github.com/arwoosa/vulpes/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/arwoosa/event/internal/errors"
	"github.com/arwoosa/event/internal/models"
)

// MongoFormRepository implements FormRepository using MongoDB
type MongoFormRepository struct {
	client     *mongo.Client
	database   string
	collection *mongo.Collection
}

// NewMongoFormRepository creates a new MongoDB-based form repository
func NewMongoFormRepository(client *mongo.Client, database string) FormRepository {
	return &MongoFormRepository{
		client:     client,
		database:   database,
		collection: client.Database(database).Collection("forms"),
	}
}

// Create inserts a new form
func (r *MongoFormRepository) Create(ctx context.Context, form *models.EventForm) (*models.EventForm, error) {
	if form.ID.IsZero() {
		form.ID = primitive.NewObjectID()
	}

	if _, err := r.collection.InsertOne(ctx, form); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, errors.ErrFormAlreadyExists
		}
		return nil, fmt.Errorf("failed to create form: %w", err)
	}

	log.Info("Form created successfully",
		log.String("form_id", form.ID.Hex()),
		log.String("event_id", form.EventID.Hex()))

	return form, nil
}

// FindByEventID finds a form by event ID
func (r *MongoFormRepository) FindByEventID(ctx context.Context, eventID primitive.ObjectID) (*models.EventForm, error) {
	var form models.EventForm

	filter := bson.M{"event_id": eventID}

	if err := r.collection.FindOne(ctx, filter).Decode(&form); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrFormNotFound
		}
		return nil, fmt.Errorf("failed to find form by event ID: %w", err)
	}

	return &form, nil
}

// FindByID finds a form by ID
func (r *MongoFormRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.EventForm, error) {
	var form models.EventForm

	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&form)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrFormNotFound
		}
		return nil, fmt.Errorf("failed to find form by ID: %w", err)
	}

	return &form, nil
}

// Update updates an existing form
func (r *MongoFormRepository) Update(ctx context.Context, id primitive.ObjectID, form *models.EventForm) (*models.EventForm, error) {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"schema":     form.Schema,
			"uischema":   form.UISchema,
			"updated_at": form.UpdatedAt,
			"updated_by": form.UpdatedBy,
		},
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var updatedForm models.EventForm

	err := r.collection.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updatedForm)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errors.ErrFormNotFound
		}
		return nil, fmt.Errorf("failed to update form: %w", err)
	}

	log.Info("Form updated successfully",
		log.String("form_id", updatedForm.ID.Hex()),
		log.String("event_id", updatedForm.EventID.Hex()))

	return &updatedForm, nil
}

// Delete deletes a form by ID
func (r *MongoFormRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete form: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.ErrFormNotFound
	}

	log.Info("Form deleted successfully", log.String("form_id", id.Hex()))

	return nil
}

// DeleteByEventID deletes a form by event ID
func (r *MongoFormRepository) DeleteByEventID(ctx context.Context, eventID primitive.ObjectID) error {
	filter := bson.M{"event_id": eventID}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete form by event ID: %w", err)
	}

	if result.DeletedCount == 0 {
		return errors.ErrFormNotFound
	}

	log.Info("Form deleted by event ID successfully", log.String("event_id", eventID.Hex()))

	return nil
}

// ExistsByEventID checks if a form exists for the given event ID
func (r *MongoFormRepository) ExistsByEventID(ctx context.Context, eventID primitive.ObjectID) (bool, error) {
	filter := bson.M{"event_id": eventID}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check form existence by event ID: %w", err)
	}

	return count > 0, nil
}

// ExistsByID checks if a form exists by ID
func (r *MongoFormRepository) ExistsByID(ctx context.Context, id primitive.ObjectID) (bool, error) {
	filter := bson.M{"_id": id}
	count, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to check form existence by ID: %w", err)
	}

	return count > 0, nil
}
