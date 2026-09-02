package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicPlanner struct {
	APIKey   string
	Model    string
	Endpoint string
}

func (p *AnthropicPlanner) Plan(goal string) ([]Task, error) {
	if p.Endpoint == "" {
		p.Endpoint = "https://api.anthropic.com/v1/messages"
	}
	if p.Model == "" {
		p.Model = "claude-3-5-sonnet-20241022"
	}

	prompt := fmt.Sprintf(`You are a Hybrid CLI Orchestrator.
Available tools: find, grep, awk, sed, jq, curl, git, docker, kubectl, python3, node, go.
Available AI agents: ollama, claude-code, gemini-cli.
Classify tasks as "tool" (traditional CLI) or "agent" (AI).
Return ONLY a JSON object with a "tasks" array. Each task: id, type, tool, args, depends_on, description.
Use "$OUTPUT" for piping between tasks.
Goal: "%s"`, goal)

	body := map[string]interface{}{
		"model":      p.Model,
		"max_tokens": 1024,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("x-api-key", p.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to Anthropic: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(resBody))
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

	text := wrapper.Content[0].Text

	// Strip markdown code fences if present
	if len(text) > 7 && text[:7] == "```json" {
		text = text[7:]
		if idx := len(text) - 3; idx > 0 && text[idx:] == "```" {
			text = text[:idx]
		}
	}

	var result struct {
		Tasks []Task `json:"tasks"`
	}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}
	return result.Tasks, nil
}
