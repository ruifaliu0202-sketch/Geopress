package xiaohongshu

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultLoginURL      = "https://creator.xiaohongshu.com/login"
	DefaultLoginSelector = ".sso-login-wrapper img, canvas, img"
)

type BrowserLoginService interface {
	Start(ctx context.Context, req BrowserLoginStartRequest) (BrowserLoginStartResult, error)
	Complete(ctx context.Context, req BrowserLoginCompleteRequest) (BrowserLoginCompleteResult, error)
}

type BrowserLoginStartRequest struct {
	WorkspaceID string
	AccountID   string
	SessionID   string
	ProfileDir  string
	LoginURL    string
	StateFile   string
}

type BrowserLoginStartResult struct {
	SessionID        string
	LoginURL         string
	PageURL          string
	QRScreenshotData string
	QRSelector       string
	ProfileDir       string
	StateFile        string
	StartedAt        time.Time
	RawStatus        map[string]any
	AlreadyLoggedIn  bool
}

type BrowserLoginCompleteRequest struct {
	WorkspaceID string
	AccountID   string
	SessionID   string
	ProfileDir  string
	LoginURL    string
	StateFile   string
}

type BrowserLoginCompleteResult struct {
	SessionID   string
	PageURL     string
	ProfileDir  string
	StateFile   string
	LoggedIn    bool
	CompletedAt time.Time
	RawStatus   map[string]any
}

type PlaywrightBrowserLoginService struct {
	NodeBin       string
	ScriptPath    string
	ChromePath    string
	LoginURL      string
	QRSelector    string
	ActionTimeout time.Duration
}

func NewPlaywrightBrowserLoginService() PlaywrightBrowserLoginService {
	return PlaywrightBrowserLoginService{
		NodeBin:       defaultNodeBin(),
		ScriptPath:    defaultBrowserLoginScriptPath(),
		ChromePath:    defaultChromePath(),
		LoginURL:      DefaultLoginURL,
		QRSelector:    DefaultLoginSelector,
		ActionTimeout: 30 * time.Second,
	}
}

func (s PlaywrightBrowserLoginService) Start(ctx context.Context, req BrowserLoginStartRequest) (BrowserLoginStartResult, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return BrowserLoginStartResult{}, errors.New("session id is required")
	}
	if strings.TrimSpace(req.ProfileDir) == "" {
		return BrowserLoginStartResult{}, errors.New("browser profile dir is required")
	}
	args := s.baseArgs("watch", req.SessionID, req.ProfileDir, firstNonEmpty(req.LoginURL, s.LoginURL, DefaultLoginURL), req.StateFile)
	out, err := s.startWatcher(ctx, args...)
	if err != nil {
		return BrowserLoginStartResult{}, err
	}

	var result BrowserLoginStartResult
	if err := json.Unmarshal(out, &result); err != nil {
		return BrowserLoginStartResult{}, fmt.Errorf("decode playwright start output: %w", err)
	}
	if result.SessionID == "" {
		result.SessionID = req.SessionID
	}
	if result.ProfileDir == "" {
		result.ProfileDir = req.ProfileDir
	}
	if result.StateFile == "" {
		result.StateFile = BrowserLoginStateFile(req.ProfileDir)
	}
	if result.LoginURL == "" {
		result.LoginURL = firstNonEmpty(req.LoginURL, s.LoginURL, DefaultLoginURL)
	}
	if result.StartedAt.IsZero() {
		result.StartedAt = time.Now().UTC()
	}
	if result.QRScreenshotData == "" && !result.AlreadyLoggedIn {
		return BrowserLoginStartResult{}, errors.New("playwright did not find a login QR image")
	}
	return result, nil
}

func (s PlaywrightBrowserLoginService) Complete(ctx context.Context, req BrowserLoginCompleteRequest) (BrowserLoginCompleteResult, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return BrowserLoginCompleteResult{}, errors.New("session id is required")
	}
	if strings.TrimSpace(req.ProfileDir) == "" {
		return BrowserLoginCompleteResult{}, errors.New("browser profile dir is required")
	}
	if result, ok, err := readLoginState(req.StateFile, req.ProfileDir); err != nil {
		return BrowserLoginCompleteResult{}, err
	} else if ok {
		if result.SessionID == "" {
			result.SessionID = req.SessionID
		}
		if result.ProfileDir == "" {
			result.ProfileDir = req.ProfileDir
		}
		if result.StateFile == "" {
			result.StateFile = BrowserLoginStateFile(req.ProfileDir)
		}
		if result.CompletedAt.IsZero() {
			result.CompletedAt = time.Now().UTC()
		}
		if !result.LoggedIn {
			return BrowserLoginCompleteResult{}, errors.New("xiaohongshu login is not confirmed yet")
		}
		return result, nil
	}

	args := s.baseArgs("complete", req.SessionID, req.ProfileDir, firstNonEmpty(req.LoginURL, s.LoginURL, DefaultLoginURL), req.StateFile)
	out, err := s.run(ctx, args...)
	if err != nil {
		return BrowserLoginCompleteResult{}, err
	}

	var result BrowserLoginCompleteResult
	if err := json.Unmarshal(out, &result); err != nil {
		return BrowserLoginCompleteResult{}, fmt.Errorf("decode playwright complete output: %w", err)
	}
	if result.SessionID == "" {
		result.SessionID = req.SessionID
	}
	if result.ProfileDir == "" {
		result.ProfileDir = req.ProfileDir
	}
	if result.StateFile == "" {
		result.StateFile = BrowserLoginStateFile(req.ProfileDir)
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now().UTC()
	}
	if !result.LoggedIn {
		return BrowserLoginCompleteResult{}, errors.New("xiaohongshu login is not confirmed yet")
	}
	return result, nil
}

func (s PlaywrightBrowserLoginService) baseArgs(action string, sessionID string, profileDir string, loginURL string, stateFile string) []string {
	args := []string{
		s.scriptPath(),
		"--action", action,
		"--session-id", sessionID,
		"--profile-dir", profileDir,
		"--login-url", loginURL,
		"--qr-selector", firstNonEmpty(s.QRSelector, DefaultLoginSelector),
		"--state-file", firstNonEmpty(stateFile, BrowserLoginStateFile(profileDir)),
	}
	if chromePath := strings.TrimSpace(s.ChromePath); chromePath != "" {
		args = append(args, "--chrome-path", chromePath)
	}
	return args
}

func (s PlaywrightBrowserLoginService) startWatcher(ctx context.Context, args ...string) ([]byte, error) {
	nodeBin := strings.TrimSpace(s.NodeBin)
	if nodeBin == "" {
		nodeBin = defaultNodeBin()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	cmd := exec.Command(nodeBin, args...)
	cmd.Env = os.Environ()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("start playwright browser watcher failed: %s", message)
	}

	line, err := readFirstLine(stdout, 20*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("playwright browser watcher did not return QR state: %s", message)
	}
	go func() {
		_ = cmd.Wait()
	}()
	return line, nil
}

func readFirstLine(reader io.Reader, timeout time.Duration) ([]byte, error) {
	type result struct {
		line []byte
		err  error
	}
	done := make(chan result, 1)
	go func() {
		scanner := bufio.NewScanner(reader)
		if scanner.Scan() {
			done <- result{line: bytes.TrimSpace(scanner.Bytes())}
			return
		}
		if err := scanner.Err(); err != nil {
			done <- result{err: err}
			return
		}
		done <- result{err: errors.New("watcher exited before writing initial state")}
	}()

	select {
	case result := <-done:
		return result.line, result.err
	case <-time.After(timeout):
		return nil, errors.New("timeout waiting for watcher initial state")
	}
}

func readLoginState(stateFile string, profileDir string) (BrowserLoginCompleteResult, bool, error) {
	statePath := firstNonEmpty(stateFile, BrowserLoginStateFile(profileDir))
	if statePath == "" {
		return BrowserLoginCompleteResult{}, false, nil
	}
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return BrowserLoginCompleteResult{}, false, nil
	}
	if err != nil {
		return BrowserLoginCompleteResult{}, false, err
	}

	var state struct {
		SessionID   string         `json:"sessionId"`
		PageURL     string         `json:"pageUrl"`
		ProfileDir  string         `json:"profileDir"`
		StateFile   string         `json:"stateFile"`
		LoggedIn    bool           `json:"loggedIn"`
		CompletedAt time.Time      `json:"completedAt"`
		RawStatus   map[string]any `json:"rawStatus"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return BrowserLoginCompleteResult{}, false, err
	}
	return BrowserLoginCompleteResult{
		SessionID:   state.SessionID,
		PageURL:     state.PageURL,
		ProfileDir:  state.ProfileDir,
		StateFile:   state.StateFile,
		LoggedIn:    state.LoggedIn,
		CompletedAt: state.CompletedAt,
		RawStatus:   state.RawStatus,
	}, true, nil
}

func BrowserLoginStateFile(profileDir string) string {
	if strings.TrimSpace(profileDir) == "" {
		return ""
	}
	return filepath.Join(profileDir, "geopress-login-state.json")
}

func (s PlaywrightBrowserLoginService) run(ctx context.Context, args ...string) ([]byte, error) {
	timeout := s.ActionTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	nodeBin := strings.TrimSpace(s.NodeBin)
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
		return nil, fmt.Errorf("playwright browser login failed: %s", message)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil
}

func (s PlaywrightBrowserLoginService) scriptPath() string {
	if strings.TrimSpace(s.ScriptPath) != "" {
		return s.ScriptPath
	}
	return defaultBrowserLoginScriptPath()
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

func defaultBrowserLoginScriptPath() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_XHS_BROWSER_LOGIN_SCRIPT")); value != "" {
		return value
	}
	root := projectRoot()
	if root == "" {
		return filepath.Join("scripts", "xiaohongshu-browser-login.mjs")
	}
	return filepath.Join(root, "scripts", "xiaohongshu-browser-login.mjs")
}

func RuntimeBrowserProfilePath(workspaceID, accountID string) string {
	root := projectRoot()
	if root == "" {
		return filepath.Join("runtime", "browser-profiles", workspaceID, accountID)
	}
	return filepath.Join(root, "runtime", "browser-profiles", workspaceID, accountID)
}

func projectRoot() string {
	if value := strings.TrimSpace(os.Getenv("GEOPRESS_PROJECT_ROOT")); value != "" {
		return value
	}
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	if filepath.Base(wd) == "backend" {
		return filepath.Dir(wd)
	}
	return wd
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
