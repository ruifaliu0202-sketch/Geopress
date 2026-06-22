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
	"geopress/backend/internal/domain"
	"geopress/backend/internal/http/middleware"
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
	if len(response.Items) != 1 {
		t.Fatalf("media platform count = %d, want 1: %#v", len(response.Items), response.Items)
	}
	item := response.Items[0]
	if item.ID != "plt_xiaohongshu" || item.Name != "小红书" || item.Type != "xiaohongshu" {
		t.Fatalf("updated platform not found in list: %#v", response.Items)
	}
	if !item.Capabilities.HasCapability(domain.ConnectorCapabilityAuthorization) ||
		!item.Capabilities.HasCapability(domain.ConnectorCapabilityContentPublish) {
		t.Fatalf("list response should include xiaohongshu capability contract: %#v", item.Capabilities)
	}
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

func TestKnowledgeItemsCanBelongToMultipleBases(t *testing.T) {
	router := testWorkspaceRouter()
	base := createWorkspaceKnowledgeBase(t, router, "选题素材包")

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseIds": ["kb_brand", %q],
		"type": "case",
		"title": "客户增长案例",
		"content": "客户通过内容矩阵提升线索质量。"
	}`, base.ID))
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-items", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create item status = %d, want %d, body = %s", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var item model.KnowledgeItem
	if err := json.Unmarshal(rec.Body.Bytes(), &item); err != nil {
		t.Fatalf("unmarshal item response: %v", err)
	}
	if !sameStringSet(item.KnowledgeBaseIDs, []string{"kb_brand", base.ID}) {
		t.Fatalf("knowledgeBaseIds = %#v, want kb_brand and %s", item.KnowledgeBaseIDs, base.ID)
	}

	assignBody := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeItemIds": ["kbi_1001"],
		"knowledgeBaseIds": [%q]
	}`, base.ID))
	assignReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-items/assign-bases", assignBody)
	assignReq.Header.Set("Authorization", "Bearer demo-token")
	assignReq.Header.Set("X-Workspace-ID", "wks_acme")
	assignReq.Header.Set("Content-Type", "application/json")

	assignRec := httptest.NewRecorder()
	router.ServeHTTP(assignRec, assignReq)
	if assignRec.Code != http.StatusOK {
		t.Fatalf("assign status = %d, want %d, body = %s", assignRec.Code, http.StatusOK, assignRec.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/knowledge-items", nil)
	listReq.Header.Set("Authorization", "Bearer demo-token")
	listReq.Header.Set("X-Workspace-ID", "wks_acme")
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d, body = %s", listRec.Code, http.StatusOK, listRec.Body.String())
	}

	var listResponse struct {
		Items []model.KnowledgeItem `json:"items"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listResponse); err != nil {
		t.Fatalf("unmarshal list response: %v", err)
	}

	for _, listedItem := range listResponse.Items {
		if listedItem.ID == "kbi_1001" && sameStringSet(listedItem.KnowledgeBaseIDs, []string{"kb_brand", base.ID}) {
			return
		}
	}
	t.Fatalf("assigned item with both bases not found in list: %#v", listResponse.Items)
}

func TestFormatKnowledgeItemRequiresVIP(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"type": "brand",
		"title": "品牌提示词",
		"content": "面向增长团队，语气专业。"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-items/format", body)
	req.Header.Set("Authorization", "Bearer growth-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestFormatKnowledgeItemForVIP(t *testing.T) {
	router := testWorkspaceRouter()
	body := bytes.NewBufferString(`{
		"type": "brand",
		"title": "品牌提示词",
		"content": "面向增长团队，语气专业。不要承诺效果。"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-items/format", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Content  string `json:"content"`
		Provider string `json:"provider"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Provider != ai.ProviderMock {
		t.Fatalf("provider = %q, want %q", response.Provider, ai.ProviderMock)
	}
	if !bytes.Contains([]byte(response.Content), []byte("## 品牌提示词")) || !bytes.Contains([]byte(response.Content), []byte("### 使用边界")) {
		t.Fatalf("formatted content is not structured markdown: %s", response.Content)
	}
}

func TestFormatGenerationKeywordsFallsBackToMockWhenOpenAIUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	handler := NewWorkspaceHandler(nil, ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderOpenAI}))
	handler.Register(apiGroup, middleware.AuthWithTokenResolver(handler.ResolveUserSession))

	body := bytes.NewBufferString(`{
		"type": "generation_keywords",
		"title": "发布内容关键词",
		"content": "内容营销,增长,小红书长文"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-items/format", body)
	req.Header.Set("Authorization", "Bearer demo-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var response struct {
		Content       string `json:"content"`
		Provider      string `json:"provider"`
		Fallback      bool   `json:"fallback"`
		FallbackError string `json:"fallbackError"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if response.Provider != ai.ProviderMock {
		t.Fatalf("provider = %q, want %q", response.Provider, ai.ProviderMock)
	}
	if !response.Fallback || response.FallbackError == "" {
		t.Fatalf("fallback marker not returned: %#v", response)
	}
	if !bytes.Contains([]byte(response.Content), []byte("## 生成目标")) ||
		!bytes.Contains([]byte(response.Content), []byte("## 核心主题")) ||
		!bytes.Contains([]byte(response.Content), []byte("## 事实边界")) {
		t.Fatalf("formatted keywords are not a markdown prompt: %s", response.Content)
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
		"type": "brand",
		"title": "升级后格式化",
		"content": "增长用户升级 VIP 后可以格式化。"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/knowledge-items/format", body)
	req.Header.Set("Authorization", "Bearer growth-token")
	req.Header.Set("X-Workspace-ID", "wks_acme")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("format status = %d, want %d, body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGenerateContentAcceptsMultipleKnowledgeBases(t *testing.T) {
	router := testWorkspaceRouter()
	base := createWorkspaceKnowledgeBase(t, router, "生成联调素材包")

	createItemBody := bytes.NewBufferString(fmt.Sprintf(`{
		"knowledgeBaseIds": [%q],
		"type": "case",
		"title": "生成补充案例",
		"content": "多知识库包生成时应能检索到这个补充案例。"
	}`, base.ID))
	createItemReq := httptest.NewRequest(http.MethodPost, "/api/knowledge-items", createItemBody)
	createItemReq.Header.Set("Authorization", "Bearer demo-token")
	createItemReq.Header.Set("X-Workspace-ID", "wks_acme")
	createItemReq.Header.Set("Content-Type", "application/json")
	createItemRec := httptest.NewRecorder()
	router.ServeHTTP(createItemRec, createItemReq)
	if createItemRec.Code != http.StatusCreated {
		t.Fatalf("create item status = %d, want %d, body = %s", createItemRec.Code, http.StatusCreated, createItemRec.Body.String())
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

func testWorkspaceRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	apiGroup := router.Group("/api")
	handler := NewWorkspaceHandler(nil, ai.NewRuntimeConfig(ai.Config{Provider: ai.ProviderMock}))
	handler.browserLogin = fakeBrowserLoginService{}
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
