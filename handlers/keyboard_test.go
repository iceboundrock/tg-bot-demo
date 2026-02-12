package handlers

import (
	"testing"
	"tg-bot-demo/session"
	"time"

	"github.com/google/uuid"
)

func TestBuildSessionKeyboard(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name            string
		sessions        []*session.Session
		offset          int
		hasMore         bool
		expectedRows    int
		expectedMoreBtn bool
	}{
		{
			name: "single session without more button",
			sessions: []*session.Session{
				{
					ID:          uuid.New(),
					UserID:      123,
					Title:       "Test Session",
					UpdatedAt:   now,
					CreatedAt:   now,
					LastMessage: "Hello",
				},
			},
			offset:          0,
			hasMore:         false,
			expectedRows:    1,
			expectedMoreBtn: false,
		},
		{
			name: "multiple sessions without more button",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
				{ID: uuid.New(), UserID: 123, Title: "Session 2", UpdatedAt: now, CreatedAt: now, LastMessage: "Hello"},
				{ID: uuid.New(), UserID: 123, Title: "Session 3", UpdatedAt: now, CreatedAt: now, LastMessage: "Hey"},
			},
			offset:          0,
			hasMore:         false,
			expectedRows:    3,
			expectedMoreBtn: false,
		},
		{
			name: "sessions with more button",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
				{ID: uuid.New(), UserID: 123, Title: "Session 2", UpdatedAt: now, CreatedAt: now, LastMessage: "Hello"},
				{ID: uuid.New(), UserID: 123, Title: "Session 3", UpdatedAt: now, CreatedAt: now, LastMessage: "Hey"},
			},
			offset:          0,
			hasMore:         true,
			expectedRows:    4, // 3 sessions + 1 more button
			expectedMoreBtn: true,
		},
		{
			name:            "empty sessions",
			sessions:        []*session.Session{},
			offset:          0,
			hasMore:         false,
			expectedRows:    0,
			expectedMoreBtn: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard := buildSessionKeyboard(tt.sessions, tt.offset, tt.hasMore, 6)

			if keyboard == nil {
				t.Fatal("keyboard is nil")
			}

			if len(keyboard.InlineKeyboard) != tt.expectedRows {
				t.Errorf("expected %d rows, got %d", tt.expectedRows, len(keyboard.InlineKeyboard))
			}

			// Check if more button exists when expected
			if tt.expectedMoreBtn {
				lastRow := keyboard.InlineKeyboard[len(keyboard.InlineKeyboard)-1]
				if len(lastRow) != 1 {
					t.Errorf("expected 1 button in last row, got %d", len(lastRow))
				}
				if lastRow[0].Text != "📄 More..." {
					t.Errorf("expected more button text '📄 More...', got %q", lastRow[0].Text)
				}
			}
		})
	}
}

func TestBuildSessionKeyboardCallbackData(t *testing.T) {
	now := time.Now()
	sessionID := uuid.New()

	sessions := []*session.Session{
		{
			ID:          sessionID,
			UserID:      123,
			Title:       "Test Session",
			UpdatedAt:   now,
			CreatedAt:   now,
			LastMessage: "Hello",
		},
	}

	t.Run("session button callback format", func(t *testing.T) {
		keyboard := buildSessionKeyboard(sessions, 0, false, 6)

		if len(keyboard.InlineKeyboard) != 1 {
			t.Fatalf("expected 1 row, got %d", len(keyboard.InlineKeyboard))
		}

		button := keyboard.InlineKeyboard[0][0]
		expectedCallback := "open_s_" + sessionID.String()

		if button.CallbackData != expectedCallback {
			t.Errorf("expected callback_data %q, got %q", expectedCallback, button.CallbackData)
		}
	})

	t.Run("more button callback format", func(t *testing.T) {
		offset := 6
		keyboard := buildSessionKeyboard(sessions, offset, true, 6)

		if len(keyboard.InlineKeyboard) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(keyboard.InlineKeyboard))
		}

		moreButton := keyboard.InlineKeyboard[1][0]
		expectedCallback := "more_sessions_12" // offset + SessionsPerPage = 6 + 6 = 12

		if moreButton.CallbackData != expectedCallback {
			t.Errorf("expected callback_data %q, got %q", expectedCallback, moreButton.CallbackData)
		}
	})
}

func TestFormatSessionButton(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		session  *session.Session
		contains []string // strings that should be in the result
	}{
		{
			name: "short title",
			session: &session.Session{
				ID:          uuid.New(),
				UserID:      123,
				Title:       "Short",
				UpdatedAt:   now.Add(-5 * time.Minute),
				CreatedAt:   now,
				LastMessage: "Hello",
			},
			contains: []string{"Short", "5m ago"},
		},
		{
			name: "long title gets truncated",
			session: &session.Session{
				ID:          uuid.New(),
				UserID:      123,
				Title:       "This is a very long title that should be truncated to 40 characters",
				UpdatedAt:   now.Add(-2 * time.Hour),
				CreatedAt:   now,
				LastMessage: "Hello",
			},
			contains: []string{"...", "2h ago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSessionButton(tt.session)

			for _, substr := range tt.contains {
				if !contains(result, substr) {
					t.Errorf("expected result to contain %q, got %q", substr, result)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
