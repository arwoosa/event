package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"event/internal/conf"
)

// OrderServiceClientImpl implements OrderServiceClient interface
type OrderServiceClientImpl struct {
	endpoint string
	timeout  time.Duration
	client   *http.Client
}

// NewOrderServiceClient creates a new order service client
func NewOrderServiceClient(config conf.ServiceConfig) OrderServiceClient {
	return &OrderServiceClientImpl{
		endpoint: config.Endpoint,
		timeout:  config.Timeout,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// OrderExistsResponse represents the response from order service
type OrderExistsResponse struct {
	HasOrders bool `json:"has_orders"`
}

// HasOrders checks if an event has any orders
func (c *OrderServiceClientImpl) HasOrders(ctx context.Context, eventID string) (bool, error) {
	url := fmt.Sprintf("http://%s/orders/events/%s/exists", c.endpoint, eventID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call order service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// If event not found in order service, assume no orders
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("order service returned status %d", resp.StatusCode)
	}

	var response OrderExistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.HasOrders, nil
}

// MockOrderServiceClient is a mock implementation for testing
type MockOrderServiceClient struct {
	hasOrders bool
	err       error
}

// NewMockOrderServiceClient creates a new mock order service client
func NewMockOrderServiceClient(hasOrders bool, err error) OrderServiceClient {
	return &MockOrderServiceClient{
		hasOrders: hasOrders,
		err:       err,
	}
}

// HasOrders returns the mock result
func (m *MockOrderServiceClient) HasOrders(ctx context.Context, eventID string) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.hasOrders, nil
}
