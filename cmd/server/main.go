package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"log/slog"
	"net/http"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/logging"
	"github.com/ag/ai-agent-builder/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	flag.Parse()

	cfg := config.Load()

	logger := logging.NewLogger(cfg.LogLevel)
	slog.SetDefault(logger)
	langflowClient := client.NewClient(cfg)
	langflowClient.SetLogger(logger)

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

	_, httpAddr := config.TransportFlags()

	// stdio is the default transport. HTTP is used only when --http was
	// explicitly passed (with or without an address).
	if config.HTTPRequested() {
		if httpAddr == "" {
			httpAddr = resolveHTTPAddr(httpAddr, cfg)
		}
		slog.Info("langflow mcp server listening", "addr", httpAddr)
		mcpHandler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
			return server
		}, nil)

		mux := http.NewServeMux()
		mux.Handle("/health", healthHandler())
		mux.Handle("/mcp", mcpHandler)
		mux.Handle("/", mcpHandler)

		handler := withCORS(mux)

		log.Fatal(http.ListenAndServe(httpAddr, handler))
	} else {
		if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
			slog.Error("server stopped with error", "error", err.Error())
			log.Fatal(err)
		}
	}
}

// resolveHTTPAddr determines the HTTP listen address. If --http was not
// specified, fall back to the config host/port defaults.
func resolveHTTPAddr(httpAddr string, cfg *config.Config) string {
	if httpAddr != "" {
		return httpAddr
	}
	if cfg.HTTPHost == "" && cfg.HTTPPort == "" {
		return ""
	}
	host := cfg.HTTPHost
	if host == "" {
		host = "0.0.0.0"
	}
	port := cfg.HTTPPort
	if port == "" {
		port = "8080"
	}
	return host + ":" + port
}

func healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":  "ok",
			"service": "langflow-builder",
		})
	})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Mcp-Session-Id, mcp-protocol-version")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
