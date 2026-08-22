package client

import (
	"context"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/schema"
)

func TestNDJSON_SingleEvent(t *testing.T) {
	input := `{"build_status":"building","build_id":"abc123","flow_id":"flow-1","message":"starting build"}`
	events, err := ParseNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.BuildStatus != "building" {
		t.Errorf("expected build_status=building, got %q", e.BuildStatus)
	}
	if e.BuildID != "abc123" {
		t.Errorf("expected build_id=abc123, got %q", e.BuildID)
	}
	if e.FlowID != "flow-1" {
		t.Errorf("expected flow_id=flow-1, got %q", e.FlowID)
	}
	if e.Message != "starting build" {
		t.Errorf("expected message='starting build', got %q", e.Message)
	}
}

func TestNDJSON_MultipleEvents(t *testing.T) {
	input := `{"build_status":"building","message":"step 1"}
{"build_status":"building","message":"step 2"}
{"build_status":"complete","message":"done"}`
	events, err := ParseNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[2].BuildStatus != "complete" {
		t.Errorf("expected last event build_status=complete, got %q", events[2].BuildStatus)
	}
	if events[1].Message != "step 2" {
		t.Errorf("expected second event message='step 2', got %q", events[1].Message)
	}
}

func TestNDJSON_EmptyLinesSkipped(t *testing.T) {
	input := "\n\n{\"build_status\":\"building\",\"message\":\"step 1\"}\n\n\n{\"build_status\":\"complete\",\"message\":\"done\"}\n\n"
	events, err := ParseNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events (empty lines skipped), got %d", len(events))
	}
}

func TestNDJSON_MalformedJSONTreatedAsStatus(t *testing.T) {
	input := "not valid json\n{\"build_status\":\"building\",\"message\":\"step 1\"}\nanother bad line"
	events, err := ParseNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Message != "not valid json" {
		t.Errorf("expected first event message='not valid json', got %q", events[0].Message)
	}
	if events[2].Message != "another bad line" {
		t.Errorf("expected third event message='another bad line', got %q", events[2].Message)
	}
	if events[0].BuildStatus != "" {
		t.Errorf("expected first event build_status='' for status string, got %q", events[0].BuildStatus)
	}
}

func TestNDJSON_EmptyInput(t *testing.T) {
	events, err := ParseNDJSON(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected 0 events for empty input, got %d", len(events))
	}
}

func TestNDJSON_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	input := strings.NewReader("not checked")
	eventCh := make(chan schema.BuildEvent, 10)
	err := StreamNDJSON(ctx, input, eventCh)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if ctx.Err() == nil {
		t.Fatal("expected context to be cancelled")
	}
}

func TestNDJSON_StreamingMultipleEvents(t *testing.T) {
	input := `{"build_status":"building","message":"step 1"}
{"build_status":"building","message":"step 2"}
{"build_status":"complete","message":"done"}`
	ctx := context.Background()
	eventCh := make(chan schema.BuildEvent, 10)

	err := StreamNDJSON(ctx, strings.NewReader(input), eventCh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var received []schema.BuildEvent
	for e := range eventCh {
		received = append(received, e)
	}
	if len(received) != 3 {
		t.Fatalf("expected 3 events via streaming, got %d", len(received))
	}
	if received[2].BuildStatus != "complete" {
		t.Errorf("expected last streaming event build_status=complete, got %q", received[2].BuildStatus)
	}
}

func TestNDJSON_LargePayload(t *testing.T) {
	bigData := strings.Repeat("x", 5000)
	input := `{"build_status":"building","message":"` + bigData + `"}`
	events, err := ParseNDJSON(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if len(events[0].Message) != 5000 {
		t.Errorf("expected message length 5000, got %d", len(events[0].Message))
	}
}

func TestNDJSON_ParseLine(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		e := parseLine(`{"build_status":"complete","vertex_id":"v1"}`)
		if e == nil {
			t.Fatal("expected non-nil event")
		}
		if e.BuildStatus != "complete" {
			t.Errorf("expected build_status=complete, got %q", e.BuildStatus)
		}
		if e.VertexID != "v1" {
			t.Errorf("expected vertex_id=v1, got %q", e.VertexID)
		}
	})

	t.Run("plain string", func(t *testing.T) {
		e := parseLine("starting build...")
		if e == nil {
			t.Fatal("expected non-nil event")
		}
		if e.Message != "starting build..." {
			t.Errorf("expected message='starting build...', got %q", e.Message)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		e := parseLine("")
		if e != nil {
			t.Fatal("expected nil for empty line")
		}
	})
}
