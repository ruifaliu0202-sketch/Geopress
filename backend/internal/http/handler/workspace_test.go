package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/domain"
	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/integration/browserplatform"
	publishing "geopress/backend/internal/integration/publisher"
	"geopress/backend/internal/integration/xiaohongshu"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

const demoPasswordHash = "$2a$10$RZ9nf/MK8Gn8.tJ4uIfnPOR0KCfQfzwvhapNoXKrpaVQ0UROabcpG"

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
	if account.HealthStatus != "needs_authorization" {
		t.Fatalf("health status = %q, want needs_authorization", account.HealthStatus)
	}
	if account.ContentCategories == nil {
		t.Fatal("contentCategories should be an empty array, not null")
	}
	if account.MatrixMetadata == nil {
		t.Fatal("matrixMetadata should be an empty object, not null")
	}
}

func TestMediaAccountMatrixListsAccounts(t *testing.T) {
	router := testWorkspaceRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/media-account-matrix", nil)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Items []model.MediaAccountMatrixItem `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Items == nil {
		t.Fatal("items should be an empty array or populated array, not null")
	}
	if len(response.Items) == 0 {
		t.Fatal("expected demo matrix account")
	}
	if response.Items[0].Account.ID == "" || response.Items[0].Platform.ID == "" {
		t.Fatalf("unexpected matrix item: %#v", response.Items[0])
	}
	if response.Items[0].Warnings == nil {
		t.Fatal("warnings should be an empty array or populated array, not null")
	}
}

func TestMediaMatrixMetricListsReturnEmptyArrays(t *testing.T) {
	router := testWorkspaceRouter()
	for _, path := range []string{
		"/api/media-account-matrix/acc_xhs_acme/metric-snapshots",
		"/api/content-metrics?mediaAccountId=acc_xhs_acme",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("Authorization", "Bearer demo-token")
		req.Header.Set("X-Workspace-ID", "wks_acme")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d, body = %s", path, rec.Code, http.StatusOK, rec.Body.String())
		}

		var response struct {
			Items []json.RawMessage `json:"items"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("unmarshal %s response: %v", path, err)
		}
		if response.Items == nil {
			t.Fatalf("%s items should be [], not null", path)
		}
		if len(response.Items) != 0 {
			t.Fatalf("%s items = %d, want 0", path, len(response.Items))
		}
	}
}

func TestCreateMediaAccountSyncJobRecordsQueuedRequest(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"syncType": "metrics",
		"idempotencyKey": "test-sync-key",
		"requestPayload": {"reason": "manual"}
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media-account-matrix/acc_xhs_acme/sync-jobs", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusAccepted, rec.Body.String())
	}

	var job model.MediaAccountSyncJob
	if err := json.Unmarshal(rec.Body.Bytes(), &job); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if job.Status != "queued" || job.IdempotencyKey != "test-sync-key" {
		t.Fatalf("unexpected sync job: %#v", job)
	}
	if job.RequestPayload == nil || job.ResultSummary == nil {
		t.Fatalf("sync job maps should not be nil: %#v", job)
	}
}

func TestDemoPasswordHashMatchesDocumentedPassword(t *testing.T) {
	if err := bcrypt.CompareHashAndPassword([]byte(demoPasswordHash), []byte("demo")); err != nil {
		t.Fatalf("demo password hash does not match documented password: %v", err)
	}
}

func TestRegisterUserCreatesPersonalWorkspace(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"name": "注册用户",
		"email": "new-user@example.com",
		"password": "password123",
		"workspaceName": "新用户工作区"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Token      string            `json:"token"`
		User       model.User        `json:"user"`
		Workspaces []model.Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Token == "" {
		t.Fatal("expected token")
	}
	if response.User.Email != "new-user@example.com" || response.User.IsPlatformAdmin {
		t.Fatalf("unexpected user: %#v", response.User)
	}
	if response.User.SubscriptionTier != model.SubscriptionTierFree || response.User.SubscriptionStatus != model.SubscriptionStatusActive {
		t.Fatalf("unexpected subscription: %#v", response.User)
	}
	if len(response.Workspaces) != 1 {
		t.Fatalf("workspace count = %d, want 1: %#v", len(response.Workspaces), response.Workspaces)
	}
	workspace := response.Workspaces[0]
	if workspace.Name != "新用户工作区" || workspace.Type != model.WorkspacePersonal {
		t.Fatalf("unexpected workspace: %#v", workspace)
	}
	if response.User.OnboardingCompleted {
		t.Fatal("registered user should require onboarding")
	}
}

func TestCompleteOnboardingUpdatesWorkspaceAndSubscription(t *testing.T) {
	router := testWorkspaceRouter()
	registerBody := bytes.NewBufferString(`{
		"name": "引导用户",
		"email": "onboarding@example.com",
		"password": "password123",
		"workspaceName": "引导工作区"
	}`)
	registerReq := httptest.NewRequest(http.MethodPost, "/api/auth/register", registerBody)
	registerReq.Header.Set("Content-Type", "application/json")
	registerRec := httptest.NewRecorder()
	router.ServeHTTP(registerRec, registerReq)
	if registerRec.Code != http.StatusCreated {
		t.Fatalf("register status = %d, want %d, body = %s", registerRec.Code, http.StatusCreated, registerRec.Body.String())
	}

	var registered struct {
		Token      string            `json:"token"`
		Workspaces []model.Workspace `json:"workspaces"`
	}
	if err := json.Unmarshal(registerRec.Body.Bytes(), &registered); err != nil {
		t.Fatalf("unmarshal register response: %v", err)
	}
	workspaceID := registered.Workspaces[0].ID

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"workspaceId": %q,
		"industry": "本地生活",
		"tones": ["专业", "亲和"],
		"subscriptionPlanId": "vip"
	}`, workspaceID))
	req := httptest.NewRequest(http.MethodPost, "/api/onboarding/complete", body)
	req.Header.Set("Authorization", "Bearer "+registered.Token)
	req.Header.Set("X-Workspace-ID", workspaceID)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		User      model.User      `json:"user"`
		Workspace model.Workspace `json:"workspace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal onboarding response: %v", err)
	}
	if !response.User.OnboardingCompleted || response.User.SubscriptionPlanID != model.SubscriptionPlanVIP || response.User.MonthlyTokenBudgetCents != 10000 {
		t.Fatalf("unexpected user after onboarding: %#v", response.User)
	}
	if response.Workspace.Industry != "本地生活" || response.Workspace.Tone != "专业、亲和" {
		t.Fatalf("unexpected workspace after onboarding: %#v", response.Workspace)
	}
}

func TestCampaignListReturnsEmptyArray(t *testing.T) {
	router := testWorkspaceRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/campaigns", nil)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Items []model.Campaign `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Items == nil {
		t.Fatalf("items is nil, want empty array")
	}
	if len(response.Items) != 0 {
		t.Fatalf("items length = %d, want 0", len(response.Items))
	}
}

func TestCreateCampaignAndLooseCalendarItem(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"name": "新品种草战役",
		"status": "planned",
		"goal": "验证新品内容方向",
		"products": ["新品 A"],
		"targetAudiences": ["年轻白领"],
		"channels": ["xiaohongshu"],
		"mediaAccountIds": ["acc_xhs_personal"],
		"contentQuota": 3,
		"successMetrics": ["publish_count", "engagement"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}
	if campaign.Status != model.CampaignPlanned {
		t.Fatalf("campaign status = %q, want planned", campaign.Status)
	}

	itemBody := bytes.NewBufferString(`{
		"title": "第一篇预热选题",
		"brief": "先验证用户痛点，不绑定已有内容或发布计划",
		"contentType": "note",
		"channel": "xiaohongshu",
		"mediaAccountId": "acc_xhs_personal"
	}`)
	itemReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("X-Workspace-ID", "wks_personal")
	itemReq.Header.Set("Content-Type", "application/json")

	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)

	if itemRec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", itemRec.Code, http.StatusCreated, itemRec.Body.String())
	}
	var item model.CampaignCalendarItem
	if err := json.Unmarshal(itemRec.Body.Bytes(), &item); err != nil {
		t.Fatalf("unmarshal calendar item: %v", err)
	}
	if item.ContentID != "" || item.PublishScheduleID != "" || item.PublishJobID != "" {
		t.Fatalf("calendar item should not require content/schedule/job links: %#v", item)
	}

	reportReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%s/report-summary", campaign.ID), nil)
	reportReq.Header.Set("Authorization", "Bearer demo-token")
	reportReq.Header.Set("X-Workspace-ID", "wks_personal")
	reportRec := httptest.NewRecorder()
	router.ServeHTTP(reportRec, reportReq)

	if reportRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", reportRec.Code, http.StatusOK, reportRec.Body.String())
	}
	var summary model.CampaignReportSummary
	if err := json.Unmarshal(reportRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("unmarshal report summary: %v", err)
	}
	if summary.CalendarItemCount != 1 || summary.Metrics == nil || summary.Rollups == nil || summary.Recommendations == nil {
		t.Fatalf("unexpected report summary: %#v", summary)
	}
}

func TestCampaignCalendarListReturnsEmptyArray(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{"name": "空日历战役"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	listReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_personal")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var response struct {
		Items []model.CampaignCalendarItem `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	if response.Items == nil {
		t.Fatal("items is nil, want empty array")
	}
	if len(response.Items) != 0 {
		t.Fatalf("items length = %d, want 0", len(response.Items))
	}
}

func TestUpdateCampaignCanClearTimeline(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"name": "时间窗口战役",
		"startAt": "2026-07-01T00:00:00Z",
		"endAt": "2026-07-31T23:59:59Z"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}
	if campaign.StartAt == nil || campaign.EndAt == nil {
		t.Fatalf("expected initial timeline, got %#v", campaign)
	}

	updateBody := bytes.NewBufferString(`{"startAt": null, "endAt": null}`)
	updateReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/campaigns/%s", campaign.ID), updateBody)
	updateReq.Header.Set("Authorization", "Bearer demo-token")
	updateReq.Header.Set("X-Workspace-ID", "wks_personal")
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	router.ServeHTTP(updateRec, updateReq)

	if updateRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", updateRec.Code, http.StatusOK, updateRec.Body.String())
	}
	var updated model.Campaign
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("unmarshal updated campaign: %v", err)
	}
	if updated.StartAt != nil || updated.EndAt != nil {
		t.Fatalf("timeline was not cleared: %#v", updated)
	}
}

func TestCampaignRejectsIllegalStatusTransition(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{"name": "归档测试战役"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	archiveBody := bytes.NewBufferString(`{"status": "archived"}`)
	archiveReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/campaigns/%s", campaign.ID), archiveBody)
	archiveReq.Header.Set("Authorization", "Bearer demo-token")
	archiveReq.Header.Set("X-Workspace-ID", "wks_personal")
	archiveReq.Header.Set("Content-Type", "application/json")
	archiveRec := httptest.NewRecorder()
	router.ServeHTTP(archiveRec, archiveReq)
	if archiveRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", archiveRec.Code, http.StatusOK, archiveRec.Body.String())
	}

	activeBody := bytes.NewBufferString(`{"status": "active"}`)
	activeReq := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/campaigns/%s", campaign.ID), activeBody)
	activeReq.Header.Set("Authorization", "Bearer demo-token")
	activeReq.Header.Set("X-Workspace-ID", "wks_personal")
	activeReq.Header.Set("Content-Type", "application/json")
	activeRec := httptest.NewRecorder()
	router.ServeHTTP(activeRec, activeReq)
	if activeRec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", activeRec.Code, http.StatusConflict, activeRec.Body.String())
	}
}

func TestCampaignCalendarItemRejectsPublishedCreation(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{"name": "发布态保护战役"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	itemBody := bytes.NewBufferString(`{
		"title": "不能直接已发布",
		"status": "published"
	}`)
	itemReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("X-Workspace-ID", "wks_personal")
	itemReq.Header.Set("Content-Type", "application/json")
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)

	if itemRec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d, body = %s", itemRec.Code, http.StatusConflict, itemRec.Body.String())
	}
}

func TestCampaignCalendarItemRejectsCrossWorkspaceAssignedUser(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{"name": "指派校验战役"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	itemBody := bytes.NewBufferString(`{
		"title": "跨工作区指派",
		"assignedUserId": "usr_growth"
	}`)
	itemReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("X-Workspace-ID", "wks_personal")
	itemReq.Header.Set("Content-Type", "application/json")
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)

	if itemRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", itemRec.Code, http.StatusNotFound, itemRec.Body.String())
	}
}

func TestCampaignCalendarItemRejectsMissingDependency(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{"name": "依赖校验战役"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/campaigns", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(rec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	itemBody := bytes.NewBufferString(`{
		"title": "依赖不存在",
		"dependencyItemIds": ["cci_missing"]
	}`)
	itemReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("X-Workspace-ID", "wks_personal")
	itemReq.Header.Set("Content-Type", "application/json")
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)

	if itemRec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d, body = %s", itemRec.Code, http.StatusNotFound, itemRec.Body.String())
	}
}

func TestCampaignReportCountsScheduleLinkedThroughCalendarItem(t *testing.T) {
	router := testWorkspaceRouter()
	contentBody := bytes.NewBufferString(`{
		"title": "战役内容",
		"summary": "用于发布计划关联",
		"body": "正文"
	}`)
	contentReq := httptest.NewRequest(http.MethodPost, "/api/contents", contentBody)
	contentReq.Header.Set("Authorization", "Bearer demo-token")
	contentReq.Header.Set("X-Workspace-ID", "wks_personal")
	contentReq.Header.Set("Content-Type", "application/json")
	contentRec := httptest.NewRecorder()
	router.ServeHTTP(contentRec, contentReq)
	if contentRec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", contentRec.Code, http.StatusCreated, contentRec.Body.String())
	}
	var content model.Content
	if err := json.Unmarshal(contentRec.Body.Bytes(), &content); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}

	scheduleBody := bytes.NewBufferString(fmt.Sprintf(`{
		"name": "战役发布计划",
		"contentId": %q,
		"mediaAccountId": "acc_xhs_personal",
		"nextRunAt": "2026-07-01T10:00:00Z"
	}`, content.ID))
	scheduleReq := httptest.NewRequest(http.MethodPost, "/api/publish-schedules", scheduleBody)
	scheduleReq.Header.Set("Authorization", "Bearer demo-token")
	scheduleReq.Header.Set("X-Workspace-ID", "wks_personal")
	scheduleReq.Header.Set("Content-Type", "application/json")
	scheduleRec := httptest.NewRecorder()
	router.ServeHTTP(scheduleRec, scheduleReq)
	if scheduleRec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", scheduleRec.Code, http.StatusCreated, scheduleRec.Body.String())
	}
	var scheduleResponse struct {
		Schedule model.PublishSchedule `json:"schedule"`
		Job      model.PublishJob      `json:"job"`
	}
	if err := json.Unmarshal(scheduleRec.Body.Bytes(), &scheduleResponse); err != nil {
		t.Fatalf("unmarshal schedule response: %v", err)
	}

	campaignBody := bytes.NewBufferString(`{"name": "发布链路报表战役"}`)
	campaignReq := httptest.NewRequest(http.MethodPost, "/api/campaigns", campaignBody)
	campaignReq.Header.Set("Authorization", "Bearer demo-token")
	campaignReq.Header.Set("X-Workspace-ID", "wks_personal")
	campaignReq.Header.Set("Content-Type", "application/json")
	campaignRec := httptest.NewRecorder()
	router.ServeHTTP(campaignRec, campaignReq)
	if campaignRec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", campaignRec.Code, http.StatusCreated, campaignRec.Body.String())
	}
	var campaign model.Campaign
	if err := json.Unmarshal(campaignRec.Body.Bytes(), &campaign); err != nil {
		t.Fatalf("unmarshal campaign: %v", err)
	}

	itemBody := bytes.NewBufferString(fmt.Sprintf(`{
		"title": "已排期内容",
		"contentId": %q,
		"publishScheduleId": %q,
		"mediaAccountId": "acc_xhs_personal",
		"status": "scheduled"
	}`, content.ID, scheduleResponse.Schedule.ID))
	itemReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/campaigns/%s/calendar-items", campaign.ID), itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("X-Workspace-ID", "wks_personal")
	itemReq.Header.Set("Content-Type", "application/json")
	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)
	if itemRec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d, body = %s", itemRec.Code, http.StatusCreated, itemRec.Body.String())
	}

	reportReq := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/campaigns/%s/report-summary", campaign.ID), nil)
	reportReq.Header.Set("Authorization", "Bearer demo-token")
	reportReq.Header.Set("X-Workspace-ID", "wks_personal")
	reportRec := httptest.NewRecorder()
	router.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", reportRec.Code, http.StatusOK, reportRec.Body.String())
	}
	var summary model.CampaignReportSummary
	if err := json.Unmarshal(reportRec.Body.Bytes(), &summary); err != nil {
		t.Fatalf("unmarshal report summary: %v", err)
	}
	if summary.ContentCount != 1 || summary.PublishJobCount != 1 || summary.ScheduledItemCount != 1 {
		t.Fatalf("unexpected report summary: %#v", summary)
	}
}

func TestLoginRejectsWrongDemoPassword(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"email": "demo@geopress.local",
		"password": "wrong-password"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
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

func TestDefaultManualMediaPlatformsSupportAccountBindingAndPrepare(t *testing.T) {
	router := testWorkspaceRouter()

	listReq := httptest.NewRequest(http.MethodGet, "/api/media-platforms", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_personal")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("platform list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResponse struct {
		Items []model.MediaPlatform `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	for _, expected := range []struct {
		id              string
		name            string
		typ             string
		credentialField string
		authMethod      domain.AuthorizationMethod
	}{
		{id: "plt_netease", name: "网易号", typ: platformTypeNetease, credentialField: "qrLogin", authMethod: domain.AuthorizationMethodQRLogin},
		{id: "plt_toutiao", name: "头条号", typ: platformTypeToutiao, credentialField: "qrLogin", authMethod: domain.AuthorizationMethodQRLogin},
		{id: "plt_sohu", name: "搜狐号", typ: platformTypeSohu, credentialField: "phoneNumber", authMethod: domain.AuthorizationMethodPhoneSMS},
	} {
		platform, ok := findMediaPlatformForTest(listResponse.Items, expected.id)
		if !ok {
			t.Fatalf("platform %s not found in %#v", expected.id, listResponse.Items)
		}
		if platform.Name != expected.name || platform.Type != expected.typ || !platform.Enabled || !platform.SupportsArticle || !platform.SupportsImage || platform.SupportsScheduling {
			t.Fatalf("unexpected platform metadata for %s: %#v", expected.id, platform)
		}
		if len(platform.CredentialFields) != 1 || platform.CredentialFields[0] != expected.credentialField {
			t.Fatalf("browser platform should require %s: %#v", expected.credentialField, platform)
		}
		if len(platform.Capabilities.AuthorizationMethods) != 1 || platform.Capabilities.AuthorizationMethods[0] != expected.authMethod {
			t.Fatalf("browser platform auth methods = %#v, want %s", platform.Capabilities.AuthorizationMethods, expected.authMethod)
		}
		if !platform.Capabilities.HasCapability(domain.ConnectorCapabilityAuthorization) ||
			!platform.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
			t.Fatalf("browser platform should expose authorization and publish capabilities: %#v", platform.Capabilities)
		}
	}

	accountBody := bytes.NewBufferString(`{
		"platformId": "plt_toutiao",
		"name": "头条号账号",
		"externalId": "toutiao-demo",
		"loginMethod": "qr"
	}`)
	accountReq := httptest.NewRequest(http.MethodPost, "/api/media-accounts", accountBody)
	accountReq.Header.Set("Authorization", "Bearer demo-token")
	accountReq.Header.Set("X-Workspace-ID", "wks_personal")
	accountReq.Header.Set("Content-Type", "application/json")
	accountRec := httptest.NewRecorder()
	router.ServeHTTP(accountRec, accountReq)
	if accountRec.Code != http.StatusCreated {
		t.Fatalf("account create status = %d, want %d, body = %s", accountRec.Code, http.StatusCreated, accountRec.Body.String())
	}
	var account model.MediaAccount
	if err := json.Unmarshal(accountRec.Body.Bytes(), &account); err != nil {
		t.Fatalf("unmarshal account response: %v", err)
	}
	if account.PlatformID != "plt_toutiao" || account.Status != "pending_login" || account.LoginMethod != "qr" {
		t.Fatalf("unexpected browser login account: %#v", account)
	}
	_, handler := testWorkspaceRouterWithHandler()
	handler.accounts = append([]model.MediaAccount{{
		ID:             account.ID,
		WorkspaceID:    "wks_personal",
		PlatformID:     "plt_toutiao",
		Name:           account.Name,
		ExternalID:     account.ExternalID,
		LoginMethod:    "qr",
		CredentialMeta: map[string]string{"browserProfile": "/tmp/geopress-test-profile"},
		Status:         "connected",
	}}, handler.accounts...)
	router = testRouterForHandler(handler)

	prepareBody := bytes.NewBufferString(fmt.Sprintf(`{
		"contentId": "cnt_2001",
		"mediaAccountId": %q
	}`, account.ID))
	prepareReq := httptest.NewRequest(http.MethodPost, "/api/publish/prepare", prepareBody)
	prepareReq.Header.Set("Authorization", "Bearer demo-token")
	prepareReq.Header.Set("X-Workspace-ID", "wks_personal")
	prepareReq.Header.Set("Content-Type", "application/json")
	prepareRec := httptest.NewRecorder()
	router.ServeHTTP(prepareRec, prepareReq)
	if prepareRec.Code != http.StatusCreated {
		t.Fatalf("prepare status = %d, want %d, body = %s", prepareRec.Code, http.StatusCreated, prepareRec.Body.String())
	}
	var prepareResponse struct {
		Job          model.PublishJob        `json:"job"`
		PreparedPost publishing.PreparedPost `json:"preparedPost"`
	}
	if err := json.Unmarshal(prepareRec.Body.Bytes(), &prepareResponse); err != nil {
		t.Fatalf("unmarshal prepare response: %v", err)
	}
	if prepareResponse.PreparedPost.PlatformType != platformTypeToutiao || prepareResponse.PreparedPost.PublishMode != "article" || prepareResponse.PreparedPost.Mode != publishing.ModeBrowserAutomation {
		t.Fatalf("unexpected prepared post: %#v", prepareResponse.PreparedPost)
	}
	if prepareResponse.Job.Status != model.PublishJobManual || prepareResponse.Job.MediaAccountID != account.ID {
		t.Fatalf("unexpected publish job: %#v", prepareResponse.Job)
	}
}

func TestAdminUpdateMediaPlatform(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"name": "小红书",
		"type": "xiaohongshu",
		"enabled": false,
		"supportsArticle": true,
		"supportsImage": false,
		"supportsScheduling": false,
		"credentialFields": ["qrLogin"]
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/admin/media-platforms/plt_xiaohongshu", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var platform model.MediaPlatform
	if err := json.Unmarshal(rec.Body.Bytes(), &platform); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if platform.ID != "plt_xiaohongshu" || platform.Name != "小红书" || platform.Type != "xiaohongshu" {
		t.Fatalf("unexpected platform identity: %#v", platform)
	}
	if platform.Enabled || !platform.SupportsArticle || platform.SupportsImage || platform.SupportsScheduling {
		t.Fatalf("unexpected platform capabilities: %#v", platform)
	}
	if len(platform.CredentialFields) != 1 || platform.CredentialFields[0] != "qrLogin" {
		t.Fatalf("credential fields = %#v, want qrLogin", platform.CredentialFields)
	}
	if !platform.Capabilities.HasCapability(domain.ConnectorCapabilityAuthorization) ||
		!platform.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("legacy admin update should retain xiaohongshu capability contract: %#v", platform.Capabilities)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/media-platforms", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var response struct {
		Items []model.MediaPlatform `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	item, ok := findMediaPlatformForTest(response.Items, "plt_xiaohongshu")
	if !ok {
		t.Fatalf("updated platform not found in list: %#v", response.Items)
	}
	if item.ID != "plt_xiaohongshu" || item.Name != "小红书" || item.Type != "xiaohongshu" {
		t.Fatalf("updated platform not found in list: %#v", response.Items)
	}
	if !item.Capabilities.HasCapability(domain.ConnectorCapabilityAuthorization) ||
		!item.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("list response should include xiaohongshu capability contract: %#v", item.Capabilities)
	}
}

func findMediaPlatformForTest(items []model.MediaPlatform, id string) (model.MediaPlatform, bool) {
	for _, item := range items {
		if item.ID == id {
			return item, true
		}
	}
	return model.MediaPlatform{}, false
}

func TestAdminManagePlatformKnowledgeMarketplace(t *testing.T) {
	router := testWorkspaceRouter()
	baseBody := bytes.NewBufferString(`{
		"name": "医美合规表达包",
		"description": "医美内容生成的表达边界和风险提示。",
		"category": "合规",
		"priceCents": 19900,
		"currency": "cny",
		"marketplaceListed": true
	}`)
	baseReq := httptest.NewRequest(http.MethodPost, "/api/admin/platform-knowledge-bases", baseBody)
	baseReq.Header.Set("Authorization", "Bearer demo-token")
	baseReq.Header.Set("Content-Type", "application/json")

	baseRec := httptest.NewRecorder()
	router.ServeHTTP(baseRec, baseReq)
	if baseRec.Code != http.StatusCreated {
		t.Fatalf("create base status = %d, want %d, body = %s", baseRec.Code, http.StatusCreated, baseRec.Body.String())
	}

	var base model.PlatformKnowledgeBase
	if err := json.Unmarshal(baseRec.Body.Bytes(), &base); err != nil {
		t.Fatalf("unmarshal base response: %v", err)
	}
	if base.ID == "" || base.Name != "医美合规表达包" || base.Currency != "CNY" || !base.MarketplaceListed {
		t.Fatalf("unexpected base response: %#v", base)
	}
	if base.ItemCount != 0 {
		t.Fatalf("item count = %d, want 0", base.ItemCount)
	}

	itemBody := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseId": %q,
		"type": "compliance",
		"title": "禁止绝对化承诺",
		"content": "不要承诺治疗效果，不要使用保证、根治、最安全等绝对化表达。"
	}`, base.ID))
	itemReq := httptest.NewRequest(http.MethodPost, "/api/admin/platform-knowledge-items", itemBody)
	itemReq.Header.Set("Authorization", "Bearer demo-token")
	itemReq.Header.Set("Content-Type", "application/json")

	itemRec := httptest.NewRecorder()
	router.ServeHTTP(itemRec, itemReq)
	if itemRec.Code != http.StatusCreated {
		t.Fatalf("create item status = %d, want %d, body = %s", itemRec.Code, http.StatusCreated, itemRec.Body.String())
	}

	var item model.PlatformKnowledgeItem
	if err := json.Unmarshal(itemRec.Body.Bytes(), &item); err != nil {
		t.Fatalf("unmarshal item response: %v", err)
	}
	if !sameStringSet(item.KnowledgeBaseIDs, []string{base.ID}) || item.Type != "compliance" || !item.Enabled {
		t.Fatalf("unexpected item response: %#v", item)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/admin/platform-knowledge-bases", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list bases status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResponse struct {
		Items []model.PlatformKnowledgeBase `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal base list response: %v", err)
	}
	for _, listedBase := range listResponse.Items {
		if listedBase.ID == base.ID {
			if listedBase.ItemCount != 1 {
				t.Fatalf("listed base item count = %d, want 1", listedBase.ItemCount)
			}
			return
		}
	}
	t.Fatalf("created base not found in list: %#v", listResponse.Items)
}

func TestKnowledgeAssetBasesCanBeReassigned(t *testing.T) {
	router := testWorkspaceRouter()
	base := createWorkspaceKnowledgeBase(t, router, "选题素材包")

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "客户增长案例",
		"text": "客户通过内容矩阵提升线索质量。",
		"mimeType": "text/markdown",
		"originalFilename": "case.md"
	}`))
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var created struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}
	if !sameStringSet(created.Asset.KnowledgeBaseIDs, []string{"kb_brand"}) {
		t.Fatalf("created asset knowledgeBaseIds = %#v, want kb_brand", created.Asset.KnowledgeBaseIDs)
	}
	if len(created.Chunks) == 0 {
		t.Fatal("created asset chunks should not be empty")
	}

	assignBody := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseIds": [%q]
	}`, base.ID))
	assignReq := httptest.NewRequest(http.MethodPut, "/api/knowledge-assets/"+created.Asset.ID+"/bases", assignBody)
	assignReq.Header.Set("Authorization", "Bearer demo-token")
	assignReq.Header.Set("X-Workspace-ID", "wks_acme")
	assignReq.Header.Set("Content-Type", "application/json")

	assignRec := httptest.NewRecorder()
	router.ServeHTTP(assignRec, assignReq)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want %d, body = %s", assignRec.Code, http.StatusOK, assignRec.Body.String())
	}

	var reassigned struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(assignRec.Body.Bytes(), &reassigned); err != nil {
		t.Fatalf("unmarshal reassigned response: %v", err)
	}
	if !sameStringSet(reassigned.Asset.KnowledgeBaseIDs, []string{base.ID}) {
		t.Fatalf("reassigned asset knowledgeBaseIds = %#v, want %s", reassigned.Asset.KnowledgeBaseIDs, base.ID)
	}
	if len(reassigned.Chunks) == 0 || !sameStringSet(reassigned.Chunks[0].KnowledgeBaseIDs, []string{base.ID}) {
		t.Fatalf("reassigned chunk knowledgeBaseIds = %#v, want %s", reassigned.Chunks, base.ID)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets?knowledgeBaseId="+base.ID, nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_acme")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResponse struct {
		Items []model.KnowledgeAsset `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}

	for _, listedAsset := range listResponse.Items {
		if listedAsset.ID == created.Asset.ID && sameStringSet(listedAsset.KnowledgeBaseIDs, []string{base.ID}) {
			return
		}
	}
	t.Fatalf("reassigned asset not found in base-filtered list: %#v", listResponse.Items)
}

func TestCreateKnowledgeAssetFromJSONProcessesText(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "增长素材",
		"text": "# 增长素材\n\n客户通过内容矩阵提升线索质量，并建立稳定复盘机制。",
		"mimeType": "text/markdown",
		"originalFilename": "growth.md",
		"tags": ["增长", "复盘"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Asset  model.KnowledgeAsset          `json:"asset"`
		Task   model.KnowledgeProcessingTask `json:"task"`
		Chunks []model.KnowledgeChunk        `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}
	if response.Asset.Status != "ready" || response.Asset.Progress != 100 {
		t.Fatalf("asset status/progress = %s/%d, want ready/100", response.Asset.Status, response.Asset.Progress)
	}
	if response.Task.Status != "succeeded" {
		t.Fatalf("task status = %s, want succeeded", response.Task.Status)
	}
	if len(response.Chunks) == 0 {
		t.Fatalf("chunks is empty")
	}
	if !sameStringSet(response.Chunks[0].KnowledgeBaseIDs, []string{"kb_brand"}) {
		t.Fatalf("chunk knowledgeBaseIds = %#v, want kb_brand", response.Chunks[0].KnowledgeBaseIDs)
	}
	if !bytes.Contains([]byte(response.Chunks[0].SearchText), []byte("品牌与产品资料")) {
		t.Fatalf("searchText does not include knowledge base name: %s", response.Chunks[0].SearchText)
	}

	chunkReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets/"+response.Asset.ID+"/chunks", nil)
	chunkReq.Header.Set("Authorization", "Bearer demo-token")
	chunkReq.Header.Set("X-Workspace-ID", "wks_acme")
	chunkRec := httptest.NewRecorder()
	router.ServeHTTP(chunkRec, chunkReq)
	if chunkRec.Code != http.StatusOK {
		t.Fatalf("list chunks status = %d, want %d, body = %s", chunkRec.Code, http.StatusOK, chunkRec.Body.String())
	}
}

func TestCreateKnowledgeAssetAllowsUnclassifiedAsset(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"title": "未分类产品定位",
		"text": "产品定位：面向内容团队的自动发布平台。",
		"mimeType": "text/markdown",
		"originalFilename": "positioning.md"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create unclassified asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal unclassified asset response: %v", err)
	}
	if len(response.Asset.KnowledgeBaseIDs) != 0 {
		t.Fatalf("asset knowledgeBaseIds = %#v, want unclassified empty list", response.Asset.KnowledgeBaseIDs)
	}
	if len(response.Chunks) == 0 {
		t.Fatal("unclassified asset should still produce chunks")
	}
	if len(response.Chunks[0].KnowledgeBaseIDs) != 0 {
		t.Fatalf("chunk knowledgeBaseIds = %#v, want empty list", response.Chunks[0].KnowledgeBaseIDs)
	}
	if !bytes.Contains([]byte(response.Chunks[0].SearchText), []byte("未分类产品定位")) {
		t.Fatalf("searchText should include asset title: %s", response.Chunks[0].SearchText)
	}
}

func TestKnowledgeAssetBasesCanBeCleared(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "可解绑素材",
		"text": "这个资产可以从知识库中移出，成为未分类资产。",
		"mimeType": "text/markdown",
		"originalFilename": "unbind.md"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var created struct {
		Asset model.KnowledgeAsset `json:"asset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}

	clearReq := httptest.NewRequest(http.MethodPut, "/api/knowledge-assets/"+created.Asset.ID+"/bases", bytes.NewBufferString(`{"knowledgeBaseIds":[]}`))
	clearReq.Header.Set("Authorization", "Bearer demo-token")
	clearReq.Header.Set("X-Workspace-ID", "wks_acme")
	clearReq.Header.Set("Content-Type", "application/json")
	clearRec := httptest.NewRecorder()
	router.ServeHTTP(clearRec, clearReq)
	if clearRec.Code != http.StatusOK {
		t.Fatalf("clear bases status = %d, want %d, body = %s", clearRec.Code, http.StatusOK, clearRec.Body.String())
	}

	var cleared struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(clearRec.Body.Bytes(), &cleared); err != nil {
		t.Fatalf("unmarshal cleared response: %v", err)
	}
	if len(cleared.Asset.KnowledgeBaseIDs) != 0 {
		t.Fatalf("asset knowledgeBaseIds = %#v, want empty", cleared.Asset.KnowledgeBaseIDs)
	}
	if len(cleared.Chunks) == 0 || len(cleared.Chunks[0].KnowledgeBaseIDs) != 0 {
		t.Fatalf("chunk knowledgeBaseIds = %#v, want empty", cleared.Chunks)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets?knowledgeBaseId=kb_brand", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_acme")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list assets status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var listResponse struct {
		Items []model.KnowledgeAsset `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	for _, item := range listResponse.Items {
		if item.ID == created.Asset.ID {
			t.Fatalf("cleared asset should not be listed under kb_brand: %#v", listResponse.Items)
		}
	}
}

func TestKnowledgeAssetCanMoveToTrashAndRestore(t *testing.T) {
	router := testWorkspaceRouter()

	createBody := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "可删除素材",
		"text": "这个资产可以进入垃圾箱，再恢复。",
		"mimeType": "text/markdown",
		"originalFilename": "trash.md"
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", createBody)
	createReq.Header.Set("Authorization", "Bearer demo-token")
	createReq.Header.Set("X-Workspace-ID", "wks_acme")
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var created struct {
		Asset model.KnowledgeAsset `json:"asset"`
	}
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created asset: %v", err)
	}

	trashReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+created.Asset.ID+"/trash", nil)
	trashReq.Header.Set("Authorization", "Bearer demo-token")
	trashReq.Header.Set("X-Workspace-ID", "wks_acme")
	trashRec := httptest.NewRecorder()
	router.ServeHTTP(trashRec, trashReq)
	if trashRec.Code != http.StatusOK {
		t.Fatalf("trash asset status = %d, want %d, body = %s", trashRec.Code, http.StatusOK, trashRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_acme")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var listResponse struct {
		Items []model.KnowledgeAsset `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}
	for _, item := range listResponse.Items {
		if item.ID == created.Asset.ID {
			t.Fatalf("trashed asset should not be listed: %#v", listResponse.Items)
		}
	}

	trashListReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-trash", nil)
	trashListReq.Header.Set("Authorization", "Bearer demo-token")
	trashListReq.Header.Set("X-Workspace-ID", "wks_acme")
	trashListRec := httptest.NewRecorder()
	router.ServeHTTP(trashListRec, trashListReq)
	if trashListRec.Code != http.StatusOK {
		t.Fatalf("trash list status = %d, want %d, body = %s", trashListRec.Code, http.StatusOK, trashListRec.Body.String())
	}
	var trashListResponse struct {
		KnowledgeAssets []model.KnowledgeAsset `json:"knowledgeAssets"`
	}
	if err := json.Unmarshal(trashListRec.Body.Bytes(), &trashListResponse); err != nil {
		t.Fatalf("unmarshal trash list response: %v", err)
	}
	if len(trashListResponse.KnowledgeAssets) == 0 {
		t.Fatal("trash list should include deleted asset")
	}

	restoreReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+created.Asset.ID+"/restore", nil)
	restoreReq.Header.Set("Authorization", "Bearer demo-token")
	restoreReq.Header.Set("X-Workspace-ID", "wks_acme")
	restoreRec := httptest.NewRecorder()
	router.ServeHTTP(restoreRec, restoreReq)
	if restoreRec.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want %d, body = %s", restoreRec.Code, http.StatusOK, restoreRec.Body.String())
	}
	restored := getKnowledgeAssetForTest(t, router, created.Asset.ID)
	if restored.Status != "ready" || restored.DeletedAt != nil {
		t.Fatalf("restored asset status/deletedAt = %s/%v, want ready/nil", restored.Status, restored.DeletedAt)
	}
}

func TestKnowledgeAssetProcessingCanBeRetried(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "可重试素材",
		"text": "第一次拆分后，可以重新解析并生成新的 chunk。",
		"mimeType": "text/markdown",
		"originalFilename": "retry.md"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created response: %v", err)
	}
	if len(created.Chunks) == 0 {
		t.Fatal("created chunks should not be empty")
	}

	retryReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+created.Asset.ID+"/retry", nil)
	retryReq.Header.Set("Authorization", "Bearer demo-token")
	retryReq.Header.Set("X-Workspace-ID", "wks_acme")
	retryRec := httptest.NewRecorder()
	router.ServeHTTP(retryRec, retryReq)
	if retryRec.Code != http.StatusOK {
		t.Fatalf("retry status = %d, want %d, body = %s", retryRec.Code, http.StatusOK, retryRec.Body.String())
	}
	var retried struct {
		Asset  model.KnowledgeAsset          `json:"asset"`
		Task   model.KnowledgeProcessingTask `json:"task"`
		Chunks []model.KnowledgeChunk        `json:"chunks"`
	}
	if err := json.Unmarshal(retryRec.Body.Bytes(), &retried); err != nil {
		t.Fatalf("unmarshal retry response: %v", err)
	}
	if retried.Asset.Status != "ready" || retried.Task.Status != "succeeded" || len(retried.Chunks) == 0 {
		t.Fatalf("retry result asset/task/chunks = %s/%s/%d, want ready/succeeded/non-empty", retried.Asset.Status, retried.Task.Status, len(retried.Chunks))
	}
	if !strings.HasPrefix(retried.Task.ID, "kbpt_retry_") {
		t.Fatalf("retry task id = %q, want retry task", retried.Task.ID)
	}
}

func TestReadyKnowledgeAssetCanBeAIEnhancedLater(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "后置增强素材",
		"text": "先用基础拆分上线，之后再使用 AI 增强。",
		"mimeType": "text/markdown",
		"originalFilename": "later-enhance.md",
		"tags": ["后置增强"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created response: %v", err)
	}
	if created.Asset.Status != "ready" || created.Asset.AIEnhancementEnabled || created.Asset.AIEnhancementStatus != "disabled" {
		t.Fatalf("created asset status/ai enabled/ai status = %s/%v/%s, want ready/false/disabled", created.Asset.Status, created.Asset.AIEnhancementEnabled, created.Asset.AIEnhancementStatus)
	}
	if len(created.Chunks) == 0 {
		t.Fatal("created chunks should not be empty")
	}

	enhanceReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+created.Asset.ID+"/ai-enhancement", nil)
	enhanceReq.Header.Set("Authorization", "Bearer demo-token")
	enhanceReq.Header.Set("X-Workspace-ID", "wks_acme")
	enhanceRec := httptest.NewRecorder()
	router.ServeHTTP(enhanceRec, enhanceReq)
	if enhanceRec.Code != http.StatusOK {
		t.Fatalf("enhance status = %d, want %d, body = %s", enhanceRec.Code, http.StatusOK, enhanceRec.Body.String())
	}
	var queued struct {
		Asset  model.KnowledgeAsset          `json:"asset"`
		Task   model.KnowledgeProcessingTask `json:"task"`
		Chunks []model.KnowledgeChunk        `json:"chunks"`
	}
	if err := json.Unmarshal(enhanceRec.Body.Bytes(), &queued); err != nil {
		t.Fatalf("unmarshal enhance response: %v", err)
	}
	if !queued.Asset.AIEnhancementEnabled || queued.Asset.AIEnhancementStatus != "pending" {
		t.Fatalf("queued asset ai enabled/status = %v/%s, want true/pending", queued.Asset.AIEnhancementEnabled, queued.Asset.AIEnhancementStatus)
	}
	if queued.Task.TaskType != "ai_enhance" || queued.Task.Status != "queued" {
		t.Fatalf("queued task = %#v, want queued ai_enhance", queued.Task)
	}
	if len(queued.Chunks) == 0 {
		t.Fatal("enhance response should keep existing chunks")
	}

	finalTasks := waitForKnowledgeTaskStatus(t, router, created.Asset.ID, "ai_enhance", "succeeded")
	if !hasKnowledgeTask(finalTasks, "ai_enhance", "succeeded") {
		t.Fatalf("ai_enhance succeeded task not found: %#v", finalTasks)
	}
	finalAsset := getKnowledgeAssetForTest(t, router, created.Asset.ID)
	if finalAsset.Status != "ready" || finalAsset.AIEnhancementStatus != "succeeded" {
		t.Fatalf("final asset status/ai status = %s/%s, want ready/succeeded", finalAsset.Status, finalAsset.AIEnhancementStatus)
	}
}

func TestReadyKnowledgeAssetAIEnhancementRequiresVIP(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "非 VIP 后置增强素材",
		"text": "普通用户可以基础拆分，但不能后续 AI 增强。",
		"mimeType": "text/markdown",
		"originalFilename": "free-later-enhance.md"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer growth-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created struct {
		Asset model.KnowledgeAsset `json:"asset"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created response: %v", err)
	}

	enhanceReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+created.Asset.ID+"/ai-enhancement", nil)
	enhanceReq.Header.Set("Authorization", "Bearer growth-token")
	enhanceReq.Header.Set("X-Workspace-ID", "wks_acme")
	enhanceRec := httptest.NewRecorder()
	router.ServeHTTP(enhanceRec, enhanceReq)
	if enhanceRec.Code != http.StatusForbidden {
		t.Fatalf("enhance status = %d, want %d, body = %s", enhanceRec.Code, http.StatusForbidden, enhanceRec.Body.String())
	}
}

func TestCreateKnowledgeAssetPDFStoresFailedProcessingTask(t *testing.T) {
	router := testWorkspaceRouter()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.WriteField("knowledgeBaseIds", "kb_brand"); err != nil {
		t.Fatalf("write knowledgeBaseIds: %v", err)
	}
	if err := writer.WriteField("title", "产品白皮书"); err != nil {
		t.Fatalf("write title: %v", err)
	}
	part, err := writer.CreateFormFile("file", "whitepaper.pdf")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("%PDF-1.4\nunsupported pdf body")); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create pdf asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Asset model.KnowledgeAsset          `json:"asset"`
		Task  model.KnowledgeProcessingTask `json:"task"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal pdf response: %v", err)
	}
	if response.Asset.Status != "failed" || response.Task.Status != "failed" {
		t.Fatalf("asset/task status = %s/%s, want failed/failed", response.Asset.Status, response.Task.Status)
	}
	if response.Asset.ErrorMessage == "" || response.Task.ErrorMessage == "" {
		t.Fatalf("expected extraction error messages, got asset=%q task=%q", response.Asset.ErrorMessage, response.Task.ErrorMessage)
	}

	taskReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets/"+response.Asset.ID+"/tasks", nil)
	taskReq.Header.Set("Authorization", "Bearer demo-token")
	taskReq.Header.Set("X-Workspace-ID", "wks_acme")
	taskRec := httptest.NewRecorder()
	router.ServeHTTP(taskRec, taskReq)
	if taskRec.Code != http.StatusOK {
		t.Fatalf("list tasks status = %d, want %d, body = %s", taskRec.Code, http.StatusOK, taskRec.Body.String())
	}

	enhanceReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets/"+response.Asset.ID+"/ai-enhancement", nil)
	enhanceReq.Header.Set("Authorization", "Bearer demo-token")
	enhanceReq.Header.Set("X-Workspace-ID", "wks_acme")
	enhanceRec := httptest.NewRecorder()
	router.ServeHTTP(enhanceRec, enhanceReq)
	if enhanceRec.Code != http.StatusConflict {
		t.Fatalf("enhance failed asset status = %d, want %d, body = %s", enhanceRec.Code, http.StatusConflict, enhanceRec.Body.String())
	}
}

func TestCreateKnowledgeAssetAIEnhancementRequiresVIP(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "增长素材",
		"text": "客户通过内容矩阵提升线索质量。",
		"aiEnhancementEnabled": true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer growth-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestCreateKnowledgeAssetAIEnhancementForVIP(t *testing.T) {
	router := testWorkspaceRouter()

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "增长素材",
		"text": "客户通过内容矩阵提升线索质量，并建立稳定复盘机制。",
		"mimeType": "text/markdown",
		"originalFilename": "growth.md",
		"tags": ["增长", "复盘"],
		"aiEnhancementEnabled": true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Asset  model.KnowledgeAsset          `json:"asset"`
		Task   model.KnowledgeProcessingTask `json:"task"`
		Chunks []model.KnowledgeChunk        `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}
	if response.Asset.Status != "ready" || response.Asset.AIEnhancementStatus != "pending" {
		t.Fatalf("asset status/ai status = %s/%s, want ready/pending", response.Asset.Status, response.Asset.AIEnhancementStatus)
	}
	if response.Task.TaskType != "extract" || response.Task.Status != "succeeded" {
		t.Fatalf("response task = %#v, want succeeded extract task", response.Task)
	}
	if len(response.Chunks) == 0 {
		t.Fatal("default chunks should be returned")
	}
	if bytes.Contains([]byte(response.Asset.ExtractedText), []byte("### 使用边界")) || knowledgeChunksContain(response.Chunks, "仅使用上方已确认信息") {
		t.Fatalf("POST response waited for AI-enhanced markdown: asset=%s chunks=%#v", response.Asset.ExtractedText, response.Chunks)
	}
	if !sameStringSet(response.Chunks[0].KnowledgeBaseIDs, []string{"kb_brand"}) {
		t.Fatalf("chunk knowledgeBaseIds = %#v, want kb_brand", response.Chunks[0].KnowledgeBaseIDs)
	}
	if !knowledgeChunksSearchTextContain(response.Chunks, "品牌与产品资料") {
		t.Fatalf("searchText missing base name: %#v", response.Chunks)
	}

	initialTasks := listKnowledgeAssetTasksForTest(t, router, response.Asset.ID)
	if !hasKnowledgeTaskWithStatuses(initialTasks, "ai_enhance", []string{"queued", "running", "succeeded"}) {
		t.Fatalf("ai_enhance queued/running/succeeded task not found: %#v", initialTasks)
	}

	finalTasks := waitForKnowledgeTaskStatus(t, router, response.Asset.ID, "ai_enhance", "succeeded")
	if !hasKnowledgeTask(finalTasks, "ai_enhance", "succeeded") {
		t.Fatalf("ai_enhance succeeded task not found: %#v", finalTasks)
	}
	finalAsset := getKnowledgeAssetForTest(t, router, response.Asset.ID)
	if finalAsset.Status != "ready" || finalAsset.AIEnhancementStatus != "succeeded" {
		t.Fatalf("final asset status/ai status = %s/%s, want ready/succeeded", finalAsset.Status, finalAsset.AIEnhancementStatus)
	}
	finalChunks := listKnowledgeAssetChunksForTest(t, router, response.Asset.ID)
	if len(finalChunks) == 0 {
		t.Fatal("enhanced chunks should be persisted")
	}
	if !bytes.Contains([]byte(finalAsset.ExtractedText), []byte("### 使用边界")) || !knowledgeChunksContain(finalChunks, "仅使用上方已确认信息") {
		t.Fatalf("chunks were not AI-enhanced markdown: asset=%s chunks=%#v", finalAsset.ExtractedText, finalChunks)
	}
	if !knowledgeChunksSearchTextContain(finalChunks, "品牌与产品资料") ||
		!knowledgeChunksSearchTextContain(finalChunks, "使用边界") {
		t.Fatalf("searchText missing base name or enhanced content: %#v", finalChunks)
	}
}

func TestCreateKnowledgeAssetAIEnhancementFallsBackWhenOpenAIUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	handler := NewWorkspaceHandler(nil, ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderOpenAI}))
	handler.browserLogin = fakeBrowserLoginService{}
	handler.Register(apiGroup, middleware.AuthWithTokenResolver(handler.ResolveUserSession))

	body := bytes.NewBufferString(`{
		"knowledgeBaseIds": ["kb_brand"],
		"title": "增长素材",
		"text": "客户通过内容矩阵提升线索质量。",
		"mimeType": "text/markdown",
		"originalFilename": "growth.md",
		"aiEnhancementEnabled": true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Asset  model.KnowledgeAsset   `json:"asset"`
		Chunks []model.KnowledgeChunk `json:"chunks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}
	if response.Asset.Status != "ready" || response.Asset.AIEnhancementStatus != "pending" {
		t.Fatalf("asset status/ai status = %s/%s, want ready/pending fallback", response.Asset.Status, response.Asset.AIEnhancementStatus)
	}
	if len(response.Chunks) == 0 {
		t.Fatal("fallback should leave usable chunks")
	}

	finalTasks := waitForKnowledgeTaskStatus(t, router, response.Asset.ID, "ai_enhance", "succeeded")
	if !hasKnowledgeTask(finalTasks, "ai_enhance", "succeeded") {
		t.Fatalf("fallback ai_enhance succeeded task not found: %#v", finalTasks)
	}
	finalAsset := getKnowledgeAssetForTest(t, router, response.Asset.ID)
	if finalAsset.Status != "ready" || finalAsset.AIEnhancementStatus != "succeeded" {
		t.Fatalf("final asset status/ai status = %s/%s, want ready/succeeded fallback", finalAsset.Status, finalAsset.AIEnhancementStatus)
	}
	if fallback, ok := finalAsset.Metadata["aiEnhancementFallback"].(bool); !ok || !fallback {
		t.Fatalf("fallback metadata missing: %#v", finalAsset.Metadata)
	}
	if finalAsset.Metadata["aiEnhancementFallbackError"] == "" {
		t.Fatalf("fallback error metadata missing: %#v", finalAsset.Metadata)
	}
}

func TestWorkspaceKnowledgeItemRoutesAreNotRegistered(t *testing.T) {
	router := testWorkspaceRouter()

	cases := []struct {
		method string
		path   string
		body   string
	}{
		{method: http.MethodGet, path: "/api/knowledge-items"},
		{method: http.MethodPost, path: "/api/knowledge-items", body: `{"title":"旧条目","content":"旧内容","knowledgeBaseIds":["kb_brand"]}`},
		{method: http.MethodPost, path: "/api/knowledge-items/assign-bases", body: `{"knowledgeItemIds":["kbi_1001"],"knowledgeBaseIds":["kb_brand"]}`},
		{method: http.MethodPost, path: "/api/knowledge-items/format", body: `{"title":"旧格式化","content":"旧内容"}`},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
		req.Header.Set("Authorization", "Bearer demo-token")
		req.Header.Set("X-Workspace-ID", "wks_acme")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s %s status = %d, want %d, body = %s", tc.method, tc.path, rec.Code, http.StatusNotFound, rec.Body.String())
		}
	}
}

func TestExtractKeywordsFromMarkdownPromptUsesCoreThemes(t *testing.T) {
	keywords := extractKeywordsFromMarkdownPrompt("## 生成目标\n\n- 写增长内容\n\n## 核心主题\n\n- 内容营销\n- 增长\n\n## 事实边界\n\n- 不编造")
	if !sameStringSet(keywords, []string{"内容营销", "增长"}) {
		t.Fatalf("keywords = %#v, want core themes", keywords)
	}
}

func TestAdminCanUpgradeUserSubscriptionForFormatting(t *testing.T) {
	router := testWorkspaceRouter()
	upgradeBody := bytes.NewBufferString(`{
		"subscriptionTier": "vip",
		"subscriptionStatus": "active",
		"subscriptionExpiresAt": "2099-01-01T00:00:00Z"
	}`)
	upgradeReq := httptest.NewRequest(http.MethodPut, "/api/admin/users/usr_growth/subscription", upgradeBody)
	upgradeReq.Header.Set("Authorization", "Bearer demo-token")
	upgradeReq.Header.Set("Content-Type", "application/json")

	upgradeRec := httptest.NewRecorder()
	router.ServeHTTP(upgradeRec, upgradeReq)
	if upgradeRec.Code != http.StatusOK {
		t.Fatalf("upgrade status = %d, want %d, body = %s", upgradeRec.Code, http.StatusOK, upgradeRec.Body.String())
	}

	body := bytes.NewBufferString(`{
		"title": "升级后增强",
		"text": "增长用户升级 VIP 后可以使用 AI 增强资产。",
		"mimeType": "text/markdown",
		"originalFilename": "vip.md",
		"aiEnhancementEnabled": true
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
	req.Header.Set("Authorization", "Bearer growth-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create enhanced asset status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}
}

func TestFreeUserImageAndPDFKnowledgeAssetsRequirePaidOCR(t *testing.T) {
	router := testWorkspaceRouter()

	cases := []struct {
		name             string
		mimeType         string
		originalFilename string
		text             string
	}{
		{
			name:             "image",
			mimeType:         "image/png",
			originalFilename: "poster.png",
			text:             "not-a-real-image",
		},
		{
			name:             "pdf",
			mimeType:         "application/pdf",
			originalFilename: "deck.pdf",
			text:             "%PDF-1.4\nnot-a-real-pdf",
		},
	}
	for _, tc := range cases {
		body := bytes.NewBufferString(fmt.Sprintf(`{
			"title": "OCR %s",
			"text": %q,
			"mimeType": %q,
			"originalFilename": %q
		}`, tc.name, tc.text, tc.mimeType, tc.originalFilename))
		req := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", body)
		req.Header.Set("Authorization", "Bearer growth-token")
		req.Header.Set("X-Workspace-ID", "wks_acme")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s create status = %d, want %d, body = %s", tc.name, rec.Code, http.StatusCreated, rec.Body.String())
		}

		var response struct {
			Asset model.KnowledgeAsset `json:"asset"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("%s unmarshal response: %v", tc.name, err)
		}
		if response.Asset.Status != "failed" {
			t.Fatalf("%s asset status = %q, want failed", tc.name, response.Asset.Status)
		}
		if !strings.Contains(response.Asset.ErrorMessage, "paid subscription") {
			t.Fatalf("%s error = %q, want paid subscription hint", tc.name, response.Asset.ErrorMessage)
		}
		if required, ok := response.Asset.Metadata["ocrRequired"].(bool); !ok || !required {
			t.Fatalf("%s ocrRequired metadata missing: %#v", tc.name, response.Asset.Metadata)
		}
	}
}

func TestGenerateContentAcceptsMultipleKnowledgeBases(t *testing.T) {
	router := testWorkspaceRouter()
	base := createWorkspaceKnowledgeBase(t, router, "生成联调素材包")

	createAssetBody := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseIds": [%q],
		"title": "生成补充案例",
		"text": "多知识库包生成时应能检索到这个内容营销补充案例。",
		"mimeType": "text/markdown",
		"originalFilename": "generation-case.md"
	}`, base.ID))
	createAssetReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-assets", createAssetBody)
	createAssetReq.Header.Set("Authorization", "Bearer demo-token")
	createAssetReq.Header.Set("X-Workspace-ID", "wks_acme")
	createAssetReq.Header.Set("Content-Type", "application/json")
	createAssetRec := httptest.NewRecorder()
	router.ServeHTTP(createAssetRec, createAssetReq)
	if createAssetRec.Code != http.StatusCreated {
		t.Fatalf("create asset status = %d, want %d, body = %s", createAssetRec.Code, http.StatusCreated, createAssetRec.Body.String())
	}

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"keywords": ["内容营销"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand", %q]
	}`, base.ID))
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Content model.Content      `json:"content"`
		Trace   ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated content: %v", err)
	}
	if response.Content.KnowledgeBaseID != "kb_brand" {
		t.Fatalf("primary knowledge base = %q, want kb_brand", response.Content.KnowledgeBaseID)
	}
	if len(response.Trace.Steps) == 0 {
		t.Fatal("expected generation trace steps")
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("内容营销")) {
		t.Fatalf("generated content does not include keyword context: %s", rec.Body.String())
	}
}

func TestGenerateContentPrefersKnowledgeChunksAndRecordsChunkID(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	now := time.Now().UTC()

	handler.mu.Lock()
	handler.knowledgeAssets = append(handler.knowledgeAssets, model.KnowledgeAsset{
		ID:               "kba_generation_ready",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{"kb_brand"},
		Title:            "资产知识素材",
		Status:           "ready",
		UpdatedAt:        now,
	})
	handler.knowledgeAssets = append(handler.knowledgeAssets, model.KnowledgeAsset{
		ID:               "kba_generation_failed",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{"kb_brand"},
		Title:            "失败资产素材",
		Status:           "failed",
		UpdatedAt:        now,
	})
	handler.knowledgeChunks = append(handler.knowledgeChunks,
		model.KnowledgeChunk{
			ID:               "kbc_generation_ready_001",
			AssetID:          "kba_generation_ready",
			WorkspaceID:      "wks_acme",
			KnowledgeBaseIDs: []string{"kb_brand"},
			Title:            "内容营销资产片段",
			Content:          "这是来自 ready 资产的内容营销 chunk，应优先进入生成上下文。",
			SearchText:       "内容营销 ready asset chunk",
			Tags:             []string{"内容营销"},
			Summary:          "ready chunk summary",
			Metadata:         map[string]any{"type": "asset_chunk"},
			Enabled:          true,
			UpdatedAt:        now,
		},
		model.KnowledgeChunk{
			ID:               "kbc_generation_failed_001",
			AssetID:          "kba_generation_failed",
			WorkspaceID:      "wks_acme",
			KnowledgeBaseIDs: []string{"kb_brand"},
			Title:            "内容营销失败片段",
			Content:          "这个 failed 资产 chunk 不应该进入生成上下文。",
			SearchText:       "内容营销 failed asset chunk",
			Tags:             []string{"内容营销"},
			Metadata:         map[string]any{"type": "asset_chunk"},
			Enabled:          true,
			UpdatedAt:        now,
		},
	)
	handler.mu.Unlock()

	body := bytes.NewBufferString(`{
		"keywords": ["内容营销"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if !containsString(response.Trace.RetrievedIDs, "kbc_generation_ready_001") {
		t.Fatalf("trace retrieved ids = %#v, want ready chunk id", response.Trace.RetrievedIDs)
	}
	if containsString(response.Trace.RetrievedIDs, "kbc_generation_failed_001") {
		t.Fatalf("trace retrieved ids includes failed asset chunk: %#v", response.Trace.RetrievedIDs)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("资产片段")) || !bytes.Contains(rec.Body.Bytes(), []byte("asset_chunk")) {
		t.Fatalf("trace should include chunk title and source type: %s", rec.Body.String())
	}

	handler.mu.RLock()
	if len(handler.generations) == 0 {
		handler.mu.RUnlock()
		t.Fatal("expected generation request record")
	}
	retrievedIDs := append([]string(nil), handler.generations[0].RetrievedKnowledgeIDs...)
	handler.mu.RUnlock()
	if len(retrievedIDs) != 1 || retrievedIDs[0] != "kbc_generation_ready_001" {
		t.Fatalf("generation retrieved ids = %#v, want ready chunk only", retrievedIDs)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("kbc_generation_ready_001")) {
		t.Fatalf("generated response should expose used chunk id: %s", rec.Body.String())
	}
}

func TestGenerateContentCanRetrieveUnclassifiedAssetWhenNoBaseSelected(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	now := time.Now().UTC()

	handler.mu.Lock()
	handler.knowledgeAssets = append(handler.knowledgeAssets, model.KnowledgeAsset{
		ID:               "kba_generation_unclassified",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{},
		Title:            "未分类定位素材",
		Status:           "ready",
		UpdatedAt:        now,
	})
	handler.knowledgeChunks = append(handler.knowledgeChunks, model.KnowledgeChunk{
		ID:               "kbc_generation_unclassified_001",
		AssetID:          "kba_generation_unclassified",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{},
		Title:            "未分类产品定位",
		Content:          "未分类资产也可以在未选择知识库时进入工作区级检索。",
		SearchText:       "未分类 产品定位 工作区级检索",
		Tags:             []string{"未分类"},
		Metadata:         map[string]any{"type": "asset_chunk"},
		Enabled:          true,
		UpdatedAt:        now,
	})
	handler.mu.Unlock()

	body := bytes.NewBufferString(`{
		"keywords": ["未分类", "产品定位"],
		"contentType": "article"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate unclassified status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if !containsString(response.Trace.RetrievedIDs, "kbc_generation_unclassified_001") {
		t.Fatalf("trace retrieved ids = %#v, want unclassified chunk id", response.Trace.RetrievedIDs)
	}
}

func TestGenerateContentExcludesUnclassifiedAssetWhenBaseSelected(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	now := time.Now().UTC()

	handler.mu.Lock()
	handler.knowledgeAssets = append(handler.knowledgeAssets, model.KnowledgeAsset{
		ID:               "kba_generation_unclassified_scoped",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{},
		Title:            "未分类定向过滤素材",
		Status:           "ready",
		UpdatedAt:        now,
	})
	handler.knowledgeChunks = append(handler.knowledgeChunks, model.KnowledgeChunk{
		ID:               "kbc_generation_unclassified_scoped_001",
		AssetID:          "kba_generation_unclassified_scoped",
		WorkspaceID:      "wks_acme",
		KnowledgeBaseIDs: []string{},
		Title:            "未分类定向过滤",
		Content:          "用户选择知识库包时，这条未分类资产 chunk 不应进入检索上下文。",
		SearchText:       "品牌定位 未分类 定向过滤",
		Tags:             []string{"品牌定位"},
		Metadata:         map[string]any{"type": "asset_chunk"},
		Enabled:          true,
		UpdatedAt:        now,
	})
	handler.mu.Unlock()

	body := bytes.NewBufferString(`{
		"keywords": ["品牌定位", "未分类"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate scoped status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if containsString(response.Trace.RetrievedIDs, "kbc_generation_unclassified_scoped_001") {
		t.Fatalf("trace retrieved ids includes unclassified chunk despite selected base: %#v", response.Trace.RetrievedIDs)
	}
}

func TestGenerateContentDoesNotFallbackToLegacyKnowledgeItems(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	now := time.Now().UTC()

	handler.mu.Lock()
	handler.knowledgeAssets = nil
	handler.knowledgeChunks = nil
	handler.knowledgeItems = []model.KnowledgeItem{
		{
			ID:               "kbi_legacy_only",
			KnowledgeBaseIDs: []string{"kb_brand"},
			WorkspaceID:      "wks_acme",
			Type:             "brand",
			Title:            "品牌定位",
			Content:          "这条旧知识条目没有对应 chunk，不应被生成检索直接读取。",
			Enabled:          true,
			UpdatedAt:        now,
		},
	}
	handler.mu.Unlock()

	body := bytes.NewBufferString(`{
		"keywords": ["品牌定位"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if len(response.Trace.RetrievedIDs) != 0 {
		t.Fatalf("trace retrieved ids = %#v, want no legacy item fallback", response.Trace.RetrievedIDs)
	}

	handler.mu.RLock()
	if len(handler.generations) == 0 {
		handler.mu.RUnlock()
		t.Fatal("expected generation request record")
	}
	retrievedIDs := append([]string(nil), handler.generations[0].RetrievedKnowledgeIDs...)
	handler.mu.RUnlock()
	if len(retrievedIDs) != 0 {
		t.Fatalf("generation retrieved ids = %#v, want no legacy item fallback", retrievedIDs)
	}
}

func TestSeededLegacyKnowledgeItemsHaveAssetChunksForGeneration(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()

	handler.mu.RLock()
	hasAsset := false
	hasChunk := false
	for _, asset := range handler.knowledgeAssets {
		if asset.ID == "kba_legacy_kbi_1001" && asset.Status == "ready" {
			hasAsset = true
		}
	}
	for _, chunk := range handler.knowledgeChunks {
		if chunk.ID == "kbc_legacy_kbi_1001_0000" && chunk.AssetID == "kba_legacy_kbi_1001" {
			hasChunk = true
		}
	}
	handler.mu.RUnlock()
	if !hasAsset || !hasChunk {
		t.Fatalf("seeded legacy asset/chunk missing: hasAsset=%v hasChunk=%v", hasAsset, hasChunk)
	}

	body := bytes.NewBufferString(`{
		"keywords": ["品牌定位"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if !containsString(response.Trace.RetrievedIDs, "kbc_legacy_kbi_1001_0000") {
		t.Fatalf("trace retrieved ids = %#v, want migrated legacy chunk id", response.Trace.RetrievedIDs)
	}
}

func TestGenerateContentTraceUsesVIPPipeline(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"keywords": ["内容营销"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"]
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var response struct {
		Trace ai.GenerationTrace `json:"trace"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal generated response: %v", err)
	}
	if response.Trace.SubscriptionTier != string(model.SubscriptionTierVIP) {
		t.Fatalf("subscription tier = %q, want vip", response.Trace.SubscriptionTier)
	}
	if !response.Trace.Pipeline.QualityCheck || response.Trace.Pipeline.RewriteRounds != 1 {
		t.Fatalf("unexpected vip pipeline: %#v", response.Trace.Pipeline)
	}
	if !traceHasStep(response.Trace, ai.GenerationStageQualityCheck, "succeeded") || !traceHasStep(response.Trace, ai.GenerationStageRewrite, "succeeded") {
		t.Fatalf("expected quality check and rewrite steps: %#v", response.Trace.Steps)
	}
}

func TestGenerateContentRejectsUninstalledSkillPackageVersion(t *testing.T) {
	router := testWorkspaceRouter()

	createBody := bytes.NewBufferString(`{
		"name": "小红书爆款结构包",
		"slug": "xhs-growth-structure",
		"description": "用于测试未安装技能包的权益校验",
		"category": "xiaohongshu",
		"targetPlatform": "xiaohongshu",
		"supportedContentFormats": ["article"],
		"listingStatus": "published",
		"version": {
			"id": "skv_test_uninstalled",
			"version": "1.0.0",
			"promptContract": "补充内容结构约束",
			"outputSchema": "{}"
		}
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/skill-packages", createBody)
	createReq.Header.Set("Authorization", "Bearer demo-token")
	createReq.Header.Set("X-Workspace-ID", "wks_acme")
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create skill package status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}

	body := bytes.NewBufferString(`{
		"keywords": ["内容营销"],
		"contentType": "article",
		"knowledgeBaseIds": ["kb_brand"],
		"skillPackageVersionId": "skv_test_uninstalled"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("generate status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestGenerateContentRejectsUnsupportedContentType(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"keywords": ["内容营销"],
		"contentType": "ignore_previous_instructions"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/contents/generate", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}

func TestCreatorCollaborationWorkflow(t *testing.T) {
	router := testWorkspaceRouter()

	listReq := httptest.NewRequest(http.MethodGet, "/api/creators", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_acme")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("creator list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}
	var creatorList struct {
		Items []model.Creator `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &creatorList); err != nil {
		t.Fatalf("unmarshal creator list: %v", err)
	}
	if len(creatorList.Items) == 0 {
		t.Fatal("expected seeded creators")
	}
	creatorID := creatorList.Items[0].ID

	detailReq := httptest.NewRequest(http.MethodGet, "/api/creators/"+creatorID, nil)
	detailReq.Header.Set("Authorization", "Bearer demo-token")
	detailReq.Header.Set("X-Workspace-ID", "wks_acme")
	detailRec := httptest.NewRecorder()
	router.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK {
		t.Fatalf("creator detail status = %d, want %d, body = %s", detailRec.Code, http.StatusOK, detailRec.Body.String())
	}
	var detail struct {
		Creator       model.Creator               `json:"creator"`
		MediaAccounts []model.CreatorMediaAccount `json:"mediaAccounts"`
		Shortlists    []model.CreatorShortlist    `json:"shortlists"`
	}
	if err := json.Unmarshal(detailRec.Body.Bytes(), &detail); err != nil {
		t.Fatalf("unmarshal creator detail: %v", err)
	}
	if detail.Creator.ID != creatorID || len(detail.MediaAccounts) == 0 {
		t.Fatalf("unexpected creator detail: %#v", detail)
	}
	if detail.MediaAccounts[0].AccountAccessMode != "creator_operated" {
		t.Fatalf("creator account should not be tenant-login managed: %#v", detail.MediaAccounts[0])
	}

	shortlistBody := bytes.NewBufferString(fmt.Sprintf(`{
		"creatorId": %q,
		"name": "Q3 Launch",
		"fitScore": 86,
		"qualificationStatus": "qualified",
		"brandSafetyLevel": "low",
		"brandSafetyNotes": "历史内容无明显高风险宣称",
		"operatorNotes": "适合新品教育内容"
	}`, creatorID))
	shortlistReq := httptest.NewRequest(http.MethodPost, "/api/creator-shortlists", shortlistBody)
	shortlistReq.Header.Set("Authorization", "Bearer demo-token")
	shortlistReq.Header.Set("X-Workspace-ID", "wks_acme")
	shortlistReq.Header.Set("Content-Type", "application/json")
	shortlistRec := httptest.NewRecorder()
	router.ServeHTTP(shortlistRec, shortlistReq)
	if shortlistRec.Code != http.StatusCreated {
		t.Fatalf("shortlist status = %d, want %d, body = %s", shortlistRec.Code, http.StatusCreated, shortlistRec.Body.String())
	}

	briefBody := bytes.NewBufferString(`{
		"title": "Q3 产品发布达人合作",
		"objective": "教育潜在客户理解产品场景",
		"productName": "Acme Growth",
		"targetAudience": "B2B SaaS 市场负责人",
		"platformTargets": ["小红书"],
		"deliverableRequirements": ["1 篇图文笔记", "发布后保留 30 天"],
		"disclosureRequirements": ["正文需标注品牌合作"],
		"prohibitedClaims": ["不得承诺增长效果"],
		"authorizationScope": "达人自行发布，品牌不得登录达人账号",
		"contentUsageRights": "品牌可在自有渠道二次使用 90 天",
		"reviewWindowHours": 48,
		"budgetCents": 100000,
		"currency": "cny",
		"status": "active"
	}`)
	briefReq := httptest.NewRequest(http.MethodPost, "/api/creator-briefs", briefBody)
	briefReq.Header.Set("Authorization", "Bearer demo-token")
	briefReq.Header.Set("X-Workspace-ID", "wks_acme")
	briefReq.Header.Set("Content-Type", "application/json")
	briefRec := httptest.NewRecorder()
	router.ServeHTTP(briefRec, briefReq)
	if briefRec.Code != http.StatusCreated {
		t.Fatalf("brief status = %d, want %d, body = %s", briefRec.Code, http.StatusCreated, briefRec.Body.String())
	}
	var brief model.CreatorCampaignBrief
	if err := json.Unmarshal(briefRec.Body.Bytes(), &brief); err != nil {
		t.Fatalf("unmarshal brief: %v", err)
	}
	if brief.Currency != "CNY" || len(brief.DisclosureRequirements) == 0 || brief.AuthorizationScope == "" || brief.ContentUsageRights == "" {
		t.Fatalf("brief missed compliance contract fields: %#v", brief)
	}

	orderBody := bytes.NewBufferString(fmt.Sprintf(`{
		"briefId": %q,
		"creatorId": %q,
		"depositCents": 30000,
		"lastMessage": "请按 brief 提交初稿"
	}`, brief.ID, creatorID))
	orderReq := httptest.NewRequest(http.MethodPost, "/api/creator-orders", orderBody)
	orderReq.Header.Set("Authorization", "Bearer demo-token")
	orderReq.Header.Set("X-Workspace-ID", "wks_acme")
	orderReq.Header.Set("Content-Type", "application/json")
	orderRec := httptest.NewRecorder()
	router.ServeHTTP(orderRec, orderReq)
	if orderRec.Code != http.StatusCreated {
		t.Fatalf("order status = %d, want %d, body = %s", orderRec.Code, http.StatusCreated, orderRec.Body.String())
	}
	var orderResponse struct {
		Order      model.CreatorOrder      `json:"order"`
		Settlement model.CreatorSettlement `json:"settlement"`
	}
	if err := json.Unmarshal(orderRec.Body.Bytes(), &orderResponse); err != nil {
		t.Fatalf("unmarshal order: %v", err)
	}
	order := orderResponse.Order
	if order.Status != model.CreatorOrderProposed || order.PriceCents != brief.BudgetCents || order.AuthorizationScope != brief.AuthorizationScope {
		t.Fatalf("unexpected order: %#v", order)
	}
	if orderResponse.Settlement.Status != model.CreatorSettlementPending || orderResponse.Settlement.CreatorPayoutCents != 90000 {
		t.Fatalf("unexpected settlement: %#v", orderResponse.Settlement)
	}

	deliverableBody := bytes.NewBufferString(`{
		"type": "draft",
		"title": "产品发布笔记初稿",
		"content": "这是一版达人自行创作的品牌合作内容。",
		"assetUrls": ["https://example.com/assets/draft-1.png"]
	}`)
	deliverableReq := httptest.NewRequest(http.MethodPost, "/api/creator-orders/"+order.ID+"/deliverables", deliverableBody)
	deliverableReq.Header.Set("Authorization", "Bearer demo-token")
	deliverableReq.Header.Set("X-Workspace-ID", "wks_acme")
	deliverableReq.Header.Set("Content-Type", "application/json")
	deliverableRec := httptest.NewRecorder()
	router.ServeHTTP(deliverableRec, deliverableReq)
	if deliverableRec.Code != http.StatusCreated {
		t.Fatalf("deliverable status = %d, want %d, body = %s", deliverableRec.Code, http.StatusCreated, deliverableRec.Body.String())
	}
	var deliverable model.CreatorDeliverable
	if err := json.Unmarshal(deliverableRec.Body.Bytes(), &deliverable); err != nil {
		t.Fatalf("unmarshal deliverable: %v", err)
	}
	if deliverable.Status != model.CreatorDeliverableSubmitted || deliverable.Revision != 1 {
		t.Fatalf("unexpected deliverable: %#v", deliverable)
	}

	duplicateSubmitReq := httptest.NewRequest(http.MethodPost, "/api/creator-orders/"+order.ID+"/deliverables", bytes.NewBufferString(`{"title":"重复提交"}`))
	duplicateSubmitReq.Header.Set("Authorization", "Bearer demo-token")
	duplicateSubmitReq.Header.Set("X-Workspace-ID", "wks_acme")
	duplicateSubmitReq.Header.Set("Content-Type", "application/json")
	duplicateSubmitRec := httptest.NewRecorder()
	router.ServeHTTP(duplicateSubmitRec, duplicateSubmitReq)
	if duplicateSubmitRec.Code != http.StatusConflict {
		t.Fatalf("duplicate submit status = %d, want %d, body = %s", duplicateSubmitRec.Code, http.StatusConflict, duplicateSubmitRec.Body.String())
	}

	reviewBody := bytes.NewBufferString(`{
		"decision": "approve",
		"feedback": "披露和禁用宣称都符合 brief"
	}`)
	reviewReq := httptest.NewRequest(http.MethodPost, "/api/creator-deliverables/"+deliverable.ID+"/review", reviewBody)
	reviewReq.Header.Set("Authorization", "Bearer demo-token")
	reviewReq.Header.Set("X-Workspace-ID", "wks_acme")
	reviewReq.Header.Set("Content-Type", "application/json")
	reviewRec := httptest.NewRecorder()
	router.ServeHTTP(reviewRec, reviewReq)
	if reviewRec.Code != http.StatusOK {
		t.Fatalf("review status = %d, want %d, body = %s", reviewRec.Code, http.StatusOK, reviewRec.Body.String())
	}
	var reviewed model.CreatorDeliverable
	if err := json.Unmarshal(reviewRec.Body.Bytes(), &reviewed); err != nil {
		t.Fatalf("unmarshal reviewed deliverable: %v", err)
	}
	if reviewed.Status != model.CreatorDeliverableApproved || reviewed.ReviewedAt == nil {
		t.Fatalf("unexpected reviewed deliverable: %#v", reviewed)
	}

	proofWithoutDisclosureReq := httptest.NewRequest(http.MethodPost, "/api/creator-deliverables/"+deliverable.ID+"/publication-proof", bytes.NewBufferString(`{
		"externalUrl": "https://example.com/posts/creator-1"
	}`))
	proofWithoutDisclosureReq.Header.Set("Authorization", "Bearer demo-token")
	proofWithoutDisclosureReq.Header.Set("X-Workspace-ID", "wks_acme")
	proofWithoutDisclosureReq.Header.Set("Content-Type", "application/json")
	proofWithoutDisclosureRec := httptest.NewRecorder()
	router.ServeHTTP(proofWithoutDisclosureRec, proofWithoutDisclosureReq)
	if proofWithoutDisclosureRec.Code != http.StatusBadRequest {
		t.Fatalf("proof without disclosure status = %d, want %d, body = %s", proofWithoutDisclosureRec.Code, http.StatusBadRequest, proofWithoutDisclosureRec.Body.String())
	}

	proofBody := bytes.NewBufferString(`{
		"externalUrl": "https://example.com/posts/creator-1",
		"publicationProofUrl": "https://example.com/proofs/screenshot-1.png",
		"publicationProofNote": "截图保留了发布时间和账号主页",
		"disclosureText": "品牌合作：Acme Growth",
		"notes": "已确认外链可访问"
	}`)
	proofReq := httptest.NewRequest(http.MethodPost, "/api/creator-deliverables/"+deliverable.ID+"/publication-proof", proofBody)
	proofReq.Header.Set("Authorization", "Bearer demo-token")
	proofReq.Header.Set("X-Workspace-ID", "wks_acme")
	proofReq.Header.Set("Content-Type", "application/json")
	proofRec := httptest.NewRecorder()
	router.ServeHTTP(proofRec, proofReq)
	if proofRec.Code != http.StatusOK {
		t.Fatalf("proof status = %d, want %d, body = %s", proofRec.Code, http.StatusOK, proofRec.Body.String())
	}
	var proofResponse struct {
		Deliverable model.CreatorDeliverable        `json:"deliverable"`
		Order       model.CreatorOrder              `json:"order"`
		Settlement  model.CreatorSettlement         `json:"settlement"`
		Evidence    model.CreatorComplianceEvidence `json:"evidence"`
	}
	if err := json.Unmarshal(proofRec.Body.Bytes(), &proofResponse); err != nil {
		t.Fatalf("unmarshal proof response: %v", err)
	}
	if proofResponse.Deliverable.Status != model.CreatorDeliverablePublished || proofResponse.Order.Status != model.CreatorOrderPublished {
		t.Fatalf("unexpected proof state: %#v", proofResponse)
	}
	if proofResponse.Settlement.Status != model.CreatorSettlementPayable {
		t.Fatalf("settlement status = %q, want payable", proofResponse.Settlement.Status)
	}
	if proofResponse.Evidence.EvidenceType != model.CreatorEvidencePublicationProof ||
		proofResponse.Evidence.DisclosureText == "" ||
		proofResponse.Evidence.AuthorizationScope == "" ||
		proofResponse.Evidence.ContentUsageRights == "" {
		t.Fatalf("publication evidence missed compliance fields: %#v", proofResponse.Evidence)
	}

	settlementReq := httptest.NewRequest(http.MethodGet, "/api/creator-settlements", nil)
	settlementReq.Header.Set("Authorization", "Bearer demo-token")
	settlementReq.Header.Set("X-Workspace-ID", "wks_acme")
	settlementRec := httptest.NewRecorder()
	router.ServeHTTP(settlementRec, settlementReq)
	if settlementRec.Code != http.StatusOK {
		t.Fatalf("settlement list status = %d, want %d, body = %s", settlementRec.Code, http.StatusOK, settlementRec.Body.String())
	}
	if !bytes.Contains(settlementRec.Body.Bytes(), []byte(`"status":"payable"`)) {
		t.Fatalf("settlement list does not include payable settlement: %s", settlementRec.Body.String())
	}
}

func TestBrandComplianceApprovalAndReportsAPIs(t *testing.T) {
	router := testWorkspaceRouter()

	assetReq := httptest.NewRequest(http.MethodPost, "/api/brand-assets", bytes.NewBufferString(`{
		"type": "forbidden_phrase",
		"name": "禁用承诺表达",
		"content": "稳赚",
		"channels": ["xiaohongshu"],
		"tags": ["compliance"],
		"metadata": {"owner": "legal"}
	}`))
	assetReq.Header.Set("Authorization", "Bearer demo-token")
	assetReq.Header.Set("X-Workspace-ID", "wks_acme")
	assetReq.Header.Set("Content-Type", "application/json")
	assetRec := httptest.NewRecorder()
	router.ServeHTTP(assetRec, assetReq)
	if assetRec.Code != http.StatusCreated {
		t.Fatalf("brand asset status = %d, want %d, body = %s", assetRec.Code, http.StatusCreated, assetRec.Body.String())
	}

	var asset model.BrandAsset
	if err := json.Unmarshal(assetRec.Body.Bytes(), &asset); err != nil {
		t.Fatalf("unmarshal asset: %v", err)
	}
	if asset.ID == "" || asset.Status != model.BrandAssetActive || !sameStringSet(asset.Channels, []string{"xiaohongshu"}) {
		t.Fatalf("unexpected asset: %#v", asset)
	}

	guardrailBody := fmt.Sprintf(`{
		"assetId": %q,
		"name": "禁止收益保证",
		"category": "claim_risk",
		"channel": "xiaohongshu",
		"severity": "high",
		"rules": ["稳赚"],
		"action": "提交法务复核"
	}`, asset.ID)
	guardrailReq := httptest.NewRequest(http.MethodPost, "/api/brand-guardrails", bytes.NewBufferString(guardrailBody))
	guardrailReq.Header.Set("Authorization", "Bearer demo-token")
	guardrailReq.Header.Set("X-Workspace-ID", "wks_acme")
	guardrailReq.Header.Set("Content-Type", "application/json")
	guardrailRec := httptest.NewRecorder()
	router.ServeHTTP(guardrailRec, guardrailReq)
	if guardrailRec.Code != http.StatusCreated {
		t.Fatalf("guardrail status = %d, want %d, body = %s", guardrailRec.Code, http.StatusCreated, guardrailRec.Body.String())
	}

	checkReq := httptest.NewRequest(http.MethodPost, "/api/compliance-checks", bytes.NewBufferString(`{
		"resourceType": "content",
		"channel": "xiaohongshu",
		"title": "高收益方案",
		"content": "这个方案保证稳赚，欢迎合作。联系人 13800138000"
	}`))
	checkReq.Header.Set("Authorization", "Bearer demo-token")
	checkReq.Header.Set("X-Workspace-ID", "wks_acme")
	checkReq.Header.Set("Content-Type", "application/json")
	checkRec := httptest.NewRecorder()
	router.ServeHTTP(checkRec, checkReq)
	if checkRec.Code != http.StatusCreated {
		t.Fatalf("compliance check status = %d, want %d, body = %s", checkRec.Code, http.StatusCreated, checkRec.Body.String())
	}

	var check model.ComplianceCheck
	if err := json.Unmarshal(checkRec.Body.Bytes(), &check); err != nil {
		t.Fatalf("unmarshal compliance check: %v", err)
	}
	if check.Status != model.ComplianceCheckCompleted || check.RiskLevel != "high" || len(check.Findings) < 3 {
		t.Fatalf("unexpected compliance check: %#v", check)
	}
	for _, finding := range check.Findings {
		if finding.Evidence == "" || finding.Finding == "" || finding.Action == "" {
			t.Fatalf("finding misses evidence/finding/action: %#v", finding)
		}
	}

	workflowReq := httptest.NewRequest(http.MethodPost, "/api/approval-workflows", bytes.NewBufferString(`{
		"resourceType": "content",
		"resourceId": "cnt_1001",
		"name": "品牌法务双审",
		"stages": [
			{"name": "品牌审核", "approverRole": "brand_manager", "requiredApprovals": 1},
			{"name": "法务审核", "approverRole": "legal", "requiredApprovals": 1}
		]
	}`))
	workflowReq.Header.Set("Authorization", "Bearer demo-token")
	workflowReq.Header.Set("X-Workspace-ID", "wks_acme")
	workflowReq.Header.Set("Content-Type", "application/json")
	workflowRec := httptest.NewRecorder()
	router.ServeHTTP(workflowRec, workflowReq)
	if workflowRec.Code != http.StatusCreated {
		t.Fatalf("workflow status = %d, want %d, body = %s", workflowRec.Code, http.StatusCreated, workflowRec.Body.String())
	}

	var workflowResponse struct {
		Workflow model.ApprovalWorkflow `json:"workflow"`
		Tasks    []model.ApprovalTask   `json:"tasks"`
	}
	if err := json.Unmarshal(workflowRec.Body.Bytes(), &workflowResponse); err != nil {
		t.Fatalf("unmarshal workflow response: %v", err)
	}
	if workflowResponse.Workflow.Status != model.ApprovalWorkflowActive || len(workflowResponse.Tasks) != 2 {
		t.Fatalf("unexpected workflow response: %#v", workflowResponse)
	}

	taskID := workflowResponse.Tasks[0].ID
	processReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/approval-tasks/%s/process", taskID), bytes.NewBufferString(`{
		"decision": "approve",
		"comment": "品牌表达通过"
	}`))
	processReq.Header.Set("Authorization", "Bearer demo-token")
	processReq.Header.Set("X-Workspace-ID", "wks_acme")
	processReq.Header.Set("Content-Type", "application/json")
	processRec := httptest.NewRecorder()
	router.ServeHTTP(processRec, processReq)
	if processRec.Code != http.StatusOK {
		t.Fatalf("process status = %d, want %d, body = %s", processRec.Code, http.StatusOK, processRec.Body.String())
	}

	var processed model.ApprovalTask
	if err := json.Unmarshal(processRec.Body.Bytes(), &processed); err != nil {
		t.Fatalf("unmarshal processed task: %v", err)
	}
	if processed.Status != model.ApprovalTaskApproved || processed.ProcessedByUserID != "usr_demo" {
		t.Fatalf("unexpected processed task: %#v", processed)
	}

	reportReq := httptest.NewRequest(http.MethodPost, "/api/report-packages/generate", bytes.NewBufferString(`{
		"name": "六月经营报告",
		"reportType": "monthly",
		"periodStart": "2026-06-01",
		"periodEnd": "2026-06-30",
		"sections": ["content_delivery", "compliance_risks"]
	}`))
	reportReq.Header.Set("Authorization", "Bearer demo-token")
	reportReq.Header.Set("X-Workspace-ID", "wks_acme")
	reportReq.Header.Set("Content-Type", "application/json")
	reportRec := httptest.NewRecorder()
	router.ServeHTTP(reportRec, reportReq)
	if reportRec.Code != http.StatusCreated {
		t.Fatalf("report status = %d, want %d, body = %s", reportRec.Code, http.StatusCreated, reportRec.Body.String())
	}

	var report model.ReportPackage
	if err := json.Unmarshal(reportRec.Body.Bytes(), &report); err != nil {
		t.Fatalf("unmarshal report: %v", err)
	}
	if report.Status != "generated" || len(report.Sections) != 2 || report.Metrics["brandAssetCount"] == nil {
		t.Fatalf("unexpected report: %#v", report)
	}

	recommendationReq := httptest.NewRequest(http.MethodGet, "/api/strategy-recommendations", nil)
	recommendationReq.Header.Set("Authorization", "Bearer demo-token")
	recommendationReq.Header.Set("X-Workspace-ID", "wks_acme")
	recommendationRec := httptest.NewRecorder()
	router.ServeHTTP(recommendationRec, recommendationReq)
	if recommendationRec.Code != http.StatusOK {
		t.Fatalf("recommendations status = %d, want %d, body = %s", recommendationRec.Code, http.StatusOK, recommendationRec.Body.String())
	}

	var recommendations struct {
		Items []model.StrategyRecommendation `json:"items"`
	}
	if err := json.Unmarshal(recommendationRec.Body.Bytes(), &recommendations); err != nil {
		t.Fatalf("unmarshal recommendations: %v", err)
	}
	if len(recommendations.Items) == 0 {
		t.Fatalf("expected placeholder recommendations, got %#v", recommendations.Items)
	}
}

func traceHasStep(trace ai.GenerationTrace, id string, status string) bool {
	for _, step := range trace.Steps {
		if step.ID == id && step.Status == status {
			return true
		}
	}
	return false
}

func TestPublishResultSucceededTreatsLeftEditorAsSubmitted(t *testing.T) {
	result := publishing.PublishResult{
		Status: "submitted_pending_verification",
		RawResponse: map[string]any{
			"rawStatus": map[string]any{
				"publishOutcome": map[string]any{
					"leftEditor": true,
				},
			},
		},
	}
	if !publishResultSucceeded(result) {
		t.Fatal("left-editor publish outcome should be treated as succeeded")
	}

	result.RawResponse = map[string]any{
		"rawStatus": map[string]any{
			"publishOutcome": map[string]any{
				"stillOnFinalSettings": true,
			},
		},
	}
	if publishResultSucceeded(result) {
		t.Fatal("still-on-settings publish outcome should not be treated as succeeded")
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

func TestSohuPhoneSMSInteractiveLoginFlow(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	handler.interactiveLoginForPlatform = func(platformType string) (interactiveLoginService, bool) {
		if platformType != platformTypeSohu {
			return nil, false
		}
		return fakeInteractiveLoginService{}, true
	}

	body := bytes.NewBufferString(`{
		"platformId": "plt_sohu",
		"name": "搜狐号账号",
		"externalId": "sohu-demo",
		"loginMethod": "phone",
		"phoneNumber": "13800000000"
	}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/media-accounts", body)
	createReq.Header.Set("Authorization", "Bearer demo-token")
	createReq.Header.Set("X-Workspace-ID", "wks_personal")
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d, body = %s", createRec.Code, http.StatusCreated, createRec.Body.String())
	}
	var account model.MediaAccount
	if err := json.Unmarshal(createRec.Body.Bytes(), &account); err != nil {
		t.Fatalf("unmarshal account: %v", err)
	}
	if account.Status != "pending_login" || account.LoginMethod != "phone" {
		t.Fatalf("unexpected account login state: %#v", account)
	}

	startReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media-accounts/%s/auth/start", account.ID), bytes.NewBufferString(`{}`))
	startReq.Header.Set("Authorization", "Bearer demo-token")
	startReq.Header.Set("X-Workspace-ID", "wks_personal")
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	router.ServeHTTP(startRec, startReq)
	if startRec.Code != http.StatusOK {
		t.Fatalf("start status = %d, want %d, body = %s", startRec.Code, http.StatusOK, startRec.Body.String())
	}
	var startResponse struct {
		SessionID string                                `json:"sessionId"`
		Strategy  string                                `json:"strategy"`
		State     browserplatform.InteractiveLoginState `json:"state"`
	}
	if err := json.Unmarshal(startRec.Body.Bytes(), &startResponse); err != nil {
		t.Fatalf("unmarshal start response: %v", err)
	}
	if startResponse.Strategy != string(mediaAuthStrategyKindPhoneSMSBrowser) || startResponse.State.Status != "phone_required" {
		t.Fatalf("unexpected start response: %#v", startResponse)
	}

	actionBody := bytes.NewBufferString(fmt.Sprintf(`{"sessionId":%q,"action":"submit_phone","phoneNumber":"13800000000"}`, startResponse.SessionID))
	actionReq := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media-accounts/%s/auth/actions", account.ID), actionBody)
	actionReq.Header.Set("Authorization", "Bearer demo-token")
	actionReq.Header.Set("X-Workspace-ID", "wks_personal")
	actionReq.Header.Set("Content-Type", "application/json")
	actionRec := httptest.NewRecorder()
	router.ServeHTTP(actionRec, actionReq)
	if actionRec.Code != http.StatusOK {
		t.Fatalf("action status = %d, want %d, body = %s", actionRec.Code, http.StatusOK, actionRec.Body.String())
	}
	var actionResponse struct {
		State browserplatform.InteractiveLoginState `json:"state"`
	}
	if err := json.Unmarshal(actionRec.Body.Bytes(), &actionResponse); err != nil {
		t.Fatalf("unmarshal action response: %v", err)
	}
	if actionResponse.State.Status != "captcha_required" || actionResponse.State.CaptchaScreenshotData == "" {
		t.Fatalf("unexpected action state: %#v", actionResponse.State)
	}
}

func TestSohuPhoneSMSInteractiveLoginStartReusesActiveSession(t *testing.T) {
	router, handler := testWorkspaceRouterWithHandler()
	startCalls := 0
	handler.interactiveLoginForPlatform = func(platformType string) (interactiveLoginService, bool) {
		if platformType != platformTypeSohu {
			return nil, false
		}
		return fakeInteractiveLoginService{startCalls: &startCalls}, true
	}

	account := createSohuPhoneAccount(t, router)
	first := startSohuPhoneAuthForTest(t, router, account.ID)
	second := startSohuPhoneAuthForTest(t, router, account.ID)

	if startCalls != 1 {
		t.Fatalf("interactive login Start called %d times, want 1", startCalls)
	}
	if first.SessionID == "" || second.SessionID != first.SessionID {
		t.Fatalf("expected reused session id, first=%q second=%q", first.SessionID, second.SessionID)
	}
	if !second.Reused {
		t.Fatalf("second start reused = false, want true")
	}
	if second.State.Status != "phone_required" {
		t.Fatalf("second state status = %q, want phone_required", second.State.Status)
	}
}

func TestMediaAuthStrategyRegistryResolvesQRBrowserOnlyForBrowserAuthorization(t *testing.T) {
	registry := mediaAuthStrategyRegistry{
		browserLoginForPlatform: func(platformType string) (xiaohongshu.BrowserLoginService, string) {
			return fakeBrowserLoginService{}, "https://example.test/login"
		},
	}
	account := model.MediaAccount{LoginMethod: "qr"}
	platform := model.MediaPlatform{
		ID:      "plt_strategy_qr",
		Name:    "策略二维码平台",
		Type:    "strategy_qr",
		Enabled: true,
		Capabilities: domain.MediaPlatformCapabilities{
			AuthorizationMethods: []domain.AuthorizationMethod{domain.AuthorizationMethodQRLogin},
			Capabilities: []domain.ConnectorCapabilityContract{
				{
					Name:    domain.ConnectorCapabilityAuthorization,
					Mode:    domain.ConnectorCapabilityModeBrowser,
					Enabled: true,
				},
			},
		},
	}

	strategy, ok := registry.Resolve(platform, account)
	if !ok {
		t.Fatal("expected qr browser auth strategy")
	}
	if strategy.Kind() != mediaAuthStrategyKindQRBrowser || !strategy.SupportsBrowserLogin() {
		t.Fatalf("unexpected strategy: kind=%s browser=%v", strategy.Kind(), strategy.SupportsBrowserLogin())
	}
}

func TestMediaAuthStrategyRegistryRejectsManualAuthorizationForBrowserLogin(t *testing.T) {
	registry := mediaAuthStrategyRegistry{
		browserLoginForPlatform: func(platformType string) (xiaohongshu.BrowserLoginService, string) {
			return fakeBrowserLoginService{}, "https://example.test/login"
		},
	}
	account := model.MediaAccount{LoginMethod: "manual"}
	platform := model.MediaPlatform{
		ID:      "plt_strategy_manual",
		Name:    "手动授权平台",
		Type:    "strategy_manual",
		Enabled: true,
		Capabilities: domain.MediaPlatformCapabilities{
			AuthorizationMethods: []domain.AuthorizationMethod{domain.AuthorizationMethodManualOnly},
			Capabilities: []domain.ConnectorCapabilityContract{
				{
					Name:    domain.ConnectorCapabilityAuthorization,
					Mode:    domain.ConnectorCapabilityModeManual,
					Enabled: true,
				},
			},
		},
	}

	if strategy, ok := registry.Resolve(platform, account); ok {
		t.Fatalf("manual platform should not resolve browser login strategy: %#v", strategy)
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

func createSohuPhoneAccount(t *testing.T, router *gin.Engine) model.MediaAccount {
	t.Helper()

	body := bytes.NewBufferString(`{
		"platformId": "plt_sohu",
		"name": "搜狐号账号",
		"externalId": "sohu-demo",
		"loginMethod": "phone",
		"phoneNumber": "13800000000"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/media-accounts", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create sohu account status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var account model.MediaAccount
	if err := json.Unmarshal(rec.Body.Bytes(), &account); err != nil {
		t.Fatalf("unmarshal sohu account response: %v", err)
	}
	return account
}

type mediaAccountAuthStartTestResponse struct {
	SessionID string                                `json:"sessionId"`
	Strategy  string                                `json:"strategy"`
	Reused    bool                                  `json:"reused"`
	State     browserplatform.InteractiveLoginState `json:"state"`
}

func startSohuPhoneAuthForTest(t *testing.T, router *gin.Engine, accountID string) mediaAccountAuthStartTestResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/media-accounts/%s/auth/start", accountID), bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_personal")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start sohu phone auth status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response mediaAccountAuthStartTestResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal sohu phone auth start response: %v", err)
	}
	return response
}

func createWorkspaceKnowledgeBase(t *testing.T, router *gin.Engine, name string) model.KnowledgeBase {
	t.Helper()

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"name": %q,
		"description": "测试知识库包"
	}`, name))
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-bases", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create knowledge base status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var base model.KnowledgeBase
	if err := json.Unmarshal(rec.Body.Bytes(), &base); err != nil {
		t.Fatalf("unmarshal knowledge base response: %v", err)
	}
	return base
}

func sameStringSet(actual []string, expected []string) bool {
	if len(actual) != len(expected) {
		return false
	}
	counts := map[string]int{}
	for _, value := range actual {
		counts[value]++
	}
	for _, value := range expected {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

func hasKnowledgeTask(items []model.KnowledgeProcessingTask, taskType string, status string) bool {
	for _, item := range items {
		if item.TaskType == taskType && item.Status == status {
			return true
		}
	}
	return false
}

func hasKnowledgeTaskWithStatuses(items []model.KnowledgeProcessingTask, taskType string, statuses []string) bool {
	for _, status := range statuses {
		if hasKnowledgeTask(items, taskType, status) {
			return true
		}
	}
	return false
}

func waitForKnowledgeTaskStatus(t *testing.T, router *gin.Engine, assetID string, taskType string, status string) []model.KnowledgeProcessingTask {
	t.Helper()
	var items []model.KnowledgeProcessingTask
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		items = listKnowledgeAssetTasksForTest(t, router, assetID)
		if hasKnowledgeTask(items, taskType, status) {
			return items
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("task %s did not reach %s, last tasks: %#v", taskType, status, items)
	return nil
}

func listKnowledgeAssetTasksForTest(t *testing.T, router *gin.Engine, assetID string) []model.KnowledgeProcessingTask {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets/"+assetID+"/tasks", nil)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list tasks status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Items []model.KnowledgeProcessingTask `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal task response: %v", err)
	}
	return response.Items
}

func getKnowledgeAssetForTest(t *testing.T, router *gin.Engine, assetID string) model.KnowledgeAsset {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets/"+assetID, nil)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("get asset status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var asset model.KnowledgeAsset
	if err := json.Unmarshal(rec.Body.Bytes(), &asset); err != nil {
		t.Fatalf("unmarshal asset response: %v", err)
	}
	return asset
}

func listKnowledgeAssetChunksForTest(t *testing.T, router *gin.Engine, assetID string) []model.KnowledgeChunk {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/api/knowledge-assets/"+assetID+"/chunks", nil)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list chunks status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Items []model.KnowledgeChunk `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal chunks response: %v", err)
	}
	return response.Items
}

func knowledgeChunksContain(items []model.KnowledgeChunk, value string) bool {
	for _, item := range items {
		if strings.Contains(item.Content, value) || strings.Contains(item.SearchText, value) {
			return true
		}
	}
	return false
}

func knowledgeChunksSearchTextContain(items []model.KnowledgeChunk, value string) bool {
	for _, item := range items {
		if strings.Contains(item.SearchText, value) {
			return true
		}
	}
	return false
}

func testWorkspaceRouter() *gin.Engine {
	router, _ := testWorkspaceRouterWithHandler()
	return router
}

func testWorkspaceRouterWithHandler() (*gin.Engine, *WorkspaceHandler) {
	gin.SetMode(gin.TestMode)
	handler := NewWorkspaceHandler(nil, ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderMock}))
	handler.browserLogin = fakeBrowserLoginService{}
	router := testRouterForHandler(handler)
	return router, handler
}

func testRouterForHandler(handler *WorkspaceHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	handler.Register(apiGroup, middleware.AuthWithTokenResolver(handler.ResolveUserSession))
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

type fakeInteractiveLoginService struct {
	startCalls *int
}

func (fakeInteractiveLoginService) LoginURLValue() string {
	return "https://mp.sohu.com/mpfe/v4/login"
}

func (service fakeInteractiveLoginService) Start(_ context.Context, req browserplatform.InteractiveLoginStartRequest) (browserplatform.InteractiveLoginState, error) {
	if service.startCalls != nil {
		(*service.startCalls)++
	}
	return browserplatform.InteractiveLoginState{
		SessionID:      req.SessionID,
		Platform:       platformTypeSohu,
		LoginURL:       "https://mp.sohu.com/mpfe/v4/login",
		PageURL:        "https://mp.sohu.com/mpfe/v4/login",
		ProfileDir:     req.ProfileDir,
		StateFile:      req.StateFile,
		CommandFile:    req.CommandFile,
		Status:         "phone_required",
		Message:        "请输入搜狐号绑定手机号",
		AllowedActions: []string{"submit_phone"},
		Warnings:       []string{},
		StartedAt:      time.Now().UTC().Format(time.RFC3339),
		LastCheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (fakeInteractiveLoginService) State(_ context.Context, sessionID string, profileDir string, stateFile string, commandFile string) (browserplatform.InteractiveLoginState, bool, error) {
	return browserplatform.InteractiveLoginState{
		SessionID:      sessionID,
		Platform:       platformTypeSohu,
		LoginURL:       "https://mp.sohu.com/mpfe/v4/login",
		PageURL:        "https://mp.sohu.com/mpfe/v4/login",
		ProfileDir:     profileDir,
		StateFile:      stateFile,
		CommandFile:    commandFile,
		Status:         "phone_required",
		Message:        "请输入搜狐号绑定手机号",
		AllowedActions: []string{"submit_phone"},
		Warnings:       []string{},
		LastCheckedAt:  time.Now().UTC().Format(time.RFC3339),
	}, true, nil
}

func (fakeInteractiveLoginService) Action(_ context.Context, profileDir string, stateFile string, commandFile string, req browserplatform.InteractiveLoginActionRequest) (browserplatform.InteractiveLoginState, bool, error) {
	return browserplatform.InteractiveLoginState{
		SessionID:             req.SessionID,
		Platform:              platformTypeSohu,
		LoginURL:              "https://mp.sohu.com/mpfe/v4/login",
		PageURL:               "https://mp.sohu.com/mpfe/v4/login",
		ProfileDir:            profileDir,
		StateFile:             stateFile,
		CommandFile:           commandFile,
		Status:                "captcha_required",
		Message:               "请输入图形验证码",
		CaptchaScreenshotData: "data:image/png;base64,test",
		AllowedActions:        []string{"submit_captcha", "refresh_captcha"},
		Warnings:              []string{},
		LastCheckedAt:         time.Now().UTC().Format(time.RFC3339),
		LastCommandID:         req.Action,
	}, true, nil
}
