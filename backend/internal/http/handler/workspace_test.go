package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/integration/xiaohongshu"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

func TestCreateXiaohongshuMediaAccountWithQRLogin(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"platformId": "plt_xiaohongshu",
		"name": "小红书账号",
		"externalId": "xhs-demo",
		"loginMethod": "qr"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media-accounts", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var account model.MediaAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if account.LoginMethod != "qr" {
		t.Fatalf("login method = %q, want qr", account.LoginMethod)
	}
	if account.Status != "pending_login" {
		t.Fatalf("status = %q, want pending_login", account.Status)
	}
}

func TestCreateXiaohongshuMediaAccountRejectsPhoneLogin(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"platformId": "plt_xiaohongshu",
		"name": "小红书账号",
		"externalId": "xhs-demo",
		"loginMethod": "phone",
		"phoneNumber": "17864293035"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media-accounts", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestXiaohongshuBrowserLoginFlow(t *testing.T) {
	router := testWorkspaceRouter()
	account := createXiaohongshuAccount(t, router)

	sendBody := bytes.NewBufferString(`{}`)
	sendReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media-accounts/%s/browser-login/start", account.ID), sendBody)
	sendReq.Header.Set("Authorization", "Bearer demo-token")
	sendReq.Header.Set("X-Workspace-ID", "wks_personal")
	sendReq.Header.Set("Content-Type", "application/json")

	sendRec := httptest.NewRecorder()
	router.ServeHTTP(sendRec, sendReq)
	if sendRec.Code != http.StatusOK {
		t.Fatalf("send status = %d, want %d, body = %s", sendRec.Code, http.StatusOK, sendRec.Body.String())
	}

	var sendResponse struct {
		Account          model.MediaAccount `json:"account"`
		QRScreenshotData string             `json:"qrScreenshotData"`
		SessionID        string             `json:"sessionId"`
	}
	if err := json.Unmarshal(sendRec.Body.Bytes(), &sendResponse); err != nil {
		t.Fatalf("unmarshal send response: %v", err)
	}
	if sendResponse.Account.Status != "qr_waiting" {
		t.Fatalf("status after send = %q, want qr_waiting", sendResponse.Account.Status)
	}
	if sendResponse.QRScreenshotData == "" || sendResponse.SessionID == "" {
		t.Fatalf("expected qr screenshot and session id, got image=%q session=%q", sendResponse.QRScreenshotData, sendResponse.SessionID)
	}
	if sendResponse.Account.CredentialMeta["browserProfile"] == "" {
		t.Fatal("expected browser profile metadata")
	}

	verifyBody := bytes.NewBufferString(fmt.Sprintf(`{"sessionId":%q}`, sendResponse.SessionID))
	verifyReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media-accounts/%s/browser-login/complete", account.ID), verifyBody)
	verifyReq.Header.Set("Authorization", "Bearer demo-token")
	verifyReq.Header.Set("X-Workspace-ID", "wks_personal")
	verifyReq.Header.Set("Content-Type", "application/json")

	verifyRec := httptest.NewRecorder()
	router.ServeHTTP(verifyRec, verifyReq)
	if verifyRec.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want %d, body = %s", verifyRec.Code, http.StatusOK, verifyRec.Body.String())
	}

	var verified model.MediaAccount
	if err := json.Unmarshal(verifyRec.Body.Bytes(), &verified); err != nil {
		t.Fatalf("unmarshal verify response: %v", err)
	}
	if verified.Status != "connected" {
		t.Fatalf("verified status = %q, want connected", verified.Status)
	}
	if verified.CredentialMeta["qrLoginCompletedAt"] == "" {
		t.Fatal("expected qrLoginCompletedAt metadata")
	}
}

func createXiaohongshuAccount(t *testing.T, router *gin.Engine) model.MediaAccount {
	t.Helper()

	body := bytes.NewBufferString(`{
		"platformId": "plt_xiaohongshu",
		"name": "小红书账号",
		"externalId": "xhs-demo",
		"loginMethod": "qr"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media-accounts", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create account status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var account model.MediaAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		t.Fatalf("unmarshal account response: %v", err)
	}
	return account
}

func testWorkspaceRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	handler := NewWorkspaceHandler(nil, ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderMock}))
	handler.browserLogin = fakeBrowserLoginService{}
	handler.Register(apiGroup, middleware.Auth())
	return router
}

type fakeBrowserLoginService struct{}

func (fakeBrowserLoginService) Start(_ context.Context, req xiaohongshu.BrowserLoginStartRequest) (xiaohongshu.BrowserLoginStartResult, error) {
	return xiaohongshu.BrowserLoginStartResult{
		SessionID:        req.SessionID,
		LoginURL:         xiaohongshu.DefaultLoginURL,
		PageURL:          xiaohongshu.DefaultLoginURL,
		QRScreenshotData: "data:image/png;base64,test",
		ProfileDir:       req.ProfileDir,
		StartedAt:        time.Now().UTC(),
	}, nil
}

func (fakeBrowserLoginService) Complete(_ context.Context, req xiaohongshu.BrowserLoginCompleteRequest) (xiaohongshu.BrowserLoginCompleteResult, error) {
	return xiaohongshu.BrowserLoginCompleteResult{
		SessionID:   req.SessionID,
		PageURL:     xiaohongshu.DefaultLoginURL,
		ProfileDir:  req.ProfileDir,
		LoggedIn:    true,
		CompletedAt: time.Now().UTC(),
	}, nil
}
