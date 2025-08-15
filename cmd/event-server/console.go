package main

import (
	"context"
	"io"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	appConfig := GetAppConfig()

	// A slice to hold all resources that need to be closed gracefully.
	var closers []io.Closer

	// Initialize MongoDB singleton first - Console service requires database
	// Note: We could also add the mongo client to the closers list if its client has a Close method.
	if _, err := mongodb.InitMongoDB(ctx, appConfig.MongodbConfig); err != nil {
		vulpeslog.Error("Failed to initialize MongoDB", vulpeslog.Err(err))
		vulpeslog.Fatal("Console service requires MongoDB connection - cannot start without database")
	}

	// Register only console services and collect closers
	service.RegisterConsoleServices(appConfig, &closers)

	// Channel to listen for server errors
	errChan := make(chan error, 1)

	// Run the gRPC + Gateway server in a goroutine
	go func() {
		if err := ezgrpc.RunGrpcGateway(ctx, appConfig.Port); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case <-ctx.Done():
		vulpeslog.Info("Shutdown signal received, shutting down server gracefully...")
		// Close all registered resources.
		for _, closer := range closers {
			if err := closer.Close(); err != nil {
				vulpeslog.Error("error closing resource during shutdown", vulpeslog.Err(err))
			}
		}
	case err := <-errChan:
		vulpeslog.Fatal("failed to run console server", vulpeslog.Err(err))
	}

	vulpeslog.Info("Server shut down gracefully")
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
