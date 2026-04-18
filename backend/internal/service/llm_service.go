package service

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"content-creator-imm/config"
)

const claudeModel = "glm-5"

var httpClient = &http.Client{Timeout: 300 * time.Second}

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
// maxTokens=0 uses default (2048).
func CallClaude(system, userPrompt string, maxTokens ...int) (string, error) {
	tokens := 2048
	if len(maxTokens) > 0 && maxTokens[0] > 0 {
		tokens = maxTokens[0]
	}
	reqBody := claudeRequest{
		Model:     claudeModel,
		MaxTokens: tokens,
		System:    system,
		Messages:  []Message{{Role: "user", Content: userPrompt}},
		Stream:    false,
	}
	data, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", apiURL(), bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", config.C.AnthropicKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := httpClient.Do(req)
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

	resp, err := httpClient.Do(req)
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
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // Increased buffer for large responses
	for scanner.Scan() {
		line := scanner.Text()
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
				Type     string `json:"type"`
				Text     string `json:"text"`
				Thinking string `json:"thinking"` // 百炼 API thinking_delta 格式
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(payload), &event); err != nil {
			continue
		}
		// 支持 Anthropic 标准 text_delta 和百炼 thinking_delta 两种格式
		if event.Type == "content_block_delta" {
			var token string
			if event.Delta.Type == "text_delta" {
				token = event.Delta.Text
			} else if event.Delta.Type == "thinking_delta" {
				token = event.Delta.Thinking
			}
			if token != "" {
				sb.WriteString(token)
				if cb != nil && !cb(token) {
					break
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		// Return partial content even on error, but report the error
		if sb.Len() > 0 {
			return sb.String(), nil // Return what we got
		}
		return "", fmt.Errorf("stream read error: %w", err)
	}
	return sb.String(), nil
}

