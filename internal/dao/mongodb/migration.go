package mongodb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"event/internal/conf"
	"log"
)

// Migration defines the structure for a collection migration
type Migration struct {
	Collection string
	Indexes    []mongo.IndexModel
}

// collection相關的index
var migrations = []Migration{
	{
		Collection: "events",
		Indexes: []mongo.IndexModel{
			// Basic query index for console API
			{
				Keys: bson.D{
					{Key: "brand_id", Value: 1},
					{Key: "status", Value: 1},
					{Key: "visibility", Value: 1},
				},
			},
			// Time range query index for sessions
			{
				Keys: bson.D{
					{Key: "brand_id", Value: 1},
					{Key: "sessions.start_time", Value: 1},
				},
			},
			// Geospatial index for location-based queries
			{
				Keys: bson.D{{Key: "location.coordinates", Value: "2dsphere"}},
			},
			// Text search index for title
			{
				Keys: bson.D{{Key: "title", Value: "text"}},
			},
			// Sorting index for console list view
			{
				Keys: bson.D{
					{Key: "brand_id", Value: 1},
					{Key: "created_at", Value: -1},
				},
			},
			// Sorting index for updated time
			{
				Keys: bson.D{
					{Key: "brand_id", Value: 1},
					{Key: "updated_at", Value: -1},
				},
			},
			// Public API query index
			{
				Keys: bson.D{
					{Key: "status", Value: 1},
					{Key: "visibility", Value: 1},
					{Key: "sessions.start_time", Value: 1},
				},
			},
			// Performance index for session time sorting
			{
				Keys: bson.D{
					{Key: "sessions.start_time", Value: 1},
				},
			},
		},
	},
}

// Migrate runs all the defined migrations.
func Migrate(client *mongo.Client, cfg *conf.MongodbConfig) error {
	log.Println("Running MongoDB migrations...")
	db := client.Database(cfg.DB)

	for _, m := range migrations {
		coll := db.Collection(m.Collection)

		if len(m.Indexes) > 0 {
			_, err := coll.Indexes().CreateMany(context.Background(), m.Indexes)
			if err != nil {
				return fmt.Errorf("failed to create indexes for collection '%s': %w", m.Collection, err)
			}
		}
	}

	return nil
}
