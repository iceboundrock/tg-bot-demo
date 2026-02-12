package session

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store interface using SQLite
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath)
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

// Close closes the database connection
func (s *SQLiteStore) Close() error {
	return s.db.Close()
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

// ClearActiveSession removes the current active session for a user.
func (s *SQLiteStore) ClearActiveSession(ctx context.Context, userID int64) error {
	query := `DELETE FROM active_sessions WHERE user_id = ?`

	if _, err := s.db.ExecContext(ctx, query, userID); err != nil {
		return fmt.Errorf("failed to clear active session: %w", err)
	}

	return nil
}
