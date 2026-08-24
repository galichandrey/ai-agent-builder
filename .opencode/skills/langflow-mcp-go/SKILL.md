---
name: langflow-mcp-go
description: Use when driving the Go-based langflow-mcp server to manage LangFlow — instantiate flows from the native template library in one call, add nodes (built-in or inline Python), wire Agent + model flows, run them via build_flow or REST /run, toggle tool_mode for agent tools, contribute verified flows back as templates, arrange layouts, or explore LangFlow source. Trigger when an E-agent needs programmatic LangFlow control, configures free-model providers (opencode zen etc.), or hits errors like "unhashable type", "component not found", "Missing credentials", "No model selected", 403/404 Invalid API key, an EMPTY flow canvas ("Untitled Flow") in UI, Prompt build KeyError on template variables, "Edge ... has no matched type", or when exposing flows as MCP tools / connecting LangFlow's project MCP server.
metadata:
  version: 2.7.0
---

# LangFlow MCP Go Server

You manage LangFlow through the **Go `langflow-mcp` server** (42 tools, incl. `assistant_chat`). Every rule below is backed by a live-verified failure.

## Core Rules

**R0. Auth = `x-api-key`, never Bearer JWT.**
- Set `LANGFLOW_MCP_API_KEY=sk-...`. Do NOT pass `Authorization: Bearer <JWT>` via `LANGFLOW_MCP_CUSTOM_HEADERS` — GET `/flows/{id}` then returns 404 while POST works (auth-context mismatch).
- API keys are created ONLY in the Web UI (Settings → Langflow API Keys) or CLI when `AUTO_LOGIN=true`. There is no REST endpoint. After a container restart keys survive; a stale key gives `403 {"detail":"Invalid API key"}` → recreate in UI.
- Keys adopt their creator's rights; superuser key sees everything.

**R1. Component names are TYPE names, not display names.**
Call `search_components` first and use the returned `name`: `ChatInput` (display "Chat Input"), model is `ext:openai:OpenAIModelComponent@official`, `Agent`. Extension types carry `ext:<provider>:<Class>@official`.

**R2. Prefer the template library over hand-building.**
- `list_templates()` first — the library ships LangFlow's **29 official native templates** (`templates/native/`), contributed ones (`templates/custom/`), and **100 gallery templates scraped from langflow.org** (`templates/gallery/`, organized by category: business 49, processing 14, automation 11, analytics 11, productivity 10, data 3, documents 2). Verification levels: `tier_a` = builds clean on live instance (29/29 native), `tier_b` = full run HTTP 200 with only an LLM credential (15/29, see `templates/verification.json`).
- Search before instantiating: `list_templates(source="gallery", query="caption social")` — every whitespace token must appear in name/description/tags (order-independent). Narrow with `category="business"`.
- `create_flow_from_template(template_name, new_name?, params?, verify?)` = ONE call → fully wired flow.
- **Always pass `verify: true` for gallery templates.** The result then includes `build_ok`, `errors[]`, `needs_keys[]` (blank credentials: `{node_id, node_type, field}`), and `model_used`/`model_provider`. Interpretation: `build_ok:false` → inspect `errors[]`, fix, re-verify. `build_ok:true` + non-empty `needs_keys` → advisory only; blank fields likely resolve via instance global variables — supply your own keys via `params`/`update_node` only if the run fails or you need a different credential. Gallery templates ship secrets blanked by design.
- `params` are generic: each key is set on every node whose template has that field (`model_name`, `api_key`, `temperature`, `system_prompt`, ...). `load_from_db` auto-clears on touched fields.
- Bulk (re)import of the whole gallery into the dashboard: `scripts/sync_gallery.py --api-key <LF key>` (idempotent; skips already-imported names). Re-collect from langflow.org with `scripts/scrape_gallery.py`.

**R3. Model selection has TWO shapes — parametrize both blindly.**
Passing `model_name` handles both automatically:
- Modern nodes (Agent, LanguageModelComponent) use a `model` **ModelInput** field → value becomes `[{"name": "<model>", "provider": "OpenAI Compatible"}]`.
- Legacy model nodes have `provider`+`model_name` fields → both set (empty provider filled with "OpenAI Compatible").
"OpenAI Compatible" provider resolves base URL/key from instance global variables `OPENAI_COMPATIBLE_BASE_URL` / `OPENAI_COMPATIBLE_API_KEY` (create once via `POST /api/v1/variables/`, type Generic/Credential). If you set `model_name` to a provider-specific name without this plumbing you get `No model selected` / `Model name/provider overrides require a built-in model selection`.

**R4. Tool wiring depends on component kind.**
- Native tool components (`CalculatorTool`, …): they ALREADY expose a Tool output — connect its `api_build_tool` output straight to `Agent.tools`. `set_tool_mode` not needed.
- Custom components (carry `code`): `set_tool_mode(enabled=true)` runs LangFlow's server-side `/custom_component/update` transform → adds `component_as_tool` (method `to_toolkit`). Connect THAT to `Agent.tools`.
- Other built-ins: `set_tool_mode` flips the `tool_mode` flag only.

**R5. Config changes don't execute.** After any mutation verify with `build_flow(flow_id, input_value=...)` (or POST `/api/v1/run/{id}`). A run error of kind "Error building Component X: <credentials/file>" still proves the graph structure is sound — configure that component and re-run.

**R6. Literal values win.** `update_node` and template params auto-flip `load_from_db=true → false` on touched fields (e.g. `api_key`) — otherwise LangFlow treats the value as a global-variable name and fails at runtime with "Missing credentials".

**R7. Templates are NATIVE LangFlow files — never hand-assemble JSON.**
The library format is exactly what LangFlow itself ships (`initial_setup/starter_projects/*.json`) and documents for contributions. Never write flow JSON by hand: instantiate from the library, adjust with tools, then give back:
- After building AND running (HTTP 200) a flow not in the library, OFFER to save it: `save_flow_as_template(flow_id, template_name, description?, tags?)` → lands in `templates/custom/`, secrets sanitized automatically.
- Then re-instantiate once from the saved template to prove it is self-sufficient.
This self-learning loop grows the library and shrinks future error surface.

## Agent Wiring Cookbook (spike-verified 2026-08-23, LangFlow 1.11)

**AW-детерминизм. LLM не принимает порядковые решения в конвейере.** Порядок стадий, условия переходов, «что дальше» — считает код (отдельный state-инструмент типа `next_stage` поверх манифеста). Промпт агента описывает ТОЛЬКО механику: «вызови X → передай Y → заверши строкой Z». Симптом для переноса логики в код: агент повторяет/пропускает шаги недетерминированно.

**AW0. СНАЧАЛА живой пример, потом свой конфиг.** Перед настройкой любой ноды: `GET /api/v1/flows/` (gzip!), grep по JSON на паттерн ("OpenAI Compatible", "tool_mode", имя компонента) и копируйте рабочую форму. 90% ошибок конфигурации = выдуманный формат вместо скопированного. Массовое создание однотипных флоу — скриптом-клоном проверенного скелета (пример: scripts/build_stubs.py в content-factory).

**AW0b. Сеть контейнера может быть pasta, а не docker0.** Проверяйте `docker inspect langflow --format '{{.HostConfig.NetworkMode}}'`: при `pasta` хост из контейнера = `http://host.containers.internal:<port>`; 172.17.0.1 не работает.

**AW-graph. Хэндлы эджей: строка `œ`-формата сверху + словарь в `edge.data`.** При сборке флоу через REST каждый эдж несёт `sourceHandle`/`targetHandle` ДВАЖДЫ: словарём в `edge.data.*` (его читает бэкенд) и СТРОКОЙ в `edge.sourceHandle/targetHandle` (её читает UI). Строка — это reactflow-сериализация, где символ `œ` (U+0153) играет роль кавычки: `{œdataTypeœ:œChatInputœ,œidœ:Xœ,...}`, список `[œMessageœ]`, запятые/двоеточия голые. Фронт делает `string.replaceAll(œ,'"')` → `JSON.parse`; неверный формат = SyntaxError в консоли и **ПУСТОЙ канвас** («Untitled Flow», нод нет вовсе) при полностью рабочем REST `/run`. Энкодер: см. `scripts/build_real_stages.py::oe_handle` в content-factory. Лечение существующих флоу без пересоздания: перегенерировать строки из `edge.data.*` (scripts/fix_handle_strings.py). Диагностика UI: Playwright → логин суперпользователя (`AUTO_LOGIN=false`) → консоль страницы; позиция ошибки в JSON.parse точно указывает на битый токен.

**AW-prompt. Переменные PromptTemplate НЕ биндятся через эджи у API-собранных флоу.** Даже точный клон рабочего паттерна (ChatInput→Prompt c `{var}` и targetHandle fieldName=var) падает при билде `Error building Component Prompt Template: '<var>'` — динамические входы промпта материализуются только из UI-редактирования. Решение: инструкции класть в **`Agent.system_prompt`** (обычная строка, не шаблон), данные — напрямую в `input_value` агента; цепочка ChatInput→Agent→ChatOutput.

**AW-custom-src. CustomComponent как ИСТОЧНИК эджа валиден только в сторону ChatOutput.** Ребро CustomComponent→Agent/Prompt даёт валидацию «Edge between X and Y has no matched type» даже при корректных output_types ["Message"] (CustomComponent к ChatOutput проходит). Нативные ноды (ChatInput→Prompt→Agent) — свободно. Если нужен кодовый узел в середине цепочки — выносить логику на сторону сервиса (шим/REST) либо завершать им цепочку в ChatOutput.

**AW-donor. Донор — только РАБОЧИЙ флоу инстанса.** Формы нод для клонирования брать из флоу, который реально исполняется на этой версии (например, Deep Research Agent из starter_project), а не из библиотечных шаблонов «как есть» — они могут быть устаревших версий (подсказка инстанса: «The flow contains N outdated components»). Проверяй состав донора GET /flows/{id} перед клоном.

**AW1. Конфиг модели у `Agent` — только эта форма работает:**
`model = [{"name":"hy3-free","provider":"OpenAI Compatible"}]` (список словарей!), `api_key` = литеральный ключ с `load_from_db=false`. Строки `"Provider/model"` и `{VAR}`-плейсхолдеры НЕ резолвятся (уходят буквально → OpenAI 401). base_url провайдер берёт из global vars `OPENAI_COMPATIBLE_*` по соглашению имён.

**AW2. tool_mode у CustomComponent:** после `set_tool_mode(true)` именованные выходы исчезают, остаётся один хэндл **`component_as_tool`** (types Tool) — именно его в `connect_nodes(..., target_input="tools")`. Тул `connect_nodes` v2.4.0 сам подставляет единственный выход, если запрошенный не найден.

**AW3. Оркестрация «агент → под-агент» = FlowProxy-паттерн.** Штатный `FlowTool` сломан в рантайме (`Graph is not defined`, legacy), `RunFlow` (beta) как тул через API не регистрируется без UI-рефреша. Рабочий путь: кастомный компонент делает POST `http://127.0.0.1:7860/api/v1/run/{sub_flow_id}` (opener `ProxyHandler({})`, timeout ≥240s) и вытаскивает последний `message`. Константы (flow_id, api_key) — ЗАХАРДКОЖЕНЫ в коде компонента: при вызове как тула инстанс создаётся заново и значения полей шаблона теряются.

**AW4. Кэш модулей кастомных компонентов:** модуль = `custom_components/<snake(display_name)>`. Правка кода существующего класса может не примениться в tool-контексте — при значимых правках меняйте имя класса (FlowProxy → FlowProxyV2).

**AW5. Флоу для REST `/run` обязан иметь ChatInput И ChatOutput** — иначе `'NoneType' object is not iterable` или пустые outputs. Флоу-тулы без чат-нод напрямую не ранятся (норма; проверять их через агента или build_flow).

**AW6. Верификация флоу без curl-циклов:** новый тул **`run_flow(flow_id, input_value?, tweaks?, session_id?, timeout_sec?)`** — синхронный POST `/run/{id}` (без трейлинг-слэша! 405 со слэшем), возвращает `{last_message, raw}`; last_message — последний непустой `message|text` из дерева ответа.

**AW7. REST-мелочи:** `/api/v1/variables/` и `/api/v1/flows/` — С трейлинг-слэшем; `/api/v1/run/{id}` — БЕЗ; списки отдаются gzip (`curl --compressed`); server-side фильтра `?name=` нет — фильтруйте клиентски; рукописные ноды в flow JSON требуют `template._type="Component"`.

## Verified Port Map (Agent flow)

| From | output | To | input |
|---|---|---|---|
| ChatInput | `message` | Agent | `input_value` |
| OpenAIModelComponent | `model_output` | Agent | `model` |
| Agent | `response` | ChatOutput | `input_value` |
| any tool | `api_build_tool` / `component_as_tool` | Agent | `tools` |

Full recipes: [workflows.md](workflows.md) · All 40 tools: [tools-reference.md](tools-reference.md)

## Native LangFlow endpoints worth knowing

| Endpoint | Purpose |
|---|---|
| `GET /api/v1/starter-projects/` | lfx starter dumps (5, minimal; names come back null) |
| `GET /api/v1/models?type=language` | providers + models incl. "OpenAI Compatible" variables |
| `GET /api/v1/variables/` · `POST /api/v1/variables/` | global variables (provider credentials) |
| `POST /api/v1/flows/batch/` | push multiple flows WITH native IDs (upsert-ish guard) |
| `POST /api/v1/flows/upload/` | import template JSON/ZIP file(s), upsert semantics |
| `GET /api/v1/flows/basic_examples/` | tiny example flows |

The package gallery ("New Flow" modal) regenerates from package files at startup — it CANNOT be extended via API. Our file library + these endpoints cover every practical need.

## Flows-as-MCP-Tools (родной механизм «флоу-как-тул», 1.11.4)

**Сервер (project MCP):**
- Тул = флоу с полями `mcp_enabled=true`, `action_name` (snake_case имя тула), `action_description` — всё управляется `PATCH /api/v1/flows/{id}` БЕЗ UI.
- Список тулов проекта: `GET /api/v1/mcp/project/{folder_id}?mcp_enabled=true`.
- Транспорты: SSE `/api/v1/mcp/project/{id}/sse` и Streamable HTTP `.../streamable` (auth `x-api-key`).
- Контекст вызова тула = отдельный run флоу: аргументы `input_value` (+опц. `session_id` для сквозной трассировки), остальное уходит в tweaks. HITL-флоу тулами НЕ поддерживаются (RuntimeError).
- UI: страница `/mcp/folder/{id}` → таб MCP Server → Edit Tools (галочки = те же поля).

**Клиент внутри флоу (компонент `MCPTools`):**
- Выход `component_as_tool` (Toolset, types Tool, cache=False) → `Agent.tools`: агент получает ВСЕ включённые флоу проекта разом; новый стадийный флоу появляется у агента без пересборки.
- Поле `mcp_server`: значение `{"name": "<имя>", "config": {"url": ".../streamable", "headers": {"x-api-key": "..."}}}` — конфиг с ключом `url` = Streamable HTTP (без subprocess!). DB-конфиг из Settings → MCP Client приоритетнее value.
- `use_cache=false` по умолчанию; TTL списка тулов 30с; `tool_execution_timeout` задавать ≥ времени стадийного рана.
- Settings → MCP Client (`lfx-mcp` через uvx/stdio) — для внешних редакторов кода; внутри контейнера НЕ нужен (uvx отсутствует).

**Стратегия (проверено боем):** родной project-MCP — основной способ агент↔агент вызовов; кастомные FlowProxy-компоненты с REST self-call — fallback. Детерминизм гейтов держать на стороне state-сервиса (шим отклоняет запись стадии без апрува), а не в промпте/нодах.

## Langflow Assistant (agentic, tool `assistant_chat`)

**Статус на PG — ПОЛНОСТЬЮ РАБОЧИЙ (verified 2026-08-24):** чат-токены, `set_flow` с готовым канвасом, применение через `apply_to_flow_id`.
**На SQLite — блокер апстрима:** `database is locked` в фазе generating_flow (request-scoped сессия держит write-лок; busy_timeout/wal_checkpoint не помогают). Лечение — документированный env `LANGFLOW_DATABASE_URL` → PostgreSQL.

**Контракт:** `POST /agentic/assist/stream` (SSE): `{flow_id, input_value, session_id?, provider?, model_name?, iterations_limit?}`.
События: progress → token → **flow_update(action=set_flow, flow=<полный UI-формат канваса>)** → complete. Применение as-is: POST/PATCH /flows с этим data — сервер принимает genericNode-формат без конверсии.

**Тул `assistant_chat`:** `apply_to_flow_id` = существующий id (PATCH) или `"new:<имя>"` (создать). Возвращает text/progress/update_actions/preview/applied_to.

**Требования окружения (все — официальные настройки):**
1. `LANGFLOW_DATABASE_URL` → PostgreSQL (SQLite=блокер ассистента).
2. Канонические переменные: `OPENAI_API_KEY` (Credential) + опц. `OPENAI_BASE_URL`. Валидация по имени неизбежна: invoke каталоговой gpt-* модели. Если ваш шлюз не обслуживает gpt-* — прокси-маппер (пример services/zen_proxy.py в content-factory) + `OPENAI_BASE_URL` на него.
3. `LANGFLOW_SSRF_ALLOWED_HOSTS` — если base_url указывает на link-local/hostname хоста.
4. `LANGFLOW_SECRET_KEY` фиксировать ДО создания Credential-переменных: иначе пересоздание контейнера = новые ключи шифрования = старые Credential не расшифруются.
5. `GET /agentic/check-config`; отключение — `LANGFLOW_AGENTIC_EXPERIENCE=false`.

**Модель (verified):** ✅ `x-preview-f-free` — надёжно собирает канвас; hy3-free флапает («no canvas actions»); nemotron-3-ultra-free ловил 502 upstream. Передавайте `model_name` явно в каждом вызове.

## Failure Modes (live-verified)## Failure Modes (live-verified)
- **Пустой канвас в UI при рабочем REST** → битая top-level handle-строка эджа (см. AW-graph); лечится fix_handle_strings.py.
- **`Error building Component Prompt Template: '<var>'`** → переменная промпта не забиндилась (см. AW-prompt); переносить инструкцию в system_prompt.
- **`Edge between X and Y has no matched type`** → источник CustomComponent в строгий вход (см. AW-custom-src).

| Symptom | Cause | Fix |
|---|---|---|
| Gallery template builds but run 500s | blank credential in `needs_keys` you skipped, or model flap | fill `needs_keys` fields; on model-level 403/500 swap `model_name` to another free model and re-run |
| `list_templates` returns too much | no query given (129 templates) | pass `query` tokens + `source="gallery"` |
| Run 500 "No model selected" (Agent) | modern Agent needs `model` ModelInput value or a wired model node | pass `model_name` param (R3) or connect OpenAIModelComponent `.model_output`→`.model` |
| Run 500 "Model name/provider overrides require a built-in model selection" | `model_name` set without matching provider plumbing | use R3 params; ensure `OPENAI_COMPATIBLE_*` globals exist |
| `add_node` "component not found" | display name instead of type name | `search_components` → exact `name` |
| Run 500 `unhashable type: 'dict'` | flow saved by OLD server version (corrupted `_type`) | rebuild that flow from scratch; current server round-trips scalars safely |
| Run 500 `'NoneType' object is not iterable` | old broken edge format | recreate edges with current `connect_nodes` |
| Run "Missing credentials" despite api_key set | fixed since v2: literals clear `load_from_db` | check value non-empty; for OpenAI Compatible prefer globals (R3) |
| 403 "Invalid API key" | stale key after reinstall | create new key in UI Settings |
| GET flow 404 right after create | Bearer-JWT auth path | switch to `LANGFLOW_MCP_API_KEY` |
| `explore_langflow` "not found" | source not cloned | `setup_langflow_source` once |

## Transport & Config

stdio (default) or `--http :8080` (`/mcp`, `/health`). Priority CLI > ENV > default. Key vars: `LANGFLOW_MCP_LANGFLOW_URL`, `LANGFLOW_MCP_API_KEY`, `LANGFLOW_MCP_TEMPLATES_DIR` (default `./templates`), `LANGFLOW_MCP_LOG_LEVEL=debug`.
