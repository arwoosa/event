package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/arwoosa/event/internal/models"
)

// MongoContainer wraps testcontainers MongoDB container
type MongoContainer struct {
	Container testcontainers.Container
	URI       string
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupMongoContainer creates and starts a MongoDB test container
func SetupMongoContainer(ctx context.Context, t *testing.T) *MongoContainer {
	t.Helper()

	// Create MongoDB container
	mongodbContainer, err := mongodb.Run(ctx, "mongo:7",
		mongodb.WithUsername("testuser"),
		mongodb.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Waiting for connections").
				WithOccurrence(1).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start MongoDB container: %v", err)
	}

	// Get connection string
	uri, err := mongodbContainer.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("Failed to get MongoDB URI: %v", err)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Test connection
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	database := client.Database("testdb")

	return &MongoContainer{
		Container: mongodbContainer,
		URI:       uri,
		Client:    client,
		Database:  database,
	}
}

// Cleanup closes the MongoDB connection and terminates the container
func (mc *MongoContainer) Cleanup(ctx context.Context, t *testing.T) {
	t.Helper()

	if mc.Client != nil {
		if err := mc.Client.Disconnect(ctx); err != nil {
			t.Logf("Failed to disconnect MongoDB client: %v", err)
		}
	}

	if mc.Container != nil {
		if err := mc.Container.Terminate(ctx); err != nil {
			t.Logf("Failed to terminate MongoDB container: %v", err)
		}
	}
}

// CleanCollections drops all collections in the test database
func (mc *MongoContainer) CleanCollections(ctx context.Context, t *testing.T) {
	t.Helper()

	collections, err := mc.Database.ListCollectionNames(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list collections: %v", err)
	}

	for _, collection := range collections {
		if err := mc.Database.Collection(collection).Drop(ctx); err != nil {
			t.Fatalf("Failed to drop collection %s: %v", collection, err)
		}
	}
}

// CreateIndexes creates the necessary indexes for testing
func (mc *MongoContainer) CreateIndexes(ctx context.Context, t *testing.T) {
	t.Helper()

	// Event collection indexes
	eventCollection := mc.Database.Collection("events")

	// Merchant ID index
	_, err := eventCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"merchant_id": 1},
	})
	if err != nil {
		t.Fatalf("Failed to create merchant_id index: %v", err)
	}

	// Status index
	_, err = eventCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"status": 1},
	})
	if err != nil {
		t.Fatalf("Failed to create status index: %v", err)
	}

	// Geospatial index for location coordinates
	_, err = eventCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"location.coordinates": "2dsphere"},
	})
	if err != nil {
		t.Fatalf("Failed to create geospatial index: %v", err)
	}

	// Compound index for merchant_id and status
	_, err = eventCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"merchant_id": 1,
			"status":      1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create compound merchant_id+status index: %v", err)
	}

	// Session collection indexes
	sessionCollection := mc.Database.Collection("sessions")

	// Event ID and Merchant ID compound index
	_, err = sessionCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{
			"event_id":    1,
			"merchant_id": 1,
		},
	})
	if err != nil {
		t.Fatalf("Failed to create event_id+merchant_id index: %v", err)
	}

	// Start time index for time-based queries
	_, err = sessionCollection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: map[string]interface{}{"start_time": 1},
	})
	if err != nil {
		t.Fatalf("Failed to create start_time index: %v", err)
	}
}

// GetEventCollection returns the events collection
func (mc *MongoContainer) GetEventCollection() *mongo.Collection {
	return mc.Database.Collection("events")
}

// GetSessionCollection returns the sessions collection
func (mc *MongoContainer) GetSessionCollection() *mongo.Collection {
	return mc.Database.Collection("sessions")
}

// InsertTestEvent inserts a test event and returns its ID
func (mc *MongoContainer) InsertTestEvent(ctx context.Context, t *testing.T, event *models.Event) string {
	t.Helper()

	if event == nil {
		event = TestEvent() // Create default test event
	}

	result, err := mc.GetEventCollection().InsertOne(ctx, event)
	if err != nil {
		t.Fatalf("Failed to insert test event: %v", err)
	}

	return result.InsertedID.(primitive.ObjectID).Hex()
}

// InsertTestSession inserts a test session and returns its ID
func (mc *MongoContainer) InsertTestSession(ctx context.Context, t *testing.T, session *models.Session) string {
	t.Helper()

	if session == nil {
		session = TestSession() // Create default test session
	}

	result, err := mc.GetSessionCollection().InsertOne(ctx, session)
	if err != nil {
		t.Fatalf("Failed to insert test session: %v", err)
	}

	return result.InsertedID.(primitive.ObjectID).Hex()
}

// CountEvents returns the number of events in the collection
func (mc *MongoContainer) CountEvents(ctx context.Context, t *testing.T) int64 {
	t.Helper()

	count, err := mc.GetEventCollection().CountDocuments(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to count events: %v", err)
	}

	return count
}

// CountSessions returns the number of sessions in the collection
func (mc *MongoContainer) CountSessions(ctx context.Context, t *testing.T) int64 {
	t.Helper()

	count, err := mc.GetSessionCollection().CountDocuments(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}

	return count
}
