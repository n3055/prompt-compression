// Package llm provides adapters for calling various LLM APIs.
// This is an optional component — the engine works standalone without it.
// Wire this in when you want the engine to call the LLM directly.
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider defines which LLM service to use.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"    // OpenAI, Azure OpenAI, vLLM, LiteLLM
	ProviderAnthropic Provider = "anthropic" // Anthropic Claude
	ProviderOllama    Provider = "ollama"    // Ollama (local models)
	ProviderCustom    Provider = "custom"    // Any OpenAI-compatible endpoint
)

// Config holds LLM connection settings.
type Config struct {
	Provider Provider      // Which provider to use.
	BaseURL  string        // API base URL (e.g., "https://api.openai.com/v1").
	APIKey   string        // API key (optional for Ollama).
	Model    string        // Model name (e.g., "gpt-4o", "llama3").
	Timeout  time.Duration // HTTP timeout for LLM calls.
}

// Client calls an LLM API with a compressed prompt.
type Client struct {
	cfg  Config
	http *http.Client
}

// New creates an LLM client with the given configuration.
func New(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
	}
}

// Complete sends the compressed prompt to the configured LLM and returns the text response.
func (c *Client) Complete(compressedPrompt string) (string, error) {
	switch c.cfg.Provider {
	case ProviderOpenAI, ProviderCustom:
		return c.openAIStyle(compressedPrompt)
	case ProviderAnthropic:
		return c.anthropicStyle(compressedPrompt)
	case ProviderOllama:
		return c.ollamaStyle(compressedPrompt)
	default:
		return "", fmt.Errorf("unsupported LLM provider: %s", c.cfg.Provider)
	}
}

// --- OpenAI-compatible request format ---
// Works with: OpenAI, Azure OpenAI, vLLM, LiteLLM, LocalAI, text-generation-webui

type openAIRequest struct {
	Model    string           `json:"model"`
	Messages []openAIMessage  `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) openAIStyle(prompt string) (string, error) {
	reqBody := openAIRequest{
		Model: c.cfg.Model,
		Messages: []openAIMessage{
			{Role: "user", Content: prompt},
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.cfg.BaseURL+"/chat/completions", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("LLM request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read LLM response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result openAIResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("LLM returned zero choices")
	}

	return result.Choices[0].Message.Content, nil
}

// --- Anthropic Claude request format ---

type anthropicRequest struct {
	Model     string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (c *Client) anthropicStyle(prompt string) (string, error) {
	reqBody := anthropicRequest{
		Model:     c.cfg.Model,
		MaxTokens: 4096,
		Messages: []anthropicMessage{
			{Role: "user", Content: prompt},
		},
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.cfg.BaseURL+"/messages", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.cfg.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("Anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Anthropic returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result anthropicResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse Anthropic response: %w", err)
	}

	if len(result.Content) == 0 {
		return "", fmt.Errorf("Anthropic returned zero content blocks")
	}

	return result.Content[0].Text, nil
}

// --- Ollama request format (local models) ---

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

func (c *Client) ollamaStyle(prompt string) (string, error) {
	reqBody := ollamaRequest{
		Model:  c.cfg.Model,
		Prompt: prompt,
		Stream: false,
	}
	data, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.http.Post(c.cfg.BaseURL+"/api/generate", "application/json", bytes.NewReader(data))
	if err != nil {
		return "", fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Ollama response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result ollamaResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return result.Response, nil
}
