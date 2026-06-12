package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAIProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewOpenAIProvider(cfg Config) *OpenAIProvider {
	cfg = normalizeConfig(cfg)
	return &OpenAIProvider{
		apiKey:  strings.TrimSpace(cfg.OpenAIAPIKey),
		baseURL: strings.TrimRight(emptyAs(cfg.OpenAIBaseURL, "https://api.openai.com/v1"), "/"),
		model:   emptyAs(cfg.OpenAIModel, "gpt-5.5"),
		client:  &http.Client{Timeout: time.Duration(cfg.RequestTimeout) * time.Second},
	}
}

func (p *OpenAIProvider) Name() string {
	return ProviderOpenAI
}

func (p *OpenAIProvider) Model() string {
	return p.model
}

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if p.apiKey == "" {
		return GenerateResponse{}, errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
	}

	prompt := BuildPrompt(req)
	payload := map[string]any{
		"model":        p.model,
		"instructions": prompt.System,
		"input":        prompt.User,
		"store":        false,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "geopress_generated_draft",
				"schema": prompt.Schema,
				"strict": true,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return GenerateResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return GenerateResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return GenerateResponse{}, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return GenerateResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GenerateResponse{}, fmt.Errorf("openai request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var response responsesAPIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return GenerateResponse{}, err
	}

	content := strings.TrimSpace(response.outputText())
	if content == "" {
		return GenerateResponse{}, errors.New("openai response content is empty")
	}

	var draft GeneratedDraft
	if err := json.Unmarshal([]byte(content), &draft); err != nil {
		return GenerateResponse{}, fmt.Errorf("openai response is not valid generated draft JSON: %w", err)
	}
	if err := draft.Validate(); err != nil {
		return GenerateResponse{}, err
	}

	return GenerateResponse{
		Draft:         draft,
		Prompt:        prompt,
		RawOutput:     json.RawMessage(content),
		Provider:      p.Name(),
		Model:         response.Model,
		TokenUsage:    response.Usage.toTokenUsage(),
		RetrievedIDs:  knowledgeIDs(req.KnowledgeChunks),
		PromptVersion: PromptVersion,
		SkillID:       req.Skill.ID,
		SkillVersion:  req.Skill.Version,
	}, nil
}

type responsesAPIResponse struct {
	Model      string               `json:"model"`
	OutputText string               `json:"output_text"`
	Output     []responsesAPIOutput `json:"output"`
	Usage      responsesAPIUsage    `json:"usage"`
}

type responsesAPIOutput struct {
	Type    string                      `json:"type"`
	Content []responsesAPIOutputContent `json:"content"`
}

type responsesAPIOutputContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responsesAPIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

func (response responsesAPIResponse) outputText() string {
	if response.OutputText != "" {
		return response.OutputText
	}
	var builder strings.Builder
	for _, item := range response.Output {
		if item.Type != "message" {
			continue
		}
		for _, content := range item.Content {
			if content.Type == "output_text" || content.Type == "text" {
				builder.WriteString(content.Text)
			}
		}
	}
	return builder.String()
}

func (usage responsesAPIUsage) toTokenUsage() TokenUsage {
	return TokenUsage{
		InputTokens:  usage.InputTokens,
		OutputTokens: usage.OutputTokens,
		TotalTokens:  usage.TotalTokens,
	}
}
