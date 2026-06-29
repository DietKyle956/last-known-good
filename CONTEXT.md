# LKG-001: Project Scaffold & CI Pipeline — Complete
# LKG-002: AgentEvent Types & Core Agent Loop — Complete
# LKG-003: DeepSeek API Client — Complete
# LKG-004: Docker Sandbox Lifecycle — Complete
# LKG-006: Built-in Sandbox Tool Set — Complete
# LKG-007: SQLite Session Persistence — Complete

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

## Package structure

```
cmd/agent/          main.go + cmd/ (root, chat, run)
internal/
  core/             shared domain types (Message, ToolCall, ToolResult, Result)
  agent/            core agent loop + event types (complete)
  llm/              DeepSeek API client (complete)
  sandbox/          Docker sandbox lifecycle (complete)
  tools/            tool registry + 7 built-in sandbox tools (complete)
  store/            SQLite session persistence (complete)
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
