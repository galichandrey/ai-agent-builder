package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func newTestServer(t *testing.T, handler http.Handler) (*mcp.Server, *client.LangflowClient) {
	t.Helper()
	ts := httptest.NewServer(handler)
	cfg := &config.Config{
		LangflowURL:    ts.URL,
		APIKey:         "test-api-key",
		RequestTimeout: 30,
	}
	c := client.NewClient(cfg)
	t.Cleanup(ts.Close)
	return mcp.NewServer(&mcp.Implementation{Name: "test"}, nil), c
}

func TestFlowTools_ListFlows(t *testing.T) {
	flows := []schema.Flow{
		{ID: "flow-1", Name: "Test Flow"},
		{ID: "flow-2", Name: "Another Flow"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flows": flows,
			"total": 2,
			"page":  1,
			"size":  50,
			"pages": 1,
		})
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, _, err := c.ListFlows(context.Background(), 1, 50, "")
	if err != nil {
		t.Fatalf("ListFlows error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(result))
	}
	if result[0].ID != "flow-1" {
		t.Errorf("expected first flow ID 'flow-1', got %q", result[0].ID)
	}
}

func TestFlowTools_ListFlowsWithFolderID(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		folderID := r.URL.Query().Get("folder_id")
		if folderID != "folder-abc" {
			t.Errorf("expected folder_id 'folder-abc', got %q", folderID)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flows": []schema.Flow{},
			"total": 0,
			"page":  1,
			"size":  50,
			"pages": 0,
		})
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, _, err := c.ListFlows(context.Background(), 1, 50, "folder-abc")
	if err != nil {
		t.Fatalf("ListFlows error: %v", err)
	}
	if len(result) != 0 {
		t.Fatalf("expected 0 flows, got %d", len(result))
	}
}

func TestFlowTools_ListAllFlows(t *testing.T) {
	flows := []schema.Flow{
		{ID: "flow-1", Name: "Normal Flow"},
		{ID: "flow-backup", Name: "Backup Flow"},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"flows": flows,
			"total": 2,
		})
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, err := c.ListAllFlows(context.Background())
	if err != nil {
		t.Fatalf("ListAllFlows error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 flows, got %d", len(result))
	}
}

func TestFlowTools_GetFlow(t *testing.T) {
	flow := schema.Flow{
		ID:          "flow-123",
		Name:        "My Flow",
		Description: "A test flow",
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-123", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(flow)
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, err := c.GetFlow(context.Background(), "flow-123")
	if err != nil {
		t.Fatalf("GetFlow error: %v", err)
	}
	if result.ID != "flow-123" {
		t.Errorf("expected ID 'flow-123', got %q", result.ID)
	}
	if result.Name != "My Flow" {
		t.Errorf("expected name 'My Flow', got %q", result.Name)
	}
}

func TestFlowTools_CreateFlow(t *testing.T) {
	var receivedBody map[string]string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		json.NewDecoder(r.Body).Decode(&receivedBody)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schema.Flow{
			ID:          "new-flow-id",
			Name:        receivedBody["name"],
			Description: receivedBody["description"],
		})
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, err := c.CreateFlow(context.Background(), "New Flow", "My description")
	if err != nil {
		t.Fatalf("CreateFlow error: %v", err)
	}
	if result.ID != "new-flow-id" {
		t.Errorf("expected ID 'new-flow-id', got %q", result.ID)
	}
	if result.Name != "New Flow" {
		t.Errorf("expected name 'New Flow', got %q", result.Name)
	}
	if receivedBody["description"] != "My description" {
		t.Errorf("expected description 'My description', got %q", receivedBody["description"])
	}
}

func TestFlowTools_DeleteFlow(t *testing.T) {
	var deletedID string

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/flow-to-delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		deletedID = "flow-to-delete"
		w.WriteHeader(http.StatusOK)
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	err := c.DeleteFlow(context.Background(), "flow-to-delete")
	if err != nil {
		t.Fatalf("DeleteFlow error: %v", err)
	}
	if deletedID != "flow-to-delete" {
		t.Errorf("expected deleted ID 'flow-to-delete', got %q", deletedID)
	}
}

func TestFlowTools_DuplicateFlow(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/original-id", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(schema.Flow{
			ID:          "original-id",
			Name:        "Original Name",
			Description: "Original Desc",
		})
	})
	mux.HandleFunc("/api/v1/flows/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(schema.Flow{
				ID:          "duplicated-id",
				Name:        body["name"],
				Description: body["description"],
			})
			return
		}
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	result, err := c.DuplicateFlow(context.Background(), "original-id", "Copy Name")
	if err != nil {
		t.Fatalf("DuplicateFlow error: %v", err)
	}
	if result.ID != "duplicated-id" {
		t.Errorf("expected ID 'duplicated-id', got %q", result.ID)
	}
	if result.Name != "Copy Name" {
		t.Errorf("expected name 'Copy Name', got %q", result.Name)
	}
}

func TestFlowTools_DeleteFlowError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/bad-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	err := c.DeleteFlow(context.Background(), "bad-id")
	if err == nil {
		t.Fatal("expected error for bad flow ID, got nil")
	}
}

func TestFlowTools_GetFlowError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/flows/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})

	srv, c := newTestServer(t, mux)
	registerFlowTools(srv, c)

	_, err := c.GetFlow(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing flow, got nil")
	}
}
