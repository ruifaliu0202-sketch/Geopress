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

const DefaultContentMetadataURL = "https://creator.xiaohongshu.com/new/note-manager"

type ContentMetadataCollector interface {
	CollectContent(ctx context.Context, req ContentMetadataCollectRequest) (ContentMetadataCollectResult, error)
}

type ContentMetadataCollectRequest struct {
	WorkspaceID       string
	AccountID         string
	ProfileDir        string
	ExternalContentID string
	ExternalURL       string
	Title             string
	ContentID         string
	PublishJobID      string
	OutputFile        string
	DebugDir          string
	ScreenshotDir     string
	SettleDelay       time.Duration
	ActionTimeout     time.Duration
}

type ContentMetadataCollectResult struct {
	OK                bool                     `json:"ok"`
	Platform          string                   `json:"platform"`
	Status            string                   `json:"status"`
	ProfileDir        string                   `json:"profileDir"`
	ExternalContentID string                   `json:"externalContentId"`
	ExternalURL       string                   `json:"externalUrl"`
	Title             string                   `json:"title"`
	ContentID         string                   `json:"contentId"`
	PublishJobID      string                   `json:"publishJobId"`
	NoteManagerURL    string                   `json:"noteManagerUrl"`
	PageURL           string                   `json:"pageUrl"`
	PageTitle         string                   `json:"pageTitle"`
	CapturedAt        time.Time                `json:"capturedAt"`
	DataSource        string                   `json:"dataSource"`
	LoginState        map[string]any           `json:"loginState"`
	Metadata          *ContentMetadataSnapshot `json:"metadata"`
	Diagnostics       map[string]any           `json:"diagnostics"`
	Error             string                   `json:"error,omitempty"`
}

type ContentMetadataSnapshot struct {
	ExternalContentID string              `json:"externalContentId"`
	ExternalURL       string              `json:"externalUrl"`
	Title             string              `json:"title"`
	Status            string              `json:"status"`
	StatusText        string              `json:"statusText"`
	PublishedAt       string              `json:"publishedAt"`
	CapturedAt        time.Time           `json:"capturedAt"`
	Confidence        string              `json:"confidence"`
	MatchStrategy     string              `json:"matchStrategy"`
	MatchScore        float64             `json:"matchScore"`
	MatchKeyword      string              `json:"matchKeyword"`
	SourceURL         string              `json:"sourceUrl"`
	Metrics           ContentMetricValues `json:"metrics"`
	RawMetrics        map[string]any      `json:"rawMetrics"`
}

type ContentMetricValues struct {
	ImpressionCount int     `json:"impressionCount"`
	ViewCount       int     `json:"viewCount"`
	LikeCount       int     `json:"likeCount"`
	CommentCount    int     `json:"commentCount"`
	ShareCount      int     `json:"shareCount"`
	FavoriteCount   int     `json:"favoriteCount"`
	ClickCount      int     `json:"clickCount"`
	EngagementRate  float64 `json:"engagementRate"`
}

type PlaywrightContentMetadataCollector struct {
	NodeBin       string
	ScriptPath    string
	ChromePath    string
	ActionTimeout time.Duration
}

func NewPlaywrightContentMetadataCollector() PlaywrightContentMetadataCollector {
	return PlaywrightContentMetadataCollector{
		NodeBin:       defaultNodeBin(),
		ScriptPath:    defaultContentMetadataScriptPath(),
		ChromePath:    defaultChromePath(),
		ActionTimeout: defaultDurationSeconds("GEOPRESS_XHS_CONTENT_METADATA_TIMEOUT_SECONDS", 90*time.Second),
	}
}

func (c PlaywrightContentMetadataCollector) CollectContent(ctx context.Context, req ContentMetadataCollectRequest) (ContentMetadataCollectResult, error) {
	profileDir := strings.TrimSpace(req.ProfileDir)
	if profileDir == "" {
		profileDir = RuntimeBrowserProfilePath(req.WorkspaceID, req.AccountID)
	}
	if profileDir == "" {
		return ContentMetadataCollectResult{}, errors.New("browser profile dir is required")
	}
	externalContentID := strings.TrimSpace(req.ExternalContentID)
	if externalContentID == "" {
		return ContentMetadataCollectResult{}, errors.New("external content id is required")
	}

	args := []string{
		c.scriptPath(),
		"--profile-dir", profileDir,
		"--external-content-id", externalContentID,
	}
	if externalURL := strings.TrimSpace(req.ExternalURL); externalURL != "" {
		args = append(args, "--external-url", externalURL)
	}
	if title := strings.TrimSpace(req.Title); title != "" {
		args = append(args, "--title", title)
	}
	if contentID := strings.TrimSpace(req.ContentID); contentID != "" {
		args = append(args, "--content-id", contentID)
	}
	if publishJobID := strings.TrimSpace(req.PublishJobID); publishJobID != "" {
		args = append(args, "--publish-job-id", publishJobID)
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
	if screenshotDir := strings.TrimSpace(req.ScreenshotDir); screenshotDir != "" {
		args = append(args, "--screenshot-dir", screenshotDir)
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
		var result ContentMetadataCollectResult
		if decodeErr := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); decodeErr == nil && result.Status != "" {
			if result.ProfileDir == "" {
				result.ProfileDir = profileDir
			}
			return result, fmt.Errorf("xiaohongshu content metadata status: %s", defaultString(result.Status, "unknown"))
		}
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = err.Error()
		}
		return ContentMetadataCollectResult{}, fmt.Errorf("xiaohongshu content metadata collection failed: %s", message)
	}

	var result ContentMetadataCollectResult
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &result); err != nil {
		return ContentMetadataCollectResult{}, fmt.Errorf("decode xiaohongshu content metadata output: %w", err)
	}
	if result.ProfileDir == "" {
		result.ProfileDir = profileDir
	}
	if !result.OK && result.Status != "pending_reconcile" {
		return result, fmt.Errorf("xiaohongshu content metadata status: %s", defaultString(result.Status, "unknown"))
	}
	return result, nil
}

func (c PlaywrightContentMetadataCollector) scriptPath() string {
	if strings.TrimSpace(c.ScriptPath) != "" {
		return c.ScriptPath
	}
	return defaultContentMetadataScriptPath()
}

func defaultContentMetadataScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_XHS_CONTENT_METADATA_SCRIPT")); value != "" {
		return value
	}
	root := installRoot()
	if root == "" {
		return filepath.Join("scripts", "xiaohongshu-content-metadata.mjs")
	}
	return filepath.Join(root, "scripts", "xiaohongshu-content-metadata.mjs")
}
