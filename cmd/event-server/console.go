package main

import (
	"context"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/arwoosa/event/internal/dao/mongodb"
	"github.com/arwoosa/event/internal/service"

	"github.com/arwoosa/vulpes/ezgrpc"
	vulpeslog "github.com/arwoosa/vulpes/log"
	"github.com/arwoosa/vulpes/relation"
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

	// Initialize MongoDB singleton first - Console service requires database
	if _, err := mongodb.InitMongoDB(ctx, appConfig.MongodbConfig); err != nil {
		vulpeslog.Error("Failed to initialize MongoDB", vulpeslog.Err(err))
		vulpeslog.Fatal("Console service requires MongoDB connection - cannot start without database")
	}

	// Initialize Keto relation client
	if appConfig.KetoConfig != nil {
		vulpeslog.Info("Initializing Keto relation client",
			vulpeslog.String("write_addr", appConfig.KetoConfig.WriteAddr),
			vulpeslog.String("read_addr", appConfig.KetoConfig.ReadAddr))

		relation.Initialize(
			relation.WithWriteAddr(appConfig.KetoConfig.WriteAddr),
			relation.WithReadAddr(appConfig.KetoConfig.ReadAddr),
		)

		// Ensure relation connection is closed when server shuts down
		defer relation.Close()

		vulpeslog.Info("Keto relation client initialized successfully")
	} else {
		vulpeslog.Warn("Keto configuration not found - authorization features may not work")
	}

	// Register only console services
	service.RegisterConsoleServices(appConfig)

	ezgrpc.SetServeMuxOpts(
		ezgrpc.DefaultHeaderMatcher,
		ezgrpc.OutgoingHeaderMatcher,
	)

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
		// The context is canceled, the server will shut down.
		// We can add a timeout for graceful shutdown if ezgrpc supports it.
		// For now, relying on context cancellation.
	case err := <-errChan:
		vulpeslog.Fatal("failed to run console server", vulpeslog.Err(err))
	}

	vulpeslog.Info("Server shut down gracefully")
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
