package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefault(t *testing.T) {
	cfg := Default()

	if cfg.ListenAddr != ":3000" {
		t.Errorf("expected default ListenAddr ':3000', got %q", cfg.ListenAddr)
	}

	if cfg.WebhookPath != "/webhook" {
		t.Errorf("expected default WebhookPath '/webhook', got %q", cfg.WebhookPath)
	}

	if cfg.DefaultStatus != 200 {
		t.Errorf("expected default DefaultStatus 200, got %d", cfg.DefaultStatus)
	}

	if cfg.SessionsPerPage != 6 {
		t.Errorf("expected default SessionsPerPage 6, got %d", cfg.SessionsPerPage)
	}

	if cfg.DatabasePath != "./data/sessions.db" {
		t.Errorf("expected default DatabasePath './data/sessions.db', got %q", cfg.DatabasePath)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// Save original env vars
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	origSecretToken := os.Getenv("TELEGRAM_SECRET_TOKEN")
	origListenAddr := os.Getenv("LISTEN_ADDR")
	origWebhookPath := os.Getenv("WEBHOOK_PATH")
	origDefaultStatus := os.Getenv("DEFAULT_STATUS")
	origSessionsPerPage := os.Getenv("SESSIONS_PER_PAGE")
	origDatabasePath := os.Getenv("DATABASE_PATH")

	// Restore env vars after test
	defer func() {
		os.Setenv("TELEGRAM_BOT_TOKEN", origToken)
		os.Setenv("TELEGRAM_SECRET_TOKEN", origSecretToken)
		os.Setenv("LISTEN_ADDR", origListenAddr)
		os.Setenv("WEBHOOK_PATH", origWebhookPath)
		os.Setenv("DEFAULT_STATUS", origDefaultStatus)
		os.Setenv("SESSIONS_PER_PAGE", origSessionsPerPage)
		os.Setenv("DATABASE_PATH", origDatabasePath)
	}()

	// Set test env vars
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	os.Setenv("TELEGRAM_SECRET_TOKEN", "test-secret")
	os.Setenv("LISTEN_ADDR", ":8080")
	os.Setenv("WEBHOOK_PATH", "/test-webhook")
	os.Setenv("DEFAULT_STATUS", "201")
	os.Setenv("SESSIONS_PER_PAGE", "10")
	os.Setenv("DATABASE_PATH", "/tmp/test.db")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("expected Token 'test-token', got %q", cfg.Token)
	}

	if cfg.SecretToken != "test-secret" {
		t.Errorf("expected SecretToken 'test-secret', got %q", cfg.SecretToken)
	}

	if cfg.ListenAddr != ":8080" {
		t.Errorf("expected ListenAddr ':8080', got %q", cfg.ListenAddr)
	}

	if cfg.WebhookPath != "/test-webhook" {
		t.Errorf("expected WebhookPath '/test-webhook', got %q", cfg.WebhookPath)
	}

	if cfg.DefaultStatus != 201 {
		t.Errorf("expected DefaultStatus 201, got %d", cfg.DefaultStatus)
	}

	if cfg.SessionsPerPage != 10 {
		t.Errorf("expected SessionsPerPage 10, got %d", cfg.SessionsPerPage)
	}

	if cfg.DatabasePath != "/tmp/test.db" {
		t.Errorf("expected DatabasePath '/tmp/test.db', got %q", cfg.DatabasePath)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"token": "file-token",
		"secret_token": "file-secret",
		"listen_addr": ":9000",
		"webhook_path": "/file-webhook",
		"default_status": 202,
		"sessions_per_page": 8,
		"database_path": "/tmp/file.db"
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Clear env vars to ensure we're loading from file
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	defer os.Setenv("TELEGRAM_BOT_TOKEN", origToken)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if cfg.Token != "file-token" {
		t.Errorf("expected Token 'file-token', got %q", cfg.Token)
	}

	if cfg.SecretToken != "file-secret" {
		t.Errorf("expected SecretToken 'file-secret', got %q", cfg.SecretToken)
	}

	if cfg.ListenAddr != ":9000" {
		t.Errorf("expected ListenAddr ':9000', got %q", cfg.ListenAddr)
	}

	if cfg.WebhookPath != "/file-webhook" {
		t.Errorf("expected WebhookPath '/file-webhook', got %q", cfg.WebhookPath)
	}

	if cfg.DefaultStatus != 202 {
		t.Errorf("expected DefaultStatus 202, got %d", cfg.DefaultStatus)
	}

	if cfg.SessionsPerPage != 8 {
		t.Errorf("expected SessionsPerPage 8, got %d", cfg.SessionsPerPage)
	}

	if cfg.DatabasePath != "/tmp/file.db" {
		t.Errorf("expected DatabasePath '/tmp/file.db', got %q", cfg.DatabasePath)
	}
}

func TestEnvOverridesFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	configContent := `{
		"token": "file-token",
		"listen_addr": ":9000"
	}`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	// Set env var that should override file
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	os.Setenv("TELEGRAM_BOT_TOKEN", "env-token")
	defer os.Setenv("TELEGRAM_BOT_TOKEN", origToken)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Env var should override file
	if cfg.Token != "env-token" {
		t.Errorf("expected Token 'env-token' (from env), got %q", cfg.Token)
	}

	// File value should be used when no env var
	if cfg.ListenAddr != ":9000" {
		t.Errorf("expected ListenAddr ':9000' (from file), got %q", cfg.ListenAddr)
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		cfg       *Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Token:           "valid-token",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   200,
				SessionsPerPage: 6,
				DatabasePath:    "./data/sessions.db",
			},
			expectErr: false,
		},
		{
			name: "missing token",
			cfg: &Config{
				Token:           "",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   200,
				SessionsPerPage: 6,
				DatabasePath:    "./data/sessions.db",
			},
			expectErr: true,
			errMsg:    "bot token is required",
		},
		{
			name: "invalid default status - too low",
			cfg: &Config{
				Token:           "valid-token",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   99,
				SessionsPerPage: 6,
				DatabasePath:    "./data/sessions.db",
			},
			expectErr: true,
			errMsg:    "default_status must be between 100 and 599",
		},
		{
			name: "invalid default status - too high",
			cfg: &Config{
				Token:           "valid-token",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   600,
				SessionsPerPage: 6,
				DatabasePath:    "./data/sessions.db",
			},
			expectErr: true,
			errMsg:    "default_status must be between 100 and 599",
		},
		{
			name: "invalid sessions per page",
			cfg: &Config{
				Token:           "valid-token",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   200,
				SessionsPerPage: 0,
				DatabasePath:    "./data/sessions.db",
			},
			expectErr: true,
			errMsg:    "sessions_per_page must be at least 1",
		},
		{
			name: "missing database path",
			cfg: &Config{
				Token:           "valid-token",
				ListenAddr:      ":3000",
				WebhookPath:     "/webhook",
				DefaultStatus:   200,
				SessionsPerPage: 6,
				DatabasePath:    "",
			},
			expectErr: true,
			errMsg:    "database_path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
				} else if !contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	// Set required env var
	origToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	os.Setenv("TELEGRAM_BOT_TOKEN", "test-token")
	defer os.Setenv("TELEGRAM_BOT_TOKEN", origToken)

	// Load with non-existent file should use defaults + env
	cfg, err := Load("/non/existent/config.json")
	if err != nil {
		t.Fatalf("Load() with non-existent file should not fail: %v", err)
	}

	if cfg.Token != "test-token" {
		t.Errorf("expected Token 'test-token', got %q", cfg.Token)
	}

	// Should have default values
	if cfg.SessionsPerPage != 6 {
		t.Errorf("expected default SessionsPerPage 6, got %d", cfg.SessionsPerPage)
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
