package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// Store wraps the SQL DB and exposes operations for todos.
type Store struct {
	SQL *sql.DB
}

// NewStore opens a PostgreSQL connection using the provided DSN and runs migrations.
// Example DSN: postgres://user:pass@host:5432/dbname?sslmode=disable
func NewStore(dsn string) (*Store, error) {
	if dsn == "" {
		return nil, errors.New("database dsn must not be empty")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	// Reasonable defaults for local dev
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
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
			id BIGSERIAL PRIMARY KEY,
			title TEXT NOT NULL,
			completed BOOLEAN NOT NULL DEFAULT FALSE,
			tags JSONB NOT NULL DEFAULT '[]'::jsonb,
			duration_minutes INTEGER NOT NULL DEFAULT 0,
			priority_score DOUBLE PRECISION NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,
		`ALTER TABLE todos ADD COLUMN IF NOT EXISTS tags JSONB NOT NULL DEFAULT '[]'::jsonb;`,
		`ALTER TABLE todos ADD COLUMN IF NOT EXISTS duration_minutes INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE todos ADD COLUMN IF NOT EXISTS priority_score DOUBLE PRECISION NOT NULL DEFAULT 0;`,
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
	ID              int64     `json:"id"`
	Title           string    `json:"title"`
	Completed       bool      `json:"completed"`
	Tags            []string  `json:"tags"`
	DurationMinutes int       `json:"durationMinutes"`
	PriorityScore   float64   `json:"priorityScore"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// SaveTodoInput represents the fields accepted for create/update operations.
type SaveTodoInput struct {
	Title           string
	Completed       bool
	Tags            []string
	DurationMinutes int
	PriorityScore   float64
}

// ListTodos returns all todos ordered by created_at ascending.
func (s *Store) ListTodos(ctx context.Context) ([]Todo, error) {
	rows, err := s.SQL.QueryContext(ctx, `SELECT id, title, completed, tags, duration_minutes, priority_score, created_at, updated_at FROM todos ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Todo
	for rows.Next() {
		t, err := scanTodo(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	if out == nil {
		out = []Todo{}
	}
	return out, rows.Err()
}

// CreateTodo creates a new todo.
func (s *Store) CreateTodo(ctx context.Context, input SaveTodoInput) (Todo, error) {
	if len(input.Title) == 0 {
		return Todo{}, errors.New("title must not be empty")
	}
	if len(input.Title) > 200 {
		return Todo{}, errors.New("title too long")
	}
	if input.DurationMinutes < 0 {
		return Todo{}, errors.New("duration must be >= 0")
	}

	tagsJSON, err := encodeTags(input.Tags)
	if err != nil {
		return Todo{}, err
	}

	row := s.SQL.QueryRowContext(ctx,
		`INSERT INTO todos (title, completed, tags, duration_minutes, priority_score)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, title, completed, tags, duration_minutes, priority_score, created_at, updated_at`,
		input.Title, input.Completed, tagsJSON, input.DurationMinutes, input.PriorityScore,
	)
	t, err := scanTodo(row)
	if err != nil {
		return Todo{}, err
	}
	slog.Info("todo.created", "id", t.ID, "title", t.Title)
	return t, nil
}

// UpdateTodo updates fields for a todo by id.
func (s *Store) UpdateTodo(ctx context.Context, id int64, input SaveTodoInput) (Todo, error) {
	if len(input.Title) == 0 {
		return Todo{}, errors.New("title must not be empty")
	}
	if len(input.Title) > 200 {
		return Todo{}, errors.New("title too long")
	}
	if input.DurationMinutes < 0 {
		return Todo{}, errors.New("duration must be >= 0")
	}

	tagsJSON, err := encodeTags(input.Tags)
	if err != nil {
		return Todo{}, err
	}

	row := s.SQL.QueryRowContext(ctx,
		`UPDATE todos
		 SET title = $1,
		     completed = $2,
		     tags = $3,
		     duration_minutes = $4,
		     priority_score = $5,
		     updated_at = NOW()
		 WHERE id = $6
		 RETURNING id, title, completed, tags, duration_minutes, priority_score, created_at, updated_at`,
		input.Title, input.Completed, tagsJSON, input.DurationMinutes, input.PriorityScore, id,
	)
	t, err := scanTodo(row)
	if err != nil {
		return Todo{}, err
	}
	slog.Info("todo.updated", "id", t.ID, "title", t.Title, "completed", t.Completed)
	return t, nil
}

// DeleteTodo deletes a todo by id.
func (s *Store) DeleteTodo(ctx context.Context, id int64) error {
	res, err := s.SQL.ExecContext(ctx, `DELETE FROM todos WHERE id = $1`, id)
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
	row := s.SQL.QueryRowContext(ctx,
		`SELECT id, title, completed, tags, duration_minutes, priority_score, created_at, updated_at FROM todos WHERE id = $1`, id,
	)
	t, err := scanTodo(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Todo{}, sql.ErrNoRows
		}
		return Todo{}, err
	}
	if t.Tags == nil {
		t.Tags = []string{}
	}
	return t, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTodo(row rowScanner) (Todo, error) {
	var t Todo
	var tagsRaw []byte
	if err := row.Scan(
		&t.ID,
		&t.Title,
		&t.Completed,
		&tagsRaw,
		&t.DurationMinutes,
		&t.PriorityScore,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return Todo{}, err
	}
	if len(tagsRaw) == 0 {
		t.Tags = []string{}
	} else if err := json.Unmarshal(tagsRaw, &t.Tags); err != nil {
		return Todo{}, fmt.Errorf("decode tags: %w", err)
	}
	return t, nil
}

func encodeTags(tags []string) ([]byte, error) {
	if len(tags) == 0 {
		return []byte("[]"), nil
	}
	data, err := json.Marshal(tags)
	if err != nil {
		return nil, fmt.Errorf("encode tags: %w", err)
	}
	return data, nil
}
