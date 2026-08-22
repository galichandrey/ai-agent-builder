package tools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/instructions"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// LangflowRepoURL is the upstream Langflow source repository.
const LangflowRepoURL = "https://github.com/langflow-ai/langflow.git"

// registerSourceTools registers the 5 source exploration MCP tools.
func registerSourceTools(server *mcp.Server, _ *client.LangflowClient, cfg *config.Config) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "setup_langflow_source",
		Description: "Clone/update the Langflow source code repository for exploration. PREREQUISITE: run this before explore_langflow, read_langflow_file, or list_langflow_directory.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ schema.SetupLangflowSourceInput) (*mcp.CallToolResult, any, error) {
		result, err := setupLangflowSource(ctx, cfg)
		return result, nil, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "explore_langflow",
		Description: "Search the Langflow source code using git grep. Returns matching files with line numbers and content snippets.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ExploreLangflowInput) (*mcp.CallToolResult, any, error) {
		result, err := exploreLangflow(ctx, cfg, input)
		return result, nil, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "read_langflow_file",
		Description: "Read a specific file from the Langflow source code repository with optional line range. File must exist within the cloned repo.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ReadLangflowFileInput) (*mcp.CallToolResult, any, error) {
		result, err := readLangflowFile(ctx, cfg, input)
		return result, nil, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_langflow_directory",
		Description: "List files and subdirectories in a directory of the Langflow source code repository.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.ListLangflowDirectoryInput) (*mcp.CallToolResult, any, error) {
		result, err := listLangflowDirectory(ctx, cfg, input)
		return result, nil, err
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "langflow_concepts",
		Description: "Quick reference documentation about Langflow concepts. Topics: custom_components, tool_mode, building, component_structure, outputs, inputs, connections, common_mistakes, layout. Empty topic returns the overview.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input schema.LangflowConceptsInput) (*mcp.CallToolResult, any, error) {
		return langflowConcepts(input), nil, nil
	})
}

func textResult(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
	}
}

func textError(format string, args ...any) *mcp.CallToolResult {
	res := textResult(format, args...)
	res.IsError = true
	return res
}

// expandCacheDir expands ~/ prefixes in SourceCacheDir.
func getSourceCacheDir(cfg *config.Config) (string, error) {
	dir := cfg.SourceCacheDir
	if strings.HasPrefix(dir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dir = filepath.Join(home, dir[2:])
	}
	if dir == "" {
		return "", fmt.Errorf("source cache directory is not configured")
	}
	return dir, nil
}

// repoRoot returns the expected path of the cloned langflow repo.
func repoRoot(cfg *config.Config) (string, error) {
	dir, err := getSourceCacheDir(cfg)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "langflow"), nil
}

// isWithinRepo verifies that target resolves inside repo (prevents path traversal).
func isWithinRepo(repo, target string) bool {
	absRepo, err := filepath.Abs(repo)
	if err != nil {
		return false
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absRepo, absTarget)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ensureLangflowRepo clones the repo if missing, or refreshes it best-effort
// if present. A failed fetch (e.g., offline) never fails the call — the
// cached copy remains usable.
func ensureLangflowRepo(ctx context.Context, cfg *config.Config) (string, error) {
	repoPath, err := repoRoot(cfg)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		// Best-effort update of existing clone.
		cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "fetch", "origin")
		if out, ferr := cmd.CombinedOutput(); ferr != nil {
			_ = out // tolerate: offline or restricted environment keeps cached copy usable
		}
		return repoPath, nil
	}

	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", LangflowRepoURL, repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone langflow repository: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return repoPath, nil
}

// requireRepo ensures the repo has been set up previously (without network access).
func requireRepo(cfg *config.Config) (string, error) {
	repoPath, err := repoRoot(cfg)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return "", fmt.Errorf("langflow source repository not found at %s — run setup_langflow_source first", repoPath)
	}
	return repoPath, nil
}

func setupLangflowSource(ctx context.Context, cfg *config.Config) (*mcp.CallToolResult, error) {
	repoPath, err := ensureLangflowRepo(ctx, cfg)
	if err != nil {
		return textError("Error: %v", err), nil
	}

	cmd := exec.CommandContext(ctx, "git", "-C", repoPath, "log", "-1", "--oneline")
	output, _ := cmd.Output()
	commit := strings.TrimSpace(string(output))

	cmd = exec.CommandContext(ctx, "git", "-C", repoPath, "branch", "--show-current")
	output, _ = cmd.Output()
	branch := strings.TrimSpace(string(output))

	return textResult("Langflow source repository ready at: %s\nBranch: %s\nLatest commit: %s", repoPath, branch, commit), nil
}

func exploreLangflow(_ context.Context, cfg *config.Config, input schema.ExploreLangflowInput) (*mcp.CallToolResult, error) {
	repoPath, err := requireRepo(cfg)
	if err != nil {
		return textError("Error: %v", err), nil
	}

	query := strings.TrimSpace(input.Query)
	if query == "" {
		return textError("Error: query must not be empty"), nil
	}

	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 20
	}

	// git grep filters paths via a trailing pathspec (--include is not supported).
	args := []string{"grep", "-n", "--color=never", "-I", "-e", query}
	if pf := strings.TrimSpace(input.PathFilter); pf != "" {
		args = append(args, "--", pf)
	}

	cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	cmd.Dir = repoPath
	output, err := cmd.Output()

	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return textResult("No matches found."), nil
		}
		return textError("Search error: %v", err), nil
	}

	lines := strings.Split(strings.TrimRight(string(output), "\n"), "\n")
	total := len(lines)
	truncated := false
	if total > maxResults {
		lines = lines[:maxResults]
		truncated = true
	}

	var b strings.Builder
	if total != 1 {
		fmt.Fprintf(&b, "Found %d matches", total)
	} else {
		fmt.Fprintf(&b, "Found 1 match")
	}
	if truncated {
		fmt.Fprintf(&b, " (showing first %d)", maxResults)
	}
	b.WriteString(":\n\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	return textResult("%s", b.String()), nil
}

func readLangflowFile(_ context.Context, cfg *config.Config, input schema.ReadLangflowFileInput) (*mcp.CallToolResult, error) {
	repoPath, err := requireRepo(cfg)
	if err != nil {
		return textError("Error: %v", err), nil
	}

	filePath := filepath.Join(repoPath, input.FilePath)
	if !isWithinRepo(repoPath, filePath) {
		return textError("Error: file path must be within the langflow repository"), nil
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return textError("Error reading file: %v", err), nil
	}

	lines := strings.Split(string(content), "\n")
	start := input.StartLine
	end := input.EndLine

	if start < 1 {
		start = 1
	}
	if end <= 0 || end > len(lines) {
		end = len(lines)
	}
	if start > len(lines) || start > end {
		return textError("Error: invalid line range (%d-%d for file with %d lines)", input.StartLine, input.EndLine, len(lines)), nil
	}

	var b strings.Builder
	for i := start - 1; i < end; i++ {
		fmt.Fprintf(&b, "%d: %s\n", i+1, lines[i])
	}

	return textResult("%s", b.String()), nil
}

func listLangflowDirectory(_ context.Context, cfg *config.Config, input schema.ListLangflowDirectoryInput) (*mcp.CallToolResult, error) {
	repoPath, err := requireRepo(cfg)
	if err != nil {
		return textError("Error: %v", err), nil
	}

	dirName := strings.TrimSpace(input.Directory)
	dirPath := filepath.Join(repoPath, dirName)
	if !isWithinRepo(repoPath, dirPath) {
		return textError("Error: directory path must be within the langflow repository"), nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return textError("Error reading directory: %v", err), nil
	}

	dirs := []string{}
	files := []string{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue // hide internals like .git unless listing the root explicitly
		}
		if entry.IsDir() {
			dirs = append(dirs, name+"/")
		} else {
			files = append(files, name)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Directory: %s\n\n", dirName)
	if len(dirs) > 0 {
		b.WriteString("Directories:\n")
		for _, d := range dirs {
			fmt.Fprintf(&b, "  %s\n", d)
		}
		b.WriteString("\n")
	}
	if len(files) > 0 {
		b.WriteString("Files:\n")
		for _, f := range files {
			fmt.Fprintf(&b, "  %s\n", f)
		}
	}
	if len(dirs) == 0 && len(files) == 0 {
		b.WriteString("(empty)")
	}

	return textResult("%s", b.String()), nil
}

func langflowConcepts(input schema.LangflowConceptsInput) *mcp.CallToolResult {
	topic := strings.TrimSpace(input.Topic)
	return textResult("%s", instructions.GetConcept(topic))
}
