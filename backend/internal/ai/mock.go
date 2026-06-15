package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Name() string {
	return ProviderMock
}

func (p *MockProvider) Model() string {
	return "geopress-mock-writer-v1"
}

func (p *MockProvider) Generate(_ context.Context, req GenerateRequest) (GenerateResponse, error) {
	prompt := BuildPrompt(req)
	ids := knowledgeIDs(req.KnowledgeChunks)
	knowledgeSummary := "当前工作区尚未维护可用知识条目。"
	if len(req.KnowledgeChunks) > 0 {
		parts := make([]string, 0, len(req.KnowledgeChunks))
		for _, chunk := range req.KnowledgeChunks {
			parts = append(parts, chunk.Title+" - "+chunk.Content)
		}
		knowledgeSummary = strings.Join(parts, "；")
	}

	title := fmt.Sprintf("%s：从策略到执行", req.Keywords[0])
	body := fmt.Sprintf("关键词：%s\n\n知识库上下文：%s\n\n这是一篇由 mock AI Provider 生成的结构化草稿，用于打通关键词、知识检索、写作技能、结构化输出、草稿保存和生成日志链路。接入真实模型后，Provider 会替换为配置的外部模型。", strings.Join(req.Keywords, "、"), knowledgeSummary)
	sections := []GeneratedSection{{Heading: "生成目标", Body: "根据关键词生成可编辑草稿。"}, {Heading: "知识上下文", Body: knowledgeSummary}, {Heading: "下一步", Body: "人工审校后再进入排程发布。"}}
	if req.PublishFormat.ID == FormatXiaohongshuLongArticle {
		title = truncateRunes(fmt.Sprintf("%s怎么做更稳定", req.Keywords[0]), req.PublishFormat.TitleMaxRunes)
		body = fmt.Sprintf("如果你正在做%s，最容易卡住的不是缺想法，而是缺少一套能反复执行的方法。\n\n一、先把目标读者说清楚\n不要先追热点，先确认这篇内容写给谁、他们正在遇到什么问题、看完以后能做什么。\n\n二、把知识库变成可复用素材\n这次检索到的上下文是：%s\n发布前建议把其中能公开的事实、案例和边界再人工确认一遍。\n\n三、正文要像清单一样可执行\n每一段只讲一个动作，尽量给判断标准，例如什么时候开始、需要准备什么、做完如何检查。\n\n四、发布前做最后检查\n确认标题不超过 64 字，正文没有敏感隐私，没有编造数据，话题和正文主题一致。\n\n%s", strings.Join(req.Keywords, "、"), knowledgeSummary, strings.Join(xiaohongshuMockHashtags(req.Keywords), " "))
		sections = []GeneratedSection{
			{Heading: "先把目标读者说清楚", Body: "确认内容写给谁、解决什么问题、读者看完能做什么。"},
			{Heading: "把知识库变成可复用素材", Body: knowledgeSummary},
			{Heading: "发布前做最后检查", Body: "检查标题长度、事实边界、隐私信息和话题相关性。"},
		}
	}
	draft := GeneratedDraft{
		Title:            title,
		Summary:          fmt.Sprintf("围绕 %s 的 AI 草稿，已结合当前工作区知识库上下文。", strings.Join(req.Keywords, "、")),
		Body:             body,
		Keywords:         req.Keywords,
		Sections:         sections,
		UsedKnowledgeIDs: ids,
		Warnings:         warningsForChunks(req.KnowledgeChunks),
	}

	raw, err := json.Marshal(draft)
	if err != nil {
		return GenerateResponse{}, err
	}

	return GenerateResponse{
		Draft:         draft,
		Prompt:        prompt,
		RawOutput:     raw,
		Provider:      p.Name(),
		Model:         p.Model(),
		TokenUsage:    mockTokenUsage(prompt.User, body),
		RetrievedIDs:  ids,
		PromptVersion: PromptVersion,
		SkillID:       req.Skill.ID,
		SkillVersion:  req.Skill.Version,
	}, nil
}

func (p *MockProvider) AnalyzeGenerationInput(_ context.Context, req GenerateRequest) (GenerationStageResponse, error) {
	prompt := BuildGenerationStagePrompt(GenerationStageInputAnalysis, req, nil, nil)
	details := []string{
		"主题关键词：" + strings.Join(req.Keywords, "、"),
		"内容类型由系统限定为：" + req.ContentType,
		"用户输入只作为主题和素材线索，不会覆盖发布格式或安全边界。",
	}
	if len(req.KnowledgeBaseIDs) == 0 {
		details = append(details, "知识库范围：全部可用知识库包。")
	} else {
		details = append(details, fmt.Sprintf("知识库范围：%d 个已选择知识库包。", len(req.KnowledgeBaseIDs)))
	}
	return mockStageResponse(p, prompt, "已完成关键词和输入边界分析。", details, nil)
}

func (p *MockProvider) PlanGeneration(_ context.Context, req GenerateRequest, _ GenerationStageResponse) (GenerationStageResponse, error) {
	prompt := BuildGenerationStagePrompt(GenerationStagePlan, req, nil, nil)
	details := []string{
		"按发布格式组织标题、摘要、正文、章节和风险提示。",
		"优先使用检索到的知识条目，不补写未确认事实。",
		"正文会围绕读者问题、可执行建议和发布前检查组织。",
	}
	for _, item := range req.PublishFormat.Structure {
		details = append(details, "结构要求："+item)
		if len(details) >= 6 {
			break
		}
	}
	return mockStageResponse(p, prompt, "已生成受系统规则约束的创作计划。", details, nil)
}

func (p *MockProvider) CheckGeneratedDraft(_ context.Context, req GenerateRequest, draft GeneratedDraft) (GenerationStageResponse, error) {
	prompt := BuildGenerationStagePrompt(GenerationStageQualityCheck, req, &draft, nil)
	warnings := append([]string{}, draft.Warnings...)
	if len(draft.UsedKnowledgeIDs) == 0 {
		warnings = append(warnings, "草稿没有引用具体知识条目，请人工确认事实来源。")
	}
	details := []string{
		"草稿标题、摘要、正文和关键词字段均已生成。",
		fmt.Sprintf("实际使用知识条目：%d 个。", len(draft.UsedKnowledgeIDs)),
		"内容仍保持 draft 状态，发布前需要人工审校。",
	}
	return mockStageResponse(p, prompt, "已完成草稿质量和事实边界检查。", details, warnings)
}

func (p *MockProvider) RewriteGeneratedDraft(_ context.Context, req GenerateRequest, draft GeneratedDraft, check GenerationStageResponse) (GenerateResponse, error) {
	prompt := BuildPrompt(req)
	if len(check.Warnings) > 0 {
		draft.Warnings = uniqueStrings(append(draft.Warnings, check.Warnings...))
	}
	if !strings.Contains(draft.Body, "人工复核") {
		draft.Body += "\n\n发布前请人工复核知识来源、事实边界和平台格式要求。"
	}
	raw, err := json.Marshal(draft)
	if err != nil {
		return GenerateResponse{}, err
	}
	return GenerateResponse{
		Draft:         draft,
		Prompt:        prompt,
		RawOutput:     raw,
		Provider:      p.Name(),
		Model:         p.Model(),
		TokenUsage:    mockTokenUsage(prompt.User, draft.Body),
		RetrievedIDs:  knowledgeIDs(req.KnowledgeChunks),
		PromptVersion: PromptVersion,
		SkillID:       req.Skill.ID,
		SkillVersion:  req.Skill.Version,
	}, nil
}

func (p *MockProvider) FormatKnowledgeContent(_ context.Context, req FormatKnowledgeContentRequest) (FormatKnowledgeContentResponse, error) {
	prompt := BuildKnowledgeContentFormatPrompt(req)
	formatted := formatKnowledgeMarkdown(req)
	raw, err := json.Marshal(map[string]string{"content": formatted})
	if err != nil {
		return FormatKnowledgeContentResponse{}, err
	}
	return FormatKnowledgeContentResponse{
		Content:    formatted,
		Prompt:     prompt,
		RawOutput:  raw,
		Provider:   p.Name(),
		Model:      p.Model(),
		TokenUsage: mockTokenUsage(prompt.User, formatted),
	}, nil
}

func formatKnowledgeMarkdown(req FormatKnowledgeContentRequest) string {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "未命名知识条目"
	}
	itemType := strings.TrimSpace(req.Type)
	if itemType == "" {
		itemType = "note"
	}
	content := normalizeKnowledgeContent(req.Content)
	if content == "" {
		content = "待补充。"
	}

	return strings.Join([]string{
		"## " + title,
		"",
		"**类型**：" + itemType,
		"",
		"### 核心内容",
		content,
		"",
		"### 使用边界",
		"- 仅使用上方已确认信息生成内容。",
		"- 未出现的数据、案例、承诺和效果不得补写。",
		"",
		"### 待补充",
		"- 如需更稳定输出，请补充适用场景、禁用表达和可公开案例。",
	}, "\n")
}

func normalizeKnowledgeContent(value string) string {
	lines := strings.Split(strings.TrimSpace(value), "\n")
	cleaned := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || strings.HasPrefix(line, ">") {
			cleaned = append(cleaned, line)
			continue
		}
		cleaned = append(cleaned, "- "+line)
	}
	return strings.Join(cleaned, "\n")
}

func mockStageResponse(p *MockProvider, prompt PromptTranscript, summary string, details []string, warnings []string) (GenerationStageResponse, error) {
	if warnings == nil {
		warnings = []string{}
	}
	raw, err := json.Marshal(map[string]any{
		"summary":  summary,
		"details":  details,
		"warnings": warnings,
	})
	if err != nil {
		return GenerationStageResponse{}, err
	}
	return GenerationStageResponse{
		Summary:    summary,
		Details:    details,
		Warnings:   warnings,
		Prompt:     prompt,
		RawOutput:  raw,
		Provider:   p.Name(),
		Model:      p.Model(),
		TokenUsage: mockTokenUsage(prompt.User, summary+strings.Join(details, "")),
	}, nil
}

func mockTokenUsage(input string, output string) TokenUsage {
	inputTokens := estimateTokens(input)
	outputTokens := estimateTokens(output)
	return TokenUsage{
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  inputTokens + outputTokens,
	}
}

func estimateTokens(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	return len([]rune(value))/4 + len(strings.Fields(value)) + 1
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func xiaohongshuMockHashtags(values []string) []string {
	hashtags := make([]string, 0, len(values))
	for _, value := range values {
		tag := strings.Trim(strings.TrimSpace(value), "#")
		if tag == "" {
			continue
		}
		hashtags = append(hashtags, "#"+strings.Join(strings.Fields(tag), ""))
		if len(hashtags) == 5 {
			break
		}
	}
	if len(hashtags) == 0 {
		return []string{"#小红书长文"}
	}
	return hashtags
}

func warningsForChunks(chunks []KnowledgeChunk) []string {
	if len(chunks) == 0 {
		return []string{"未检索到可用知识片段，请人工补充事实和案例。"}
	}
	return []string{}
}
