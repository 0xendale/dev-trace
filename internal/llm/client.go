package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Completer is the interface implemented by all LLM clients.
type Completer interface {
	Complete(ctx context.Context, systemPrompt, userContent string) (string, error)
}

const defaultEndpoint = "https://api.openai.com/v1/chat/completions"

// Client calls the OpenAI chat completions API.
type Client struct {
	apiKey   string
	model    string
	endpoint string
	http     *http.Client
}

// NewClient returns a Client targeting the production OpenAI endpoint.
func NewClient(apiKey, model string) *Client {
	return NewClientWithURL(apiKey, model, defaultEndpoint)
}

// NewClientWithURL returns a Client targeting a custom endpoint (used in tests).
func NewClientWithURL(apiKey, model, endpoint string) *Client {
	return &Client{apiKey: apiKey, model: model, endpoint: endpoint, http: &http.Client{Timeout: 120 * time.Second}}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// Complete sends systemPrompt + userContent to the model and returns the reply.
func (c *Client) Complete(ctx context.Context, systemPrompt, userContent string) (string, error) {
	body, err := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, raw)
	}

	var cr chatResponse
	if err := json.Unmarshal(raw, &cr); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return cr.Choices[0].Message.Content, nil
}
