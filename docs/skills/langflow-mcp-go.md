# LangFlow MCP Go — Agent Skill

This directory documents the `langflow-mcp-go` skill used by E-agents to drive the
`langflow-mcp` server effectively.

## Skill Location

The skill lives in the agent's skill directory (not in this repo, as skills are
personal agent config):

```
~/.config/opencode/skills/langflow-mcp-go/
├── SKILL.md            # Core rules, failure modes, canonical workflow
├── tools-reference.md  # All 37 tools with exact signatures
└── workflows.md        # E-agent recipes (W1–W7)
```

## Why a Skill?

Driving the MCP server blind leads to token-wasting mistakes:

1. **tool_mode confusion** — assuming a built-in Calculator "already exposes a Tool
   output" when in fact `set_tool_mode` is required for ANY component to be callable
   by an Agent.
2. **Wrong component names** — guessing `"OpenAIModel"` when it's `"OpenAI"`.
3. **Forgetting to build** — adding/connecting nodes is config-only; you must
   `build_flow` to verify.
4. **Type validation failures** — connecting incompatible edges without
   `validate_connection` first.

The skill was built using TDD-for-documentation: a baseline agent (no skill) made
all 4 mistakes above; the skill was written to counter them; a second agent (with
skill) avoided all of them.

## Installing the Skill

Copy the `langflow-mcp-go/` directory to your agent's skills folder:

- **OpenCode:** `~/.config/opencode/skills/`
- **Claude Code:** `~/.claude/skills/`
- **Codex:** `~/.agents/skills/`

No restart needed — skills are discovered on next session start.

## Tool Categories (37 total)

| Category | Count | Tools |
|----------|-------|-------|
| Component Discovery | 4 | list_component_categories, list_components, get_component_schema, search_components |
| Flow Management | 6 | list_flows, list_all_flows, get_flow, create_flow, delete_flow, duplicate_flow |
| Build & Execution | 3 | build_flow, build_node, get_build_status |
| Node Manipulation | 14 | add_node, add_custom_component, update_node, set_tool_mode, remove_node, get_node_details, list_nodes, move_node, move_nodes_batch, auto_arrange_flow, analyze_flow_layout, get_layout_suggestions, add_note, update_note |
| Connection Management | 5 | connect_nodes, disconnect_nodes, list_connections, validate_connection, find_compatible_connections |
| Source Exploration | 5 | setup_langflow_source, explore_langflow, read_langflow_file, list_langflow_directory, langflow_concepts |

See `tools-reference.md` (in the skill dir) for exact JSON signatures.
