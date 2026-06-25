package ai

import (
	"bytes"
	"context"
	"encoding/base64"
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

func (p *OpenAIProvider) ExtractDocumentText(ctx context.Context, req OCRRequest) (OCRResponse, error) {
	if p.apiKey == "" {
		return OCRResponse{}, errors.New("OPENAI_API_KEY is required when AI_PROVIDER=openai")
	}

	system := "你是 Geopress 知识库 OCR 服务。只提取图片或 PDF 中真实可见的文字，不要编造看不清的内容。保持原始语言，尽量保留标题、列表、表格和段落结构。返回严格 JSON。"
	userText := strings.TrimSpace(fmt.Sprintf(`请从这张图片中提取可用于知识库检索的文字。

文件名：%s
MIME：%s

要求：
- 如果文件里没有可读文字，content 返回空字符串，warnings 说明原因。
- 如果有表格、截图界面或多页 PDF，使用 Markdown 标题、表格或项目符号表达。
- 不要添加图片中不存在的事实。`, req.Filename, req.MimeType))
	prompt := PromptTranscript{
		System: system,
		User:   userText,
		Schema: map[string]any{
			"type":                 "object",
			"additionalProperties": false,
			"properties": map[string]any{
				"content": map[string]any{"type": "string"},
				"warnings": map[string]any{
					"type":  "array",
					"items": map[string]any{"type": "string"},
				},
			},
			"required": []string{"content", "warnings"},
		},
	}

	content, response, err := p.callDocumentJSONSchema(ctx, prompt, req, "geopress_ocr_result", 4<<20)
	if err != nil {
		return OCRResponse{}, err
	}

	var parsed struct {
		Content  string   `json:"content"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return OCRResponse{}, fmt.Errorf("openai OCR response is not valid JSON: %w", err)
	}
	parsed.Content = strings.TrimSpace(parsed.Content)
	if parsed.Content == "" {
		return OCRResponse{}, errors.New("AI vision OCR returned empty content")
	}

	return OCRResponse{
		Text:       parsed.Content,
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

func (p *OpenAIProvider) callDocumentJSONSchema(ctx context.Context, prompt PromptTranscript, req OCRRequest, schemaName string, limit int64) (string, responsesAPIResponse, error) {
	normalizedMIME := strings.ToLower(strings.TrimSpace(req.MimeType))
	var fileContent map[string]any
	switch {
	case strings.HasPrefix(normalizedMIME, "image/"):
		fileContent = map[string]any{
			"type":      "input_image",
			"image_url": fmt.Sprintf("data:%s;base64,%s", emptyAs(normalizedMIME, "image/png"), base64.StdEncoding.EncodeToString(req.Data)),
		}
	case normalizedMIME == "application/pdf" || strings.EqualFold(strings.TrimSpace(req.FileKind), "pdf"):
		filename := strings.TrimSpace(req.Filename)
		if filename == "" {
			filename = "document.pdf"
		}
		fileContent = map[string]any{
			"type":      "input_file",
			"filename":  filename,
			"file_data": fmt.Sprintf("data:application/pdf;base64,%s", base64.StdEncoding.EncodeToString(req.Data)),
		}
	default:
		return "", responsesAPIResponse{}, fmt.Errorf("AI vision OCR does not support MIME type %q", req.MimeType)
	}

	payload := map[string]any{
		"model":        p.model,
		"instructions": prompt.System,
		"input": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "input_text",
						"text": prompt.User,
					},
					fileContent,
				},
			},
		},
		"store": false,
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
		return "", responsesAPIResponse{}, fmt.Errorf("openai OCR request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}

	var response responsesAPIResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", responsesAPIResponse{}, err
	}

	content := strings.TrimSpace(response.outputText())
	if content == "" {
		return "", responsesAPIResponse{}, errors.New("openai OCR response content is empty")
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
