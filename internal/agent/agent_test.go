package agent

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DietKyle956/last-known-good/internal/core"
)

type noToolLLM struct {
	response string
}

func (m *noToolLLM) Chat(messages []core.Message) (<-chan core.Result, error) {
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

	agent.Run([]core.Message{{Role: "user", Content: "Hi"}})
	<-done
}

type scriptedLLM struct {
	mu       sync.Mutex
	results  [][]core.Result
	callIdx  int
	messages [][]core.Message
}

func (s *scriptedLLM) Chat(messages []core.Message) (<-chan core.Result, error) {
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

	agent.Run([]core.Message{{Role: "user", Content: "List files"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "Read file"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "Say hi"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "Do something"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "Do something"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "Read both"}})
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

	agent.Run([]core.Message{{Role: "user", Content: "write"}})
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

func (s *sequentialExecutor) Execute(call core.ToolCall) core.ToolResult {
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

	agent.Run([]core.Message{{Role: "user", Content: "go"}})
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

func (o *orderingExecutor) Execute(call core.ToolCall) core.ToolResult {
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

func (e *errLLM) Chat(messages []core.Message) (<-chan core.Result, error) {
	return nil, errors.New(e.err)
}

type recordingExecutor struct {
	mu    sync.Mutex
	calls []core.ToolCall
}

func (r *recordingExecutor) Execute(call core.ToolCall) core.ToolResult {
	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()
	return core.ToolResult{ToolCallID: call.ID, Content: "file contents"}
}

func (r *recordingExecutor) IsReadOnly(name string) bool {
	return true
}

type spyExecutor struct{}

func (s *spyExecutor) Execute(call core.ToolCall) core.ToolResult {
	return core.ToolResult{}
}

func (s *spyExecutor) IsReadOnly(name string) bool {
	return true
}
