package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config holds all configuration for the Telegram bot
type Config struct {
	// Bot configuration
	Token       string `json:"token"`
	SecretToken string `json:"secret_token"`

	// Server configuration
	ListenAddr    string `json:"listen_addr"`
	WebhookPath   string `json:"webhook_path"`
	DefaultStatus int    `json:"default_status"`

	// Session configuration
	SessionsPerPage int    `json:"sessions_per_page"`
	DatabasePath    string `json:"database_path"`
}

// Default returns a Config with sensible defaults
func Default() *Config {
	return &Config{
		Token:           "",
		SecretToken:     "",
		ListenAddr:      ":3000",
		WebhookPath:     "/webhook",
		DefaultStatus:   200,
		SessionsPerPage: 6,
		DatabasePath:    "./data/sessions.db",
	}
}

// Load loads configuration from environment variables and optional config file
// Environment variables take precedence over config file values
func Load(configPath string) (*Config, error) {
	cfg := Default()

	// Load from config file if provided
	if configPath != "" {
		if err := cfg.loadFromFile(configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Override with environment variables
	cfg.loadFromEnv()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// loadFromFile loads configuration from a JSON file
func (c *Config) loadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Config file is optional
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return nil
}

// loadFromEnv loads configuration from environment variables
func (c *Config) loadFromEnv() {
	if token := os.Getenv("TELEGRAM_BOT_TOKEN"); token != "" {
		c.Token = token
	}

	if secretToken := os.Getenv("TELEGRAM_SECRET_TOKEN"); secretToken != "" {
		c.SecretToken = secretToken
	}

	if listenAddr := os.Getenv("LISTEN_ADDR"); listenAddr != "" {
		c.ListenAddr = listenAddr
	}

	if webhookPath := os.Getenv("WEBHOOK_PATH"); webhookPath != "" {
		c.WebhookPath = webhookPath
	}

	if defaultStatus := os.Getenv("DEFAULT_STATUS"); defaultStatus != "" {
		if status, err := strconv.Atoi(defaultStatus); err == nil {
			c.DefaultStatus = status
		}
	}

	if sessionsPerPage := os.Getenv("SESSIONS_PER_PAGE"); sessionsPerPage != "" {
		if perPage, err := strconv.Atoi(sessionsPerPage); err == nil {
			c.SessionsPerPage = perPage
		}
	}

	if dbPath := os.Getenv("DATABASE_PATH"); dbPath != "" {
		c.DatabasePath = dbPath
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Token == "" {
		return fmt.Errorf("bot token is required (set TELEGRAM_BOT_TOKEN or provide in config file)")
	}

	if c.DefaultStatus < 100 || c.DefaultStatus > 599 {
		return fmt.Errorf("default_status must be between 100 and 599, got %d", c.DefaultStatus)
	}

	if c.SessionsPerPage < 1 {
		return fmt.Errorf("sessions_per_page must be at least 1, got %d", c.SessionsPerPage)
	}

	if c.DatabasePath == "" {
		return fmt.Errorf("database_path is required")
	}

	return nil
}
