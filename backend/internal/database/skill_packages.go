package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"geopress/backend/internal/model"
)

type SkillPackageSnapshot struct {
	Packages      []model.SkillPackage
	Versions      []model.SkillPackageVersion
	Assets        []model.SkillPackageAsset
	Examples      []model.SkillPackageExample
	Reviews       []model.SkillPackageReview
	Entitlements  []model.WorkspaceSkillEntitlement
	UsageMetrics  []model.SkillPackageUsageMetric
	RevenueMetric []model.SkillPackageRevenueMetric
}

func (db *DB) LoadSkillPackageSnapshot(ctx context.Context) (SkillPackageSnapshot, error) {
	if db == nil || db.conn == nil {
		return SkillPackageSnapshot{}, nil
	}

	var snapshot SkillPackageSnapshot
	var err error
	if snapshot.Packages, err = db.ListSkillPackages(ctx); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.Versions, err = db.ListSkillPackageVersions(ctx, ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.Assets, err = db.ListSkillPackageAssets(ctx, ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.Examples, err = db.ListSkillPackageExamples(ctx, ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.Reviews, err = db.ListSkillPackageReviews(ctx, ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.Entitlements, err = db.ListWorkspaceSkillEntitlements(ctx, ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.UsageMetrics, err = db.ListSkillPackageUsageMetrics(ctx, "", ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	if snapshot.RevenueMetric, err = db.ListSkillPackageRevenueMetrics(ctx, "", ""); err != nil {
		return SkillPackageSnapshot{}, err
	}
	return snapshot, nil
}

func (db *DB) ListSkillPackages(ctx context.Context) ([]model.SkillPackage, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackage{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, name, slug, description, category, target_platform, target_industry,
			to_json(supported_content_formats)::text,
			COALESCE(author_id, ''), author_name, listing_status, price_cents, currency,
			revenue_share_bps, latest_version_id, published_version_id, created_at, updated_at
		FROM skill_packages
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackage{}
	for rows.Next() {
		var item model.SkillPackage
		var formats string
		var status string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.Category,
			&item.TargetPlatform,
			&item.TargetIndustry,
			&formats,
			&item.AuthorID,
			&item.AuthorName,
			&status,
			&item.PriceCents,
			&item.Currency,
			&item.RevenueShareBps,
			&item.LatestVersionID,
			&item.PublishedVersionID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.SupportedContentFormats = decodeStringSlice(formats)
		item.ListingStatus = model.SkillPackageStatus(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListPublishedSkillPackages(ctx context.Context) ([]model.SkillPackage, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackage{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, name, slug, description, category, target_platform, target_industry,
			to_json(supported_content_formats)::text,
			COALESCE(author_id, ''), author_name, listing_status, price_cents, currency,
			revenue_share_bps, latest_version_id, published_version_id, created_at, updated_at
		FROM skill_packages
		WHERE listing_status = 'published'
		  AND published_version_id <> ''
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackage{}
	for rows.Next() {
		var item model.SkillPackage
		var formats string
		var status string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Slug,
			&item.Description,
			&item.Category,
			&item.TargetPlatform,
			&item.TargetIndustry,
			&formats,
			&item.AuthorID,
			&item.AuthorName,
			&status,
			&item.PriceCents,
			&item.Currency,
			&item.RevenueShareBps,
			&item.LatestVersionID,
			&item.PublishedVersionID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.SupportedContentFormats = decodeStringSlice(formats)
		item.ListingStatus = model.SkillPackageStatus(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SkillPackageByID(ctx context.Context, packageID string) (model.SkillPackage, bool, error) {
	if db == nil || db.conn == nil {
		return model.SkillPackage{}, false, nil
	}

	var item model.SkillPackage
	var formats string
	var status string
	err := db.conn.QueryRowContext(ctx, `
		SELECT
			id, name, slug, description, category, target_platform, target_industry,
			to_json(supported_content_formats)::text,
			COALESCE(author_id, ''), author_name, listing_status, price_cents, currency,
			revenue_share_bps, latest_version_id, published_version_id, created_at, updated_at
		FROM skill_packages
		WHERE id = $1
	`, packageID).Scan(
		&item.ID,
		&item.Name,
		&item.Slug,
		&item.Description,
		&item.Category,
		&item.TargetPlatform,
		&item.TargetIndustry,
		&formats,
		&item.AuthorID,
		&item.AuthorName,
		&status,
		&item.PriceCents,
		&item.Currency,
		&item.RevenueShareBps,
		&item.LatestVersionID,
		&item.PublishedVersionID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return model.SkillPackage{}, false, nil
	}
	if err != nil {
		return model.SkillPackage{}, false, err
	}
	item.SupportedContentFormats = decodeStringSlice(formats)
	item.ListingStatus = model.SkillPackageStatus(status)
	return item, true, nil
}

func (db *DB) SaveSkillPackage(ctx context.Context, item model.SkillPackage) error {
	if db == nil || db.conn == nil {
		return nil
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO skill_packages (
			id, name, slug, description, category, target_platform, target_industry,
			supported_content_formats, author_id, author_name, listing_status,
			price_cents, currency, revenue_share_bps, latest_version_id, published_version_id,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8::text[], nullif($9, ''), $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			target_platform = EXCLUDED.target_platform,
			target_industry = EXCLUDED.target_industry,
			supported_content_formats = EXCLUDED.supported_content_formats,
			author_id = EXCLUDED.author_id,
			author_name = EXCLUDED.author_name,
			listing_status = EXCLUDED.listing_status,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			revenue_share_bps = EXCLUDED.revenue_share_bps,
			latest_version_id = EXCLUDED.latest_version_id,
			published_version_id = EXCLUDED.published_version_id,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.Name,
		item.Slug,
		item.Description,
		item.Category,
		item.TargetPlatform,
		item.TargetIndustry,
		pgTextArray(item.SupportedContentFormats),
		item.AuthorID,
		item.AuthorName,
		defaultSkillPackageStatus(item.ListingStatus),
		item.PriceCents,
		defaultCurrency(item.Currency),
		item.RevenueShareBps,
		item.LatestVersionID,
		item.PublishedVersionID,
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) ListSkillPackageVersions(ctx context.Context, packageID string) ([]model.SkillPackageVersion, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageVersion{}, nil
	}
	query := `
		SELECT
			id, package_id, version, status, prompt_contract, output_schema::text,
			quality_rules, qa_rules, publish_prep_rules, change_note,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		FROM skill_package_versions
	`
	args := []any{}
	if packageID != "" {
		query += " WHERE package_id = $1"
		args = append(args, packageID)
	}
	query += " ORDER BY created_at DESC, id ASC"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageVersion{}
	for rows.Next() {
		var item model.SkillPackageVersion
		var status string
		var submittedAt sql.NullTime
		var reviewedAt sql.NullTime
		var publishedAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.PackageID,
			&item.Version,
			&status,
			&item.PromptContract,
			&item.OutputSchema,
			&item.QualityRules,
			&item.QARules,
			&item.PublishPrepRules,
			&item.ChangeNote,
			&submittedAt,
			&reviewedAt,
			&publishedAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Status = model.SkillPackageVersionStatus(status)
		item.SubmittedAt = nullableTime(submittedAt)
		item.ReviewedAt = nullableTime(reviewedAt)
		item.PublishedAt = nullableTime(publishedAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SkillPackageVersionByID(ctx context.Context, versionID string) (model.SkillPackageVersion, bool, error) {
	if db == nil || db.conn == nil {
		return model.SkillPackageVersion{}, false, nil
	}

	var item model.SkillPackageVersion
	var status string
	var submittedAt sql.NullTime
	var reviewedAt sql.NullTime
	var publishedAt sql.NullTime
	err := db.conn.QueryRowContext(ctx, `
		SELECT
			id, package_id, version, status, prompt_contract, output_schema::text,
			quality_rules, qa_rules, publish_prep_rules, change_note,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		FROM skill_package_versions
		WHERE id = $1
	`, versionID).Scan(
		&item.ID,
		&item.PackageID,
		&item.Version,
		&status,
		&item.PromptContract,
		&item.OutputSchema,
		&item.QualityRules,
		&item.QARules,
		&item.PublishPrepRules,
		&item.ChangeNote,
		&submittedAt,
		&reviewedAt,
		&publishedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return model.SkillPackageVersion{}, false, nil
	}
	if err != nil {
		return model.SkillPackageVersion{}, false, err
	}
	item.Status = model.SkillPackageVersionStatus(status)
	item.SubmittedAt = nullableTime(submittedAt)
	item.ReviewedAt = nullableTime(reviewedAt)
	item.PublishedAt = nullableTime(publishedAt)
	return item, true, nil
}

func (db *DB) SaveSkillPackageVersion(ctx context.Context, item model.SkillPackageVersion) error {
	if db == nil || db.conn == nil {
		return nil
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	outputSchema := jsonString(item.OutputSchema)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO skill_package_versions (
			id, package_id, version, status, prompt_contract, output_schema,
			quality_rules, qa_rules, publish_prep_rules, change_note,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			version = EXCLUDED.version,
			status = EXCLUDED.status,
			prompt_contract = EXCLUDED.prompt_contract,
			output_schema = EXCLUDED.output_schema,
			quality_rules = EXCLUDED.quality_rules,
			qa_rules = EXCLUDED.qa_rules,
			publish_prep_rules = EXCLUDED.publish_prep_rules,
			change_note = EXCLUDED.change_note,
			submitted_at = EXCLUDED.submitted_at,
			reviewed_at = EXCLUDED.reviewed_at,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.PackageID,
		item.Version,
		defaultSkillPackageVersionStatus(item.Status),
		item.PromptContract,
		outputSchema,
		item.QualityRules,
		item.QARules,
		item.PublishPrepRules,
		item.ChangeNote,
		item.SubmittedAt,
		item.ReviewedAt,
		item.PublishedAt,
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveSkillPackageWithVersion(ctx context.Context, item model.SkillPackage, version model.SkillPackageVersion) error {
	return db.SaveSkillPackageBundle(ctx, item, version, nil, nil)
}

func (db *DB) SaveSkillPackageBundle(ctx context.Context, item model.SkillPackage, version model.SkillPackageVersion, assets []model.SkillPackageAsset, examples []model.SkillPackageExample) error {
	if db == nil || db.conn == nil {
		return nil
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err = saveSkillPackageTx(ctx, tx, item); err != nil {
		return err
	}
	if err = saveSkillPackageVersionTx(ctx, tx, version); err != nil {
		return err
	}
	for _, asset := range assets {
		if err = saveSkillPackageAssetTx(ctx, tx, asset); err != nil {
			return err
		}
	}
	for _, example := range examples {
		if err = saveSkillPackageExampleTx(ctx, tx, example); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) UpdateSkillPackageVersionReviewState(ctx context.Context, version model.SkillPackageVersion, pkg model.SkillPackage, review model.SkillPackageReview) error {
	if db == nil || db.conn == nil {
		return nil
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err = saveSkillPackageVersionTx(ctx, tx, version); err != nil {
		return err
	}
	if err = saveSkillPackageTx(ctx, tx, pkg); err != nil {
		return err
	}
	if err = saveSkillPackageReviewTx(ctx, tx, review); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) SaveSkillPackageReview(ctx context.Context, item model.SkillPackageReview) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO skill_package_reviews (id, package_id, version_id, reviewer_id, decision, comment, created_at)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, $7)
	`, item.ID, item.PackageID, item.VersionID, item.ReviewerID, item.Decision, item.Comment, defaultTime(item.CreatedAt))
	return err
}

func (db *DB) ListSkillPackageReviews(ctx context.Context, versionID string) ([]model.SkillPackageReview, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageReview{}, nil
	}
	query := `
		SELECT id, package_id, version_id, COALESCE(reviewer_id, ''), decision, comment, created_at
		FROM skill_package_reviews
	`
	args := []any{}
	if versionID != "" {
		query += " WHERE version_id = $1"
		args = append(args, versionID)
	}
	query += " ORDER BY created_at DESC, id ASC"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageReview{}
	for rows.Next() {
		var item model.SkillPackageReview
		var decision string
		if err := rows.Scan(&item.ID, &item.PackageID, &item.VersionID, &item.ReviewerID, &decision, &item.Comment, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Decision = model.SkillPackageReviewDecision(decision)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListSkillPackageAssets(ctx context.Context, versionID string) ([]model.SkillPackageAsset, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageAsset{}, nil
	}
	query := `
		SELECT id, package_id, version_id, type, title, content, metadata::text, created_at
		FROM skill_package_assets
	`
	args := []any{}
	if versionID != "" {
		query += " WHERE version_id = $1"
		args = append(args, versionID)
	}
	query += " ORDER BY created_at DESC, id ASC"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageAsset{}
	for rows.Next() {
		var item model.SkillPackageAsset
		var assetType string
		var metadata string
		if err := rows.Scan(&item.ID, &item.PackageID, &item.VersionID, &assetType, &item.Title, &item.Content, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Type = model.SkillPackageAssetType(assetType)
		item.Metadata = decodeStringMap(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveSkillPackageAsset(ctx context.Context, item model.SkillPackageAsset) error {
	if db == nil || db.conn == nil {
		return nil
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO skill_package_assets (id, package_id, version_id, type, title, content, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata
	`, item.ID, item.PackageID, item.VersionID, item.Type, item.Title, item.Content, string(metadata), defaultTime(item.CreatedAt))
	return err
}

func (db *DB) SaveSkillPackageExample(ctx context.Context, item model.SkillPackageExample) error {
	if db == nil || db.conn == nil {
		return nil
	}
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO skill_package_examples (id, package_id, version_id, title, input, expected_output, notes, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			input = EXCLUDED.input,
			expected_output = EXCLUDED.expected_output,
			notes = EXCLUDED.notes,
			metadata = EXCLUDED.metadata
	`, item.ID,
		item.PackageID,
		item.VersionID,
		item.Title,
		item.Input,
		item.ExpectedOutput,
		item.Notes,
		string(metadata),
		defaultTime(item.CreatedAt),
	)
	return err
}

func (db *DB) ListSkillPackageExamples(ctx context.Context, versionID string) ([]model.SkillPackageExample, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageExample{}, nil
	}
	query := `
		SELECT id, package_id, version_id, title, input, expected_output, notes, metadata::text, created_at
		FROM skill_package_examples
	`
	args := []any{}
	if versionID != "" {
		query += " WHERE version_id = $1"
		args = append(args, versionID)
	}
	query += " ORDER BY created_at DESC, id ASC"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageExample{}
	for rows.Next() {
		var item model.SkillPackageExample
		var metadata string
		if err := rows.Scan(&item.ID, &item.PackageID, &item.VersionID, &item.Title, &item.Input, &item.ExpectedOutput, &item.Notes, &metadata, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Metadata = decodeStringMap(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) WorkspaceSkillEntitlementByVersion(ctx context.Context, workspaceID string, versionID string) (model.WorkspaceSkillEntitlement, bool, error) {
	if db == nil || db.conn == nil {
		return model.WorkspaceSkillEntitlement{}, false, nil
	}

	var item model.WorkspaceSkillEntitlement
	var status string
	var source string
	var periodStartedAt sql.NullTime
	var periodEndsAt sql.NullTime
	var expiresAt sql.NullTime
	err := db.conn.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, package_id, version_id, status, source, seats, price_cents, currency,
			current_period, current_period_started_at, current_period_ends_at,
			installed_at, expires_at, created_at, updated_at
		FROM workspace_skill_entitlements
		WHERE workspace_id = $1
		  AND version_id = $2
		  AND status = 'active'
		ORDER BY installed_at DESC
		LIMIT 1
	`, workspaceID, versionID).Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.PackageID,
		&item.VersionID,
		&status,
		&source,
		&item.Seats,
		&item.PriceCents,
		&item.Currency,
		&item.CurrentPeriod,
		&periodStartedAt,
		&periodEndsAt,
		&item.InstalledAt,
		&expiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return model.WorkspaceSkillEntitlement{}, false, nil
	}
	if err != nil {
		return model.WorkspaceSkillEntitlement{}, false, err
	}
	item.Status = model.WorkspaceSkillEntitlementStatus(status)
	item.Source = model.WorkspaceSkillEntitlementSource(source)
	item.CurrentPeriodStartedAt = nullableTime(periodStartedAt)
	item.CurrentPeriodEndsAt = nullableTime(periodEndsAt)
	item.ExpiresAt = nullableTime(expiresAt)
	return item, true, nil
}

func (db *DB) SaveWorkspaceSkillEntitlementWithRevenue(ctx context.Context, entitlement model.WorkspaceSkillEntitlement, revenue *model.SkillPackageRevenueMetric) error {
	if db == nil || db.conn == nil {
		return nil
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()
	if err = saveWorkspaceSkillEntitlementTx(ctx, tx, entitlement); err != nil {
		return err
	}
	if revenue != nil {
		if err = saveSkillPackageRevenueMetricTx(ctx, tx, *revenue); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) SaveWorkspaceSkillEntitlement(ctx context.Context, item model.WorkspaceSkillEntitlement) error {
	if db == nil || db.conn == nil {
		return nil
	}
	installedAt := defaultTime(item.InstalledAt)
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO workspace_skill_entitlements (
			id, workspace_id, package_id, version_id, status, source, seats, price_cents, currency,
			current_period, current_period_started_at, current_period_ends_at,
			installed_at, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (workspace_id, package_id, version_id) DO UPDATE SET
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			seats = EXCLUDED.seats,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			current_period = EXCLUDED.current_period,
			current_period_started_at = EXCLUDED.current_period_started_at,
			current_period_ends_at = EXCLUDED.current_period_ends_at,
			installed_at = EXCLUDED.installed_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.PackageID,
		item.VersionID,
		defaultWorkspaceSkillEntitlementStatus(item.Status),
		defaultWorkspaceSkillEntitlementSource(item.Source),
		defaultSeats(item.Seats),
		item.PriceCents,
		defaultCurrency(item.Currency),
		item.CurrentPeriod,
		item.CurrentPeriodStartedAt,
		item.CurrentPeriodEndsAt,
		installedAt,
		item.ExpiresAt,
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) ListWorkspaceSkillEntitlements(ctx context.Context, workspaceID string) ([]model.WorkspaceSkillEntitlement, error) {
	if db == nil || db.conn == nil {
		return []model.WorkspaceSkillEntitlement{}, nil
	}
	query := `
		SELECT
			id, workspace_id, package_id, version_id, status, source, seats, price_cents, currency,
			current_period, current_period_started_at, current_period_ends_at,
			installed_at, expires_at, created_at, updated_at
		FROM workspace_skill_entitlements
	`
	args := []any{}
	if workspaceID != "" {
		query += " WHERE workspace_id = $1"
		args = append(args, workspaceID)
	}
	query += " ORDER BY installed_at DESC, id ASC"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.WorkspaceSkillEntitlement{}
	for rows.Next() {
		var item model.WorkspaceSkillEntitlement
		var status string
		var source string
		var periodStartedAt sql.NullTime
		var periodEndsAt sql.NullTime
		var expiresAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.PackageID,
			&item.VersionID,
			&status,
			&source,
			&item.Seats,
			&item.PriceCents,
			&item.Currency,
			&item.CurrentPeriod,
			&periodStartedAt,
			&periodEndsAt,
			&item.InstalledAt,
			&expiresAt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Status = model.WorkspaceSkillEntitlementStatus(status)
		item.Source = model.WorkspaceSkillEntitlementSource(source)
		item.CurrentPeriodStartedAt = nullableTime(periodStartedAt)
		item.CurrentPeriodEndsAt = nullableTime(periodEndsAt)
		item.ExpiresAt = nullableTime(expiresAt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) RecordSkillPackageUsageMetric(ctx context.Context, item model.SkillPackageUsageMetric) error {
	if db == nil || db.conn == nil {
		return nil
	}
	return saveSkillPackageUsageMetricExec(ctx, db.conn, item)
}

func (db *DB) ListSkillPackageUsageMetrics(ctx context.Context, workspaceID string, packageID string) ([]model.SkillPackageUsageMetric, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageUsageMetric{}, nil
	}
	query := `
		SELECT id, workspace_id, package_id, version_id, generation_request_id, content_id, metric_type, count, status, created_at
		FROM skill_package_usage_metrics
	`
	args := []any{}
	if workspaceID != "" {
		args = append(args, workspaceID)
		query += " WHERE workspace_id = $1"
	}
	if packageID != "" {
		if len(args) == 0 {
			query += " WHERE package_id = $1"
		} else {
			query += " AND package_id = $2"
		}
		args = append(args, packageID)
	}
	query += " ORDER BY created_at DESC, id ASC LIMIT 1000"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageUsageMetric{}
	for rows.Next() {
		var item model.SkillPackageUsageMetric
		var metricType string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.PackageID,
			&item.VersionID,
			&item.GenerationRequestID,
			&item.ContentID,
			&metricType,
			&item.Count,
			&item.Status,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.MetricType = model.SkillPackageUsageMetricType(metricType)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) RecordSkillPackageRevenueMetric(ctx context.Context, item model.SkillPackageRevenueMetric) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO skill_package_revenue_metrics (
			id, package_id, version_id, workspace_id, entitlement_id, metric_type,
			amount_cents, currency, author_revenue_cents, platform_fee_cents, billing_period, created_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, ''), $6, $7, $8, $9, $10, $11, $12)
	`, item.ID,
		item.PackageID,
		item.VersionID,
		item.WorkspaceID,
		item.EntitlementID,
		item.MetricType,
		item.AmountCents,
		defaultCurrency(item.Currency),
		item.AuthorRevenueCents,
		item.PlatformFeeCents,
		item.BillingPeriod,
		defaultTime(item.CreatedAt),
	)
	return err
}

func (db *DB) ListSkillPackageRevenueMetrics(ctx context.Context, packageID string, versionID string) ([]model.SkillPackageRevenueMetric, error) {
	if db == nil || db.conn == nil {
		return []model.SkillPackageRevenueMetric{}, nil
	}
	query := `
		SELECT
			id, package_id, version_id, COALESCE(workspace_id, ''), COALESCE(entitlement_id, ''),
			metric_type, amount_cents, currency, author_revenue_cents, platform_fee_cents, billing_period, created_at
		FROM skill_package_revenue_metrics
	`
	args := []any{}
	if packageID != "" {
		args = append(args, packageID)
		query += " WHERE package_id = $1"
	}
	if versionID != "" {
		if len(args) == 0 {
			query += " WHERE version_id = $1"
		} else {
			query += " AND version_id = $2"
		}
		args = append(args, versionID)
	}
	query += " ORDER BY created_at DESC, id ASC LIMIT 1000"
	rows, err := db.conn.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SkillPackageRevenueMetric{}
	for rows.Next() {
		var item model.SkillPackageRevenueMetric
		var metricType string
		if err := rows.Scan(
			&item.ID,
			&item.PackageID,
			&item.VersionID,
			&item.WorkspaceID,
			&item.EntitlementID,
			&metricType,
			&item.AmountCents,
			&item.Currency,
			&item.AuthorRevenueCents,
			&item.PlatformFeeCents,
			&item.BillingPeriod,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.MetricType = model.SkillPackageRevenueMetricType(metricType)
		items = append(items, item)
	}
	return items, rows.Err()
}

func saveSkillPackageTx(ctx context.Context, tx *sql.Tx, item model.SkillPackage) error {
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO skill_packages (
			id, name, slug, description, category, target_platform, target_industry,
			supported_content_formats, author_id, author_name, listing_status,
			price_cents, currency, revenue_share_bps, latest_version_id, published_version_id,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			$8::text[], nullif($9, ''), $10, $11,
			$12, $13, $14, $15, $16,
			$17, $18
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			slug = EXCLUDED.slug,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			target_platform = EXCLUDED.target_platform,
			target_industry = EXCLUDED.target_industry,
			supported_content_formats = EXCLUDED.supported_content_formats,
			author_id = EXCLUDED.author_id,
			author_name = EXCLUDED.author_name,
			listing_status = EXCLUDED.listing_status,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			revenue_share_bps = EXCLUDED.revenue_share_bps,
			latest_version_id = EXCLUDED.latest_version_id,
			published_version_id = EXCLUDED.published_version_id,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.Name,
		item.Slug,
		item.Description,
		item.Category,
		item.TargetPlatform,
		item.TargetIndustry,
		pgTextArray(item.SupportedContentFormats),
		item.AuthorID,
		item.AuthorName,
		defaultSkillPackageStatus(item.ListingStatus),
		item.PriceCents,
		defaultCurrency(item.Currency),
		item.RevenueShareBps,
		item.LatestVersionID,
		item.PublishedVersionID,
		createdAt,
		updatedAt,
	)
	return err
}

func saveSkillPackageVersionTx(ctx context.Context, tx *sql.Tx, item model.SkillPackageVersion) error {
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	outputSchema := jsonString(item.OutputSchema)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO skill_package_versions (
			id, package_id, version, status, prompt_contract, output_schema,
			quality_rules, qa_rules, publish_prep_rules, change_note,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO UPDATE SET
			version = EXCLUDED.version,
			status = EXCLUDED.status,
			prompt_contract = EXCLUDED.prompt_contract,
			output_schema = EXCLUDED.output_schema,
			quality_rules = EXCLUDED.quality_rules,
			qa_rules = EXCLUDED.qa_rules,
			publish_prep_rules = EXCLUDED.publish_prep_rules,
			change_note = EXCLUDED.change_note,
			submitted_at = EXCLUDED.submitted_at,
			reviewed_at = EXCLUDED.reviewed_at,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.PackageID,
		item.Version,
		defaultSkillPackageVersionStatus(item.Status),
		item.PromptContract,
		outputSchema,
		item.QualityRules,
		item.QARules,
		item.PublishPrepRules,
		item.ChangeNote,
		item.SubmittedAt,
		item.ReviewedAt,
		item.PublishedAt,
		createdAt,
		updatedAt,
	)
	return err
}

func saveSkillPackageReviewTx(ctx context.Context, tx *sql.Tx, item model.SkillPackageReview) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO skill_package_reviews (id, package_id, version_id, reviewer_id, decision, comment, created_at)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, $7)
	`, item.ID, item.PackageID, item.VersionID, item.ReviewerID, item.Decision, item.Comment, defaultTime(item.CreatedAt))
	return err
}

func saveSkillPackageAssetTx(ctx context.Context, tx *sql.Tx, item model.SkillPackageAsset) error {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO skill_package_assets (id, package_id, version_id, type, title, content, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			metadata = EXCLUDED.metadata
	`, item.ID, item.PackageID, item.VersionID, item.Type, item.Title, item.Content, string(metadata), defaultTime(item.CreatedAt))
	return err
}

func saveSkillPackageExampleTx(ctx context.Context, tx *sql.Tx, item model.SkillPackageExample) error {
	metadata, err := json.Marshal(item.Metadata)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO skill_package_examples (id, package_id, version_id, title, input, expected_output, notes, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			input = EXCLUDED.input,
			expected_output = EXCLUDED.expected_output,
			notes = EXCLUDED.notes,
			metadata = EXCLUDED.metadata
	`, item.ID,
		item.PackageID,
		item.VersionID,
		item.Title,
		item.Input,
		item.ExpectedOutput,
		item.Notes,
		string(metadata),
		defaultTime(item.CreatedAt),
	)
	return err
}

func saveWorkspaceSkillEntitlementTx(ctx context.Context, tx *sql.Tx, item model.WorkspaceSkillEntitlement) error {
	installedAt := defaultTime(item.InstalledAt)
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO workspace_skill_entitlements (
			id, workspace_id, package_id, version_id, status, source, seats, price_cents, currency,
			current_period, current_period_started_at, current_period_ends_at,
			installed_at, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (workspace_id, package_id, version_id) DO UPDATE SET
			status = EXCLUDED.status,
			source = EXCLUDED.source,
			seats = EXCLUDED.seats,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			current_period = EXCLUDED.current_period,
			current_period_started_at = EXCLUDED.current_period_started_at,
			current_period_ends_at = EXCLUDED.current_period_ends_at,
			installed_at = EXCLUDED.installed_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.PackageID,
		item.VersionID,
		defaultWorkspaceSkillEntitlementStatus(item.Status),
		defaultWorkspaceSkillEntitlementSource(item.Source),
		defaultSeats(item.Seats),
		item.PriceCents,
		defaultCurrency(item.Currency),
		item.CurrentPeriod,
		item.CurrentPeriodStartedAt,
		item.CurrentPeriodEndsAt,
		installedAt,
		item.ExpiresAt,
		createdAt,
		updatedAt,
	)
	return err
}

func saveSkillPackageUsageMetricExec(ctx context.Context, exec interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}, item model.SkillPackageUsageMetric) error {
	_, err := exec.ExecContext(ctx, `
		INSERT INTO skill_package_usage_metrics (
			id, workspace_id, package_id, version_id, generation_request_id, content_id,
			metric_type, count, status, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, item.ID,
		item.WorkspaceID,
		item.PackageID,
		item.VersionID,
		item.GenerationRequestID,
		item.ContentID,
		item.MetricType,
		defaultMetricCount(item.Count),
		item.Status,
		defaultTime(item.CreatedAt),
	)
	return err
}

func saveSkillPackageRevenueMetricTx(ctx context.Context, tx *sql.Tx, item model.SkillPackageRevenueMetric) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO skill_package_revenue_metrics (
			id, package_id, version_id, workspace_id, entitlement_id, metric_type,
			amount_cents, currency, author_revenue_cents, platform_fee_cents, billing_period, created_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, ''), $6, $7, $8, $9, $10, $11, $12)
	`, item.ID,
		item.PackageID,
		item.VersionID,
		item.WorkspaceID,
		item.EntitlementID,
		item.MetricType,
		item.AmountCents,
		defaultCurrency(item.Currency),
		item.AuthorRevenueCents,
		item.PlatformFeeCents,
		item.BillingPeriod,
		defaultTime(item.CreatedAt),
	)
	return err
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	utc := value.Time.UTC()
	return &utc
}

func defaultSkillPackageStatus(value model.SkillPackageStatus) string {
	if value == "" {
		return string(model.SkillPackageStatusDraft)
	}
	return string(value)
}

func defaultSkillPackageVersionStatus(value model.SkillPackageVersionStatus) string {
	if value == "" {
		return string(model.SkillPackageVersionDraft)
	}
	return string(value)
}

func defaultWorkspaceSkillEntitlementStatus(value model.WorkspaceSkillEntitlementStatus) string {
	if value == "" {
		return string(model.WorkspaceSkillEntitlementActive)
	}
	return string(value)
}

func defaultWorkspaceSkillEntitlementSource(value model.WorkspaceSkillEntitlementSource) string {
	if value == "" {
		return string(model.WorkspaceSkillEntitlementPurchase)
	}
	return string(value)
}

func defaultCurrency(value string) string {
	if value == "" {
		return "USD"
	}
	return value
}

func defaultSeats(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func defaultMetricCount(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}
