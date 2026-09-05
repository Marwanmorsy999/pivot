package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OllamaPlanner calls a locally running Ollama instance.
type OllamaPlanner struct {
	Endpoint string
	Model    string
}

func (p *OllamaPlanner) Plan(goal string) ([]Task, error) {
	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	model := p.Model
	if model == "" {
		model = "llama3.2:3b"
	}

	prompt := systemPrompt + "\n\nGoal: " + goal + "\n\nJSON:"

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"format": "json",
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(endpoint+"/api/generate", "application/json", bytes.NewBuffer(jsonData)) // #nosec G107 -- endpoint is user-configured via config file
	if err != nil {
		return nil, fmt.Errorf("request to Ollama: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return ParseTasks(ollamaResp.Response)
}
