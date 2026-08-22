package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ag/ai-agent-builder/internal/config"
)

func testConfig(url string) *config.Config {
	cfg := &config.Config{
		LangflowURL:    url,
		APIKey:         "test-key-123",
		RequestTimeout: 5,
		CustomHeaders:  map[string]string{"X-Custom": "custom-value"},
	}
	return cfg
}

func TestDoGet_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows" {
			t.Errorf("expected /api/v1/flows, got %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "test-key-123" {
			t.Errorf("expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("X-Custom") != "custom-value" {
			t.Errorf("expected X-Custom header, got %s", r.Header.Get("X-Custom"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"flows": []}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	data, err := c.doGet(context.Background(), "/flows")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"flows": []}` {
		t.Errorf("unexpected body: %s", string(data))
	}
}

func TestDoPost_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if payload["name"] != "test-flow" {
			t.Errorf("expected name=test-flow, got %s", payload["name"])
		}
		w.WriteHeader(201)
		w.Write([]byte(`{"id": "flow-1"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	body := map[string]string{"name": "test-flow"}
	data, err := c.doPost(context.Background(), "/flows", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"id": "flow-1"}` {
		t.Errorf("unexpected body: %s", string(data))
	}
}

func TestDoPatch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if payload["name"] != "updated" {
			t.Errorf("expected name=updated, got %s", payload["name"])
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	body := map[string]string{"name": "updated"}
	data, err := c.doPatch(context.Background(), "/flows/flow-1", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"ok": true}` {
		t.Errorf("unexpected body: %s", string(data))
	}
}

func TestDoDelete_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"deleted": true}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	if err := c.doDelete(context.Background(), "/flows/flow-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDoGetStream_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"building"}
{"status":"complete"}
`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	stream, err := c.doGetStream(context.Background(), "/build/build-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		t.Fatalf("failed to read stream: %v", err)
	}
	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Errorf("expected 2 lines, got %d: %s", len(lines), string(data))
	}
}

func TestAuthHeader_OmittedWhenEmpty(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.APIKey = ""
	c := NewClient(cfg)
	c.doGet(context.Background(), "/test")
	if gotKey != "" {
		t.Errorf("expected no x-api-key header, got %q", gotKey)
	}
}

func TestCustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Tenant") != "org-42" {
			t.Errorf("expected X-Tenant=org-42, got %q", r.Header.Get("X-Tenant"))
		}
		if r.Header.Get("X-Region") != "us-east" {
			t.Errorf("expected X-Region=us-east, got %q", r.Header.Get("X-Region"))
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	cfg := testConfig(srv.URL)
	cfg.CustomHeaders = map[string]string{
		"X-Tenant": "org-42",
		"X-Region": "us-east",
	}
	c := NewClient(cfg)
	c.doGet(context.Background(), "/test")
}

func TestErrorHandling_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Flow not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.doGet(context.Background(), "/flows/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 404 {
		t.Errorf("expected status 404, got %d", httpErr.StatusCode)
	}
	if httpErr.Body != `{"detail": "Flow not found"}` {
		t.Errorf("unexpected body: %s", httpErr.Body)
	}
}

func TestErrorHandling_500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "internal server error"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.doPost(context.Background(), "/flows", map[string]string{})
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	httpErr, ok := err.(*HTTPError)
	if !ok {
		t.Fatalf("expected *HTTPError, got %T", err)
	}
	if httpErr.StatusCode != 500 {
		t.Errorf("expected status 500, got %d", httpErr.StatusCode)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := c.doGet(ctx, "/slow")
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

func TestStreamErrorHandling_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.doGetStream(context.Background(), "/nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 stream, got nil")
	}
}

func TestHTTPError_Message(t *testing.T) {
	e := &HTTPError{StatusCode: 400, Body: `{"error":"bad"}`, Message: "Bad Request"}
	expected := "HTTP 400 (Bad Request): {\"error\":\"bad\"}"
	if e.Error() != expected {
		t.Errorf("expected %q, got %q", expected, e.Error())
	}

	e2 := &HTTPError{StatusCode: 503, Message: "Service Unavailable"}
	expected2 := "HTTP 503 (Service Unavailable)"
	if e2.Error() != expected2 {
		t.Errorf("expected %q, got %q", expected2, e2.Error())
	}
}

func TestNewClient_TimeoutHandling(t *testing.T) {
	// The http.Client has no client-level timeout: cancellation is handled
	// via context (ctx) to avoid silently truncating long streaming responses
	// such as NDJSON build output.
	cfg := &config.Config{LangflowURL: "http://localhost:9999", RequestTimeout: 0}
	c := NewClient(cfg)
	if c.httpClient.Timeout != 0 {
		t.Errorf("expected 0 client timeout (ctx-driven), got %v", c.httpClient.Timeout)
	}

	// Explicit RequestTimeout must still be ignored in favor of ctx (Timeout stays 0)
	cfg2 := &config.Config{LangflowURL: "http://localhost:9999", RequestTimeout: 60}
	c2 := NewClient(cfg2)
	if c2.httpClient.Timeout != 0 {
		t.Errorf("expected 0 client timeout, got %v", c2.httpClient.Timeout)
	}
}

func TestNewClient_BaseURL(t *testing.T) {
	cfg := &config.Config{LangflowURL: "http://localhost:7860/", RequestTimeout: 10}
	c := NewClient(cfg)
	if c.baseURL != "http://localhost:7860" {
		t.Errorf("expected trailing slash trimmed, got %q", c.baseURL)
	}
}

func TestDoPost_NilBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		// Should not have Content-Type when body is nil
		if r.Header.Get("Content-Type") != "" {
			t.Errorf("expected no Content-Type for nil body, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.doPost(context.Background(), "/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConcurrentRequests(t *testing.T) {
	var count int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(200)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	done := make(chan struct{}, 10)
	for i := 0; i < 10; i++ {
		go func() {
			c.doGet(context.Background(), "/test")
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if atomic.LoadInt64(&count) != 10 {
		t.Errorf("expected 10 requests, got %d", count)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
