package handler

import (
	"context"
	"errors"
	"strings"
	"time"

	"geopress/backend/internal/domain"
	"geopress/backend/internal/integration/browserplatform"
	"geopress/backend/internal/integration/xiaohongshu"
	"geopress/backend/internal/model"
)

type mediaAuthStrategyKind string

const (
	mediaAuthStrategyKindQRBrowser       mediaAuthStrategyKind = "qr_browser"
	mediaAuthStrategyKindPhoneSMSBrowser mediaAuthStrategyKind = "phone_sms_browser"
)

var (
	errUnsupportedBrowserLogin     = errors.New("browser login is not supported by this authorization strategy")
	errUnsupportedInteractiveLogin = errors.New("interactive login is not supported by this authorization strategy")
)

type mediaAuthStrategy interface {
	Kind() mediaAuthStrategyKind
	Method() domain.AuthorizationMethod
	Mode() domain.ConnectorCapabilityMode
	SupportsBrowserLogin() bool
	SupportsInteractiveLogin() bool
	StartBrowserLogin(ctx context.Context, req xiaohongshu.BrowserLoginStartRequest) (xiaohongshu.BrowserLoginStartResult, error)
	CompleteBrowserLogin(ctx context.Context, req xiaohongshu.BrowserLoginCompleteRequest) (xiaohongshu.BrowserLoginCompleteResult, error)
	StartInteractiveLogin(ctx context.Context, req browserplatform.InteractiveLoginStartRequest) (browserplatform.InteractiveLoginState, error)
	InteractiveLoginState(ctx context.Context, sessionID string, profileDir string, stateFile string, commandFile string) (browserplatform.InteractiveLoginState, bool, error)
	InteractiveLoginAction(ctx context.Context, profileDir string, stateFile string, commandFile string, req browserplatform.InteractiveLoginActionRequest) (browserplatform.InteractiveLoginState, bool, error)
}

type interactiveLoginService interface {
	Start(ctx context.Context, req browserplatform.InteractiveLoginStartRequest) (browserplatform.InteractiveLoginState, error)
	State(ctx context.Context, sessionID string, profileDir string, stateFile string, commandFile string) (browserplatform.InteractiveLoginState, bool, error)
	Action(ctx context.Context, profileDir string, stateFile string, commandFile string, req browserplatform.InteractiveLoginActionRequest) (browserplatform.InteractiveLoginState, bool, error)
	LoginURLValue() string
}

type mediaAuthStrategyRegistry struct {
	browserLoginForPlatform     func(platformType string) (xiaohongshu.BrowserLoginService, string)
	interactiveLoginForPlatform func(platformType string) (interactiveLoginService, bool)
}

func (registry mediaAuthStrategyRegistry) Resolve(platform model.MediaPlatform, account model.MediaAccount) (mediaAuthStrategy, bool) {
	if !platform.Enabled || account.LoginMethod == "" {
		return nil, false
	}
	platform.EnsureCapabilities()

	for _, method := range platform.Capabilities.AuthorizationMethods {
		switch method {
		case domain.AuthorizationMethodQRLogin:
			if account.LoginMethod != "qr" || !platformAuthorizationUsesMode(platform, domain.ConnectorCapabilityModeBrowser) {
				continue
			}
			if registry.browserLoginForPlatform == nil {
				continue
			}
			service, loginURL := registry.browserLoginForPlatform(platform.Type)
			if service == nil || strings.TrimSpace(loginURL) == "" {
				continue
			}
			return qrBrowserAuthStrategy{
				service:  service,
				loginURL: loginURL,
			}, true
		case domain.AuthorizationMethodPhoneSMS:
			if account.LoginMethod != "phone" || !platformAuthorizationUsesMode(platform, domain.ConnectorCapabilityModeBrowser) {
				continue
			}
			if registry.interactiveLoginForPlatform == nil {
				continue
			}
			service, ok := registry.interactiveLoginForPlatform(platform.Type)
			if !ok || strings.TrimSpace(service.LoginURLValue()) == "" {
				continue
			}
			return phoneSMSBrowserAuthStrategy{service: service}, true
		}
	}
	return nil, false
}

type qrBrowserAuthStrategy struct {
	service  xiaohongshu.BrowserLoginService
	loginURL string
}

func (strategy qrBrowserAuthStrategy) Kind() mediaAuthStrategyKind {
	return mediaAuthStrategyKindQRBrowser
}

func (strategy qrBrowserAuthStrategy) Method() domain.AuthorizationMethod {
	return domain.AuthorizationMethodQRLogin
}

func (strategy qrBrowserAuthStrategy) Mode() domain.ConnectorCapabilityMode {
	return domain.ConnectorCapabilityModeBrowser
}

func (strategy qrBrowserAuthStrategy) SupportsBrowserLogin() bool {
	return true
}

func (strategy qrBrowserAuthStrategy) SupportsInteractiveLogin() bool {
	return false
}

func (strategy qrBrowserAuthStrategy) StartBrowserLogin(ctx context.Context, req xiaohongshu.BrowserLoginStartRequest) (xiaohongshu.BrowserLoginStartResult, error) {
	req.LoginURL = firstNonEmptyString(req.LoginURL, strategy.loginURL)
	return strategy.service.Start(ctx, req)
}

func (strategy qrBrowserAuthStrategy) CompleteBrowserLogin(ctx context.Context, req xiaohongshu.BrowserLoginCompleteRequest) (xiaohongshu.BrowserLoginCompleteResult, error) {
	req.LoginURL = firstNonEmptyString(req.LoginURL, strategy.loginURL)
	return strategy.service.Complete(ctx, req)
}

func (strategy qrBrowserAuthStrategy) StartInteractiveLogin(context.Context, browserplatform.InteractiveLoginStartRequest) (browserplatform.InteractiveLoginState, error) {
	return browserplatform.InteractiveLoginState{}, errUnsupportedInteractiveLogin
}

func (strategy qrBrowserAuthStrategy) InteractiveLoginState(context.Context, string, string, string, string) (browserplatform.InteractiveLoginState, bool, error) {
	return browserplatform.InteractiveLoginState{}, false, errUnsupportedInteractiveLogin
}

func (strategy qrBrowserAuthStrategy) InteractiveLoginAction(context.Context, string, string, string, browserplatform.InteractiveLoginActionRequest) (browserplatform.InteractiveLoginState, bool, error) {
	return browserplatform.InteractiveLoginState{}, false, errUnsupportedInteractiveLogin
}

type phoneSMSBrowserAuthStrategy struct {
	service interactiveLoginService
}

func (strategy phoneSMSBrowserAuthStrategy) Kind() mediaAuthStrategyKind {
	return mediaAuthStrategyKindPhoneSMSBrowser
}

func (strategy phoneSMSBrowserAuthStrategy) Method() domain.AuthorizationMethod {
	return domain.AuthorizationMethodPhoneSMS
}

func (strategy phoneSMSBrowserAuthStrategy) Mode() domain.ConnectorCapabilityMode {
	return domain.ConnectorCapabilityModeBrowser
}

func (strategy phoneSMSBrowserAuthStrategy) SupportsBrowserLogin() bool {
	return false
}

func (strategy phoneSMSBrowserAuthStrategy) SupportsInteractiveLogin() bool {
	return true
}

func (strategy phoneSMSBrowserAuthStrategy) StartBrowserLogin(context.Context, xiaohongshu.BrowserLoginStartRequest) (xiaohongshu.BrowserLoginStartResult, error) {
	return xiaohongshu.BrowserLoginStartResult{}, errUnsupportedBrowserLogin
}

func (strategy phoneSMSBrowserAuthStrategy) CompleteBrowserLogin(context.Context, xiaohongshu.BrowserLoginCompleteRequest) (xiaohongshu.BrowserLoginCompleteResult, error) {
	return xiaohongshu.BrowserLoginCompleteResult{}, errUnsupportedBrowserLogin
}

func (strategy phoneSMSBrowserAuthStrategy) StartInteractiveLogin(ctx context.Context, req browserplatform.InteractiveLoginStartRequest) (browserplatform.InteractiveLoginState, error) {
	return strategy.service.Start(ctx, req)
}

func (strategy phoneSMSBrowserAuthStrategy) InteractiveLoginState(ctx context.Context, sessionID string, profileDir string, stateFile string, commandFile string) (browserplatform.InteractiveLoginState, bool, error) {
	return strategy.service.State(ctx, sessionID, profileDir, stateFile, commandFile)
}

func (strategy phoneSMSBrowserAuthStrategy) InteractiveLoginAction(ctx context.Context, profileDir string, stateFile string, commandFile string, req browserplatform.InteractiveLoginActionRequest) (browserplatform.InteractiveLoginState, bool, error) {
	return strategy.service.Action(ctx, profileDir, stateFile, commandFile, req)
}

func platformAuthorizationUsesMode(platform model.MediaPlatform, mode domain.ConnectorCapabilityMode) bool {
	for _, capability := range platform.Capabilities.Capabilities {
		if capability.Name == domain.ConnectorCapabilityAuthorization && capability.Enabled && capability.Mode == mode {
			return true
		}
	}
	return false
}

func (h *WorkspaceHandler) mediaAuthStrategyRegistry() mediaAuthStrategyRegistry {
	interactiveFactory := h.interactiveLoginForPlatform
	if interactiveFactory == nil {
		interactiveFactory = h.interactiveLoginServiceForPlatform
	}
	return mediaAuthStrategyRegistry{
		browserLoginForPlatform:     h.browserLoginServiceForPlatform,
		interactiveLoginForPlatform: interactiveFactory,
	}
}

func loginSessionExpiresAt(now time.Time) time.Time {
	return now.Add(5 * time.Minute)
}
