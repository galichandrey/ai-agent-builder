# LangFlow MCP Go — Tool Reference

All 37 tools. Exact names + key args. Categories: Component Discovery (4), Flow Management (6), Build & Execution (3), Node Manipulation (14), Connection Management (5), Source Exploration (5).

## Component Discovery (4)

| Tool | Args | Purpose |
|---|---|---|
| `list_component_categories` | `{}` | All categories |
| `list_components` | `{"category": "agents"}` | Components in a category |
| `get_component_schema` | `{"component_type": "Agent"}` | Full inputs/outputs/types |
| `search_components` | `{"query": "openai"}` | Search by name/desc |

## Flow Management (6)

| Tool | Args | Purpose |
|---|---|---|
| `list_flows` | `{"page": 1, "size": 50, "folder_id": "opt"}` | List (excludes backups) |
| `list_all_flows` | `{}` | All incl. backups |
| `get_flow` | `{"flow_id": "uuid"}` | Full flow + nodes/edges |
| `create_flow` | `{"name": "X", "description": "opt"}` | New empty flow |
| `delete_flow` | `{"flow_id": "uuid"}` | Permanent delete |
| `duplicate_flow` | `{"flow_id": "uuid", "new_name": "opt"}` | Clone (copies nodes/edges) |

## Build & Execution (3)

| Tool | Args | Purpose |
|---|---|---|
| `build_flow` | `{"flow_id", "input_value", "input_type": "chat", "wait_for_completion": true, "timeout_seconds": 120}` | Run flow (NDJSON stream) |
| `build_node` | `{"flow_id", "node_id"}` | Build single vertex |
| `get_build_status` | `{"job_id"}` | Poll async job |

## Node Manipulation (14)

| Tool | Args | Purpose |
|---|---|---|
| `add_node` | `{"flow_id", "component_type", "position_x", "position_y", "config": {}}` | Built-in node. `component_type` = TYPE name from search_components (e.g. `ext:openai:OpenAIModelComponent@official`) |
| `add_custom_component` | `{"flow_id", "code": "py", "position_x", "position_y", "tool_mode": false}` | Inline Python (no restart) |
| `update_node` | `{"flow_id", "node_id", "config": {}}` | Set literal template values; auto-clears `load_from_db` on touched fields (api_key etc.) |
| `set_tool_mode` | `{"flow_id", "node_id", "enabled": true}` | Custom components: server-side transform → `component_as_tool`. Built-ins: flip tool_mode flag only (native tools already expose `api_build_tool`) |
| `remove_node` | `{"flow_id", "node_id"}` | Remove + its edges |
| `get_node_details` | `{"flow_id", "node_id"}` | Full node info |
| `list_nodes` | `{"flow_id"}` | All nodes |
| `move_node` | `{"flow_id", "node_id", "x", "y"}` | Reposition one |
| `move_nodes_batch` | `{"flow_id", "moves": [{"node_id", "x", "y"}]}` | Reposition many |
| `auto_arrange_flow` | `{"flow_id", "direction": "horizontal", "spacing": 300}` | Topological layout |
| `analyze_flow_layout` | `{"flow_id"}` | Structure, collisions, depths |
| `get_layout_suggestions` | `{"flow_id"}` | Recommendations |
| `add_note` | `{"flow_id", "content", "x", "y", "width", "height", "background_color"}` | Sticky note |
| `update_note` | `{"flow_id", "note_id", "content", "background_color"}` | Edit note |

## Connection Management (5)

| Tool | Args | Purpose |
|---|---|---|
| `connect_nodes` | `{"flow_id", "source_node_id", "source_output", "target_node_id", "target_input"}` | Edge w/ validation |
| `disconnect_nodes` | `{"flow_id", "source_node_id", "target_node_id", "target_input": "opt"}` | Remove edges |
| `list_connections` | `{"flow_id", "node_id": "opt"}` | List (filter by node) |
| `validate_connection` | `{"source_component_type", "source_output", "target_component_type", "target_input"}` | Type check |
| `find_compatible_connections` | `{"flow_id", "node_id", "direction": "inputs|outputs"}` | Discover valid |

## Source Exploration (5)

| Tool | Args | Purpose |
|---|---|---|
| `setup_langflow_source` | `{"force_update": false}` | Clone LangFlow repo |
| `explore_langflow` | `{"query": "class Component", "path_filter": "opt", "max_results": 20}` | Grep source |
| `read_langflow_file` | `{"file_path": "src/...", "start_line": 1, "end_line": 0}` | Read file |
| `list_langflow_directory` | `{"directory": "src/backend"}` | Browse |
| `langflow_concepts` | `{"topic": "tool_mode"}` | Concept docs |

## Key Types

- **Node ID**: generated UUID-style string (e.g. `Agent-Ab12Cd`)
- **Edge ID**: `reactflow__edge-{src}{src_handle}-{tgt}{tgt_handle}`
- **Output handle**: `{node_id}|{output_name}` (Langflow convention)
- **Input handle**: `{node_id}|{input_name}`
