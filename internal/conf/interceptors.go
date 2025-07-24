package conf

import (
	"google.golang.org/grpc"
	"grpc_gateway_framework/internal/middleware"
)

// NewUnaryInterceptors creates and returns a slice of gRPC UnaryServerInterceptor.
func NewUnaryInterceptors() []grpc.UnaryServerInterceptor {
	return []grpc.UnaryServerInterceptor{
		middleware.UnaryPanicInterceptor,
		middleware.UnaryValidatorInterceptor,
	}
}
