package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAPlanner struct {
	APIKey   string
	Model    string
	Endpoint string
}

func (p *OpenAPlanner) Plan(goal string) ([]Task, error) {
	if p.Endpoint == "" {
		p.Endpoint = "https://api.openai.com/v1/chat/completions"
	}
	if p.Model == "" {
		p.Model = "gpt-4o-mini"
	}

	prompt := fmt.Sprintf(`
You are a Hybrid CLI Orchestrator.
Available tools: find, grep, awk, sed, jq, curl, git, docker.
Available AI agents: ollama, claude-code, gemini-cli.
Classify tasks as "tool" (traditional) or "agent" (AI).
Return JSON with "tasks" array. Each task: id, type, tool, args, depends_on, description.
Use "$OUTPUT" for piping.
Goal: "%s"`, goal)

	body := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a CLI orchestrator. Output strict JSON."},
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"},
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", p.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to API: %w", err)
	}
	defer resp.Body.Close()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var wrapper struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(wrapper.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from API")
	}

	var result struct{ Tasks []Task `json:"tasks"` }
	if err := json.Unmarshal([]byte(wrapper.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("unmarshal tasks: %w", err)
	}
	return result.Tasks, nil
}
