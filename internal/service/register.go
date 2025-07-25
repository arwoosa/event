package service

import (
	"google.golang.org/grpc"

	"github.com/arwoosa/vulpes/ezgrpc"
)

// This file registers the services with the Vulpes framework
// It will be called automatically when the package is imported

func init() {
	// Register the service initialization function
	ezgrpc.InjectGrpcService(registerEventServices)
}

// registerEventServices sets up and registers all event-related gRPC services
func registerEventServices(s grpc.ServiceRegistrar) {
	// Note: This is a placeholder for when proto files are generated
	// For now, we'll skip the actual registration to avoid configuration conflicts
	
	// TODO: Implement actual gRPC service registration when proto files are generated
	// Example:
	// pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
	// pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
}