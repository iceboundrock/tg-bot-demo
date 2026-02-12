# Design Document: Telegram Session List

## Overview

This design document describes the implementation of a session list feature for a Telegram Bot built in Go using the `github.com/go-telegram/bot` library. The feature enables users to view, paginate, and switch between their AI conversation sessions through Telegram's Inline Keyboard interface.

The system follows a layered architecture with clear separation between:
- Command handling and Telegram API interaction
- Business logic for session management
- Data persistence layer

The implementation uses SQLite for data persistence, providing reliable storage with ACID guarantees while maintaining simplicity for deployment.

## Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Telegram Platform                     │
└────────────────────┬────────────────────────────────────┘
                     │ Webhook/Updates
                     ▼
┌─────────────────────────────────────────────────────────┐
│                   Bot Handler Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │   Command    │  │   Callback   │  │   Message    │  │
│  │   Handler    │  │   Handler    │  │   Handler    │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                 Session Service Layer                    │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Session Manager                                  │   │
│  │  - List sessions                                  │   │
│  │  - Switch active session                          │   │
│  │  - Create session                                 │   │
│  │  - Generate titles                                │   │
│  └──────────────────────────────────────────────────┘   │
└────────────────────┬────────────────────────────────────┘
                     │
                     ▼
┌─────────────────────────────────────────────────────────┐
│                  Storage Layer                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │  Session Store Interface                          │   │
│  │  - Create/Read/Update/Delete                      │   │
│  │  - Query with pagination                          │   │
│  │  - Filter by user                                 │   │
│  └──────────────────────────────────────────────────┘   │
│                                                           │
│  ┌──────────────────────────────────────────────────┐   │
│  │  SQLite Implementation                            │   │
│  │  - ACID transactions                              │   │
│  │  - Indexed queries                                │   │
│  │  - Connection pooling                             │   │
│  └──────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────┘
```

### Component Interaction Flow

1. **Session List Request Flow**:
   ```
   User sends /sessions → Command Handler → Session Manager
   → Session Store (query) → Format response with Inline Keyboard
   → Send to Telegram API
   ```

2. **Pagination Flow**:
   ```
   User clicks "More" → Callback Handler → Parse offset
   → Session Manager → Session Store (query with offset)
   → Update message with new keyboard → Telegram API
   ```

3. **Session Switch Flow**:
   ```
   User clicks session button → Callback Handler → Parse session ID
   → Verify ownership → Session Manager (set active)
   → Send confirmation → Telegram API
   ```

## Components and Interfaces

### 1. Session Model

```go
package session

import (
    "time"
    "github.com/google/uuid"
)

// Session represents a conversation session between a user and AI
type Session struct {
    ID          uuid.UUID  `json:"id"`
    UserID      int64      `json:"user_id"`
    Title       string     `json:"title"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    LastMessage string     `json:"last_message"`
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
    // Implementation details in helper functions section
}
```

### 2. Session Store Interface

```go
package session

import (
    "context"
    "github.com/google/uuid"
)

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
```

### 3. SQLite Store Implementation

```go
package session

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "github.com/google/uuid"
    _ "modernc.org/sqlite"
)

var (
    ErrSessionNotFound = errors.New("session not found")
    ErrUnauthorized    = errors.New("unauthorized access to session")
)

// SQLiteStore implements Store interface using SQLite
type SQLiteStore struct {
    db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
    db, err := sql.Open("sqlite3", dbPath)
    if err != nil {
        return nil, fmt.Errorf("failed to open database: %w", err)
    }
    
    // Enable WAL mode for better concurrency
    if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
        return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
    }
    
    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
        return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
    }
    
    store := &SQLiteStore{db: db}
    
    // Initialize schema
    if err := store.initSchema(); err != nil {
        return nil, fmt.Errorf("failed to initialize schema: %w", err)
    }
    
    return store, nil
}

// initSchema creates the necessary tables and indexes
func (s *SQLiteStore) initSchema() error {
    schema := `
    CREATE TABLE IF NOT EXISTS sessions (
        id TEXT PRIMARY KEY,
        user_id INTEGER NOT NULL,
        title TEXT NOT NULL,
        created_at DATETIME NOT NULL,
        updated_at DATETIME NOT NULL,
        last_message TEXT NOT NULL
    );
    
    CREATE INDEX IF NOT EXISTS idx_sessions_user_updated 
        ON sessions(user_id, updated_at DESC);
    
    CREATE TABLE IF NOT EXISTS active_sessions (
        user_id INTEGER PRIMARY KEY,
        session_id TEXT NOT NULL,
        FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
    );
    
    CREATE INDEX IF NOT EXISTS idx_active_sessions_user 
        ON active_sessions(user_id);
    `
    
    _, err := s.db.Exec(schema)
    return err
}

// Create stores a new session
func (s *SQLiteStore) Create(ctx context.Context, session *Session) error {
    query := `
        INSERT INTO sessions (id, user_id, title, created_at, updated_at, last_message)
        VALUES (?, ?, ?, ?, ?, ?)
    `
    
    _, err := s.db.ExecContext(ctx, query,
        session.ID.String(),
        session.UserID,
        session.Title,
        session.CreatedAt,
        session.UpdatedAt,
        session.LastMessage,
    )
    
    if err != nil {
        return fmt.Errorf("failed to create session: %w", err)
    }
    
    return nil
}

// Get retrieves a session by ID
func (s *SQLiteStore) Get(ctx context.Context, id uuid.UUID) (*Session, error) {
    query := `
        SELECT id, user_id, title, created_at, updated_at, last_message
        FROM sessions
        WHERE id = ?
    `
    
    var session Session
    var idStr string
    
    err := s.db.QueryRowContext(ctx, query, id.String()).Scan(
        &idStr,
        &session.UserID,
        &session.Title,
        &session.CreatedAt,
        &session.UpdatedAt,
        &session.LastMessage,
    )
    
    if err == sql.ErrNoRows {
        return nil, ErrSessionNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get session: %w", err)
    }
    
    session.ID, err = uuid.Parse(idStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse session ID: %w", err)
    }
    
    return &session, nil
}

// Update modifies an existing session
func (s *SQLiteStore) Update(ctx context.Context, session *Session) error {
    query := `
        UPDATE sessions
        SET title = ?, updated_at = ?, last_message = ?
        WHERE id = ?
    `
    
    result, err := s.db.ExecContext(ctx, query,
        session.Title,
        session.UpdatedAt,
        session.LastMessage,
        session.ID.String(),
    )
    
    if err != nil {
        return fmt.Errorf("failed to update session: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return ErrSessionNotFound
    }
    
    return nil
}

// Delete removes a session
func (s *SQLiteStore) Delete(ctx context.Context, id uuid.UUID) error {
    query := `DELETE FROM sessions WHERE id = ?`
    
    result, err := s.db.ExecContext(ctx, query, id.String())
    if err != nil {
        return fmt.Errorf("failed to delete session: %w", err)
    }
    
    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("failed to get rows affected: %w", err)
    }
    
    if rows == 0 {
        return ErrSessionNotFound
    }
    
    return nil
}

// ListByUser returns sessions for a specific user with pagination
func (s *SQLiteStore) ListByUser(ctx context.Context, userID int64, offset, limit int) ([]*Session, error) {
    query := `
        SELECT id, user_id, title, created_at, updated_at, last_message
        FROM sessions
        WHERE user_id = ?
        ORDER BY updated_at DESC
        LIMIT ? OFFSET ?
    `
    
    rows, err := s.db.QueryContext(ctx, query, userID, limit, offset)
    if err != nil {
        return nil, fmt.Errorf("failed to list sessions: %w", err)
    }
    defer rows.Close()
    
    var sessions []*Session
    
    for rows.Next() {
        var session Session
        var idStr string
        
        err := rows.Scan(
            &idStr,
            &session.UserID,
            &session.Title,
            &session.CreatedAt,
            &session.UpdatedAt,
            &session.LastMessage,
        )
        
        if err != nil {
            return nil, fmt.Errorf("failed to scan session: %w", err)
        }
        
        session.ID, err = uuid.Parse(idStr)
        if err != nil {
            return nil, fmt.Errorf("failed to parse session ID: %w", err)
        }
        
        sessions = append(sessions, &session)
    }
    
    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("error iterating sessions: %w", err)
    }
    
    return sessions, nil
}

// CountByUser returns total number of sessions for a user
func (s *SQLiteStore) CountByUser(ctx context.Context, userID int64) (int, error) {
    query := `SELECT COUNT(*) FROM sessions WHERE user_id = ?`
    
    var count int
    err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
    if err != nil {
        return 0, fmt.Errorf("failed to count sessions: %w", err)
    }
    
    return count, nil
}

// GetActiveSession returns the current active session for a user
func (s *SQLiteStore) GetActiveSession(ctx context.Context, userID int64) (*Session, error) {
    query := `
        SELECT s.id, s.user_id, s.title, s.created_at, s.updated_at, s.last_message
        FROM sessions s
        INNER JOIN active_sessions a ON s.id = a.session_id
        WHERE a.user_id = ?
    `
    
    var session Session
    var idStr string
    
    err := s.db.QueryRowContext(ctx, query, userID).Scan(
        &idStr,
        &session.UserID,
        &session.Title,
        &session.CreatedAt,
        &session.UpdatedAt,
        &session.LastMessage,
    )
    
    if err == sql.ErrNoRows {
        return nil, ErrSessionNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get active session: %w", err)
    }
    
    session.ID, err = uuid.Parse(idStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse session ID: %w", err)
    }
    
    return &session, nil
}

// SetActiveSession sets the active session for a user
func (s *SQLiteStore) SetActiveSession(ctx context.Context, userID int64, sessionID uuid.UUID) error {
    query := `
        INSERT INTO active_sessions (user_id, session_id)
        VALUES (?, ?)
        ON CONFLICT(user_id) DO UPDATE SET session_id = excluded.session_id
    `
    
    _, err := s.db.ExecContext(ctx, query, userID, sessionID.String())
    if err != nil {
        return fmt.Errorf("failed to set active session: %w", err)
    }
    
    return nil
}

// Close closes the database connection
func (s *SQLiteStore) Close() error {
    return s.db.Close()
}
```

### 4. Session Manager

```go
package session

import (
    "context"
    "fmt"
    "github.com/google/uuid"
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
```

### 5. Command Handler

```go
package handlers

import (
    "context"
    "fmt"
    "github.com/go-telegram/bot"
    "github.com/go-telegram/bot/models"
    "your-project/session"
)

const (
    SessionsPerPage = 6
)

// SessionsCommandHandler handles the /sessions command
func SessionsCommandHandler(sessionMgr *session.Manager) bot.HandlerFunc {
    return func(ctx context.Context, b *bot.Bot, update *models.Update) {
        userID := update.Message.From.ID
        
        // Get first page of sessions
        sessions, hasMore, err := sessionMgr.ListSessions(ctx, userID, 0, SessionsPerPage)
        if err != nil {
            b.SendMessage(ctx, &bot.SendMessageParams{
                ChatID: update.Message.Chat.ID,
                Text:   "Failed to retrieve sessions. Please try again.",
            })
            return
        }
        
        // Handle empty sessions
        if len(sessions) == 0 {
            b.SendMessage(ctx, &bot.SendMessageParams{
                ChatID: update.Message.Chat.ID,
                Text:   "You don't have any sessions yet. Start chatting to create one!",
            })
            return
        }
        
        // Build inline keyboard
        keyboard := buildSessionKeyboard(sessions, 0, hasMore)
        
        b.SendMessage(ctx, &bot.SendMessageParams{
            ChatID:      update.Message.Chat.ID,
            Text:        "Your sessions:",
            ReplyMarkup: keyboard,
        })
    }
}

// buildSessionKeyboard creates an inline keyboard for session list
func buildSessionKeyboard(sessions []*session.Session, offset int, hasMore bool) *models.InlineKeyboardMarkup {
    var rows [][]models.InlineKeyboardButton
    
    // Add session buttons (one per row)
    for _, s := range sessions {
        button := models.InlineKeyboardButton{
            Text:         formatSessionButton(s),
            CallbackData: fmt.Sprintf("open_s_%s", s.ID.String()),
        }
        rows = append(rows, []models.InlineKeyboardButton{button})
    }
    
    // Add "More" button if needed
    if hasMore {
        moreButton := models.InlineKeyboardButton{
            Text:         "📄 More...",
            CallbackData: fmt.Sprintf("more_sessions_%d", offset+SessionsPerPage),
        }
        rows = append(rows, []models.InlineKeyboardButton{moreButton})
    }
    
    return &models.InlineKeyboardMarkup{
        InlineKeyboard: rows,
    }
}

// formatSessionButton formats a session for display in button
func formatSessionButton(s *session.Session) string {
    // Format: "Title - 2h ago"
    timeAgo := formatTimeAgo(s.UpdatedAt)
    return fmt.Sprintf("%s - %s", truncate(s.Title, 40), timeAgo)
}
```

### 6. Callback Handler

```go
package handlers

import (
    "context"
    "fmt"
    "strconv"
    "strings"
    "github.com/go-telegram/bot"
    "github.com/go-telegram/bot/models"
    "github.com/google/uuid"
    "your-project/session"
)

// CallbackQueryHandler handles inline keyboard button clicks
func CallbackQueryHandler(sessionMgr *session.Manager) bot.HandlerFunc {
    return func(ctx context.Context, b *bot.Bot, update *models.Update) {
        callback := update.CallbackQuery
        userID := callback.From.ID
        data := callback.Data
        
        // Answer callback immediately
        b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
            CallbackQueryID: callback.ID,
        })
        
        // Route based on callback data prefix
        switch {
        case strings.HasPrefix(data, "open_s_"):
            handleOpenSession(ctx, b, callback, sessionMgr, userID, data)
            
        case strings.HasPrefix(data, "more_sessions_"):
            handleMoreSessions(ctx, b, callback, sessionMgr, userID, data)
            
        default:
            // Invalid callback data, log warning
            fmt.Printf("Warning: invalid callback data: %s\n", data)
        }
    }
}

// handleOpenSession processes session switch requests
func handleOpenSession(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery, 
                       sessionMgr *session.Manager, userID int64, data string) {
    // Parse session ID
    sessionIDStr := strings.TrimPrefix(data, "open_s_")
    sessionID, err := uuid.Parse(sessionIDStr)
    if err != nil {
        b.SendMessage(ctx, &bot.SendMessageParams{
            ChatID: callback.Message.Chat.ID,
            Text:   "Invalid session ID.",
        })
        return
    }
    
    // Switch session
    session, err := sessionMgr.SwitchSession(ctx, userID, sessionID)
    if err != nil {
        if err == session.ErrUnauthorized {
            b.SendMessage(ctx, &bot.SendMessageParams{
                ChatID: callback.Message.Chat.ID,
                Text:   "You don't have permission to access this session.",
            })
            return
        }
        
        b.SendMessage(ctx, &bot.SendMessageParams{
            ChatID: callback.Message.Chat.ID,
            Text:   "Failed to switch session. Please try again.",
        })
        return
    }
    
    // Send confirmation
    b.SendMessage(ctx, &bot.SendMessageParams{
        ChatID: callback.Message.Chat.ID,
        Text:   fmt.Sprintf("✅ Switched to session: %s", session.Title),
    })
}

// handleMoreSessions processes pagination requests
func handleMoreSessions(ctx context.Context, b *bot.Bot, callback *models.CallbackQuery,
                        sessionMgr *session.Manager, userID int64, data string) {
    // Parse offset
    offsetStr := strings.TrimPrefix(data, "more_sessions_")
    offset, err := strconv.Atoi(offsetStr)
    if err != nil {
        return
    }
    
    // Get next page
    sessions, hasMore, err := sessionMgr.ListSessions(ctx, userID, offset, SessionsPerPage)
    if err != nil {
        return
    }
    
    // Update message with new keyboard
    keyboard := buildSessionKeyboard(sessions, offset, hasMore)
    
    b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
        ChatID:      callback.Message.Chat.ID,
        MessageID:   callback.Message.ID,
        ReplyMarkup: keyboard,
    })
}
```

## Data Models

### Session Structure

```go
type Session struct {
    ID          uuid.UUID  // Unique identifier (UUID v4)
    UserID      int64      // Telegram user ID
    Title       string     // Session title (max 100 chars)
    CreatedAt   time.Time  // Creation timestamp
    UpdatedAt   time.Time  // Last update timestamp
    LastMessage string     // Last message content (max 500 chars)
}
```

### In-Memory Storage Structure

```go
// Database Schema (SQLite)

// sessions table
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,              -- UUID as string
    user_id INTEGER NOT NULL,         -- Telegram user ID
    title TEXT NOT NULL,              -- Session title (max 100 chars)
    created_at DATETIME NOT NULL,     -- Creation timestamp
    updated_at DATETIME NOT NULL,     -- Last update timestamp
    last_message TEXT NOT NULL        -- Last message content (max 500 chars)
);

CREATE INDEX idx_sessions_user_updated ON sessions(user_id, updated_at DESC);

// active_sessions table
CREATE TABLE active_sessions (
    user_id INTEGER PRIMARY KEY,      -- Telegram user ID
    session_id TEXT NOT NULL,         -- Current active session UUID
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE INDEX idx_active_sessions_user ON active_sessions(user_id);
```

### Callback Data Formats

1. **Session Button**: `open_s_{sessionID}`
   - Example: `open_s_550e8400-e29b-41d4-a716-446655440000`
   - Max length: 43 bytes (within 64-byte limit)

2. **Pagination Button**: `more_sessions_{offset}`
   - Example: `more_sessions_6`
   - Max length: ~20 bytes (within 64-byte limit)

## Helper Functions

### Title Generation

```go
package session

import (
    "fmt"
    "strings"
    "time"
    "unicode/utf8"
)

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
    if utf8.RuneCountInString(message) <= 10 {
        return message
    }
    
    if utf8.RuneCountInString(message) > 30 {
        runes := []rune(message)
        return string(runes[:30]) + "..."
    }
    
    return message
}
```

### Time Formatting

```go
package handlers

import (
    "fmt"
    "time"
)

// formatTimeAgo converts a timestamp to relative time string
func formatTimeAgo(t time.Time) string {
    duration := time.Since(t)
    
    switch {
    case duration < time.Minute:
        return "just now"
    case duration < time.Hour:
        mins := int(duration.Minutes())
        return fmt.Sprintf("%dm ago", mins)
    case duration < 24*time.Hour:
        hours := int(duration.Hours())
        return fmt.Sprintf("%dh ago", hours)
    case duration < 7*24*time.Hour:
        days := int(duration.Hours() / 24)
        return fmt.Sprintf("%dd ago", days)
    default:
        return t.Format("Jan 2")
    }
}

// truncate limits string length
func truncate(s string, maxLen int) string {
    if utf8.RuneCountInString(s) <= maxLen {
        return s
    }
    runes := []rune(s)
    return string(runes[:maxLen-3]) + "..."
}
```

### Sorting Sessions

```go
package session

import (
    "sort"
)

// sortSessionsByUpdatedAt sorts sessions by UpdatedAt in descending order
func sortSessionsByUpdatedAt(sessions []*Session) {
    sort.Slice(sessions, func(i, j int) bool {
        return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
    })
}
```

## Correctness Properties

A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.


### Property 1: Session List Returns User's Sessions Only

*For any* user with sessions in the system, querying the session list should return only sessions belonging to that user, and no sessions from other users.

**Validates: Requirements 1.1, 5.3, 6.3**

### Property 2: Session Display Limit

*For any* user, when displaying sessions in the Inline Keyboard, the system should show at most 6 sessions at a time, regardless of how many total sessions exist.

**Validates: Requirements 1.2**

### Property 3: More Button Presence

*For any* user with more than 6 sessions, the session list keyboard should include a "More" button; for users with 6 or fewer sessions, the "More" button should not be present.

**Validates: Requirements 1.3, 2.3**

### Property 4: Session Display Format

*For any* session displayed in the keyboard, the button text should contain both the session title and a formatted time string representing the last update time.

**Validates: Requirements 1.5**

### Property 5: Pagination Returns Correct Subset

*For any* user with sessions and any valid offset, requesting the next page should return the correct subset of sessions starting at that offset, limited to 6 sessions.

**Validates: Requirements 2.1, 5.5**

### Property 6: Sessions Sorted by Update Time

*For any* query result (any page, any user), the returned sessions should be sorted by updated_at in descending order (most recent first).

**Validates: Requirements 2.4, 5.4**

### Property 7: Active Session Update

*For any* user and any session belonging to that user, switching to that session should result in that session becoming the user's active session.

**Validates: Requirements 3.1**

### Property 8: Switch Confirmation Contains Title

*For any* successful session switch, the confirmation message should contain the title of the newly active session.

**Validates: Requirements 3.2**

### Property 9: Messages Route to Active Session

*For any* user with an active session, when a new message is sent, that message should be associated with the active session.

**Validates: Requirements 3.3**

### Property 10: Auto-Create Session When None Active

*For any* user without an active session, when a message is sent, a new session should be automatically created and set as active.

**Validates: Requirements 3.4**

### Property 11: Session Button Callback Format

*For any* session, the callback_data for its button should match the format "open_s_{sessionID}" where sessionID is the session's UUID string.

**Validates: Requirements 4.1**

### Property 12: Pagination Button Callback Format

*For any* pagination button with offset N, the callback_data should match the format "more_sessions_{N}".

**Validates: Requirements 4.2**

### Property 13: Callback Data Length Constraint

*For any* callback_data generated by the system (session buttons, pagination buttons), the byte length should not exceed 64 bytes.

**Validates: Requirements 4.3**

### Property 14: Session ID Uniqueness

*For any* set of sessions created by the system, all session IDs should be unique valid UUIDs.

**Validates: Requirements 5.1**

### Property 15: Session Data Round-Trip

*For any* session created with specific field values (user_id, title, last_message), storing and then retrieving that session should return a session with equivalent field values.

**Validates: Requirements 5.2**

### Property 16: Updated Timestamp Auto-Update

*For any* session, when it is updated, the updated_at timestamp should be greater than or equal to the previous updated_at value.

**Validates: Requirements 5.6**

### Property 17: Session Ownership Verification

*For any* attempt to access a session, if the requesting user's ID does not match the session's user_id, the operation should be rejected with an authorization error.

**Validates: Requirements 6.1, 6.2**

### Property 18: Concurrent Updates Use Last-Write-Wins

*For any* session that receives concurrent update operations, the final state should reflect the last completed write operation.

**Validates: Requirements 7.4**

### Property 19: Title Generation from Message

*For any* message used to create a session, if the message length is greater than 30 characters, the title should be the first 30 characters; if the message is between 10 and 30 characters, the title should be the complete message; if less than 10 characters, the title should be the complete message.

**Validates: Requirements 8.1, 8.2**

### Property 20: Newline Replacement in Titles

*For any* message containing newline characters, when used to generate a session title, all newline characters should be replaced with spaces in the resulting title.

**Validates: Requirements 8.4**

## Error Handling

### Error Types

The system defines the following error types:

```go
var (
    ErrSessionNotFound = errors.New("session not found")
    ErrUnauthorized    = errors.New("unauthorized access to session")
    ErrInvalidUUID     = errors.New("invalid session ID format")
    ErrStorageFailure  = errors.New("storage operation failed")
)
```

### Error Handling Strategy

1. **Session Not Found**: Return friendly error message to user
   - User-facing: "Session not found. It may have been deleted."
   - Log: Warning level with session ID

2. **Unauthorized Access**: Return error and reject operation
   - User-facing: "You don't have permission to access this session."
   - Log: Warning level with user ID and session ID

3. **Invalid Callback Data**: Ignore silently, log warning
   - User-facing: No message (callback answered silently)
   - Log: Warning level with callback data

4. **Storage Failures**: Return generic error, log details
   - User-facing: "An error occurred. Please try again."
   - Log: Error level with full error details

5. **Concurrent Conflicts**: Use last-write-wins strategy
   - No user-facing error
   - Log: Debug level with conflict details

### Error Response Format

```go
// ErrorResponse represents a user-facing error
type ErrorResponse struct {
    Message string
    Code    string
}

// Common error responses
var (
    ErrResponseNotFound = ErrorResponse{
        Message: "Session not found. It may have been deleted.",
        Code:    "SESSION_NOT_FOUND",
    }
    
    ErrResponseUnauthorized = ErrorResponse{
        Message: "You don't have permission to access this session.",
        Code:    "UNAUTHORIZED",
    }
    
    ErrResponseGeneric = ErrorResponse{
        Message: "An error occurred. Please try again.",
        Code:    "INTERNAL_ERROR",
    }
)
```

## Testing Strategy

### Dual Testing Approach

This feature requires both unit tests and property-based tests to ensure comprehensive coverage:

- **Unit tests**: Verify specific examples, edge cases, and error conditions
- **Property tests**: Verify universal properties across all inputs

Both testing approaches are complementary and necessary. Unit tests catch concrete bugs in specific scenarios, while property tests verify general correctness across a wide range of inputs.

### Property-Based Testing

We will use the `gopter` library for property-based testing in Go. Each property test should:

- Run a minimum of 100 iterations (due to randomization)
- Reference its corresponding design document property
- Use the tag format: `Feature: telegram-session-list, Property {number}: {property_text}`

Example property test structure:

```go
func TestProperty1_SessionListReturnsUserSessionsOnly(t *testing.T) {
    // Feature: telegram-session-list, Property 1: Session List Returns User's Sessions Only
    
    properties := gopter.NewProperties(nil)
    
    properties.Property("sessions filtered by user", prop.ForAll(
        func(userID int64, otherUserID int64, sessions []*Session) bool {
            // Test implementation
        },
        gen.Int64(),
        gen.Int64(),
        genSessions(),
    ))
    
    properties.TestingRun(t, gopter.ConsoleReporter(false))
}
```

### Unit Testing Focus

Unit tests should focus on:

1. **Specific Examples**:
   - User with exactly 6 sessions (boundary case)
   - User with 0 sessions (empty state)
   - Session with exactly 30 character title (boundary case)

2. **Edge Cases**:
   - Empty or whitespace-only messages (Requirement 8.3)
   - Non-existent session IDs (Requirement 7.1)
   - Invalid callback data formats (Requirement 7.3)

3. **Integration Points**:
   - Command handler integration with session manager
   - Callback handler integration with session manager
   - Store interface implementation correctness

4. **Error Conditions**:
   - Unauthorized access attempts
   - Malformed UUIDs
   - Storage operation failures

### Test Organization

```
session/
├── session.go
├── session_test.go          # Unit tests for session model
├── session_property_test.go # Property tests for session model
├── store.go
├── store_test.go            # Unit tests for store interface
├── store_property_test.go   # Property tests for store
├── manager.go
├── manager_test.go          # Unit tests for manager
├── manager_property_test.go # Property tests for manager

handlers/
├── commands.go
├── commands_test.go         # Unit tests for command handlers
├── callbacks.go
├── callbacks_test.go        # Unit tests for callback handlers
├── handlers_property_test.go # Property tests for handlers
```

### Test Data Generators

For property-based testing, we need generators for:

```go
// Generator for random sessions
func genSession() gopter.Gen {
    return gopter.CombineGens(
        gen.Int64(),           // UserID
        gen.AlphaString(),     // Title
        gen.AlphaString(),     // LastMessage
    ).Map(func(values []interface{}) *Session {
        return &Session{
            ID:          uuid.New(),
            UserID:      values[0].(int64),
            Title:       values[1].(string),
            LastMessage: values[2].(string),
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        }
    })
}

// Generator for random user IDs
func genUserID() gopter.Gen {
    return gen.Int64Range(1, 1000000)
}

// Generator for random messages
func genMessage() gopter.Gen {
    return gen.OneGenOf(
        gen.AlphaString(),                    // Normal messages
        gen.Const(""),                        // Empty messages
        gen.Const("   "),                     // Whitespace messages
        gen.RegexMatch("[a-z\n]+"),          // Messages with newlines
    )
}
```

### Coverage Goals

- Unit test coverage: >80% of lines
- Property test coverage: All 20 correctness properties
- Integration test coverage: All handler flows (command, callback, message)

### Continuous Integration

Tests should run on:
- Every commit (unit tests + property tests with 100 iterations)
- Nightly builds (property tests with 10,000 iterations for deeper coverage)

## Performance Considerations

### SQLite Store Performance

The SQLite implementation uses the following optimizations:

1. **WAL Mode**:
   - Write-Ahead Logging enabled for better concurrency
   - Allows concurrent reads while writing
   - Better performance for write-heavy workloads

2. **Index Strategy**:
   - Composite index on (user_id, updated_at DESC) for efficient session listing
   - Index on user_id in active_sessions for fast active session lookup
   - Primary key indexes on id fields

3. **Query Optimization**:
   - Prepared statements for repeated queries
   - Efficient pagination using LIMIT/OFFSET
   - Foreign key constraints for data integrity

4. **Connection Management**:
   - Single database connection (suitable for SQLite)
   - Context-based query cancellation
   - Proper connection cleanup

### Expected Performance

With SQLite storage:
- Session lookup: <5ms
- Session list query: <10ms (for typical user with <100 sessions)
- Session switch: <5ms
- Pagination: <10ms

These meet the requirements of:
- Query response: <500ms ✓
- Switch operation: <200ms ✓

### Database File Location

Recommended database file locations:
- Development: `./data/sessions.db`
- Production: `/var/lib/telegram-bot/sessions.db`
- Docker: `/data/sessions.db` (mounted volume)

### Backup Strategy

SQLite backup recommendations:
1. Use SQLite's backup API or `.backup` command
2. Schedule regular backups (e.g., daily)
3. Store backups in separate location
4. Test restore procedures regularly

## Deployment Considerations

### Configuration

```go
type Config struct {
    SessionsPerPage int           // Default: 6
    MaxTitleLength  int           // Default: 100
    MaxMessageCache int           // Default: 500
    DatabasePath    string        // SQLite database file path
}
```

### Monitoring

Key metrics to track:

1. **Usage Metrics**:
   - Sessions created per day
   - Average sessions per user
   - Session switches per day
   - Pagination usage rate

2. **Performance Metrics**:
   - Query latency (p50, p95, p99)
   - Switch operation latency
   - Store operation latency

3. **Error Metrics**:
   - Unauthorized access attempts
   - Session not found errors
   - Storage failures

### Logging

Log levels:

- **Debug**: Pagination operations, callback data parsing
- **Info**: Session creation, session switches, command usage
- **Warning**: Unauthorized access, invalid callbacks, session not found
- **Error**: Storage failures, unexpected errors

Log format:
```
[timestamp] [level] [component] message user_id=X session_id=Y
```

## Future Enhancements

Potential future improvements (not in current scope):

1. **Session Search**: Allow users to search sessions by title or content
2. **Session Deletion**: Allow users to delete old sessions
3. **Session Renaming**: Allow users to rename sessions
4. **Session Sharing**: Allow users to share sessions with others
5. **Session Export**: Export session history to file
6. **Session Statistics**: Show session usage statistics
7. **Session Archiving**: Archive old inactive sessions
8. **Rich Session Metadata**: Add tags, categories, or custom metadata

These features would require additional requirements, design work, and implementation.
