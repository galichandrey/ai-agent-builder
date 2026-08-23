# Template Library

File-based template library in **LangFlow's official native format** (same shape as
`langflow/initial_setup/starter_projects/*.json` and the flows API). This is not a
parallel format — it is the format LangFlow itself defines for templates
(see docs.langflow.org/contributing-templates).

## Layout

- `native/` — the 29 official LangFlow starter templates, extracted verbatim from
  the running container (`/app/.venv/lib/python3.14/site-packages/langflow/
  initial_setup/starter_projects/`, LangFlow 1.11). Filenames are snake_case slugs;
  the display name lives inside each file.
- `custom/` — templates contributed through the self-learning loop: after an agent
  builds and verifies a flow that is NOT in the library, `save_flow_as_template`
  exports it here in the same native format (secrets sanitized).
- `gallery/<category>/<subcategory>/<slug>.json` — **100 templates scraped from
  langflow.org/use-cases** (business 49, processing 14, automation 11, analytics 11,
  productivity 10, data 3, documents 2). Collected with `scripts/scrape_gallery.py`
  (sitemap → flow JSON → secret sanitization); bulk-imported into a dashboard via
  `scripts/sync_gallery.py`. Secrets are blanked at scrape time — expect non-empty
  `needs_keys` when instantiating with `verify: true`; fields may resolve via
  instance global variables ("OpenAI Compatible").

## Instantiation

One MCP call: `create_flow_from_template(template_name, params={...}, verify=?)`.
Params are applied generically — any template field name (`model_name`, `api_key`,
`temperature`, ...) is set on every node that has it; `load_from_db` is cleared for
touched fields so literals take effect.

Discover templates first: `list_templates(source="gallery", query="caption social")`
(every whitespace token must match name/description/tags) or by `category`.

With `verify: true` the tool builds the created flow inline (5-min cap) and returns
`build_ok`, `errors[]`, `needs_keys[]` (blank credential fields — advisory when
`build_ok: true`), and `model_used`/`model_provider`.

## Verification levels

`verification.json` records, per template:
- `tier_a` — instantiates and builds on the live instance without errors;
- `tier_b` — runs end-to-end (HTTP 200) with only an LLM credential configured
  (opencode zen free model).

Templates needing extra credentials/files (vector stores, API-keyed tools) may be
tier_a-only; instantiate them and configure credentials via `update_node`.

## Notes

- The package gallery cannot be extended via the API (it regenerates from package
  files at startup). To push a template into a LangFlow instance as regular flows,
  use the native endpoints `POST /api/v1/flows/batch/` or `POST /api/v1/flows/upload/`
  (upsert semantics) with these same files.
- Never commit secrets into this library; `save_flow_as_template` blanks password
  fields automatically and reports warnings.
