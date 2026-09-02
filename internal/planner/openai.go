package planner

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAPlanner calls any OpenAI-compatible chat completions endpoint.
// Works for OpenAI, Groq, Gemini, and other OpenAI-compatible providers.
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

	body := map[string]interface{}{
		"model": p.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("Goal: %q\n\nReturn the task plan JSON.", goal)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"max_tokens":      2048,
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, p.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to API: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	resBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(resBody))
	}

	var wrapper struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(resBody, &wrapper); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if len(wrapper.Choices) == 0 {
		return nil, fmt.Errorf("no response choices from API")
	}

	return ParseTasks(wrapper.Choices[0].Message.Content)
}
