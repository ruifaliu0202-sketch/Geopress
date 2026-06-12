package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildPrompt(req GenerateRequest) PromptTranscript {
	schema := DraftJSONSchema()
	system := strings.Join([]string{
		"你是 Geopress 的内容创作引擎。",
		"只使用用户提供的工作区、关键词和知识库片段来创作；不要编造品牌事实、客户、数据或发布结果。",
		"输出必须是符合 JSON Schema 的 JSON 对象，不要输出 Markdown 代码块。",
		"AI 输出只能作为 draft 草稿，必须便于人工继续编辑审核。",
	}, "\n")

	chunks, _ := json.MarshalIndent(req.KnowledgeChunks, "", "  ")
	publishFormat, _ := json.MarshalIndent(req.PublishFormat, "", "  ")
	user := fmt.Sprintf(`工作区：
- 名称：%s
- 类型：%s
- 行业：%s
- 语言：%s
- 语气：%s

内容要求：
- 内容类型：%s
- 写作技能：%s %s
- 技能合同：%s
- 关键词：%s

目标媒体平台发布格式 JSON：
%s

可用知识片段 JSON：
%s

生成要求：
- 必须严格匹配目标媒体平台发布格式，不要靠猜测平台格式。
- title: 中文标题；如果发布格式设置了 titleMaxRunes，标题不得超过该限制。
- summary: 80 字以内摘要。
- body: 完整正文，使用自然段，必要时使用小标题；如果发布格式包含结构、风格、校验规则，正文必须满足这些要求。
- sections: 从正文中抽取 3 到 6 个结构化章节。
- keywords: 保留并补充关键词。
- usedKnowledgeIds: 只填实际使用的知识片段 ID。
- warnings: 如果知识不足、事实不确定或需要人工补充，写入警告；没有则为空数组。`,
		req.Workspace.Name,
		req.Workspace.Type,
		emptyAs(req.Workspace.Industry, "未设置"),
		emptyAs(req.Workspace.Language, "zh-CN"),
		emptyAs(req.Workspace.Tone, "专业、清晰、克制"),
		req.ContentType,
		req.Skill.ID,
		req.Skill.Version,
		req.Skill.Contract,
		strings.Join(req.Keywords, "、"),
		string(publishFormat),
		string(chunks),
	)

	return PromptTranscript{System: system, User: user, Schema: schema}
}

func DraftJSONSchema() map[string]any {
	sectionSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"heading": map[string]any{"type": "string"},
			"body":    map[string]any{"type": "string"},
		},
		"required":             []string{"heading", "body"},
		"additionalProperties": false,
	}

	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title":            map[string]any{"type": "string"},
			"summary":          map[string]any{"type": "string"},
			"body":             map[string]any{"type": "string"},
			"keywords":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"sections":         map[string]any{"type": "array", "items": sectionSchema},
			"usedKnowledgeIds": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"warnings":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
		"required":             []string{"title", "summary", "body", "keywords", "sections", "usedKnowledgeIds", "warnings"},
		"additionalProperties": false,
	}
}

func knowledgeIDs(chunks []KnowledgeChunk) []string {
	ids := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		ids = append(ids, chunk.ID)
	}
	return ids
}

func emptyAs(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
