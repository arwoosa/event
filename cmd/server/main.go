package main

import (
	"context"
	"log"
	"os"

	"event/internal/conf"
	vulpeslog "github.com/arwoosa/vulpes/log"
	"github.com/arwoosa/vulpes/ezgrpc"
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

	// Register services (will be done in service init() functions)
	// Services will use ezgrpc.InjectGrpcService() and ezgrpc.RegisterHandlerFromEndpoint()

	// Run the complete gRPC + Gateway server using vulpes
	if err := ezgrpc.RunGrpcGateway(context.Background(), appConfig.Port); err != nil {
		vulpeslog.Fatal("failed to run server", vulpeslog.Err(err))
		os.Exit(1)
	}
}
