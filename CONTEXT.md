# LKG-001: Project Scaffold & CI Pipeline — Complete
# LKG-002: AgentEvent Types & Core Agent Loop — Complete
# LKG-003: DeepSeek API Client — Complete
# LKG-004: Docker Sandbox Lifecycle — Complete
# LKG-006: Built-in Sandbox Tool Set — Complete
# LKG-007: SQLite Session Persistence — Complete
# LKG-008: Session Resume — Complete
# LKG-009: Full TUI Shell — Complete
# LKG-010: Sandbox Network Policy & Resource Limits — Complete
# LKG-013: Hooks Framework — Complete

## Summary

Initialized Last Known Good as a Go binary with Cobra CLI, internal package skeletons,
Docker Compose dev environment, and GitHub Actions CI.

## What was built (LKG-001)

- **Go module**: `github.com/DietKyle956/last-known-good` (Go 1.26)
- **Entrypoint**: `cmd/agent/main.go` with Cobra root command
- **Subcommands**: `agent chat` and `agent run` (stubs)
- **Internal packages**: `core`, `agent`, `llm`, `sandbox`
- **Docker Compose**: `Dockerfile` + `docker-compose.yml` for dev environment
- **CI**: `.github/workflows/ci.yml` — build, test, lint on push/PR to main
- **Import constraint**: `.github/check-imports.sh` prevents domain → infra imports
- **Lint**: `.golangci.yml` config with gofmt, govet, errcheck, staticcheck, unused, ineffassign

## What was built (LKG-002)

- **`AgentEvent`** typed event stream with `EventModelResponseChunk`, `EventToolCallStarted`, `EventToolCallFinished`, `EventTurnComplete`, `EventError`
- **`Agent`** core loop that builds messages, calls the model, dispatches tool calls, appends tool results, and loops until a final content turn completes
- **`Message`**, **`ToolCall`**, **`ToolResult`** — core conversation types
- **`LLM` interface** (`Chat(messages) → <-chan Result`) for model backends
- **`ToolExecutor` interface** (`Execute`, `IsReadOnly`) for tool dispatch
- Read-only tool calls execute concurrently; write tool calls execute sequentially
- All lifecycle events emitted synchronously in deterministic order
- Event channel is the sole instrumentation boundary

## What was built (LKG-003)

- **`DeepSeekClient`** implements `agent.LLM` for DeepSeek's OpenAI-compatible chat completions endpoint
- **Project-owned request/response structs** — `DeepSeekRequest`, `DeepSeekResponse`, `DeepSeekChunk`, etc. — no third-party SDK types
- **Non-streaming mode**: returns complete content in one response
- **Streaming mode**: yields content chunks via server-sent events and terminates on `[DONE]`
- **Model targeting**: supports `deepseek-v4-pro` and `deepseek-v4-flash` via `DeepSeekConfig.Model`
- **Thinking mode**: `{"thinking": {"type": "enabled"}}` in the request body
- **Reasoning effort**: supports `reasoning_effort` values (`high`, `max`, etc.)
- **Tool call parsing**: extracts `ToolCall` (ID, name, arguments) from response
- **Error handling**: malformed responses return errors via the result channel, no panics
- **Test coverage**: 12 tests across types, non-streaming, streaming, thinking, reasoning effort, request payload shape, and error paths

## What was built (LKG-004)

- **`SessionHandle`** — opaque handle with no host filesystem path (tools receive only this handle)
- **`Start(projectDir)`** — creates one Docker container per session with `docker run -d --rm`, bind-mounts project dir at `/workspace`, uses `alpine` image with `sleep infinity`
- **`Exec(handle, command)`** — runs commands in the same container via `docker exec`, reuses the container across calls within a session
- **`Stop(handle)`** — removes the container with `docker rm -f`, no orphaned containers
- **Bind mount**: files written on host are visible at `/workspace` inside container and vice versa
- **Isolation**: files outside the mounted project directory are not accessible from inside the container
- **Test coverage**: 7 tests against real Docker daemon — container creation/removal, command reuse, bidirectional file visibility, filesystem isolation, interrupt cleanup, orphan prevention

## What was built (LKG-006)

- **`ToolFn`** signature changed to `func(*sandbox.SessionHandle, core.ToolCall) core.ToolResult` — tools receive a sandbox handle for all execution
- **`Registry`** stores a `*sandbox.SessionHandle` instead of `sandbox.Sandbox`, removes the need for mock sandboxes in tool tests
- **`RegisterAll(reg)`** — registers all seven built-in tools at once
- **`read_file`** — reads a file at the given path inside the sandbox via `cat`, returns contents or error
- **`write_file`** — writes content to a file inside the sandbox via `printf`, overwrites if exists
- **`edit_file`** — find-and-replace text in a file using `sed -i`, leaves rest of file intact
- **`bash`** — runs a shell command inside the sandbox with stderr merged into stdout on failure
- **`grep`** — searches for a pattern in a file using `grep -rn`, returns matching lines
- **`glob`** — lists files matching a glob pattern using `find -type f -name`
- **`git_diff`** — shows unstaged changes at `/workspace` via `git diff`
- **`sandbox.Exec`** — changed to `CombinedOutput()` for stderr capture; adds `-w /workspace` for consistent working directory
- **All tools** execute entirely through the `sandbox.SessionHandle` — no direct host execution
- **Test coverage**: 21 tests against real Docker containers — registry dispatch, schema validation, read/write/edit file round-trips, bash stdout/error, grep match/no-match, glob match/no-match, git diff with/without changes

## What was built (LKG-007)

- **`internal/store`** — SQLite-backed session persistence layer using `modernc.org/sqlite` (pure Go, no CGo, no external daemon)
- **Schema migration**: `CREATE TABLE IF NOT EXISTS` applied on every `New()` call — tables: `sessions`, `messages`, `tool_calls`, `hook_events`
- **`CreateSession()`** — creates a session record and returns its auto-increment ID
- **`SaveMessage(sessionID, role, content, model)`** — saves a message with ordinal ordering
- **`GetMessages(sessionID)`** — returns all messages for a session in insertion order
- **`SaveToolCall(sessionID, name, args, result, isError, durationMs)`** — saves a tool call with full metadata
- **`SaveHookEvent(sessionID, eventType, payload)`** — saves a hook lifecycle event
- **Durability**: data survives close/reopen because it's written to a real SQLite file
- **Test coverage**: 7 tests against real temporary SQLite files — schema application, CRUD, ordering, durability across restart, session isolation

## What was built (LKG-008)

- **`Store.SessionExists(id)`** — checks whether a session exists in the database, returns a clear boolean
- **`internal/session`** package — bridges `store` and `core` for session lifecycle
- **`Resume(store, sessionID)`** — loads a session's messages from SQLite as `[]core.Message`, returns an error if the session does not exist
- **`SaveMessages(store, sessionID, messages)`** — persists new `core.Message` values to an existing session record
- **State equivalence**: round-trip (save → resume → compare) preserves role, content, and ordering
- **No domain → infra imports**: `session` is not in the domain list, so it can safely import `store` without violating the import constraint
- **Test coverage**: 5 tests — resume loads messages, non-existent session error, save to session, round-trip equivalence, message type checks

## What was built (LKG-009)

- **`internal/tui`** — Bubble Tea terminal UI with `tea.Model` interface
- **`tui.New(events, submit)`** — creates a model that subscribes to an `<-chan agent.AgentEvent` and sends user prompts via `chan<- string`
- **Omarchy palette**: dark navy background (`#00172e`), cream text (`#f6dcac`), teal assistant labels (`#028391`), orange user text (`#faa968`), muted teal tool bodies (`#3f8f8a`), light teal results (`#8cbfb8`), orange-red errors (`#f85525`)
- **Scrolling viewport**: single conversation viewport using `bubbles/viewport.Model`
- **Streaming chunks**: model response chunks accumulate into a single assistant message line
- **Tool call blocks**: inline collapsed blocks with one-line summary; expand to show full detail
- **Error tool calls**: shown expanded by default with bold orange-red styling
- **Fixed input bar**: always visible at the bottom, styled with the Omarchy palette
- **Coordinator loop**: goroutine manages agent lifecycle across turns — creates new agent per turn, forwards events to shared channel, waits for user prompts via submit channel
- **Chat command**: `agent chat` launches the TUI with a DeepSeek LLM client and stub tool executor
- **Import constraint**: `tui` is a domain package; does not import any infrastructure packages
- **Test coverage**: 11 tests — model initialization, event handling, chunk accumulation, channel consumption, prompt submission, tool call lifecycle, and error display

## What was built (LKG-010)

- **`SandboxConfig`** struct with `Network` (allowlist) and `CPU`/`Memory` fields, passed to `Start(projectDir, cfg)`
- **Default isolation**: nil or empty `Network` config applies `--network=none` to block all outbound traffic
- **Domain allowlist**: resolves each domain to its IPv4 address, writes `/etc/hosts` entries, and overrides `/etc/resolv.conf` to `nameserver 127.0.0.1` to block DNS for non-listed domains
- **CPU limits**: `--cpus` flag passed to `docker run` when `cfg.CPU` is set
- **Memory limits**: `--memory` flag passed to `docker run` when `cfg.Memory` is set
- **Test coverage**: 4 new tests against real Docker containers — default no-network, allowlist reachable, allowlist blocks others, CPU/memory limits verified via `docker inspect`

## Package structure

```
cmd/agent/          main.go + cmd/ (root, chat, run)
internal/
  core/             shared domain types + AgentEvent types (complete)
  agent/            core agent loop + event types + Session (complete)
  hooks/            typed hooks framework (complete)
  llm/              DeepSeek API client (complete)
  router/           pluggable model router (complete)
  sandbox/          Docker sandbox lifecycle + Execer interface (complete)
  tools/            tool registry + 7 built-in sandbox tools (complete)
  singleshot/       single-shot CLI renderer (text + JSON) (complete)
  store/            SQLite session persistence + SaveMessages/Resume (complete)
  tui/              Bubble Tea terminal UI (complete)
```

## TDD approach

Built with vertical tracer-bullet slices — one test → one implementation per cycle.

### LKG-001 slices

| Slice | What |
|-------|------|
| 1 | Module + binary that builds and prints help |
| 2 | All 9 internal packages compile |
| 3 | `chat` and `run` subcommands registered |
| 4 | Full `go test ./...` suite passes |
| 5 | Docker Compose dev environment |
| 6 | GitHub Actions CI pipeline |
| 7 | Lint config + import constraint enforcement |

### LKG-002 slices

| Slice | What |
|-------|------|
| 1 | Core types + agent emits TurnComplete when model returns content |
| 2 | Agent dispatches tool calls and loops back to model |
| 3 | Tool call lifecycle events (ToolCallStarted, ToolCallFinished) |
| 4 | Chunk events (ModelResponseChunk) |
| 5 | Error events (LLM failure + stream error) |
| 6 | Event ordering for multi-tool sequence |
| 7 | Parallel read-only tool execution |
| 8 | Sequential write tool execution |

### LKG-003 slices

| Slice | What |
|-------|------|
| 1 | Request/response structs with JSON round-trip |
| 2 | Non-streaming Chat returns complete content |
| 3 | Tool call parsing from response |
| 4 | Streaming yields content chunks |
| 5 | Malformed response returns error |
| 6 | Thinking mode in request body |
| 7 | Reasoning effort in request body |
| 8 | Request payload shape matches DeepSeek API format |

### LKG-004 slices

| Slice | What |
|-------|------|
| 1 | `Start` creates a container, `Stop` removes it |
| 2 | `Exec` runs commands and reuses the same container |
| 3 | File written on host is visible inside container |
| 4 | File written inside container is visible on host |
| 5 | Files outside mount are inaccessible from container |
| 6 | Container removed on simulated interrupt |
| 7 | No orphaned containers after session ends |

### LKG-007 slices

| Slice | What |
|-------|------|
| 1 | Schema applies cleanly to new DB file |
| 2 | CreateSession creates a record and returns its ID |
| 3 | SaveMessage + GetMessages preserves ordering |
| 4 | SaveToolCall stores all fields (name, args, result, error, duration) |
| 5 | SaveHookEvent stores event type and payload |
| 6 | Close & reopen — data persists in the SQLite file |
| 7 | Multiple sessions are isolated (messages don't leak between sessions) |

### LKG-008 slices

| Slice | What |
|-------|------|
| 1 | `Store.SessionExists` — check if a session exists |
| 2 | `session.Resume` — load messages from a session, error on missing |
| 3 | `session.SaveMessages` — persist new messages to an existing session |
| 4 | Full round-trip state equivalence — save, resume, verify identity |

### LKG-009 slices

| Slice | What |
|-------|------|
| 1 | Package compiles, minimal Bubble Tea model initializes |
| 2 | Event channel consumption — streaming chunks render in viewport |
| 3 | Input bar and prompt submission via submit channel |
| 4 | Tool call blocks (collapsed by default) |
| 5 | Error tool calls (expanded, distinct styling) |
| 6 | Wire into chat command |

### LKG-010 slices

| Slice | What |
|-------|------|
| 1 | `SandboxConfig` type + `Start` signature change, `--network=none` default |
| 2 | Allowlist with per-domain `/etc/hosts` entries and DNS block via `/etc/resolv.conf` |
| 3 | CPU and memory limits via `--cpus` and `--memory` |
| 4 | Tests: default blocks outbound, allowlist reaches domain, allowlist blocks other, limits verified |

### LKG-006 slices

| Slice | What |
|-------|------|
| 1 | Interface change: `ToolFn` takes `*SessionHandle`, Registry stores handle |
| 2 | `read_file` tool — file exists returns content, missing returns error |
| 3 | `write_file` tool — creates file, overwrites existing |
| 4 | `edit_file` tool — find-and-replace leaves rest intact |
| 5 | `bash` tool — stdout on success, stderr on non-zero exit |
| 6 | `grep` tool — matches return lines, no matches returns empty |
| 7 | `glob` tool — matches return paths, no matches returns empty |
| 8 | `git_diff` tool — changes return diff, clean returns empty |
| 9 | All seven tools registered and schemas validated |
| 10 | `Exec` uses `CombinedOutput` + `-w /workspace` |

## What was built (LKG-012)

### Architecture deepening (slices 1-6)

- **Unit tests for pure functions**: 27 tests for `validateArgs`, `validateProperty`, `quote`, `escapeSed` in `internal/tools/tools_internal_test.go` — zero external dependencies, run without Docker
- **Iterative agent loop**: `agent.loop()` removed, `Run()` now uses a `for` loop — stack depth is O(1) regardless of tool call turns
- **Session package eliminated**: `internal/session/` deleted; `SaveMessages` and `Resume` moved as methods on `*store.Store` — tests migrated to `store_test.go`
- **`sandbox.Execer` interface**: narrow `Exec(command) (string, error)` interface extracted; `Sandbox` dead interface removed; `NewDockerExecer` adapter wraps `*SessionHandle` for production; `mockExecer` enables unit tests without Docker
- **`context.Context` threaded through stack**: all key interfaces (`LLM.Chat`, `ToolExecutor.Execute`, `Agent.Run`, `sandbox.Execer.Exec`, `sandbox.Exec`) now accept `ctx`; `http.NewRequestWithContext` used in LLM client; `exec.CommandContext` used in sandbox; cancellation tests added
- **`Session` type extracted**: multi-turn agent lifecycle formalized as `internal/agent.Session` with `Run(ctx, messages, submit, events)` — replaces inline `coordinator` goroutine in `chat.go`; tested for single/multi-turn, empty prompt, and context cancellation

### Heuristic Model Router & Thinking Mode (slice 7)

- **`internal/router`** package — pluggable model routing for the agent loop
- **`Router` interface** with `Route(ctx, RouteRequest) RouteDecision` — routing logic can be replaced without touching the agent loop
- **`HeuristicRouter`** implementation deciding model and thinking mode based on:
  - **Single-file turn**: routes to `deepseek-v4-flash` with thinking disabled
  - **Multi-file turn** (above configurable threshold): routes to `deepseek-v4-pro` with thinking enabled
  - **Post-failure turn**: routes to `deepseek-v4-pro` with thinking enabled
  - **Complexity signal word** in prompt: routes to `deepseek-v4-pro` with thinking enabled
  - **Flash retry after failure** (opt-in via `PostFailureModel`): Flash with thinking enabled
- **`NewSessionWithRouter`** — alternate Session constructor using a Router + LLM factory per turn
- 6 independent router unit tests + 1 Session integration test covering all routing scenarios

## Package structure

```
cmd/agent/          main.go + cmd/ (root, chat, run)
internal/
  core/             shared domain types + AgentEvent types (complete)
  agent/            core agent loop + event types + Session (complete)
  hooks/            typed hooks framework (complete)
  llm/              DeepSeek API client (complete)
  router/           pluggable model router (complete)
  sandbox/          Docker sandbox lifecycle + Execer interface (complete)
  tools/            tool registry + 7 built-in sandbox tools (complete)
  singleshot/       single-shot CLI renderer (text + JSON) (complete)
  store/            SQLite session persistence + SaveMessages/Resume (complete)
  tui/              Bubble Tea terminal UI (complete)
```

### LKG-012 slices

| Slice | What |
|-------|------|
| 1 | Unit tests for validateArgs, validateProperty, quote, escapeSed |
| 2 | Flatten recursive agent loop into iterative for loop |
| 3 | Eliminate session package, move logic to store |
| 4 | Activate sandbox Execer interface, remove dead Sandbox interface |
| 5 | Thread context.Context through agent, llm, sandbox, tools, CLI |
| 6 | Extract Session type from chat.go coordinator |
| 7 | Heuristic Model Router & Thinking Mode |

### LKG-006 updated

- **`ToolFn`** signature changed to `func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult` — tools receive an `Execer` instead of `*SessionHandle`
- **`Registry`** stores a `sandbox.Execer` instead of `*SessionHandle`
- **Mock-based tests**: `mockExecer` enables tool tests without Docker (2 new unit tests)
- **`dockerExecer` adapter**: `NewDockerExecer(h)` wraps `*SessionHandle` for production use

## What was built (LKG-013)

- **`internal/hooks`** — typed hooks system that fires around session lifecycle, model calls, and tool calls
- **`HookType`** enum with six types: `SessionStarted`, `SessionEnded`, `BeforeModelCall`, `AfterModelCall`, `BeforeToolCall`, `AfterToolCall`
- **`HookFunc`** callback type — non-nil `*HookResult{Block: true}` prevents tool execution (BeforeToolCall only)
- **`System`** — registry of `map[HookType][]HookFunc` with `Register` and `Notify`; synchronous dispatch returns block flag
- **`New(events)`** — subscribes to `<-chan core.AgentEvent` and dispatches channel events as hook events (same channel as TUI/rendered)
- **`Session.SetHooks`** / **`Agent.SetHooks`** — attaches hooks system; Session emits `SessionStarted`/`SessionEnded`, Agent emits `BeforeModelCall`/`AfterModelCall`/`BeforeToolCall`/`AfterToolCall`
- **Blocking**: `BeforeToolCall` hooks checked before tool execution; blocked tools never reach `Execute` and get "blocked by hook" error result
- **AfterToolCall ignores block**: block return value from `AfterToolCall` hooks is silently ignored — the tool already ran
- **Registration order**: multiple hooks for the same event fire in registration order
- **No dynamic loading**: all hooks registered as compiled Go code at startup
- **`AgentEvent` types** moved from `internal/agent` to `internal/core` to break import cycle (`hooks`→`core`←`agent`)
- **Type aliases** in `agent` package (`AgentEvent`, `AgentEventType`, event constants) preserve backward compatibility
- **Test coverage**: 10 hooks unit tests + 3 agent integration tests covering all acceptance criteria

## Package structure

```
cmd/agent/          main.go + cmd/ (root, chat, run)
internal/
  core/             shared domain types + AgentEvent types
  agent/            core agent loop + event types + Session (complete)
  hooks/            typed hooks framework (complete)
  llm/              DeepSeek API client (complete)
  router/           pluggable model router (complete)
  sandbox/          Docker sandbox lifecycle + Execer interface (complete)
  tools/            tool registry + 7 built-in sandbox tools (complete)
  singleshot/       single-shot CLI renderer (text + JSON) (complete)
  store/            SQLite session persistence + SaveMessages/Resume (complete)
  tui/              Bubble Tea terminal UI (complete)
```

### LKG-013 slices

| Slice | What |
|-------|------|
| 1 | Tracer bullet: package compiles, Register+Notify for SessionStarted |
| 2 | SessionEnded hook fires |
| 3 | BeforeModelCall / AfterModelCall hooks fire |
| 4 | BeforeToolCall can block, blocked tool not executed |
| 5 | AfterToolCall cannot block, multiple hooks fire in registration order |
| 6 | Agent integration: Session/Agent lifecycle calls |
| 7 | Channel subscription from AgentEvent channel (criterion 12) |
