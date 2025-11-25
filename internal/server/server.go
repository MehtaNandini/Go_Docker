package server

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"todoapp/internal/db"
	"todoapp/internal/mlclient"
)

// We declare a dummy variable to ensure the embed package is retained in builds even if not used directly elsewhere in this file.
var _ embed.FS

type Server struct {
	store  *db.Store
	static fs.FS
	scorer priorityScorer
}

type priorityScorer interface {
	Score(ctx context.Context, todo mlclient.TodoPayload) (float64, error)
}

func NewServer(store *db.Store, staticFS fs.FS, scorer priorityScorer) *Server {
	return &Server{store: store, static: staticFS, scorer: scorer}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	// Basic hardening headers and middleware
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(s.securityHeaders)

	// Health check endpoint for Kubernetes probes
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})

	r.Route("/api/todos", func(r chi.Router) {
		r.Get("/", s.handleListTodos)
		r.Post("/", s.handleCreateTodo)
		r.Put("/{id}", s.handleUpdateTodo)
		r.Delete("/{id}", s.handleDeleteTodo)
	})

	// Serve static frontend
	web, err := fs.Sub(s.static, "web")
	if err != nil {
		// If embedding fails, provide a helpful message
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "static assets not found", http.StatusInternalServerError)
		})
		return r
	}
	fileServer := http.FileServer(http.FS(web))

	// Serve index.html for root and unknown paths (client-side navigation not needed but friendly)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeFileFS(w, r, web, "index.html")
	})
	r.Handle("/*", fileServer)

	return r
}

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", ww.Status(),
			"bytes", ww.BytesWritten(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		// HSTS only makes sense behind HTTPS; harmless if HTTP
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleListTodos(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	items, err := s.store.ListTodos(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list todos")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type createTodoRequest struct {
	Title           string   `json:"title"`
	Tags            []string `json:"tags"`
	DurationMinutes int      `json:"durationMinutes"`
}

func (s *Server) handleCreateTodo(w http.ResponseWriter, r *http.Request) {
	body := http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
	defer body.Close()
	var req createTodoRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	// Trim spaces
	req.Title = strings.TrimSpace(req.Title)
	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	tags := normalizeTags(req.Tags)
	duration := clampDuration(req.DurationMinutes)
	priority := s.computePriority(ctx, priorityCandidate{
		Title:           req.Title,
		Completed:       false,
		Tags:            tags,
		DurationMinutes: duration,
		CreatedAt:       time.Now().UTC(),
	}, 0)

	item, err := s.store.CreateTodo(ctx, db.SaveTodoInput{
		Title:           req.Title,
		Completed:       false,
		Tags:            tags,
		DurationMinutes: duration,
		PriorityScore:   priority,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type updateTodoRequest struct {
	Title           string   `json:"title"`
	Completed       bool     `json:"completed"`
	Tags            []string `json:"tags"`
	DurationMinutes int      `json:"durationMinutes"`
}

func (s *Server) handleUpdateTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	defer body.Close()
	var req updateTodoRequest
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	existing, err := s.store.GetTodo(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, "todo not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to load todo")
		return
	}

	title := strings.TrimSpace(req.Title)
	tags := normalizeTags(req.Tags)
	duration := clampDuration(req.DurationMinutes)

	priority := s.computePriority(ctx, priorityCandidate{
		Title:           title,
		Completed:       req.Completed,
		Tags:            tags,
		DurationMinutes: duration,
		CreatedAt:       existing.CreatedAt,
	}, existing.PriorityScore)

	item, err := s.store.UpdateTodo(ctx, id, db.SaveTodoInput{
		Title:           title,
		Completed:       req.Completed,
		Tags:            tags,
		DurationMinutes: duration,
		PriorityScore:   priority,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) handleDeleteTodo(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	ctx, cancel := contextWithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := s.store.DeleteTodo(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete")
		return
	}
	w.WriteHeader(http.StatusNoContent)
	_, _ = io.WriteString(w, "")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func contextWithTimeout(parentCtx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parentCtx, d)
}

type priorityCandidate struct {
	Title           string
	Completed       bool
	Tags            []string
	DurationMinutes int
	CreatedAt       time.Time
}

func (s *Server) computePriority(ctx context.Context, candidate priorityCandidate, fallback float64) float64 {
	if s.scorer == nil {
		return fallback
	}
	payload := mlclient.TodoPayload{
		Title:           candidate.Title,
		Completed:       candidate.Completed,
		Tags:            candidate.Tags,
		DurationMinutes: candidate.DurationMinutes,
	}
	if !candidate.CreatedAt.IsZero() {
		c := candidate.CreatedAt
		payload.CreatedAt = &c
	}
	score, err := s.scorer.Score(ctx, payload)
	if err != nil {
		slog.Warn("ml.score_failed", "error", err)
		return fallback
	}
	return score
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, raw := range tags {
		tag := strings.TrimSpace(strings.ToLower(raw))
		if tag == "" {
			continue
		}
		if len(tag) > 32 {
			tag = tag[:32]
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
	}
	return out
}

func clampDuration(val int) int {
	if val < 0 {
		return 0
	}
	if val > 24*60 {
		return 24 * 60
	}
	return val
}
