package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"event/internal/dao/mongodb"
	"event/internal/service"

	"github.com/arwoosa/vulpes/ezgrpc"
	vulpeslog "github.com/arwoosa/vulpes/log"
)

// consoleCmd represents the console command
var consoleCmd = &cobra.Command{
	Use:   "console",
	Short: "Start the console (management) API server",
	Long: `Start the console API server that provides management endpoints for events.

This includes full CRUD operations for events and sessions, typically used by
internal tools and admin interfaces.

API endpoints:
- /console/events/* (full management API)

This service is intended for internal use and requires proper authentication.`,
	Run: runConsoleServer,
}

func runConsoleServer(cmd *cobra.Command, args []string) {
	vulpeslog.Info("Starting Event microservice - Console Mode")

	ctx := context.Background()
	appConfig := GetAppConfig()

	// Initialize MongoDB singleton first - Console service requires database
	if _, err := mongodb.InitMongoDB(ctx, appConfig.MongodbConfig); err != nil {
		vulpeslog.Error("Failed to initialize MongoDB", vulpeslog.Err(err))
		vulpeslog.Fatal("Console service requires MongoDB connection - cannot start without database")
	}

	// Register only console services
	service.RegisterConsoleServices(appConfig)

	// Run the gRPC + Gateway server
	if err := ezgrpc.RunGrpcGateway(ctx, appConfig.Port); err != nil {
		vulpeslog.Fatal("failed to run console server", vulpeslog.Err(err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
