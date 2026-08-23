package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ag/ai-agent-builder/internal/client"
	"github.com/ag/ai-agent-builder/internal/config"
	"github.com/ag/ai-agent-builder/internal/schema"
	"github.com/ag/ai-agent-builder/internal/templates"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// verifyBuildTimeout caps the inline build performed by create_flow_from_template
// when verify=true, so a hung build cannot block the tool call indefinitely.
const verifyBuildTimeout = 5 * time.Minute

// buildAndCollect runs the build for flowID and reports whether it completed
// cleanly, collecting any error messages from the event stream. The build
// endpoint may answer with only a {"job_id"} descriptor; in that case the
// job status endpoint is polled until a terminal status or the deadline.
func buildAndCollect(ctx context.Context, c *client.LangflowClient, flowID string) (bool, []string) {
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, verifyBuildTimeout)
		defer cancel()
	}
	eventCh, err := c.BuildFlow(ctx, flowID, client.BuildFlowRequest{})
	if err != nil {
		return false, []string{err.Error()}
	}
	errs := []string{}
	completed := false
	failed := false
	jobID := ""
	for ev := range eventCh {
		if ev.JobID != "" && jobID == "" {
			jobID = ev.JobID
		}
		collectBuildEvent(ev, &completed, &failed, &errs)
	}
	if !completed && !failed && jobID != "" {
		evCh, err := c.StreamBuildEvents(ctx, jobID)
		if err != nil {
			errs = append(errs, err.Error())
		}
		for ev := range evCh {
			collectBuildEvent(ev, &completed, &failed, &errs)
		}
	}
	if ctx.Err() != nil {
		errs = append(errs, fmt.Sprintf("build did not finish within %s", verifyBuildTimeout))
	}
	return completed && !failed, errs
}

func collectBuildEvent(ev schema.BuildEvent, completed, failed *bool, errs *[]string) {
	if ev.Error != "" {
		*errs = append(*errs, ev.Error)
	}
	switch ev.BuildStatus {
	case schema.BuildStatusComplete:
		*completed = true
	case schema.BuildStatusError:
		*failed = true
		if ev.Message != "" {
			*errs = append(*errs, ev.Message)
		}
	}
}

type ListTemplatesInput struct {
	Source   string `json:"source,omitempty"`   // "", "native", "custom" or "gallery"
	Category string `json:"category,omitempty"` // gallery only: filter by category folder
	Query    string `json:"query,omitempty"`    // free-text search over name, description and tags
}

type CreateFlowFromTemplateInput struct {
	TemplateName string            `json:"template_name"`
	NewName      string            `json:"new_name,omitempty"`
	Description  string            `json:"description,omitempty"`
	Params       map[string]string `json:"params,omitempty"`
	Verify       bool              `json:"verify,omitempty"` // build the flow and report missing credentials
}

type SaveFlowAsTemplateInput struct {
	FlowID       string   `json:"flow_id"`
	TemplateName string   `json:"template_name"`
	Description  string   `json:"description,omitempty"`
	Tags         []string `json:"tags,omitempty"`
}

func errResult(err error) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
		IsError: true,
	}, nil, nil
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, _ := json.Marshal(v)
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

// loadVerification reads templates/verification.json if present.
func loadVerification(root string) map[string]map[string]any {
	ver := map[string]map[string]any{}
	raw, err := os.ReadFile(filepath.Join(root, "verification.json"))
	if err != nil {
		return ver
	}
	var parsed map[string]struct {
		TierA *bool `json:"tier_a"`
		TierB *bool `json:"tier_b"`
	}
	if json.Unmarshal(raw, &parsed) != nil {
		return ver
	}
	for name, v := range parsed {
		entry := map[string]any{}
		if v.TierA != nil {
			entry["tier_a"] = *v.TierA
		}
		if v.TierB != nil {
			entry["tier_b"] = *v.TierB
		}
		ver[templates.Slugify(name)] = entry
	}
	return ver
}

// filterTemplateFiles applies source, category and free-text query filters.
func filterTemplateFiles(files []*templates.File, source, category, query string) []*templates.File {
	out := []*templates.File{}
	for _, f := range files {
		if source != "" && f.Dir != source {
			continue
		}
		if category != "" && !strings.EqualFold(f.Category, category) {
			continue
		}
		if !templates.MatchesQuery(f, query) {
			continue
		}
		out = append(out, f)
	}
	return out
}

func registerTemplateTools(server *mcp.Server, c *client.LangflowClient, cfg *config.Config) {
	addTool(server, &mcp.Tool{
		Name:        "list_templates",
		Description: "List the native-format template library (templates/native = official LangFlow starters, templates/custom = contributed, templates/gallery = scraped from langflow.org organized by category folders). Returns name, source dir, category (gallery), description, tags, node count and verification levels. Optional free-text query (matches all tokens against name/description/tags), plus source and category filters. Prefer instantiating from here over hand-building flows.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input ListTemplatesInput) (*mcp.CallToolResult, any, error) {
		files, err := templates.LoadDir(cfg.TemplatesDir)
		if err != nil {
			return errResult(fmt.Errorf("load templates: %w", err))
		}
		files = filterTemplateFiles(files, input.Source, input.Category, input.Query)
		ver := loadVerification(cfg.TemplatesDir)
		type entry struct {
			Name        string         `json:"name"`
			Slug        string         `json:"slug"`
			Source      string         `json:"source"`
			Category    string         `json:"category,omitempty"`
			Subcategory string         `json:"subcategory,omitempty"`
			Description string         `json:"description,omitempty"`
			Tags        []string       `json:"tags,omitempty"`
			Nodes       int            `json:"nodes"`
			Verified    map[string]any `json:"verified,omitempty"`
		}
		out := []entry{}
		for _, f := range files {
			e := entry{
				Name:        f.Name,
				Slug:        templates.Slugify(f.Name),
				Source:      f.Dir,
				Category:    f.Category,
				Subcategory: f.Subcategory,
				Description: f.Description,
				Tags:        f.Tags,
				Nodes:       f.NodeCount,
			}
			if v, ok := ver[templates.Slugify(f.Name)]; ok {
				e.Verified = v
			}
			out = append(out, e)
		}
		return jsonResult(map[string]any{"templates": out, "total": len(out)})
	})

	addTool(server, &mcp.Tool{
		Name:        "create_flow_from_template",
		Description: "Create a fully wired flow on the LangFlow instance from a library template in ONE call. Params apply generically: each key is set on every node whose template has that field (e.g. model_name, api_key, temperature); load_from_db is cleared for touched fields so literals take effect. Set verify=true to build the flow immediately and report build_ok, errors, needs_keys (blank credentials you must supply) and model_used. Prefer verify=true for gallery templates. Returns flow_id — run via POST /api/v1/run.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input CreateFlowFromTemplateInput) (*mcp.CallToolResult, any, error) {
		if input.TemplateName == "" {
			return errResult(fmt.Errorf("template_name is required"))
		}
		files, err := templates.LoadDir(cfg.TemplatesDir)
		if err != nil {
			return errResult(fmt.Errorf("load templates: %w", err))
		}
		tpl, ok := templates.Lookup(files, input.TemplateName)
		if !ok {
			return errResult(fmt.Errorf("template %q not found; use list_templates", input.TemplateName))
		}
		dataRaw, err := tpl.DataRaw()
		if err != nil {
			return errResult(fmt.Errorf("extract data: %w", err))
		}
		dataRaw, err = templates.ApplyParams(dataRaw, input.Params)
		if err != nil {
			return errResult(fmt.Errorf("apply params: %w", err))
		}
		name := input.NewName
		if name == "" {
			name = tpl.Name + " (from template)"
		}
		desc := input.Description
		if desc == "" {
			desc = tpl.Description
		}
		flow, err := c.CreateFlowFromRaw(ctx, name, desc, dataRaw)
		if err != nil {
			return errResult(fmt.Errorf("instantiate template: %w", err))
		}
		result := map[string]any{
			"flow_id": flow.ID,
			"name":    flow.Name,
			"source":  tpl.Dir + "/" + filepath.Base(tpl.Path),
		}
		hint := "verify with build_flow, then run via POST /api/v1/run/{flow_id}"
		if input.Verify {
			buildOK, buildErrs := buildAndCollect(ctx, c, flow.ID)
			needsKeys := templates.DetectBlankSecrets(dataRaw)
			if needsKeys == nil {
				needsKeys = []templates.SecretNeed{}
			}
			result["build_ok"] = buildOK
			result["errors"] = buildErrs
			result["needs_keys"] = needsKeys
			modelUsed, modelProvider := templates.DetectModel(dataRaw)
			if modelUsed != "" {
				result["model_used"] = modelUsed
				result["model_provider"] = modelProvider
			}
			switch {
			case !buildOK:
				hint = "build failed — inspect errors before running"
			case len(needsKeys) > 0:
				hint = "built OK; credential fields in needs_keys are blank — they may resolve via instance global variables; supply your own keys via params/update_node if needed"
			default:
				hint = "verified: built clean, no missing credentials; run via POST /api/v1/run/{flow_id}"
			}
		}
		result["hint"] = hint
		return jsonResult(result)
	})

	addTool(server, &mcp.Tool{
		Name:        "save_flow_as_template",
		Description: "Export a verified flow into the template library (templates/custom/) in LangFlow's native format. Sanitizes secret fields automatically and reports warnings. Gate on success: only save flows that built AND ran (HTTP 200). After saving, re-instantiate once to confirm the template is self-sufficient.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input SaveFlowAsTemplateInput) (*mcp.CallToolResult, any, error) {
		if input.FlowID == "" || input.TemplateName == "" {
			return errResult(fmt.Errorf("flow_id and template_name are required"))
		}
		dataRaw, err := c.GetFlowRaw(ctx, input.FlowID)
		if err != nil {
			return errResult(err)
		}
		sanitized, warnings := templates.SanitizeForTemplate(dataRaw)
		env, err := templates.BuildEnvelope(sanitized, input.TemplateName, input.Description, input.Tags)
		if err != nil {
			return errResult(err)
		}
		dir := filepath.Join(cfg.TemplatesDir, "custom")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return errResult(fmt.Errorf("create custom dir: %w", err))
		}
		path := filepath.Join(dir, templates.Slugify(input.TemplateName)+".json")
		if err := os.WriteFile(path, env, 0o644); err != nil {
			return errResult(fmt.Errorf("write template: %w", err))
		}
		return jsonResult(map[string]any{
			"path":     path,
			"warnings": warnings,
			"hint":     "re-instantiate via create_flow_from_template to confirm the template works standalone",
		})
	})
}
