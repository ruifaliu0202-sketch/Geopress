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
	content, response, err := p.callJSONSchema(ctx, prompt, "geopress_generated_draft", 4<<20)
	if err != nil {
		return GenerateResponse{}, err
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

func (p *OpenAIProvider) AnalyzeGenerationInput(ctx context.Context, req GenerateRequest) (GenerationStageResponse, error) {
	return p.generateStage(ctx, BuildGenerationStagePrompt(GenerationStageInputAnalysis, req, nil, nil))
}

func (p *OpenAIProvider) PlanGeneration(ctx context.Context, req GenerateRequest, analysis GenerationStageResponse) (GenerationStageResponse, error) {
	return p.generateStage(ctx, BuildGenerationStagePrompt(GenerationStagePlan, req, nil, &analysis))
}

func (p *OpenAIProvider) CheckGeneratedDraft(ctx context.Context, req GenerateRequest, draft GeneratedDraft) (GenerationStageResponse, error) {
	return p.generateStage(ctx, BuildGenerationStagePrompt(GenerationStageQualityCheck, req, &draft, nil))
}

func (p *OpenAIProvider) RewriteGeneratedDraft(ctx context.Context, req GenerateRequest, draft GeneratedDraft, check GenerationStageResponse) (GenerateResponse, error) {
	if p.apiKey == "" {
		return GenerateResponse{}, errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
	}

	rewriteReq := req
	rewriteReq.KnowledgeChunks = append([]KnowledgeChunk(nil), req.KnowledgeChunks...)
	prompt := BuildPrompt(rewriteReq)
	checkJSON, _ := json.MarshalIndent(check, "", "  ")
	draftJSON, _ := json.MarshalIndent(draft, "", "  ")
	prompt.User += fmt.Sprintf(`

重写要求：
- 下方是上一版草稿和质检结果，请在不新增事实的前提下修正风险。
- 如果质检风险无法靠重写解决，把风险写入 warnings，不要编造。

上一版草稿 JSON：
%s

质检结果 JSON：
%s`, string(draftJSON), string(checkJSON))

	content, response, err := p.callJSONSchema(ctx, prompt, "geopress_generated_draft_rewrite", 4<<20)
	if err != nil {
		return GenerateResponse{}, err
	}

	var rewritten GeneratedDraft
	if err := json.Unmarshal([]byte(content), &rewritten); err != nil {
		return GenerateResponse{}, fmt.Errorf("openai rewrite response is not valid generated draft JSON: %w", err)
	}
	if err := rewritten.Validate(); err != nil {
		return GenerateResponse{}, err
	}

	return GenerateResponse{
		Draft:         rewritten,
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

func (p *OpenAIProvider) generateStage(ctx context.Context, prompt PromptTranscript) (GenerationStageResponse, error) {
	if p.apiKey == "" {
		return GenerationStageResponse{}, errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
	}
	content, response, err := p.callJSONSchema(ctx, prompt, "geopress_generation_stage", 2<<20)
	if err != nil {
		return GenerationStageResponse{}, err
	}

	var stage struct {
		Summary  string   `json:"summary"`
		Details  []string `json:"details"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(content), &stage); err != nil {
		return GenerationStageResponse{}, fmt.Errorf("openai response is not valid generation stage JSON: %w", err)
	}
	stage.Summary = strings.TrimSpace(stage.Summary)
	if stage.Summary == "" {
		return GenerationStageResponse{}, errors.New("generation stage summary is empty")
	}
	return GenerationStageResponse{
		Summary:    stage.Summary,
		Details:    cleanStrings(stage.Details),
		Warnings:   cleanStrings(stage.Warnings),
		Prompt:     prompt,
		RawOutput:  json.RawMessage(content),
		Provider:   p.Name(),
		Model:      response.Model,
		TokenUsage: response.Usage.toTokenUsage(),
	}, nil
}

func (p *OpenAIProvider) FormatKnowledgeContent(ctx context.Context, req FormatKnowledgeContentRequest) (FormatKnowledgeContentResponse, error) {
	if p.apiKey == "" {
		return FormatKnowledgeContentResponse{}, errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
	}

	prompt := BuildKnowledgeContentFormatPrompt(req)
	content, response, err := p.callJSONSchema(ctx, prompt, "geopress_knowledge_content_format", 2<<20)
	if err != nil {
		return FormatKnowledgeContentResponse{}, err
	}

	var formatted struct {
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(content), &formatted); err != nil {
		return FormatKnowledgeContentResponse{}, fmt.Errorf("openai response is not valid knowledge format JSON: %w", err)
	}
	formatted.Content = strings.TrimSpace(formatted.Content)
	if formatted.Content == "" {
		return FormatKnowledgeContentResponse{}, errors.New("formatted knowledge content is empty")
	}

	return FormatKnowledgeContentResponse{
		Content:    formatted.Content,
		Prompt:     prompt,
		RawOutput:  json.RawMessage(content),
		Provider:   p.Name(),
		Model:      response.Model,
		TokenUsage: response.Usage.toTokenUsage(),
	}, nil
}

func (p *OpenAIProvider) callJSONSchema(ctx context.Context, prompt PromptTranscript, schemaName string, limit int64) (string, responsesAPIResponse, error) {
	payload := map[string]any{
		"model":        p.model,
		"instructions": prompt.System,
		"input":        prompt.User,
		"store":        false,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   schemaName,
				"schema": prompt.Schema,
				"strict": true,
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", responsesAPIResponse{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return "", responsesAPIResponse{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", responsesAPIResponse{}, err
	}
	defer resp.Body.Close()

	if limit <= 0 {
		limit = 2 << 20
	}
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, limit))
	if err != nil {
		return "", responsesAPIResponse{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", responsesAPIResponse{}, fmt.Errorf("openai request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var response responsesAPIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", responsesAPIResponse{}, err
	}

	content := strings.TrimSpace(response.outputText())
	if content == "" {
		return "", responsesAPIResponse{}, errors.New("openai response content is empty")
	}
	return content, response, nil
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

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
