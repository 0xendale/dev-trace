package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xendale/devtrace/internal/llm"
)

func TestNewClientUsesDefaultEndpoint(t *testing.T) {
	c := llm.NewClient("key", "gpt-4o")
	// NewClient should not return nil and should wire to the production endpoint.
	if c == nil {
		t.Fatal("NewClient returned nil")
	}
}

func TestCompleteEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[]}`))
	}))
	defer srv.Close()

	c := llm.NewClientWithURL("key", "gpt-4o", srv.URL+"/v1/chat/completions")
	_, err := c.Complete(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
}

func TestCompleteSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header present
		if r.Header.Get("Authorization") == "" {
			t.Error("missing Authorization header")
		}
		json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": "Test report content"}},
			},
		})
	}))
	defer srv.Close()

	c := llm.NewClientWithURL("test-api-key", "gpt-4o", srv.URL+"/v1/chat/completions")
	result, err := c.Complete(context.Background(), "system prompt", "user content")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if result != "Test report content" {
		t.Errorf("want 'Test report content', got %q", result)
	}
}

func TestCompleteAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key"}}`))
	}))
	defer srv.Close()

	c := llm.NewClientWithURL("bad-key", "gpt-4o", srv.URL+"/v1/chat/completions")
	_, err := c.Complete(context.Background(), "sys", "user")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}
