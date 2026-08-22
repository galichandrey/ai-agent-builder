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

func TestFlowListFlows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows/" {
			t.Errorf("expected /api/v1/flows/, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" {
			t.Errorf("expected page=2, got %s", r.URL.Query().Get("page"))
		}
		if r.URL.Query().Get("size") != "10" {
			t.Errorf("expected size=10, got %s", r.URL.Query().Get("size"))
		}
		if r.URL.Query().Get("folder_id") != "folder-abc" {
			t.Errorf("expected folder_id=folder-abc, got %s", r.URL.Query().Get("folder_id"))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListFlowsResponse{
			Flows: []schema.Flow{
				{ID: "f1", Name: "Flow 1"},
				{ID: "f2", Name: "Flow 2"},
			},
			Total: 25,
			Page:  2,
			Size:  10,
			Pages: 3,
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flows, total, err := c.ListFlows(context.Background(), 2, 10, "folder-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 25 {
		t.Errorf("expected total=25, got %d", total)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(flows))
	}
	if flows[0].ID != "f1" || flows[0].Name != "Flow 1" {
		t.Errorf("unexpected first flow: %+v", flows[0])
	}
}

func TestFlowListFlows_NoFolderID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("folder_id") != "" {
			t.Errorf("expected no folder_id, got %s", r.URL.Query().Get("folder_id"))
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListFlowsResponse{Flows: []schema.Flow{}, Total: 0})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, _, err := c.ListFlows(context.Background(), 1, 50, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlowListFlows_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "internal"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, _, err := c.ListFlows(context.Background(), 1, 50, "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestFlowListAllFlows(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListFlowsResponse{
			Flows: []schema.Flow{
				{ID: "f1", Name: "Flow 1"},
				{ID: "f2", Name: "Flow 2"},
				{ID: "f3", Name: "Flow 3"},
			},
			Total: 3,
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flows, err := c.ListAllFlows(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flows) != 3 {
		t.Errorf("expected 3 flows, got %d", len(flows))
	}
}

func TestFlowGetFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows/my-flow-id" {
			t.Errorf("expected /api/v1/flows/my-flow-id, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.Flow{
			ID:          "my-flow-id",
			Name:        "Test Flow",
			Description: "A test flow",
			FolderID:    "folder-1",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flow, err := c.GetFlow(context.Background(), "my-flow-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.ID != "my-flow-id" {
		t.Errorf("expected ID=my-flow-id, got %s", flow.ID)
	}
	if flow.Name != "Test Flow" {
		t.Errorf("expected Name=Test Flow, got %s", flow.Name)
	}
}

func TestFlowGetFlow_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Flow not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.GetFlow(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestFlowCreateFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows/" {
			t.Errorf("expected /api/v1/flows/, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["name"] != "New Flow" {
			t.Errorf("expected name=New Flow, got %s", payload["name"])
		}
		if payload["description"] != "desc" {
			t.Errorf("expected description=desc, got %s", payload["description"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(schema.Flow{
			ID:          "created-1",
			Name:        "New Flow",
			Description: "desc",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flow, err := c.CreateFlow(context.Background(), "New Flow", "desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.ID != "created-1" {
		t.Errorf("expected ID=created-1, got %s", flow.ID)
	}
}

func TestFlowCreateFlow_EmptyDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		json.Unmarshal(body, &payload)
		if _, ok := payload["description"]; !ok {
			// LangFlow accepts omitted description
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(schema.Flow{ID: "created-2", Name: "Flow"})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.CreateFlow(context.Background(), "Flow", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlowUpdateFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows/flow-1" {
			t.Errorf("expected /api/v1/flows/flow-1, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["name"] != "Updated Name" {
			t.Errorf("expected name=Updated Name, got %v", payload["name"])
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.Flow{
			ID:   "flow-1",
			Name: "Updated Name",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flow, err := c.UpdateFlow(context.Background(), "flow-1", map[string]any{"name": "Updated Name"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.Name != "Updated Name" {
		t.Errorf("expected Name=Updated Name, got %s", flow.Name)
	}
}

func TestFlowDeleteFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/flows/flow-1" {
			t.Errorf("expected /api/v1/flows/flow-1, got %s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	if err := c.DeleteFlow(context.Background(), "flow-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlowDeleteFlow_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Flow not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	err := c.DeleteFlow(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestFlowDuplicateFlow(t *testing.T) {
	var requests []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/flows/flow-1":
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(schema.Flow{
				ID:          "flow-1",
				Name:        "Original",
				Description: "original desc",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/flows/":
			body, _ := io.ReadAll(r.Body)
			var payload map[string]string
			json.Unmarshal(body, &payload)
			if payload["name"] != "Copy of Original" {
				t.Errorf("expected name='Copy of Original', got %s", payload["name"])
			}
			if payload["description"] != "original desc" {
				t.Errorf("expected description='original desc', got %s", payload["description"])
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(schema.Flow{
				ID:          "flow-copy-1",
				Name:        "Copy of Original",
				Description: "original desc",
			})
		default:
			w.WriteHeader(400)
		}
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flow, err := c.DuplicateFlow(context.Background(), "flow-1", "Copy of Original")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flow.ID != "flow-copy-1" {
		t.Errorf("expected ID=flow-copy-1, got %s", flow.ID)
	}
	if len(requests) != 2 {
		t.Errorf("expected 2 requests, got %d", len(requests))
	}
}

func TestFlowDuplicateFlow_DefaultName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet:
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(schema.Flow{
				ID:          "flow-1",
				Name:        "My Flow",
				Description: "desc",
			})
		case r.Method == http.MethodPost:
			body, _ := io.ReadAll(r.Body)
			var payload map[string]string
			json.Unmarshal(body, &payload)
			if payload["name"] != "My Flow" {
				t.Errorf("expected name='My Flow' (original), got %s", payload["name"])
			}
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(schema.Flow{ID: "copy-1", Name: "My Flow"})
		default:
			w.WriteHeader(400)
		}
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.DuplicateFlow(context.Background(), "flow-1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFlowDuplicateFlow_SourceNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.DuplicateFlow(context.Background(), "nonexistent", "Copy")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
