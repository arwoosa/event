package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"

	orderpb "event/api/order"
	"event/conf"

	"github.com/arwoosa/vulpes/log"
)

// OrderServiceClientImpl implements OrderServiceClient interface
type OrderServiceClientImpl struct {
	conn    *grpc.ClientConn
	client  orderpb.OrdersAdminServiceClient
	timeout time.Duration
}

// NewOrderServiceClient creates a new order service client.
// It establishes a non-blocking connection to the order service.
func NewOrderServiceClient(config conf.ServiceConfig) OrderServiceClient {
	// Use grpc.NewClient for a non-blocking connection. The connection is managed
	// in the background by gRPC. Errors will be returned on RPC calls if the
	// connection is unavailable, not during initialization.
	conn, err := grpc.NewClient(config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// This error is critical and likely due to a misconfiguration (e.g., invalid endpoint).
		// The application cannot function correctly without a valid client configuration.
		// We are changing the behavior from falling back to a mock to failing fast.
		log.Fatalf("Failed to initialize gRPC client for order service due to configuration error: %v", err)
	}

	client := orderpb.NewOrdersAdminServiceClient(conn)

	return &OrderServiceClientImpl{
		conn:    conn,
		client:  client,
		timeout: config.Timeout,
	}
}

// HasOrders checks if an event has any orders using gRPC
func (c *OrderServiceClientImpl) HasOrders(ctx context.Context, eventID string) (bool, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Create request
	req := &orderpb.IsEventHasOrdersRequest{
		Event: eventID,
	}

	// Call gRPC service
	resp, err := c.client.IsEventHasOpenOrders(timeoutCtx, req)
	if err != nil {
		return false, fmt.Errorf("failed to call order service: %w", err)
	}

	// Parse response data to get has_orders boolean
	if resp.Data == nil {
		return false, fmt.Errorf("order service returned empty data")
	}

	// Convert Any to JSON and parse
	jsonBytes, err := protojson.Marshal(resp.Data)
	if err != nil {
		return false, fmt.Errorf("failed to marshal response data: %w", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return false, fmt.Errorf("failed to unmarshal response data: %w", err)
	}

	hasOrders, ok := data["value"].(bool)
	if !ok {
		return false, fmt.Errorf("order service returned invalid data format: expected bool, got %T", data["value"])
	}

	return hasOrders, nil
}

// Close closes the gRPC connection
func (c *OrderServiceClientImpl) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// MockOrderServiceClient is a mock implementation for testing
type MockOrderServiceClient struct {
	hasOrders        bool
	hasSessionOrders bool
	err              error
}

// NewMockOrderServiceClient creates a new mock order service client
func NewMockOrderServiceClient(hasOrders bool, err error) OrderServiceClient {
	log.Warn("Using mock order service client")
	return &MockOrderServiceClient{
		hasOrders:        hasOrders,
		hasSessionOrders: false, // Default to no session orders
		err:              err,
	}
}

// HasOrders returns the mock result
func (m *MockOrderServiceClient) HasOrders(ctx context.Context, eventID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.hasOrders, nil
}
