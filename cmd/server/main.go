package main

import (
	"context"
	"log"
	"os"

	"event/internal/conf"
	"github.com/arwoosa/vulpes/ezgrpc"
	vulpeslog "github.com/arwoosa/vulpes/log"
	
	// Import service package
	"event/internal/service"
)

func main() {
	// Load configuration
	appConfig, err := conf.NewConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Initialize vulpes logger
	isDev := appConfig.Mode == "dev"
	vulpeslog.SetConfig(
		vulpeslog.WithDev(isDev),
		vulpeslog.WithLevel(appConfig.LogConfig.Level),
	)

	vulpeslog.Info("Starting Event microservice")

	// Register services directly
	service.RegisterServices(appConfig)

	// Run the complete gRPC + Gateway server using vulpes
	if err := ezgrpc.RunGrpcGateway(context.Background(), appConfig.Port); err != nil {
		vulpeslog.Fatal("failed to run server", vulpeslog.Err(err))
		os.Exit(1)
	}
}
