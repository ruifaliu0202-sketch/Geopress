package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"geopress/backend/internal/model"
)

const (
	ProviderMock   = "mock"
	ProviderOpenAI = "openai"
	PromptVersion  = "content-generation-v1"

	GenerationStageInputAnalysis = "input_analysis"
	GenerationStageRetrieval     = "knowledge_retrieval"
	GenerationStagePlan          = "content_plan"
	GenerationStageDraft         = "draft_generation"
	GenerationStageQualityCheck  = "quality_check"
	GenerationStageRewrite       = "rewrite"
	GenerationStagePersist       = "persist_draft"
)

type Config struct {
	Provider           string
	OpenAIAPIKey       string
	OpenAIBaseURL      string
	OpenAIModel        string
	RequestTimeout     int
	GenerationPipeline GenerationPipelineSettings
}

type RuntimeConfig struct {
	mu     sync.RWMutex
	config Config
}

type PublicConfig struct {
	Provider           string                     `json:"provider"`
	OpenAIBaseURL      string                     `json:"openAIBaseUrl"`
	OpenAIModel        string                     `json:"openAIModel"`
	RequestTimeout     int                        `json:"requestTimeoutSeconds"`
	APIKeyConfigured   bool                       `json:"apiKeyConfigured"`
	APIKeyPreview      string                     `json:"apiKeyPreview"`
	GenerationPipeline GenerationPipelineSettings `json:"generationPipeline"`
}

type Provider interface {
	Name() string
	Model() string
	Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error)
	AnalyzeGenerationInput(ctx context.Context, req GenerateRequest) (GenerationStageResponse, error)
	PlanGeneration(ctx context.Context, req GenerateRequest, analysis GenerationStageResponse) (GenerationStageResponse, error)
	CheckGeneratedDraft(ctx context.Context, req GenerateRequest, draft GeneratedDraft) (GenerationStageResponse, error)
	RewriteGeneratedDraft(ctx context.Context, req GenerateRequest, draft GeneratedDraft, check GenerationStageResponse) (GenerateResponse, error)
	FormatKnowledgeContent(ctx context.Context, req FormatKnowledgeContentRequest) (FormatKnowledgeContentResponse, error)
}

type GenerationPipelineSettings struct {
	Free GenerationPipelinePlan `json:"free"`
	VIP  GenerationPipelinePlan `json:"vip"`
}

type GenerationPipelinePlan struct {
	InputAnalysis bool `json:"inputAnalysis"`
	ContentPlan   bool `json:"contentPlan"`
	QualityCheck  bool `json:"qualityCheck"`
	RewriteRounds int  `json:"rewriteRounds"`
}

type GenerateRequest struct {
	WorkspaceID      string
	UserID           string
	KnowledgeBaseIDs []string
	ContentType      string
	Keywords         []string
	Workspace        WorkspaceContext
	Skill            WritingSkill
	PublishFormat    PublishFormat
	KnowledgeChunks  []KnowledgeChunk
}

type WorkspaceContext struct {
	Name     string              `json:"name"`
	Type     model.WorkspaceType `json:"type"`
	Industry string              `json:"industry"`
	Language string              `json:"language"`
	Tone     string              `json:"tone"`
}

type KnowledgeChunk struct {
	ID               string   `json:"id"`
	KnowledgeBaseIDs []string `json:"knowledgeBaseIds"`
	Type             string   `json:"type"`
	Title            string   `json:"title"`
	Content          string   `json:"content"`
}

type GenerateResponse struct {
	Draft           GeneratedDraft   `json:"draft"`
	Prompt          PromptTranscript `json:"prompt"`
	RawOutput       json.RawMessage  `json:"rawOutput"`
	Provider        string           `json:"provider"`
	Model           string           `json:"model"`
	TokenUsage      TokenUsage       `json:"tokenUsage"`
	RetrievedIDs    []string         `json:"retrievedKnowledgeIds"`
	PromptVersion   string           `json:"promptVersion"`
	SkillID         string           `json:"skillId"`
	SkillVersion    string           `json:"skillVersion"`
	GenerationError string           `json:"generationError,omitempty"`
}

type GenerationStageResponse struct {
	Summary    string           `json:"summary"`
	Details    []string         `json:"details"`
	Warnings   []string         `json:"warnings"`
	Prompt     PromptTranscript `json:"prompt"`
	RawOutput  json.RawMessage  `json:"rawOutput"`
	Provider   string           `json:"provider"`
	Model      string           `json:"model"`
	TokenUsage TokenUsage       `json:"tokenUsage"`
}

type GenerationTrace struct {
	SubscriptionTier string                 `json:"subscriptionTier"`
	Pipeline         GenerationPipelinePlan `json:"pipeline"`
	Steps            []GenerationTraceStep  `json:"steps"`
	Warnings         []string               `json:"warnings"`
	RetrievedIDs     []string               `json:"retrievedKnowledgeIds"`
}

type GenerationTraceStep struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Status   string   `json:"status"`
	Summary  string   `json:"summary"`
	Details  []string `json:"details"`
	Warnings []string `json:"warnings"`
}

func (trace *GenerationTrace) AddStage(id, label string, stage GenerationStageResponse) {
	trace.AddStep(id, label, "succeeded", stage.Summary, stage.Details, stage.Warnings)
}

func (trace *GenerationTrace) AddStep(id, label, status, summary string, details []string, warnings []string) {
	trace.Steps = append(trace.Steps, GenerationTraceStep{
		ID:       id,
		Label:    label,
		Status:   status,
		Summary:  summary,
		Details:  cleanTraceStrings(details),
		Warnings: cleanTraceStrings(warnings),
	})
}

type FormatKnowledgeContentRequest struct {
	WorkspaceID string
	UserID      string
	Type        string
	Title       string
	Content     string
}

type FormatKnowledgeContentResponse struct {
	Content    string           `json:"content"`
	Prompt     PromptTranscript `json:"prompt"`
	RawOutput  json.RawMessage  `json:"rawOutput"`
	Provider   string           `json:"provider"`
	Model      string           `json:"model"`
	TokenUsage TokenUsage       `json:"tokenUsage"`
}

type PromptTranscript struct {
	System string `json:"system"`
	User   string `json:"user"`
	Schema any    `json:"schema,omitempty"`
}

type TokenUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

type GeneratedDraft struct {
	Title            string             `json:"title"`
	Summary          string             `json:"summary"`
	Body             string             `json:"body"`
	Keywords         []string           `json:"keywords"`
	Sections         []GeneratedSection `json:"sections"`
	UsedKnowledgeIDs []string           `json:"usedKnowledgeIds"`
	Warnings         []string           `json:"warnings"`
}

type GeneratedSection struct {
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

type PublishFormat struct {
	ID                string   `json:"id"`
	PlatformType      string   `json:"platformType"`
	PlatformName      string   `json:"platformName"`
	Mode              string   `json:"mode"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	TitleMaxRunes     int      `json:"titleMaxRunes"`
	BodyMaxRunes      int      `json:"bodyMaxRunes"`
	RequiredFields    []string `json:"requiredFields"`
	Structure         []string `json:"structure"`
	StyleGuidelines   []string `json:"styleGuidelines"`
	ValidationRules   []string `json:"validationRules"`
	AutomationChannel string   `json:"automationChannel"`
}

func (draft GeneratedDraft) Validate() error {
	if strings.TrimSpace(draft.Title) == "" {
		return errors.New("generated title is empty")
	}
	if strings.TrimSpace(draft.Summary) == "" {
		return errors.New("generated summary is empty")
	}
	if strings.TrimSpace(draft.Body) == "" {
		return errors.New("generated body is empty")
	}
	if len(draft.Keywords) == 0 {
		return errors.New("generated keywords are empty")
	}
	return nil
}

func NewProvider(cfg Config) Provider {
	if strings.EqualFold(strings.TrimSpace(cfg.Provider), ProviderOpenAI) {
		return NewOpenAIProvider(cfg)
	}
	return NewMockProvider()
}

func NewRuntimeConfig(cfg Config) *RuntimeConfig {
	return &RuntimeConfig{config: normalizeConfig(cfg)}
}

func (c *RuntimeConfig) Snapshot() Config {
	if c == nil {
		return normalizeConfig(Config{})
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config
}

func (c *RuntimeConfig) Public() PublicConfig {
	cfg := c.Snapshot()
	return PublicConfig{
		Provider:           cfg.Provider,
		OpenAIBaseURL:      cfg.OpenAIBaseURL,
		OpenAIModel:        cfg.OpenAIModel,
		RequestTimeout:     cfg.RequestTimeout,
		APIKeyConfigured:   cfg.OpenAIAPIKey != "",
		APIKeyPreview:      maskSecret(cfg.OpenAIAPIKey),
		GenerationPipeline: cfg.GenerationPipeline,
	}
}

func (c *RuntimeConfig) Update(next Config, clearAPIKey bool) Config {
	if c == nil {
		return normalizeConfig(next)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.config
	if strings.TrimSpace(next.OpenAIAPIKey) == "" && !clearAPIKey {
		next.OpenAIAPIKey = current.OpenAIAPIKey
	}
	if clearAPIKey {
		next.OpenAIAPIKey = ""
	}
	c.config = normalizeConfig(next)
	return c.config
}

func (c *RuntimeConfig) Provider() Provider {
	return NewProvider(c.Snapshot())
}

func (c *RuntimeConfig) GenerationPipelineForUser(user model.User) GenerationPipelinePlan {
	cfg := c.Snapshot()
	if user.HasActiveVIP(time.Now()) {
		return cfg.GenerationPipeline.VIP
	}
	return cfg.GenerationPipeline.Free
}

func normalizeConfig(cfg Config) Config {
	cfg.Provider = strings.TrimSpace(strings.ToLower(cfg.Provider))
	if cfg.Provider == "" {
		cfg.Provider = ProviderMock
	}
	if cfg.Provider != ProviderOpenAI {
		cfg.Provider = ProviderMock
	}
	cfg.OpenAIAPIKey = strings.TrimSpace(cfg.OpenAIAPIKey)
	cfg.OpenAIBaseURL = strings.TrimRight(emptyAs(cfg.OpenAIBaseURL, "https://api.openai.com/v1"), "/")
	cfg.OpenAIModel = emptyAs(cfg.OpenAIModel, "gpt-5.5")
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 45
	}
	cfg.GenerationPipeline = normalizeGenerationPipeline(cfg.GenerationPipeline)
	return cfg
}

func normalizeGenerationPipeline(value GenerationPipelineSettings) GenerationPipelineSettings {
	if !value.Free.InputAnalysis && !value.Free.ContentPlan && !value.Free.QualityCheck && value.Free.RewriteRounds == 0 {
		value.Free = GenerationPipelinePlan{
			InputAnalysis: true,
			ContentPlan:   false,
			QualityCheck:  false,
			RewriteRounds: 0,
		}
	}
	if !value.VIP.InputAnalysis && !value.VIP.ContentPlan && !value.VIP.QualityCheck && value.VIP.RewriteRounds == 0 {
		value.VIP = GenerationPipelinePlan{
			InputAnalysis: true,
			ContentPlan:   true,
			QualityCheck:  true,
			RewriteRounds: 1,
		}
	}
	value.Free.RewriteRounds = clampInt(value.Free.RewriteRounds, 0, 2)
	value.VIP.RewriteRounds = clampInt(value.VIP.RewriteRounds, 0, 3)
	return value
}

func clampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func maskSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func cleanTraceStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}
