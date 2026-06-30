package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/DietKyle956/last-known-good/internal/logger"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
	"github.com/DietKyle956/last-known-good/internal/singleshot"
	"github.com/DietKyle956/last-known-good/internal/tools"
	"github.com/spf13/cobra"
)

var (
	jsonFlag    bool
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

		hookSys := hooks.New(nil)
		dangerous := hooks.NewDangerousCommandHook(nil)
		hookSys.Register(hooks.BeforeToolCall, dangerous.Handler)
		autoFormat := hooks.NewAutoFormatHook(shell, nil, nil)
		hookSys.Register(hooks.AfterToolCall, autoFormat.Handler)

		sessionID := time.Now().UnixMilli()
		logDir := filepath.Join(cwd, "logs")
		os.MkdirAll(logDir, 0755)
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

		events := make(chan agent.AgentEvent, 128)
		a := agent.New(client, reg)
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
				{Role: "system", Content: "You are a helpful assistant."},
				{Role: "user", Content: prompt},
			})
		}()

		renderer := singleshot.New(events, os.Stdout, jsonFlag)
		return renderer.Run()
	},
}

func init() {
	runCmd.Flags().BoolVar(&jsonFlag, "json", false, "Output structured JSON")
	rootCmd.AddCommand(runCmd)
}
