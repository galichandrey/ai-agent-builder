package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ag/ai-agent-builder/internal/schema"
)

// ListProjectsResponse is the response from the projects list endpoint.
type ListProjectsResponse struct {
	Projects []schema.Project `json:"projects"`
}

// ListProjects returns all projects.
func (c *LangflowClient) ListProjects(ctx context.Context) ([]schema.Project, error) {
	data, err := c.doGet(ctx, "/projects/")
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	var resp ListProjectsResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("decode list projects response: %w", err)
	}

	return resp.Projects, nil
}

// CreateProject creates a new project with the given name.
func (c *LangflowClient) CreateProject(ctx context.Context, name string) (*schema.Project, error) {
	body := map[string]string{
		"name": name,
	}

	data, err := c.doPost(ctx, "/projects/", body)
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	var project schema.Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("decode create project response: %w", err)
	}

	return &project, nil
}

// GetProject returns a single project by ID.
func (c *LangflowClient) GetProject(ctx context.Context, projectID string) (*schema.Project, error) {
	data, err := c.doGet(ctx, "/projects/"+projectID)
	if err != nil {
		return nil, fmt.Errorf("get project %s: %w", projectID, err)
	}

	var project schema.Project
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, fmt.Errorf("decode project response: %w", err)
	}

	return &project, nil
}

// DeleteProject deletes a project by ID.
func (c *LangflowClient) DeleteProject(ctx context.Context, projectID string) error {
	if err := c.doDelete(ctx, "/projects/"+projectID); err != nil {
		return fmt.Errorf("delete project %s: %w", projectID, err)
	}
	return nil
}
