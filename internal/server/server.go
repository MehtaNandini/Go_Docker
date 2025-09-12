package server

import (
	"context"
	"embed"
	"encoding/json"
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
)

// We declare a dummy variable to ensure the embed package is retained in builds even if not used directly elsewhere in this file.
var _ embed.FS

type Server struct {
	store   *db.Store
	static  fs.FS
}

func NewServer(store *db.Store, staticFS fs.FS) *Server {
	return &Server{store: store, static: staticFS}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()

	// Basic hardening headers and middleware
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger)
	r.Use(s.securityHeaders)

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
	Title string `json:"title"`
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
	item, err := s.store.CreateTodo(ctx, req.Title)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

type updateTodoRequest struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
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
	item, err := s.store.UpdateTodo(ctx, id, strings.TrimSpace(req.Title), req.Completed)
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


