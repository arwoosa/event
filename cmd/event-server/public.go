package main

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"event/internal/service"
	"github.com/arwoosa/vulpes/ezgrpc"
	vulpeslog "github.com/arwoosa/vulpes/log"
)

// publicCmd represents the public command
var publicCmd = &cobra.Command{
	Use:   "public",
	Short: "Start the public API server",
	Long: `Start the public API server that provides read-only access to published events.

This includes search and retrieval operations for public events, typically used by
web applications and mobile apps for end users.

API endpoints:
- /events/* (public read-only API)

This service is intended for public access and provides only published events.`,
	Run: runPublicServer,
}

func runPublicServer(cmd *cobra.Command, args []string) {
	vulpeslog.Info("Starting Event microservice - Public Mode")

	// Register only public services
	service.RegisterPublicServices(GetAppConfig())

	// Run the gRPC + Gateway server
	if err := ezgrpc.RunGrpcGateway(context.Background(), GetAppConfig().Port); err != nil {
		vulpeslog.Fatal("failed to run public server", vulpeslog.Err(err))
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(publicCmd)
}
