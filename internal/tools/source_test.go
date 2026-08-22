package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/instructions"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// newTestConfig returns a config pointing SourceCacheDir at a fresh temp dir.
func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	return &config.Config{
		SourceCacheDir: t.TempDir(),
	}
}

// makeFixtureRepo creates a minimal "langflow" git repository inside cfg's
// cache dir so tests exercise the real code paths without network access.
func makeFixtureRepo(t *testing.T, cfg *config.Config) string {
	t.Helper()

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	repoPath, err := repoRoot(cfg)
	if err != nil {
		t.Fatalf("repoRoot failed: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoPath, "src", "backend"), 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	files := map[string]string{
		"README.md":            "# Fixture Repo\n\nA tiny stand-in for langflow.\n",
		"main.py":              "class Component:\n    pass\n\n\nclass Other(Component):\n    pass\n",
		"src/backend/app.py":   "from langflow.custom import Component\n\nclass App(Component):\n    pass\n\n\ndef build():\n    return Component\n",
		"src/backend/utils.py": "class Utils:\n    pass\n\ndef helper(value):\n    return value.upper()\n",
	}
	for rel, content := range files {
		full := filepath.Join(repoPath, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir failed: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write failed: %v", err)
		}
	}

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, out)
		}
	}

	run("init", "-q")
	run("add", ".")
	run("commit", "-q", "-m", "fixture")

	return repoPath
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected non-empty result content")
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return tc.Text
}

// ── Pure helpers ──────────────────────────────────────────────────────────────

func TestSourceCacheDirExpandsHome(t *testing.T) {
	cfg := &config.Config{SourceCacheDir: "~/.cache/langflow-mcp"}
	dir, err := getSourceCacheDir(cfg)
	if err != nil {
		t.Fatalf("getSourceCacheDir failed: %v", err)
	}
	home, _ := os.UserHomeDir()
	if want := filepath.Join(home, ".cache", "langflow-mcp"); dir != want {
		t.Errorf("got %s, want %s", dir, want)
	}

	cfg.SourceCacheDir = "/tmp/opencode/abs-cache"
	dir, err = getSourceCacheDir(cfg)
	if err != nil || dir != "/tmp/opencode/abs-cache" {
		t.Errorf("absolute path handling broken: dir=%s err=%v", dir, err)
	}

	cfg.SourceCacheDir = ""
	if _, err := getSourceCacheDir(cfg); err == nil {
		t.Error("expected error for empty cache dir")
	}
}

func TestSourceIsWithinRepo(t *testing.T) {
	repo := "/tmp/opencode/repo"
	cases := []struct {
		target string
		want   bool
	}{
		{"/tmp/opencode/repo/file.py", true},
		{"/tmp/opencode/repo/sub/dir", true},
		{"/tmp/opencode/repo", true},
		{"/tmp/opencode/repo-evil/file.py", false}, // prefix-collision guard
		{"/etc/passwd", false},
		{"/tmp/opencode/repo/../secrets.txt", false},
	}
	for _, tc := range cases {
		if got := isWithinRepo(repo, tc.target); got != tc.want {
			t.Errorf("isWithinRepo(%s) = %v, want %v", tc.target, got, tc.want)
		}
	}
}

// ── Setup ─────────────────────────────────────────────────────────────────────

func TestSourceSetupWithExistingRepo(t *testing.T) {
	cfg := newTestConfig(t)
	makeFixtureRepo(t, cfg)

	res, err := setupLangflowSource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setupLangflowSource failed: %v", err)
	}
	text := resultText(t, res)

	if !strings.Contains(text, "Langflow source repository ready") {
		t.Errorf("missing ready message in: %s", text)
	}
	if !strings.Contains(text, "Branch: ") {
		t.Errorf("missing branch info in: %s", text)
	}
	if !strings.Contains(text, "Latest commit:") || !strings.Contains(text, "fixture") {
		t.Errorf("missing commit info in: %s", text)
	}
}

// TestSetupLangflowSourceClonesRealRepo exercises the actual clone path.
// It is skipped unless LANGFLOW_MCP_TEST_NETWORK=1 because cloning the full
// langflow repository requires network access and takes minutes.
func TestSourceSetupClonesRealRepo(t *testing.T) {
	if os.Getenv("LANGFLOW_MCP_TEST_NETWORK") != "1" {
		t.Skip("set LANGFLOW_MCP_TEST_NETWORK=1 to run the network clone test")
	}

	cfg := &config.Config{
		SourceCacheDir: filepath.Join(os.TempDir(), "langflow-mcp-network-test"),
	}
	defer os.RemoveAll(cfg.SourceCacheDir)

	res, err := setupLangflowSource(context.Background(), cfg)
	if err != nil {
		t.Fatalf("setupLangflowSource failed: %v", err)
	}
	if text := resultText(t, res); !strings.Contains(text, "ready") {
		t.Errorf("unexpected output: %s", text)
	}
}

// ── Explore (search) ─────────────────────────────────────────────────────────

func TestSourceExploreSearch(t *testing.T) {
	cfg := newTestConfig(t)
	repo := makeFixtureRepo(t, cfg)
	_ = repo
	ctx := context.Background()

	// Known term with path filter.
	res, err := exploreLangflow(ctx, cfg, schema.ExploreLangflowInput{
		Query:      "class Component",
		PathFilter: "*.py",
	})
	if err != nil {
		t.Fatalf("exploreLangflow failed: %v", err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "main.py") || !strings.Contains(text, "class Component") {
		t.Errorf("expected matches from main.py, got:\n%s", text)
	}
	if strings.Contains(text, "README.md") {
		t.Errorf("path filter *.py not honored, got:\n%s", text)
	}

	// Subdirectory filter.
	res, err = exploreLangflow(ctx, cfg, schema.ExploreLangflowInput{
		Query:      "Component",
		PathFilter: "src/",
	})
	if err != nil {
		t.Fatalf("exploreLangflow failed: %v", err)
	}
	text = resultText(t, res)
	if !strings.Contains(text, "src/backend/app.py") {
		t.Errorf("expected src/backend/app.py match, got:\n%s", text)
	}
	if strings.Contains(text, "main.py:") {
		t.Errorf("results outside src/ leaked through filter, got:\n%s", text)
	}

	// No matches.
	res, err = exploreLangflow(ctx, cfg, schema.ExploreLangflowInput{
		Query: "definitely_no_such_token_zzz123",
	})
	if err != nil {
		t.Fatalf("exploreLangflow failed: %v", err)
	}
	if text := resultText(t, res); !strings.Contains(text, "No matches found.") {
		t.Errorf("expected no-match message, got: %s", text)
	}

	// MaxResults caps output (fixture has 4 "class" occurrences).
	res, err = exploreLangflow(ctx, cfg, schema.ExploreLangflowInput{
		Query:      "class",
		MaxResults: 2,
	})
	if err != nil {
		t.Fatalf("exploreLangflow failed: %v", err)
	}
	text = resultText(t, res)
	if !strings.Contains(text, "(showing first 2)") {
		t.Errorf("expected truncation notice, got: %s", text)
	}
	parts := strings.SplitN(text, "\n\n", 2)
	if len(parts) < 2 {
		t.Fatalf("unexpected result format: %s", text)
	}
	matchLines := strings.Split(strings.TrimSpace(parts[1]), "\n")
	if len(matchLines) != 2 {
		t.Errorf("expected exactly 2 capped lines, got %d:\n%s", len(matchLines), text)
	}

	// Empty query rejected.
	res, _ = exploreLangflow(ctx, cfg, schema.ExploreLangflowInput{Query: "   "})
	if res == nil || !res.IsError {
		t.Error("empty query should produce an error result")
	}
}

func TestSourceExploreRequiresSetup(t *testing.T) {
	cfg := newTestConfig(t) // no fixture repo

	res, err := exploreLangflow(context.Background(), cfg, schema.ExploreLangflowInput{Query: "anything"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := resultText(t, res)
	if !res.IsError || !strings.Contains(text, "setup_langflow_source") {
		t.Errorf("expected setup hint error, got IsError=%v text=%s", res.IsError, text)
	}
}

// ── Read ─────────────────────────────────────────────────────────────────────

func TestSourceReadFile(t *testing.T) {
	cfg := newTestConfig(t)
	makeFixtureRepo(t, cfg)
	ctx := context.Background()

	// Full file gets line numbers starting at 1.
	res, err := readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "main.py"})
	if err != nil {
		t.Fatalf("readLangflowFile failed: %v", err)
	}
	text := resultText(t, res)
	if !strings.HasPrefix(text, "1: class Component:\n") {
		t.Errorf("missing numbered first line, got:\n%s", text)
	}
	if !strings.Contains(text, "5: class Other(Component):") {
		t.Errorf("missing line 5, got:\n%s", text)
	}

	// Line range.
	res, err = readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "main.py", StartLine: 2, EndLine: 3})
	if err != nil {
		t.Fatalf("readLangflowFile failed: %v", err)
	}
	text = resultText(t, res)
	if strings.TrimSpace(text) != "2:     pass\n3:" {
		t.Errorf("range read mismatch, got:\n%q", text)
	}
	if strings.Contains(text, "1: ") || strings.Contains(text, "4: ") {
		t.Errorf("lines outside range leaked, got:\n%s", text)
	}

	// EndLine beyond EOF clamps.
	res, err = readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "README.md", StartLine: 999})
	if err != nil {
		t.Fatalf("readLangflowFile failed: %v", err)
	}
	if text := resultText(t, res); !strings.Contains(text, "invalid line range") {
		t.Errorf("start beyond EOF should error, got: %s", text)
	}

	res, err = readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "README.md", StartLine: 1, EndLine: 5000})
	if err != nil {
		t.Fatalf("readLangflowFile failed: %v", err)
	}
	if text := resultText(t, res); !strings.Contains(text, "3: A tiny") {
		t.Errorf("clamp-to-EOF failed, got:\n%s", text)
	}

	// Missing file errors cleanly.
	res, err = readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "no_such_file.py"})
	if err != nil {
		t.Fatalf("unexpected transport error: %v", err)
	}
	if text := resultText(t, res); !res.IsError || !strings.Contains(text, "Error reading file") {
		t.Errorf("expected file-read error, got: %s", text)
	}

	// Path traversal blocked.
	res, _ = readLangflowFile(ctx, cfg, schema.ReadLangflowFileInput{FilePath: "../../../etc/passwd"})
	if res == nil || !res.IsError {
		t.Error("path traversal must be blocked")
	}
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestSourceListDirectory(t *testing.T) {
	cfg := newTestConfig(t)
	repo := makeFixtureRepo(t, cfg)
	ctx := context.Background()

	// Root listing shows dirs and files but hides .git.
	res, err := listLangflowDirectory(ctx, cfg, schema.ListLangflowDirectoryInput{Directory: "."})
	if err != nil {
		t.Fatalf("listLangflowDirectory failed: %v", err)
	}
	text := resultText(t, res)
	if !strings.Contains(text, "Directories:") || !strings.Contains(text, "src/") {
		t.Errorf("root listing missing directories, got:\n%s", text)
	}
	if !strings.Contains(text, "Files:") || !strings.Contains(text, "README.md") || !strings.Contains(text, "main.py") {
		t.Errorf("root listing missing files, got:\n%s", text)
	}
	if strings.Contains(text, ".git") {
		t.Errorf(".git should be hidden, got:\n%s", text)
	}

	// Nested directory.
	res, err = listLangflowDirectory(ctx, cfg, schema.ListLangflowDirectoryInput{Directory: "src/backend"})
	if err != nil {
		t.Fatalf("listLangflowDirectory failed: %v", err)
	}
	text = resultText(t, res)
	for _, want := range []string{"app.py", "utils.py"} {
		if !strings.Contains(text, want) {
			t.Errorf("nested listing missing %s, got:\n%s", want, text)
		}
	}

	// Empty directory reports empty.
	emptyRel := filepath.Join(repo, "nothing")
	if err := os.MkdirAll(emptyRel, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	res, err = listLangflowDirectory(ctx, cfg, schema.ListLangflowDirectoryInput{Directory: "nothing"})
	if err != nil {
		t.Fatalf("listLangflowDirectory failed: %v", err)
	}
	if text := resultText(t, res); !strings.Contains(text, "(empty)") {
		t.Errorf("expected empty marker, got: %s", text)
	}

	// Nonexistent directory errors.
	res, _ = listLangflowDirectory(ctx, cfg, schema.ListLangflowDirectoryInput{Directory: "does/not/exist"})
	if res == nil || !res.IsError {
		t.Error("nonexistent directory should error")
	}

	// Traversal blocked.
	res, _ = listLangflowDirectory(ctx, cfg, schema.ListLangflowDirectoryInput{Directory: "../../.."})
	if res == nil || !res.IsError {
		t.Error("directory traversal must be blocked")
	}
}

func TestSourceListDirRequiresSetup(t *testing.T) {
	cfg := newTestConfig(t)

	res, err := listLangflowDirectory(context.Background(), cfg, schema.ListLangflowDirectoryInput{Directory: "."})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !res.IsError || !strings.Contains(resultText(t, res), "setup_langflow_source") {
		t.Errorf("expected setup hint error, got: %+v", res)
	}
}

// ── Concepts ─────────────────────────────────────────────────────────────────

func TestSourceConceptsAllTopics(t *testing.T) {
	topics := []string{
		"custom_components", "tool_mode", "building", "component_structure",
		"outputs", "inputs", "connections", "common_mistakes", "layout",
	}
	for _, topic := range topics {
		res := langflowConcepts(schema.LangflowConceptsInput{Topic: topic})
		text := resultText(t, res)
		if len(text) < 100 {
			t.Errorf("topic %s returned too little content (%d chars)", topic, len(text))
		}
		if strings.Contains(text, instructions.OverviewMarker) {
			t.Errorf("topic %s returned the overview instead of its own doc", topic)
		}
	}
}

func TestSourceConceptsUnknownReturnsOverview(t *testing.T) {
	res := langflowConcepts(schema.LangflowConceptsInput{Topic: "not_a_real_topic"})
	text := resultText(t, res)
	if !strings.Contains(text, instructions.OverviewMarker) {
		t.Errorf("unknown topic should return overview containing marker, got first 80 chars: %.80s", text)
	}

	// Empty topic also returns overview.
	res = langflowConcepts(schema.LangflowConceptsInput{})
	if !strings.Contains(resultText(t, res), instructions.OverviewMarker) {
		t.Error("empty topic should return overview")
	}
}
