package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	flag.Parse()

	cfg := config.Load()

	langflowClient := client.NewClient(cfg)

	server := mcp.NewServer(
		&mcp.Implementation{
			Name:    "langflow-builder",
			Version: "1.0.0",
		},
		&mcp.ServerOptions{
			Instructions: "LangFlow MCP Server - manage LangFlow flows, nodes, connections, and components.",
		},
	)

	tools.RegisterAll(server, langflowClient, cfg)

	stdio, httpAddr := config.TransportFlags()

	if httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			return server
		}, nil)
		log.Printf("LangFlow MCP Server listening on %s", httpAddr)
		log.Fatal(http.ListenAndServe(httpAddr, handler))
	} else if stdio || httpAddr == "" {
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			log.Fatal(err)
		}
	}
}
