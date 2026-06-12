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
		RetrievedIDs:  ids,
		PromptVersion: PromptVersion,
		SkillID:       req.Skill.ID,
		SkillVersion:  req.Skill.Version,
	}, nil
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
