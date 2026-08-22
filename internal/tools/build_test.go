package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildTools_BuildFlow_Streaming(t *testing.T) {
	events := []schema.BuildEvent{
		{BuildStatus: schema.BuildStatusBuilding, VertexID: "node-1", Message: "building"},
		{BuildStatus: schema.BuildStatusComplete, VertexID: "node-2", Message: "done"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/flow-123/flow", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		for _, e := range events {
			line, _ := json.Marshal(e)
			w.Write(line)
			w.Write([]byte("\n"))
		}
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	ctx := context.Background()
	eventCh, err := c.BuildFlow(ctx, "flow-123", client.BuildFlowRequest{
		InputValue: "hello",
		InputType:  "chat",
	})
	if err != nil {
		t.Fatalf("BuildFlow error: %v", err)
	}

	var got []schema.BuildEvent
	for e := range eventCh {
		got = append(got, e)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].BuildStatus != schema.BuildStatusBuilding {
		t.Errorf("expected first event status 'building', got %q", got[0].BuildStatus)
	}
	if got[1].BuildStatus != schema.BuildStatusComplete {
		t.Errorf("expected second event status 'complete', got %q", got[1].BuildStatus)
	}
}

func TestBuildTools_BuildFlow_ContextCancel(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/flow-abc/flow", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ndjson")
		e := schema.BuildEvent{BuildStatus: schema.BuildStatusBuilding, Message: "building"}
		line, _ := json.Marshal(e)
		w.Write(line)
		w.Write([]byte("\n"))
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	ctx, cancel := context.WithCancel(context.Background())
	eventCh, err := c.BuildFlow(ctx, "flow-abc", client.BuildFlowRequest{})
	if err != nil {
		t.Fatalf("BuildFlow error: %v", err)
	}

	// Cancel immediately to trigger ctx.Done path
	cancel()

	// Should drain whatever was in the channel or return on ctx.Done
	select {
	case <-eventCh:
		// OK
	case <-ctx.Done():
		// OK
	}
}

func TestBuildTools_BuildNode(t *testing.T) {
	event := schema.BuildEvent{
		BuildStatus: schema.BuildStatusComplete,
		VertexID:    "ChatInput-abc",
		Message:     "built",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/flow-456/vertices/ChatInput-abc", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(event)
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	result, err := c.BuildVertex(context.Background(), "flow-456", "ChatInput-abc", nil)
	if err != nil {
		t.Fatalf("BuildVertex error: %v", err)
	}
	if result.BuildStatus != schema.BuildStatusComplete {
		t.Errorf("expected status 'complete', got %q", result.BuildStatus)
	}
	if result.VertexID != "ChatInput-abc" {
		t.Errorf("expected vertex ID 'ChatInput-abc', got %q", result.VertexID)
	}
}

func TestBuildTools_BuildNode_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/flow-err/vertices/bad-node", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("vertex not found"))
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	_, err := c.BuildVertex(context.Background(), "flow-err", "bad-node", nil)
	if err == nil {
		t.Fatal("expected error for bad vertex, got nil")
	}
}

func TestBuildTools_GetBuildStatus(t *testing.T) {
	events := []schema.BuildEvent{
		{BuildStatus: schema.BuildStatusBuilding, VertexID: "node-1"},
		{BuildStatus: schema.BuildStatusComplete, VertexID: "node-1", Message: "done"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/job-789/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(events)
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	result, err := c.GetBuildStatus(context.Background(), "job-789")
	if err != nil {
		t.Fatalf("GetBuildStatus error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 events, got %d", len(result))
	}
	if result[0].BuildStatus != schema.BuildStatusBuilding {
		t.Errorf("expected first event status 'building', got %q", result[0].BuildStatus)
	}
}

func TestBuildTools_GetBuildStatus_Empty(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/build/job-empty/events", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]schema.BuildEvent{})
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	result, err := c.GetBuildStatus(context.Background(), "job-empty")
	if err != nil {
		t.Fatalf("GetBuildStatus error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 events, got %d", len(result))
	}
}

func TestBuildTools_RegisterAll(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	srv, c := newTestServer(t, mux)
	registerBuildTools(srv, c)

	// Verify that calling registerBuildTools doesn't panic
	// and the server is usable
	_ = srv
}

func TestBuildTools_CollectAllEvents(t *testing.T) {
	eventCh := make(chan schema.BuildEvent, 2)
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusBuilding, Message: "step 1"}
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusComplete, Message: "done"}
	close(eventCh)

	result, _, err := collectAllEvents(context.Background(), eventCh, 0)
	if err != nil {
		t.Fatalf("collectAllEvents error: %v", err)
	}
	if result.IsError {
		t.Fatal("expected no error result")
	}

	var events []schema.BuildEvent
	text := result.Content[0].(*mcp.TextContent).Text
	if err := json.Unmarshal([]byte(text), &events); err != nil {
		t.Fatalf("unmarshal events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestBuildTools_CollectAllEvents_ContextTimeout(t *testing.T) {
	eventCh := make(chan schema.BuildEvent)
	// Channel never sends, so context should time out

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	result, _, err := collectAllEvents(ctx, eventCh, 0)
	if err != nil {
		t.Fatalf("collectAllEvents error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result on context cancel")
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if text != "[]" {
		t.Errorf("expected empty array, got %q", text)
	}
}

func TestBuildTools_CollectAllEvents_ErrorStatus(t *testing.T) {
	eventCh := make(chan schema.BuildEvent, 2)
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusBuilding, Message: "step 1"}
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusError, Error: "boom"}
	close(eventCh)

	result, _, err := collectAllEvents(context.Background(), eventCh, 0)
	if err != nil {
		t.Fatalf("collectAllEvents error: %v", err)
	}

	var events []schema.BuildEvent
	text := result.Content[0].(*mcp.TextContent).Text
	json.Unmarshal([]byte(text), &events)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[1].Error != "boom" {
		t.Errorf("expected error 'boom', got %q", events[1].Error)
	}
}

func TestBuildTools_StreamEvents(t *testing.T) {
	eventCh := make(chan schema.BuildEvent, 2)
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusBuilding, Message: "a"}
	eventCh <- schema.BuildEvent{BuildStatus: schema.BuildStatusComplete, Message: "b"}
	close(eventCh)

	result, _, err := streamEvents(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("streamEvents error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	lines := strings.Split(strings.TrimSpace(text), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d", len(lines))
	}

	var e1, e2 schema.BuildEvent
	json.Unmarshal([]byte(lines[0]), &e1)
	json.Unmarshal([]byte(lines[1]), &e2)
	if e1.Message != "a" {
		t.Errorf("expected first message 'a', got %q", e1.Message)
	}
	if e2.Message != "b" {
		t.Errorf("expected second message 'b', got %q", e2.Message)
	}
}

func TestBuildTools_StreamEvents_Empty(t *testing.T) {
	eventCh := make(chan schema.BuildEvent)
	close(eventCh)

	result, _, err := streamEvents(context.Background(), eventCh)
	if err != nil {
		t.Fatalf("streamEvents error: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	if text != "[]" {
		t.Errorf("expected '[]', got %q", text)
	}
}
