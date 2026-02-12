package main

import (
	"os"
	"path/filepath"
	"testing"

	"tg-bot-demo/config"
)

func TestInitializeBot(t *testing.T) {
	// Create a temporary directory for the test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_sessions.db")

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

	// Verify store was created
	if store == nil {
		t.Fatal("store is nil")
	}

	// Verify database file was created
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Fatalf("database file was not created: %s", dbPath)
	}
}

func TestInitializeBotWithInvalidDBPath(t *testing.T) {
	// Test with an invalid database path (directory that doesn't exist and can't be created)
	dbPath := "/invalid/path/that/does/not/exist/sessions.db"

	cfg := &config.Config{
		Token:           "123456:test-token",
		SecretToken:     "test-secret",
		ListenAddr:      ":3000",
		WebhookPath:     "/webhook",
		DefaultStatus:   200,
		SessionsPerPage: 6,
		DatabasePath:    dbPath,
	}

	// Initialize the bot - should fail
	_, _, err := initializeBot(cfg)
	if err == nil {
		t.Fatal("expected error with invalid database path, got nil")
	}
}
