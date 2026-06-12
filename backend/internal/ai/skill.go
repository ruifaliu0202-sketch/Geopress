package ai

import "strings"

type WritingSkill struct {
	ID          string `json:"id"`
	Version     string `json:"version"`
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Contract    string `json:"contract"`
}

func SelectWritingSkill(contentType string) WritingSkill {
	switch strings.TrimSpace(contentType) {
	case "brief":
		return WritingSkill{
			ID:          "brief-actionable",
			Version:     "2026-06-12",
			Name:        "短文创作",
			ContentType: "brief",
			Contract:    "输出一篇适合社媒或短博客的中文短文，观点直接，段落紧凑，保留可执行建议。",
		}
	case "case_study":
		return WritingSkill{
			ID:          "case-study-practical",
			Version:     "2026-06-12",
			Name:        "案例稿创作",
			ContentType: "case_study",
			Contract:    "输出案例稿，包含背景、挑战、做法、结果和可复用经验，避免编造具体客户和数据。",
		}
	case "product_intro":
		return WritingSkill{
			ID:          "product-intro-clear",
			Version:     "2026-06-12",
			Name:        "产品介绍创作",
			ContentType: "product_intro",
			Contract:    "输出产品介绍稿，突出受众、场景、能力、价值和下一步行动，语气可信克制。",
		}
	default:
		return WritingSkill{
			ID:          "article-structured",
			Version:     "2026-06-12",
			Name:        "长文章创作",
			ContentType: "article",
			Contract:    "输出一篇结构清晰的中文长文章，包含引入、关键观点、执行步骤和总结。",
		}
	}
}
