package browserplatform

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

type InteractiveLoginService struct {
	PlatformType        string
	PlatformName        string
	LoginURL            string
	ScriptPath          string
	NodeBin             string
	ChromePath          string
	InitialStateTimeout time.Duration
}

func (s InteractiveLoginService) LoginURLValue() string {
	return s.LoginURL
}

type InteractiveLoginStartRequest struct {
	WorkspaceID string
	AccountID   string
	SessionID   string
	ProfileDir  string
	StateFile   string
	CommandFile string
}

type InteractiveLoginActionRequest struct {
	SessionID   string         `json:"sessionId"`
	Action      string         `json:"action"`
	PhoneNumber string         `json:"phoneNumber,omitempty"`
	CaptchaCode string         `json:"captchaCode,omitempty"`
	SMSCode     string         `json:"smsCode,omitempty"`
	Payload     map[string]any `json:"payload,omitempty"`
}

type InteractiveLoginState struct {
	SessionID             string         `json:"sessionId"`
	Platform              string         `json:"platform"`
	LoginURL              string         `json:"loginUrl"`
	PageURL               string         `json:"pageUrl"`
	ProfileDir            string         `json:"profileDir"`
	StateFile             string         `json:"stateFile"`
	CommandFile           string         `json:"commandFile,omitempty"`
	Status                string         `json:"status"`
	Message               string         `json:"message"`
	LoggedIn              bool           `json:"loggedIn"`
	CaptchaScreenshotData string         `json:"captchaScreenshotData,omitempty"`
	AllowedActions        []string       `json:"allowedActions"`
	Warnings              []string       `json:"warnings"`
	StartedAt             string         `json:"startedAt"`
	LastCheckedAt         string         `json:"lastCheckedAt"`
	CompletedAt           string         `json:"completedAt,omitempty"`
	LastCommandID         string         `json:"lastCommandId,omitempty"`
	RawStatus             map[string]any `json:"rawStatus,omitempty"`
}

func (s InteractiveLoginService) Start(ctx context.Context, req InteractiveLoginStartRequest) (InteractiveLoginState, error) {
	if strings.TrimSpace(req.SessionID) == "" {
		return InteractiveLoginState{}, errors.New("session id is required")
	}
	if strings.TrimSpace(req.ProfileDir) == "" {
		return InteractiveLoginState{}, errors.New("browser profile dir is required")
	}
	stateFile := firstNonEmpty(req.StateFile, BrowserLoginStateFile(req.ProfileDir))
	commandFile := firstNonEmpty(req.CommandFile, BrowserLoginCommandFile(req.ProfileDir))
	args := s.baseArgs("watch", req.SessionID, req.ProfileDir, stateFile, commandFile)
	out, err := s.startWatcher(ctx, args...)
	if err != nil {
		return InteractiveLoginState{}, err
	}
	var state InteractiveLoginState
	if err := json.Unmarshal(out, &state); err != nil {
		return InteractiveLoginState{}, fmt.Errorf("decode interactive login initial state: %w", err)
	}
	return s.withDefaults(req.SessionID, req.ProfileDir, stateFile, commandFile, state), nil
}

func (s InteractiveLoginService) State(_ context.Context, sessionID string, profileDir string, stateFile string, commandFile string) (InteractiveLoginState, bool, error) {
	statePath := firstNonEmpty(stateFile, BrowserLoginStateFile(profileDir))
	if statePath == "" {
		return InteractiveLoginState{}, false, nil
	}
	data, err := os.ReadFile(statePath)
	if errors.Is(err, os.ErrNotExist) {
		return InteractiveLoginState{}, false, nil
	}
	if err != nil {
		return InteractiveLoginState{}, false, err
	}
	var state InteractiveLoginState
	if err := json.Unmarshal(data, &state); err != nil {
		return InteractiveLoginState{}, false, err
	}
	return s.withDefaults(sessionID, profileDir, statePath, firstNonEmpty(commandFile, BrowserLoginCommandFile(profileDir)), state), true, nil
}

func (s InteractiveLoginService) Action(ctx context.Context, profileDir string, stateFile string, commandFile string, req InteractiveLoginActionRequest) (InteractiveLoginState, bool, error) {
	commandPath := firstNonEmpty(commandFile, BrowserLoginCommandFile(profileDir))
	if commandPath == "" {
		return InteractiveLoginState{}, false, errors.New("browser login command file is required")
	}
	if strings.TrimSpace(req.SessionID) == "" {
		return InteractiveLoginState{}, false, errors.New("sessionId is required")
	}
	commandID := fmt.Sprintf("%s_%d", req.Action, time.Now().UTC().UnixNano())
	command := map[string]any{
		"sessionId": req.SessionID,
		"type":      req.Action,
		"commandId": commandID,
	}
	if req.PhoneNumber != "" {
		command["phoneNumber"] = req.PhoneNumber
	}
	if req.CaptchaCode != "" {
		command["captchaCode"] = req.CaptchaCode
	}
	if req.SMSCode != "" {
		command["smsCode"] = req.SMSCode
	}
	for key, value := range req.Payload {
		command[key] = value
	}
	data, err := json.Marshal(command)
	if err != nil {
		return InteractiveLoginState{}, false, err
	}
	if err := os.MkdirAll(filepath.Dir(commandPath), 0o755); err != nil {
		return InteractiveLoginState{}, false, err
	}
	if err := os.WriteFile(commandPath, data, 0o600); err != nil {
		return InteractiveLoginState{}, false, err
	}

	deadline := time.Now().Add(8 * time.Second)
	var lastState InteractiveLoginState
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return InteractiveLoginState{}, false, ctx.Err()
		case <-time.After(300 * time.Millisecond):
		}
		state, ok, err := s.State(ctx, req.SessionID, profileDir, stateFile, commandPath)
		if err != nil {
			return InteractiveLoginState{}, false, err
		}
		if ok {
			lastState = state
			if state.LastCommandID == commandID {
				return state, true, nil
			}
		}
	}
	if lastState.SessionID != "" {
		return lastState, true, nil
	}
	return InteractiveLoginState{}, false, nil
}

func (s InteractiveLoginService) baseArgs(action string, sessionID string, profileDir string, stateFile string, commandFile string) []string {
	args := []string{
		s.scriptPath(),
		"--action", action,
		"--session-id", sessionID,
		"--profile-dir", profileDir,
		"--login-url", s.LoginURL,
		"--state-file", stateFile,
		"--command-file", commandFile,
	}
	if chromePath := strings.TrimSpace(s.ChromePath); chromePath != "" {
		args = append(args, "--chrome-path", chromePath)
	}
	return args
}

func (s InteractiveLoginService) startWatcher(ctx context.Context, args ...string) ([]byte, error) {
	nodeBin := strings.TrimSpace(s.NodeBin)
	if nodeBin == "" {
		nodeBin = defaultNodeBin()
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
		return nil, fmt.Errorf("%s interactive login watcher failed: %s", s.platformName(), message)
	}

	timeout := s.InitialStateTimeout
	if timeout <= 0 {
		timeout = 90 * time.Second
	}
	line, err := readFirstLine(stdout, timeout)
	if err != nil {
		_ = cmd.Process.Kill()
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, fmt.Errorf("%s interactive login watcher did not return initial state: %s", s.platformName(), message)
	}
	go func() {
		_ = cmd.Wait()
	}()
	return line, nil
}

func (s InteractiveLoginService) withDefaults(sessionID string, profileDir string, stateFile string, commandFile string, state InteractiveLoginState) InteractiveLoginState {
	if state.SessionID == "" {
		state.SessionID = sessionID
	}
	if state.Platform == "" {
		state.Platform = s.PlatformType
	}
	if state.LoginURL == "" {
		state.LoginURL = s.LoginURL
	}
	if state.ProfileDir == "" {
		state.ProfileDir = profileDir
	}
	if state.StateFile == "" {
		state.StateFile = stateFile
	}
	if state.CommandFile == "" {
		state.CommandFile = commandFile
	}
	if state.AllowedActions == nil {
		state.AllowedActions = []string{}
	}
	if state.Warnings == nil {
		state.Warnings = []string{}
	}
	return state
}

func (s InteractiveLoginService) scriptPath() string {
	if strings.TrimSpace(s.ScriptPath) != "" {
		return s.ScriptPath
	}
	return filepath.Join(installRoot(), "scripts", fmt.Sprintf("%s-browser-phone-login.mjs", s.PlatformType))
}

func (s InteractiveLoginService) platformName() string {
	if strings.TrimSpace(s.PlatformName) != "" {
		return strings.TrimSpace(s.PlatformName)
	}
	return s.PlatformType
}

func BrowserLoginCommandFile(profileDir string) string {
	if strings.TrimSpace(profileDir) == "" {
		return ""
	}
	return filepath.Join(profileDir, "geopress-login-command.json")
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
