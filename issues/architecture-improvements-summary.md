# Architecture Improvements Summary

## Overview

Implemented four architecture improvement candidates identified by the `improve-codebase-architecture` skill. All changes pass existing tests.

## Changes

### 1. Extract shared domain types into `internal/core`

**Why:** The `llm` package imported domain types (`Message`, `ToolCall`, `Result`) from `internal/agent`. Every LLM provider had to depend on agent-loop concepts. Adding a second provider would have duplicated the leak.

**What:** Created `internal/core/core.go` with `Message`, `ToolCall`, `ToolResult`, and `Result` types. Updated `internal/agent` and `internal/llm` to import from `core` instead of each other. The seam now sits at the module boundary — domain types live in one place, agent loop types stay in agent, wire types stay in llm.

**Files:** `internal/core/core.go` (created), `internal/agent/agent.go` (modified), `internal/agent/agent_test.go` (modified), `internal/llm/client.go` (modified), `internal/llm/client_test.go` (modified)

### 2. Fix ToolResult leak in LLM seam (critical bug fix)

**Why:** `buildRequest` in `internal/llm/client.go` only copied `Role` and `Content` from `agent.Message` to `DeepSeekMessage`. Tool results appended by the agent loop (`Message{Role: "tool", ToolResult: &r}`) were silently dropped — the model never saw tool outputs across turns. This broke multi-turn tool use entirely.

**What:** Added `ToolCallID` and `Content` mapping from `ToolResult` into the `DeepSeekMessage` during request construction. Also added `ToolCalls []DeepSeekToolCall` field to `DeepSeekMessage` for assistant tool call message support.

**Files:** `internal/llm/client.go` (modified), `internal/llm/types.go` (modified)

### 3. Add Sandbox interface

**Why:** The sandbox exported concrete functions (`Start`, `Exec`, `Stop`) with no interface. Consumers were coupled to Docker. No path existed for unit-testing tool execution without Docker installed.

**What:** Defined a `Sandbox` interface in the sandbox package with `Start`, `Exec`, `Stop` methods. The existing functions remain as the concrete implementation.

**Files:** `internal/sandbox/sandbox.go` (modified)

### 4. Consolidate six empty stub modules

**Why:** Six of nine internal packages were single-line stubs with no types, tests, or consumers. They declared speculative boundaries (`router` overlaps with `llm`, `hooks` overlaps with the event system in `agent`) that may not survive contact with real requirements.

**What:** Removed `internal/tools`, `internal/hooks`, `internal/skills`, `internal/store`, `internal/router`, and `internal/tui`. Updated the compilation check in `internal/internal_test.go` to only test the three real modules. Updated `CONTEXT.md` to reflect the new structure.

**Files removed:** `internal/tools/tools.go`, `internal/hooks/hooks.go`, `internal/skills/skills.go`, `internal/store/store.go`, `internal/router/router.go`, `internal/tui/tui.go`
**Files modified:** `internal/internal_test.go`, `CONTEXT.md`

## Verification

All existing tests pass:

- `go vet ./...` — no errors
- `go test ./internal/core/...` — compiles
- `go test ./internal/agent/...` — passes (7 tests)
- `go test ./internal/llm/...` — passes (8 tests)
- `go test ./internal/sandbox/...` — passes (requires Docker)
- `go test ./internal/...` — all pass
- `go build ./cmd/agent` — builds successfully
- `go test ./...` — all tests pass
