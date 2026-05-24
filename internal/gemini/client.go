// Package gemini wraps the Google Gen AI SDK for schema-constrained generation.
package gemini

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"google.golang.org/genai"
)

// ModelInfo is a model that can be used for generation, as shown in the UI.
type ModelInfo struct {
	ID          string // usable model id, e.g. "gemini-2.5-flash"
	DisplayName string
}

// ListModels returns the models that support generateContent, sorted by id.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	var out []ModelInfo
	for m, err := range c.models.All(ctx) {
		if err != nil {
			return nil, fmt.Errorf("list models: %w", err)
		}
		if !supportsGenerate(m.SupportedActions) {
			continue
		}
		id := strings.TrimPrefix(m.Name, "models/")
		label := m.DisplayName
		if label == "" {
			label = id
		}
		out = append(out, ModelInfo{ID: id, DisplayName: label})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func supportsGenerate(actions []string) bool {
	for _, a := range actions {
		if a == "generateContent" {
			return true
		}
	}
	return false
}

// Client generates structured JSON output from Gemini.
type Client struct {
	models *genai.Models
}

// New builds a Client for the Gemini API using the given API key.
func New(ctx context.Context, apiKey string) (*Client, error) {
	c, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create genai client: %w", err)
	}
	return &Client{models: c.Models}, nil
}

// GenerateList runs the prompt against the model, constraining the output to
// the given response schema, and returns the raw JSON bytes the model emitted.
func (c *Client) GenerateList(ctx context.Context, model, prompt string, respSchema *genai.Schema) ([]byte, error) {
	cfg := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema:   respSchema,
	}

	resp, err := c.models.GenerateContent(ctx, model, genai.Text(prompt), cfg)
	if err != nil {
		return nil, fmt.Errorf("gemini generate: %w", err)
	}

	out := strings.TrimSpace(resp.Text())
	if out == "" {
		return nil, fmt.Errorf("gemini returned no content (possibly blocked or empty response)")
	}
	return []byte(out), nil
}
