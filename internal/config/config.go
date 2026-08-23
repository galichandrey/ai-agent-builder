package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the LangFlow MCP server.
type Config struct {
	LangflowURL     string
	APIKey          string
	CacheTTL        int
	RequestTimeout  int
	AutoBackup      bool
	BackupFolder    string
	CustomHeaders   map[string]string
	HTTPPort        string
	HTTPHost        string
	LogLevel        string
	LangflowVersion string
	SourceCacheDir  string
}

// CLI flag variables registered at package level so they're available
// before flag.Parse() is called by the test framework or main.
var (
	flagLangflowURL     string
	flagAPIKey          string
	flagCacheTTL        int
	flagRequestTimeout  int
	flagAutoBackup      bool
	flagBackupFolder    string
	flagCustomHeaders   string
	flagHTTPPort        string
	flagHTTPHost        string
	flagLogLevel        string
	flagLangflowVersion string
	flagSourceCacheDir  string
	flagStdio           bool
	flagHTTPAddr        string
)

func init() {
	flag.StringVar(&flagLangflowURL, "langflow-url", "http://localhost:7860", "LangFlow API URL")
	flag.StringVar(&flagAPIKey, "api-key", "", "API key for LangFlow")
	flag.IntVar(&flagCacheTTL, "cache-ttl", 300, "Component schema cache TTL (seconds)")
	flag.IntVar(&flagRequestTimeout, "request-timeout", 120, "HTTP request timeout (seconds)")
	flag.BoolVar(&flagAutoBackup, "auto-backup", false, "Auto-backup before mutations")
	flag.StringVar(&flagBackupFolder, "backup-folder", "MCP Backups", "Backup folder name")
	flag.StringVar(&flagCustomHeaders, "custom-headers", "{}", "Extra HTTP headers (JSON)")
	flag.StringVar(&flagHTTPPort, "http-port", "8080", "HTTP transport port")
	flag.StringVar(&flagHTTPHost, "http-host", "0.0.0.0", "HTTP transport host")
	flag.StringVar(&flagLogLevel, "log-level", "info", "Log level")
	flag.StringVar(&flagLangflowVersion, "langflow-version", "", "Override LangFlow version")
	flag.StringVar(&flagSourceCacheDir, "source-cache-dir", "~/.cache/langflow-mcp", "Source cache directory")
	flag.BoolVar(&flagStdio, "stdio", false, "Use stdio transport (default)")
	flag.StringVar(&flagHTTPAddr, "http", "", "Use streamable HTTP transport (e.g., :8080)")
}

// defaultConfig returns a Config with all default values.
func defaultConfig() *Config {
	return &Config{
		LangflowURL:     "http://localhost:7860",
		APIKey:          "",
		CacheTTL:        300,
		RequestTimeout:  120,
		AutoBackup:      false,
		BackupFolder:    "MCP Backups",
		CustomHeaders:   map[string]string{},
		HTTPPort:        "8080",
		HTTPHost:        "0.0.0.0",
		LogLevel:        "info",
		LangflowVersion: "",
		SourceCacheDir:  "~/.cache/langflow-mcp",
	}
}

// flagWasSet checks if a named flag was explicitly set on the command line.
func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

// splitHostPort splits an address string into host and port.
func splitHostPort(addr string) (host, port string, err error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return "", "", fmt.Errorf("empty address")
	}

	if strings.HasPrefix(addr, ":") {
		return "0.0.0.0", addr[1:], nil
	}

	parts := strings.SplitN(addr, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1], nil
	}

	return addr, "", nil
}

// Load reads configuration from CLI flags and environment variables.
// Priority: CLI flag > ENV > default value.
func Load() *Config {
	// Build config from flag values (already parsed by test framework or main)
	cfg := &Config{
		LangflowURL:     flagLangflowURL,
		APIKey:          flagAPIKey,
		CacheTTL:        flagCacheTTL,
		RequestTimeout:  flagRequestTimeout,
		AutoBackup:      flagAutoBackup,
		BackupFolder:    flagBackupFolder,
		HTTPPort:        flagHTTPPort,
		HTTPHost:        flagHTTPHost,
		LogLevel:        flagLogLevel,
		LangflowVersion: flagLangflowVersion,
		SourceCacheDir:  flagSourceCacheDir,
	}

	// Apply ENV overrides only if the CLI flag was NOT explicitly set
	applyEnvOverride("langflow-url", "LANGFLOW_MCP_LANGFLOW_URL", func(v string) { cfg.LangflowURL = v })
	applyEnvOverride("api-key", "LANGFLOW_MCP_API_KEY", func(v string) { cfg.APIKey = v })
	applyEnvOverrideInt("cache-ttl", "LANGFLOW_MCP_CACHE_TTL", func(v int) { cfg.CacheTTL = v })
	applyEnvOverrideInt("request-timeout", "LANGFLOW_MCP_REQUEST_TIMEOUT", func(v int) { cfg.RequestTimeout = v })
	applyEnvOverrideBool("auto-backup", "LANGFLOW_MCP_AUTO_BACKUP", func(v bool) { cfg.AutoBackup = v })
	applyEnvOverride("backup-folder", "LANGFLOW_MCP_BACKUP_FOLDER", func(v string) { cfg.BackupFolder = v })
	applyEnvOverride("http-port", "LANGFLOW_MCP_HTTP_PORT", func(v string) { cfg.HTTPPort = v })
	applyEnvOverride("http-host", "LANGFLOW_MCP_HTTP_HOST", func(v string) { cfg.HTTPHost = v })
	applyEnvOverride("log-level", "LANGFLOW_MCP_LOG_LEVEL", func(v string) { cfg.LogLevel = v })
	applyEnvOverride("langflow-version", "LANGFLOW_MCP_LANGFLOW_VERSION", func(v string) { cfg.LangflowVersion = v })
	applyEnvOverride("source-cache-dir", "LANGFLOW_MCP_SOURCE_CACHE_DIR", func(v string) { cfg.SourceCacheDir = v })

	// Handle custom headers: CLI flag or ENV
	customHeadersRaw := flagCustomHeaders
	if !flagWasSet("custom-headers") {
		if v := os.Getenv("LANGFLOW_MCP_CUSTOM_HEADERS"); v != "" {
			customHeadersRaw = v
		}
	}

	// Apply --http transport flag (overrides host/port if specified)
	if flagHTTPAddr != "" {
		host, port, err := splitHostPort(flagHTTPAddr)
		if err == nil {
			if host != "" {
				cfg.HTTPHost = host
			}
			if port != "" {
				cfg.HTTPPort = port
			}
		}
	}

	// Parse custom headers JSON
	cfg.CustomHeaders = map[string]string{}
	if customHeadersRaw != "" && customHeadersRaw != "{}" {
		var headers map[string]string
		if err := json.Unmarshal([]byte(customHeadersRaw), &headers); err == nil {
			cfg.CustomHeaders = headers
		}
	}

	return cfg
}

func applyEnvOverride(flagName, envName string, setter func(string)) {
	if !flagWasSet(flagName) {
		if v := os.Getenv(envName); v != "" {
			setter(v)
		}
	}
}

func applyEnvOverrideInt(flagName, envName string, setter func(int)) {
	if !flagWasSet(flagName) {
		if v := os.Getenv(envName); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				setter(n)
			}
		}
	}
}

func applyEnvOverrideBool(flagName, envName string, setter func(bool)) {
	if !flagWasSet(flagName) {
		if v := os.Getenv(envName); v != "" {
			if b, err := strconv.ParseBool(v); err == nil {
				setter(b)
			}
		}
	}
}

// TransportFlags returns the transport selection flag values.
// These are not part of Config but are needed by main.go.
func TransportFlags() (stdio bool, httpAddr string) {
	return flagStdio, flagHTTPAddr
}

// HTTPRequested reports whether the --http flag was explicitly passed on the
// command line. stdio is the default transport; HTTP is only enabled when the
// user opts in via --http.
func HTTPRequested() bool {
	return flagWasSet("http")
}
