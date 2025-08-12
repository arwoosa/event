package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

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

	// Register only console services
	service.RegisterConsoleServices(GetAppConfig())

	// Run the gRPC + Gateway server
	if err := ezgrpc.RunGrpcGateway(context.Background(), GetAppConfig().Port); err != nil {
		vulpeslog.Fatal("failed to run console server", vulpeslog.Err(err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(consoleCmd)
}
