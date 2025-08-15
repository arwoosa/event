package service

import (
	"io"

	"google.golang.org/grpc"

	pb "event/api/event"
	"event/conf"
	"event/internal/dao/mongodb"
	"event/internal/dao/repository"

	"github.com/arwoosa/vulpes/ezgrpc"
	"github.com/arwoosa/vulpes/log"
)

// This file registers the services with the Vulpes framework

// RegisterConsoleServices registers only the console (management) services
// It now accepts a slice of io.Closer to register resources for graceful shutdown.
func RegisterConsoleServices(appConfig *conf.AppConfig, closers *[]io.Closer) {
	// Register console gRPC services
	ezgrpc.InjectGrpcService(func(s grpc.ServiceRegistrar) {
		registerConsoleServices(s, appConfig, closers)
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
func registerConsoleServices(s grpc.ServiceRegistrar, appConfig *conf.AppConfig, closers *[]io.Closer) {
	if appConfig == nil {
		log.Warn("Console services initialized with nil config - using mock services")
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc, nil))
		pb.RegisterInternalServiceServer(s, NewInternalServiceServer(nil))
		return
	}

	// Get MongoDB singleton (should be initialized by main)
	mongoClient := mongodb.GetMongoDB()
	if mongoClient == nil {
		log.Warn("Console services initialized without MongoDB - using mock services")
		mockOrderService := NewMockOrderServiceClient(false, nil)
		eventSvc := &EventService{eventRepo: nil, sessionService: nil, orderService: mockOrderService}
		pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc, appConfig.PaginationConfig))
		pb.RegisterInternalServiceServer(s, NewInternalServiceServer(nil))
		return
	}

	// Initialize repositories
	log.Info("Console services initialized with MongoDB connection")
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB, appConfig.PaginationConfig)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize external services
	var orderService OrderServiceClient
	if appConfig.ExternalConfig != nil {
		log.Info("Console services using real order service")
		orderService = NewOrderServiceClient(appConfig.ExternalConfig.OrderService)
		// Add the order service client to the list of closers for graceful shutdown
		*closers = append(*closers, orderService)
	} else {
		log.Warn("Console services initialized without external config - using mock order service")
		orderService = NewMockOrderServiceClient(false, nil)
	}

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo)
	eventSvc := NewEventService(eventRepo, sessionSvc, orderService)

	// Register console services
	pb.RegisterEventServiceServer(s, NewEventServiceServer(eventSvc, appConfig.PaginationConfig))
	pb.RegisterInternalServiceServer(s, NewInternalServiceServer(eventRepo))
}

// registerPublicServices sets up and registers only public (PublicEventService) related gRPC services
func registerPublicServices(s grpc.ServiceRegistrar, appConfig *conf.AppConfig) {
	if appConfig == nil {
		log.Warn("Public services initialized with nil config - using mock services")
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil, paginationConfig: nil}
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Get MongoDB singleton (should be initialized by main)
	mongoClient := mongodb.GetMongoDB()
	if mongoClient == nil {
		log.Warn("Public services initialized without MongoDB - using mock services")
		publicSvc := &PublicService{eventRepo: nil, sessionService: nil, paginationConfig: appConfig.PaginationConfig}
		pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
		return
	}

	// Initialize repositories
	log.Info("Public services initialized with MongoDB connection")
	eventRepo := repository.NewMongoEventRepository(mongoClient, appConfig.MongodbConfig.DB, appConfig.PaginationConfig)
	sessionRepo := repository.NewMongoSessionRepository(mongoClient, appConfig.MongodbConfig.DB)

	// Initialize business services
	sessionSvc := NewSessionService(sessionRepo, eventRepo)
	publicSvc := NewPublicService(eventRepo, sessionSvc, appConfig.PaginationConfig)

	// Register only PublicEventService (public API)
	pb.RegisterPublicEventServiceServer(s, NewPublicEventServiceServer(publicSvc))
}
