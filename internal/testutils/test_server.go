package testutils

import (
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	consolepb "github.com/arwoosa/event/gen/pb/console"
	publicpb "github.com/arwoosa/event/gen/pb/public"
)

// TestServer wraps gRPC test server with bufconn for in-memory testing
type TestServer struct {
	Server     *grpc.Server
	Listener   *bufconn.Listener
	Connection *grpc.ClientConn

	// Service clients
	EventClient  consolepb.EventServiceClient
	PublicClient publicpb.PublicEventServiceClient
}

// NOTE: SetupTestServer implementation moved to avoid circular dependency
// This will be implemented in service package test files directly

// Cleanup stops the test server and closes connections
func (ts *TestServer) Cleanup(t *testing.T) {
	t.Helper()

	if ts.Connection != nil {
		if err := ts.Connection.Close(); err != nil {
			t.Logf("Failed to close gRPC client connection: %v", err)
		}
	}

	if ts.Server != nil {
		ts.Server.Stop()
	}

	if ts.Listener != nil {
		if err := ts.Listener.Close(); err != nil {
			t.Logf("Failed to close listener: %v", err)
		}
	}
}
