package service

import (
	"fmt"

	"google.golang.org/grpc"

	pb "event/api/event"
	"event/conf"
	"event/internal/dao/mongodb"
	"event/internal/dao/repository"

	"github.com/arwoosa/vulpes/ezgrpc"
)

// This file registers the services with the Vulpes framework

// RegisterConsoleServices registers only the console (management) services
func RegisterConsoleServices(appConfig *conf.AppConfig) {
	// Register console gRPC services
	ezgrpc.InjectGrpcService(func(s grpc.ServiceRegistrar) {
		registerConsoleServices(s, appConfig)
	})

	// Register console gRPC-Gateway handlers
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterEventServiceHandlerFromEndpoint)
}

// RegisterPublicServices registers only the public services
func RegisterPublicServices(appConfig *conf.AppConfig) {
	// Register public gRPC services
	ezgrpc.InjectGrpcService(func(s grpc.ServiceRegistrar) {
		registerPublicServices(s, appConfig)
	})

	// Register public gRPC-Gateway handlers
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterPublicEventServiceHandlerFromEndpoint)
}

// registerConsoleServices sets up and registers only console (EventService) related gRPC services
func registerConsoleServices(s grpc.ServiceRegistrar, appConfig *conf.AppConfig) {
	if appConfig == nil {
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
		pb.RegisterInternalServiceServer(s, NewInternalServiceServer(nil))
		return
	}

	// Get MongoDB singleton (should be initialized by main)
	mongoClient := mongodb.GetMongoDB()
	if mongoClient == nil {
		// MongoDB not available, use mock services
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
		pb.RegisterInternalServiceServer(s, NewInternalServiceServer(nil))
		return
	}

	// Initialize repositories
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize external services
	var orderService OrderServiceClient
	if appConfig.ExternalConfig != nil {
		orderService = NewOrderServiceClient(appConfig.ExternalConfig.OrderService)
		fmt.Println("Using real order service for console services")
	} else {
		fmt.Println("Using mock order service for console services")
		orderService = NewMockOrderServiceClient(false, nil)
	}

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo)
	eventSvc := NewEventService(eventRepo, sessionSvc, orderService)

	// Register console services
	pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
	pb.RegisterInternalServiceServer(s, NewInternalServiceServer(eventRepo))
}

// registerPublicServices sets up and registers only public (PublicEventService) related gRPC services
func registerPublicServices(s grpc.ServiceRegistrar, appConfig *conf.AppConfig) {
	if appConfig == nil {
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil}
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Get MongoDB singleton (should be initialized by main)
	mongoClient := mongodb.GetMongoDB()
	if mongoClient == nil {
		// MongoDB not available, use mock service
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil}
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Initialize repositories
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo)
	publicSvc := NewPublicService(eventRepo, sessionSvc)

	// Register only PublicEventService (public API)
	pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
}
