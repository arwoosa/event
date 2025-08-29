package service

import (
	"context"
	"fmt"
	"time"

	"github.com/arwoosa/vulpes/ezgrpc"
	"github.com/arwoosa/vulpes/log"

	"github.com/arwoosa/event/conf"
)

// OrderServiceClientImpl implements OrderServiceClient interface
type OrderServiceClientImpl struct {
	endpoint string
	timeout  time.Duration
}

// IsEventHasOrdersRequest represents the request to check if event has orders
type IsEventHasOrdersRequest struct {
	Event string `json:"event"`
}

// IsEventHasOrdersResponse represents the response containing order status
type IsEventHasOrdersResponse struct {
	Data *OrderData `json:"data,omitempty"`
}

// OrderData represents the order data in the response
type OrderData struct {
	Value bool `json:"value"`
}

// NewOrderServiceClient creates a new order service client
func NewOrderServiceClient(config conf.ServiceConfig) OrderServiceClient {
	// No need to pre-establish connection with reflection-based approach
	return &OrderServiceClientImpl{
		endpoint: config.Endpoint,
		timeout:  config.Timeout,
	}
}

// HasOrders checks if an event has any orders using reflection-based gRPC
func (c *OrderServiceClientImpl) HasOrders(ctx context.Context, eventID string) (bool, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Create request
	req := &IsEventHasOrdersRequest{
		Event: eventID,
	}

	// Call gRPC service using reflection
	resp, err := ezgrpc.Invoke[*IsEventHasOrdersRequest, *IsEventHasOrdersResponse](
		timeoutCtx,
		c.endpoint,                 // service endpoint
		"api.orders.OrdersAdminService", // service name
		"IsEventHasOpenOrders",     // method name
		req,
	)
	if err != nil {
		return false, fmt.Errorf("failed to call order service: %w", err)
	}

	// Parse response data
	if resp == nil || resp.Data == nil {
		return false, nil
	}

	return resp.Data.Value, nil
}

// Close closes the gRPC connection (no-op for reflection-based client)
func (c *OrderServiceClientImpl) Close() error {
	// No persistent connection to close with reflection-based approach
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
