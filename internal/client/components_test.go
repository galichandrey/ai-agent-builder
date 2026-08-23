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

func TestComponentGetComponentTypes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/all" {
			t.Errorf("expected /api/v1/all, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(allComponentsResponse{
			Result: map[string]schema.ComponentSchema{
				"ChatInput": {
					Display:     "Chat Input",
					Description: "Get chat messages from the user.",
					Name:        "ChatInput",
					BaseClasses: []string{"Message"},
				},
				"Agent": {
					Display:     "Agent",
					Description: "Run an autonomous agent.",
					Name:        "Agent",
					BaseClasses: []string{"Agent"},
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	components, err := c.GetComponentTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 2 {
		t.Fatalf("expected 2 components, got %d", len(components))
	}
	if components["ChatInput"].Name != "ChatInput" {
		t.Errorf("expected ChatInput name, got %s", components["ChatInput"].Name)
	}
	if components["Agent"].Description != "Run an autonomous agent." {
		t.Errorf("unexpected description: %s", components["Agent"].Description)
	}
}

// TestComponentGetComponentTypes_CategoryNested reproduces the real LangFlow
// 1.11+ GET /api/v1/all response shape: top-level categories map to
// {ComponentTypeName: definition}. Components must be keyed by the actual
// type name (the map key), not by display_name ("Chat Input" vs "ChatInput").
// The special "component_display_names" category is a lookup table of plain
// strings and must not leak into results.
func TestComponentGetComponentTypes_CategoryNested(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{
  "input_output": {
    "ChatInput": {
      "display_name": "Chat Input",
      "description": "Get chat messages from the user.",
      "template": {"_type": "Component"},
      "output_types": ["Message"]
    },
    "ChatOutput": {
      "display_name": "Chat Output",
      "description": "Display a chat message.",
      "template": {"_type": "Component"}
    }
  },
  "models_and_agents": {
    "Agent": {
      "display_name": "Agent",
      "description": "Run an autonomous agent.",
      "template": {"_type": "Component"}
    }
  },
  "openai": {
    "ext:openai:OpenAIModelComponent@official": {
      "display_name": "OpenAI",
      "description": "OpenAI models.",
      "template": {"_type": "Component"}
    }
  },
  "component_display_names": {
    "chat input": "ChatInput",
    "chat output": "ChatOutput",
    "agent": "Agent"
  }
}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	components, err := c.GetComponentTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ci, ok := components["ChatInput"]
	if !ok {
		t.Fatalf("expected ChatInput keyed by type name; got keys sample: %v", sampleKeys(components, 10))
	}
	if ci.Name != "ChatInput" {
		t.Errorf("expected Name=ChatInput, got %q", ci.Name)
	}
	if ci.DisplayName != "Chat Input" {
		t.Errorf("expected DisplayName='Chat Input', got %q", ci.DisplayName)
	}
	if ci.Category != "input_output" {
		t.Errorf("expected Category=input_output, got %q", ci.Category)
	}
	if _, ok := ci.Template["_type"]; !ok {
		t.Error("expected Raw template with _type to be preserved")
	}

	if _, ok := components["ext:openai:OpenAIModelComponent@official"]; !ok {
		t.Error("expected extension-namespaced component key to be preserved")
	}

	for _, banned := range []string{"chat input", "agent"} {
		if _, ok := components[banned]; ok {
			t.Errorf("component_display_names lookup key %q leaked into result", banned)
		}
	}

	if _, ok := components["ChatOutput"]; !ok {
		t.Error("expected ChatOutput in result")
	}
	if _, ok := components["Agent"]; !ok {
		t.Error("expected Agent in result")
	}
}

func sampleKeys(m map[string]schema.ComponentSchema, n int) []string {
	keys := make([]string, 0, n)
	for k := range m {
		if len(keys) >= n {
			break
		}
		keys = append(keys, k)
	}
	return keys
}

func TestComponentGetComponentTypes_EmptyResult(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(allComponentsResponse{
			Result: map[string]schema.ComponentSchema{},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	components, err := c.GetComponentTypes(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 0 {
		t.Errorf("expected 0 components, got %d", len(components))
	}
}

func TestComponentGetComponentTypes_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "internal"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.GetComponentTypes(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestComponentGetAllComponents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(allComponentsResponse{
			Result: map[string]schema.ComponentSchema{
				"OpenAIModel": {
					Display:     "OpenAI Model",
					Description: "Use OpenAI LLMs.",
					Name:        "OpenAIModel",
				},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(components))
	}
	if _, ok := components["OpenAIModel"]; !ok {
		t.Error("expected OpenAIModel in result")
	}
}

func TestComponentValidateCustomComponent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/custom_component" {
			t.Errorf("expected /api/v1/custom_component, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["code"] != "class MyComponent(Component): pass" {
			t.Errorf("unexpected code: %s", payload["code"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.ComponentSchema{
			Display:     "My Component",
			Description: "A custom component.",
			Name:        "MyComponent",
			BaseClasses: []string{"Component"},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	cs, err := c.ValidateCustomComponent(context.Background(), "class MyComponent(Component): pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Name != "MyComponent" {
		t.Errorf("expected Name=MyComponent, got %s", cs.Name)
	}
	if cs.Display != "My Component" {
		t.Errorf("expected Display=My Component, got %s", cs.Display)
	}
}

func TestComponentValidateCustomComponent_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"detail": "Invalid component code"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.ValidateCustomComponent(context.Background(), "bad code")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestComponentUpdateCustomComponent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/custom_component/update" {
			t.Errorf("expected /api/v1/custom_component/update, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["code"] != "class ToolComponent(Component): pass" {
			t.Errorf("unexpected code: %s", payload["code"])
		}
		if payload["field"] != "tool_mode" || payload["tool_mode"] != true {
			t.Errorf("unexpected tool_mode request fields: %v", payload)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]any{
			"display_name": "Tool Component",
			"description":  "A tool-enabled component.",
			"base_classes": []string{"Tool"},
			"outputs": []map[string]any{
				{"name": "component_as_tool", "method": "to_toolkit", "types": []string{"Tool"}},
			},
			"template": map[string]any{},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	payload, err := c.UpdateCustomComponent(context.Background(), UpdateCustomComponentRequest{
		Code:     "class ToolComponent(Component): pass",
		Field:    "tool_mode",
		Template: map[string]any{},
		ToolMode: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var cs schema.ComponentSchema
	if err := json.Unmarshal(payload, &cs); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if cs.DisplayName != "Tool Component" && cs.Description != "A tool-enabled component." {
		t.Errorf("expected Tool Component payload, got display=%q desc=%q", cs.DisplayName, cs.Description)
	}
	if len(cs.Outputs) != 1 || cs.Outputs[0].Name != "component_as_tool" {
		t.Errorf("expected component_as_tool output, got %+v", cs.Outputs)
	}
}

func TestComponentUpdateCustomComponent_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"detail": "Cannot update component"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.UpdateCustomComponent(context.Background(), UpdateCustomComponentRequest{
		Code:     "bad code",
		Field:    "tool_mode",
		Template: map[string]any{},
		ToolMode: true,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestComponentValidateCustomComponent_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["code"] != "" {
			t.Errorf("expected empty code, got %s", payload["code"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.ComponentSchema{})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.ValidateCustomComponent(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
