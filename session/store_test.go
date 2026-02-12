package session

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSQLiteStore_BasicOperations(t *testing.T) {
	// Create temporary database
	dbPath := "test_sessions.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test Create
	session := NewSession(12345, "Hello, world!")
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get session: %v", err)
	}
	if retrieved.ID != session.ID {
		t.Errorf("Expected ID %v, got %v", session.ID, retrieved.ID)
	}
	if retrieved.UserID != session.UserID {
		t.Errorf("Expected UserID %v, got %v", session.UserID, retrieved.UserID)
	}
	if retrieved.Title != session.Title {
		t.Errorf("Expected Title %v, got %v", session.Title, retrieved.Title)
	}

	// Test Update
	time.Sleep(10 * time.Millisecond) // Ensure updated_at changes
	retrieved.Title = "Updated Title"
	retrieved.UpdatedAt = time.Now()
	if err := store.Update(ctx, retrieved); err != nil {
		t.Fatalf("Failed to update session: %v", err)
	}

	updated, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}
	if updated.Title != "Updated Title" {
		t.Errorf("Expected updated title, got %v", updated.Title)
	}

	// Test Delete
	if err := store.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Failed to delete session: %v", err)
	}

	_, err = store.Get(ctx, session.ID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}
}

func TestSQLiteStore_ListByUser(t *testing.T) {
	dbPath := "test_sessions_list.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	userID := int64(12345)

	// Create multiple sessions for the same user
	for i := 0; i < 10; i++ {
		session := NewSession(userID, "Message "+string(rune('A'+i)))
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Failed to create session %d: %v", i, err)
		}
	}

	// Test CountByUser
	count, err := store.CountByUser(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to count sessions: %v", err)
	}
	if count != 10 {
		t.Errorf("Expected 10 sessions, got %d", count)
	}

	// Test ListByUser with pagination
	sessions, err := store.ListByUser(ctx, userID, 0, 5)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) != 5 {
		t.Errorf("Expected 5 sessions, got %d", len(sessions))
	}

	// Verify sorting (most recent first)
	for i := 0; i < len(sessions)-1; i++ {
		if sessions[i].UpdatedAt.Before(sessions[i+1].UpdatedAt) {
			t.Errorf("Sessions not sorted by updated_at DESC")
		}
	}

	// Test pagination offset
	sessions2, err := store.ListByUser(ctx, userID, 5, 5)
	if err != nil {
		t.Fatalf("Failed to list sessions with offset: %v", err)
	}
	if len(sessions2) != 5 {
		t.Errorf("Expected 5 sessions, got %d", len(sessions2))
	}

	// Verify no overlap
	for _, s1 := range sessions {
		for _, s2 := range sessions2 {
			if s1.ID == s2.ID {
				t.Errorf("Found duplicate session in pagination")
			}
		}
	}
}

func TestSQLiteStore_UserIsolation(t *testing.T) {
	dbPath := "test_sessions_isolation.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create sessions for different users
	user1 := int64(111)
	user2 := int64(222)

	session1 := NewSession(user1, "User 1 message")
	session2 := NewSession(user2, "User 2 message")

	if err := store.Create(ctx, session1); err != nil {
		t.Fatalf("Failed to create session for user1: %v", err)
	}
	if err := store.Create(ctx, session2); err != nil {
		t.Fatalf("Failed to create session for user2: %v", err)
	}

	// Verify user1 only sees their session
	sessions, err := store.ListByUser(ctx, user1, 0, 10)
	if err != nil {
		t.Fatalf("Failed to list sessions for user1: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for user1, got %d", len(sessions))
	}
	if sessions[0].UserID != user1 {
		t.Errorf("Expected session for user1, got user %d", sessions[0].UserID)
	}

	// Verify user2 only sees their session
	sessions, err = store.ListByUser(ctx, user2, 0, 10)
	if err != nil {
		t.Fatalf("Failed to list sessions for user2: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session for user2, got %d", len(sessions))
	}
	if sessions[0].UserID != user2 {
		t.Errorf("Expected session for user2, got user %d", sessions[0].UserID)
	}
}

func TestSQLiteStore_ActiveSession(t *testing.T) {
	dbPath := "test_active_session.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	userID := int64(12345)

	// Create a session
	session := NewSession(userID, "Test message")
	if err := store.Create(ctx, session); err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Test GetActiveSession when none exists
	_, err = store.GetActiveSession(ctx, userID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}

	// Test SetActiveSession
	if err := store.SetActiveSession(ctx, userID, session.ID); err != nil {
		t.Fatalf("Failed to set active session: %v", err)
	}

	// Test GetActiveSession
	active, err := store.GetActiveSession(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get active session: %v", err)
	}
	if active.ID != session.ID {
		t.Errorf("Expected active session ID %v, got %v", session.ID, active.ID)
	}

	// Create another session and switch to it
	session2 := NewSession(userID, "Another message")
	if err := store.Create(ctx, session2); err != nil {
		t.Fatalf("Failed to create second session: %v", err)
	}

	if err := store.SetActiveSession(ctx, userID, session2.ID); err != nil {
		t.Fatalf("Failed to switch active session: %v", err)
	}

	active, err = store.GetActiveSession(ctx, userID)
	if err != nil {
		t.Fatalf("Failed to get active session after switch: %v", err)
	}
	if active.ID != session2.ID {
		t.Errorf("Expected active session ID %v, got %v", session2.ID, active.ID)
	}

	// Test ClearActiveSession
	if err := store.ClearActiveSession(ctx, userID); err != nil {
		t.Fatalf("Failed to clear active session: %v", err)
	}

	_, err = store.GetActiveSession(ctx, userID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound after clear, got %v", err)
	}

	// Clearing again should be idempotent
	if err := store.ClearActiveSession(ctx, userID); err != nil {
		t.Fatalf("Second clear should not fail: %v", err)
	}
}

func TestSQLiteStore_ErrorCases(t *testing.T) {
	dbPath := "test_errors.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Test Get with non-existent ID
	nonExistentID := uuid.New()
	_, err = store.Get(ctx, nonExistentID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound, got %v", err)
	}

	// Test Update with non-existent ID
	session := NewSession(12345, "Test")
	session.ID = nonExistentID
	err = store.Update(ctx, session)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound on update, got %v", err)
	}

	// Test Delete with non-existent ID
	err = store.Delete(ctx, nonExistentID)
	if err != ErrSessionNotFound {
		t.Errorf("Expected ErrSessionNotFound on delete, got %v", err)
	}
}

func TestSession_TitleGeneration(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "Short message (< 10 chars)",
			message:  "Hello",
			expected: "Hello",
		},
		{
			name:     "Medium message (10-30 chars)",
			message:  "This is a test message",
			expected: "This is a test message",
		},
		{
			name:     "Long message (> 30 chars)",
			message:  "This is a very long message that exceeds thirty characters",
			expected: "This is a very long message th...",
		},
		{
			name:     "Message with newlines",
			message:  "Line 1\nLine 2\nLine 3",
			expected: "Line 1 Line 2 Line 3",
		},
		{
			name:     "Whitespace only",
			message:  "   \n\t  ",
			expected: "New Session",
		},
		{
			name:     "Empty message",
			message:  "",
			expected: "New Session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := NewSession(12345, tt.message)
			// For whitespace/empty, just check it starts with "New Session"
			if tt.message == "" || strings.TrimSpace(tt.message) == "" {
				if !strings.HasPrefix(session.Title, "New Session") {
					t.Errorf("Expected title to start with 'New Session', got %v", session.Title)
				}
			} else {
				if session.Title != tt.expected {
					t.Errorf("Expected title %v, got %v", tt.expected, session.Title)
				}
			}
		})
	}
}

func TestManager_ListSessions(t *testing.T) {
	dbPath := "test_manager_list.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	manager := NewManager(store)
	ctx := context.Background()

	// Create test sessions
	for i := 0; i < 10; i++ {
		session := NewSession(123, fmt.Sprintf("Message %d", i))
		if err := store.Create(ctx, session); err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
	}

	// Test pagination
	sessions, hasMore, err := manager.ListSessions(ctx, 123, 0, 6)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 6 {
		t.Errorf("Expected 6 sessions, got %d", len(sessions))
	}

	if !hasMore {
		t.Error("Expected hasMore to be true")
	}

	// Test last page
	sessions, hasMore, err = manager.ListSessions(ctx, 123, 6, 6)
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 4 {
		t.Errorf("Expected 4 sessions, got %d", len(sessions))
	}

	if hasMore {
		t.Error("Expected hasMore to be false")
	}
}

func TestManager_SwitchSession(t *testing.T) {
	dbPath := "test_manager_switch.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	manager := NewManager(store)
	ctx := context.Background()

	// Create sessions for two users
	session1 := NewSession(123, "User 1 message")
	session2 := NewSession(456, "User 2 message")

	if err := store.Create(ctx, session1); err != nil {
		t.Fatalf("Failed to create session1: %v", err)
	}
	if err := store.Create(ctx, session2); err != nil {
		t.Fatalf("Failed to create session2: %v", err)
	}

	// Test successful switch
	result, err := manager.SwitchSession(ctx, 123, session1.ID)
	if err != nil {
		t.Fatalf("SwitchSession failed: %v", err)
	}

	if result.ID != session1.ID {
		t.Errorf("Expected session ID %v, got %v", session1.ID, result.ID)
	}

	// Verify active session was set
	active, err := store.GetActiveSession(ctx, 123)
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}

	if active.ID != session1.ID {
		t.Errorf("Expected active session ID %v, got %v", session1.ID, active.ID)
	}

	// Test unauthorized access
	_, err = manager.SwitchSession(ctx, 123, session2.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}

	// Test non-existent session
	nonExistentID := uuid.New()
	_, err = manager.SwitchSession(ctx, 123, nonExistentID)
	if err == nil {
		t.Error("Expected error when switching to non-existent session, got nil")
	}
}

func TestManager_CreateSession(t *testing.T) {
	dbPath := "test_manager_create.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	manager := NewManager(store)
	ctx := context.Background()

	// Create session
	session, err := manager.CreateSession(ctx, 123, "Test message")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	if session.UserID != 123 {
		t.Errorf("Expected UserID 123, got %d", session.UserID)
	}

	if session.Title != "Test message" {
		t.Errorf("Expected title 'Test message', got '%s'", session.Title)
	}

	// Verify session was set as active
	active, err := store.GetActiveSession(ctx, 123)
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}

	if active.ID != session.ID {
		t.Errorf("Expected active session ID %v, got %v", session.ID, active.ID)
	}
}

func TestManager_GetOrCreateActiveSession(t *testing.T) {
	dbPath := "test_manager_get_or_create.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	manager := NewManager(store)
	ctx := context.Background()

	// Test auto-create when no active session
	session1, err := manager.GetOrCreateActiveSession(ctx, 123, "First message")
	if err != nil {
		t.Fatalf("GetOrCreateActiveSession failed: %v", err)
	}

	if session1.Title != "First message" {
		t.Errorf("Expected title 'First message', got '%s'", session1.Title)
	}

	// Test return existing active session
	session2, err := manager.GetOrCreateActiveSession(ctx, 123, "Second message")
	if err != nil {
		t.Fatalf("GetOrCreateActiveSession failed: %v", err)
	}

	if session2.ID != session1.ID {
		t.Error("Expected to get the same active session")
	}

	if session2.Title != "First message" {
		t.Errorf("Expected title 'First message', got '%s'", session2.Title)
	}
}

func TestManager_CloseActiveSession(t *testing.T) {
	dbPath := "test_manager_close_active.db"
	defer os.Remove(dbPath)

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}
	defer store.Close()

	manager := NewManager(store)
	ctx := context.Background()
	userID := int64(123)

	// No active session should return closed=false without error.
	sess, closed, err := manager.CloseActiveSession(ctx, userID)
	if err != nil {
		t.Fatalf("CloseActiveSession failed: %v", err)
	}
	if closed {
		t.Fatal("Expected closed=false when no active session exists")
	}
	if sess != nil {
		t.Fatal("Expected session=nil when no active session exists")
	}

	// Create one active session and close it.
	created, err := manager.CreateSession(ctx, userID, "Test message")
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	sess, closed, err = manager.CloseActiveSession(ctx, userID)
	if err != nil {
		t.Fatalf("CloseActiveSession failed: %v", err)
	}
	if !closed {
		t.Fatal("Expected closed=true when active session exists")
	}
	if sess == nil {
		t.Fatal("Expected closed session details")
	}
	if sess.ID != created.ID {
		t.Fatalf("Expected closed session ID %v, got %v", created.ID, sess.ID)
	}

	_, err = store.GetActiveSession(ctx, userID)
	if err != ErrSessionNotFound {
		t.Fatalf("Expected no active session after close, got %v", err)
	}
}
