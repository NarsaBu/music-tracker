package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"music-tracker/backend/internal/domain"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func newTestRouter(p Pinger) http.Handler {
	return NewRouter(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Readiness: p})
}

func TestHealthEndpoints(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		pingErr    error
		wantStatus int
	}{
		{"liveness", "/healthz", nil, http.StatusOK},
		{"liveness ignores db", "/healthz", errors.New("down"), http.StatusOK},
		{"readiness ok", "/readyz", nil, http.StatusOK},
		{"readiness db down", "/readyz", errors.New("down"), http.StatusServiceUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			newTestRouter(fakePinger{tt.pingErr}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestAPINotFound_IsProblemJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestRouter(fakePinger{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Errorf("Content-Type = %q, want application/problem+json", ct)
	}
}

func TestWriteError_Mapping(t *testing.T) {
	tests := []struct {
		err        error
		wantStatus int
	}{
		{&domain.ValidationError{Fields: []domain.FieldError{{Field: "mbid", Message: "required"}}}, http.StatusUnprocessableEntity},
		{domain.ErrNotFound, http.StatusNotFound},
		{domain.ErrAlreadyExists, http.StatusConflict},
		{domain.ErrUpstreamUnavailable, http.StatusServiceUnavailable},
		{errors.New("boom"), http.StatusInternalServerError},
	}
	for _, tt := range tests {
		rec := httptest.NewRecorder()
		writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), tt.err)
		if rec.Code != tt.wantStatus {
			t.Errorf("writeError(%v) status = %d, want %d", tt.err, rec.Code, tt.wantStatus)
		}
	}
}

func TestWriteError_ValidationFields(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil),
		&domain.ValidationError{Fields: []domain.FieldError{{Field: "mbid", Message: "required"}}})

	var p problem
	if err := json.NewDecoder(rec.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if len(p.Errors) != 1 || p.Errors[0].Field != "mbid" {
		t.Errorf("errors = %+v, want one error for field mbid", p.Errors)
	}
}

func TestSPA_ServesFilesAndFallsBackToIndex(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("index"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "assets"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets", "app.js"), []byte("js"), 0o600); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(Deps{Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), Readiness: fakePinger{}, StaticDir: dir})

	tests := []struct {
		path      string
		wantCode  int
		wantBody  string
		wantCache string
	}{
		{"/", http.StatusOK, "index", "no-cache"},
		{"/artists/123", http.StatusOK, "index", "no-cache"},
		{"/assets/app.js", http.StatusOK, "js", "public, max-age=31536000, immutable"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != tt.wantCode || rec.Body.String() != tt.wantBody {
				t.Errorf("got %d %q, want %d %q", rec.Code, rec.Body.String(), tt.wantCode, tt.wantBody)
			}
			if cc := rec.Header().Get("Cache-Control"); cc != tt.wantCache {
				t.Errorf("Cache-Control = %q, want %q", cc, tt.wantCache)
			}
		})
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("API 404 status = %d, want 404", rec.Code)
	}

	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/../../etc/passwd", nil))
	if rec.Code == http.StatusOK {
		t.Errorf("path traversal served %q", rec.Body.String())
	}
}
