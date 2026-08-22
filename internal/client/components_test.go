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
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["code"] != "class ToolComponent(Component): pass" {
			t.Errorf("unexpected code: %s", payload["code"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.ComponentSchema{
			Display:     "Tool Component",
			Description: "A tool-enabled component.",
			Name:        "ToolComponent",
			BaseClasses: []string{"Tool"},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	cs, err := c.UpdateCustomComponent(context.Background(), "class ToolComponent(Component): pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Name != "ToolComponent" {
		t.Errorf("expected Name=ToolComponent, got %s", cs.Name)
	}
	if cs.BaseClasses[0] != "Tool" {
		t.Errorf("expected base class Tool, got %s", cs.BaseClasses[0])
	}
}

func TestComponentUpdateCustomComponent_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"detail": "Cannot update component"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.UpdateCustomComponent(context.Background(), "bad code")
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
