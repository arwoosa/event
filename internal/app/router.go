package app

import (
	"context"
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
)

// registerGRPCServices registers all the gRPC services.
func registerGRPCServices(s *grpc.Server, services ...interface{}) {
	//example
	for _, serviceImpl := range services {
		switch srv := serviceImpl.(type) {
		//case *service.CommentsService:
		//	commentv1.RegisterCommentsServer(s, srv)
		}
	}
}

// registerGatewayHandlers registers the gRPC-Gateway handlers for all services.
func registerGatewayHandlers(ctx context.Context, gwmux *runtime.ServeMux, endpoint string) error {
	//opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	//
	//if err := feedbackv1.RegisterFeedbackHandlerFromEndpoint(ctx, gwmux, endpoint, opts); err != nil {
	//	return fmt.Errorf("failed to register feedback handler: %w", err)
	//}

	return nil
}

// newGatewayMux creates and configures a new gRPC-Gateway mux
func newGatewayMux() *runtime.ServeMux {
	return runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(k string) (string, bool) {
			if _, ok := allowedHeadersSet[k]; ok {
				return k, true
			}
			return runtime.DefaultHeaderMatcher(k)
		}),
		runtime.WithErrorHandler(func(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
			st, ok := status.FromError(err)
			if !ok {
				st = status.New(codes.Unknown, "Unknown error")
			}

			httpCode := runtime.HTTPStatusFromCode(st.Code())

			resp := map[string]interface{}{
				"status":  "error",
				"code":    httpCode,
				"message": st.Message(),
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(httpCode)
			if err := json.NewEncoder(w).Encode(resp); err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
		}),
	)
}

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
