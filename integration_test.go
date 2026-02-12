package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"tg-bot-demo/config"
	"tg-bot-demo/session"
)

// TestIntegration_BotInitializationFlow tests the complete bot initialization flow
func TestIntegration_BotInitializationFlow(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "integration_test.db")

	// Create test config
	cfg := &config.Config{
		Token:           "123456:test-token",
		SecretToken:     "test-secret",
		ListenAddr:      ":3000",
		WebhookPath:     "/webhook",
		DefaultStatus:   200,
		SessionsPerPage: 6,
		DatabasePath:    dbPath,
	}

	// Initialize the bot
	bot, store, err := initializeBot(cfg)
	if err != nil {
		t.Fatalf("initializeBot failed: %v", err)
	}
	defer store.Close()

	// Verify bot was created
	if bot == nil {
		t.Fatal("bot is nil")
	}

	// Create a test session to verify the store is working
	ctx := context.Background()
	testSession := session.NewSession(12345, "Test message for integration")

	err = store.Create(ctx, testSession)
	if err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	// Retrieve the session
	retrieved, err := store.Get(ctx, testSession.ID)
	if err != nil {
		t.Fatalf("failed to retrieve session: %v", err)
	}

	// Verify session data
	if retrieved.UserID != testSession.UserID {
		t.Errorf("UserID mismatch: got %d, want %d", retrieved.UserID, testSession.UserID)
	}
	if retrieved.Title != testSession.Title {
		t.Errorf("Title mismatch: got %s, want %s", retrieved.Title, testSession.Title)
	}
	if retrieved.LastMessage != testSession.LastMessage {
		t.Errorf("LastMessage mismatch: got %s, want %s", retrieved.LastMessage, testSession.LastMessage)
	}

	// Test active session functionality
	err = store.SetActiveSession(ctx, testSession.UserID, testSession.ID)
	if err != nil {
		t.Fatalf("failed to set active session: %v", err)
	}

	activeSession, err := store.GetActiveSession(ctx, testSession.UserID)
	if err != nil {
		t.Fatalf("failed to get active session: %v", err)
	}

	if activeSession.ID != testSession.ID {
		t.Errorf("Active session ID mismatch: got %s, want %s", activeSession.ID, testSession.ID)
	}
}

// TestIntegration_MultipleUsers tests session isolation between users
func TestIntegration_MultipleUsers(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "multi_user_test.db")

	cfg := &config.Config{
		Token:           "123456:test-token",
		SecretToken:     "test-secret",
		ListenAddr:      ":3000",
		WebhookPath:     "/webhook",
		DefaultStatus:   200,
		SessionsPerPage: 6,
		DatabasePath:    dbPath,
	}

	_, store, err := initializeBot(cfg)
	if err != nil {
		t.Fatalf("initializeBot failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()

	// Create sessions for two different users
	user1ID := int64(111)
	user2ID := int64(222)

	session1 := session.NewSession(user1ID, "User 1 message")
	session2 := session.NewSession(user2ID, "User 2 message")

	if err := store.Create(ctx, session1); err != nil {
		t.Fatalf("failed to create session1: %v", err)
	}
	if err := store.Create(ctx, session2); err != nil {
		t.Fatalf("failed to create session2: %v", err)
	}

	// List sessions for user 1
	user1Sessions, err := store.ListByUser(ctx, user1ID, 0, 10)
	if err != nil {
		t.Fatalf("failed to list user1 sessions: %v", err)
	}

	// List sessions for user 2
	user2Sessions, err := store.ListByUser(ctx, user2ID, 0, 10)
	if err != nil {
		t.Fatalf("failed to list user2 sessions: %v", err)
	}

	// Verify each user only sees their own sessions
	if len(user1Sessions) != 1 {
		t.Errorf("user1 should have 1 session, got %d", len(user1Sessions))
	}
	if len(user2Sessions) != 1 {
		t.Errorf("user2 should have 1 session, got %d", len(user2Sessions))
	}

	if len(user1Sessions) > 0 && user1Sessions[0].UserID != user1ID {
		t.Errorf("user1 session has wrong UserID: got %d, want %d", user1Sessions[0].UserID, user1ID)
	}
	if len(user2Sessions) > 0 && user2Sessions[0].UserID != user2ID {
		t.Errorf("user2 session has wrong UserID: got %d, want %d", user2Sessions[0].UserID, user2ID)
	}
}

// TestIntegration_SessionPagination tests pagination functionality
func TestIntegration_SessionPagination(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "pagination_test.db")

	cfg := &config.Config{
		Token:           "123456:test-token",
		SecretToken:     "test-secret",
		ListenAddr:      ":3000",
		WebhookPath:     "/webhook",
		DefaultStatus:   200,
		SessionsPerPage: 6,
		DatabasePath:    dbPath,
	}

	_, store, err := initializeBot(cfg)
	if err != nil {
		t.Fatalf("initializeBot failed: %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	userID := int64(333)

	// Create 10 sessions for the user
	for i := 0; i < 10; i++ {
		sess := session.NewSession(userID, "Message "+string(rune('A'+i)))
		// Add a small delay to ensure different timestamps
		time.Sleep(time.Millisecond)
		if err := store.Create(ctx, sess); err != nil {
			t.Fatalf("failed to create session %d: %v", i, err)
		}
	}

	// Test first page (6 sessions)
	page1, err := store.ListByUser(ctx, userID, 0, 6)
	if err != nil {
		t.Fatalf("failed to get page 1: %v", err)
	}
	if len(page1) != 6 {
		t.Errorf("page 1 should have 6 sessions, got %d", len(page1))
	}

	// Test second page (4 sessions)
	page2, err := store.ListByUser(ctx, userID, 6, 6)
	if err != nil {
		t.Fatalf("failed to get page 2: %v", err)
	}
	if len(page2) != 4 {
		t.Errorf("page 2 should have 4 sessions, got %d", len(page2))
	}

	// Verify total count
	count, err := store.CountByUser(ctx, userID)
	if err != nil {
		t.Fatalf("failed to count sessions: %v", err)
	}
	if count != 10 {
		t.Errorf("total count should be 10, got %d", count)
	}

	// Verify sessions are sorted by updated_at DESC (most recent first)
	if len(page1) >= 2 {
		if page1[0].UpdatedAt.Before(page1[1].UpdatedAt) {
			t.Error("sessions are not sorted by updated_at DESC")
		}
	}
}
