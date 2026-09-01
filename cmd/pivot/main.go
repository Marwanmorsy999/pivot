package main

import (
"context"
"fmt"
"os"
"os/signal"
"syscall"

"pivot/internal/config"
"pivot/internal/core"
"pivot/internal/planner"
"pivot/internal/state"
"pivot/internal/tui"

"github.com/spf13/cobra"
)

var version = "dev"

func main() {
rootCmd := &cobra.Command{
Use:     "pivot",
Short:   "Universal Hybrid CLI Orchestrator (AI + Tools)",
Long:    "Orchestrate AI agents and CLI tools together in a single workflow with TUI, cost tracking, and worktree isolation.",
Version: version,
Run: func(cmd *cobra.Command, args []string) {
cmd.Help()
},
}
rootCmd.SetVersionTemplate("PIVOT {{.Version}}\n")

initCmd := &cobra.Command{
Use:   "init",
Short: "Initialize pivot (config, state DB, discover providers & local setup)",
Run: func(cmd *cobra.Command, args []string) {
// Detect everything automatically
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
defer s.Close()

fmt.Println("✅ Detected providers & local setup automatically.")
fmt.Printf("🔍 AI Provider: %s | Model: %s\n", cfg.Planner.Provider, cfg.Planner.Model)
fmt.Printf("🔌 Endpoint: %s\n", cfg.Planner.Endpoint)
if cfg.Planner.APIKey != "" {
fmt.Println("🔑 API Key: configured (hidden)")
}
fmt.Printf("🛠 Local Tools Found: %d (git=%v docker=%v node=%v python=%v)\n", len(detected.LocalTools), detected.LocalTools["git"], detected.LocalTools["docker"], detected.LocalTools["node"], detected.LocalTools["python3"] || detected.LocalTools["python"])
fmt.Println("✅ Initialized ~/.pivot/config.yaml (auto-detected)")
fmt.Println("✅ Initialized ~/.pivot/state.db")
fmt.Println("🔧 Run 'pivot run \"your goal\"' to start orchestrating.")
},
}

runCmd := &cobra.Command{
Use:   "run [goal]",
Short: "Execute a goal with hybrid orchestration (AI + CLI tools)",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
goal := args[0]

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
go func() {
<-sigCh
cancel()
}()

cfg, err := config.Load()
if err != nil {
fmt.Printf("❌ Failed to load config: %v\n", err)
return
}

s, err := state.New()
if err != nil {
fmt.Printf("❌ Failed to initialize state: %v\n", err)
return
}
defer s.Close()

sessionID, err := s.CreateSession(goal)
if err != nil {
fmt.Printf("❌ Failed to create session: %v\n", err)
return
}
fmt.Printf("📝 Session: %s\n", sessionID)

var p planner.Planner
switch cfg.Planner.Provider {
case "openai", "groq", "gemini", "anthropic":
endpoint := cfg.Planner.Endpoint
if endpoint == "" {
if cfg.Planner.Provider == "groq" {
endpoint = "https://api.groq.com/openai/v1/chat/completions"
} else if cfg.Planner.Provider == "gemini" {
endpoint = "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions"
} else {
endpoint = "https://api.openai.com/v1/chat/completions"
}
}
p = &planner.OpenAPlanner{
APIKey:   cfg.Planner.APIKey,
Model:    cfg.Planner.Model,
Endpoint: endpoint,
}
default:
p = &planner.OllamaPlanner{
Endpoint: cfg.Planner.Endpoint,
Model:    cfg.Planner.Model,
}
}

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

eventCh := make(chan core.Event, 100)
go func() {
orchestrator := core.NewOrchestrator(tasks, sessionID, s, eventCh)
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

resumeCmd := &cobra.Command{
Use:   "resume [session-id]",
Short: "Resume a failed or paused session",
Args:  cobra.ExactArgs(1),
Run: func(cmd *cobra.Command, args []string) {
sessionID := args[0]
s, err := state.New()
if err != nil {
fmt.Printf("❌ Failed to initialize state: %v\n", err)
return
}
defer s.Close()

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

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

cfg, err := config.Load()
if err != nil {
fmt.Printf("❌ Failed to load config: %v\n", err)
return
}
var p planner.Planner
if cfg.Planner.Provider == "ollama" {
p = &planner.OllamaPlanner{Endpoint: cfg.Planner.Endpoint, Model: cfg.Planner.Model}
} else {
p = &planner.OpenAPlanner{APIKey: cfg.Planner.APIKey, Model: cfg.Planner.Model, Endpoint: cfg.Planner.Endpoint}
}

tasks, _ := p.Plan(goal)
eventCh := make(chan core.Event, 100)
go func() {
orchestrator := core.NewOrchestrator(tasks, sessionID, s, eventCh)
err := orchestrator.Run(ctx)
if err != nil {
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

detectCmd := &cobra.Command{
Use:   "detect",
Short: "Detect all providers and local setup (super easy)",
Run: func(cmd *cobra.Command, args []string) {
r := config.Detect()
fmt.Println("🔍 Pivot Auto-Detection Report")
fmt.Println("──────────────────────────────")
fmt.Println("AI Providers Found:")
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
fmt.Printf("🏆 Best Provider: %s (model: %s)\n", r.DetectedProvider, r.DetectedModel)
fmt.Printf("🔌 Endpoint: %s\n", r.DetectedEndpoint)
if r.DetectedAPIKey != "" {
fmt.Println("🔑 API Key: configured")
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
defer s.Close()

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
goal, _ := s.GetGoal(id)
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
