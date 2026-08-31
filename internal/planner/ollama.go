package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaPlanner struct {
	Endpoint string
	Model    string
}

type ollamaReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format"`
}

func (p *OllamaPlanner) Plan(goal string) ([]Task, error) {
	prompt := fmt.Sprintf(`
You are a Hybrid CLI Orchestrator.
Available tools: find, grep, awk, sed, jq, curl, git, docker, kubectl, python3, node, go.
Available AI agents: ollama (local), claude-code, gemini-cli.
You MUST classify each task as "tool" (traditional CLI) or "agent" (AI).
If a task modifies code or requires reasoning, use "agent".
If it processes data (JSON, text, files), use "tool".
Return ONLY a JSON object with a "tasks" array.
Each task has: id (single letter), type ("tool" or "agent"), tool (string), args (array), depends_on (array), description.
Use "$OUTPUT" to pipe data between tasks.
Goal: "%s"
JSON:`, goal)

	reqBody := ollamaReq{Model: p.Model, Prompt: prompt, Stream: false, Format: "json"}
	jsonData, _ := json.Marshal(reqBody)
	resp, err := http.Post(p.Endpoint+"/api/generate", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var or struct{ Response string `json:"response"` }
	json.Unmarshal(body, &or)

	var result struct{ Tasks []Task `json:"tasks"` }
	err = json.Unmarshal([]byte(or.Response), &result)
	return result.Tasks, err
}
