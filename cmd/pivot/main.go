package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/Marwanmorsy999/pivot/internal/config"
	"github.com/Marwanmorsy999/pivot/internal/core"
	"github.com/Marwanmorsy999/pivot/internal/export"
	gh "github.com/Marwanmorsy999/pivot/internal/github"
	"github.com/Marwanmorsy999/pivot/internal/planner"
	"github.com/Marwanmorsy999/pivot/internal/state"
	"github.com/Marwanmorsy999/pivot/internal/tui"
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
	go func() {
		select {
		case <-sigCh:
			cancel()
		case <-ctx.Done():
			// Normal exit: drain signal channel to avoid goroutine leak.
		}
		signal.Stop(sigCh)
	}()
	return ctx, cancel
}

func runOrchestrator(ctx context.Context, tasks []planner.Task, sessionID string, s *state.State, maxParallel int, cfg *config.Config) error {
	opts := core.OrchestratorOptions{MaxParallel: maxParallel, Provider: cfg.Planner.Provider, Model: cfg.Planner.Model}
	eventCh := make(chan core.Event, 100)
	orchErrCh := make(chan error, 1) // buffered — goroutine never blocks
	go func() {
		orchestrator := core.NewOrchestrator(tasks, sessionID, s, eventCh, opts)
		err := orchestrator.Run(ctx)
		if err != nil && err != context.Canceled {
			eventCh <- core.Event{Type: core.EventError, Message: err.Error()}
		}
		close(eventCh)
		orchErrCh <- err
	}()

	goal, _ := s.GetGoal(sessionID)
	tuiModel := tui.NewModel(sessionID, goal, tasks, eventCh)
	if err := tuiModel.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return <-orchErrCh
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
	var workflowFile string
	var issueNumber int
	var githubToken string
	var githubRepo string

	runCmd := &cobra.Command{
		Use:   "run [goal]",
		Short: "Execute a goal with hybrid orchestration (AI + CLI tools)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := newSignalCtx()
			defer cancel()

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("❌ Failed to load config: %v\n", err)
				return
			}

			var goal string
			var tasks []planner.Task

			if workflowFile != "" {
				// Load from YAML file — no LLM needed.
				fileGoal, fileTasks, err := planner.LoadWorkflowFile(workflowFile)
				if err != nil {
					fmt.Printf("❌ Failed to load workflow file: %v\n", err)
					return
				}
				goal = fileGoal
				if len(args) > 0 {
					goal = args[0] // CLI arg overrides file goal
				}
				if goal == "" {
					goal = filepath.Base(workflowFile)
				}
				tasks = fileTasks
			} else if issueNumber > 0 {
				// GitHub issue path.
				client, err := gh.New(githubToken, githubRepo)
				if err != nil {
					fmt.Printf("❌ GitHub client error: %v\n", err)
					return
				}
				issue, err := client.GetIssue(issueNumber)
				if err != nil {
					fmt.Printf("❌ Failed to fetch issue #%d: %v\n", issueNumber, err)
					return
				}
				goal = gh.IssueGoal(issue)
				fmt.Printf("📋 Issue #%d: %s\n", issue.Number, issue.Title)
				fmt.Printf("🧠 Planning for issue...\n")
				if cfg.Planner.Provider == "" {
					fmt.Println("❌ No AI provider configured. Run 'pivot init' first.")
					return
				}
				p := buildPlanner(cfg)
				tasks, err = p.Plan(goal)
				if err != nil {
					fmt.Printf("❌ Planning failed: %v\n", err)
					return
				}
			} else {
				// AI planning path.
				if len(args) == 0 {
					fmt.Println("❌ Provide a goal, use --file to load a workflow, or use --issue N.")
					return
				}
				goal = args[0]

				if cfg.Planner.Provider == "" || (cfg.Planner.Provider == "ollama" && cfg.Planner.Endpoint == "") {
					fmt.Println("❌ No AI provider configured. Run 'pivot detect' then 'pivot init', or set an API key env var.")
					return
				}

				p := buildPlanner(cfg)
				fmt.Println("🧠 Planning...")
				tasks, err = p.Plan(goal)
				if err != nil {
					fmt.Printf("❌ Planning failed: %v\n", err)
					return
				}
				if len(tasks) == 0 {
					fmt.Println("❌ No tasks generated.")
					return
				}
			}

			if err := planner.Validate(tasks); err != nil {
				fmt.Printf("❌ Invalid task plan: %v\n", err)
				return
			}

			if dryRun {
				fmt.Println("\n📋 Dry run — task plan (not executing):")
				for i, t := range tasks {
					var icon string
					switch t.Type {
					case planner.TypeAgent:
						icon = "🤖"
					case planner.TypeCheckpoint:
						icon = "⏸ "
					default:
						icon = "🔧"
					}
					fmt.Printf("  %d. %s [%s] %s: %s\n", i+1, icon, t.Type, t.Tool, t.Description)
					if len(t.DependsOn) > 0 {
						fmt.Printf("     depends_on: %v\n", t.DependsOn)
					}
					if t.Before != "" {
						fmt.Printf("     before: %s\n", t.Before)
					}
					if t.After != "" {
						fmt.Printf("     after: %s\n", t.After)
					}
				}
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

			if tasksJSON, err := json.Marshal(tasks); err == nil {
				if err := s.SaveSessionTasks(sessionID, tasksJSON); err != nil {
					fmt.Fprintf(os.Stderr, "warning: failed to persist task plan: %v\n", err)
				}
			}

			orchErr := runOrchestrator(ctx, tasks, sessionID, s, maxParallel, cfg)
			if orchErr != nil && orchErr != context.Canceled {
				fmt.Printf("❌ Execution error: %v\n", orchErr)
			}

			// Post-run GitHub issue actions.
			closeOnSuccess, _ := cmd.Flags().GetBool("close-on-success")
			if issueNumber > 0 && closeOnSuccess && orchErr == nil {
				client, ghErr := gh.New(githubToken, githubRepo)
				if ghErr == nil {
					entries, _ := s.GetJournalEntries(sessionID)
					var totalCost float64
					for _, e := range entries {
						totalCost += e.Cost
					}
					comment := fmt.Sprintf(
						"✅ **Pivot completed** session `%s`\n\n"+
							"**Tasks:** %d | **Cost:** $%.6f\n\n"+
							"Run `pivot export %s` for the full report.",
						sessionID, len(entries), totalCost, sessionID,
					)
					if err := client.CreateComment(issueNumber, comment); err != nil {
						fmt.Fprintf(os.Stderr, "⚠️  Failed to comment on issue: %v\n", err)
					} else if err := client.CloseIssue(issueNumber); err != nil {
						fmt.Fprintf(os.Stderr, "⚠️  Failed to close issue: %v\n", err)
					} else {
						fmt.Printf("✅ Closed issue #%d\n", issueNumber)
					}
				}
			}
		},
	}
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show the task plan without executing")
	runCmd.Flags().IntVar(&maxParallel, "parallel", 4, "Max tasks to run in parallel (0 = unlimited)")
	runCmd.Flags().StringVarP(&workflowFile, "file", "f", "", "Load task graph from a YAML workflow file (skips AI planning)")
	runCmd.Flags().IntVar(&issueNumber, "issue", 0, "Fetch goal from a GitHub issue number (requires GITHUB_TOKEN)")
	runCmd.Flags().StringVar(&githubToken, "github-token", "", "GitHub personal access token (overrides GITHUB_TOKEN env)")
	runCmd.Flags().StringVar(&githubRepo, "github-repo", "", "GitHub repo as owner/repo (auto-detected from git remote)")
	runCmd.Flags().Bool("close-on-success", false, "Comment on and close the GitHub issue when all tasks complete (requires --issue)")

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

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("❌ Failed to load config: %v\n", err)
				return
			}

			var tasks []planner.Task
			savedJSON, err := s.GetSessionTasks(sessionID)
			if err != nil || len(savedJSON) == 0 {
				fmt.Println("⚠️  No persisted plan found, re-planning...")
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

			if err := runOrchestrator(ctx, tasks, sessionID, s, maxParallel, cfg); err != nil && err != context.Canceled {
				fmt.Printf("❌ Execution error: %v\n", err)
			}
		},
	}
	resumeCmd.Flags().IntVar(&maxParallel, "parallel", 4, "Max tasks to run in parallel")

	exportCmd := &cobra.Command{
		Use:   "export [session-id]",
		Short: "Export a session as a Markdown report",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			sessionID := args[0]
			outFile, _ := cmd.Flags().GetString("out")

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

			goal, err := s.GetGoal(sessionID)
			if err != nil {
				fmt.Printf("❌ Failed to get session goal: %v\n", err)
				return
			}

			entries, err := s.GetJournalEntries(sessionID)
			if err != nil {
				fmt.Printf("❌ Failed to get journal entries: %v\n", err)
				return
			}

			// Build a type lookup from the persisted task plan.
			taskTypes := make(map[string]string)
			if savedJSON, err := s.GetSessionTasks(sessionID); err == nil && len(savedJSON) > 0 {
				var savedTasks []planner.Task
				if json.Unmarshal(savedJSON, &savedTasks) == nil {
					for _, t := range savedTasks {
						taskTypes[t.ID] = string(t.Type)
					}
				}
			}

			var records []export.TaskRecord
			var totalCost float64
			var totalTokens int
			for _, e := range entries {
				taskType := taskTypes[e.TaskID]
				if taskType == "" {
					taskType = "tool"
				}
				records = append(records, export.TaskRecord{
					ID:     e.TaskID,
					Type:   taskType,
					Tool:   e.Tool,
					Status: e.Status,
					Output: e.Output,
					Error:  e.Error,
					Cost:   e.Cost,
					Tokens: e.Tokens,
				})
				totalCost += e.Cost
				totalTokens += e.Tokens
			}

			report := export.Report(sessionID, goal, records, totalCost, totalTokens)

			if outFile == "" {
				fmt.Print(report)
				return
			}
			if err := os.WriteFile(outFile, []byte(report), 0600); err != nil {
				fmt.Printf("❌ Failed to write report: %v\n", err)
				return
			}
			fmt.Printf("✅ Report saved to %s\n", outFile)
		},
	}
	exportCmd.Flags().StringP("out", "o", "", "Write report to file instead of stdout")

	// pivot workflow scaffold — generates an example workflow YAML
	scaffoldCmd := &cobra.Command{
		Use:   "scaffold [name]",
		Short: "Generate an example workflow YAML file",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			name := "workflow"
			if len(args) > 0 {
				name = args[0]
			}
			outPath := name + ".yaml"

			example := planner.WorkflowFile{
				Goal: "Example workflow — edit tasks to suit your needs",
				Tasks: []planner.Task{
					{
						ID: "check-env", Type: planner.TypeTool, Tool: "sh",
						Args:        []string{"-c", "echo Node $(node --version 2>/dev/null || echo N/A) && echo Go $(go version 2>/dev/null || echo N/A)"},
						Description: "Check environment",
					},
					{
						ID: "pause", Type: planner.TypeCheckpoint,
						Description: "Review environment output before continuing",
						Prompt:      "Environment looks good?",
						DependsOn:   []string{"check-env"},
					},
					{
						ID: "done", Type: planner.TypeTool, Tool: "echo",
						Args:        []string{"All done!"},
						Description: "Final step",
						DependsOn:   []string{"pause"},
						Before:      "echo starting final step",
						After:       "echo finished",
					},
				},
			}

			data, err := yaml.Marshal(example)
			if err != nil {
				fmt.Printf("❌ Failed to generate scaffold: %v\n", err)
				return
			}
			if err := os.WriteFile(outPath, data, 0600); err != nil {
				fmt.Printf("❌ Failed to write file: %v\n", err)
				return
			}
			fmt.Printf("✅ Scaffold written to %s\n", outPath)
			fmt.Printf("   Edit it and run: pivot run --file %s\n", outPath)
		},
	}

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
				status, _ := s.GetSessionStatus(id)
				icon := "○"
				switch status {
				case "completed":
					icon = "✅"
				case "failed":
					icon = "❌"
				case "active":
					icon = "▶"
				}
				fmt.Printf("  %s %s  %s\n", icon, id, goal)
			}
		},
	}


	watchCmd := &cobra.Command{
		Use:   "watch",
		Short: "Poll GitHub for labeled issues and auto-dispatch pivot runs",
		Run: func(cmd *cobra.Command, args []string) {
			label, _ := cmd.Flags().GetString("label")
			intervalSec, _ := cmd.Flags().GetInt("interval")
			ghToken, _ := cmd.Flags().GetString("github-token")
			ghRepo, _ := cmd.Flags().GetString("github-repo")
			maxPar, _ := cmd.Flags().GetInt("parallel")

			client, err := gh.New(ghToken, ghRepo)
			if err != nil {
				fmt.Printf("❌ GitHub client error: %v\n", err)
				return
			}

			cfg, err := config.Load()
			if err != nil {
				fmt.Printf("❌ Failed to load config: %v\n", err)
				return
			}

			fmt.Printf("👁  Watching for issues labeled %q every %ds. Ctrl+C to stop.\n", label, intervalSec)
			seen := make(map[int]bool)

			ctx, cancel := newSignalCtx()
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					fmt.Println("\n👋 Watch stopped.")
					return
				default:
				}

				issues, err := client.ListIssues(label)
				if err != nil {
					fmt.Fprintf(os.Stderr, "⚠️  ListIssues error: %v\n", err)
				} else {
					for _, issue := range issues {
						if seen[issue.Number] {
							continue
						}
						seen[issue.Number] = true
						fmt.Printf("\n🚀 Dispatching issue #%d: %s\n", issue.Number, issue.Title)

						// Mark issue in-progress immediately.
						_ = client.AddLabel(issue.Number, "pivot:in-progress")

						goal := gh.IssueGoal(&issue)
						p := buildPlanner(cfg)
						tasks, planErr := p.Plan(goal)
						if planErr != nil {
							fmt.Fprintf(os.Stderr, "❌ Planning failed for #%d: %v\n", issue.Number, planErr)
							_ = client.CreateComment(issue.Number, fmt.Sprintf("❌ Pivot planning failed: %v", planErr))
							continue
						}
						if err := planner.Validate(tasks); err != nil {
							fmt.Fprintf(os.Stderr, "❌ Invalid plan for #%d: %v\n", issue.Number, err)
							continue
						}

						s, stateErr := state.New()
						if stateErr != nil {
							fmt.Fprintf(os.Stderr, "❌ State error: %v\n", stateErr)
							continue
						}
						sessionID, _ := s.CreateSession(goal)
						if tasksJSON, err := json.Marshal(tasks); err == nil {
							_ = s.SaveSessionTasks(sessionID, tasksJSON)
						}

						orchErr := runOrchestrator(ctx, tasks, sessionID, s, maxPar, cfg)
						_ = s.Close()

						if orchErr == nil {
							comment := fmt.Sprintf(
								"✅ **Pivot completed** session `%s`\n\nRun `pivot export %s` for the full report.",
								sessionID, sessionID,
							)
							_ = client.CreateComment(issue.Number, comment)
							_ = client.CloseIssue(issue.Number)
							fmt.Printf("✅ Issue #%d done and closed.\n", issue.Number)
						} else {
							_ = client.CreateComment(issue.Number, fmt.Sprintf("❌ Pivot failed: %v\nSession: `%s`", orchErr, sessionID))
							fmt.Fprintf(os.Stderr, "❌ Issue #%d failed: %v\n", issue.Number, orchErr)
						}
					}
				}

				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(intervalSec) * time.Second):
				}
			}
		},
	}
	watchCmd.Flags().String("label", "pivot", "GitHub issue label to watch for")
	watchCmd.Flags().Int("interval", 30, "Poll interval in seconds")
	watchCmd.Flags().String("github-token", "", "GitHub personal access token (overrides GITHUB_TOKEN env)")
	watchCmd.Flags().String("github-repo", "", "GitHub repo as owner/repo (auto-detected from git remote)")
	watchCmd.Flags().Int("parallel", 4, "Max parallel tasks per dispatched run")

	rootCmd.AddCommand(initCmd, runCmd, resumeCmd, statusCmd, detectCmd, exportCmd, scaffoldCmd, watchCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
