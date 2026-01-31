package scrobbler

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Queue manages a persistent queue of scrobbles using SQLite
type Queue struct {
	db *sql.DB
}

// QueuedScrobble represents a scrobble in the queue
type QueuedScrobble struct {
	ID        int64
	TrackName string
	Artist    string
	Album     string
	Duration  time.Duration
	Timestamp time.Time
	Scrobbled bool
	Error     string
}

// NewQueue creates a new scrobble queue backed by SQLite
func NewQueue(dbPath string) (*Queue, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool size to 1 for in-memory databases to ensure consistency
	// For file-based databases, this still works well for our use case
	db.SetMaxOpenConns(1)

	// Configure SQLite for optimal performance and safety
	pragmas := []string{
		"PRAGMA foreign_keys = ON",           // Enforce foreign key constraints
		"PRAGMA busy_timeout = 10000",        // Wait up to 10 seconds on lock
		"PRAGMA synchronous = NORMAL",        // Balance between safety and performance
		"PRAGMA journal_mode = WAL",          // Write-Ahead Logging for concurrent access
		"PRAGMA temp_store = MEMORY",         // Use memory for temp tables
		"PRAGMA cache_size = -64000",         // 64MB cache
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma: %w", err)
		}
	}

	// Create the schema
	schema := `
		CREATE TABLE IF NOT EXISTS scrobbles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			track_name TEXT NOT NULL,
			artist TEXT NOT NULL,
			album TEXT,
			duration INTEGER NOT NULL,
			timestamp INTEGER NOT NULL,
			scrobbled BOOLEAN DEFAULT 0,
			error TEXT,
			created_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now'))
		);

		CREATE INDEX IF NOT EXISTS idx_scrobbled ON scrobbles(scrobbled, timestamp);
		CREATE INDEX IF NOT EXISTS idx_timestamp ON scrobbles(timestamp);
	`

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &Queue{db: db}, nil
}

// Close closes the database connection
func (q *Queue) Close() error {
	if q.db != nil {
		return q.db.Close()
	}
	return nil
}

// Add adds a new scrobble to the queue
func (q *Queue) Add(ctx context.Context, scrobble Scrobble) (int64, error) {
	query := `
		INSERT INTO scrobbles (track_name, artist, album, duration, timestamp)
		VALUES (?, ?, ?, ?, ?)
	`

	result, err := q.db.ExecContext(ctx, query,
		scrobble.Track,
		scrobble.Artist,
		scrobble.Album,
		int64(scrobble.Duration.Seconds()),
		scrobble.Timestamp.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert scrobble: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get insert id: %w", err)
	}

	return id, nil
}

// MarkScrobbled marks a scrobble as successfully scrobbled
func (q *Queue) MarkScrobbled(ctx context.Context, id int64) error {
	query := `
		UPDATE scrobbles
		SET scrobbled = 1, error = NULL
		WHERE id = ?
	`

	result, err := q.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to mark scrobble as scrobbled: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("scrobble with id %d not found", id)
	}

	return nil
}

// MarkScrobbledBatch marks multiple scrobbles as successfully scrobbled
func (q *Queue) MarkScrobbledBatch(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	tx, err := q.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, "UPDATE scrobbles SET scrobbled = 1, error = NULL WHERE id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, id); err != nil {
			return fmt.Errorf("failed to mark scrobble %d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// MarkError marks a scrobble as failed with an error message
func (q *Queue) MarkError(ctx context.Context, id int64, errMsg string) error {
	query := `
		UPDATE scrobbles
		SET error = ?
		WHERE id = ?
	`

	result, err := q.db.ExecContext(ctx, query, errMsg, id)
	if err != nil {
		return fmt.Errorf("failed to mark scrobble error: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("scrobble with id %d not found", id)
	}

	return nil
}

// GetPending retrieves all pending (unscrobbled) scrobbles, ordered by timestamp
// Optionally limits the number of results
func (q *Queue) GetPending(ctx context.Context, limit int) ([]QueuedScrobble, error) {
	query := `
		SELECT id, track_name, artist, album, duration, timestamp, scrobbled, COALESCE(error, '')
		FROM scrobbles
		WHERE scrobbled = 0
		ORDER BY timestamp ASC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending scrobbles: %w", err)
	}
	defer rows.Close()

	var scrobbles []QueuedScrobble
	for rows.Next() {
		var s QueuedScrobble
		var durationSecs int64
		var timestampUnix int64

		err := rows.Scan(
			&s.ID,
			&s.TrackName,
			&s.Artist,
			&s.Album,
			&durationSecs,
			&timestampUnix,
			&s.Scrobbled,
			&s.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scrobble: %w", err)
		}

		s.Duration = time.Duration(durationSecs) * time.Second
		s.Timestamp = time.Unix(timestampUnix, 0)

		scrobbles = append(scrobbles, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scrobbles: %w", err)
	}

	return scrobbles, nil
}

// GetAll retrieves all scrobbles (for debugging/testing)
func (q *Queue) GetAll(ctx context.Context) ([]QueuedScrobble, error) {
	query := `
		SELECT id, track_name, artist, album, duration, timestamp, scrobbled, COALESCE(error, '')
		FROM scrobbles
		ORDER BY timestamp DESC
	`

	rows, err := q.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all scrobbles: %w", err)
	}
	defer rows.Close()

	var scrobbles []QueuedScrobble
	for rows.Next() {
		var s QueuedScrobble
		var durationSecs int64
		var timestampUnix int64

		err := rows.Scan(
			&s.ID,
			&s.TrackName,
			&s.Artist,
			&s.Album,
			&durationSecs,
			&timestampUnix,
			&s.Scrobbled,
			&s.Error,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scrobble: %w", err)
		}

		s.Duration = time.Duration(durationSecs) * time.Second
		s.Timestamp = time.Unix(timestampUnix, 0)

		scrobbles = append(scrobbles, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating scrobbles: %w", err)
	}

	return scrobbles, nil
}

// Cleanup removes old scrobbled records to prevent unbounded growth
// Keeps scrobbles newer than the given age, and always keeps unscrobbled ones
func (q *Queue) Cleanup(ctx context.Context, maxAge time.Duration) (int64, error) {
	cutoff := time.Now().Add(-maxAge).Unix()

	query := `
		DELETE FROM scrobbles
		WHERE scrobbled = 1
		AND timestamp < ?
	`

	result, err := q.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old scrobbles: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return deleted, nil
}

// CleanupOldFailed removes old failed scrobbles that are beyond Last.fm's 2-week limit
func (q *Queue) CleanupOldFailed(ctx context.Context) (int64, error) {
	// Last.fm won't accept scrobbles older than 2 weeks
	twoWeeksAgo := time.Now().Add(-14 * 24 * time.Hour).Unix()

	query := `
		DELETE FROM scrobbles
		WHERE scrobbled = 0
		AND error IS NOT NULL
		AND timestamp < ?
	`

	result, err := q.db.ExecContext(ctx, query, twoWeeksAgo)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old failed scrobbles: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return deleted, nil
}

// Count returns the number of scrobbles in the queue
// If includeScrobbled is false, only counts pending scrobbles
func (q *Queue) Count(ctx context.Context, includeScrobbled bool) (int, error) {
	query := "SELECT COUNT(*) FROM scrobbles"
	if !includeScrobbled {
		query += " WHERE scrobbled = 0"
	}

	var count int
	err := q.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count scrobbles: %w", err)
	}

	return count, nil
}
