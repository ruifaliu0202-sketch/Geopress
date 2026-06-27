package toutiao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"geopress/backend/internal/integration/browserplatform"
)

const PlatformType = "toutiao"
const DefaultAccountMetadataURL = "https://mp.toutiao.com/profile_v4/"

type AccountMetadataCollector interface {
	Collect(ctx context.Context, req AccountMetadataCollectRequest) (AccountMetadataCollectResult, error)
}

type AccountMetadataCollectRequest struct {
	WorkspaceID   string
	AccountID     string
	ProfileDir    string
	MetadataURL   string
	OutputFile    string
	DebugDir      string
	SettleDelay   time.Duration
	ActionTimeout time.Duration
}

type AccountMetadataCollectResult struct {
	OK          bool                    `json:"ok"`
	Platform    string                  `json:"platform"`
	Status      string                  `json:"status"`
	ProfileDir  string                  `json:"profileDir"`
	MetadataURL string                  `json:"metadataUrl"`
	PageURL     string                  `json:"pageUrl"`
	Title       string                  `json:"title"`
	CapturedAt  time.Time               `json:"capturedAt"`
	DataSource  string                  `json:"dataSource"`
	Selectors   map[string]any          `json:"selectors"`
	LoginState  map[string]any          `json:"loginState"`
	Metadata    AccountMetadataSnapshot `json:"metadata"`
	Diagnostics map[string]any          `json:"diagnostics"`
	Error       string                  `json:"error,omitempty"`
}

type AccountMetadataSnapshot struct {
	DisplayName        string                     `json:"displayName"`
	AvatarURL          string                     `json:"avatarUrl"`
	UserID             string                     `json:"userId"`
	MediaID            string                     `json:"mediaId"`
	ProfileURL         string                     `json:"profileUrl"`
	IsCreator          bool                       `json:"isCreator"`
	AuthType           *float64                   `json:"authType"`
	FollowerCount      int                        `json:"followerCount"`
	ContentCount       int                        `json:"contentCount"`
	TotalReadPlayCount int                        `json:"totalReadPlayCount"`
	YesterdayReadCount int                        `json:"yesterdayReadCount"`
	YesterdayPlayCount int                        `json:"yesterdayPlayCount"`
	YesterdayFansCount int                        `json:"yesterdayFansCount"`
	TotalIncome        float64                    `json:"totalIncome"`
	VisibleMetrics     map[string]CollectedMetric `json:"visibleMetrics"`
	VisibleAccountLink string                     `json:"visibleAccountLink"`
	UserInfo           map[string]any             `json:"userInfo"`
	HomeStatistic      map[string]any             `json:"homeStatistic"`
	WorksSummary       map[string]any             `json:"worksSummary"`
	WorksListSummary   map[string]any             `json:"worksListSummary"`
}

type CollectedMetric struct {
	Raw       string   `json:"raw"`
	ValueText string   `json:"valueText"`
	Value     *float64 `json:"value"`
}

type PlaywrightAccountMetadataCollector struct {
	NodeBin       string
	ScriptPath    string
	ChromePath    string
	MetadataURL   string
	ActionTimeout time.Duration
}

func NewPlaywrightAccountMetadataCollector() PlaywrightAccountMetadataCollector {
	return PlaywrightAccountMetadataCollector{
		NodeBin:       defaultNodeBin(),
		ScriptPath:    defaultAccountMetadataScriptPath(),
		ChromePath:    defaultChromePath(),
		MetadataURL:   DefaultAccountMetadataURL,
		ActionTimeout: defaultDurationSeconds("GEOPRESS_TOUTIAO_METADATA_TIMEOUT_SECONDS", 90*time.Second),
	}
}

func (c PlaywrightAccountMetadataCollector) Collect(ctx context.Context, req AccountMetadataCollectRequest) (AccountMetadataCollectResult, error) {
	profileDir := strings.TrimSpace(req.ProfileDir)
	if profileDir == "" {
		profileDir = browserplatform.RuntimeBrowserProfilePath(req.WorkspaceID, req.AccountID)
	}
	if profileDir == "" {
		return AccountMetadataCollectResult{}, errors.New("browser profile dir is required")
	}

	args := []string{
		c.scriptPath(),
		"--profile-dir", profileDir,
		"--metadata-url", firstNonEmpty(req.MetadataURL, c.MetadataURL, DefaultAccountMetadataURL),
	}
	if chromePath := strings.TrimSpace(c.ChromePath); chromePath != "" {
		args = append(args, "--chrome-path", chromePath)
	}
	if outputFile := strings.TrimSpace(req.OutputFile); outputFile != "" {
		args = append(args, "--output", outputFile)
	}
	if debugDir := strings.TrimSpace(req.DebugDir); debugDir != "" {
		args = append(args, "--debug-dir", debugDir)
	}
	if req.SettleDelay > 0 {
		args = append(args, "--settle-ms", fmt.Sprintf("%d", req.SettleDelay.Milliseconds()))
	}

	timeout := c.ActionTimeout
	if req.ActionTimeout > 0 {
		timeout = req.ActionTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nodeBin := strings.TrimSpace(c.NodeBin)
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
		var result AccountMetadataCollectResult
		if decodeErr := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); decodeErr == nil && result.Status != "" {
			if result.ProfileDir == "" {
				result.ProfileDir = profileDir
			}
			return result, fmt.Errorf("toutiao account metadata status: %s", defaultString(result.Status, "unknown"))
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return AccountMetadataCollectResult{}, fmt.Errorf("toutiao account metadata collection failed: %s", message)
	}

	var result AccountMetadataCollectResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		return AccountMetadataCollectResult{}, fmt.Errorf("decode toutiao metadata output: %w", err)
	}
	if result.ProfileDir == "" {
		result.ProfileDir = profileDir
	}
	if !result.OK {
		return result, fmt.Errorf("toutiao account metadata status: %s", defaultString(result.Status, "unknown"))
	}
	return result, nil
}

func (c PlaywrightAccountMetadataCollector) scriptPath() string {
	if strings.TrimSpace(c.ScriptPath) != "" {
		return c.ScriptPath
	}
	return defaultAccountMetadataScriptPath()
}

func defaultAccountMetadataScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_TOUTIAO_ACCOUNT_METADATA_SCRIPT")); value != "" {
		return value
	}
	return filepath.Join(installRoot(), "scripts", "toutiao-account-metadata.mjs")
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

func defaultDurationSeconds(envName string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(envName))
	if value == "" {
		return fallback
	}
	duration, err := time.ParseDuration(value)
	if err == nil {
		return duration
	}
	if parsed, parseErr := strconv.Atoi(value); parseErr == nil {
		return time.Duration(parsed) * time.Second
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
