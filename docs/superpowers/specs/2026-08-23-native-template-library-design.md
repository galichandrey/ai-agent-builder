# Native Template Library — Design

Date: 2026-08-23
Status: Approved (user, 2026-08-23)

## Problem

E-agents building LangFlow flows via the MCP server had only one demo flow and
hand-rolled recipes. LangFlow ships an official template library (29 JSON files in
the package at `langflow/initial_setup/starter_projects/`, plus the
`GET /api/v1/starter-projects/` endpoint), and the official template format is
documented as the standard for contributions
(docs.langflow.org/contributing-templates). Our code and skill must align with this
native format instead of inventing a parallel one.

## Research findings

- Native template top-level keys: `data, description, endpoint_name, id,
  is_component, last_tested_version, locked, name, tags`.
- Edges carry full reactflow fields: top-level `sourceHandle/targetHandle`,
  `animated`, `className`, `selected`, `id` + rich `data.sourceHandle/targetHandle`.
  Our Go `Edge` marshals only `{source,target,data}` → fidelity loss.
- Nodes: full reactflow set (`dragging, measured, positionAbsolute, resizing…`).
  Our `Node` already raw-captures — safe.
- No hardcoded `api_key` in any of the 29 templates; `load_from_db` used widely.
- 39 note nodes across templates — canvas documentation is part of template UX.
- The package gallery cannot be extended via API (regenerated from package files
  at startup, setup.py `create_or_update_starter_projects`). Flows can be pushed
  into an instance via native `POST /flows/`, `/flows/batch/` (preserves IDs),
  `/flows/upload/` (upsert).
- Template data = exactly `{nodes, edges, viewport}` — matches our FlowData shape.

## Design decisions

1. **Raw-first**: template files are treated as opaque JSON; only catalog fields
   (name/description/tags) are typed. Instantiation posts raw bytes with a minimal
   name patch — field loss impossible by construction. Same principle as Node.raw.
2. **Edge raw-capture** (parity with Node): GET→PATCH cycles preserve every field.
3. **File-based library** in two dirs:
   - `templates/native/` — the 29 official templates extracted once from the
     container (seed; source of truth for format).
   - `templates/custom/` — user-contributed templates produced by the
     self-learning loop.
4. **Parametrization**: generic applier — set `template.<field>.value` on every node
   whose template contains `<field>` (e.g. `model_name`, `api_key`), reusing the
   proven `TemplateField` scalarOnly semantics incl. auto-clearing `load_from_db`
   when a literal is set. Optional per-template param declarations may narrow or
   default values.
5. **Sanitization on save**: `save_flow_as_template` refuses/warns on non-empty
   password-type fields so secrets never enter the library.
6. **MCP tools (3 new, total 40)**:
   - `list_templates(source?)` → catalog with name, description, tags, node count,
     verification level.
   - `create_flow_from_template(template_name, new_name?, params?)` → one call,
     fully wired flow on the instance; returns flow_id.
   - `save_flow_as_template(flow_id, template_name, description?, tags?,
     params_hint?)` → export current flow to native-format file in custom/.
7. **Config**: `LANGFLOW_MCP_TEMPLATES_DIR` (default `./templates`).

## Verification

- Tier A (all 29): instantiate via MCP tool + build OK on live instance.
- Tier B (LLM-only templates): live run HTTP 200 using opencode zen free model;
  level recorded in catalog/docs.
- Round-trip: save_flow_as_template → create_flow_from_template → structurally
  identical to source flow (modulo params).
- Skill application test: fresh agent builds a flow from the library following
  only the updated skill.

## Skill v2.2.0

- SKILL.md rule R7: templates are NATIVE LangFlow files — never hand-assemble JSON;
  prefer library instantiation; contribute back after verified success.
- workflows.md W8 (instantiate from library) and W9 (contribute template: success
  criteria, sanitization, re-instantiation check); native endpoints table
  (`/starter-projects/`, `/flows/basic_examples/`, `/flows/upload/`,
  `/flows/batch/`).
- Sync repo skill == installed OpenCode skill == docs/skills + README.

## Out of scope

- Extending the package gallery itself (impossible via API by design).
- Store/marketplace integration (external service).
- Per-template bespoke param schemas beyond generic field targeting.
