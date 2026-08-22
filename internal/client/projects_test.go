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

func TestProjectListProjects(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/" {
			t.Errorf("expected /api/v1/projects/, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListProjectsResponse{
			Projects: []schema.Project{
				{ID: "p1", Name: "Project 1"},
				{ID: "p2", Name: "Project 2"},
			},
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
	if projects[0].ID != "p1" || projects[0].Name != "Project 1" {
		t.Errorf("unexpected first project: %+v", projects[0])
	}
}

func TestProjectListProjects_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(ListProjectsResponse{Projects: []schema.Project{}})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestProjectListProjects_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error": "internal"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.ListProjects(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProjectCreateProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/" {
			t.Errorf("expected /api/v1/projects/, got %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("failed to unmarshal body: %v", err)
		}
		if payload["name"] != "New Project" {
			t.Errorf("expected name=New Project, got %s", payload["name"])
		}
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(schema.Project{
			ID:   "created-1",
			Name: "New Project",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	project, err := c.CreateProject(context.Background(), "New Project")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != "created-1" {
		t.Errorf("expected ID=created-1, got %s", project.ID)
	}
	if project.Name != "New Project" {
		t.Errorf("expected Name=New Project, got %s", project.Name)
	}
}

func TestProjectCreateProject_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"detail": "bad request"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.CreateProject(context.Background(), "Bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestProjectGetProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/my-project-id" {
			t.Errorf("expected /api/v1/projects/my-project-id, got %s", r.URL.Path)
		}
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(schema.Project{
			ID:          "my-project-id",
			Name:        "Test Project",
			Description: "A test project",
		})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	project, err := c.GetProject(context.Background(), "my-project-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if project.ID != "my-project-id" {
		t.Errorf("expected ID=my-project-id, got %s", project.ID)
	}
	if project.Name != "Test Project" {
		t.Errorf("expected Name=Test Project, got %s", project.Name)
	}
}

func TestProjectGetProject_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Project not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	_, err := c.GetProject(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestProjectDeleteProject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/projects/project-1" {
			t.Errorf("expected /api/v1/projects/project-1, got %s", r.URL.Path)
		}
		w.WriteHeader(204)
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	if err := c.DeleteProject(context.Background(), "project-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectDeleteProject_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"detail": "Project not found"}`))
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	err := c.DeleteProject(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestProjectDeleteProject_OkStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	if err := c.DeleteProject(context.Background(), "project-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
