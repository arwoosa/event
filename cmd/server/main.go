package main

import (
	"log"
	"os"

	"grpc_gateway_framework/internal/conf"
)

func main() {
	// Load configuration
	appConfig, err := conf.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Initialize application
	app, cleanup, err := initApp(appConfig)
	if err != nil {
		log.Fatalf("failed to init app: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Run application
	if err := app.Run(); err != nil {
		log.Fatalf("failed to run app: %v", err)
		os.Exit(1)
	}
}
