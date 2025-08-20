package mongodb

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/arwoosa/event/conf"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/arwoosa/vulpes/log"
)

var (
	// clientInstance holds the singleton MongoDB client
	clientInstance *mongo.Client
	// once ensures the client is initialized only once
	once sync.Once
	// initErr stores any initialization error
	initErr error
	// disconnected is an atomic flag to prevent race conditions during cleanup
	disconnected int64
)

// InitMongoDB initializes the singleton MongoDB client with context-based lifecycle management.
// This function is safe to call multiple times - the client will only be initialized once.
func InitMongoDB(ctx context.Context, cfg *conf.MongodbConfig) (*mongo.Client, error) {
	once.Do(func() {
		// Use separate context for connection with timeout
		connectCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var dsn string
		if cfg.User != "" {
			dsn = fmt.Sprintf("mongodb://%s:%s@%s:%d", cfg.User, cfg.Password, cfg.Host, cfg.Port)
		} else {
			dsn = fmt.Sprintf("mongodb://%s:%d", cfg.Host, cfg.Port)
		}

		// Configure connection pool and timeouts
		clientOptions := options.Client().
			ApplyURI(dsn).
			SetMaxPoolSize(100).
			SetMinPoolSize(10).
			SetMaxConnIdleTime(30 * time.Second).
			SetConnectTimeout(10 * time.Second).
			SetSocketTimeout(30 * time.Second)

		clientInstance, initErr = mongo.Connect(connectCtx, clientOptions)
		if initErr != nil {
			initErr = fmt.Errorf("failed to connect to mongodb: %w", initErr)
			return
		}

		// Check mongodb service working
		if err := clientInstance.Ping(connectCtx, nil); err != nil {
			initErr = fmt.Errorf("failed to ping mongodb: %w", err)
			return
		}

		// Run migrations
		if err := Migrate(clientInstance, cfg); err != nil {
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}

		// Start cleanup goroutine with original context
		go func() {
			<-ctx.Done()
			// Use atomic operation to prevent race condition
			if atomic.CompareAndSwapInt64(&disconnected, 0, 1) {
				if err := clientInstance.Disconnect(context.Background()); err != nil {
					log.Error("failed to disconnect from mongodb", log.Err(err))
				} else {
					log.Info("MongoDB connection closed gracefully")
				}
			}
		}()
	})

	return clientInstance, initErr
}

// GetMongoDB returns the singleton MongoDB client.
// InitMongoDB must be called first, otherwise this will return nil.
// Returns nil if the client has been disconnected.
func GetMongoDB() *mongo.Client {
	if atomic.LoadInt64(&disconnected) == 1 {
		return nil
	}
	return clientInstance
}

// HealthCheck verifies that the MongoDB connection is healthy.
// Returns an error if the client is not initialized or cannot ping the database.
func HealthCheck() error {
	client := GetMongoDB()
	if client == nil {
		return fmt.Errorf("mongodb client not initialized or disconnected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongodb health check failed: %w", err)
	}

	return nil
}

// NewMongoDB is deprecated, use InitMongoDB for initial setup and GetMongoDB for subsequent access.
// This function is kept for backward compatibility and now delegates to the singleton pattern.
func NewMongoDB(ctx context.Context, cfg *conf.MongodbConfig) (*mongo.Client, error) {
	return InitMongoDB(ctx, cfg)
}
