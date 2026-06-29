# LKG-001: Project Scaffold & CI Pipeline — Complete
# LKG-002: AgentEvent Types & Core Agent Loop — Complete
# LKG-003: DeepSeek API Client — Complete
# LKG-004: Docker Sandbox Lifecycle — Complete

## Summary

Initialized Last Known Good as a Go binary with Cobra CLI, internal package skeletons,
Docker Compose dev environment, and GitHub Actions CI.

## What was built (LKG-001)

- **Go module**: `github.com/DietKyle956/last-known-good` (Go 1.26)
- **Entrypoint**: `cmd/agent/main.go` with Cobra root command
- **Subcommands**: `agent chat` and `agent run` (stubs)
- **Internal packages**: `agent`, `llm`, `sandbox`, `tools`, `hooks`, `skills`, `store`, `router`, `tui`
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

## Package structure

```
cmd/agent/          main.go + cmd/ (root, chat, run)
internal/
  agent/            core agent loop + event types (complete)
  llm/              DeepSeek API client (complete)
  sandbox/          Docker sandbox lifecycle (complete)
  tools/            tool interface & registry (stub)
  hooks/            hooks framework (stub)
  skills/           skills system (stub)
  store/            persistence layer (stub)
  router/           model router (stub)
  tui/              terminal UI (stub)
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
