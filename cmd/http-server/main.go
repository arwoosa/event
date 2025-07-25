package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	"event/internal/conf"
	"event/internal/dao/mongodb"
	"event/internal/dao/repository"
	"event/internal/service"
)

func main() {
	// Load configuration with explicit config file path
	configFile := "internal/conf/config.yaml"
	if len(os.Args) > 1 {
		for i, arg := range os.Args {
			if arg == "-c" && i+1 < len(os.Args) {
				configFile = os.Args[i+1]
				break
			}
		}
	}
	
	appConfig, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Initialize MongoDB
	mongoClient, cleanup, err := mongodb.NewMongoDB(appConfig.MongodbConfig)
	if err != nil {
		log.Fatalf("failed to connect to MongoDB: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize repository
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize external services (using mock for now)
	orderService := service.NewMockOrderServiceClient(false, nil)

	// Initialize business services
	eventSvc := service.NewEventService(eventRepo, orderService)
	publicSvc := service.NewPublicService(eventRepo)

	// Initialize handlers
	eventHandler := service.NewEventHandler(eventSvc, publicSvc)
	httpHandler := service.NewHTTPHandler(eventHandler)

	// Setup routes
	router := mux.NewRouter()
	httpHandler.SetupRoutes(router)

	// Add CORS middleware for testing
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-User-Id, X-Brand-Id, X-User-Email, X-User-Name, X-User-Avatar")
			
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	})

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"event-microservice"}`))
	}).Methods("GET")

	log.Printf("HTTP server starting on port %d", appConfig.Port)
	log.Printf("Endpoints available:")
	log.Printf("  Health: http://localhost:%d/health", appConfig.Port)
	log.Printf("  Console Events: http://localhost:%d/console/events", appConfig.Port)
	log.Printf("  Public Events: http://localhost:%d/events", appConfig.Port)
	log.Printf("  OpenAPI Spec: Available in openapi.json file")
	
	port := fmt.Sprintf(":%d", appConfig.Port)
	if err := http.ListenAndServe(port, router); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// loadConfig loads configuration from file without using flags
func loadConfig(configFile string) (*conf.AppConfig, error) {
	v := viper.New()
	v.SetConfigFile(configFile)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config conf.AppConfig
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set timezone
	loc, err := time.LoadLocation(config.TimeZone)
	if err != nil {
		return nil, fmt.Errorf("failed to load timezone: %w", err)
	}
	time.Local = loc

	return &config, nil
}