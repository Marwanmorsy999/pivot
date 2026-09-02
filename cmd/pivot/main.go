package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/Marwanmorsy999/pivot/internal/config"
	"github.com/Marwanmorsy999/pivot/internal/core"
	"github.com/Marwanmorsy999/pivot/internal/planner"
	"github.com/Marwanmorsy999/pivot/internal/state"
	"github.com/Marwanmorsy999/pivot/internal/tui"

	"github.com/spf13/cobra"
)

var version = "dev"

func buildPlanner(cfg *config.Config) planner.Planner {
	switch cfg.Planner.Provider {
	case "anthropic":
		return &planner.AnthropicPlanner{
			APIKey:   cfg.Planner.APIKey,
			Model:    cfg.Planner.Model,
			Endpoint: cfg.Planner.Endpoint,
		}
	case "openai", "groq", "gemini":
		endpoint := cfg.Planner.Endpoint
		if endpoint == "" {
			switch cfg.Planner.Provider {
			case "groq":
				endpoint = "https://api.groq.com/openai/v1/chat/completions"
			case "gemini":
				endpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
			default:
				endpoint = "https://api.openai.com/v1/chat/completions"
			}
		}
		return &planner.OpenAPlanner{
			APIKey:   cfg.Planner.APIKey,
			Model:    cfg.Planner.Model,
			Endpoint: endpoint,
		}
	default:
		return &planner.OllamaPlanner{
			Endpoint: cfg.Planner.Endpoint,
			Model:    cfg.Planner.Model,
		}
	}
}

func newSignalCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sigCh; cancel() }()
	return ctx, cancel
}

func main() {
	rootCmd := &cobra.Command{
		Use:     "pivot",
		Short:   "Universal Hybrid CLI Orchestrator (AI + Tools)",
		Long:    "Orchestrate AI agents and CLI tools together in a single workflow with TUI, cost tracking, and worktree isolation.",
		Version: version,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to display help: %v\n", err)
			}
		},
	}
	rootCmd.SetVersionTemplate("PIVOT {{.Version}}\n")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize pivot (config, state DB, discover providers & local setup)",
		Run: func(cmd *cobra.Command, args []string) {
			detected := config.Detect()
			cfg := config.ConfigFromDetection(detected)

			if err := config.SaveDetected(cfg); err != nil {
				fmt.Printf("❌ Failed to save config: %v\n", err)
				os.Exit(1)
			}
			s, err := state.New()
			if err != nil {
				fmt.Printf("❌ Failed to initialize state: %v\n", err)
				os.Exit(1)
			}
			defer func() {
				if err := s.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "state close warning: %v\n", err)
				}
			}()

			fmt.Println("✅ Detected providers & local setup automatically.")
			fmt.Printf("🔍 AI Provider: %s | Model: %s\n", cfg.Planner.Provider, cfg.Planner.Model)
			fmt.Printf("🔌 Endpoint: %s\n", cfg.Planner.Endpoint)
			if cfg.Planner.APIKey != "" {
				fmt.Println("🔑 API Key: configured (hidden)")
			}
			fmt.Printf("🛠 Local Tools Found: %d (git=%v docker=%v node=%v python=%v)\n",
				len(detected.LocalTools),
				detected.LocalTools["git"],
				detected.LocalTools["docker"],
				detected.LocalTools["node"],
				detected.LocalTools["python3"] || detected.LocalTools["python"],
			)
			fmt.Println("✅ Initialized ~/.pivot/config.yaml (auto-detected)")
			fmt.Println("✅ Initialized ~/.pivot/state.db")
			fmt.Println("🔧 Run 'pivot run \"your goal\"' to start orchestrating.")
		},
	}

	var dryRun bool
	var maxParallel int

	runCmd := &cobra.Command{
		Use:   "run [goal]",
		Short: "Execute a goal with hybrid orchestration (AI + CLI tools)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			goal := args[0]

			ctx, cancel := newSignalCtx()
			defer cancel()

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("❌ Failed to load config: %v\n", err)
				return
			}
			if cfg.Planner.Provider == "" || (cfg.Planner.Provider == "ollama" && cfg.Planner.Endpoint == "") {
				fmt.Println("❌ No AI provider configured. Run 'pivot detect' then 'pivot init', or set an API key env var.")
				return
			}

			s, err := state.New()
			if err != nil {
				fmt.Printf("❌ Failed to initialize state: %v\n", err)
				return
			}
			defer func() {
				if err := s.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "state close warning: %v\n", err)
				}
			}()

			sessionID, err := s.CreateSession(goal)
			if err != nil {
				fmt.Printf("❌ Failed to create session: %v\n", err)
				return
			}
			fmt.Printf("📝 Session: %s\n", sessionID)

			p := buildPlanner(cfg)

			fmt.Println("🧠 Planning...")
			tasks, err := p.Plan(goal)
			if err != nil {
				fmt.Printf("❌ Planning failed: %v\n", err)
				return
			}
			if len(tasks) == 0 {
				fmt.Println("❌ No tasks generated.")
				return
			}

			// Validate tasks before execution.
			if err := planner.Validate(tasks); err != nil {
				fmt.Printf("❌ Invalid task plan: %v\n", err)
				return
			}

			// Dry run: show plan without executing.
			if dryRun {
				fmt.Println("\n📋 Dry run — task plan (not executing):")
				for i, t := range tasks {
					fmt.Printf("  %d. [%s] %s: %s\n", i+1, t.Type, t.Tool, t.Description)
					if len(t.DependsOn) > 0 {
						fmt.Printf("     depends_on: %v\n", t.DependsOn)
					}
				}
				return
			}

			// Persist the task plan for deterministic resume.
			if tasksJSON, err := json.Marshal(tasks); err == nil {
				if err := s.SaveSessionTasks(sessionID, tasksJSON); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to persist task plan: %v\n", err)
				}
			}

			opts := core.OrchestratorOptions{MaxParallel: maxParallel}
			eventCh := make(chan core.Event, 100)
			go func() {
				orchestrator := core.NewOrchestrator(tasks, sessionID, s, eventCh, opts)
				err := orchestrator.Run(ctx)
				if err != nil && err != context.Canceled {
					eventCh <- core.Event{Type: core.EventError, Message: err.Error()}
				}
				close(eventCh)
			}()

			tuiModel := tui.NewModel(sessionID, goal, tasks, eventCh)
			if err := tuiModel.Run(); err != nil {
				fmt.Printf("❌ TUI error: %v\n", err)
			}
		},
	}
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the task plan without executing")
	runCmd.Flags().IntVar(&maxParallel, "parallel", 4, "Max tasks to run in parallel (0 = unlimited)")

	resumeCmd := &cobra.Command{
		Use:   "resume [session-id]",
		Short: "Resume a failed or paused session (uses persisted task plan)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sessionID := args[0]

			s, err := state.New()
			if err != nil {
				fmt.Printf("❌ Failed to initialize state: %v\n", err)
				return
			}
			defer func() {
				if err := s.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "state close warning: %v\n", err)
				}
			}()

			failed, err := s.GetFailedTasks(sessionID)
			if err != nil {
				fmt.Printf("❌ Failed to get failed tasks: %v\n", err)
				return
			}
			if len(failed) == 0 {
				fmt.Println("✅ No failed tasks to resume.")
				return
			}

			fmt.Printf("🔄 Resuming session %s (%d failed tasks)\n", sessionID, len(failed))
			goal, err := s.GetGoal(sessionID)
			if err != nil {
				fmt.Printf("❌ Failed to get session goal: %v\n", err)
				return
			}

			// Load persisted task plan — avoids non-deterministic re-planning.
			var tasks []planner.Task
			savedJSON, err := s.GetSessionTasks(sessionID)
			if err != nil || len(savedJSON) == 0 {
				// Fallback: re-plan (old sessions without saved plan).
				fmt.Println("⚠️  No persisted plan found, re-planning...")
				cfg, err := config.Load()
				if err != nil {
					fmt.Printf("❌ Failed to load config: %v\n", err)
					return
				}
				p := buildPlanner(cfg)
				tasks, err = p.Plan(goal)
				if err != nil {
					fmt.Printf("❌ Planning failed: %v\n", err)
					return
				}
			} else {
				if err := json.Unmarshal(savedJSON, &tasks); err != nil {
					fmt.Printf("❌ Failed to parse saved plan: %v\n", err)
					return
				}
			}

			ctx, cancel := newSignalCtx()
			defer cancel()

			opts := core.OrchestratorOptions{MaxParallel: maxParallel}
			eventCh := make(chan core.Event, 100)
			go func() {
				orchestrator := core.NewOrchestrator(tasks, sessionID, s, eventCh, opts)
				err := orchestrator.Run(ctx)
				if err != nil && err != context.Canceled {
					eventCh <- core.Event{Type: core.EventError, Message: err.Error()}
				}
				close(eventCh)
			}()

			tuiModel := tui.NewModel(sessionID, goal, tasks, eventCh)
			if err := tuiModel.Run(); err != nil {
				fmt.Printf("❌ TUI error: %v\n", err)
			}
		},
	}
	resumeCmd.Flags().IntVar(&maxParallel, "parallel", 4, "Max tasks to run in parallel")

	detectCmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect all providers and local setup",
		Run: func(cmd *cobra.Command, args []string) {
			r := config.Detect()
			fmt.Println("🔍 Pivot Auto-Detection Report")
			fmt.Println("──────────────────────────────")
			fmt.Println("AI Providers Found:")
			if len(r.Providers) == 0 {
				fmt.Println("  (none — set ANTHROPIC_API_KEY, OPENAI_API_KEY, GROQ_API_KEY, or run Ollama locally)")
			}
			for name, found := range r.Providers {
				status := "❌"
				if found {
					status = "✅"
				}
				fmt.Printf("  %s %s\n", status, name)
			}
			fmt.Println("Local Tools Detected:")
			if len(r.LocalTools) == 0 {
				fmt.Println("  (none found)")
			} else {
				for name := range r.LocalTools {
					fmt.Printf("  ✅ %s\n", name)
				}
			}
			fmt.Println("──────────────────────────────")
			if r.DetectedProvider == "" {
				fmt.Println("⚠️  No provider detected. Set an API key env var or start Ollama.")
			} else {
				fmt.Printf("🏆 Best Provider: %s (model: %s)\n", r.DetectedProvider, r.DetectedModel)
				fmt.Printf("🔌 Endpoint: %s\n", r.DetectedEndpoint)
				if r.DetectedAPIKey != "" {
					fmt.Println("🔑 API Key: configured")
				}
			}
			fmt.Println("\n💡 Run 'pivot init' to apply auto-config.")
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show recent sessions",
		Run: func(cmd *cobra.Command, args []string) {
			s, err := state.New()
			if err != nil {
				fmt.Printf("❌ Failed to initialize state: %v\n", err)
				return
			}
			defer func() {
				if err := s.Close(); err != nil {
					fmt.Fprintf(os.Stderr, "state close warning: %v\n", err)
				}
			}()

			sessions, err := s.GetSessions()
			if err != nil {
				fmt.Printf("❌ Failed to get sessions: %v\n", err)
				return
			}
			if len(sessions) == 0 {
				fmt.Println("No sessions found.")
				return
			}
			fmt.Println("📂 Recent Sessions:")
			for _, id := range sessions {
				goal, err := s.GetGoal(id)
				if err != nil {
					fmt.Printf("  - %s: <failed to load goal: %v>\n", id, err)
					continue
				}
				fmt.Printf("  - %s: %s\n", id, goal)
			}
		},
	}

	rootCmd.AddCommand(initCmd, runCmd, resumeCmd, statusCmd, detectCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
