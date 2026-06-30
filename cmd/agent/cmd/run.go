package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/agents"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/DietKyle956/last-known-good/internal/logger"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
	"github.com/DietKyle956/last-known-good/internal/singleshot"
	"github.com/DietKyle956/last-known-good/internal/skills"
	"github.com/DietKyle956/last-known-good/internal/tools"
	"github.com/spf13/cobra"
)

var (
	jsonFlag   bool
	agentFlag  string
	errNoPrompt = errors.New("prompt is required")
)

var runCmd = &cobra.Command{
	Use:   "run <prompt>",
	Short: "Run a single task and exit",
	Long:  "Run a single task with a prompt and exit without opening the TUI.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		prompt := args[0]

		apiKey := llm.APIKeyForModel(modelFlag)
		if apiKey == "" {
			return fmt.Errorf("no API key found for model %q — set DEEPSEEK_FLASH_KEY, DEEPSEEK_PRO_KEY, or DEEPSEEK_API_KEY", modelFlag)
		}

		client := llm.NewDeepSeekClient(llm.DeepSeekConfig{
			APIKey: apiKey,
			Model:  modelFlag,
			Stream: true,
		})

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting working directory: %w", err)
		}

		handle, err := sandbox.Start(cwd, sandbox.SandboxConfig{})
		if err != nil {
			return fmt.Errorf("starting sandbox: %w", err)
		}
		defer func() {
			if err := sandbox.Stop(handle); err != nil {
				fmt.Fprintf(os.Stderr, "error stopping sandbox: %v\n", err)
			}
		}()

		shell := sandbox.NewDockerExecer(handle)
		reg := tools.New(shell)
		tools.RegisterAll(reg)

		client.SetTools(toDeepSeekTools(reg.ToolDefinitions()))

		hookSys := hooks.New(nil)
		dangerous := hooks.NewDangerousCommandHook(nil)
		hookSys.Register(hooks.BeforeToolCall, dangerous.Handler)
		autoFormat := hooks.NewAutoFormatHook(shell, nil, nil)
		hookSys.Register(hooks.AfterToolCall, autoFormat.Handler)

		sessionID := time.Now().UnixMilli()
		logDir := filepath.Join(cwd, "logs")
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return fmt.Errorf("create log directory: %w", err)
		}
		log, err := logger.New(sessionID, logDir)
		if err != nil {
			return fmt.Errorf("create session logger: %w", err)
		}
		defer log.Close()
		for _, ht := range []hooks.HookType{
			hooks.SessionStarted,
			hooks.SessionEnded,
			hooks.BeforeModelCall,
			hooks.AfterModelCall,
			hooks.BeforeToolCall,
			hooks.AfterToolCall,
		} {
			hookSys.Register(ht, log.Hook)
		}
		hookSys.Notify(context.Background(), hooks.HookEvent{Type: hooks.SessionStarted, SessionID: sessionID})
		defer hookSys.Notify(context.Background(), hooks.HookEvent{Type: hooks.SessionEnded, SessionID: sessionID})

		var a *agent.Agent
		var events chan agent.AgentEvent

		if agentFlag != "" {
			al := agents.NewLoader("agents")
			if err := al.Load(); err != nil {
				return fmt.Errorf("load agents: %w", err)
			}
			agt, err := al.Get(agentFlag)
			if err != nil {
				return fmt.Errorf("agent %q not found: %w", agentFlag, err)
			}
			if len(agt.AllowedTools) > 0 {
				reg.Restrict(agt.AllowedTools)
			}
			if agt.ModelPreference != "" {
				client = llm.NewDeepSeekClient(llm.DeepSeekConfig{
					APIKey: apiKey,
					Model:  agt.ModelPreference,
					Stream: true,
				})
				client.SetTools(toDeepSeekTools(reg.ToolDefinitions()))
			}
			events = make(chan agent.AgentEvent, 128)
			a = agent.New(client, reg)
			a.SetHooks(hookSys)
			go func() {
				for ev := range a.Events() {
					events <- ev
				}
				close(events)
			}()
			ctx := context.Background()
			go func() {
				a.Run(ctx, []core.Message{
					{Role: "system", Content: agt.SystemPrompt},
					{Role: "user", Content: prompt},
				})
			}()
			renderer := singleshot.New(events, os.Stdout, jsonFlag)
			return renderer.Run()
		}

		events = make(chan agent.AgentEvent, 128)
		a = agent.New(client, reg)
		a.SetHooks(hookSys)
		go func() {
			for ev := range a.Events() {
				events <- ev
			}
			close(events)
		}()

		loader := skills.NewLoader("skills")
		if err := loader.Load(); err != nil {
			return fmt.Errorf("load skills: %w", err)
		}
		var skillSummaries string
		if ss := loader.Summaries(); len(ss) > 0 {
			var b strings.Builder
			for _, s := range ss {
				b.WriteString("- **" + s.Name + "**: " + s.Description + "\n")
			}
			skillSummaries = b.String()
		}

		var toolDefs string
		if defs := reg.ToolDefinitions(); len(defs) > 0 {
			var b strings.Builder
			for _, d := range defs {
				b.WriteString("- **" + d.Name + "**: " + d.Description + "\n")
			}
			toolDefs = b.String()
		}

		ctx := context.Background()
		go func() {
			a.Run(ctx, []core.Message{
				{Role: "system", Content: core.BuildSystemPrompt(skillSummaries, toolDefs)},
				{Role: "user", Content: prompt},
			})
		}()

		renderer := singleshot.New(events, os.Stdout, jsonFlag)
		return renderer.Run()
	},
}

func toDeepSeekTools(defs []tools.ToolDefinition) []llm.DeepSeekToolDef {
	out := make([]llm.DeepSeekToolDef, 0, len(defs))
	for _, d := range defs {
		var params json.RawMessage
		if len(d.Parameters) > 0 {
			b, err := llm.DeterministicMarshal(d.Parameters)
			if err == nil {
				params = json.RawMessage(b)
			}
		}
		strict := tools.SchemaSupportsStrict(d.Parameters)
		out = append(out, llm.DeepSeekToolDef{
			Type: "function",
			Function: llm.DeepSeekFunction{
				Name:        d.Name,
				Description: d.Description,
				Parameters:  params,
				Strict:      strict,
			},
		})
	}
	return out
}

func init() {
	runCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output structured JSON")
	runCmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Subagent short name to use for this run")
	rootCmd.AddCommand(runCmd)
}
