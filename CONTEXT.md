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
# LKG-014: Blocking Hook for Dangerous Commands — Complete
# LKG-017: Structured JSONL Logging — Complete
# LKG-018: CLI Session & Log Commands — Complete
# LKG-019: Prompt-Cache-Friendly Request Shaping — Complete
# LKG-020: Strict JSON Schema Tool Mode — Complete
# LKG-023: Last Known Good System Prompt — Complete
# LKG-021: Per-Tool Timeout & Max-Iteration Guard — Complete
# LKG-022: Full TUI Redesign — Complete

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

## What was built (LKG-009 — superseded by LKG-022)

The original minimal Bubble Tea TUI shell. Completely redesigned in LKG-022 with a full modern chat interface, welcome screen, and brand palette. See LKG-022 section for current implementation.

## What was built (LKG-010)

- **`SandboxConfig`** struct with `Network` (allowlist) and `CPU`/`Memory` fields, passed to `Start(projectDir, cfg)`
- **Default isolation**: nil or empty `Network` config applies `--network=none` to block all outbound traffic
- **Domain allowlist**: resolves each domain to its IPv4 address, writes `/etc/hosts` entries, and overrides `/etc/resolv.conf` to `nameserver 127.0.0.1` to block DNS for non-listed domains
- **CPU limits**: `--cpus` flag passed to `docker run` when `cfg.CPU` is set
- **Memory limits**: `--memory` flag passed to `docker run` when `cfg.Memory` is set
- **Test coverage**: 4 new tests against real Docker containers — default no-network, allowlist reachable, allowlist blocks others, CPU/memory limits verified via `docker inspect`

## TDD approach

Built with vertical tracer-bullet slices — one test → one implementation per cycle.

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
cmd/agent/          main.go + cmd/ (root, chat, run, sessions, logs)
internal/
  core/             shared domain types + AgentEvent types
  agent/            core agent loop + event types + Session (complete)
  hooks/            typed hooks framework (complete)
  llm/              DeepSeek API client (complete)
  router/           pluggable model router (complete)
   sandbox/          Docker sandbox lifecycle + Execer interface (complete)
  skills/           file-based skills system with lazy loading (complete)
  tools/            tool registry + 7 built-in sandbox tools (complete)
  singleshot/       single-shot CLI renderer (text + JSON) (complete)
  store/            SQLite session persistence + SaveMessages/Resume (complete)
  tui/              Bubble Tea terminal UI with full redesign (complete)
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

## What was built (LKG-014)

- **`HookResult.Reason`** — field added so blocking hooks can explain why a command was denied
- **`Notify` returns `*HookResult`** — changed from `bool` to `*HookResult`; `nil` means no block, non-nil carries the first blocking hook's `Block`/`Reason`
- **`DangerousCommandHook`** — `BeforeToolCall` hook in `internal/hooks/dangerous.go` that inspects `bash` tool commands for dangerous patterns
- **Configurable patterns**: `NewDangerousCommandHook(patterns)` takes a `[]string` of regex patterns; pass `nil` to use `DefaultDangerousPatterns()`
- **Default dangerous patterns**: recursive delete (`rm -rf /`), filesystem formatting (`mkfs.*`), raw disk writes (`dd`, `> /dev/sdX`), fork bombs (`:(){...}`), mass permissions change (`chmod -R 777 /`), remote code execution pipes (`wget ... | sh`, `curl ... | sh`)
- **Non-bash tools pass through**: only `bash` tool calls are inspected; `read_file`, `write_file`, etc. are not affected
- **Invalid JSON or nil ToolCall**: the hook is lenient — parse failures pass through without blocking
- **Blocked commands return structured error**: agent receives `ToolResult{IsError: true, Content: "blocked: command matches dangerous pattern %q"}` instead of crashing; the reason flows back to the model as the tool response
- **Wired into `agent run`**: `run.go` creates a `hooks.System`, registers `DangerousCommandHook`, and attaches it to the agent
- **Test coverage**: 9 dangerous hook unit tests (block/no-block, safe command, non-bash, custom patterns, nil/invalid edge cases) + 2 agent integration tests (blocked produces error result, safe passes through)

## What was built (LKG-015)

- **`AutoFormatHook`** — `AfterToolCall` hook in `internal/hooks/autoformat.go` that runs a language formatter inside the sandbox after a file write to a recognized source file type
- **Configurable formatters**: `NewAutoFormatHook(execer, formatters, onFailure)` takes a `map[string]string` of extension → formatter command template; pass `nil` to use `DefaultFormatters()` (`.go` → `gofmt -w %s`)
- **Recognized extension triggers formatter**: when a `write_file` tool call writes a file whose extension matches a configured formatter, the formatter runs inside the sandbox via the execer
- **Unrecognized extension is skipped**: files with extensions not in the formatters map (e.g., `.txt`, `.md`) do not invoke any formatter
- **Non-write tool calls pass through**: `bash`, `read_file`, etc. are not inspected
- **Format failure is non-fatal**: a formatting error does not crash or halt the agent loop; the hook returns nil (AfterToolCall blocks are ignored)
- **Failure callback**: `onFailure(path, command, err)` is called when a formatter fails, allowing callers to record the event (e.g., via `store.SaveHookEvent`)
- **Wired into `agent run`**: `run.go` creates the `AutoFormatHook` with the sandbox execer and registers it as an `AfterToolCall` hook
- **Test coverage**: 13 unit tests — formatter runs for .go, skipped for unrecognized extensions, skipped for non-write_file tools, nil/invalid JSON edge cases, failure callback invoked, custom formatters work, default formatters, Formatters() returns a copy

## What was built (LKG-016)

- **`internal/skills`** — file-based skills system with lazy loading
- **`Skill`** struct with `Name`, `Description`, `Body` — body is empty until explicitly read
- **`Loader`** — discovers skills from a base directory; each skill lives in its own subfolder containing a markdown file with YAML frontmatter
- **`Load()`** — scans the base directory for skill subfolders, reads each `.md` file, parses frontmatter for `name` and `description`; skips folders without markdown files, folders with missing/malformed frontmatter, and frontmatter without required fields — all silently, without crashing the loader
- **`Summaries()`** — returns `[]Skill` with only `Name` and `Description` populated (body not loaded) — suitable for system prompt injection at session start
- **`ReadBody(name)`** — lazily reads the full markdown body from disk for a named skill; returns an error for unknown skill names
- **Lazy loading**: the full markdown body of a skill is never loaded into memory until `ReadBody` is called for that specific skill
- **Frontmatter format**: standard YAML-style `---` delimited block with `name:` and `description:` keys; quoted and unquoted values supported; unclosed quotes treated as malformed
- **No external dependencies**: pure Go standard library — no YAML parser, no additional modules
- **Test coverage**: 10 tests — valid frontmatter with body, valid frontmatter without body, missing frontmatter, malformed frontmatter, missing required fields, multi-skill directory discovery, empty directory, lazy body loading, unknown skill returns error, skill without body returns empty

## What was built (LKG-017)

- **`internal/logger`** — per-session structured JSONL logging package
- **`Logger`** struct with `New(sessionID, dir)` constructor and `Close()` — creates `session_<id>.jsonl` in the specified directory
- **`Hook(ev) *hooks.HookResult`** — implements `hooks.HookFunc`; writes a JSON object per line with `timestamp`, `session_id`, `type`, and optional `model`, `tool_call`, `tool_result`, `error` fields
- **All 6 hook types** are logged: `SessionStarted`, `SessionEnded`, `BeforeModelCall`, `AfterModelCall`, `BeforeToolCall`, `AfterToolCall`
- **Tool call details**: `id`, `name`, `arguments` captured from `BeforeToolCall` / `AfterToolCall`
- **Tool result details**: `tool_call_id`, `content`, `is_error` captured from `AfterToolCall`
- **Error events**: error message string captured from `AfterModelCall` error events
- **Model field**: model name logged when present in the hook event
- **Concurrent-safe**: mutex-protected writes; file is readable by other processes while the session is still running
- **File persists**: log file remains on disk after `Close()` — not deleted when the session ends
- **Per-session isolation**: each session ID gets its own `session_<id>.jsonl` file
- **Wired into `agent run`**: logger created with timestamp-based session ID in `./logs/` directory; registered as a hook for all 6 lifecycle event types
- **Test coverage**: 9 tests — file creation, JSONL format, concurrent readability, session isolation, all 6 hook types, tool call fields, error field, model field, file persistence after close

### LKG-017 slices

| Slice | What |
|-------|------|
| 1 | Logger creates session log file |
| 2 | Hook events written as JSONL lines with correct fields |
| 3 | File readable while logger is active (no deferred close) |
| 4 | Separate log files for different sessions |
| 5 | All 6 hook types logged with correct type strings |
| 6 | Tool call fields (id, name, arguments) and tool result fields (content, is_error) |
| 7 | Error field captured from error events |
| 8 | File persists on disk after Close() |

## What was built (LKG-018)

- **`sessions list`** — CLI command that queries the store for past sessions and prints ID + created_at to stdout; when no sessions exist, prints "no sessions found" instead of an error
- **`sessions resume <id>`** — CLI command that loads a session's messages from the store via `Store.Resume(id)`, initializes an LLM client, and enters the agent loop in TUI mode with the loaded messages; invalid IDs produce a clear error; missing `id` arg is rejected by Cobra's `ExactArgs(1)`
- **`logs <id>`** — CLI command that reads a session's `session_<id>.jsonl` file from the `./logs/` directory and prints it to stdout; missing session IDs produce a clear error
- **`logs --follow <id>`** / **`logs -f <id>`** — streams new JSONL log lines to stdout as they are written, using a polling `bufio.Reader` loop
- **`Store.ListSessions()`** — new store method returning `[]SessionRecord` with `ID` and `CreatedAt`, ordered by ID descending; returns an empty slice (not nil) when no sessions exist
- **`SessionRecord`** type — exported struct with `ID int64` and `CreatedAt string` fields
- **Shared defaults**: sessions commands default to `./lkg.db` for the store DB path; logs command defaults to `./logs/` for the log directory; both paths can be overridden via `--db` / `--log-dir` flags
- **No duplicate persistence**: sessions commands read from the same store that `Store.CreateSession()` writes to; logs command reads from the same log files the `logger` package creates
- **Test coverage**: 9 CLI unit tests (registration, arg validation, flag presence, error handling) + 2 store unit tests for `ListSessions` (returns sessions, empty when none)

### LKG-018 slices

| Slice | What |
|-------|------|
| 1 | `Store.ListSessions` — query sessions table, return records with ID + created_at |
| 2 | `sessions list` command — reads from store, prints formatted rows, handles empty DB |
| 3 | `sessions resume` command — validates ID arg, opens store, calls `Resume`, enters TUI agent loop |
| 4 | `logs` command — reads `session_<id>.jsonl` from logs directory, prints to stdout |
| 5 | `logs --follow` — polls log file for new lines and streams them in real time |

## What was built (LKG-023)

- **`internal/core/prompt.go`** — new file containing the Last Known Good persona as a `const Persona` and a `BuildSystemPrompt(skillSummaries, toolDescriptions string) string` function
- **Persona**: `"You are Last Known Good, a software development assistant."` — direct, deadpan, sarcastic, and witty; replaces the generic `"You are a helpful assistant."`
- **`BuildSystemPrompt`** composes the persona, an optional `## Available Skills` section from skill summaries, and an optional `## Available Tools` section from tool definitions
- **No import coupling**: `BuildSystemPrompt` takes pre-formatted strings so `core` does not import `skills` or `tools`
- **`cmd/agent/cmd/chat.go`** — instantiates `skills.NewLoader("skills")`, calls `BuildSystemPrompt` with skill summaries and empty tools
- **`cmd/agent/cmd/run.go`** — instantiates `skills.NewLoader("skills")`, formats tool definitions from `reg.ToolDefinitions()`, calls `BuildSystemPrompt` with both
- **Skills directory**: if `skills/` does not exist, the loader is a no-op; if it exists, skill names and descriptions are injected into the system prompt
- **Test coverage**: 3 unit tests for `BuildSystemPrompt` — returns persona without sections, includes skills section when provided, includes tools section when provided

### LKG-023 slices

| Slice | What |
|-------|------|
| 1 | `BuildSystemPrompt` returns persona text without sections |
| 2 | Skills summaries section included when provided |
| 3 | Tool definitions section included when provided |
| 4 | Wired into `chat.go` |
| 5 | Wired into `run.go` |

## What was built (LKG-019)

- **Deterministic `ToolDefinitions()` order**: `Registry.ToolDefinitions()` now returns tools sorted by name, eliminating non-deterministic map iteration order from the system prompt and tool definition payload
- **`internal/llm/types.go`**: Added `Tools []DeepSeekToolDef` field to `DeepSeekRequest`, and `DeepSeekToolDef`/`DeepSeekFunction` types for the OpenAI-compatible `tools` API parameter
- **`internal/llm/json.go`**: `DeterministicMarshal(v)` helper that produces JSON with map keys sorted lexicographically (recursively), ensuring byte-identical output for the same data structure
- **`internal/llm/client.go`**: Added `SetTools(tools)` method on `DeepSeekClient`; `buildRequest` includes the `tools` field in every API request when tools are configured
- **`cmd/agent/cmd/run.go`**: After registering all tools, calls `client.SetTools(toDeepSeekTools(reg.ToolDefinitions()))` with sorted, deterministically-serialized tool definitions
- **No timestamp/counter**: The system prompt and tool definitions contain no per-call dynamic values — all fields are session-stable
- **Test coverage**: 2 tool-level tests (sorted order, byte-identical payload across calls) + 1 LLM client test (full request body byte-identical across consecutive calls) + 5 deterministic JSON unit tests

## What was built (LKG-020)

- **`internal/tools/strict.go`**: `SchemaSupportsStrict(schema map[string]any) bool` — recursive check that determines whether a JSON Schema qualifies for DeepSeek's strict schema mode. Rejects schemas with `enum`, `anyOf`, `oneOf`, `allOf`, `not`, `$ref`, `const`, `nullable`, `additionalProperties: true`, non-object root, or unsupported property types (only `string`, `number`, `boolean`, `array` with valid `items`, and `object` with valid `properties` are allowed).
- **`Strict` field on `DeepSeekFunction`**: `internal/llm/types.go` — boolean `strict` field added to the function definition struct; serialized as `"strict": true` in the API request when set, omitted via `omitempty` when false.
- **`toDeepSeekTools` integration**: `cmd/agent/cmd/run.go` — each tool definition is individually checked with `SchemaSupportsStrict`; strict-compatible schemas get `Strict: true`, incompatible schemas get `Strict: false` (safe fallback with no error).
- **Coexistence**: strict-mode and non-strict-mode tools can be sent in the same API request — each tool is evaluated independently.
- **No result changes**: strict mode only affects the outgoing tool definition; tool call dispatch and result handling are unchanged.
- **Test coverage**: 8 `SchemaSupportsStrict` tests (simple object, all supported types, enum rejection, anyOf rejection, nullable rejection, additionalProperties rejection, non-object root rejection, $ref rejection) + 3 `toDeepSeekTools` integration tests (strict enabled, strict omitted, mixed strict/non-strict) + 2 LLM client tests (strict included in request body, strict omitted for unsupported schema).

### LKG-020 slices

| Slice | What |
|-------|------|
| 1 | `SchemaSupportsStrict` for simple object with string property |
| 2 | Reject enum, anyOf, oneOf, allOf, not, $ref, const, nullable, additionalProperties |
| 3 | Accept all supported property types (string, number, boolean, array, object) |
| 4 | Reject unsupported types (null, non-object root) |
| 5 | `Strict` field on `DeepSeekFunction` type |
| 6 | `toDeepSeekTools` sets strict mode per tool definition |
| 7 | Mixed strict/non-strict tools in same request |
| 8 | Request body tests — strict included/omitted in JSON |

### LKG-019 slices

| Slice | What |
|-------|------|
| 1 | Sort `ToolDefinitions()` by name + test |
| 2 | Deterministic JSON marshaler + test |
| 3 | `Tools` field on `DeepSeekRequest` + wired into `buildRequest` |
| 4 | `SetTools` on `DeepSeekClient` + wired into `run.go` |
| 5 | Byte-level identity acceptance test across consecutive calls |

## What was built (LKG-021)

- **`Agent.SetToolTimeout(duration)`** — configurable per-tool wall-clock timeout; zero means no timeout
- **`Agent.SetMaxToolCalls(n)`** — configurable max tool-call iterations per `Run` call; zero means unlimited
- **`timeoutContext`** — helper returns a derived context with `context.WithTimeout` and its cancel function; callers properly `defer cancel()` to avoid context leaks
- **Max iteration guard**: the agent loop counts tool-call iterations; when `MaxToolCalls > 0` and the counter reaches the limit, the loop emits `EventError` and returns instead of looping further
- **Per-tool timeout**: each tool execution in `executeToolCalls` (both parallel read-only and sequential write tools) gets a timeout-scoped context; when the deadline fires, `exec.CommandContext` kills the sandbox process and the tool returns a structured `ToolResult{IsError: true}`
- **No new event types**: timeouts flow through existing `EventToolCallFinished` with an error `ToolResult`; iteration limit uses `EventError`
- **No panics**: both guards return cleanly via the event channel
- **4 new tests**: timeout produces error result, timeout=0 is unlimited, max iteration stops with error, max=0 is unlimited

### LKG-021 slices

| Slice | What |
|-------|------|
| 1 | Config fields (`toolTimeout`, `maxToolCalls`) + setter methods |
| 2 | Max iteration guard: counter in `Run` loop, `EventError` on limit |
| 3 | Per-tool timeout: `timeoutContext` helper wrapping `context.WithTimeout` |
| 4 | Tests: max iteration stops with error, unlimited at zero |
| 5 | Tests: timeout returns error result, no-op at zero |

## What was built (LKG-022)

- **`internal/tui/styles.go`** — Omarchy retro-82 palette constants with 10 named colors (`#00172e` background, `#f6dcac` text, `#faa968` accent, `#3f8f8a` teal, `#8cbfb8` cyan, `#a7c9c6` muted, `#028391` green, `#e97b3c` yellow, `#f85525` red, `#134e5a` status bg) and 20+ reusable lipgloss styles for all UI components
- **`internal/tui/tui.go`** — Main Bubble Tea model with 4-zone layout (header, viewport, status bar, input), welcome screen with LKG ASCII art brand that transitions to full chat on first message or event, model badge in header, `SetModelName`/`SetSandboxState`/`SetTokenCount` public API for status bar updates
- **`internal/tui/input.go`** — Multi-line `bubbles/textarea` input with Omarchy-styled prompt (`>` in `#faa968`), placeholder text, and paste support
- **`internal/tui/messages.go`** — Role-based message bubbles with lipgloss containers: user messages with `#faa968` border, assistant messages with `#3f8f8a` border; inline markdown rendering (bold/italic) and code block detection with chroma syntax highlighting (`formatters.TTY16m`, `monokai` style)
- **`internal/tui/tools.go`** — Tool call card state tracking with name, arguments, result, error flag, and collapsed/expanded toggle
- **`internal/tui/status.go`** — Status bar with `#134e5a` background showing model name (`💬`), token count (`⚡`), and sandbox state (`🐳`), separated by `│` dividers
- **`internal/tui/help.go`** — Keyboard shortcut overlay using `bubbles/help` with keybinding definitions for send (`alt+enter`), quit (`ctrl+c`), help toggle (`ctrl+h`), collapse toggle (`ctrl+t`); styled teal/muted key colors
- **`internal/tui/tui_test.go`** — 24 tests covering: model lifecycle, event handling, chunk accumulation, channel consumption, prompt submission, tool call lifecycle, error display, welcome screen rendering, welcome-to-chat transition (on send and on chunk), status bar rendering with live data, user/assistant bubble rendering, markdown code block highlighting, text wrapping, and channel close
- **Welcome screen**: when `Model.started` is `false`, `View()` renders centered LKG block ASCII art in `#faa968`, subtitle "Last Known Good — the open source AI coding agent" in `#a7c9c6`, and the textarea input at approximately 1/3 screen height; `sendMessage()` and `handleChunk()` both set `started = true` to transition to full chat layout
- **No import constraint violations**: `tui` is a domain package and does not import any infrastructure packages
- **New dependencies**: `github.com/alecthomas/chroma/v2` for syntax highlighting, `github.com/charmbracelet/bubbles/textarea`, `bubbles/spinner`, `bubbles/help`

### LKG-022 slices

| # | Slice | What |
|---|-------|------|
| 1 | Brand palette & layout framework | `styles.go` with palette constants, resize-aware 4-zone layout (header, viewport, status bar, input) |
| 2 | Enhanced input area | `bubbles/textarea` — multi-line, paste, Omarchy-styled prompt and text |
| 3 | Rich message bubbles | Lipgloss containers with role-colored borders (user=#faa968, assistant=#3f8f8a), padding, labels |
| 4 | Markdown + syntax highlighting | Chroma-based code block highlighting with TTY16m formatter and monokai style |
| 5 | Spinner & animated indicators | `bubbles/spinner` (dot style) in #8cbfb8 replacing static "…" |
| 6 | Tool call cards | Collapsible state tracking with success/error indicators |
| 7 | Status bar with live data | #134e5a background — model, tokens, sandbox state |
| 8 | Help overlay | `bubbles/help` with styled key bindings |
| 9 | Welcome screen | ASCII art LKG brand + centered input, transitions on first message/event |
| 10 | Polish + tests | 24 tests covering all components and welcome state transitions |
