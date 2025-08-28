package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/arwoosa/event/conf"
)

func TestOrderServiceClientImpl_HasOrders(t *testing.T) {
	tests := []struct {
		name           string
		eventID        string
		endpoint       string
		timeout        time.Duration
		expectedResult bool
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid event ID with endpoint",
			eventID:        "event_123",
			endpoint:       "localhost:8080",
			timeout:        10 * time.Second,
			expectedResult: false, // Expected to fail due to no running service
			expectError:    true,
			errorContains:  "failed to call order service",
		},
		{
			name:           "Empty event ID",
			eventID:        "",
			endpoint:       "localhost:8080",
			timeout:        5 * time.Second,
			expectedResult: false,
			expectError:    true,
			errorContains:  "failed to call order service",
		},
		{
			name:           "Invalid endpoint",
			eventID:        "event_123",
			endpoint:       "invalid-endpoint:99999",
			timeout:        1 * time.Second,
			expectedResult: false,
			expectError:    true,
			errorContains:  "failed to call order service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client with test configuration
			client := &OrderServiceClientImpl{
				endpoint: tt.endpoint,
				timeout:  tt.timeout,
			}

			ctx := context.Background()

			// Execute
			result, err := client.HasOrders(ctx, tt.eventID)

			// Assert
			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestOrderServiceClientImpl_Close(t *testing.T) {
	client := &OrderServiceClientImpl{
		endpoint: "localhost:8080",
		timeout:  10 * time.Second,
	}

	// Close should not return an error with reflection-based approach
	err := client.Close()
	assert.NoError(t, err)
}

func TestNewOrderServiceClient(t *testing.T) {
	config := conf.ServiceConfig{
		Endpoint: "localhost:8080",
		Timeout:  15 * time.Second,
	}

	client := NewOrderServiceClient(config)

	// Should return OrderServiceClientImpl instance
	impl, ok := client.(*OrderServiceClientImpl)
	require.True(t, ok, "Expected OrderServiceClientImpl instance")

	assert.Equal(t, config.Endpoint, impl.endpoint)
	assert.Equal(t, config.Timeout, impl.timeout)
}

func TestMockOrderServiceClient(t *testing.T) {
	tests := []struct {
		name           string
		hasOrders      bool
		mockError      error
		expectedResult bool
		expectError    bool
	}{
		{
			name:           "Mock returns true",
			hasOrders:      true,
			mockError:      nil,
			expectedResult: true,
			expectError:    false,
		},
		{
			name:           "Mock returns false",
			hasOrders:      false,
			mockError:      nil,
			expectedResult: false,
			expectError:    false,
		},
		{
			name:           "Mock returns error",
			hasOrders:      false,
			mockError:      assert.AnError,
			expectedResult: false,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockOrderServiceClient(tt.hasOrders, tt.mockError)

			ctx := context.Background()
			result, err := mock.HasOrders(ctx, "test_event_id")

			if tt.expectError {
				require.Error(t, err)
				assert.Equal(t, tt.mockError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
