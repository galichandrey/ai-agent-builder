package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerComponentTools(server *mcp.Server, c *client.LangflowClient) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_component_categories",
		Description: "List all available component categories (e.g. agents, models, vectorstores).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		components, err := c.GetAllComponents(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}

		categories := extractCategories(components)
		data, _ := json.Marshal(categories)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_components",
		Description: "List components in a specific category.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListComponentsInput) (*mcp.CallToolResult, any, error) {
		components, err := c.GetAllComponents(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}

		summaries := filterByCategory(components, input.Category)
		data, _ := json.Marshal(summaries)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_component_schema",
		Description: "Get full schema for a component type including all inputs and outputs.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.GetComponentSchemaInput) (*mcp.CallToolResult, any, error) {
		components, err := c.GetAllComponents(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}

		schemaData, ok := components[input.ComponentType]
		if !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: "component type not found: " + input.ComponentType}},
				IsError: true,
			}, nil, nil
		}

		data, _ := json.Marshal(schemaData)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_components",
		Description: "Search components by name or description.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.SearchComponentsInput) (*mcp.CallToolResult, any, error) {
		components, err := c.GetAllComponents(ctx)
		if err != nil {
			return errorResult(err), nil, nil
		}

		summaries := searchComponents(components, input.Query)
		data, _ := json.Marshal(summaries)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
		}, nil, nil
	})
}

func extractCategories(components map[string]schema.ComponentSchema) []string {
	seen := make(map[string]bool)
	var categories []string
	for _, cs := range components {
		cat := cs.Category
		if cat == "" {
			cat = cs.Display
		}
		if cat == "" {
			continue
		}
		if !seen[cat] {
			seen[cat] = true
			categories = append(categories, cat)
		}
	}
	return categories
}

func filterByCategory(components map[string]schema.ComponentSchema, category string) []schema.ComponentSummary {
	var summaries []schema.ComponentSummary
	for _, cs := range components {
		cat := cs.Category
		if cat == "" {
			cat = cs.Display
		}
		if category == "" || strings.EqualFold(cat, category) {
			summaries = append(summaries, schema.ComponentSummary{
				Name:        cs.Name,
				DisplayName: cs.DisplayName,
				Description: cs.Description,
				Category:    cat,
			})
		}
	}
	return summaries
}

func searchComponents(components map[string]schema.ComponentSchema, query string) []schema.ComponentSummary {
	query = strings.ToLower(query)
	var summaries []schema.ComponentSummary
	for _, cs := range components {
		if strings.Contains(strings.ToLower(cs.Name), query) ||
			strings.Contains(strings.ToLower(cs.DisplayName), query) ||
			strings.Contains(strings.ToLower(cs.Description), query) {
			summaries = append(summaries, schema.ComponentSummary{
				Name:        cs.Name,
				DisplayName: cs.DisplayName,
				Description: cs.Description,
				Category:    cs.Category,
			})
		}
	}
	return summaries
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}
}
