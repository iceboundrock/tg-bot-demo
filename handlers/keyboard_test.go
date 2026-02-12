package handlers

import (
	"strings"
	"testing"
	"tg-bot-demo/session"
	"time"

	"github.com/google/uuid"
)

func TestBuildSessionKeyboard(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name              string
		sessions          []*session.Session
		offset            int
		hasPrev           bool
		hasNext           bool
		expectedRows      int
		expectedTopNav    string
		expectedBottomNav string
	}{
		{
			name: "single session without pagination",
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
			offset:            0,
			hasPrev:           false,
			hasNext:           false,
			expectedRows:      1,
			expectedTopNav:    "",
			expectedBottomNav: "",
		},
		{
			name: "multiple sessions without pagination",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
				{ID: uuid.New(), UserID: 123, Title: "Session 2", UpdatedAt: now, CreatedAt: now, LastMessage: "Hello"},
				{ID: uuid.New(), UserID: 123, Title: "Session 3", UpdatedAt: now, CreatedAt: now, LastMessage: "Hey"},
			},
			offset:            0,
			hasPrev:           false,
			hasNext:           false,
			expectedRows:      3,
			expectedTopNav:    "",
			expectedBottomNav: "",
		},
		{
			name: "sessions with next button only",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
				{ID: uuid.New(), UserID: 123, Title: "Session 2", UpdatedAt: now, CreatedAt: now, LastMessage: "Hello"},
				{ID: uuid.New(), UserID: 123, Title: "Session 3", UpdatedAt: now, CreatedAt: now, LastMessage: "Hey"},
			},
			offset:            0,
			hasPrev:           false,
			hasNext:           true,
			expectedRows:      4, // 3 sessions + bottom nav row
			expectedTopNav:    "",
			expectedBottomNav: nextPageButtonText,
		},
		{
			name: "sessions with prev and next buttons",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
				{ID: uuid.New(), UserID: 123, Title: "Session 2", UpdatedAt: now, CreatedAt: now, LastMessage: "Hello"},
			},
			offset:            6,
			hasPrev:           true,
			hasNext:           true,
			expectedRows:      4, // top nav + 2 sessions + bottom nav
			expectedTopNav:    prevPageButtonText,
			expectedBottomNav: nextPageButtonText,
		},
		{
			name: "sessions with prev button only",
			sessions: []*session.Session{
				{ID: uuid.New(), UserID: 123, Title: "Session 1", UpdatedAt: now, CreatedAt: now, LastMessage: "Hi"},
			},
			offset:            6,
			hasPrev:           true,
			hasNext:           false,
			expectedRows:      2, // top nav + 1 session
			expectedTopNav:    prevPageButtonText,
			expectedBottomNav: "",
		},
		{
			name:              "empty sessions with both nav buttons",
			sessions:          []*session.Session{},
			offset:            6,
			hasPrev:           true,
			hasNext:           true,
			expectedRows:      2,
			expectedTopNav:    prevPageButtonText,
			expectedBottomNav: nextPageButtonText,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keyboard := buildSessionKeyboard(tt.sessions, tt.offset, tt.hasPrev, tt.hasNext, 6)

			if keyboard == nil {
				t.Fatal("keyboard is nil")
			}

			if len(keyboard.InlineKeyboard) != tt.expectedRows {
				t.Errorf("expected %d rows, got %d", tt.expectedRows, len(keyboard.InlineKeyboard))
			}

			if tt.expectedTopNav != "" {
				topRow := keyboard.InlineKeyboard[0]
				if len(topRow) != 1 {
					t.Fatalf("expected 1 button in top row, got %d", len(topRow))
				}
				if topRow[0].Text != tt.expectedTopNav {
					t.Errorf("expected top nav %q, got %q", tt.expectedTopNav, topRow[0].Text)
				}
			}

			if tt.expectedBottomNav != "" {
				lastRow := keyboard.InlineKeyboard[len(keyboard.InlineKeyboard)-1]
				if len(lastRow) != 1 {
					t.Fatalf("expected 1 button in bottom row, got %d", len(lastRow))
				}
				if lastRow[0].Text != tt.expectedBottomNav {
					t.Errorf("expected bottom nav %q, got %q", tt.expectedBottomNav, lastRow[0].Text)
				}
			}

			if len(tt.sessions) > 0 {
				sessionRowIndex := 0
				if tt.hasPrev {
					sessionRowIndex = 1
				}
				firstSessionBtn := keyboard.InlineKeyboard[sessionRowIndex][0]
				if !strings.HasPrefix(firstSessionBtn.CallbackData, "open_s_") {
					t.Errorf("expected first session row callback to start with open_s_, got %q", firstSessionBtn.CallbackData)
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
		keyboard := buildSessionKeyboard(sessions, 0, false, false, 6)

		if len(keyboard.InlineKeyboard) != 1 {
			t.Fatalf("expected 1 row, got %d", len(keyboard.InlineKeyboard))
		}

		button := keyboard.InlineKeyboard[0][0]
		expectedCallback := "open_s_" + sessionID.String()

		if button.CallbackData != expectedCallback {
			t.Errorf("expected callback_data %q, got %q", expectedCallback, button.CallbackData)
		}
	})

	t.Run("next button callback format", func(t *testing.T) {
		offset := 6
		keyboard := buildSessionKeyboard(sessions, offset, false, true, 6)

		if len(keyboard.InlineKeyboard) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(keyboard.InlineKeyboard))
		}

		nextButton := keyboard.InlineKeyboard[1][0]
		expectedCallback := "page_sessions_12" // offset + SessionsPerPage = 6 + 6 = 12

		if nextButton.CallbackData != expectedCallback {
			t.Errorf("expected callback_data %q, got %q", expectedCallback, nextButton.CallbackData)
		}
		if nextButton.Text != nextPageButtonText {
			t.Errorf("expected button text %q, got %q", nextPageButtonText, nextButton.Text)
		}
	})

	t.Run("prev and next callback format", func(t *testing.T) {
		offset := 6
		keyboard := buildSessionKeyboard(sessions, offset, true, true, 6)

		if len(keyboard.InlineKeyboard) != 3 {
			t.Fatalf("expected 3 rows, got %d", len(keyboard.InlineKeyboard))
		}

		prevRow := keyboard.InlineKeyboard[0]
		if len(prevRow) != 1 {
			t.Fatalf("expected 1 prev button, got %d", len(prevRow))
		}

		if prevRow[0].CallbackData != "page_sessions_0" {
			t.Errorf("expected prev callback_data %q, got %q", "page_sessions_0", prevRow[0].CallbackData)
		}
		if prevRow[0].Text != prevPageButtonText {
			t.Errorf("expected prev button text %q, got %q", prevPageButtonText, prevRow[0].Text)
		}

		nextRow := keyboard.InlineKeyboard[2]
		if len(nextRow) != 1 {
			t.Fatalf("expected 1 next button, got %d", len(nextRow))
		}

		if nextRow[0].CallbackData != "page_sessions_12" {
			t.Errorf("expected next callback_data %q, got %q", "page_sessions_12", nextRow[0].CallbackData)
		}
		if nextRow[0].Text != nextPageButtonText {
			t.Errorf("expected next button text %q, got %q", nextPageButtonText, nextRow[0].Text)
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
