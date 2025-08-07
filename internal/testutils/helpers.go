package testutils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"event/internal/models"
)

// AssertError checks that the error matches expected error type and message
func AssertError(t *testing.T, err error, expectedErrType string, expectedMessage string) {
	t.Helper()

	require.Error(t, err, "Expected an error but got nil")

	if businessErr, ok := err.(*models.BusinessError); ok {
		assert.Equal(t, expectedErrType, businessErr.Code, "Error code mismatch")
		if expectedMessage != "" {
			assert.Contains(t, businessErr.Message, expectedMessage, "Error message mismatch")
		}
	} else if validationErr, ok := err.(*models.ValidationError); ok {
		assert.Equal(t, "VALIDATION_ERROR", expectedErrType, "Expected validation error")
		if expectedMessage != "" {
			assert.Contains(t, validationErr.Message, expectedMessage, "Error message mismatch")
		}
	} else {
		assert.Contains(t, err.Error(), expectedMessage, "Error message mismatch")
	}
}

// AssertNoError checks that no error occurred
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err, "Unexpected error occurred")
}

// AssertEventEqual compares two events for equality (ignoring timestamps and IDs if specified)
func AssertEventEqual(t *testing.T, expected, actual *models.Event, ignoreTimestamps bool) {
	t.Helper()

	require.NotNil(t, expected, "Expected event is nil")
	require.NotNil(t, actual, "Actual event is nil")

	assert.Equal(t, expected.Title, actual.Title, "Title mismatch")
	assert.Equal(t, expected.BrandID, actual.BrandID, "BrandID mismatch")
	assert.Equal(t, expected.Summary, actual.Summary, "Summary mismatch")
	assert.Equal(t, expected.Status, actual.Status, "Status mismatch")
	assert.Equal(t, expected.Visibility, actual.Visibility, "Visibility mismatch")
	assert.Equal(t, expected.CoverImageURL, actual.CoverImageURL, "CoverImageURL mismatch")

	// Location comparison
	assert.Equal(t, expected.Location, actual.Location, "Location mismatch")

	// Detail comparison
	assert.Equal(t, expected.Detail, actual.Detail, "Detail mismatch")

	// FAQ comparison
	assert.Equal(t, expected.FAQ, actual.FAQ, "FAQ mismatch")

	if !ignoreTimestamps {
		assert.Equal(t, expected.CreatedAt.Unix(), actual.CreatedAt.Unix(), "CreatedAt mismatch")
		assert.Equal(t, expected.UpdatedAt.Unix(), actual.UpdatedAt.Unix(), "UpdatedAt mismatch")
		assert.Equal(t, expected.CreatedBy, actual.CreatedBy, "CreatedBy mismatch")
		assert.Equal(t, expected.UpdatedBy, actual.UpdatedBy, "UpdatedBy mismatch")
	}
}

// AssertSessionEqual compares two sessions for equality
func AssertSessionEqual(t *testing.T, expected, actual *models.Session, ignoreTimestamps bool) {
	t.Helper()

	require.NotNil(t, expected, "Expected session is nil")
	require.NotNil(t, actual, "Actual session is nil")

	assert.Equal(t, expected.EventID, actual.EventID, "EventID mismatch")
	assert.Equal(t, expected.StartTime.Unix(), actual.StartTime.Unix(), "StartTime mismatch")
	assert.Equal(t, expected.EndTime.Unix(), actual.EndTime.Unix(), "EndTime mismatch")

	if !ignoreTimestamps {
		assert.Equal(t, expected.CreatedAt.Unix(), actual.CreatedAt.Unix(), "CreatedAt mismatch")
		assert.Equal(t, expected.UpdatedAt.Unix(), actual.UpdatedAt.Unix(), "UpdatedAt mismatch")
	}
}

// AssertSessionsEqual compares two session slices for equality
func AssertSessionsEqual(t *testing.T, expected, actual []*models.Session, ignoreTimestamps bool) {
	t.Helper()

	require.Equal(t, len(expected), len(actual), "Session count mismatch")

	for i, expectedSession := range expected {
		AssertSessionEqual(t, expectedSession, actual[i], ignoreTimestamps)
	}
}

// AssertValidObjectID checks that a string is a valid MongoDB ObjectID
func AssertValidObjectID(t *testing.T, id string) {
	t.Helper()

	_, err := primitive.ObjectIDFromHex(id)
	require.NoError(t, err, "Invalid ObjectID: %s", id)
}

// WaitForCondition waits for a condition to be true within a timeout
func WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("Condition not met within timeout: %s", message)
}

// CreateTestContext creates a context with timeout for testing
func CreateTestContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// SetupTestData creates a complete test scenario with events and sessions
func SetupTestData(ctx context.Context, t *testing.T, mongoContainer *MongoContainer) (*models.Event, []*models.Session) {
	t.Helper()

	// Create test event
	event := TestEvent()
	eventResult, err := mongoContainer.GetEventCollection().InsertOne(ctx, event)
	require.NoError(t, err, "Failed to insert test event")

	event.ID = eventResult.InsertedID.(primitive.ObjectID)

	// Create test sessions for the event
	sessions := TestSessionsForEvent(event.ID, 3)

	// Insert sessions
	sessionDocs := make([]interface{}, len(sessions))
	for i, session := range sessions {
		sessionDocs[i] = session
	}

	sessionResults, err := mongoContainer.GetSessionCollection().InsertMany(ctx, sessionDocs)
	require.NoError(t, err, "Failed to insert test sessions")

	// Update session IDs
	for i, id := range sessionResults.InsertedIDs {
		sessions[i].ID = id.(primitive.ObjectID)
	}

	return event, sessions
}

// CleanupTestData removes all test data from collections
func CleanupTestData(ctx context.Context, t *testing.T, mongoContainer *MongoContainer) {
	t.Helper()
	mongoContainer.CleanCollections(ctx, t)
}

// StringPtr returns a pointer to a string (helper for optional fields)
func StringPtr(s string) *string {
	return &s
}

// Int32Ptr returns a pointer to an int32 (helper for optional fields)
func Int32Ptr(i int32) *int32 {
	return &i
}

// Float64Ptr returns a pointer to a float64 (helper for optional fields)
func Float64Ptr(f float64) *float64 {
	return &f
}

// IntPtr returns a pointer to an int (helper for optional fields)
func IntPtr(i int) *int {
	return &i
}

// TimePtr returns a pointer to a time.Time (helper for optional fields)
func TimePtr(t time.Time) *time.Time {
	return &t
}
