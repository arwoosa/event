package app

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// AllowedHeaders defines the list of headers that are allowed to be passed through the gRPC-Gateway.
var allowedHeadersSet map[string]struct{}

// App manages the gRPC and HTTP servers
type App struct {
	httpServer *http.Server
	gRPCServer *grpc.Server
	port       int
	logger     *zap.Logger
}

// NewApp creates and configures a new application server
func NewApp(port int, logger *zap.Logger, grpcServices []interface{}, unaryInterceptors []grpc.UnaryServerInterceptor, allowHeaders map[string]struct{}) (*App, error) {
	allowedHeadersSet = allowHeaders
	// Create gRPC server with chained interceptors
	s := grpc.NewServer(grpc.ChainUnaryInterceptor(unaryInterceptors...))

	// Register gRPC services
	registerGRPCServices(s, grpcServices...)

	// Register Health Check server
	healthcheck := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthcheck)

	// Create the gRPC-Gateway mux
	gwmux := newGatewayMux()

	// Register gRPC-Gateway handlers
	endpoint := fmt.Sprintf("localhost:%d", port)
	if err := registerGatewayHandlers(context.Background(), gwmux, endpoint); err != nil {
		return nil, err
	}

	// Create a new HTTP serve mux and register the gateway handler
	mux := http.NewServeMux()
	mux.Handle("/", gwmux)

	// Create the main HTTP server
	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: grpcHandlerFunc(s, mux),
	}

	return &App{
		httpServer: httpServer,
		gRPCServer: s,
		port:       port,
		logger:     logger,
	}, nil
}

// Run starts the application server and listens for connections.
func (a *App) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", a.port, err)
	}

	// Start HTTP server in a goroutine
	go func() {
		a.logger.Info("server started", zap.Int("port", a.port))
		if err := a.httpServer.Serve(lis); err != nil && err != http.ErrServerClosed {
			a.logger.Error("HTTP server Serve error", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	a.logger.Info("Shutting down server...")

	// Create a context with a timeout to allow ongoing requests to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // 5 seconds for graceful shutdown
	defer cancel()

	// Gracefully stop gRPC server
	a.gRPCServer.GracefulStop()

	// Shut down HTTP server
	if err := a.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("HTTP server shutdown failed: %w", err)
	}

	a.logger.Info("Server exited gracefully")
	return nil
}
