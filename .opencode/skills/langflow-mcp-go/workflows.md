# LangFlow MCP Go — E-Agent Workflows

Recipes verified end-to-end on LangFlow 1.11 (Docker, `AUTO_LOGIN=false`).

## W1: Agent + Model Flow (VERIFIED with free model)

ChatInput → Agent(model=OpenAIModel) → ChatOutput, model on opencode zen free tier:

```
create_flow(name="agent-flow")
  → add_node(flow_id, "ChatInput", 100, 300)                          # chat_in
  → add_node(flow_id, "Agent", 1000, 300,
        config={system_message: "Ты полезный ассистент."})             # agent
  → add_node(flow_id, "ext:openai:OpenAIModelComponent@official",
        600, -300)                                                     # model
  → add_node(flow_id, "ChatOutput", 1900, 300)                         # chat_out

update_node(model, config={
    model_name: "nemotron-3-ultra-free",
    openai_api_base: "https://opencode.ai/zen/v1",
    api_key: "<zen-key>",            # update_node auto-disables load_from_db (R4)
    temperature: 0.3})

connect_nodes(model.model_output → agent.model)
connect_nodes(chat_in.message   → agent.input_value)
connect_nodes(agent.response    → chat_out.input_value)

build_flow(flow_id, input_value="Привет!")     # VERIFY (R3)
```

NOTE: free models flap — one may return `403 {'model': ...}` while another works.
On model-level failures just `update_node(model_name=<next free model>)` and re-run
(verified: `hy3-free` stable with tool calls; `nemotron-3-ultra-free` flaky).

Free-model providers that work this way:
- **opencode zen**: base `https://opencode.ai/zen/v1`, models `nemotron-3-ultra-free`, `hy3-free`, `deepseek-v4-flash-free`, … (8 free)
- **kilo.ai**: base `https://api.kilo.ai/api/gateway/v1`, models `nvidia/nemotron-3.5-lightning:free`, `tencent/hy3:free`, … (17 free)
- Any OpenAI-compatible endpoint: set `openai_api_base` + `api_key` + `model_name`.

## W2: Give an Agent a Tool

```
# Native tool components already expose a Tool output:
add_node(flow_id, "CalculatorTool", x, y)                 # calc
connect_nodes(calc.api_build_tool → agent.tools)          # NO set_tool_mode needed

# Custom components need the server transform:
add_custom_component(flow_id, code="class MyTool(Component): ...")
set_tool_mode(flow_id, custom_node_id, enabled=true)      # → component_as_tool
connect_nodes(custom_node.component_as_tool → agent.tools)
```

## W3: Swap Provider/Model on Existing Node

```
get_node_details(flow_id, model_node_id)      # confirm field names
update_node(flow_id, model_node_id, config={
    model_name: "hy3-free",
    openai_api_base: "https://opencode.ai/zen/v1",
    api_key: "<key>"})
build_flow(flow_id, input_value="test")       # VERIFY
```

## W4: Fix Rejected Connection

```
validate_connection(source_component_type, source_output, target_component_type, target_input)
get_component_schema(target_type)              # check field names + show flag
connect_nodes(corrected pair)
find_compatible_connections(flow_id, node_id, direction="inputs|outputs")  # discovery
```

## W5: Layout Cleanup

```
analyze_flow_layout(flow_id) → get_layout_suggestions(flow_id)
auto_arrange_flow(flow_id, direction="horizontal", spacing=800)
move_nodes_batch(flow_id, moves=[{node_id,x,y}, ...])
```

## W6: Explore LangFlow Internals

```
setup_langflow_source()                        # once
explore_langflow(query="custom_component_update")
read_langflow_file(path, start_line, end_line)
langflow_concepts(topic="tool_mode"|"building"|"common_mistakes")
```

## W7: Duplicate as Template

```
duplicate_flow(flow_id, new_name="Variant") → get_flow(new_id) → update_node(...) → build_flow(...)
```

## Running via REST instead of build_flow

```
POST /api/v1/run/{flow_id}?stream=false
headers: x-api-key: <LF key>
body: {"output_type":"chat","input_type":"chat","input_value":"..."}
```
Useful for E-agents driving the flow without the MCP build stream. Same auth rules (R0).
