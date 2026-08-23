package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/ag/ai-agent-builder/internal/schema"
)

func mockAllComponentsHandler(components map[string]schema.ComponentSchema) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"result": components,
		})
	}
}

func testComponents() map[string]schema.ComponentSchema {
	return map[string]schema.ComponentSchema{
		"OpenAIModel": {
			Name:        "OpenAIModel",
			DisplayName: "OpenAI",
			Description: "Use OpenAI GPT models",
			Category:    "models",
			Display:     "models",
			BaseClasses: []string{"LanguageModel"},
			OutputTypes: []string{"LanguageModel"},
		},
		"Agent": {
			Name:        "Agent",
			DisplayName: "Agent",
			Description: "Create an autonomous agent",
			Category:    "agents",
			Display:     "agents",
			BaseClasses: []string{"Agent"},
			OutputTypes: []string{"Message"},
		},
		"QdrantVectorStore": {
			Name:        "QdrantVectorStore",
			DisplayName: "Qdrant",
			Description: "Qdrant vector store for RAG",
			Category:    "vectorstores",
			Display:     "vectorstores",
			BaseClasses: []string{"VectorStore"},
			OutputTypes: []string{"VectorStore"},
		},
		"ChatInput": {
			Name:        "ChatInput",
			DisplayName: "Chat Input",
			Description: "Get chat input from user",
			Category:    "input_output",
			Display:     "input_output",
			BaseClasses: []string{"MessageTextInput"},
			OutputTypes: []string{"Message"},
		},
	}
}

func TestComponentTools_ListCategories(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	cats, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result := extractCategories(cats)
	if len(result) != 4 {
		t.Fatalf("expected 4 categories, got %d: %v", len(result), result)
	}
}

func TestComponentTools_ListComponents(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	summaries := filterByCategory(components, "models")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 component in 'models', got %d", len(summaries))
	}
	if summaries[0].Name != "OpenAIModel" {
		t.Errorf("expected OpenAIModel, got %q", summaries[0].Name)
	}
	if summaries[0].Category != "models" {
		t.Errorf("expected category 'models', got %q", summaries[0].Category)
	}
}

func TestComponentTools_ListComponentsAll(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	summaries := filterByCategory(components, "")
	if len(summaries) != 4 {
		t.Fatalf("expected 4 components (all), got %d", len(summaries))
	}
}

func TestComponentTools_GetSchema(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cs, ok := components["OpenAIModel"]
	if !ok {
		t.Fatal("expected OpenAIModel component")
	}
	if cs.DisplayName != "OpenAI" {
		t.Errorf("expected display name 'OpenAI', got %q", cs.DisplayName)
	}
	if len(cs.BaseClasses) != 1 || cs.BaseClasses[0] != "LanguageModel" {
		t.Errorf("expected BaseClasses [LanguageModel], got %v", cs.BaseClasses)
	}
}

func TestComponentTools_GetSchemaNotFound(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, ok := components["NonExistent"]
	if ok {
		t.Fatal("expected NonExistent to not be found")
	}
}

func TestComponentTools_Search(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	summaries := searchComponents(components, "openai")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 match for 'openai', got %d", len(summaries))
	}
	if summaries[0].Name != "OpenAIModel" {
		t.Errorf("expected OpenAIModel, got %q", summaries[0].Name)
	}
}

func TestComponentTools_SearchDescription(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	summaries := searchComponents(components, "vector store")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 match for 'vector store', got %d", len(summaries))
	}
	if summaries[0].Name != "QdrantVectorStore" {
		t.Errorf("expected QdrantVectorStore, got %q", summaries[0].Name)
	}
}

func TestComponentTools_SearchCaseInsensitive(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	summaries := searchComponents(components, "AGENT")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 match for 'AGENT', got %d", len(summaries))
	}
	if summaries[0].Name != "Agent" {
		t.Errorf("expected Agent, got %q", summaries[0].Name)
	}
}

func TestComponentTools_SearchNoMatch(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	summaries := searchComponents(components, "zzz-nonexistent")
	if len(summaries) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(summaries))
	}
}

func TestComponentTools_SearchSubstring(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	summaries := searchComponents(components, "qdr")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 match for 'qdr', got %d", len(summaries))
	}
	if summaries[0].Name != "QdrantVectorStore" {
		t.Errorf("expected QdrantVectorStore, got %q", summaries[0].Name)
	}
}

func TestComponentTools_ListComponentsNotFound(t *testing.T) {
	components := testComponents()
	// All have explicit Category, so "nonexistent" should return 0
	summaries := filterByCategory(components, "nonexistent")
	if len(summaries) != 0 {
		t.Fatalf("expected 0 components for nonexistent category, got %d", len(summaries))
	}
}

func TestComponentTools_EmptyComponents(t *testing.T) {
	empty := map[string]schema.ComponentSchema{}
	categories := extractCategories(empty)
	if len(categories) != 0 {
		t.Fatalf("expected 0 categories, got %d", len(categories))
	}
	summaries := filterByCategory(empty, "models")
	if len(summaries) != 0 {
		t.Fatalf("expected 0 summaries, got %d", len(summaries))
	}
	summaries = searchComponents(empty, "openai")
	if len(summaries) != 0 {
		t.Fatalf("expected 0 search results, got %d", len(summaries))
	}
}

func TestComponentTools_HandlerIntegration(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	// Verify tools are callable by calling the client methods
	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	categories := extractCategories(components)
	if len(categories) == 0 {
		t.Fatal("expected categories from registered components")
	}

	summaries := filterByCategory(components, "models")
	if len(summaries) == 0 {
		t.Fatal("expected component summaries from registered tools")
	}
}

func TestComponentTools_FallbackCategory(t *testing.T) {
	// Component with empty Category but non-empty Display
	components := map[string]schema.ComponentSchema{
		"TestComp": {
			Name:        "TestComp",
			DisplayName: "Test",
			Description: "A test component",
			Category:    "",
			Display:     "custom_category",
		},
	}
	categories := extractCategories(components)
	if len(categories) != 1 || categories[0] != "custom_category" {
		t.Fatalf("expected [custom_category], got %v", categories)
	}

	summaries := filterByCategory(components, "custom_category")
	if len(summaries) != 1 {
		t.Fatalf("expected 1 component, got %d", len(summaries))
	}
}

func TestComponentTools_SearchMultipleMatches(t *testing.T) {
	components := map[string]schema.ComponentSchema{
		"OpenAI":    {Name: "OpenAI", DisplayName: "OpenAI Model", Description: "GPT models", Category: "models"},
		"OpenAI2":   {Name: "OpenAI2", DisplayName: "OpenAI Model 2", Description: "Newer GPT", Category: "models"},
		"Anthropic": {Name: "Anthropic", DisplayName: "Claude", Description: "Anthropic model", Category: "models"},
	}

	summaries := searchComponents(components, "openai")
	if len(summaries) != 2 {
		t.Fatalf("expected 2 matches for 'openai', got %d", len(summaries))
	}
}

func TestComponentTools_ListWithEmptyStringFilter(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Empty string should return all components
	summaries := filterByCategory(components, "")
	if len(summaries) != 4 {
		t.Fatalf("expected 4 components with empty filter, got %d", len(summaries))
	}
}

func TestComponentTools_SchemaDisplayName(t *testing.T) {
	srv, c := newTestServer(t, mockAllComponentsHandler(testComponents()))
	registerComponentTools(srv, c)

	components, err := c.GetAllComponents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Verify DisplayName is accessible
	for name, cs := range components {
		if cs.DisplayName == "" {
			t.Errorf("component %q has empty DisplayName", name)
		}
	}
}

func TestComponentTools_CategoryIsolation(t *testing.T) {
	components := map[string]schema.ComponentSchema{
		"A1": {Name: "A1", Category: "cat_a"},
		"A2": {Name: "A2", Category: "cat_a"},
		"B1": {Name: "B1", Category: "cat_b"},
	}

	catA := filterByCategory(components, "cat_a")
	if len(catA) != 2 {
		t.Fatalf("expected 2 in cat_a, got %d", len(catA))
	}
	for _, s := range catA {
		if s.Category != "cat_a" {
			t.Errorf("expected category 'cat_a', got %q", s.Category)
		}
	}

	catB := filterByCategory(components, "cat_b")
	if len(catB) != 1 {
		t.Fatalf("expected 1 in cat_b, got %d", len(catB))
	}
}
