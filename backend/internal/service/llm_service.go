package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"content-creator-imm/config"
)

const claudeModel = "glm-5"

func apiURL() string {
	base := strings.TrimRight(config.C.LLMBaseURL, "/")
	if base == "" {
		base = "https://api.anthropic.com"
	}
	return base + "/v1/messages"
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeRequest struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system,omitempty"`
	Messages  []Message `json:"messages"`
	Stream    bool      `json:"stream"`
}

// StreamCallback is called for each token. Return false to stop.
type StreamCallback func(token string) bool

// CallClaude sends a non-streaming request and returns the full response text.
func CallClaude(system, userPrompt string) (string, error) {
	reqBody := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 4096,
		System:    system,
		Messages:  []Message{{Role: "user", Content: userPrompt}},
		Stream:    false,
	}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", apiURL(), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.C.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude api: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("claude api %d: %s", resp.StatusCode, body)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	for _, block := range result.Content {
		if block.Type == "text" {
			return block.Text, nil
		}
	}
	return "", fmt.Errorf("empty response")
}

// StreamClaude sends a streaming request. For each token, calls cb.
// Accumulated full text is returned.
func StreamClaude(system, userPrompt string, cb StreamCallback) (string, error) {
	if config.C.AnthropicKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY 未配置，请在 config.json 中设置 anthropic_api_key")
	}

	reqBody := claudeRequest{
		Model:     claudeModel,
		MaxTokens: 4096,
		System:    system,
		Messages:  []Message{{Role: "user", Content: userPrompt}},
		Stream:    true,
	}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", apiURL(), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.C.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude api stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("claude api %d: %s", resp.StatusCode, body)
	}

	var sb strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		// Support both "data: " (standard) and "data:" (no space)
		var payload string
		if strings.HasPrefix(line, "data: ") {
			payload = strings.TrimPrefix(line, "data: ")
		} else if strings.HasPrefix(line, "data:") {
			payload = strings.TrimPrefix(line, "data:")
		} else {
			continue
		}
		if payload == "[DONE]" {
			break
		}

		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			sb.WriteString(event.Delta.Text)
			if cb != nil && !cb(event.Delta.Text) {
				break
			}
		}
	}
	return sb.String(), scanner.Err()
}
