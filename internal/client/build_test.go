package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ag/ai-agent-builder/internal/schema"
)

func TestBuildFlow_SendsCorrectBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/build/flow-123/flow" {
			t.Errorf("expected /api/v1/build/flow-123/flow, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var payload BuildFlowRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}
		if payload.InputValue != "hello world" {
			t.Errorf("expected input_value='hello world', got %s", payload.InputValue)
		}
		if payload.InputType != "chat" {
			t.Errorf("expected input_type='chat', got %s", payload.InputType)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)
		w.Write([]byte("{\"build_status\":\"building\",\"build_id\":\"job-1\"}\n{\"build_status\":\"complete\",\"build_id\":\"job-1\"}\n"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	eventCh, err := c.BuildFlow(context.Background(), "flow-123", BuildFlowRequest{
		InputValue: "hello world",
		InputType:  "chat",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var events []schema.BuildEvent
	for e := range eventCh {
		events = append(events, e)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].BuildStatus != "building" {
		t.Errorf("expected first event build_status='building', got %s", events[0].BuildStatus)
	}
	if events[1].BuildStatus != "complete" {
		t.Errorf("expected second event build_status='complete', got %s", events[1].BuildStatus)
	}
}

func TestBuildFlow_DefaultInputTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload BuildFlowRequest
		json.Unmarshal(body, &payload)
		if payload.InputType != "chat" {
			t.Errorf("expected default input_type='chat', got %s", payload.InputType)
		}
		if payload.OutputType != "chat" {
			t.Errorf("expected default output_type='chat', got %s", payload.OutputType)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)
		w.Write([]byte("{\"build_status\":\"complete\"}\n"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	eventCh, err := c.BuildFlow(context.Background(), "flow-1", BuildFlowRequest{InputValue: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for range eventCh {
	}
}

func TestBuildFlow_WithTweaks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload BuildFlowRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		tweaks, ok := payload.Tweaks["OpenAIModel"]
		if !ok {
			t.Fatal("expected tweaks['OpenAIModel']")
		}
		tweakMap, ok := tweaks.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", tweaks)
		}
		if tweakMap["temperature"] != float64(0.5) {
			t.Errorf("expected temperature=0.5, got %v", tweakMap["temperature"])
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		w.WriteHeader(200)
		w.Write([]byte("{\"build_status\":\"complete\"}\n"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.BuildFlow(context.Background(), "flow-1", BuildFlowRequest{
		InputValue: "test",
		Tweaks:     map[string]any{"OpenAIModel": map[string]any{"temperature": 0.5}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFlow_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("{\"error\": \"build failed\"}"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.BuildFlow(context.Background(), "flow-1", BuildFlowRequest{InputValue: "hi"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildVertex_SendsCorrectBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/build/flow-1/vertices/vertex-abc" {
			t.Errorf("expected /api/v1/build/flow-1/vertices/vertex-abc, got %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		body, _ := io.ReadAll(r.Body)
		var payload BuildVertexRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload.Tweaks == nil {
			t.Fatal("expected non-nil tweaks")
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.BuildEvent{
			BuildStatus: "complete",
			VertexID:    "vertex-abc",
			Message:     "built successfully",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	event, err := c.BuildVertex(context.Background(), "flow-1", "vertex-abc", map[string]any{
		"SomeComponent": map[string]any{"param": "value"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.BuildStatus != "complete" {
		t.Errorf("expected build_status='complete', got %s", event.BuildStatus)
	}
	if event.VertexID != "vertex-abc" {
		t.Errorf("expected vertex_id='vertex-abc', got %s", event.VertexID)
	}
	if event.Message != "built successfully" {
		t.Errorf("expected message='built successfully', got %s", event.Message)
	}
}

func TestBuildVertex_EmptyTweaks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload BuildVertexRequest
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload.Tweaks != nil {
			t.Errorf("expected nil tweaks, got %v", payload.Tweaks)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.BuildEvent{BuildStatus: "complete", VertexID: "v1"})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.BuildVertex(context.Background(), "flow-1", "v1", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildVertex_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("{\"detail\": \"vertex not found\"}"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.BuildVertex(context.Background(), "flow-1", "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetBuildStatus_ReturnsEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/build/job-42/events" {
			t.Errorf("expected /api/v1/build/job-42/events, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode([]schema.BuildEvent{
			{BuildStatus: "building", BuildID: "job-42"},
			{BuildStatus: "complete", BuildID: "job-42"},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	events, err := c.GetBuildStatus(context.Background(), "job-42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].BuildStatus != "building" {
		t.Errorf("expected building, got %s", events[0].BuildStatus)
	}
	if events[1].BuildStatus != "complete" {
		t.Errorf("expected complete, got %s", events[1].BuildStatus)
	}
}

func TestGetBuildStatus_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte("{\"detail\": \"job not found\"}"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.GetBuildStatus(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetTopologicalOrder_ReturnsVertices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/build/flow-1/vertices" {
			t.Errorf("expected /api/v1/build/flow-1/vertices, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(TopologicalResponse{
			Vertices: []string{"node-1", "node-2", "node-3"},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	vertices, err := c.GetTopologicalOrder(context.Background(), "flow-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vertices) != 3 {
		t.Fatalf("expected 3 vertices, got %d", len(vertices))
	}
	expected := []string{"node-1", "node-2", "node-3"}
	for i, v := range vertices {
		if v != expected[i] {
			t.Errorf("vertex[%d]: expected %s, got %s", i, expected[i], v)
		}
	}
}

func TestGetTopologicalOrder_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(TopologicalResponse{Vertices: []string{}})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	vertices, err := c.GetTopologicalOrder(context.Background(), "flow-empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(vertices) != 0 {
		t.Errorf("expected 0 vertices, got %d", len(vertices))
	}
}

func TestGetTopologicalOrder_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("{\"error\": \"internal\"}"))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.GetTopologicalOrder(context.Background(), "flow-1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
