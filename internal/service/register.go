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

// RegisterServices registers both gRPC services and Gateway handlers (legacy)
func RegisterServices(appConfig *conf.AppConfig) {
	// Register gRPC services
	ezgrpc.InjectGrpcService(func(s grpc.ServiceRegistrar) {
		registerEventServices(s, appConfig)
	})

	// Register gRPC-Gateway handlers
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterEventServiceHandlerFromEndpoint)
	ezgrpc.RegisterHandlerFromEndpoint(pb.RegisterPublicEventServiceHandlerFromEndpoint)
}

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

// registerEventServices sets up and registers all event-related gRPC services
func registerEventServices(s grpc.ServiceRegistrar, appConfig *conf.AppConfig) {
	if appConfig == nil {
		// For now, use a mock service if config fails
		// This allows the service to start even without proper config during development
		mockOrderService := NewMockOrderServiceClient(false, nil)

		// Create minimal mock services - these won't work but allow startup
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil}

		// Register with mock services
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Initialize MongoDB
	mongoClient, cleanup, err := mongodb.NewMongoDB(appConfig.MongodbConfig)
	if err != nil {
		// Use mock services if MongoDB connection fails
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil}

		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Note: In production, you'd want proper graceful shutdown handling
	_ = cleanup

	// Initialize repositories
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize external services
	var orderService OrderServiceClient
	if appConfig.ExternalConfig != nil {
		orderService = NewOrderServiceClient(appConfig.ExternalConfig.OrderService)
	} else {
		// Use mock for testing
		orderService = NewMockOrderServiceClient(false, nil)
	}

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo, orderService)
	eventSvc := NewEventService(eventRepo, sessionSvc, orderService)
	publicSvc := NewPublicService(eventRepo, sessionSvc)

	// Register gRPC services with the server
	pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
	pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
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

	// Initialize MongoDB
	mongoClient, cleanup, err := mongodb.NewMongoDB(appConfig.MongodbConfig)
	if err != nil {
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc))
		pb.RegisterInternalServiceServer(s, NewInternalServiceServer(nil))
		return
	}

	_ = cleanup

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
	sessionSvc := NewSessionService(sessionRepo, eventRepo, orderService)
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

	// Initialize MongoDB
	mongoClient, cleanup, err := mongodb.NewMongoDB(appConfig.MongodbConfig)
	if err != nil {
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil}
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	_ = cleanup

	// Initialize repositories
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize external services - public service might need some external dependencies too
	var orderService OrderServiceClient
	if appConfig.ExternalConfig != nil {
		orderService = NewOrderServiceClient(appConfig.ExternalConfig.OrderService)
	} else {
		orderService = NewMockOrderServiceClient(false, nil)
	}

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo, orderService)
	publicSvc := NewPublicService(eventRepo, sessionSvc)

	// Register only PublicEventService (public API)
	pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
}
