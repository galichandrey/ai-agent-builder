package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ag/ai-agent-builder/internal/config"
)

func TestHealthHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	healthHandler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", body["status"])
	}
}

func TestWithCORSPreflight(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/mcp", nil)
	withCORS(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS origin header")
	}
}

func TestResolveHTTPAddr(t *testing.T) {
	cases := []struct {
		name    string
		addr    string
		host    string
		port    string
		want    string
	}{
		{"explicit flag", ":9090", "1.2.3.4", "8080", ":9090"},
		{"config fallback", "", "127.0.0.1", "9999", "127.0.0.1:9999"},
		{"empty", "", "", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cfg := &struct {
				HTTPHost string
				HTTPPort string
			}{c.host, c.port}
			got := resolveHTTPAddr(c.addr, &config.Config{HTTPHost: cfg.HTTPHost, HTTPPort: cfg.HTTPPort})
			if got != c.want {
				t.Fatalf("resolveHTTPAddr(%q) = %q, want %q", c.addr, got, c.want)
			}
		})
	}
}
