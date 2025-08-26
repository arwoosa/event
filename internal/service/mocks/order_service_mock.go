package mocks

import (
	"context"
	"errors"

	"github.com/stretchr/testify/mock"
)

// MockOrderService is a mock implementation of OrderServiceClient
type MockOrderService struct {
	mock.Mock
}

// HasOrders implements OrderServiceClient interface
func (m *MockOrderService) HasOrders(ctx context.Context, eventID string) (bool, error) {
	args := m.Called(ctx, eventID)
	return args.Bool(0), args.Error(1)
}

// Helper methods for common test scenarios

// SetHasOrders sets up the mock to return specific values for HasOrders calls
func (m *MockOrderService) SetHasOrders(eventID string, hasOrders bool, err error) {
	m.On("HasOrders", mock.Anything, eventID).Return(hasOrders, err)
}

// SetHasOrdersForAnyEvent sets up the mock to return specific values for any event ID
func (m *MockOrderService) SetHasOrdersForAnyEvent(hasOrders bool, err error) {
	m.On("HasOrders", mock.Anything, mock.AnythingOfType("string")).Return(hasOrders, err)
}

// SimulateError sets up the mock to return an error for HasOrders calls
func (m *MockOrderService) SimulateError(eventID string, err error) {
	m.On("HasOrders", mock.Anything, eventID).Return(false, err)
}

// SimulateTimeout simulates a timeout error
func (m *MockOrderService) SimulateTimeout(eventID string) {
	m.On("HasOrders", mock.Anything, eventID).Return(false, errors.New("context deadline exceeded"))
}

// SimulateServiceUnavailable simulates service unavailable error
func (m *MockOrderService) SimulateServiceUnavailable(eventID string) {
	m.On("HasOrders", mock.Anything, eventID).Return(false, errors.New("service unavailable"))
}

// Reset clears all expectations and calls
func (m *MockOrderService) Reset() {
	m.ExpectedCalls = nil
	m.Calls = nil
}

// AssertExpectations verifies that all expected calls were made
func (m *MockOrderService) AssertExpectations(t mock.TestingT) bool {
	return m.Mock.AssertExpectations(t)
}

// AssertNotCalledWithEventID asserts that HasOrders was not called with specific event ID
func (m *MockOrderService) AssertNotCalledWithEventID(t mock.TestingT, eventID string) {
	m.AssertNotCalled(t, "HasOrders", mock.Anything, eventID)
}

// AssertCalledWithEventID asserts that HasOrders was called with specific event ID
func (m *MockOrderService) AssertCalledWithEventID(t mock.TestingT, eventID string) {
	m.AssertCalled(t, "HasOrders", mock.Anything, eventID)
}

// AssertNumberOfCalls asserts the number of times HasOrders was called
func (m *MockOrderService) AssertNumberOfCalls(t mock.TestingT, methodName string, expectedCalls int) {
	m.Mock.AssertNumberOfCalls(t, methodName, expectedCalls)
}
