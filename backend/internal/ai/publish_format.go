package ai

import "strings"

const (
	FormatGenericArticle         = "article"
	FormatXiaohongshuLongArticle = "xiaohongshu_long_article"
)

func SelectPublishFormat(contentType string) PublishFormat {
	switch strings.TrimSpace(contentType) {
	case FormatXiaohongshuLongArticle:
		return PublishFormat{
			ID:            FormatXiaohongshuLongArticle,
			PlatformType:  "xiaohongshu",
			PlatformName:  "小红书",
			Mode:          "long_article",
			Name:          "小红书长文",
			Description:   "适合小红书创作服务平台“写长文”的发布格式，产物为纯文本长文，不依赖图片素材。",
			TitleMaxRunes: 64,
			BodyMaxRunes:  8000,
			RequiredFields: []string{
				"title",
				"body",
				"keywords",
			},
			Structure: []string{
				"开头 1 段直接点题，说明读者为什么要继续看。",
				"正文使用 3 到 6 个短小小节，每节有清晰小标题。",
				"每节包含具体方法、判断标准、注意事项或示例，不写空泛口号。",
				"结尾给出可执行清单或下一步建议。",
				"最后自然附上 2 到 5 个相关话题标签。",
			},
			StyleGuidelines: []string{
				"中文表达，口语化但不夸张。",
				"段落短，适合移动端阅读。",
				"保留真实边界，不编造数据、客户案例、平台结果。",
				"标题不要超过 64 个中文字符。",
				"正文不要包含 Markdown 代码块。",
			},
			ValidationRules: []string{
				"标题不能为空且不超过 64 个字符。",
				"正文不能为空。",
				"正文长度建议控制在 8000 字以内。",
				"话题标签从关键词中提炼，不超过 5 个。",
				"知识库证据不足时必须在 warnings 里说明。",
			},
			AutomationChannel: "playwright_xiaohongshu_long_article",
		}
	default:
		return PublishFormat{
			ID:            FormatGenericArticle,
			PlatformType:  "",
			PlatformName:  "通用",
			Mode:          "article",
			Name:          "通用长文章",
			Description:   "通用结构化长文章草稿。",
			TitleMaxRunes: 80,
			BodyMaxRunes:  12000,
			RequiredFields: []string{
				"title",
				"summary",
				"body",
				"keywords",
			},
			Structure: []string{
				"引入",
				"关键观点",
				"执行步骤",
				"总结",
			},
			StyleGuidelines: []string{
				"结构清晰，观点具体。",
				"避免编造事实。",
			},
			ValidationRules: []string{
				"标题、摘要、正文、关键词不能为空。",
			},
			AutomationChannel: "manual_or_platform_specific",
		}
	}
}
