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
)

const DefaultAccountMetadataURL = "https://creator.xiaohongshu.com/new/home"

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
	Selectors   map[string]string       `json:"selectors"`
	LoginState  map[string]any          `json:"loginState"`
	Metadata    AccountMetadataSnapshot `json:"metadata"`
	Error       string                  `json:"error,omitempty"`
}

type AccountMetadataSnapshot struct {
	DisplayName            string                     `json:"displayName"`
	AvatarURL              string                     `json:"avatarUrl"`
	AccountStatusImageURL  string                     `json:"accountStatusImageUrl"`
	AccountStatusAlt       string                     `json:"accountStatusAlt"`
	RedAccountID           string                     `json:"redAccountId"`
	RawRedAccountText      string                     `json:"rawRedAccountText"`
	FollowingCount         int                        `json:"followingCount"`
	FollowerCount          int                        `json:"followerCount"`
	LikedAndFavoritedCount int                        `json:"likedAndFavoritedCount"`
	AccountStats           map[string]CollectedMetric `json:"accountStats"`
	OverviewMetrics        map[string]CollectedMetric `json:"overviewMetrics"`
	AccountCardText        string                     `json:"accountCardText"`
	ProfileBaseText        string                     `json:"profileBaseText"`
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
		ActionTimeout: defaultDurationSeconds("GEOPRESS_XHS_METADATA_TIMEOUT_SECONDS", 90*time.Second),
	}
}

func (c PlaywrightAccountMetadataCollector) Collect(ctx context.Context, req AccountMetadataCollectRequest) (AccountMetadataCollectResult, error) {
	profileDir := strings.TrimSpace(req.ProfileDir)
	if profileDir == "" {
		profileDir = RuntimeBrowserProfilePath(req.WorkspaceID, req.AccountID)
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
			return result, fmt.Errorf("xiaohongshu account metadata status: %s", defaultString(result.Status, "unknown"))
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return AccountMetadataCollectResult{}, fmt.Errorf("xiaohongshu account metadata collection failed: %s", message)
	}

	var result AccountMetadataCollectResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		return AccountMetadataCollectResult{}, fmt.Errorf("decode xiaohongshu metadata output: %w", err)
	}
	if result.ProfileDir == "" {
		result.ProfileDir = profileDir
	}
	if !result.OK {
		return result, fmt.Errorf("xiaohongshu account metadata status: %s", defaultString(result.Status, "unknown"))
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
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_XHS_ACCOUNT_METADATA_SCRIPT")); value != "" {
		return value
	}
	return filepath.Join(installRoot(), "scripts", "xiaohongshu-account-metadata.mjs")
}
