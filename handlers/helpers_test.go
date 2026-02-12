package handlers

import (
	"testing"
	"time"
)

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "just now - 30 seconds ago",
			time:     now.Add(-30 * time.Second),
			expected: "just now",
		},
		{
			name:     "minutes ago - 5 minutes",
			time:     now.Add(-5 * time.Minute),
			expected: "5m ago",
		},
		{
			name:     "minutes ago - 45 minutes",
			time:     now.Add(-45 * time.Minute),
			expected: "45m ago",
		},
		{
			name:     "hours ago - 2 hours",
			time:     now.Add(-2 * time.Hour),
			expected: "2h ago",
		},
		{
			name:     "hours ago - 23 hours",
			time:     now.Add(-23 * time.Hour),
			expected: "23h ago",
		},
		{
			name:     "days ago - 1 day",
			time:     now.Add(-24 * time.Hour),
			expected: "1d ago",
		},
		{
			name:     "days ago - 6 days",
			time:     now.Add(-6 * 24 * time.Hour),
			expected: "6d ago",
		},
		{
			name:     "date format - 8 days ago",
			time:     now.Add(-8 * 24 * time.Hour),
			expected: now.Add(-8 * 24 * time.Hour).Format("Jan 2"),
		},
		{
			name:     "date format - 30 days ago",
			time:     now.Add(-30 * 24 * time.Hour),
			expected: now.Add(-30 * 24 * time.Hour).Format("Jan 2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

func TestFormatTimeAgoBoundaries(t *testing.T) {
	now := time.Now()

	// Test exact boundaries
	tests := []struct {
		name     string
		time     time.Time
		expected string
	}{
		{
			name:     "exactly 1 minute",
			time:     now.Add(-1 * time.Minute),
			expected: "1m ago",
		},
		{
			name:     "exactly 1 hour",
			time:     now.Add(-1 * time.Hour),
			expected: "1h ago",
		},
		{
			name:     "exactly 24 hours",
			time:     now.Add(-24 * time.Hour),
			expected: "1d ago",
		},
		{
			name:     "exactly 7 days",
			time:     now.Add(-7 * 24 * time.Hour),
			expected: now.Add(-7 * 24 * time.Hour).Format("Jan 2"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatTimeAgo(tt.time)
			if result != tt.expected {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.time, result, tt.expected)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string - no truncation",
			input:    "Hello",
			maxLen:   10,
			expected: "Hello",
		},
		{
			name:     "exact length - no truncation",
			input:    "Hello",
			maxLen:   5,
			expected: "Hello",
		},
		{
			name:     "long string - truncate with ellipsis",
			input:    "This is a very long string",
			maxLen:   10,
			expected: "This is...",
		},
		{
			name:     "UTF-8 string - handle runes correctly",
			input:    "Hello 世界",
			maxLen:   8,
			expected: "Hello 世界",
		},
		{
			name:     "UTF-8 string - truncate with runes",
			input:    "Hello 世界 from Go",
			maxLen:   10,
			expected: "Hello 世...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
			}
		})
	}
}
