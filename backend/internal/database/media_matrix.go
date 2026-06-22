package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"geopress/backend/internal/model"
)

func (db *DB) ListMediaAccountMatrix(ctx context.Context, workspaceID string) ([]model.MediaAccountMatrixItem, error) {
	if db == nil || db.conn == nil {
		return []model.MediaAccountMatrixItem{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			a.id,
			a.workspace_id,
			a.platform_id,
			a.name,
			a.external_id,
			a.status,
			a.credentials::text,
			a.expires_at,
			COALESCE(a.last_checked_at, a.updated_at),
			a.account_group,
			a.ownership_type,
			a.operating_role,
			a.persona,
			a.positioning,
			a.target_audience,
			to_json(a.content_categories)::text,
			a.health_status,
			a.health_notes,
			to_json(a.authorization_scopes)::text,
			a.sync_enabled,
			a.last_sync_job_id,
			a.last_sync_status,
			a.last_sync_message,
			a.last_profile_synced_at,
			a.last_metrics_synced_at,
			a.next_sync_at,
			a.matrix_metadata::text,
			p.id,
			p.name,
			p.type,
			p.enabled,
			p.supports_article,
			p.supports_image,
			p.supports_scheduling,
			p.credential_fields::text,
			COALESCE(s.id, ''),
			COALESCE(s.source, ''),
			s.captured_at,
			COALESCE(s.follower_count, 0),
			COALESCE(s.following_count, 0),
			COALESCE(s.content_count, 0),
			COALESCE(s.total_like_count, 0),
			COALESCE(s.total_favorite_count, 0),
			COALESCE(s.total_comment_count, 0),
			COALESCE(s.total_share_count, 0),
			COALESCE(s.engagement_rate, 0)::float8,
			COALESCE(s.audience_signals, '{}'::jsonb)::text,
			COALESCE(s.profile, '{}'::jsonb)::text,
			COALESCE(s.raw_metrics, '{}'::jsonb)::text,
			COALESCE(s.freshness_status, ''),
			s.created_at,
			COALESCE(j.id, ''),
			COALESCE(j.requested_by_user_id, ''),
			COALESCE(j.sync_type, ''),
			COALESCE(j.status, ''),
			j.requested_at,
			j.started_at,
			j.finished_at,
			COALESCE(j.idempotency_key, ''),
			COALESCE(j.request_payload, '{}'::jsonb)::text,
			COALESCE(j.result_summary, '{}'::jsonb)::text,
			COALESCE(j.error_message, ''),
			j.created_at,
			j.updated_at,
			(
				SELECT count(*)
				FROM content_metrics cm
				WHERE cm.workspace_id = a.workspace_id
				  AND cm.media_account_id = a.id
			) AS content_metric_count
		FROM media_accounts a
		JOIN media_platforms p ON p.id = a.platform_id
		LEFT JOIN LATERAL (
			SELECT *
			FROM media_account_metric_snapshots s
			WHERE s.workspace_id = a.workspace_id
			  AND s.media_account_id = a.id
			ORDER BY s.captured_at DESC, s.created_at DESC, s.id DESC
			LIMIT 1
		) s ON TRUE
		LEFT JOIN LATERAL (
			SELECT *
			FROM media_account_sync_jobs j
			WHERE j.workspace_id = a.workspace_id
			  AND j.media_account_id = a.id
			ORDER BY j.requested_at DESC, j.created_at DESC, j.id DESC
			LIMIT 1
		) j ON TRUE
		WHERE a.workspace_id = $1
		ORDER BY a.account_group ASC, a.updated_at DESC, a.id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.MediaAccountMatrixItem{}
	for rows.Next() {
		item, err := scanMediaAccountMatrixItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) GetMediaAccountMatrixItem(ctx context.Context, workspaceID string, accountID string) (model.MediaAccountMatrixItem, bool, error) {
	items, err := db.ListMediaAccountMatrix(ctx, workspaceID)
	if err != nil {
		return model.MediaAccountMatrixItem{}, false, err
	}
	for _, item := range items {
		if item.Account.ID == accountID {
			return item, true, nil
		}
	}
	return model.MediaAccountMatrixItem{}, false, nil
}

func (db *DB) ListMediaAccountMetricSnapshots(ctx context.Context, workspaceID string, accountID string, limit int) ([]model.MediaAccountMetricSnapshot, error) {
	if db == nil || db.conn == nil {
		return []model.MediaAccountMetricSnapshot{}, nil
	}
	if limit <= 0 || limit > 365 {
		limit = 90
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			media_account_id,
			platform_id,
			source,
			captured_at,
			follower_count,
			following_count,
			content_count,
			total_like_count,
			total_favorite_count,
			total_comment_count,
			total_share_count,
			engagement_rate::float8,
			audience_signals::text,
			profile::text,
			raw_metrics::text,
			freshness_status,
			created_at
		FROM media_account_metric_snapshots
		WHERE workspace_id = $1
		  AND media_account_id = $2
		ORDER BY captured_at DESC, created_at DESC, id DESC
		LIMIT $3
	`, workspaceID, accountID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.MediaAccountMetricSnapshot{}
	for rows.Next() {
		var item model.MediaAccountMetricSnapshot
		var audienceSignals, profile, rawMetrics string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.MediaAccountID,
			&item.PlatformID,
			&item.Source,
			&item.CapturedAt,
			&item.FollowerCount,
			&item.FollowingCount,
			&item.ContentCount,
			&item.TotalLikeCount,
			&item.TotalFavoriteCount,
			&item.TotalCommentCount,
			&item.TotalShareCount,
			&item.EngagementRate,
			&audienceSignals,
			&profile,
			&rawMetrics,
			&item.FreshnessStatus,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.AudienceSignals = decodeAnyMap(audienceSignals)
		item.Profile = decodeAnyMap(profile)
		item.RawMetrics = decodeAnyMap(rawMetrics)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListContentMetrics(ctx context.Context, workspaceID string, accountID string, contentID string, limit int) ([]model.ContentMetric, error) {
	if db == nil || db.conn == nil {
		return []model.ContentMetric{}, nil
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			content_id,
			COALESCE(publish_job_id, ''),
			media_account_id,
			platform_id,
			external_content_id,
			external_url,
			metric_date::text,
			captured_at,
			impression_count,
			view_count,
			like_count,
			comment_count,
			share_count,
			favorite_count,
			click_count,
			engagement_rate::float8,
			attribution_metadata::text,
			raw_metrics::text,
			created_at
		FROM content_metrics
		WHERE workspace_id = $1
		  AND ($2 = '' OR media_account_id = $2)
		  AND ($3 = '' OR content_id = $3)
		ORDER BY metric_date DESC, captured_at DESC, id DESC
		LIMIT $4
	`, workspaceID, accountID, contentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ContentMetric{}
	for rows.Next() {
		var item model.ContentMetric
		var attributionMetadata, rawMetrics string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.ContentID,
			&item.PublishJobID,
			&item.MediaAccountID,
			&item.PlatformID,
			&item.ExternalContentID,
			&item.ExternalURL,
			&item.MetricDate,
			&item.CapturedAt,
			&item.ImpressionCount,
			&item.ViewCount,
			&item.LikeCount,
			&item.CommentCount,
			&item.ShareCount,
			&item.FavoriteCount,
			&item.ClickCount,
			&item.EngagementRate,
			&attributionMetadata,
			&rawMetrics,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.AttributionMetadata = decodeAnyMap(attributionMetadata)
		item.RawMetrics = decodeAnyMap(rawMetrics)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) CreateMediaAccountSyncJob(ctx context.Context, item model.MediaAccountSyncJob) (model.MediaAccountSyncJob, error) {
	if db == nil || db.conn == nil {
		return item, nil
	}

	requestPayload, err := json.Marshal(nonNilAnyMap(item.RequestPayload))
	if err != nil {
		return model.MediaAccountSyncJob{}, err
	}
	resultSummary, err := json.Marshal(nonNilAnyMap(item.ResultSummary))
	if err != nil {
		return model.MediaAccountSyncJob{}, err
	}

	requestedAt := item.RequestedAt
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = requestedAt
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = requestedAt
	}

	// 同步请求可能被前端重复点击或网络重试；幂等键保证同一账号的同一批次只进入队列一次。
	row := db.conn.QueryRowContext(ctx, `
		INSERT INTO media_account_sync_jobs (
			id,
			workspace_id,
			media_account_id,
			platform_id,
			requested_by_user_id,
			sync_type,
			status,
			requested_at,
			started_at,
			finished_at,
			idempotency_key,
			request_payload,
			result_summary,
			error_message,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, nullif($5, ''), $6, $7, $8, $9, $10, $11, $12::jsonb, $13::jsonb, $14, $15, $16)
		ON CONFLICT (workspace_id, media_account_id, idempotency_key) DO UPDATE SET
			updated_at = media_account_sync_jobs.updated_at
		RETURNING
			id,
			workspace_id,
			media_account_id,
			platform_id,
			COALESCE(requested_by_user_id, ''),
			sync_type,
			status,
			requested_at,
			started_at,
			finished_at,
			idempotency_key,
			request_payload::text,
			result_summary::text,
			error_message,
			created_at,
			updated_at
	`, item.ID,
		item.WorkspaceID,
		item.MediaAccountID,
		item.PlatformID,
		item.RequestedByUserID,
		defaultString(item.SyncType, "metrics"),
		defaultString(item.Status, "queued"),
		requestedAt,
		item.StartedAt,
		item.FinishedAt,
		item.IdempotencyKey,
		string(requestPayload),
		string(resultSummary),
		item.ErrorMessage,
		createdAt,
		updatedAt,
	)

	var saved model.MediaAccountSyncJob
	var requestPayloadText, resultSummaryText string
	if err := row.Scan(
		&saved.ID,
		&saved.WorkspaceID,
		&saved.MediaAccountID,
		&saved.PlatformID,
		&saved.RequestedByUserID,
		&saved.SyncType,
		&saved.Status,
		&saved.RequestedAt,
		&saved.StartedAt,
		&saved.FinishedAt,
		&saved.IdempotencyKey,
		&requestPayloadText,
		&resultSummaryText,
		&saved.ErrorMessage,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	); err != nil {
		return model.MediaAccountSyncJob{}, err
	}
	saved.RequestPayload = decodeAnyMap(requestPayloadText)
	saved.ResultSummary = decodeAnyMap(resultSummaryText)

	if err := db.updateMediaAccountSyncState(ctx, saved); err != nil {
		return model.MediaAccountSyncJob{}, err
	}
	return saved, nil
}

func (db *DB) SaveMediaAccountMetricSnapshot(ctx context.Context, item model.MediaAccountMetricSnapshot) error {
	if db == nil || db.conn == nil {
		return nil
	}

	audienceSignals, err := json.Marshal(nonNilAnyMap(item.AudienceSignals))
	if err != nil {
		return err
	}
	profile, err := json.Marshal(nonNilAnyMap(item.Profile))
	if err != nil {
		return err
	}
	rawMetrics, err := json.Marshal(nonNilAnyMap(item.RawMetrics))
	if err != nil {
		return err
	}
	capturedAt := item.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = capturedAt
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO media_account_metric_snapshots (
			id,
			workspace_id,
			media_account_id,
			platform_id,
			source,
			captured_at,
			follower_count,
			following_count,
			content_count,
			total_like_count,
			total_favorite_count,
			total_comment_count,
			total_share_count,
			engagement_rate,
			audience_signals,
			profile,
			raw_metrics,
			freshness_status,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, $16::jsonb, $17::jsonb, $18, $19)
		ON CONFLICT (workspace_id, media_account_id, captured_at, source) DO UPDATE SET
			follower_count = EXCLUDED.follower_count,
			following_count = EXCLUDED.following_count,
			content_count = EXCLUDED.content_count,
			total_like_count = EXCLUDED.total_like_count,
			total_favorite_count = EXCLUDED.total_favorite_count,
			total_comment_count = EXCLUDED.total_comment_count,
			total_share_count = EXCLUDED.total_share_count,
			engagement_rate = EXCLUDED.engagement_rate,
			audience_signals = EXCLUDED.audience_signals,
			profile = EXCLUDED.profile,
			raw_metrics = EXCLUDED.raw_metrics,
			freshness_status = EXCLUDED.freshness_status
	`, item.ID,
		item.WorkspaceID,
		item.MediaAccountID,
		item.PlatformID,
		defaultString(item.Source, "manual"),
		capturedAt,
		item.FollowerCount,
		item.FollowingCount,
		item.ContentCount,
		item.TotalLikeCount,
		item.TotalFavoriteCount,
		item.TotalCommentCount,
		item.TotalShareCount,
		item.EngagementRate,
		string(audienceSignals),
		string(profile),
		string(rawMetrics),
		defaultString(item.FreshnessStatus, "fresh"),
		createdAt,
	)
	return err
}

func (db *DB) SaveContentMetric(ctx context.Context, item model.ContentMetric) error {
	if db == nil || db.conn == nil {
		return nil
	}

	attributionMetadata, err := json.Marshal(nonNilAnyMap(item.AttributionMetadata))
	if err != nil {
		return err
	}
	rawMetrics, err := json.Marshal(nonNilAnyMap(item.RawMetrics))
	if err != nil {
		return err
	}
	capturedAt := item.CapturedAt
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
	}
	metricDate := item.MetricDate
	if metricDate == "" {
		metricDate = capturedAt.Format("2006-01-02")
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = capturedAt
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO content_metrics (
			id,
			workspace_id,
			content_id,
			publish_job_id,
			media_account_id,
			platform_id,
			external_content_id,
			external_url,
			metric_date,
			captured_at,
			impression_count,
			view_count,
			like_count,
			comment_count,
			share_count,
			favorite_count,
			click_count,
			engagement_rate,
			attribution_metadata,
			raw_metrics,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, $7, $8, $9::date, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19::jsonb, $20::jsonb, $21, $22)
		ON CONFLICT (id) DO UPDATE SET
			publish_job_id = EXCLUDED.publish_job_id,
			external_content_id = EXCLUDED.external_content_id,
			external_url = EXCLUDED.external_url,
			metric_date = EXCLUDED.metric_date,
			captured_at = EXCLUDED.captured_at,
			impression_count = EXCLUDED.impression_count,
			view_count = EXCLUDED.view_count,
			like_count = EXCLUDED.like_count,
			comment_count = EXCLUDED.comment_count,
			share_count = EXCLUDED.share_count,
			favorite_count = EXCLUDED.favorite_count,
			click_count = EXCLUDED.click_count,
			engagement_rate = EXCLUDED.engagement_rate,
			attribution_metadata = EXCLUDED.attribution_metadata,
			raw_metrics = EXCLUDED.raw_metrics,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.ContentID,
		item.PublishJobID,
		item.MediaAccountID,
		item.PlatformID,
		item.ExternalContentID,
		item.ExternalURL,
		metricDate,
		capturedAt,
		item.ImpressionCount,
		item.ViewCount,
		item.LikeCount,
		item.CommentCount,
		item.ShareCount,
		item.FavoriteCount,
		item.ClickCount,
		item.EngagementRate,
		string(attributionMetadata),
		string(rawMetrics),
		createdAt,
		time.Now().UTC(),
	)
	return err
}

func (db *DB) updateMediaAccountSyncState(ctx context.Context, item model.MediaAccountSyncJob) error {
	message := item.ErrorMessage
	if message == "" {
		message = fmt.Sprintf("%s sync queued", defaultString(item.SyncType, "metrics"))
	}
	_, err := db.conn.ExecContext(ctx, `
		UPDATE media_accounts
		SET
			last_sync_job_id = $3,
			last_sync_status = $4,
			last_sync_message = $5,
			updated_at = $6
		WHERE workspace_id = $1
		  AND id = $2
	`, item.WorkspaceID,
		item.MediaAccountID,
		item.ID,
		item.Status,
		message,
		time.Now().UTC(),
	)
	return err
}

func scanMediaAccountMatrixItem(rows interface {
	Scan(dest ...any) error
}) (model.MediaAccountMatrixItem, error) {
	var item model.MediaAccountMatrixItem
	var credentials, contentCategories, authorizationScopes, matrixMetadata string
	var expiresAt, lastProfileSyncedAt, lastMetricsSyncedAt, nextSyncAt sql.NullTime
	var credentialFields string
	var snapshotID, snapshotSource, snapshotFreshness string
	var snapshotCapturedAt, snapshotCreatedAt sql.NullTime
	var snapshotFollowerCount, snapshotFollowingCount, snapshotContentCount int
	var snapshotTotalLikeCount, snapshotTotalFavoriteCount, snapshotTotalCommentCount, snapshotTotalShareCount int
	var snapshotEngagementRate float64
	var snapshotAudienceSignals, snapshotProfile, snapshotRawMetrics string
	var syncJobID, syncRequestedByUserID, syncType, syncStatus, idempotencyKey, requestPayload, resultSummary, errorMessage string
	var syncRequestedAt, syncStartedAt, syncFinishedAt, syncCreatedAt, syncUpdatedAt sql.NullTime
	var contentMetricCount int

	if err := rows.Scan(
		&item.Account.ID,
		&item.Account.WorkspaceID,
		&item.Account.PlatformID,
		&item.Account.Name,
		&item.Account.ExternalID,
		&item.Account.Status,
		&credentials,
		&expiresAt,
		&item.Account.LastCheckedAt,
		&item.Account.AccountGroup,
		&item.Account.OwnershipType,
		&item.Account.OperatingRole,
		&item.Account.Persona,
		&item.Account.Positioning,
		&item.Account.TargetAudience,
		&contentCategories,
		&item.Account.HealthStatus,
		&item.Account.HealthNotes,
		&authorizationScopes,
		&item.Account.SyncEnabled,
		&item.Account.LastSyncJobID,
		&item.Account.LastSyncStatus,
		&item.Account.LastSyncMessage,
		&lastProfileSyncedAt,
		&lastMetricsSyncedAt,
		&nextSyncAt,
		&matrixMetadata,
		&item.Platform.ID,
		&item.Platform.Name,
		&item.Platform.Type,
		&item.Platform.Enabled,
		&item.Platform.SupportsArticle,
		&item.Platform.SupportsImage,
		&item.Platform.SupportsScheduling,
		&credentialFields,
		&snapshotID,
		&snapshotSource,
		&snapshotCapturedAt,
		&snapshotFollowerCount,
		&snapshotFollowingCount,
		&snapshotContentCount,
		&snapshotTotalLikeCount,
		&snapshotTotalFavoriteCount,
		&snapshotTotalCommentCount,
		&snapshotTotalShareCount,
		&snapshotEngagementRate,
		&snapshotAudienceSignals,
		&snapshotProfile,
		&snapshotRawMetrics,
		&snapshotFreshness,
		&snapshotCreatedAt,
		&syncJobID,
		&syncRequestedByUserID,
		&syncType,
		&syncStatus,
		&syncRequestedAt,
		&syncStartedAt,
		&syncFinishedAt,
		&idempotencyKey,
		&requestPayload,
		&resultSummary,
		&errorMessage,
		&syncCreatedAt,
		&syncUpdatedAt,
		&contentMetricCount,
	); err != nil {
		return model.MediaAccountMatrixItem{}, err
	}

	hydrateMediaAccountFromStorage(&item.Account, credentials, contentCategories, authorizationScopes, matrixMetadata, expiresAt, lastProfileSyncedAt, lastMetricsSyncedAt, nextSyncAt)
	item.Platform.CredentialFields = decodeStringSlice(credentialFields)
	item.ContentMetricCount = contentMetricCount
	item.DataFreshness = mediaAccountDataFreshness(item.Account, snapshotCapturedAt)
	item.Warnings = mediaAccountMatrixWarnings(item.Account, item.Platform, item.DataFreshness)

	if snapshotID != "" && snapshotCapturedAt.Valid {
		item.LatestSnapshot = &model.MediaAccountMetricSnapshot{
			ID:                 snapshotID,
			WorkspaceID:        item.Account.WorkspaceID,
			MediaAccountID:     item.Account.ID,
			PlatformID:         item.Account.PlatformID,
			Source:             snapshotSource,
			CapturedAt:         snapshotCapturedAt.Time,
			FollowerCount:      snapshotFollowerCount,
			FollowingCount:     snapshotFollowingCount,
			ContentCount:       snapshotContentCount,
			TotalLikeCount:     snapshotTotalLikeCount,
			TotalFavoriteCount: snapshotTotalFavoriteCount,
			TotalCommentCount:  snapshotTotalCommentCount,
			TotalShareCount:    snapshotTotalShareCount,
			EngagementRate:     snapshotEngagementRate,
			AudienceSignals:    decodeAnyMap(snapshotAudienceSignals),
			Profile:            decodeAnyMap(snapshotProfile),
			RawMetrics:         decodeAnyMap(snapshotRawMetrics),
			FreshnessStatus:    defaultString(snapshotFreshness, "fresh"),
			CreatedAt:          nullableTimeValue(snapshotCreatedAt),
		}
	}
	if syncJobID != "" && syncRequestedAt.Valid {
		item.LatestSyncJob = &model.MediaAccountSyncJob{
			ID:                syncJobID,
			WorkspaceID:       item.Account.WorkspaceID,
			MediaAccountID:    item.Account.ID,
			PlatformID:        item.Account.PlatformID,
			RequestedByUserID: syncRequestedByUserID,
			SyncType:          syncType,
			Status:            syncStatus,
			RequestedAt:       syncRequestedAt.Time,
			StartedAt:         nullableTimePtr(syncStartedAt),
			FinishedAt:        nullableTimePtr(syncFinishedAt),
			IdempotencyKey:    idempotencyKey,
			RequestPayload:    decodeAnyMap(requestPayload),
			ResultSummary:     decodeAnyMap(resultSummary),
			ErrorMessage:      errorMessage,
			CreatedAt:         nullableTimeValue(syncCreatedAt),
			UpdatedAt:         nullableTimeValue(syncUpdatedAt),
		}
	}
	return item, nil
}

func hydrateMediaAccountFromStorage(
	item *model.MediaAccount,
	credentials string,
	contentCategories string,
	authorizationScopes string,
	matrixMetadata string,
	expiresAt sql.NullTime,
	lastProfileSyncedAt sql.NullTime,
	lastMetricsSyncedAt sql.NullTime,
	nextSyncAt sql.NullTime,
) {
	item.CredentialMeta = decodeStringMap(credentials)
	item.LoginMethod = item.CredentialMeta["loginMethod"]
	if item.LoginMethod == "" {
		item.LoginMethod = "manual"
	}
	delete(item.CredentialMeta, "loginMethod")
	item.ContentCategories = decodeStringSlice(contentCategories)
	item.AuthorizationScopes = decodeStringSlice(authorizationScopes)
	item.MatrixMetadata = decodeAnyMap(matrixMetadata)
	if item.OwnershipType == "" {
		item.OwnershipType = "owned"
	}
	if item.OperatingRole == "" {
		item.OperatingRole = "primary"
	}
	if item.HealthStatus == "" {
		item.HealthStatus = mediaAccountHealthFromStatus(item.Status)
	}
	if expiresAt.Valid {
		item.ExpiresAt = &expiresAt.Time
	}
	item.LastProfileSyncedAt = nullableTimePtr(lastProfileSyncedAt)
	item.LastMetricsSyncedAt = nullableTimePtr(lastMetricsSyncedAt)
	item.NextSyncAt = nullableTimePtr(nextSyncAt)
}

func mediaAccountHealthFromStatus(status string) string {
	switch status {
	case "connected":
		return "healthy"
	case "pending_login", "qr_waiting":
		return "needs_authorization"
	case "expired":
		return "expired"
	default:
		return "unknown"
	}
}

func mediaAccountDataFreshness(account model.MediaAccount, snapshotCapturedAt sql.NullTime) string {
	now := time.Now().UTC()
	if snapshotCapturedAt.Valid {
		age := now.Sub(snapshotCapturedAt.Time)
		switch {
		case age <= 24*time.Hour:
			return "fresh"
		case age <= 72*time.Hour:
			return "stale"
		default:
			return "expired"
		}
	}
	if account.LastMetricsSyncedAt != nil {
		age := now.Sub(*account.LastMetricsSyncedAt)
		if age <= 72*time.Hour {
			return "stale"
		}
		return "expired"
	}
	return "missing"
}

func mediaAccountMatrixWarnings(account model.MediaAccount, platform model.MediaPlatform, freshness string) []string {
	warnings := []string{}
	if !platform.Enabled {
		warnings = append(warnings, "media platform is disabled")
	}
	if account.Status != "connected" {
		warnings = append(warnings, "media account authorization is not connected")
	}
	if freshness == "missing" {
		warnings = append(warnings, "metrics have not been synced")
	} else if freshness == "expired" {
		warnings = append(warnings, "metrics are older than the freshness window")
	}
	return warnings
}

func decodeAnyMap(value string) map[string]any {
	value = defaultString(value, "{}")
	var result map[string]any
	if err := json.Unmarshal([]byte(value), &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func nonNilAnyMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func nullableTimeValue(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func defaultString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func jsonText(value map[string]any) (string, error) {
	data, err := json.Marshal(nonNilAnyMap(value))
	if err != nil {
		return "", fmt.Errorf("marshal json metadata: %w", err)
	}
	return string(data), nil
}
