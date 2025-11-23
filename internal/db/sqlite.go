package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store wraps the SQL DB and exposes operations for todos.
type Store struct {
	SQL *sql.DB
}

// NewStore opens (or creates) the SQLite database at the given path and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errors.New("db path must not be empty")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Busy timeout to reduce lock contention errors
	if _, err := db.Exec("PRAGMA busy_timeout = 5000;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	// WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		slog.Warn("failed to enable WAL mode", "error", err)
	}

	store := &Store{SQL: db}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

// Close closes the underlying SQL DB.
func (s *Store) Close() error {
	if s == nil || s.SQL == nil {
		return nil
	}
	return s.SQL.Close()
}

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			completed INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_todos_completed ON todos(completed);`,
	}
	for _, stmt := range stmts {
		if _, err := s.SQL.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

// Todo represents a todo item.
type Todo struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ListTodos returns all todos ordered by created_at ascending.
func (s *Store) ListTodos(ctx context.Context) ([]Todo, error) {
	rows, err := s.SQL.QueryContext(ctx, `SELECT id, title, completed, created_at, updated_at FROM todos ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Todo
	for rows.Next() {
		var t Todo
		
		var created, updated string
		var completedInt int
		if err := rows.Scan(&t.ID, &t.Title, &completedInt, &created, &updated); err != nil {
			return nil, err
		}
		t.Completed = completedInt == 1
		ct, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, err
		}
		ut, err := time.Parse(time.RFC3339Nano, updated)
		if err != nil {
			return nil, err
		}
		t.CreatedAt = ct
		t.UpdatedAt = ut
		out = append(out, t)
	}
	if out == nil {
		out = []Todo{}
	}
	return out, rows.Err()
}

// CreateTodo creates a new todo with the given title.
func (s *Store) CreateTodo(ctx context.Context, title string) (Todo, error) {
	if len(title) == 0 {
		return Todo{}, errors.New("title must not be empty")
	}
	if len(title) > 200 {
		return Todo{}, errors.New("title too long")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := s.SQL.ExecContext(ctx,
		`INSERT INTO todos (title, completed, created_at, updated_at) VALUES (?, 0, ?, ?)`,
		title, now, now,
	)
	if err != nil {
		return Todo{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Todo{}, err
	}
	// Log creation
	slog.Info("todo.created", "id", id, "title", title)
	return Todo{
		ID:        id,
		Title:     title,
		Completed: false,
		CreatedAt: mustParseRFC3339Nano(now),
		UpdatedAt: mustParseRFC3339Nano(now),
	}, nil
}

// UpdateTodo updates title and completed fields for a todo by id.
func (s *Store) UpdateTodo(ctx context.Context, id int64, title string, completed bool) (Todo, error) {
	if len(title) == 0 {
		return Todo{}, errors.New("title must not be empty")
	}
	if len(title) > 200 {
		return Todo{}, errors.New("title too long")
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	completedInt := 0
	if completed {
		completedInt = 1
	}
	_, err := s.SQL.ExecContext(ctx,
		`UPDATE todos SET title = ?, completed = ?, updated_at = ? WHERE id = ?`,
		title, completedInt, now, id,
	)
	if err != nil {
		return Todo{}, err
	}
	updated, err := s.GetTodo(ctx, id)
	if err == nil {
		slog.Info("todo.updated", "id", updated.ID, "title", updated.Title, "completed", updated.Completed)
	}
	return updated, err
}

// DeleteTodo deletes a todo by id.
func (s *Store) DeleteTodo(ctx context.Context, id int64) error {
	res, err := s.SQL.ExecContext(ctx, `DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, err := res.RowsAffected(); err == nil {
		if n > 0 {
			slog.Info("todo.deleted", "id", id, "rows", n)
		} else {
			slog.Warn("todo.delete.miss", "id", id)
		}
	}
	return nil
}

// GetTodo returns a todo by id.
func (s *Store) GetTodo(ctx context.Context, id int64) (Todo, error) {
	var t Todo
	var created, updated string
	var completedInt int
	row := s.SQL.QueryRowContext(ctx,
		`SELECT id, title, completed, created_at, updated_at FROM todos WHERE id = ?`, id,
	)
	if err := row.Scan(&t.ID, &t.Title, &completedInt, &created, &updated); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Todo{}, sql.ErrNoRows
		}
		return Todo{}, err
	}
	t.Completed = completedInt == 1
	ct, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Todo{}, err
	}
	ut, err := time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return Todo{}, err
	}
	t.CreatedAt = ct
	t.UpdatedAt = ut
	return t, nil
}

func mustParseRFC3339Nano(v string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		panic(err)
	}
	return t
}


