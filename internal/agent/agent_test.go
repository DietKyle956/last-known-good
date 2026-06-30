package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
)

type noToolLLM struct {
	response string
}

func (m *noToolLLM) Chat(_ context.Context, messages []core.Message) (<-chan core.Result, error) {
	ch := make(chan core.Result, 2)
	ch <- core.Result{Content: m.response, IsChunk: true}
	ch <- core.Result{Content: m.response, IsChunk: false, Done: true}
	close(ch)
	return ch, nil
}

func TestAgentEmitsTurnCompleteWhenModelReturnsContent(t *testing.T) {
	llm := &noToolLLM{response: "Hello, world!"}
	exec := &spyExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Hi"}})
	<-done
}

type scriptedLLM struct {
	mu       sync.Mutex
	results  [][]core.Result
	callIdx  int
	messages [][]core.Message
}

func (s *scriptedLLM) Chat(_ context.Context, messages []core.Message) (<-chan core.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages = append(s.messages, messages)
	idx := s.callIdx
	s.callIdx++
	ch := make(chan core.Result, len(s.results[idx])+1)
	for _, r := range s.results[idx] {
		ch <- r
	}
	close(ch)
	return ch, nil
}

func TestAgentDispatchesToolCallsAndLoops(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Content: "Done", Done: true},
			},
		},
	}
	exec := &recordingExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "List files"}})
	<-done

	// Verify the executor was called with the tool
	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 tool execution, got %d", len(exec.calls))
	}
	if exec.calls[0].Name != "read_file" {
		t.Fatalf("expected tool name 'read_file', got %q", exec.calls[0].Name)
	}

	// Verify the LLM was called twice
	if len(llm.messages) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(llm.messages))
	}

	// Verify the second call includes the tool result
	lastMsg := llm.messages[1][len(llm.messages[1])-1]
	if lastMsg.Role != "tool" {
		t.Fatalf("expected last message role 'tool', got %q", lastMsg.Role)
	}
	if lastMsg.ToolResult == nil {
		t.Fatal("expected tool result on last message")
	}
}

func TestAgentEmitsToolLifecycleEvents(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "Done", Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Read file"}})
	<-done

	var started, finished bool
	for _, ev := range events {
		if ev.Type == EventToolCallStarted && ev.ToolCall != nil && ev.ToolCall.Name == "read_file" {
			started = true
		}
		if ev.Type == EventToolCallFinished && ev.ToolCall != nil && ev.ToolCall.Name == "read_file" {
			finished = true
		}
	}
	if !started {
		t.Error("expected EventToolCallStarted event, got none")
	}
	if !finished {
		t.Error("expected EventToolCallFinished event, got none")
	}
}

func TestAgentEmitsChunkEventsWithContent(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{Content: "Hel", IsChunk: true},
				{Content: "lo", IsChunk: true},
				{Content: " world", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	var chunks []string
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			if ev.Type == EventModelResponseChunk {
				chunks = append(chunks, ev.Content)
			}
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Say hi"}})
	<-done

	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunk events, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != "Hel" || chunks[1] != "lo" || chunks[2] != " world" {
		t.Fatalf("unexpected chunk contents: %v", chunks)
	}
}

func TestAgentEmitsErrorWhenLLMFails(t *testing.T) {
	errLLM := &errLLM{err: "model failure"}
	exec := &spyExecutor{}
	agent := New(errLLM, exec)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Do something"}})
	<-done

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventError {
		t.Fatalf("expected EventError, got %v", events[0].Type)
	}
	if events[0].Error == nil || events[0].Error.Error() != "model failure" {
		t.Fatalf("unexpected error: %v", events[0].Error)
	}
}

func TestAgentEmitsErrorWhenResultStreamFails(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{Content: "before", IsChunk: true},
				{Err: errors.New("stream error")},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Do something"}})
	<-done

	var errEvent *AgentEvent
	for _, ev := range events {
		if ev.Type == EventError {
			errEvent = &ev
			break
		}
	}
	if errEvent == nil {
		t.Fatal("expected EventError, got none")
	}
	if errEvent.Error == nil || errEvent.Error.Error() != "stream error" {
		t.Fatalf("unexpected error: %v", errEvent.Error)
	}
}

func TestAgentEventOrderForToolSequence(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{
					{ID: "call1", Name: "read_file", Arguments: `{"path":"a.txt"}`},
					{ID: "call2", Name: "read_file", Arguments: `{"path":"b.txt"}`},
				}, Done: true},
			},
			{
				{Content: "Final answer", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Read both"}})
	<-done

	// Filter to lifecycle events (skip chunks for ordering check)
	var lifecycle []AgentEvent
	for _, ev := range events {
		if ev.Type == EventToolCallStarted || ev.Type == EventToolCallFinished || ev.Type == EventTurnComplete {
			lifecycle = append(lifecycle, ev)
		}
	}

	// Should see: started events come before finished events, TurnComplete is last
	if len(lifecycle) != 5 {
		t.Fatalf("expected 5 lifecycle events, got %d: %+v", len(lifecycle), lifecycle)
	}

	if lifecycle[0].Type != EventToolCallStarted || lifecycle[1].Type != EventToolCallStarted {
		t.Fatal("expected two started events first")
	}
	if lifecycle[2].Type != EventToolCallFinished || lifecycle[3].Type != EventToolCallFinished {
		t.Fatal("expected two finished events after started")
	}
	if lifecycle[4].Type != EventTurnComplete {
		t.Fatal("expected TurnComplete last")
	}
}

func TestWriteToolsRunSequentially(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{
					{ID: "c1", Name: "write_tool", Arguments: `{}`},
					{ID: "c2", Name: "write_tool", Arguments: `{}`},
				}, Done: true},
			},
			{
				{Content: "done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &sequentialExecutor{}
	agent := New(llm, exec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "write"}})
	<-done

	exec.mu.Lock()
	defer exec.mu.Unlock()

	if len(exec.started) != 2 {
		t.Fatalf("expected 2 tool starts, got %d", len(exec.started))
	}
	// Second tool should have started after the first finished (sequential)
	if exec.started[1] < exec.finished[0] {
		t.Fatal("write tools did not run sequentially: tool2 started before tool1 finished")
	}
}

type sequentialExecutor struct {
	mu       sync.Mutex
	started  []int64
	finished []int64
}

func (s *sequentialExecutor) Execute(_ context.Context, call core.ToolCall) core.ToolResult {
	start := time.Now().UnixNano()
	time.Sleep(time.Millisecond)
	finish := time.Now().UnixNano()

	s.mu.Lock()
	s.started = append(s.started, start)
	s.finished = append(s.finished, finish)
	s.mu.Unlock()

	return core.ToolResult{ToolCallID: call.ID, Content: "ok"}
}

func (s *sequentialExecutor) IsReadOnly(name string) bool {
	return false
}

func TestReadOnlyToolsRunConcurrently(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{
					{ID: "c1", Name: "read_only_tool", Arguments: `{}`},
					{ID: "c2", Name: "read_only_tool", Arguments: `{}`},
				}, Done: true},
			},
			{
				{Content: "done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &orderingExecutor{readOnly: true, barrier: make(chan struct{})}
	agent := New(llm, exec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "go"}})
	<-done

	exec.mu.Lock()
	defer exec.mu.Unlock()

	if len(exec.started) != 2 {
		t.Fatalf("expected 2 tool starts, got %d", len(exec.started))
	}
	// Both tools should have started before either finished (parallelism)
	if exec.started[1] >= exec.finished[0] {
		t.Fatal("read-only tools did not run concurrently: tool2 started after tool1 finished")
	}
}

// orderingExecutor synchronises so that both tools must start before either finishes.
type orderingExecutor struct {
	mu       sync.Mutex
	started  []int64
	finished []int64
	barrier  chan struct{}
	readOnly bool
	callIdx  int
}

func (o *orderingExecutor) Execute(_ context.Context, call core.ToolCall) core.ToolResult {
	o.mu.Lock()
	idx := o.callIdx
	o.callIdx++
	o.started = append(o.started, time.Now().UnixNano())
	if idx == 0 {
		o.mu.Unlock()
		// First tool waits for second to start
		<-o.barrier
	} else {
		o.mu.Unlock()
		// Second tool signals first to proceed
		close(o.barrier)
	}

	o.mu.Lock()
	o.finished = append(o.finished, time.Now().UnixNano())
	o.mu.Unlock()

	return core.ToolResult{ToolCallID: call.ID, Content: "ok"}
}

func (o *orderingExecutor) IsReadOnly(name string) bool {
	return o.readOnly
}

type errLLM struct {
	err string
}

func (e *errLLM) Chat(_ context.Context, messages []core.Message) (<-chan core.Result, error) {
	return nil, errors.New(e.err)
}

type recordingExecutor struct {
	mu    sync.Mutex
	calls []core.ToolCall
}

func (r *recordingExecutor) Execute(_ context.Context, call core.ToolCall) core.ToolResult {
	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()
	return core.ToolResult{ToolCallID: call.ID, Content: "file contents"}
}

func (r *recordingExecutor) IsReadOnly(name string) bool {
	return true
}

type spyExecutor struct{}

func (s *spyExecutor) Execute(_ context.Context, call core.ToolCall) core.ToolResult {
	return core.ToolResult{}
}

func (s *spyExecutor) IsReadOnly(name string) bool {
	return true
}

// slowExecutor delays execution to trigger timeouts.
type slowExecutor struct {
	delay time.Duration
}

func (s *slowExecutor) Execute(ctx context.Context, call core.ToolCall) core.ToolResult {
	select {
	case <-time.After(s.delay):
		return core.ToolResult{ToolCallID: call.ID, Content: "ok"}
	case <-ctx.Done():
		return core.ToolResult{ToolCallID: call.ID, Content: ctx.Err().Error(), IsError: true}
	}
}

func (s *slowExecutor) IsReadOnly(name string) bool {
	return true
}

func TestAgentContextCancellation(t *testing.T) {
	blockingLLM := &blockingLLM{block: make(chan struct{})}
	exec := &spyExecutor{}
	a := New(blockingLLM, exec)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		for ev := range a.Events() {
			if ev.Type == EventError {
				return
			}
		}
	}()

	go a.Run(ctx, []core.Message{{Role: "user", Content: "Go"}})
	cancel()
	<-done
}

type blockingLLM struct {
	block chan struct{}
}

func (b *blockingLLM) Chat(ctx context.Context, messages []core.Message) (<-chan core.Result, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestAgentFiresModelCallHooks(t *testing.T) {
	hookSys := hooks.New(nil)
	llm := &noToolLLM{response: "hello"}
	exec := &spyExecutor{}
	agent := New(llm, exec)
	agent.SetHooks(hookSys)

	var before, after bool
	hookSys.Register(hooks.BeforeModelCall, func(hooks.HookEvent) *hooks.HookResult {
		before = true
		return nil
	})
	hookSys.Register(hooks.AfterModelCall, func(hooks.HookEvent) *hooks.HookResult {
		after = true
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Hi"}})
	<-done

	if !before {
		t.Error("expected BeforeModelCall hook to fire")
	}
	if !after {
		t.Error("expected AfterModelCall hook to fire")
	}
}

func TestAgentFiresToolCallHooks(t *testing.T) {
	hookSys := hooks.New(nil)
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &recordingExecutor{}
	agent := New(llm, exec)
	agent.SetHooks(hookSys)

	var before, after bool
	hookSys.Register(hooks.BeforeToolCall, func(hooks.HookEvent) *hooks.HookResult {
		before = true
		return nil
	})
	hookSys.Register(hooks.AfterToolCall, func(hooks.HookEvent) *hooks.HookResult {
		after = true
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Read file"}})
	<-done

	if !before {
		t.Error("expected BeforeToolCall hook to fire")
	}
	if !after {
		t.Error("expected AfterToolCall hook to fire")
	}
}

func TestAgentBlockingHookPreventsToolExecution(t *testing.T) {
	hookSys := hooks.New(nil)
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &recordingExecutor{}
	agent := New(llm, exec)
	agent.SetHooks(hookSys)

	hookSys.Register(hooks.BeforeToolCall, func(hooks.HookEvent) *hooks.HookResult {
		return &hooks.HookResult{Block: true}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Read file"}})
	<-done

	if len(exec.calls) != 0 {
		t.Fatalf("expected 0 tool executions (blocked by hook), got %d", len(exec.calls))
	}
}

func TestAgentDangerousCommandHookBlocksBashCommand(t *testing.T) {
	hookSys := hooks.New(nil)
	dangerous := hooks.NewDangerousCommandHook(nil)
	hookSys.Register(hooks.BeforeToolCall, dangerous.Handler)

	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "bash", Arguments: `{"command":"rm -rf /"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &recordingExecutor{}
	agent := New(llm, exec)
	agent.SetHooks(hookSys)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Run dangerous command"}})
	<-done

	if len(exec.calls) != 0 {
		t.Fatalf("expected 0 tool executions (blocked by dangerous hook), got %d", len(exec.calls))
	}

	// The blocked tool does not emit lifecycle events, but the model should
	// receive the blocked result as the last tool message.
	if len(llm.messages) < 2 {
		t.Fatalf("expected at least 2 LLM calls, got %d", len(llm.messages))
	}
	lastCallMessages := llm.messages[len(llm.messages)-1]
	lastMsg := lastCallMessages[len(lastCallMessages)-1]
	lastContent := lastMsg.ToolResult
	if lastContent == nil {
		t.Fatal("expected last message to have a ToolResult")
	}
	if !lastContent.IsError {
		t.Fatal("expected ToolResult.IsError to be true for a blocked command")
	}
	if lastContent.Content == "" || lastContent.Content == "blocked by hook" {
		t.Fatalf("expected a descriptive block reason, got %q", lastContent.Content)
	}
}

func TestAgentMaxToolCallsStopsAtLimit(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{ToolCalls: []core.ToolCall{{ID: "call2", Name: "read_file", Arguments: `{"path":"y.txt"}`}}, Done: true},
			},
			{
				{ToolCalls: []core.ToolCall{{ID: "call3", Name: "read_file", Arguments: `{"path":"z.txt"}`}}, Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)
	agent.SetMaxToolCalls(2)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Loop"}})
	<-done

	// Should have emitted an error due to iteration limit.
	var errEvent *AgentEvent
	for _, ev := range events {
		if ev.Type == EventError {
			errEvent = &ev
			break
		}
	}
	if errEvent == nil {
		t.Fatal("expected EventError for hitting iteration limit, got none")
	}
	if errEvent.Error == nil || errEvent.Error.Error() != "tool call iteration limit reached (2)" {
		t.Fatalf("unexpected error: %v", errEvent.Error)
	}

	// Should NOT have a TurnComplete event.
	for _, ev := range events {
		if ev.Type == EventTurnComplete {
			t.Fatal("expected no TurnComplete after iteration limit hit")
		}
	}

	// Agent should have called LLM 2 times (1st invocation returned tool calls,
	// 2nd invocation also returns tool calls, then limit hit before appending results).
	if len(llm.messages) != 2 {
		t.Fatalf("expected 2 LLM calls, got %d", len(llm.messages))
	}
}

func TestAgentMaxToolCallsZeroIsUnlimited(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"a.txt"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)
	agent.SetMaxToolCalls(0) // explicitly zero = unlimited

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Loop"}})
	<-done

	var errEvent bool
	for _, ev := range events {
		if ev.Type == EventError {
			errEvent = true
			break
		}
	}
	if errEvent {
		t.Fatal("expected no EventError when MaxToolCalls=0")
	}
}

func TestAgentToolTimeoutReturnsErrorResult(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &slowExecutor{delay: 5 * time.Second}
	agent := New(llm, exec)
	agent.SetToolTimeout(10 * time.Millisecond)

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Timeout test"}})
	<-done

	// Verify the tool call finished event has an error result (from context deadline).
	var toolFinished *AgentEvent
	for _, ev := range events {
		if ev.Type == EventToolCallFinished && ev.ToolCall != nil && ev.ToolCall.ID == "call1" {
			toolFinished = &ev
			break
		}
	}
	if toolFinished == nil {
		t.Fatal("expected EventToolCallFinished for the timed-out tool")
	}
	if toolFinished.ToolResult == nil {
		t.Fatal("expected ToolResult on finished event")
	}
	if !toolFinished.ToolResult.IsError {
		t.Fatal("expected IsError=true for timed-out tool")
	}
}

func TestAgentToolTimeoutZeroIsNoTimeout(t *testing.T) {
	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}, Done: true},
			},
			{
				{Content: "done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &spyExecutor{}
	agent := New(llm, exec)
	agent.SetToolTimeout(0) // zero = no timeout

	done := make(chan struct{})
	var events []AgentEvent
	go func() {
		defer close(done)
		for ev := range agent.Events() {
			events = append(events, ev)
			if ev.Type == EventTurnComplete {
				return
			}
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "No timeout"}})
	<-done

	for _, ev := range events {
		if ev.Type == EventToolCallFinished && ev.ToolCall != nil {
			if ev.ToolResult != nil && ev.ToolResult.IsError {
				t.Fatal("expected no error on tool result when timeout is 0")
			}
		}
	}
}

func TestAgentDangerousCommandHookAllowsSafeBashCommand(t *testing.T) {
	hookSys := hooks.New(nil)
	dangerous := hooks.NewDangerousCommandHook(nil)
	hookSys.Register(hooks.BeforeToolCall, dangerous.Handler)

	llm := &scriptedLLM{
		results: [][]core.Result{
			{
				{ToolCalls: []core.ToolCall{{ID: "call1", Name: "bash", Arguments: `{"command":"ls -la"}`}}, Done: true},
			},
			{
				{Content: "Done", IsChunk: true},
				{Done: true},
			},
		},
	}
	exec := &recordingExecutor{}
	agent := New(llm, exec)
	agent.SetHooks(hookSys)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for range agent.Events() {
		}
	}()

	agent.Run(context.Background(), []core.Message{{Role: "user", Content: "Run safe command"}})
	<-done

	if len(exec.calls) != 1 {
		t.Fatalf("expected 1 tool execution (safe command not blocked), got %d", len(exec.calls))
	}
	if exec.calls[0].Name != "bash" {
		t.Fatalf("expected bash tool call, got %q", exec.calls[0].Name)
	}
}
