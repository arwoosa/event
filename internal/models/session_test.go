package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestSession_IsDuplicateOf(t *testing.T) {
	baseTime := time.Now()
	startTime := baseTime.Add(time.Hour)
	endTime := baseTime.Add(time.Hour * 2)

	tests := []struct {
		name     string
		session1 *Session
		session2 *Session
		expected bool
	}{
		{
			name: "Identical sessions",
			session1: &Session{
				StartTime: startTime,
				EndTime:   endTime,
			},
			session2: &Session{
				StartTime: startTime,
				EndTime:   endTime,
			},
			expected: true,
		},
		{
			name: "Different start times",
			session1: &Session{
				StartTime: startTime,
				EndTime:   endTime,
			},
			session2: &Session{
				StartTime: startTime.Add(time.Minute),
				EndTime:   endTime,
			},
			expected: false,
		},
		{
			name: "Different end times",
			session1: &Session{
				StartTime: startTime,
				EndTime:   endTime,
			},
			session2: &Session{
				StartTime: startTime,
				EndTime:   endTime.Add(time.Minute),
			},
			expected: false,
		},
		{
			name: "Completely different times",
			session1: &Session{
				StartTime: startTime,
				EndTime:   endTime,
			},
			session2: &Session{
				StartTime: startTime.Add(time.Hour * 3),
				EndTime:   endTime.Add(time.Hour * 3),
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.session1.IsDuplicateOf(tt.session2)
			assert.Equal(t, tt.expected, result)

			// Test symmetry
			result2 := tt.session2.IsDuplicateOf(tt.session1)
			assert.Equal(t, tt.expected, result2)
		})
	}
}

func TestSession_IsValid(t *testing.T) {
	validEventID := primitive.NewObjectID()
	validStartTime := time.Now().Add(time.Hour)
	validEndTime := validStartTime.Add(time.Hour)

	tests := []struct {
		name        string
		session     *Session
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid session",
			session: &Session{
				EventID:   validEventID,
				StartTime: validStartTime,
				EndTime:   validEndTime,
			},
			expectError: false,
		},
		{
			name: "Missing event ID",
			session: &Session{
				StartTime: validStartTime,
				EndTime:   validEndTime,
			},
			expectError: true,
			errorMsg:    "event_id is required",
		},
		{
			name: "Missing start time",
			session: &Session{
				EventID: validEventID,
				// StartTime is zero value
				EndTime: validEndTime,
			},
			expectError: true,
			errorMsg:    "start_time is required",
		},
		{
			name: "Missing start time",
			session: &Session{
				EventID: validEventID,
				EndTime: validEndTime,
			},
			expectError: true,
			errorMsg:    "start_time is required",
		},
		{
			name: "Missing end time",
			session: &Session{
				EventID:   validEventID,
				StartTime: validStartTime,
			},
			expectError: true,
			errorMsg:    "end_time is required",
		},
		{
			name: "Start time equals end time",
			session: &Session{
				EventID:   validEventID,
				StartTime: validStartTime,
				EndTime:   validStartTime, // Same as start time
			},
			expectError: true,
			errorMsg:    "start_time must be before end_time",
		},
		{
			name: "Start time after end time",
			session: &Session{
				EventID:   validEventID,
				StartTime: validEndTime,   // After end time
				EndTime:   validStartTime, // Before start time
			},
			expectError: true,
			errorMsg:    "start_time must be before end_time",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.IsValid()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSession_Duration(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name             string
		startTime        time.Time
		endTime          time.Time
		expectedDuration time.Duration
	}{
		{
			name:             "1 hour duration",
			startTime:        baseTime,
			endTime:          baseTime.Add(time.Hour),
			expectedDuration: time.Hour,
		},
		{
			name:             "30 minutes duration",
			startTime:        baseTime,
			endTime:          baseTime.Add(30 * time.Minute),
			expectedDuration: 30 * time.Minute,
		},
		{
			name:             "2.5 hours duration",
			startTime:        baseTime,
			endTime:          baseTime.Add(2*time.Hour + 30*time.Minute),
			expectedDuration: 2*time.Hour + 30*time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := &Session{
				StartTime: tt.startTime,
				EndTime:   tt.endTime,
			}

			duration := session.Duration()
			assert.Equal(t, tt.expectedDuration, duration)
		})
	}
}

func TestValidateSessions(t *testing.T) {
	baseTime := time.Now()
	validEventID := primitive.NewObjectID()

	createValidSession := func(startOffset, endOffset time.Duration) *Session {
		return &Session{
			ID:        primitive.NewObjectID(),
			EventID:   validEventID,
			StartTime: baseTime.Add(startOffset),
			EndTime:   baseTime.Add(endOffset),
		}
	}

	tests := []struct {
		name        string
		sessions    []*Session
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Empty sessions",
			sessions:    []*Session{},
			expectError: false,
		},
		{
			name: "Single valid session",
			sessions: []*Session{
				createValidSession(time.Hour, time.Hour*2),
			},
			expectError: false,
		},
		{
			name: "Multiple valid sessions",
			sessions: []*Session{
				createValidSession(time.Hour, time.Hour*2),
				createValidSession(time.Hour*3, time.Hour*4),
				createValidSession(time.Hour*5, time.Hour*6),
			},
			expectError: false,
		},
		{
			name: "Duplicate sessions - identical times",
			sessions: []*Session{
				createValidSession(time.Hour, time.Hour*2),
				createValidSession(time.Hour, time.Hour*2), // Duplicate
			},
			expectError: true,
			errorMsg:    "sessions 0 and 1 have identical start and end times",
		},
		{
			name: "Invalid session in collection",
			sessions: []*Session{
				createValidSession(time.Hour, time.Hour*2),
				{
					ID:        primitive.NewObjectID(),
					EventID:   validEventID,
					StartTime: baseTime.Add(time.Hour * 3),
					EndTime:   baseTime.Add(time.Hour * 2), // End before start
				},
			},
			expectError: true,
			errorMsg:    "start_time must be before end_time",
		},
		{
			name: "Multiple duplicates",
			sessions: []*Session{
				createValidSession(time.Hour, time.Hour*2),
				createValidSession(time.Hour*3, time.Hour*4),
				createValidSession(time.Hour, time.Hour*2), // Duplicate of first
			},
			expectError: true,
			errorMsg:    "sessions 0 and 2 have identical start and end times",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSessions(tt.sessions)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetEarliestStartTime(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name           string
		sessions       []*Session
		expectedResult *time.Time
	}{
		{
			name:           "Empty sessions",
			sessions:       []*Session{},
			expectedResult: nil,
		},
		{
			name: "Single session",
			sessions: []*Session{
				{StartTime: baseTime.Add(time.Hour)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour)}[0],
		},
		{
			name: "Multiple sessions - ascending order",
			sessions: []*Session{
				{StartTime: baseTime.Add(time.Hour)},
				{StartTime: baseTime.Add(time.Hour * 2)},
				{StartTime: baseTime.Add(time.Hour * 3)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour)}[0],
		},
		{
			name: "Multiple sessions - descending order",
			sessions: []*Session{
				{StartTime: baseTime.Add(time.Hour * 3)},
				{StartTime: baseTime.Add(time.Hour * 2)},
				{StartTime: baseTime.Add(time.Hour)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour)}[0],
		},
		{
			name: "Multiple sessions - random order",
			sessions: []*Session{
				{StartTime: baseTime.Add(time.Hour * 2)},
				{StartTime: baseTime.Add(time.Hour * 4)},
				{StartTime: baseTime.Add(time.Hour)}, // Earliest
				{StartTime: baseTime.Add(time.Hour * 3)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour)}[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEarliestStartTime(tt.sessions)

			if tt.expectedResult == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.True(t, result.Equal(*tt.expectedResult))
			}
		})
	}
}

func TestGetLatestEndTime(t *testing.T) {
	baseTime := time.Now()

	tests := []struct {
		name           string
		sessions       []*Session
		expectedResult *time.Time
	}{
		{
			name:           "Empty sessions",
			sessions:       []*Session{},
			expectedResult: nil,
		},
		{
			name: "Single session",
			sessions: []*Session{
				{EndTime: baseTime.Add(time.Hour * 2)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour * 2)}[0],
		},
		{
			name: "Multiple sessions - ascending order",
			sessions: []*Session{
				{EndTime: baseTime.Add(time.Hour * 2)},
				{EndTime: baseTime.Add(time.Hour * 3)},
				{EndTime: baseTime.Add(time.Hour * 4)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour * 4)}[0],
		},
		{
			name: "Multiple sessions - descending order",
			sessions: []*Session{
				{EndTime: baseTime.Add(time.Hour * 4)},
				{EndTime: baseTime.Add(time.Hour * 3)},
				{EndTime: baseTime.Add(time.Hour * 2)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour * 4)}[0],
		},
		{
			name: "Multiple sessions - random order",
			sessions: []*Session{
				{EndTime: baseTime.Add(time.Hour * 3)},
				{EndTime: baseTime.Add(time.Hour * 2)},
				{EndTime: baseTime.Add(time.Hour * 5)}, // Latest
				{EndTime: baseTime.Add(time.Hour * 4)},
			},
			expectedResult: &[]time.Time{baseTime.Add(time.Hour * 5)}[0],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLatestEndTime(tt.sessions)

			if tt.expectedResult == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.True(t, result.Equal(*tt.expectedResult))
			}
		})
	}
}

func TestSession_ValidateTimeSequence(t *testing.T) {
	// Test that sessions maintain proper time sequence
	baseTime := time.Now()

	session := &Session{
		EventID:   primitive.NewObjectID(),
		StartTime: baseTime.Add(time.Hour),
		EndTime:   baseTime.Add(time.Hour * 2),
	}

	// Valid session
	err := session.IsValid()
	assert.NoError(t, err)

	// Check duration
	duration := session.Duration()
	assert.Equal(t, time.Hour, duration)

	// Check that start is before end
	assert.True(t, session.StartTime.Before(session.EndTime))
}

func TestValidateSessions_PerformanceWithLargeSets(t *testing.T) {
	// Test with larger session sets to ensure O(n) performance
	eventID := primitive.NewObjectID()
	baseTime := time.Now()

	// Create 100 unique sessions
	sessions := make([]*Session, 0, 100)
	for i := 0; i < 100; i++ {
		sessions = append(sessions, &Session{
			ID:        primitive.NewObjectID(),
			EventID:   eventID,
			StartTime: baseTime.Add(time.Duration(i*2) * time.Hour),
			EndTime:   baseTime.Add(time.Duration(i*2+1) * time.Hour),
		})
	}

	// Should validate successfully
	err := ValidateSessions(sessions)
	assert.NoError(t, err)

	// Add a duplicate at the end
	sessions = append(sessions, &Session{
		ID:        primitive.NewObjectID(),
		EventID:   eventID,
		StartTime: baseTime, // Same as first session
		EndTime:   baseTime.Add(time.Hour),
	})

	// Should detect the duplicate
	err = ValidateSessions(sessions)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "have identical start and end times")
}

// Test cases for new Session fields: Name and Capacity
func TestSession_NewFields(t *testing.T) {
	eventID := primitive.NewObjectID()
	baseTime := time.Now()

	tests := []struct {
		name        string
		session     *Session
		expectError bool
		errorMsg    string
	}{
		{
			name: "Session with name only",
			session: &Session{
				EventID:   eventID,
				Name:      "Morning Workshop",
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: false,
		},
		{
			name: "Session with capacity only",
			session: &Session{
				EventID:   eventID,
				Capacity:  &[]int{50}[0],
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: false,
		},
		{
			name: "Session with both name and capacity",
			session: &Session{
				EventID:   eventID,
				Name:      "Afternoon Session",
				Capacity:  &[]int{100}[0],
				StartTime: baseTime.Add(time.Hour * 3),
				EndTime:   baseTime.Add(time.Hour * 4),
			},
			expectError: false,
		},
		{
			name: "Session with zero capacity",
			session: &Session{
				EventID:   eventID,
				Name:      "Zero Capacity Session",
				Capacity:  &[]int{0}[0],
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: false,
		},
		{
			name: "Session with negative capacity",
			session: &Session{
				EventID:   eventID,
				Name:      "Invalid Session",
				Capacity:  &[]int{-1}[0],
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: true,
			errorMsg:    "capacity must be non-negative",
		},
		{
			name: "Session with nil capacity (unlimited)",
			session: &Session{
				EventID:   eventID,
				Name:      "Unlimited Session",
				Capacity:  nil, // nil means unlimited
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: false,
		},
		{
			name: "Session with empty name",
			session: &Session{
				EventID:   eventID,
				Name:      "", // Empty name should be allowed
				StartTime: baseTime.Add(time.Hour),
				EndTime:   baseTime.Add(time.Hour * 2),
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.session.IsValid()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSession_FieldAccessors(t *testing.T) {
	eventID := primitive.NewObjectID()
	baseTime := time.Now()
	capacity := 75

	session := &Session{
		EventID:   eventID,
		Name:      "Test Session",
		Capacity:  &capacity,
		StartTime: baseTime.Add(time.Hour),
		EndTime:   baseTime.Add(time.Hour * 2),
	}

	// Test field values
	assert.Equal(t, "Test Session", session.Name)
	assert.NotNil(t, session.Capacity)
	assert.Equal(t, 75, *session.Capacity)
	assert.Equal(t, eventID, session.EventID)
	assert.Equal(t, baseTime.Add(time.Hour), session.StartTime)
	assert.Equal(t, baseTime.Add(time.Hour*2), session.EndTime)

	// Test unlimited capacity (nil pointer)
	unlimitedSession := &Session{
		EventID:   eventID,
		Name:      "Unlimited Session",
		Capacity:  nil,
		StartTime: baseTime.Add(time.Hour),
		EndTime:   baseTime.Add(time.Hour * 2),
	}

	assert.Nil(t, unlimitedSession.Capacity)
	err := unlimitedSession.IsValid()
	assert.NoError(t, err)
}
