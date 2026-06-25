package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"geopress/backend/internal/integration/browserplatform"
	"geopress/backend/internal/model"
)

func (h *WorkspaceHandler) StartMediaAccountAuth(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}

	unlock := h.lockInteractiveLoginStart(workspaceID, accountID)
	defer unlock()

	now := time.Now().UTC()
	profileDir := browserProfilePath(workspaceID, accountID)
	stateFile := browserplatform.BrowserLoginStateFile(profileDir)
	commandFile := browserplatform.BrowserLoginCommandFile(profileDir)

	account, platform, found := h.mediaAccountAndPlatform(workspaceID, accountID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	authStrategy, strategyOK := h.mediaAuthStrategyRegistry().Resolve(platform, account)
	if !strategyOK || !authStrategy.SupportsInteractiveLogin() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media account does not support interactive login"})
		return
	}

	if session, sessionOK, err := h.latestMediaAccountLoginSession(c.Request.Context(), workspaceID, accountID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session lookup failed"})
		return
	} else if sessionOK && session.Status == "active" && now.Before(session.ExpiresAt) {
		state, stateOK, err := authStrategy.InteractiveLoginState(c.Request.Context(), session.ID, session.ProfileDir, session.StateFile, account.CredentialMeta["browserLoginCommandFile"])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login state lookup failed"})
			return
		}
		if stateOK && interactiveLoginStateReusable(state) {
			updated := h.finalizeInteractiveLoginIfConnected(c, account.WorkspaceID, account.ID, session, state)
			if updated.ID != "" {
				account = updated
			}
			h.respondMediaAccountAuthStarted(c, account, authStrategy, session, state, true)
			return
		}
	}

	expiresAt := now.Add(10 * time.Minute)
	sessionID := authSessionID(platform.Type, now)
	state, err := authStrategy.StartInteractiveLogin(c.Request.Context(), browserplatform.InteractiveLoginStartRequest{
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		SessionID:   sessionID,
		ProfileDir:  profileDir,
		StateFile:   stateFile,
		CommandFile: commandFile,
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	updated, ok := h.updateMediaAccountLoginMetadata(workspaceID, accountID, func(account *model.MediaAccount) {
		account.CredentialMeta["loginStartedAt"] = now.Format(time.RFC3339)
		account.CredentialMeta["authorizationStrategy"] = string(authStrategy.Kind())
		account.CredentialMeta["browserSessionMode"] = "playwright_persistent_context"
		account.CredentialMeta["browserProfile"] = state.ProfileDir
		account.CredentialMeta["browserLoginUrl"] = state.LoginURL
		account.CredentialMeta["browserLoginStateFile"] = state.StateFile
		account.CredentialMeta["browserLoginCommandFile"] = state.CommandFile
		account.CredentialMeta["loginSessionId"] = state.SessionID
		account.CredentialMeta["loginSessionExpiresAt"] = expiresAt.Format(time.RFC3339)
		account.Status = "login_waiting"
		account.HealthStatus = mediaAccountHealthFromStatus(account.Status)
		account.LastCheckedAt = now
	})
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	if err := h.saveMediaAccount(c.Request.Context(), updated); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login state was not persisted"})
		return
	}
	if err := h.saveMediaAccountLoginSession(c.Request.Context(), model.MediaAccountLoginSession{
		ID:          state.SessionID,
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Platform:    platform.Type,
		ProfileDir:  state.ProfileDir,
		LoginURL:    state.LoginURL,
		StateFile:   state.StateFile,
		Status:      "active",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session was not persisted"})
		return
	}

	h.respondMediaAccountAuthStarted(c, updated, authStrategy, model.MediaAccountLoginSession{
		ID:          state.SessionID,
		WorkspaceID: workspaceID,
		AccountID:   accountID,
		Platform:    platform.Type,
		ProfileDir:  state.ProfileDir,
		LoginURL:    state.LoginURL,
		StateFile:   state.StateFile,
		Status:      "active",
		ExpiresAt:   expiresAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, state, false)
}

func (h *WorkspaceHandler) MediaAccountAuthStatus(c *gin.Context) {
	h.withInteractiveAuthSession(c, func(account model.MediaAccount, platform model.MediaPlatform, strategy mediaAuthStrategy, session model.MediaAccountLoginSession) {
		state, stateOK, err := strategy.InteractiveLoginState(c.Request.Context(), session.ID, session.ProfileDir, session.StateFile, account.CredentialMeta["browserLoginCommandFile"])
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login state lookup failed"})
			return
		}
		if !stateOK {
			c.JSON(http.StatusNotFound, gin.H{"error": "media account login state not found"})
			return
		}
		h.finalizeInteractiveLoginIfConnected(c, account.WorkspaceID, account.ID, session, state)
		c.JSON(http.StatusOK, gin.H{"account": account, "state": state})
	})
}

func (h *WorkspaceHandler) MediaAccountAuthAction(c *gin.Context) {
	var req mediaAccountAuthActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	h.withInteractiveAuthSession(c, func(account model.MediaAccount, platform model.MediaPlatform, strategy mediaAuthStrategy, session model.MediaAccountLoginSession) {
		if strings.TrimSpace(req.SessionID) != session.ID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "login session is invalid"})
			return
		}
		state, stateOK, err := strategy.InteractiveLoginAction(c.Request.Context(), session.ProfileDir, session.StateFile, account.CredentialMeta["browserLoginCommandFile"], browserplatform.InteractiveLoginActionRequest{
			SessionID:   req.SessionID,
			Action:      req.Action,
			PhoneNumber: req.PhoneNumber,
			CaptchaCode: req.CaptchaCode,
			SMSCode:     req.SMSCode,
			Payload:     req.Payload,
		})
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		if !stateOK {
			c.JSON(http.StatusNotFound, gin.H{"error": "media account login state not found"})
			return
		}
		updated := h.finalizeInteractiveLoginIfConnected(c, account.WorkspaceID, account.ID, session, state)
		if updated.ID != "" {
			account = updated
		}
		c.JSON(http.StatusOK, gin.H{"account": account, "state": state})
	})
}

func (h *WorkspaceHandler) withInteractiveAuthSession(c *gin.Context, fn func(account model.MediaAccount, platform model.MediaPlatform, strategy mediaAuthStrategy, session model.MediaAccountLoginSession)) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	accountID := strings.TrimSpace(c.Param("accountId"))
	if accountID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account id is required"})
		return
	}
	account, platform, found := h.mediaAccountAndPlatform(workspaceID, accountID)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{"error": "media account not found"})
		return
	}
	strategy, strategyOK := h.mediaAuthStrategyRegistry().Resolve(platform, account)
	if !strategyOK || !strategy.SupportsInteractiveLogin() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "media account does not support interactive login"})
		return
	}
	session, sessionOK, err := h.latestMediaAccountLoginSession(c.Request.Context(), workspaceID, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "media account login session lookup failed"})
		return
	}
	if !sessionOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "login session was not started"})
		return
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = h.expireMediaAccountLoginSession(c.Request.Context(), session.ID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "login session expired"})
		return
	}
	fn(account, platform, strategy, session)
}

func (h *WorkspaceHandler) finalizeInteractiveLoginIfConnected(c *gin.Context, workspaceID string, accountID string, session model.MediaAccountLoginSession, state browserplatform.InteractiveLoginState) model.MediaAccount {
	if !state.LoggedIn && state.Status != "connected" {
		return model.MediaAccount{}
	}
	now := time.Now().UTC()
	updated, ok := h.updateMediaAccountLoginMetadata(workspaceID, accountID, func(account *model.MediaAccount) {
		account.CredentialMeta["loginCompletedAt"] = now.Format(time.RFC3339)
		account.CredentialMeta["browserProfile"] = state.ProfileDir
		account.CredentialMeta["browserLoginUrl"] = state.LoginURL
		account.CredentialMeta["browserLoginStateFile"] = state.StateFile
		account.CredentialMeta["browserLoginCommandFile"] = state.CommandFile
		account.CredentialMeta["loginSessionId"] = session.ID
		account.Status = "connected"
		account.HealthStatus = mediaAccountHealthFromStatus(account.Status)
		account.LastCheckedAt = now
	})
	if !ok {
		return model.MediaAccount{}
	}
	if err := h.saveMediaAccount(c.Request.Context(), updated); err == nil {
		_ = h.completeMediaAccountLoginSession(c.Request.Context(), session.ID)
	}
	return updated
}

func (h *WorkspaceHandler) respondMediaAccountAuthStarted(c *gin.Context, account model.MediaAccount, strategy mediaAuthStrategy, session model.MediaAccountLoginSession, state browserplatform.InteractiveLoginState, reused bool) {
	c.JSON(http.StatusOK, gin.H{
		"account":     account,
		"expiresAt":   session.ExpiresAt,
		"mode":        "playwright_persistent_context",
		"strategy":    strategy.Kind(),
		"sessionId":   session.ID,
		"state":       state,
		"stateFile":   state.StateFile,
		"commandFile": state.CommandFile,
		"reused":      reused,
	})
}

func interactiveLoginStateReusable(state browserplatform.InteractiveLoginState) bool {
	switch strings.TrimSpace(state.Status) {
	case "", "expired", "failed":
		return false
	default:
		return true
	}
}

func (h *WorkspaceHandler) mediaAccountAndPlatform(workspaceID string, accountID string) (model.MediaAccount, model.MediaPlatform, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	account, accountOK := h.mediaAccountByID(workspaceID, accountID)
	if !accountOK {
		return model.MediaAccount{}, model.MediaPlatform{}, false
	}
	platform, platformOK := h.mediaPlatformByID(account.PlatformID)
	if !platformOK {
		return model.MediaAccount{}, model.MediaPlatform{}, false
	}
	return account, platform, true
}

func (h *WorkspaceHandler) updateMediaAccountLoginMetadata(workspaceID string, accountID string, apply func(account *model.MediaAccount)) (model.MediaAccount, bool) {
	var updated model.MediaAccount
	h.mu.Lock()
	defer h.mu.Unlock()
	for index := range h.accounts {
		if h.accounts[index].WorkspaceID != workspaceID || h.accounts[index].ID != accountID {
			continue
		}
		if h.accounts[index].CredentialMeta == nil {
			h.accounts[index].CredentialMeta = map[string]string{}
		}
		apply(&h.accounts[index])
		updated = h.accounts[index]
		return updated, true
	}
	return model.MediaAccount{}, false
}

func authSessionID(platformType string, now time.Time) string {
	return strings.ReplaceAll(platformType, "-", "_") + "_login_" + strconv.FormatInt(now.UnixNano(), 10)
}
