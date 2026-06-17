package xiaohongshu

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/model"
)

const LongArticleFormatID = "xiaohongshu_long_article"

type BrowserLongArticlePublisher struct {
	NodeBin       string
	ScriptPath    string
	ChromePath    string
	ActionTimeout time.Duration
}

func NewBrowserLongArticlePublisher() BrowserLongArticlePublisher {
	return BrowserLongArticlePublisher{
		NodeBin:       defaultNodeBin(),
		ScriptPath:    defaultBrowserPublishScriptPath(),
		ChromePath:    defaultChromePath(),
		ActionTimeout: 3 * time.Minute,
	}
}

func (BrowserLongArticlePublisher) PlatformType() string {
	return PlatformType
}

func (p BrowserLongArticlePublisher) Prepare(ctx context.Context, req publisher.PrepareRequest) (publisher.PreparedPost, error) {
	select {
	case <-ctx.Done():
		return publisher.PreparedPost{}, ctx.Err()
	default:
	}

	if req.Platform.Type != PlatformType {
		return publisher.PreparedPost{}, errors.New("media account is not a xiaohongshu account")
	}
	title := truncateTitle(xiaohongshuTitle(req.Content.Title), 64)
	body := xiaohongshuLongArticleBody(req.Content.Summary, req.Content.Body, req.Content.Keywords)
	if title == "" {
		return publisher.PreparedPost{}, errors.New("content title is required")
	}
	if body == "" {
		return publisher.PreparedPost{}, errors.New("content body is required")
	}

	hashtags := xiaohongshuHashtags(req.Content.Keywords)
	preparedAt := req.RequestedAt
	if preparedAt.IsZero() {
		preparedAt = time.Now().UTC()
	}

	copyBlocks := []publisher.CopyBlock{
		{Label: "长文标题", Value: title},
		{Label: "长文正文", Value: body},
	}
	if len(hashtags) > 0 {
		copyBlocks = append(copyBlocks, publisher.CopyBlock{Label: "话题", Value: strings.Join(hashtags, " ")})
	}

	return publisher.PreparedPost{
		Mode:            publisher.ModeBrowserAutomation,
		PlatformType:    PlatformType,
		PlatformName:    defaultString(req.Platform.Name, "小红书"),
		PublishFormatID: LongArticleFormatID,
		PublishMode:     "long_article",
		Title:           title,
		Body:            body,
		Hashtags:        hashtags,
		CopyBlocks:      copyBlocks,
		Checklist: []string{
			"确认标题不超过 64 字。",
			"确认正文为小红书长文纯文本，事实、案例和隐私信息已人工检查。",
			"确认已使用目标小红书账号完成浏览器登录。",
			"点击确认发布后，服务端会打开已登录浏览器会话并点击小红书发布按钮。",
		},
		Warnings:       longArticleWarnings(title, body),
		CharacterCount: len([]rune(body)),
		PreparedAt:     preparedAt,
	}, nil
}

func (p BrowserLongArticlePublisher) Publish(ctx context.Context, req publisher.PublishRequest) (publisher.PublishResult, error) {
	if strings.TrimSpace(req.PreparedPost.PublishFormatID) != LongArticleFormatID {
		return publisher.PublishResult{}, fmt.Errorf("unsupported xiaohongshu publish format: %s", req.PreparedPost.PublishFormatID)
	}
	if strings.TrimSpace(req.PreparedPost.Title) == "" {
		return publisher.PublishResult{}, errors.New("prepared title is required")
	}
	if strings.TrimSpace(req.PreparedPost.Body) == "" {
		return publisher.PublishResult{}, errors.New("prepared body is required")
	}

	profileDir := strings.TrimSpace(req.ProfileDir)
	if profileDir == "" {
		profileDir = RuntimeBrowserProfilePath(req.Workspace.ID, req.Account.ID)
	}
	if profileDir == "" {
		return publisher.PublishResult{}, errors.New("browser profile dir is required")
	}

	args := []string{
		p.scriptPath(),
		"--profile-dir", profileDir,
		"--title", req.PreparedPost.Title,
		"--body", req.PreparedPost.Body,
		"--publish-mode", "long_article",
	}
	if stateFile := strings.TrimSpace(req.StateFile); stateFile != "" {
		args = append(args, "--state-file", stateFile)
	}
	if chromePath := strings.TrimSpace(p.ChromePath); chromePath != "" {
		args = append(args, "--chrome-path", chromePath)
	}

	out, err := p.run(ctx, args...)
	if err != nil {
		return publisher.PublishResult{}, err
	}

	var scriptResult browserPublishResult
	if err := json.Unmarshal(out, &scriptResult); err != nil {
		return publisher.PublishResult{}, fmt.Errorf("decode xiaohongshu browser publish output: %w", err)
	}
	if scriptResult.Status == "" {
		scriptResult.Status = "submitted"
	}
	message := scriptResult.Message
	if message == "" {
		message = "小红书长文已通过浏览器提交。"
	}
	return publisher.PublishResult{
		Status:      scriptResult.Status,
		Message:     message,
		ExternalID:  scriptResult.ExternalID,
		ExternalURL: scriptResult.ExternalURL,
		RawResponse: map[string]any{
			"provider":       "playwright_browser",
			"platform":       PlatformType,
			"publishFormat":  LongArticleFormatID,
			"publishMode":    "long_article",
			"profileDir":     profileDir,
			"pageUrl":        scriptResult.PageURL,
			"screenshotPath": scriptResult.ScreenshotPath,
			"submittedAt":    scriptResult.SubmittedAt,
			"rawStatus":      scriptResult.RawStatus,
		},
	}, nil
}

type browserPublishResult struct {
	Status         string         `json:"status"`
	Message        string         `json:"message"`
	PageURL        string         `json:"pageUrl"`
	ExternalID     string         `json:"externalId"`
	ExternalURL    string         `json:"externalUrl"`
	ScreenshotPath string         `json:"screenshotPath"`
	SubmittedAt    string         `json:"submittedAt"`
	RawStatus      map[string]any `json:"rawStatus"`
}

func (p BrowserLongArticlePublisher) run(ctx context.Context, args ...string) ([]byte, error) {
	timeout := p.ActionTimeout
	if timeout <= 0 {
		timeout = 3 * time.Minute
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nodeBin := strings.TrimSpace(p.NodeBin)
	if nodeBin == "" {
		nodeBin = defaultNodeBin()
	}
	cmd := exec.CommandContext(runCtx, nodeBin, args...)
	cmd.Env = os.Environ()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("xiaohongshu browser publish failed: %s", message)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func (p BrowserLongArticlePublisher) scriptPath() string {
	if strings.TrimSpace(p.ScriptPath) != "" {
		return p.ScriptPath
	}
	return defaultBrowserPublishScriptPath()
}

func defaultBrowserPublishScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_XHS_BROWSER_PUBLISH_SCRIPT")); value != "" {
		return value
	}
	root := installRoot()
	if root == "" {
		return filepath.Join("scripts", "xiaohongshu-browser-publish.mjs")
	}
	return filepath.Join(root, "scripts", "xiaohongshu-browser-publish.mjs")
}

func xiaohongshuLongArticleBody(summary string, body string, keywords []string) string {
	value := xiaohongshuBody(summary, body)
	hashtags := strings.Join(xiaohongshuHashtags(keywords), " ")
	if hashtags != "" && !strings.Contains(value, "#") {
		value = strings.TrimSpace(value + "\n\n" + hashtags)
	}
	return strings.TrimSpace(value)
}

func truncateTitle(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func longArticleWarnings(title string, body string) []string {
	warnings := []string{}
	if len([]rune(title)) > 64 {
		warnings = append(warnings, "标题超过小红书长文 64 字限制，发布前会被截断或需要人工调整。")
	}
	if len([]rune(body)) > 8000 {
		warnings = append(warnings, "正文超过建议长度 8000 字，发布前建议精简。")
	}
	if len(warnings) == 0 {
		return []string{}
	}
	return warnings
}

func BrowserProfileMetadata(account model.MediaAccount, workspaceID string) (string, string) {
	profileDir := ""
	stateFile := ""
	if account.CredentialMeta != nil {
		profileDir = strings.TrimSpace(account.CredentialMeta["browserProfile"])
		stateFile = strings.TrimSpace(account.CredentialMeta["browserLoginStateFile"])
	}
	if profileDir == "" {
		profileDir = RuntimeBrowserProfilePath(workspaceID, account.ID)
	}
	if stateFile == "" && profileDir != "" {
		stateFile = BrowserLoginStateFile(profileDir)
	}
	return profileDir, stateFile
}
