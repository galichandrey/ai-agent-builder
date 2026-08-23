package tools

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func runFlowViaMCP(t *testing.T, c *client.LangflowClient, args map[string]any) string {
	t.Helper()
	server := mcp.NewServer(&mcp.Implementation{Name: "test"}, nil)
	registerRunTools(server, c)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _, _ = server.Connect(ctx, serverTransport, nil) }()

	mc := mcp.NewClient(&mcp.Implementation{Name: "tc"}, nil)
	session, err := mc.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer session.Close()

	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "run_flow",
		Arguments: args,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var sb strings.Builder
	for _, item := range res.Content {
		if txt, ok := item.(*mcp.TextContent); ok {
			sb.WriteString(txt.Text)
		}
	}
	return sb.String()
}

func TestRunFlowTool_ReturnsLastMessage(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"outputs":[{"messages":[{"message":"pong: ok"}]}]}`))
	}))
	defer ts.Close()

	cfg := &config.Config{LangflowURL: ts.URL, APIKey: "k"}
	res := runFlowViaMCP(t, client.NewClient(cfg), map[string]any{
		"flow_id":     "f-1",
		"input_value": "ping",
	})

	if gotPath != "/api/v1/run/f-1" {
		t.Fatalf("path=%s (нужен без трейлинг-слэша)", gotPath)
	}
	if !strings.Contains(res, "pong: ok") {
		t.Fatalf("missing last message: %s", res)
	}
}

func TestRunFlowTool_RawFallback(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"no":"message-keys","n":42}`))
	}))
	defer ts.Close()

	cfg := &config.Config{LangflowURL: ts.URL, APIKey: "k"}
	res := runFlowViaMCP(t, client.NewClient(cfg), map[string]any{"flow_id": "f-2"})
	if !strings.Contains(res, "message-keys") {
		t.Fatalf("raw fallback expected, got: %s", res)
	}
}
