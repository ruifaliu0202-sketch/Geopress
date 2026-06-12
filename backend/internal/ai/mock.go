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
	draft := GeneratedDraft{
		Title:            title,
		Summary:          fmt.Sprintf("围绕 %s 的 AI 草稿，已结合当前工作区知识库上下文。", strings.Join(req.Keywords, "、")),
		Body:             body,
		Keywords:         req.Keywords,
		Sections:         []GeneratedSection{{Heading: "生成目标", Body: "根据关键词生成可编辑草稿。"}, {Heading: "知识上下文", Body: knowledgeSummary}, {Heading: "下一步", Body: "人工审校后再进入排程发布。"}},
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
		RetrievedIDs:  ids,
		PromptVersion: PromptVersion,
		SkillID:       req.Skill.ID,
		SkillVersion:  req.Skill.Version,
	}, nil
}

func warningsForChunks(chunks []KnowledgeChunk) []string {
	if len(chunks) == 0 {
		return []string{"未检索到可用知识片段，请人工补充事实和案例。"}
	}
	return []string{}
}
