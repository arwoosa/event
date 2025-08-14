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
)

// OrderServiceClientImpl implements OrderServiceClient interface
type OrderServiceClientImpl struct {
	conn    *grpc.ClientConn
	client  orderpb.OrdersAdminServiceClient
	timeout time.Duration
}

// NewOrderServiceClient creates a new order service client
func NewOrderServiceClient(config conf.ServiceConfig) OrderServiceClient {
	conn, err := grpc.NewClient(config.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		// For development, we'll use a mock client if connection fails
		return NewMockOrderServiceClient(false, fmt.Errorf("failed to connect to order service: %w", err))
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
		return false, nil
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
		return false, nil
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

// HasOrdersForSession returns the mock result for session orders
func (m *MockOrderServiceClient) HasOrdersForSession(ctx context.Context, sessionID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.hasSessionOrders, nil
}
