package middleware

import (
	"context"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// validator is an interface for validating protobuf messages.
type validator interface {
	Validate() error
}

// UnaryValidatorInterceptor is a gRPC unary server interceptor that validates incoming requests.
func UnaryValidatorInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	if v, ok := req.(validator); ok {
		if err := v.Validate(); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}
	return handler(ctx, req)
}

// UnaryPanicInterceptor is a gRPC unary server interceptor that recovers from panics.
var UnaryPanicInterceptor grpc.UnaryServerInterceptor = func(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in %s: %v", info.FullMethod, r)
			err = status.Error(codes.Internal, "Internal server error")
		}
	}()
	return handler(ctx, req)
}
