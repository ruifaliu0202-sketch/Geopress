package browserplatform

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

type Config struct {
	PlatformType    string
	PlatformName    string
	PublishFormatID string
	PublishMode     string
	PublishScript   string
	PublishURL      string
	TitleMaxRunes   int
}

type Publisher struct {
	config        Config
	NodeBin       string
	ChromePath    string
	ActionTimeout time.Duration
}

func NewPublisher(config Config) Publisher {
	if config.PublishFormatID == "" {
		config.PublishFormatID = "article"
	}
	if config.PublishMode == "" {
		config.PublishMode = "article"
	}
	if config.TitleMaxRunes <= 0 {
		config.TitleMaxRunes = 64
	}
	return Publisher{
		config:        config,
		NodeBin:       defaultNodeBin(),
		ChromePath:    defaultChromePath(),
		ActionTimeout: 3 * time.Minute,
	}
}

func (p Publisher) PlatformType() string {
	return p.config.PlatformType
}

func (p Publisher) Prepare(ctx context.Context, req publisher.PrepareRequest) (publisher.PreparedPost, error) {
	select {
	case <-ctx.Done():
		return publisher.PreparedPost{}, ctx.Err()
	default:
	}

	if req.Platform.Type != p.config.PlatformType {
		return publisher.PreparedPost{}, fmt.Errorf("media account is not a %s account", p.config.PlatformType)
	}
	title := truncateRunes(compactInline(req.Content.Title), p.config.TitleMaxRunes)
	body := compactParagraphs(strings.TrimSpace(strings.Join([]string{req.Content.Summary, req.Content.Body}, "\n\n")))
	if title == "" {
		return publisher.PreparedPost{}, errors.New("content title is required")
	}
	if body == "" {
		body = title
	}

	hashtags := hashtags(req.Content.Keywords, 5)
	topicText := strings.Join(hashtags, " ")
	if topicText != "" && !strings.Contains(body, "#") {
		body = strings.TrimSpace(body + "\n\n" + topicText)
	}

	copyBlocks := []publisher.CopyBlock{
		{Label: "标题", Value: title},
		{Label: "正文", Value: body},
	}
	if topicText != "" {
		copyBlocks = append(copyBlocks, publisher.CopyBlock{Label: "话题/标签", Value: topicText})
	}

	preparedAt := req.RequestedAt
	if preparedAt.IsZero() {
		preparedAt = time.Now().UTC()
	}
	platformName := strings.TrimSpace(req.Platform.Name)
	if platformName == "" {
		platformName = p.config.PlatformName
	}

	return publisher.PreparedPost{
		Mode:            publisher.ModeBrowserAutomation,
		PlatformType:    p.config.PlatformType,
		PlatformName:    platformName,
		PublishFormatID: defaultString(req.PublishFormatID, p.config.PublishFormatID),
		PublishMode:     p.config.PublishMode,
		Title:           title,
		Body:            body,
		Hashtags:        hashtags,
		CopyBlocks:      copyBlocks,
		Checklist: []string{
			fmt.Sprintf("确认已使用目标%s账号完成浏览器登录。", platformName),
			"确认标题、正文、封面、分类和平台合规提示。",
			fmt.Sprintf("点击确认发布后，服务端会打开已登录浏览器会话并点击%s发布按钮。", platformName),
		},
		Warnings:       browserPublishWarnings(platformName, title, body, p.config.TitleMaxRunes),
		CharacterCount: len([]rune(body)),
		PreparedAt:     preparedAt,
	}, nil
}

func (p Publisher) Publish(ctx context.Context, req publisher.PublishRequest) (publisher.PublishResult, error) {
	if strings.TrimSpace(req.PreparedPost.PublishFormatID) != p.config.PublishFormatID {
		return publisher.PublishResult{}, fmt.Errorf("unsupported %s publish format: %s", p.config.PlatformType, req.PreparedPost.PublishFormatID)
	}
	if strings.TrimSpace(req.PreparedPost.Title) == "" {
		return publisher.PublishResult{}, errors.New("prepared title is required")
	}
	if strings.TrimSpace(req.PreparedPost.Body) == "" {
		return publisher.PublishResult{}, errors.New("prepared body is required")
	}
	profileDir := strings.TrimSpace(req.ProfileDir)
	if profileDir == "" {
		return publisher.PublishResult{}, errors.New("browser profile dir is required")
	}

	args := []string{
		p.scriptPath(),
		"--profile-dir", profileDir,
		"--title", req.PreparedPost.Title,
		"--body", req.PreparedPost.Body,
		"--publish-mode", p.config.PublishMode,
		"--publish-url", p.config.PublishURL,
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
		return publisher.PublishResult{}, fmt.Errorf("decode %s browser publish output: %w", p.config.PlatformType, err)
	}
	if scriptResult.Status == "" {
		scriptResult.Status = "submitted"
	}
	message := scriptResult.Message
	if message == "" {
		message = fmt.Sprintf("%s文章已通过浏览器提交。", p.config.PlatformName)
	}
	return publisher.PublishResult{
		Status:      scriptResult.Status,
		Message:     message,
		ExternalID:  scriptResult.ExternalID,
		ExternalURL: scriptResult.ExternalURL,
		RawResponse: map[string]any{
			"provider":           "playwright_browser",
			"platform":           p.config.PlatformType,
			"publishFormat":      p.config.PublishFormatID,
			"publishMode":        p.config.PublishMode,
			"profileDir":         profileDir,
			"pageUrl":            scriptResult.PageURL,
			"screenshotPath":     scriptResult.ScreenshotPath,
			"submittedAt":        scriptResult.SubmittedAt,
			"publishIdentity":    scriptResult.RawStatus["publishIdentity"],
			"networkCapturePath": scriptResult.RawStatus["networkCapturePath"],
			"networkCandidates":  scriptResult.RawStatus["networkCandidates"],
			"rawStatus":          scriptResult.RawStatus,
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

func BrowserLoginStateFile(profileDir string) string {
	if strings.TrimSpace(profileDir) == "" {
		return ""
	}
	return filepath.Join(profileDir, "geopress-login-state.json")
}

func RuntimeBrowserProfilePath(workspaceID, accountID string) string {
	root := runtimeRoot()
	if root == "" {
		return filepath.Join("runtime", "browser-profiles", workspaceID, accountID)
	}
	return filepath.Join(root, "runtime", "browser-profiles", workspaceID, accountID)
}

func (p Publisher) run(ctx context.Context, args ...string) ([]byte, error) {
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
		return nil, fmt.Errorf("%s browser publish failed: %s", p.config.PlatformType, message)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func (p Publisher) scriptPath() string {
	if strings.TrimSpace(p.config.PublishScript) != "" {
		return p.config.PublishScript
	}
	return filepath.Join(installRoot(), "scripts", fmt.Sprintf("%s-browser-publish.mjs", p.config.PlatformType))
}

func browserPublishWarnings(platformName string, title string, body string, titleMaxRunes int) []string {
	warnings := []string{}
	if len([]rune(title)) > titleMaxRunes {
		warnings = append(warnings, fmt.Sprintf("标题超过%s建议长度，发布前会被截断或需要人工调整。", platformName))
	}
	if len([]rune(body)) > 10000 {
		warnings = append(warnings, "正文超过建议长度 10000 字，发布前建议精简。")
	}
	return warnings
}

func compactInline(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func compactParagraphs(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	lines := strings.Split(value, "\n")
	paragraphs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		paragraphs = append(paragraphs, line)
	}
	return strings.Join(paragraphs, "\n\n")
}

func hashtags(values []string, maxCount int) []string {
	seen := map[string]bool{}
	result := make([]string, 0, maxCount)
	for _, value := range values {
		tag := normalizeHashtag(value)
		if tag == "" || seen[tag] {
			continue
		}
		seen[tag] = true
		result = append(result, tag)
		if len(result) == maxCount {
			break
		}
	}
	return result
}

func normalizeHashtag(value string) string {
	value = strings.TrimSpace(strings.TrimPrefix(value, "#"))
	value = strings.Trim(value, " \t\r\n,，.。!！?？;；:：")
	value = strings.Join(strings.Fields(value), "")
	if value == "" {
		return ""
	}
	return "#" + value
}

func truncateRunes(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes])
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultNodeBin() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_NODE_BIN")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("NODE_BIN")); value != "" {
		return value
	}
	return "node"
}

func defaultChromePath() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_CHROME_PATH")); value != "" {
		return value
	}
	candidates := []string{
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func installRoot() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_INSTALL_ROOT")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_PROJECT_ROOT")); value != "" {
		return value
	}
	if hasScriptPath("/opt/geopress") {
		return "/opt/geopress"
	}
	return workingRoot()
}

func runtimeRoot() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_RUNTIME_ROOT")); value != "" {
		return value
	}
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_PROJECT_ROOT")); value != "" {
		return value
	}
	return workingRoot()
}

func hasScriptPath(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(root, "scripts"))
	return err == nil
}

func workingRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for current := wd; current != "." && current != string(filepath.Separator); current = filepath.Dir(current) {
		if _, err := os.Stat(filepath.Join(current, "scripts")); err == nil {
			return current
		}
	}
	return wd
}
