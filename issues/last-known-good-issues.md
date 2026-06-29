# Last Known Good — Engineering Issues

_22 Tracer-Bullet Vertical Slices • V1.0_

**HITL** = Human In The Loop  **AFK** = Away From Keyboard

| **Issue** | **Title** | **Type** | **Blocked By** |
|---|---|---|---|
| **LKG-001** | Project Scaffold & CI Pipeline | **AFK** | None |
| **LKG-002** | AgentEvent Types & Core Agent Loop | **AFK** | LKG-001 |
| **LKG-003** | DeepSeek API Client | **AFK** | LKG-001 |
| **LKG-004** | Docker Sandbox Lifecycle | **AFK** | LKG-001 |
| **LKG-005** | Tool Interface, Registry & Dispatch | **AFK** | LKG-002, LKG-004 |
| **LKG-006** | Built-in Sandbox Tool Set | **AFK** | LKG-005 |
| **LKG-007** | SQLite Session Persistence | **AFK** | LKG-001 |
| **LKG-008** | Session Resume | **AFK** | LKG-007 |
| **LKG-009** | Full TUI Shell | **HITL** | LKG-002, LKG-006 |
| **LKG-010** | Sandbox Network Policy & Resource Limits | **AFK** | LKG-004 |
| **LKG-011** | Single-Shot CLI Mode | **AFK** | LKG-002, LKG-006 |
| **LKG-012** | Heuristic Model Router & Thinking Mode | **AFK** | LKG-003, LKG-006 |
| **LKG-013** | Hooks Framework | **AFK** | LKG-002, LKG-005 |
| **LKG-014** | Blocking Hook for Dangerous Commands | **AFK** | LKG-013, LKG-006 |
| **LKG-015** | Auto-Format Hook on File Write | **AFK** | LKG-013, LKG-006 |
| **LKG-016** | Skills System & Lazy Skill Loading | **AFK** | LKG-005, LKG-003 |
| **LKG-017** | Structured JSONL Logging | **AFK** | LKG-007, LKG-002 |
| **LKG-018** | CLI Session & Log Commands | **AFK** | LKG-008, LKG-017 |
| **LKG-019** | Prompt-Cache-Friendly Request Shaping | **AFK** | LKG-003, LKG-005 |
| **LKG-020** | Strict JSON Schema Tool Mode | **AFK** | LKG-003, LKG-005 |
| **LKG-021** | Per-Tool Timeout & Max-Iteration Guard | **AFK** | LKG-002, LKG-004 |
| **LKG-022** | Status Line & Usage Telemetry | **HITL** | LKG-009, LKG-003, LKG-007 |

---

## LKG-001 Project Scaffold & CI Pipeline

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | None — can start immediately |
| -------------- | ---------------------------- |

| **User stories** | None directly — foundational enabler for all slices |
| ---------------- | --------------------------------------------------- |

**What to Build**

Initialize Last Known Good as a single Go binary with a Cobra CLI entrypoint, internal package skeletons for the full PRD architecture, a Docker Compose development environment, and GitHub Actions CI for build, test, and lint workflows.

**Acceptance Criteria**

- The repository contains a top-level `cmd/agent` entrypoint and internal packages for `agent`, `llm`, `sandbox`, `tools`, `hooks`, `skills`, `store`, `router`, and `tui`.
- Running the build command produces a single binary with no errors on a clean checkout.
- Running all tests against the scaffold produces a passing result with no test failures.
- The `agent` binary can be invoked from the command line and prints help output without panicking.
- The `agent chat` subcommand is recognized as a valid command even if not yet implemented.
- The `agent run` subcommand is recognized as a valid command even if not yet implemented.
- A Docker Compose file starts a development environment in a single command without errors.
- The CI pipeline runs on every push to the main branch.
- The CI pipeline runs on every pull request.
- The CI pipeline fails if the build does not compile cleanly.
- The CI pipeline fails if any test fails.
- The CI pipeline fails if the linter reports any violation.
- A domain package cannot import from an infrastructure or adapter package — this constraint is enforced automatically by the CI lint step.

---

## LKG-002 AgentEvent Types & Core Agent Loop

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-001 |
| -------------- | ------- |

| **User stories** | US-1, US-25, US-29 |
| ---------------- | ------------------ |

**What to Build**

Implement the typed event stream and the core agent loop that builds messages, calls the model, dispatches tool calls, appends tool results, and loops until a final content turn completes. This event stream is the single instrumentation boundary consumed by the TUI, the plain-stdout renderer, and the hooks system.

**Acceptance Criteria**

- When the model returns tool calls, the agent dispatches each tool and appends the results as tool-role messages before making another model call.
- When the model returns content with no tool calls, the agent stops looping and emits a turn-complete signal.
- The agent emits a signal when a model response chunk is received.
- The agent emits a signal when a tool call begins execution.
- The agent emits a signal when a tool call finishes execution.
- The agent emits a signal when a full turn is complete.
- The agent emits a signal when an unrecoverable error occurs.
- Given a scripted sequence of two tool calls followed by final content, the agent emits events in the correct order.
- Read-only tool calls that have no dependency on each other can execute at the same time.
- Tool calls that write or execute side effects run one at a time, never concurrently.
- The event channel is the only way external code receives information about what the agent did — there is no separate out-of-band callback.

---

## LKG-003 DeepSeek API Client

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-001 |
| -------------- | ------- |

| **User stories** | US-5, US-6, US-7, US-8, US-11 |
| ---------------- | ----------------------------- |

**What to Build**

Create the HTTP client for DeepSeek's OpenAI-compatible chat completions endpoint using project-owned request and response structs — no third-party SDK. Support V4-Pro, V4-Flash, streaming, thinking mode, reasoning effort, and tool-call parsing.

**Acceptance Criteria**

- The client sends requests to the DeepSeek API endpoint using only structs defined in this project — no external SDK types in the request or response path.
- The client can target `deepseek-v4-pro` as the model for a request.
- The client can target `deepseek-v4-flash` as the model for a request.
- The client can enable thinking mode on a request.
- The client can set a reasoning effort level on a request.
- The client can request a streaming response and yield chunks as they arrive.
- The client can request a non-streaming response and return the complete result.
- The client correctly parses tool calls out of a model response.
- The client correctly parses final natural-language content out of a model response.
- Given a mocked endpoint that returns a malformed response, the client returns an error rather than panicking.
- Given a mocked endpoint that returns a tool-call response, the parsed result contains the correct tool name and arguments.
- The request payload shape matches what a real DeepSeek endpoint would accept, verified against a recorded fixture.

---

## LKG-004 Docker Sandbox Lifecycle

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-001 |
| -------------- | ------- |

| **User stories** | US-2, US-3, US-30 |
| ---------------- | ----------------- |

**What to Build**

Implement the Docker-backed sandbox that creates one container per session, bind-mounts the project directory, keeps the container alive for the entire session, and tears it down on exit or interrupt.

**Acceptance Criteria**

- Starting a session creates exactly one sandbox container.
- The same container is reused for every tool call within a session — no new container is created between calls.
- A file written to the project directory on the host is visible inside the container at the mount point.
- A file written inside the container at the mount point is visible on the host in the project directory.
- Files outside the mounted project directory are not accessible from inside the container.
- When a session ends normally, the container is removed.
- When a session is interrupted with a signal, the container is removed.
- After a session ends, no orphaned containers from that session remain running.
- Tool implementations receive only a sandbox handle — they cannot reference a host filesystem path directly.
- Tests for the sandbox lifecycle run against a real Docker daemon, not a mock.

---

## LKG-005 Tool Interface, Registry & Dispatch

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-002, LKG-004 |
| -------------- | ---------------- |

| **User stories** | US-10, US-11 |
| ---------------- | ------------ |

**What to Build**

Build the tool interface contract, the result type, and the registry that serializes tool definitions for the model and dispatches incoming tool calls to their implementations through the sandbox.

**Acceptance Criteria**

- Any tool implementation must provide a name, a description, a JSON schema, and an execute function that receives a sandbox handle and returns a result.
- A tool result carries content to return to the model, an error flag, and a metadata map that is never sent to the model.
- The registry can produce a list of tool definitions suitable for inclusion in an API request payload.
- The registry dispatches a tool call by name to the matching implementation.
- Dispatching a tool call for an unknown tool name returns an error without panicking.
- Dispatching a tool call with arguments that fail schema validation returns an error without executing the tool.
- A new tool can be added to the registry without modifying the dispatch logic.
- The registry is the only place in the codebase that maps tool names to implementations.

---

## LKG-006 Built-in Sandbox Tool Set

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-005 |
| -------------- | ------- |

| **User stories** | US-9 |
| ---------------- | ---- |

**What to Build**

Implement all seven built-in tools — `read_file`, `write_file`, `edit_file`, `bash`, `grep`, `glob`, and `git_diff` — executing entirely through sandbox APIs.

**Acceptance Criteria**

- Reading a file that exists inside the sandbox returns its contents.
- Reading a file that does not exist returns an error result, not a panic.
- Writing a file creates it inside the sandbox at the specified path.
- Writing a file that already exists overwrites its contents.
- Editing a file with a find-and-replace operation changes only the matched text and leaves the rest of the file intact.
- Running a shell command inside the sandbox returns its stdout output.
- Running a shell command that exits with a non-zero status returns an error result containing the stderr output.
- Grepping for a pattern that matches returns the matching lines with their file paths and line numbers.
- Grepping for a pattern with no matches returns an empty result, not an error.
- Globbing a pattern that matches files returns the list of matching paths.
- Globbing a pattern with no matches returns an empty list, not an error.
- Running a git diff on a repository with unstaged changes returns the diff output.
- Running a git diff on a repository with no changes returns an empty result, not an error.
- None of the seven tools can execute code directly on the host — all execution goes through the sandbox handle.
- Each tool's schema correctly describes its required and optional arguments.
- Tests for each tool run against a real sandboxed container, not mocked filesystem or shell calls.

---

## LKG-007 SQLite Session Persistence

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-001 |
| -------------- | ------- |

| **User stories** | US-19 |
| ---------------- | ----- |

**What to Build**

Create the SQLite persistence layer for sessions, messages, tool calls, and hook events so every run has durable, queryable history without requiring a separate database process.

**Acceptance Criteria**

- Starting a new session creates a session record in the database.
- Each message in a session is saved to the database with its role, content, and the model that produced it.
- Each tool call is saved with the tool name, input arguments, result content, error status, and how long it took to execute.
- Each hook event is saved with the event type and its payload.
- All records for a session are associated with that session's identifier.
- After the process exits and restarts, previously saved sessions are still present in the database.
- Reading back a saved session's messages returns them in the same order they were written.
- The database schema applies cleanly to a brand-new empty database file.
- No external database daemon is required for the persistence layer to work.
- Tests run against a real temporary SQLite file, not an in-memory mock.

---

## LKG-008 Session Resume

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-007 |
| -------------- | ------- |

| **User stories** | US-20 |
| ---------------- | ----- |

**What to Build**

Implement session resume so a prior session can be reloaded from SQLite, the message history reconstructed into live agent state, and the conversation continued without re-explaining context.

**Acceptance Criteria**

- Given a session ID, the agent loads all messages for that session from the database.
- The loaded messages are presented to the model as prior conversation history on the next turn.
- A new prompt sent after resuming a session extends the existing message history rather than starting a new one.
- Resuming a session that does not exist returns a clear error message rather than silently starting a fresh session.
- The in-memory state after a resume is equivalent to what would exist if the session had never been interrupted.
- The resumed session's new messages are saved to the same session record in the database.

---

## LKG-009 Full TUI Shell

| **Type** | **HITL** |
| -------- | -------- |

| **Blocked by** | LKG-002, LKG-006 |
| -------------- | ---------------- |

| **User stories** | US-23, US-24, US-25 |
| ---------------- | ------------------- |

**What to Build**

Build the default interactive terminal UI using Bubble Tea with a single scrolling conversation viewport, inline collapsible tool-call blocks, a fixed input bar at the bottom, and live streaming model output.

**Acceptance Criteria**

- Launching the agent interactively opens a terminal UI without errors.
- The conversation history scrolls in a single viewport — there are no side panels or split panes.
- The input bar is always visible at the bottom of the terminal, even while a response is streaming.
- Model response text appears in the viewport as it streams — the user does not wait for the full response before seeing output.
- A tool call that occurred during a turn is rendered inline in the conversation at the point it happened.
- A tool call block is collapsed by default, showing only a one-line summary.
- A tool call block can be expanded to show its full detail.
- A tool call that resulted in an error is shown expanded by default with visually distinct styling.
- Submitting a new prompt from the input bar sends it to the agent loop without leaving the TUI.
- The TUI receives events from the agent loop through the shared event channel — it does not call the agent directly.
- A human review of the running TUI approves the single-viewport layout, tool-block readability, and streaming behavior before this issue is closed.

---

## LKG-010 Sandbox Network Policy & Resource Limits

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-004 |
| -------------- | ------- |

| **User stories** | US-4 |
| ---------------- | ---- |

**What to Build**

Add network isolation and resource constraints to the sandbox so outbound network access is off by default, can be selectively enabled per project, and containers are bounded by CPU and memory limits.

**Acceptance Criteria**

- A sandbox started without any network configuration cannot make outbound network calls.
- A sandbox started with a specific domain on the allowlist can reach that domain.
- A sandbox started with a specific domain on the allowlist cannot reach domains not on that list.
- The network policy is read from per-project configuration rather than a global flag.
- A sandbox has a CPU usage limit applied at container start time.
- A sandbox has a memory usage limit applied at container start time.
- Tests for network behavior run against a real Docker container, not a mock.
- The network-disabled default is in effect when no project configuration file is present.

---

## LKG-011 Single-Shot CLI Mode

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-002, LKG-006 |
| -------------- | ---------------- |

| **User stories** | US-27, US-28, US-29 |
| ---------------- | ------------------- |

**What to Build**

Implement `agent run "<prompt>"` as a non-interactive execution path that runs one task, prints results to stdout, exits with a meaningful status code, and optionally emits structured JSON — all consuming the same underlying event stream as the TUI.

**Acceptance Criteria**

- Running `agent run` with a prompt executes the task and exits without opening the TUI.
- When the task completes successfully, the process exits with status code zero.
- When the task fails, the process exits with a non-zero status code.
- Output is written to stdout so it can be captured by shell pipelines and scripts.
- Running `agent run` with the JSON flag produces structured output that a script can parse.
- Running `agent run` without the JSON flag produces human-readable plain text output.
- The single-shot renderer subscribes to the same agent event channel as the TUI renderer — there is no separate code path for driving the agent in non-interactive mode.
- The process exits on its own after the task completes — it does not wait for further input.

---

## LKG-012 Heuristic Model Router & Thinking Mode

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-003, LKG-006 |
| -------------- | ---------------- |

| **User stories** | US-5, US-6, US-7 |
| ---------------- | ---------------- |

**What to Build**

Add the pluggable model router that selects between V4-Flash and V4-Pro and decides whether to enable thinking mode, based on task complexity signals and prior turn outcomes.

**Acceptance Criteria**

- A turn that touches a single file is routed to V4-Flash.
- A turn that touches more files than the configured threshold is routed to V4-Pro.
- A turn that follows a failed tool or test call is routed to V4-Pro.
- A turn whose prompt contains a recognized complexity signal word is routed to V4-Pro.
- A Pro turn always has thinking mode enabled.
- A Flash turn has thinking mode disabled by default.
- A Flash turn that is a retry after a failure has thinking mode enabled.
- The routing logic can be replaced without changing any code in the agent loop.
- Each routing scenario — single-file, multi-file, post-failure, complexity signal — is covered by an independent test case.

---

## LKG-013 Hooks Framework

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-002, LKG-005 |
| -------------- | ---------------- |

| **User stories** | US-12, US-13, US-15 |
| ---------------- | ------------------- |

**What to Build**

Implement the typed hooks system that fires around session lifecycle, model calls, and tool calls — allowing observers to log, react, or block actions without modifying the agent loop.

**Acceptance Criteria**

- A hook can be registered to fire when a session starts.
- A hook can be registered to fire when a session ends.
- A hook can be registered to fire before a model call is made.
- A hook can be registered to fire after a model call completes.
- A hook can be registered to fire before a tool call is dispatched.
- A hook can be registered to fire after a tool call completes.
- A hook registered on the before-tool-call event can prevent the tool from executing.
- When a before-tool-call hook blocks a tool, the tool's execute function is never called.
- A hook registered on the after-tool-call event cannot block the tool — it already ran.
- Multiple hooks registered for the same event all fire in registration order.
- Hooks are registered as compiled Go code at startup — no plugin files or dynamic loading.
- The hooks system receives events from the same agent event channel as the TUI and plain renderer.

---

## LKG-014 Blocking Hook for Dangerous Commands

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-013, LKG-006 |
| -------------- | ---------------- |

| **User stories** | US-13 |
| ---------------- | ----- |

**What to Build**

Create the first concrete safety hook: a before-tool-call handler that inspects shell commands for dangerous patterns and vetoes them before they reach the sandbox.

**Acceptance Criteria**

- A shell command matching a known dangerous pattern is blocked before it executes in the sandbox.
- When a command is blocked, a clear reason is recorded explaining why it was denied.
- When a command is blocked, the agent loop receives a structured failure result rather than crashing.
- The blocked command is recorded in the session's hook event history.
- A shell command that does not match any dangerous pattern is not blocked.
- The list of dangerous patterns can be modified without changing the core hook dispatch code.
- Tests verify that dangerous commands never reach the sandbox's execute path.

---

## LKG-015 Auto-Format Hook on File Write

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-013, LKG-006 |
| -------------- | ---------------- |

| **User stories** | US-14 |
| ---------------- | ----- |

**What to Build**

Implement the auto-format hook that runs a language formatter inside the sandbox after a file write to a recognized source file type.

**Acceptance Criteria**

- After a file with a recognized extension is written, the formatter runs inside the sandbox automatically.
- After a file with an unrecognized extension is written, no formatter is invoked.
- The formatter runs inside the sandbox — it does not run on the host.
- A formatting failure is recorded in the session history as a hook event.
- A formatting failure does not cause the agent loop to crash or halt.
- The formatted file's contents reflect the formatter's output after the hook completes.
- The mapping of file extensions to formatters can be configured without changing the hook dispatch code.

---

## LKG-016 Skills System & Lazy Skill Loading

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-005, LKG-003 |
| -------------- | ---------------- |

| **User stories** | US-16, US-17, US-18 |
| ---------------- | ------------------- |

**What to Build**

Implement the file-based skills system that loads only skill names and descriptions into the system prompt at session start, with the full skill body fetched on demand via a `read_skill` tool call.

**Acceptance Criteria**

- At session start, only the name and description of each skill are included in the system prompt — not the full body.
- The full body of a skill is not loaded until the read-skill tool is invoked for that skill.
- Skills are discovered from a directory where each skill lives in its own folder containing a markdown file with frontmatter.
- A skill frontmatter block must contain at least a name and a description to be recognized.
- A skill file with missing or malformed frontmatter is skipped without crashing the loader.
- A skill file with valid frontmatter but no body is loaded without error.
- Invoking the read-skill tool with a valid skill name returns that skill's full markdown body.
- Invoking the read-skill tool with an unknown skill name returns an error result.
- Adding a new skill directory makes that skill available at the next session start without modifying any code.

---

## LKG-017 Structured JSONL Logging

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-007, LKG-002 |
| -------------- | ---------------- |

| **User stories** | US-21 |
| ---------------- | ----- |

**What to Build**

Add per-session JSONL log files written in parallel to SQLite so sessions can be tailed live or piped into external tooling without querying the database.

**Acceptance Criteria**

- When a session starts, a dedicated log file is created for that session.
- Each log entry is a valid JSON object on its own line.
- Each log entry identifies which session it belongs to.
- Model call events are written to the log as they occur.
- Tool call events — both start and finish — are written to the log as they occur.
- Hook events are written to the log as they occur.
- The log file can be read by a separate process while the session is still running.
- After the session ends, the log file remains on disk and is not deleted.
- Each session's log file is separate from other sessions' log files.

---

## LKG-018 CLI Session & Log Commands

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-008, LKG-017 |
| -------------- | ---------------- |

| **User stories** | US-22 |
| ---------------- | ----- |

**What to Build**

Implement the CLI commands for browsing past sessions, resuming a session by ID, and viewing or tailing a session's JSONL log output.

**Acceptance Criteria**

- Running the sessions list command prints a list of past sessions to stdout.
- Each row in the sessions list includes the session ID and when it was created.
- Running the sessions list command when no sessions exist prints an empty list rather than an error.
- Running the sessions resume command with a valid ID loads that session and enters the agent loop.
- Running the sessions resume command with an invalid ID prints a clear error and exits.
- Running the logs command with a valid session ID prints that session's JSONL log to stdout.
- Running the logs command with an invalid session ID prints a clear error and exits.
- The logs command supports a follow mode that streams new log lines as they are written.
- These commands do not duplicate persistence logic — they read from the same store and log files used during normal execution.

---

## LKG-019 Prompt-Cache-Friendly Request Shaping

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-003, LKG-005 |
| -------------- | ---------------- |

| **User stories** | US-8 |
| ---------------- | ---- |

**What to Build**

Stabilize the system prompt and tool definition payload so they are byte-identical across all calls within a session, preserving DeepSeek prompt-cache eligibility and avoiding unnecessary token costs.

**Acceptance Criteria**

- The system prompt sent on the first call of a session is byte-for-byte identical to the system prompt sent on subsequent calls in the same session.
- The tool definitions payload sent on the first call of a session is byte-for-byte identical to the payload sent on subsequent calls in the same session.
- No timestamp, counter, or other value that changes per call is embedded in the system prompt or tool definitions.
- Tool definitions are serialized in the same order on every call — the order does not depend on map iteration or other non-deterministic operations.
- A test compares the raw bytes of the system prompt and tool definitions across two consecutive calls and asserts they are equal.

---

## LKG-020 Strict JSON Schema Tool Mode

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-003, LKG-005 |
| -------------- | ---------------- |

| **User stories** | US-11 |
| ---------------- | ----- |

**What to Build**

Add support for DeepSeek's strict schema mode on tool definitions where the schema fits the supported subset, with safe fallback to non-strict mode for schemas that do not qualify.

**Acceptance Criteria**

- A tool whose schema uses only the supported strict-mode types is sent to the API with strict mode enabled.
- A tool whose schema contains a type not supported in strict mode is sent without strict mode rather than causing an error.
- A tool call returned by the model in strict mode with valid arguments dispatches successfully.
- A tool call returned by the model with arguments that do not match the schema returns an error result rather than passing the bad arguments to the tool.
- Strict-mode and non-strict-mode tools can coexist in the same registry and be sent in the same request.
- Adding strict mode to a tool does not change how its result is returned to the model.

---

## LKG-021 Per-Tool Timeout & Max-Iteration Guard

| **Type** | **AFK** |
| -------- | ------- |

| **Blocked by** | LKG-002, LKG-004 |
| -------------- | ---------------- |

| **User stories** | US-31, US-32 |
| ---------------- | ------------ |

**What to Build**

Implement the defensive execution limits that prevent a hung tool call or a runaway tool-call loop from stalling the agent indefinitely.

**Acceptance Criteria**

- A tool call that does not complete within the configured wall-clock timeout is terminated.
- When a tool call times out, the agent receives a structured error result rather than hanging.
- The timeout duration is configurable — it is not hardcoded.
- A turn that exceeds the configured maximum number of tool calls stops and returns an error rather than looping further.
- The maximum iteration limit is configurable — it is not hardcoded.
- When the iteration limit is hit, the agent emits an error event rather than panicking.
- A test drives a shell command that sleeps indefinitely and verifies it is cancelled after the timeout.
- A test drives a scripted model that always returns a tool call and verifies the loop terminates at the iteration cap.

---

## LKG-022 Status Line & Usage Telemetry

| **Type** | **HITL** |
| -------- | -------- |

| **Blocked by** | LKG-009, LKG-003, LKG-007 |
| -------------- | ------------------------- |

| **User stories** | US-26 |
| ---------------- | ----- |

**What to Build**

Add the minimal TUI status line showing the active model, sandbox state, and running token or cost usage, and persist per-message model and usage data to the session store.

**Acceptance Criteria**

- The TUI displays the name of the currently active model at all times.
- The TUI displays the current sandbox state — for example, whether the container is running or stopped.
- The TUI displays the cumulative token usage for the session.
- The token count updates after each model response — it does not stay static for the whole session.
- The status line is always visible and does not scroll away with the conversation.
- The model used to produce each message is saved to the database alongside that message.
- After a session ends, the persisted per-message model data is readable from the database.
- A human review of the running TUI confirms the status line remains minimal and does not disrupt the single-viewport layout before this issue is closed.
