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

// OpenAPlanner calls any OpenAI-compatible chat completions endpoint.
// Works for OpenAI, Groq, Gemini, and other OpenAI-compatible providers.
type OpenAPlanner struct {
	APIKey   string
	Model    string
	Endpoint string
}

func (p *OpenAPlanner) Plan(goal string) ([]Task, error) {
	endpoint := p.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}
	model := p.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": fmt.Sprintf("Goal: %q\n\nReturn the task plan JSON.", goal)},
		},
		"max_tokens": 2048,
	}
	// json_object response_format is OpenAI-specific; Gemini's compatible endpoint rejects it.
	if strings.Contains(endpoint, "openai.com") || strings.Contains(endpoint, "groq.com") {
		body["response_format"] = map[string]string{"type": "json_object"}
	}
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 90 * time.Second}
	resp, err := client.Do(req) // #nosec G107 -- endpoint is user-configured via config file
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
