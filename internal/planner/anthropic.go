package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AnthropicPlanner calls the Anthropic Messages API to generate a task plan.
type AnthropicPlanner struct {
	APIKey   string
	Model    string
	Endpoint string
}

const systemPrompt = `You are Pivot, a Hybrid CLI Orchestrator that turns goals into executable task graphs.

TASK TYPES:
- "tool" — runs a CLI program deterministically (no reasoning needed)
- "agent" — runs an AI agent for tasks requiring code generation, analysis, or judgment

ALLOWED TOOLS (tool type): find, grep, awk, sed, cat, echo, wc, sort, uniq, head, tail,
xargs, tar, zip, unzip, cut, tr, tee, diff, patch, ls, cp, mv, rm, mkdir, chmod, chown,
touch, stat, file, env, printenv, which, date, jq, curl, wget, ssh, rsync, git, docker,
kubectl, make, python3, python, node, go, npm, npx, pip, pip3, cargo, rustc, terraform,
helm, aws, gcloud, az, sh, bash

ALLOWED AGENTS (agent type): ollama, claude-code, gemini-cli

OUTPUT PIPING:
- Use $OUTPUT[task-id] to reference the stdout of a completed task in args
- Use $OUTPUT (legacy) to reference DependsOn[0]
- Multiple deps: use $OUTPUT[id1] and $OUTPUT[id2] in separate args

RULES:
- Return ONLY a JSON object {"tasks": [...]} — no markdown, no explanation
- Every task must have: id (short slug), type, tool, args (array), depends_on (array), description
- Optional: timeout_sec (integer, default 300), retries (integer, default 0)
- IDs must be unique slugs (e.g. "fetch-data", "parse-json", "summarize")
- depends_on must only reference IDs defined in the same plan`

func (p *AnthropicPlanner) Plan(goal string) ([]Task, error) {
	if p.Endpoint == "" {
		p.Endpoint = "https://api.anthropic.com/v1/messages"
	}
	if p.Model == "" {
		p.Model = "claude-opus-4-5"
	}

	body := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 2048,
		"system":     systemPrompt,
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Goal: %q\n\nReturn the task plan JSON.", goal)},
		},
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req) // #nosec G107 -- endpoint is user-configured via config file
	if err != nil {
		return nil, fmt.Errorf("request to Anthropic: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(resBody))
	}

	var wrapper struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(wrapper.Content) == 0 {
		return nil, fmt.Errorf("empty response from Anthropic")
	}

	return ParseTasks(wrapper.Content[0].Text)
}

// ParseTasks strips optional markdown fences and decodes the tasks JSON.
func ParseTasks(text string) ([]Task, error) {
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "```") {
		// Strip ```json ... ``` or ``` ... ```
		if idx := strings.Index(text, "\n"); idx != -1 {
			text = text[idx+1:]
		}
		if idx := strings.LastIndex(text, "```"); idx != -1 {
			text = text[:idx]
		}
		text = strings.TrimSpace(text)
	}

	var result struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("unmarshal tasks from LLM response: %w\nraw: %s", err, text)
	}
	return result.Tasks, nil
}
