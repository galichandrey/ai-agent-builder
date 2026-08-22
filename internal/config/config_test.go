package config

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestSplitHostPort tests the splitHostPort helper function.
func TestSplitHostPort(t *testing.T) {
	tests := []struct {
		name     string
		addr     string
		wantHost string
		wantPort string
		wantErr  bool
	}{
		{"host:port", "localhost:8080", "localhost", "8080", false},
		{"port only with colon", ":8080", "0.0.0.0", "8080", false},
		{"host only", "localhost", "localhost", "", false},
		{"empty string", "", "", "", true},
		{"whitespace", "  ", "", "", true},
		{"ip:port", "192.168.1.1:3000", "192.168.1.1", "3000", false},
		{"multiple colons", "host:1:2", "host", "1:2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port, err := splitHostPort(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitHostPort(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
				return
			}
			if host != tt.wantHost {
				t.Errorf("splitHostPort(%q) host = %q, want %q", tt.addr, host, tt.wantHost)
			}
			if port != tt.wantPort {
				t.Errorf("splitHostPort(%q) port = %q, want %q", tt.addr, port, tt.wantPort)
			}
		})
	}
}

// runSubprocess runs the test binary with given CLI flags and env, returns parsed Config.
func runSubprocess(t *testing.T, cliArgs []string, env map[string]string) *Config {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get test binary path: %v", err)
	}

	cmdArgs := append([]string{binary, "-test.run=TestLoadScenario", "-test.v"}, cliArgs...)
	cmd := exec.Command(cmdArgs[0], cmdArgs[1:]...)

	// Start with parent env, but clear any LANGFLOW_MCP_* vars to prevent leakage
	cmd.Env = os.Environ()
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], "LANGFLOW_MCP_") {
			cmd.Env = append(cmd.Env, parts[0]+"=")
		}
	}
	cmd.Env = append(cmd.Env, "CONFIG_TEST_SCENARIO=1")
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("subprocess failed: %v\nOutput:\n%s", err, out)
	}

	// Find JSON output line
	var result Config
	for _, line := range splitLines(string(out)) {
		if len(line) > 0 && line[0] == '{' {
			if err := json.Unmarshal([]byte(line), &result); err != nil {
				t.Fatalf("failed to parse config JSON: %v\nLine: %s", err, line)
			}
			return &result
		}
	}
	t.Fatalf("no config JSON found in subprocess output:\n%s", out)
	return nil
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// TestLoadScenario is the subprocess entry point for testing Load() with different args/env.
// It is invoked by runSubprocess and outputs the Config as JSON to stdout.
func TestLoadScenario(t *testing.T) {
	if os.Getenv("CONFIG_TEST_SCENARIO") == "" {
		return
	}

	cfg := Load()
	out, _ := json.Marshal(cfg)
	print(string(out) + "\n")
}

func TestLoadDefaults(t *testing.T) {
	cfg := runSubprocess(t, nil, nil)

	if cfg.LangflowURL != "http://localhost:7860" {
		t.Errorf("LangflowURL = %q, want %q", cfg.LangflowURL, "http://localhost:7860")
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty", cfg.APIKey)
	}
	if cfg.CacheTTL != 300 {
		t.Errorf("CacheTTL = %d, want 300", cfg.CacheTTL)
	}
	if cfg.RequestTimeout != 120 {
		t.Errorf("RequestTimeout = %d, want 120", cfg.RequestTimeout)
	}
	if cfg.AutoBackup != false {
		t.Errorf("AutoBackup = %v, want false", cfg.AutoBackup)
	}
	if cfg.BackupFolder != "MCP Backups" {
		t.Errorf("BackupFolder = %q, want %q", cfg.BackupFolder, "MCP Backups")
	}
	if len(cfg.CustomHeaders) != 0 {
		t.Errorf("CustomHeaders = %v, want empty map", cfg.CustomHeaders)
	}
	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "8080")
	}
	if cfg.HTTPHost != "0.0.0.0" {
		t.Errorf("HTTPHost = %q, want %q", cfg.HTTPHost, "0.0.0.0")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
	if cfg.LangflowVersion != "" {
		t.Errorf("LangflowVersion = %q, want empty", cfg.LangflowVersion)
	}
	if cfg.SourceCacheDir != "~/.cache/langflow-mcp" {
		t.Errorf("SourceCacheDir = %q, want %q", cfg.SourceCacheDir, "~/.cache/langflow-mcp")
	}
}

func TestLoadEnvOverride(t *testing.T) {
	env := map[string]string{
		"LANGFLOW_MCP_LANGFLOW_URL":     "http://env-host:9090",
		"LANGFLOW_MCP_API_KEY":          "env-key-123",
		"LANGFLOW_MCP_CACHE_TTL":        "600",
		"LANGFLOW_MCP_REQUEST_TIMEOUT":  "30",
		"LANGFLOW_MCP_AUTO_BACKUP":      "true",
		"LANGFLOW_MCP_BACKUP_FOLDER":    "Env Backups",
		"LANGFLOW_MCP_HTTP_PORT":        "9090",
		"LANGFLOW_MCP_HTTP_HOST":        "127.0.0.1",
		"LANGFLOW_MCP_LOG_LEVEL":        "debug",
		"LANGFLOW_MCP_LANGFLOW_VERSION": "1.0.0",
		"LANGFLOW_MCP_SOURCE_CACHE_DIR": "/tmp/cache",
	}

	cfg := runSubprocess(t, nil, env)

	if cfg.LangflowURL != "http://env-host:9090" {
		t.Errorf("LangflowURL = %q, want %q", cfg.LangflowURL, "http://env-host:9090")
	}
	if cfg.APIKey != "env-key-123" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "env-key-123")
	}
	if cfg.CacheTTL != 600 {
		t.Errorf("CacheTTL = %d, want 600", cfg.CacheTTL)
	}
	if cfg.RequestTimeout != 30 {
		t.Errorf("RequestTimeout = %d, want 30", cfg.RequestTimeout)
	}
	if cfg.AutoBackup != true {
		t.Errorf("AutoBackup = %v, want true", cfg.AutoBackup)
	}
	if cfg.BackupFolder != "Env Backups" {
		t.Errorf("BackupFolder = %q, want %q", cfg.BackupFolder, "Env Backups")
	}
	if cfg.HTTPPort != "9090" {
		t.Errorf("HTTPPort = %q, want %q", cfg.HTTPPort, "9090")
	}
	if cfg.HTTPHost != "127.0.0.1" {
		t.Errorf("HTTPHost = %q, want %q", cfg.HTTPHost, "127.0.0.1")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	if cfg.LangflowVersion != "1.0.0" {
		t.Errorf("LangflowVersion = %q, want %q", cfg.LangflowVersion, "1.0.0")
	}
	if cfg.SourceCacheDir != "/tmp/cache" {
		t.Errorf("SourceCacheDir = %q, want %q", cfg.SourceCacheDir, "/tmp/cache")
	}
}

func TestLoadCLIOverride(t *testing.T) {
	env := map[string]string{
		"LANGFLOW_MCP_LANGFLOW_URL": "http://env-host:9090",
		"LANGFLOW_MCP_API_KEY":      "env-key",
		"LANGFLOW_MCP_CACHE_TTL":    "600",
	}

	cliArgs := []string{
		"--langflow-url", "http://cli-host:3000",
		"--api-key", "cli-key",
		"--cache-ttl", "120",
	}

	cfg := runSubprocess(t, cliArgs, env)

	if cfg.LangflowURL != "http://cli-host:3000" {
		t.Errorf("LangflowURL = %q, want CLI value %q", cfg.LangflowURL, "http://cli-host:3000")
	}
	if cfg.APIKey != "cli-key" {
		t.Errorf("APIKey = %q, want CLI value %q", cfg.APIKey, "cli-key")
	}
	if cfg.CacheTTL != 120 {
		t.Errorf("CacheTTL = %d, want CLI value 120", cfg.CacheTTL)
	}
}

func TestLoadCustomHeadersJSON(t *testing.T) {
	env := map[string]string{
		"LANGFLOW_MCP_CUSTOM_HEADERS": `{"X-Custom":"value","X-Other":"other"}`,
	}

	cfg := runSubprocess(t, nil, env)

	if len(cfg.CustomHeaders) != 2 {
		t.Fatalf("CustomHeaders has %d entries, want 2", len(cfg.CustomHeaders))
	}
	if cfg.CustomHeaders["X-Custom"] != "value" {
		t.Errorf("CustomHeaders[X-Custom] = %q, want %q", cfg.CustomHeaders["X-Custom"], "value")
	}
	if cfg.CustomHeaders["X-Other"] != "other" {
		t.Errorf("CustomHeaders[X-Other] = %q, want %q", cfg.CustomHeaders["X-Other"], "other")
	}
}

func TestLoadCustomHeadersEmpty(t *testing.T) {
	cfg := runSubprocess(t, nil, nil)

	if len(cfg.CustomHeaders) != 0 {
		t.Errorf("CustomHeaders = %v, want empty map", cfg.CustomHeaders)
	}
}
