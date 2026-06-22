package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"geopress/backend/internal/ai"
	"geopress/backend/internal/database"
	"geopress/backend/internal/http/middleware"
	"geopress/backend/internal/model"

	"github.com/gin-gonic/gin"
)

type skillPackageVersionPayload struct {
	ID               string `json:"id"`
	Version          string `json:"version"`
	PromptContract   string `json:"promptContract"`
	OutputSchema     string `json:"outputSchema"`
	QualityRules     string `json:"qualityRules"`
	QARules          string `json:"qaRules"`
	PublishPrepRules string `json:"publishPrepRules"`
	ChangeNote       string `json:"changeNote"`
}

type skillPackageAssetPayload struct {
	ID       string            `json:"id"`
	Type     string            `json:"type"`
	Title    string            `json:"title"`
	Content  string            `json:"content"`
	Metadata map[string]string `json:"metadata"`
}

type skillPackageExamplePayload struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	Input          string            `json:"input"`
	ExpectedOutput string            `json:"expectedOutput"`
	Notes          string            `json:"notes"`
	Metadata       map[string]string `json:"metadata"`
}

type createSkillPackageRequest struct {
	Name                    string                       `json:"name"`
	Slug                    string                       `json:"slug"`
	Description             string                       `json:"description"`
	Category                string                       `json:"category"`
	TargetPlatform          string                       `json:"targetPlatform"`
	TargetIndustry          string                       `json:"targetIndustry"`
	SupportedContentFormats []string                     `json:"supportedContentFormats"`
	AuthorID                string                       `json:"authorId"`
	AuthorName              string                       `json:"authorName"`
	ListingStatus           model.SkillPackageStatus     `json:"listingStatus"`
	PriceCents              int                          `json:"priceCents"`
	Currency                string                       `json:"currency"`
	RevenueShareBps         int                          `json:"revenueShareBps"`
	Version                 skillPackageVersionPayload   `json:"version"`
	Assets                  []skillPackageAssetPayload   `json:"assets"`
	Examples                []skillPackageExamplePayload `json:"examples"`
}

type reviewSkillPackageVersionRequest struct {
	Decision string `json:"decision"`
	Comment  string `json:"comment"`
}

type installSkillPackageRequest struct {
	VersionID string `json:"versionId"`
	Seats     int    `json:"seats"`
}

type skillPackageListing struct {
	Package     model.SkillPackage               `json:"package"`
	Version     *model.SkillPackageVersion       `json:"version,omitempty"`
	Installed   bool                             `json:"installed"`
	Entitlement *model.WorkspaceSkillEntitlement `json:"entitlement,omitempty"`
}

type installedSkillPackage struct {
	Entitlement model.WorkspaceSkillEntitlement `json:"entitlement"`
	Package     *model.SkillPackage             `json:"package,omitempty"`
	Version     *model.SkillPackageVersion      `json:"version,omitempty"`
}

func (h *WorkspaceHandler) AdminListSkillPackages(c *gin.Context) {
	h.mu.RLock()
	items := append([]model.SkillPackage{}, h.skillPackages...)
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListSkillPackageVersions(c *gin.Context) {
	packageID := strings.TrimSpace(c.Param("packageId"))
	h.mu.RLock()
	items := make([]model.SkillPackageVersion, 0)
	for _, version := range h.skillPackageVersions {
		if packageID == "" || version.PackageID == packageID {
			items = append(items, version)
		}
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListSkillPackageReviews(c *gin.Context) {
	versionID := strings.TrimSpace(c.Query("versionId"))
	h.mu.RLock()
	items := make([]model.SkillPackageReview, 0)
	for _, review := range h.skillPackageReviews {
		if versionID == "" || review.VersionID == versionID {
			items = append(items, review)
		}
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListSkillPackageEntitlements(c *gin.Context) {
	workspaceID := strings.TrimSpace(c.Query("workspaceId"))
	packageID := strings.TrimSpace(c.Query("packageId"))
	h.mu.RLock()
	items := make([]model.WorkspaceSkillEntitlement, 0)
	for _, entitlement := range h.skillEntitlements {
		if workspaceID != "" && entitlement.WorkspaceID != workspaceID {
			continue
		}
		if packageID != "" && entitlement.PackageID != packageID {
			continue
		}
		items = append(items, entitlement)
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListSkillPackageUsage(c *gin.Context) {
	workspaceID := strings.TrimSpace(c.Query("workspaceId"))
	packageID := strings.TrimSpace(c.Query("packageId"))
	h.mu.RLock()
	items := make([]model.SkillPackageUsageMetric, 0)
	for _, metric := range h.skillUsageMetrics {
		if workspaceID != "" && metric.WorkspaceID != workspaceID {
			continue
		}
		if packageID != "" && metric.PackageID != packageID {
			continue
		}
		items = append(items, metric)
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminListSkillPackageRevenue(c *gin.Context) {
	packageID := strings.TrimSpace(c.Query("packageId"))
	versionID := strings.TrimSpace(c.Query("versionId"))
	h.mu.RLock()
	items := make([]model.SkillPackageRevenueMetric, 0)
	for _, metric := range h.skillRevenueMetrics {
		if packageID != "" && metric.PackageID != packageID {
			continue
		}
		if versionID != "" && metric.VersionID != versionID {
			continue
		}
		items = append(items, metric)
	}
	h.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) AdminCreateSkillPackage(c *gin.Context) {
	var req createSkillPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	pkg, version, assets, examples, ok := h.buildSkillPackageFromRequest(req, "")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, slug and version are required"})
		return
	}

	if err := h.saveSkillPackageBundle(c.Request.Context(), pkg, version, assets, examples); err != nil {
		log.Printf("skill package create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill package was not persisted"})
		return
	}

	h.mu.Lock()
	h.skillPackages = append([]model.SkillPackage{pkg}, h.skillPackages...)
	h.skillPackageVersions = append([]model.SkillPackageVersion{version}, h.skillPackageVersions...)
	h.skillPackageAssets = append(append([]model.SkillPackageAsset{}, assets...), h.skillPackageAssets...)
	h.skillPackageExamples = append(append([]model.SkillPackageExample{}, examples...), h.skillPackageExamples...)
	h.mu.Unlock()

	c.JSON(http.StatusCreated, gin.H{"package": pkg, "version": version, "assets": assets, "examples": examples})
}

func (h *WorkspaceHandler) AdminUpdateSkillPackage(c *gin.Context) {
	packageID := strings.TrimSpace(c.Param("packageId"))
	if packageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "packageId is required"})
		return
	}

	var req createSkillPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	h.mu.RLock()
	current, ok := h.skillPackageByIDLocked(packageID)
	h.mu.RUnlock()
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill package not found"})
		return
	}

	pkg, version, assets, examples, valid := h.buildSkillPackageFromRequest(req, packageID)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name, slug and version are required"})
		return
	}
	pkg.CreatedAt = current.CreatedAt

	if err := h.saveSkillPackageBundle(c.Request.Context(), pkg, version, assets, examples); err != nil {
		log.Printf("skill package update failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill package was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertSkillPackageLocked(pkg)
	h.upsertSkillPackageVersionLocked(version)
	h.upsertSkillPackageAssetsLocked(assets)
	h.upsertSkillPackageExamplesLocked(examples)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"package": pkg, "version": version, "assets": assets, "examples": examples})
}

func (h *WorkspaceHandler) AdminSubmitSkillPackageVersion(c *gin.Context) {
	h.updateSkillPackageVersionState(c, model.SkillPackageVersionSubmitted, model.SkillPackageStatusInReview, model.SkillPackageReviewSubmitted)
}

func (h *WorkspaceHandler) AdminReviewSkillPackageVersion(c *gin.Context) {
	var req reviewSkillPackageVersionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	decision := strings.ToLower(strings.TrimSpace(req.Decision))
	switch decision {
	case string(model.SkillPackageReviewApproved):
		h.updateSkillPackageVersionStateWithComment(c, model.SkillPackageVersionApproved, model.SkillPackageStatusApproved, model.SkillPackageReviewApproved, req.Comment)
	case string(model.SkillPackageReviewRejected):
		h.updateSkillPackageVersionStateWithComment(c, model.SkillPackageVersionRejected, model.SkillPackageStatusRejected, model.SkillPackageReviewRejected, req.Comment)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "decision must be approved or rejected"})
	}
}

func (h *WorkspaceHandler) AdminPublishSkillPackageVersion(c *gin.Context) {
	h.updateSkillPackageVersionState(c, model.SkillPackageVersionPublished, model.SkillPackageStatusPublished, model.SkillPackageReviewApproved)
}

func (h *WorkspaceHandler) ListSkillPackageMarketplace(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	h.mu.RLock()
	items := make([]skillPackageListing, 0)
	for _, pkg := range h.skillPackages {
		if pkg.ListingStatus != model.SkillPackageStatusPublished || strings.TrimSpace(pkg.PublishedVersionID) == "" {
			continue
		}
		version, _ := h.skillPackageVersionByIDLocked(pkg.PublishedVersionID)
		entitlement, installed := h.activeSkillEntitlementLocked(workspaceID, pkg.PublishedVersionID)
		row := skillPackageListing{Package: pkg, Installed: installed}
		if version.ID != "" {
			row.Version = &version
		}
		if installed {
			row.Entitlement = &entitlement
		}
		items = append(items, row)
	}
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) ListInstalledSkillPackages(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	items := make([]installedSkillPackage, 0)
	for _, entitlement := range h.skillEntitlements {
		if entitlement.WorkspaceID != workspaceID || !isActiveSkillEntitlement(entitlement, now) {
			continue
		}
		row := installedSkillPackage{Entitlement: entitlement}
		if pkg, ok := h.skillPackageByIDLocked(entitlement.PackageID); ok {
			row.Package = &pkg
		}
		if version, ok := h.skillPackageVersionByIDLocked(entitlement.VersionID); ok {
			row.Version = &version
		}
		items = append(items, row)
	}
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) InstallSkillPackage(c *gin.Context) {
	h.createWorkspaceSkillEntitlement(c, model.WorkspaceSkillEntitlementTrial, false)
}

func (h *WorkspaceHandler) PurchaseSkillPackage(c *gin.Context) {
	h.createWorkspaceSkillEntitlement(c, model.WorkspaceSkillEntitlementPurchase, true)
}

func (h *WorkspaceHandler) SubscribeSkillPackage(c *gin.Context) {
	h.createWorkspaceSkillEntitlement(c, model.WorkspaceSkillEntitlementSubscription, true)
}

func (h *WorkspaceHandler) ListWorkspaceSkillPackageUsage(c *gin.Context) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	packageID := strings.TrimSpace(c.Query("packageId"))

	h.mu.RLock()
	items := make([]model.SkillPackageUsageMetric, 0)
	for _, metric := range h.skillUsageMetrics {
		if metric.WorkspaceID != workspaceID {
			continue
		}
		if packageID != "" && metric.PackageID != packageID {
			continue
		}
		items = append(items, metric)
	}
	h.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *WorkspaceHandler) createWorkspaceSkillEntitlement(c *gin.Context, source model.WorkspaceSkillEntitlementSource, revenue bool) {
	workspaceID, ok := h.authorizedWorkspaceID(c)
	if !ok {
		return
	}
	packageID := strings.TrimSpace(c.Param("packageId"))
	if packageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "packageId is required"})
		return
	}

	var req installSkillPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	pkg, found := h.skillPackageByIDLocked(packageID)
	if !found || pkg.ListingStatus != model.SkillPackageStatusPublished {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "published skill package not found"})
		return
	}
	versionID := strings.TrimSpace(req.VersionID)
	if versionID == "" {
		versionID = pkg.PublishedVersionID
	}
	version, versionOK := h.skillPackageVersionByIDLocked(versionID)
	if !versionOK || version.PackageID != pkg.ID || version.Status != model.SkillPackageVersionPublished {
		h.mu.RUnlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "published skill package version not found"})
		return
	}
	if entitlement, installed := h.activeSkillEntitlementLocked(workspaceID, version.ID); installed {
		h.mu.RUnlock()
		c.JSON(http.StatusOK, entitlement)
		return
	}
	existingEntitlement, existingEntitlementOK := h.skillEntitlementLocked(workspaceID, version.ID)
	h.mu.RUnlock()

	periodStart, periodEnd := skillPackagePeriod(now, source)
	entitlementID := fmt.Sprintf("wse_%d", now.UnixNano())
	createdAt := now
	if existingEntitlementOK {
		entitlementID = existingEntitlement.ID
		createdAt = existingEntitlement.CreatedAt
	}
	entitlement := model.WorkspaceSkillEntitlement{
		ID:                     entitlementID,
		WorkspaceID:            workspaceID,
		PackageID:              pkg.ID,
		VersionID:              version.ID,
		Status:                 model.WorkspaceSkillEntitlementActive,
		Source:                 source,
		Seats:                  defaultPositive(req.Seats, 1),
		PriceCents:             pkg.PriceCents,
		Currency:               defaultString(strings.ToUpper(strings.TrimSpace(pkg.Currency)), "USD"),
		CurrentPeriod:          now.Format("2006-01"),
		CurrentPeriodStartedAt: periodStart,
		CurrentPeriodEndsAt:    periodEnd,
		InstalledAt:            now,
		CreatedAt:              createdAt,
		UpdatedAt:              now,
	}

	var revenueMetric *model.SkillPackageRevenueMetric
	if revenue {
		authorRevenue := pkg.PriceCents * pkg.RevenueShareBps / 10000
		metricType := model.SkillPackageRevenuePurchase
		if source == model.WorkspaceSkillEntitlementSubscription {
			metricType = model.SkillPackageRevenueSubscription
		}
		metric := model.SkillPackageRevenueMetric{
			ID:                 fmt.Sprintf("spr_%d", now.UnixNano()),
			PackageID:          pkg.ID,
			VersionID:          version.ID,
			WorkspaceID:        workspaceID,
			EntitlementID:      entitlement.ID,
			MetricType:         metricType,
			AmountCents:        pkg.PriceCents,
			Currency:           entitlement.Currency,
			AuthorRevenueCents: authorRevenue,
			PlatformFeeCents:   pkg.PriceCents - authorRevenue,
			BillingPeriod:      now.Format("2006-01"),
			CreatedAt:          now,
		}
		revenueMetric = &metric
	}

	if err := h.saveWorkspaceSkillEntitlement(c.Request.Context(), entitlement, revenueMetric); err != nil {
		log.Printf("skill entitlement create failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill package entitlement was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertWorkspaceSkillEntitlementLocked(entitlement)
	if revenueMetric != nil {
		h.skillRevenueMetrics = append([]model.SkillPackageRevenueMetric{*revenueMetric}, h.skillRevenueMetrics...)
	}
	h.mu.Unlock()

	c.JSON(http.StatusCreated, entitlement)
}

func (h *WorkspaceHandler) updateSkillPackageVersionState(c *gin.Context, versionStatus model.SkillPackageVersionStatus, packageStatus model.SkillPackageStatus, decision model.SkillPackageReviewDecision) {
	h.updateSkillPackageVersionStateWithComment(c, versionStatus, packageStatus, decision, "")
}

func (h *WorkspaceHandler) updateSkillPackageVersionStateWithComment(c *gin.Context, versionStatus model.SkillPackageVersionStatus, packageStatus model.SkillPackageStatus, decision model.SkillPackageReviewDecision, comment string) {
	packageID := strings.TrimSpace(c.Param("packageId"))
	versionID := strings.TrimSpace(c.Param("versionId"))
	if packageID == "" || versionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "packageId and versionId are required"})
		return
	}

	now := time.Now().UTC()
	h.mu.RLock()
	pkg, packageOK := h.skillPackageByIDLocked(packageID)
	version, versionOK := h.skillPackageVersionByIDLocked(versionID)
	h.mu.RUnlock()
	if !packageOK || !versionOK || version.PackageID != packageID {
		c.JSON(http.StatusNotFound, gin.H{"error": "skill package version not found"})
		return
	}

	// 技能包版本化是 AI 行为可审计的边界：审核/发布只改变当前版本状态，不重写历史生成记录。
	version.Status = versionStatus
	version.UpdatedAt = now
	switch versionStatus {
	case model.SkillPackageVersionSubmitted:
		version.SubmittedAt = &now
	case model.SkillPackageVersionApproved, model.SkillPackageVersionRejected:
		version.ReviewedAt = &now
	case model.SkillPackageVersionPublished:
		version.ReviewedAt = ensureTime(version.ReviewedAt, now)
		version.PublishedAt = &now
		pkg.PublishedVersionID = version.ID
	}

	pkg.ListingStatus = packageStatus
	pkg.LatestVersionID = defaultString(pkg.LatestVersionID, version.ID)
	pkg.UpdatedAt = now
	review := model.SkillPackageReview{
		ID:         fmt.Sprintf("sprv_%d", now.UnixNano()),
		PackageID:  pkg.ID,
		VersionID:  version.ID,
		ReviewerID: middleware.CurrentUserID(c),
		Decision:   decision,
		Comment:    strings.TrimSpace(comment),
		CreatedAt:  now,
	}

	if err := h.saveSkillPackageReviewState(c.Request.Context(), version, pkg, review); err != nil {
		log.Printf("skill package review state failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "skill package review state was not persisted"})
		return
	}

	h.mu.Lock()
	h.upsertSkillPackageLocked(pkg)
	h.upsertSkillPackageVersionLocked(version)
	h.skillPackageReviews = append([]model.SkillPackageReview{review}, h.skillPackageReviews...)
	h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"package": pkg, "version": version, "review": review})
}

func (h *WorkspaceHandler) buildSkillPackageFromRequest(req createSkillPackageRequest, packageID string) (model.SkillPackage, model.SkillPackageVersion, []model.SkillPackageAsset, []model.SkillPackageExample, bool) {
	now := time.Now().UTC()
	name := strings.TrimSpace(req.Name)
	slug := normalizeSkillPackageSlug(req.Slug, name)
	versionName := strings.TrimSpace(req.Version.Version)
	if name == "" || slug == "" || versionName == "" {
		return model.SkillPackage{}, model.SkillPackageVersion{}, nil, nil, false
	}
	if packageID == "" {
		packageID = fmt.Sprintf("skp_%d", now.UnixNano())
	}
	versionID := strings.TrimSpace(req.Version.ID)
	if versionID == "" {
		versionID = fmt.Sprintf("skv_%d", now.UnixNano())
	}

	versionStatus := model.SkillPackageVersionDraft
	listingStatus := req.ListingStatus
	if listingStatus == "" {
		listingStatus = model.SkillPackageStatusDraft
	}
	if listingStatus == model.SkillPackageStatusPublished {
		versionStatus = model.SkillPackageVersionPublished
	}

	pkg := model.SkillPackage{
		ID:                      packageID,
		Name:                    name,
		Slug:                    slug,
		Description:             strings.TrimSpace(req.Description),
		Category:                defaultString(strings.TrimSpace(req.Category), "general"),
		TargetPlatform:          strings.TrimSpace(req.TargetPlatform),
		TargetIndustry:          strings.TrimSpace(req.TargetIndustry),
		SupportedContentFormats: cleanKeywords(req.SupportedContentFormats),
		AuthorID:                strings.TrimSpace(req.AuthorID),
		AuthorName:              strings.TrimSpace(req.AuthorName),
		ListingStatus:           listingStatus,
		PriceCents:              req.PriceCents,
		Currency:                defaultString(strings.ToUpper(strings.TrimSpace(req.Currency)), "USD"),
		RevenueShareBps:         req.RevenueShareBps,
		LatestVersionID:         versionID,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if versionStatus == model.SkillPackageVersionPublished {
		pkg.PublishedVersionID = versionID
	}
	if pkg.AuthorName == "" {
		pkg.AuthorName = "Geopress"
	}

	version := model.SkillPackageVersion{
		ID:               versionID,
		PackageID:        packageID,
		Version:          versionName,
		Status:           versionStatus,
		PromptContract:   strings.TrimSpace(req.Version.PromptContract),
		OutputSchema:     defaultString(strings.TrimSpace(req.Version.OutputSchema), "{}"),
		QualityRules:     strings.TrimSpace(req.Version.QualityRules),
		QARules:          strings.TrimSpace(req.Version.QARules),
		PublishPrepRules: strings.TrimSpace(req.Version.PublishPrepRules),
		ChangeNote:       strings.TrimSpace(req.Version.ChangeNote),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if versionStatus == model.SkillPackageVersionPublished {
		version.PublishedAt = &now
	}

	assets := make([]model.SkillPackageAsset, 0, len(req.Assets))
	for index, assetReq := range req.Assets {
		assetID := strings.TrimSpace(assetReq.ID)
		if assetID == "" {
			assetID = fmt.Sprintf("ska_%d_%d", now.UnixNano(), index)
		}
		assetType := model.SkillPackageAssetType(strings.TrimSpace(assetReq.Type))
		if assetType == "" {
			assetType = model.SkillPackageAssetRule
		}
		assets = append(assets, model.SkillPackageAsset{
			ID:        assetID,
			PackageID: packageID,
			VersionID: versionID,
			Type:      assetType,
			Title:     strings.TrimSpace(assetReq.Title),
			Content:   strings.TrimSpace(assetReq.Content),
			Metadata:  defaultStringMap(assetReq.Metadata),
			CreatedAt: now,
		})
	}

	examples := make([]model.SkillPackageExample, 0, len(req.Examples))
	for index, exampleReq := range req.Examples {
		exampleID := strings.TrimSpace(exampleReq.ID)
		if exampleID == "" {
			exampleID = fmt.Sprintf("ske_%d_%d", now.UnixNano(), index)
		}
		examples = append(examples, model.SkillPackageExample{
			ID:             exampleID,
			PackageID:      packageID,
			VersionID:      versionID,
			Title:          strings.TrimSpace(exampleReq.Title),
			Input:          strings.TrimSpace(exampleReq.Input),
			ExpectedOutput: strings.TrimSpace(exampleReq.ExpectedOutput),
			Notes:          strings.TrimSpace(exampleReq.Notes),
			Metadata:       defaultStringMap(exampleReq.Metadata),
			CreatedAt:      now,
		})
	}

	return pkg, version, assets, examples, true
}

func (h *WorkspaceHandler) saveSkillPackageBundle(ctx context.Context, pkg model.SkillPackage, version model.SkillPackageVersion, assets []model.SkillPackageAsset, examples []model.SkillPackageExample) error {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return h.db.SaveSkillPackageBundle(dbCtx, pkg, version, assets, examples)
}

func (h *WorkspaceHandler) saveSkillPackageReviewState(ctx context.Context, version model.SkillPackageVersion, pkg model.SkillPackage, review model.SkillPackageReview) error {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return h.db.UpdateSkillPackageVersionReviewState(dbCtx, version, pkg, review)
}

func (h *WorkspaceHandler) saveWorkspaceSkillEntitlement(ctx context.Context, entitlement model.WorkspaceSkillEntitlement, revenue *model.SkillPackageRevenueMetric) error {
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return h.db.SaveWorkspaceSkillEntitlementWithRevenue(dbCtx, entitlement, revenue)
}

func (h *WorkspaceHandler) loadSkillPackageSnapshot(ctx context.Context) (database.SkillPackageSnapshot, error) {
	if h.db == nil || h.db.SQL() == nil {
		return database.SkillPackageSnapshot{
			Packages:      []model.SkillPackage{},
			Versions:      []model.SkillPackageVersion{},
			Assets:        []model.SkillPackageAsset{},
			Examples:      []model.SkillPackageExample{},
			Reviews:       []model.SkillPackageReview{},
			Entitlements:  []model.WorkspaceSkillEntitlement{},
			UsageMetrics:  []model.SkillPackageUsageMetric{},
			RevenueMetric: []model.SkillPackageRevenueMetric{},
		}, nil
	}
	dbCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return h.db.LoadSkillPackageSnapshot(dbCtx)
}

func (h *WorkspaceHandler) skillPackageByIDLocked(packageID string) (model.SkillPackage, bool) {
	for _, item := range h.skillPackages {
		if item.ID == packageID {
			return item, true
		}
	}
	return model.SkillPackage{}, false
}

func (h *WorkspaceHandler) skillPackageVersionByIDLocked(versionID string) (model.SkillPackageVersion, bool) {
	for _, item := range h.skillPackageVersions {
		if item.ID == versionID {
			return item, true
		}
	}
	return model.SkillPackageVersion{}, false
}

func (h *WorkspaceHandler) activeSkillEntitlementLocked(workspaceID string, versionID string) (model.WorkspaceSkillEntitlement, bool) {
	now := time.Now().UTC()
	for _, entitlement := range h.skillEntitlements {
		if entitlement.WorkspaceID == workspaceID && entitlement.VersionID == versionID && isActiveSkillEntitlement(entitlement, now) {
			return entitlement, true
		}
	}
	return model.WorkspaceSkillEntitlement{}, false
}

func (h *WorkspaceHandler) skillEntitlementLocked(workspaceID string, versionID string) (model.WorkspaceSkillEntitlement, bool) {
	for _, entitlement := range h.skillEntitlements {
		if entitlement.WorkspaceID == workspaceID && entitlement.VersionID == versionID {
			return entitlement, true
		}
	}
	return model.WorkspaceSkillEntitlement{}, false
}

func (h *WorkspaceHandler) skillPackageContextLocked(packageID string, versionID string) (ai.SkillPackageContext, model.SkillPackage, model.SkillPackageVersion, bool) {
	pkg, packageOK := h.skillPackageByIDLocked(packageID)
	version, versionOK := h.skillPackageVersionByIDLocked(versionID)
	if !packageOK || !versionOK || version.PackageID != pkg.ID {
		return ai.SkillPackageContext{}, model.SkillPackage{}, model.SkillPackageVersion{}, false
	}
	return ai.SkillPackageContext{
		PackageID:        pkg.ID,
		VersionID:        version.ID,
		Version:          version.Version,
		Name:             pkg.Name,
		Category:         pkg.Category,
		TargetPlatform:   pkg.TargetPlatform,
		TargetIndustry:   pkg.TargetIndustry,
		SupportedFormats: append([]string(nil), pkg.SupportedContentFormats...),
		PromptContract:   version.PromptContract,
		QualityRules:     version.QualityRules,
		QARules:          version.QARules,
		PublishPrepRules: version.PublishPrepRules,
	}, pkg, version, true
}

func (h *WorkspaceHandler) upsertSkillPackageLocked(pkg model.SkillPackage) {
	for index := range h.skillPackages {
		if h.skillPackages[index].ID == pkg.ID {
			h.skillPackages[index] = pkg
			return
		}
	}
	h.skillPackages = append([]model.SkillPackage{pkg}, h.skillPackages...)
}

func (h *WorkspaceHandler) upsertSkillPackageVersionLocked(version model.SkillPackageVersion) {
	for index := range h.skillPackageVersions {
		if h.skillPackageVersions[index].ID == version.ID {
			h.skillPackageVersions[index] = version
			return
		}
	}
	h.skillPackageVersions = append([]model.SkillPackageVersion{version}, h.skillPackageVersions...)
}

func (h *WorkspaceHandler) upsertSkillPackageAssetsLocked(items []model.SkillPackageAsset) {
	for _, item := range items {
		found := false
		for index := range h.skillPackageAssets {
			if h.skillPackageAssets[index].ID == item.ID {
				h.skillPackageAssets[index] = item
				found = true
				break
			}
		}
		if !found {
			h.skillPackageAssets = append([]model.SkillPackageAsset{item}, h.skillPackageAssets...)
		}
	}
}

func (h *WorkspaceHandler) upsertSkillPackageExamplesLocked(items []model.SkillPackageExample) {
	for _, item := range items {
		found := false
		for index := range h.skillPackageExamples {
			if h.skillPackageExamples[index].ID == item.ID {
				h.skillPackageExamples[index] = item
				found = true
				break
			}
		}
		if !found {
			h.skillPackageExamples = append([]model.SkillPackageExample{item}, h.skillPackageExamples...)
		}
	}
}

func (h *WorkspaceHandler) upsertWorkspaceSkillEntitlementLocked(item model.WorkspaceSkillEntitlement) {
	for index := range h.skillEntitlements {
		if h.skillEntitlements[index].WorkspaceID == item.WorkspaceID &&
			h.skillEntitlements[index].PackageID == item.PackageID &&
			h.skillEntitlements[index].VersionID == item.VersionID {
			h.skillEntitlements[index] = item
			return
		}
	}
	h.skillEntitlements = append([]model.WorkspaceSkillEntitlement{item}, h.skillEntitlements...)
}

func normalizeSkillPackageSlug(slug string, fallback string) string {
	value := strings.ToLower(strings.TrimSpace(slug))
	if value == "" {
		value = strings.ToLower(strings.TrimSpace(fallback))
	}
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func defaultStringMap(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	return values
}

func defaultPositive(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func ensureTime(value *time.Time, fallback time.Time) *time.Time {
	if value != nil {
		return value
	}
	return &fallback
}

func skillPackagePeriod(now time.Time, source model.WorkspaceSkillEntitlementSource) (*time.Time, *time.Time) {
	if source != model.WorkspaceSkillEntitlementSubscription {
		return nil, nil
	}
	start := now
	end := now.AddDate(0, 1, 0)
	return &start, &end
}

func isActiveSkillEntitlement(entitlement model.WorkspaceSkillEntitlement, now time.Time) bool {
	if entitlement.Status != model.WorkspaceSkillEntitlementActive {
		return false
	}
	if entitlement.ExpiresAt != nil && !entitlement.ExpiresAt.After(now) {
		return false
	}
	if entitlement.CurrentPeriodEndsAt != nil && !entitlement.CurrentPeriodEndsAt.After(now) {
		return false
	}
	return true
}
