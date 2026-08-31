package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	jsonData, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", p.Endpoint, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	resBody, _ := io.ReadAll(resp.Body)

	var wrapper struct {
		Choices []struct {
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(resBody, &wrapper)
	if len(wrapper.Choices) == 0 {
		return nil, fmt.Errorf("no response")
	}

	var result struct{ Tasks []Task `json:"tasks"` }
	err = json.Unmarshal([]byte(wrapper.Choices[0].Message.Content), &result)
	return result.Tasks, err
}
