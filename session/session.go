package session

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// Package session provides session management functionality for the Telegram bot.
// It includes data models, storage interfaces, and business logic for handling
// conversation sessions between users and AI.

// Session represents a conversation session between a user and AI
type Session struct {
	ID          uuid.UUID `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastMessage string    `json:"last_message"`
}

// NewSession creates a new session with generated UUID
func NewSession(userID int64, firstMessage string) *Session {
	now := time.Now()
	return &Session{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       generateTitle(firstMessage),
		CreatedAt:   now,
		UpdatedAt:   now,
		LastMessage: firstMessage,
	}
}

// generateTitle creates a meaningful title from the first message
func generateTitle(message string) string {
	// Remove leading/trailing whitespace
	message = strings.TrimSpace(message)

	// Handle empty or whitespace-only messages
	if message == "" {
		return fmt.Sprintf("New Session %s", time.Now().Format("15:04"))
	}

	// Replace newlines with spaces
	message = strings.ReplaceAll(message, "\n", " ")
	message = strings.ReplaceAll(message, "\r", " ")

	// Collapse multiple spaces
	message = strings.Join(strings.Fields(message), " ")

	// Truncate to 30 characters, but use full message if < 10 chars
	runeCount := utf8.RuneCountInString(message)
	if runeCount <= 10 {
		return message
	}

	if runeCount > 30 {
		runes := []rune(message)
		return string(runes[:30]) + "..."
	}

	return message
}

// Store defines the interface for session persistence
type Store interface {
	// Create stores a new session
	Create(ctx context.Context, session *Session) error

	// Get retrieves a session by ID
	Get(ctx context.Context, id uuid.UUID) (*Session, error)

	// Update modifies an existing session
	Update(ctx context.Context, session *Session) error

	// Delete removes a session
	Delete(ctx context.Context, id uuid.UUID) error

	// ListByUser returns sessions for a specific user with pagination
	ListByUser(ctx context.Context, userID int64, offset, limit int) ([]*Session, error)

	// CountByUser returns total number of sessions for a user
	CountByUser(ctx context.Context, userID int64) (int, error)

	// GetActiveSession returns the current active session for a user
	GetActiveSession(ctx context.Context, userID int64) (*Session, error)

	// SetActiveSession sets the active session for a user
	SetActiveSession(ctx context.Context, userID int64, sessionID uuid.UUID) error
}

// Error types
var (
	ErrSessionNotFound = fmt.Errorf("session not found")
	ErrUnauthorized    = fmt.Errorf("unauthorized access to session")
)

// Manager handles session business logic
type Manager struct {
	store Store
}

// NewManager creates a new session manager
func NewManager(store Store) *Manager {
	return &Manager{store: store}
}

// ListSessions retrieves paginated sessions for a user
func (m *Manager) ListSessions(ctx context.Context, userID int64, offset, limit int) ([]*Session, bool, error) {
	sessions, err := m.store.ListByUser(ctx, userID, offset, limit)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list sessions: %w", err)
	}

	// Check if there are more sessions
	total, err := m.store.CountByUser(ctx, userID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to count sessions: %w", err)
	}

	hasMore := offset+limit < total
	return sessions, hasMore, nil
}

// SwitchSession changes the active session for a user
func (m *Manager) SwitchSession(ctx context.Context, userID int64, sessionID uuid.UUID) (*Session, error) {
	// Verify ownership
	session, err := m.store.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session.UserID != userID {
		return nil, ErrUnauthorized
	}

	// Set as active
	if err := m.store.SetActiveSession(ctx, userID, sessionID); err != nil {
		return nil, fmt.Errorf("failed to set active session: %w", err)
	}

	return session, nil
}

// CreateSession creates a new session from a user message
func (m *Manager) CreateSession(ctx context.Context, userID int64, message string) (*Session, error) {
	session := NewSession(userID, message)

	if err := m.store.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set as active session
	if err := m.store.SetActiveSession(ctx, userID, session.ID); err != nil {
		return nil, fmt.Errorf("failed to set active session: %w", err)
	}

	return session, nil
}

// GetOrCreateActiveSession returns the active session or creates a new one
func (m *Manager) GetOrCreateActiveSession(ctx context.Context, userID int64, message string) (*Session, error) {
	session, err := m.store.GetActiveSession(ctx, userID)
	if err == nil {
		return session, nil
	}

	// No active session, create new one
	return m.CreateSession(ctx, userID, message)
}
