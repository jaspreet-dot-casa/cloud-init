package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "zero time",
			time:     time.Time{},
			expected: "never",
		},
		{
			name:     "just now",
			time:     time.Now().Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "1 minute ago",
			time:     time.Now().Add(-1 * time.Minute),
			expected: "1 minute ago",
		},
		{
			name:     "5 minutes ago",
			time:     time.Now().Add(-5 * time.Minute),
			expected: "5 minutes ago",
		},
		{
			name:     "1 hour ago",
			time:     time.Now().Add(-1 * time.Hour),
			expected: "1 hour ago",
		},
		{
			name:     "3 hours ago",
			time:     time.Now().Add(-3 * time.Hour),
			expected: "3 hours ago",
		},
		{
			name:     "1 day ago",
			time:     time.Now().Add(-24 * time.Hour),
			expected: "1 day ago",
		},
		{
			name:     "3 days ago",
			time:     time.Now().Add(-72 * time.Hour),
			expected: "3 days ago",
		},
		{
			name:     "1 week ago",
			time:     time.Now().Add(-7 * 24 * time.Hour),
			expected: "1 week ago",
		},
		{
			name:     "2 weeks ago",
			time:     time.Now().Add(-14 * 24 * time.Hour),
			expected: "2 weeks ago",
		},
		{
			name:     "future time",
			time:     time.Now().Add(1 * time.Hour),
			expected: "in the future",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTimeAgo(tt.time)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTimeAgo_OldDate(t *testing.T) {
	oldDate := time.Date(2020, 1, 15, 0, 0, 0, 0, time.UTC)
	result := FormatTimeAgo(oldDate)
	assert.Equal(t, "Jan 15, 2020", result)
}
