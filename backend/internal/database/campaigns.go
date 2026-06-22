package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"geopress/backend/internal/model"
)

func (db *DB) loadCampaigns(ctx context.Context) ([]model.Campaign, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			name,
			description,
			status,
			goal,
			to_json(products)::text,
			to_json(target_audiences)::text,
			to_json(channels)::text,
			to_json(media_account_ids)::text,
			start_at,
			end_at,
			budget_cents,
			currency,
			content_quota,
			approval_policy,
			to_json(success_metrics)::text,
			metadata::text,
			created_at,
			updated_at
		FROM campaigns
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Campaign{}
	for rows.Next() {
		var item model.Campaign
		var status string
		var products string
		var targetAudiences string
		var channels string
		var mediaAccountIDs string
		var successMetrics string
		var metadata string
		var startAt sql.NullTime
		var endAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.Name,
			&item.Description,
			&status,
			&item.Goal,
			&products,
			&targetAudiences,
			&channels,
			&mediaAccountIDs,
			&startAt,
			&endAt,
			&item.BudgetCents,
			&item.Currency,
			&item.ContentQuota,
			&item.ApprovalPolicy,
			&successMetrics,
			&metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Status = model.CampaignStatus(status)
		item.Products = decodeStringSlice(products)
		item.TargetAudiences = decodeStringSlice(targetAudiences)
		item.Channels = decodeStringSlice(channels)
		item.MediaAccountIDs = decodeStringSlice(mediaAccountIDs)
		item.SuccessMetrics = decodeStringSlice(successMetrics)
		item.Metadata = decodeJSONMap(metadata)
		if startAt.Valid {
			item.StartAt = &startAt.Time
		}
		if endAt.Valid {
			item.EndAt = &endAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadCampaignTopics(ctx context.Context) ([]model.CampaignTopic, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			campaign_id,
			title,
			brief,
			content_type,
			target_audience,
			funnel_stage,
			to_json(keywords)::text,
			status,
			COALESCE(content_id, ''),
			metadata::text,
			created_at,
			updated_at
		FROM campaign_topics
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CampaignTopic{}
	for rows.Next() {
		var item model.CampaignTopic
		var keywords string
		var status string
		var metadata string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.CampaignID,
			&item.Title,
			&item.Brief,
			&item.ContentType,
			&item.TargetAudience,
			&item.FunnelStage,
			&keywords,
			&status,
			&item.ContentID,
			&metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Keywords = decodeStringSlice(keywords)
		item.Status = model.CampaignTopicStatus(status)
		item.Metadata = decodeJSONMap(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadCampaignCalendarItems(ctx context.Context) ([]model.CampaignCalendarItem, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			campaign_id,
			COALESCE(topic_id, ''),
			COALESCE(content_id, ''),
			COALESCE(publish_schedule_id, ''),
			COALESCE(publish_job_id, ''),
			COALESCE(media_account_id, ''),
			COALESCE(assigned_user_id, ''),
			title,
			brief,
			content_type,
			channel,
			publish_window_start_at,
			publish_window_end_at,
			status,
			to_json(dependency_item_ids)::text,
			approval_required,
			approval_status,
			metadata::text,
			created_at,
			updated_at
		FROM campaign_calendar_items
		ORDER BY publish_window_start_at ASC NULLS LAST, created_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CampaignCalendarItem{}
	for rows.Next() {
		var item model.CampaignCalendarItem
		var status string
		var dependencyItemIDs string
		var metadata string
		var publishWindowStartAt sql.NullTime
		var publishWindowEndAt sql.NullTime
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.CampaignID,
			&item.TopicID,
			&item.ContentID,
			&item.PublishScheduleID,
			&item.PublishJobID,
			&item.MediaAccountID,
			&item.AssignedUserID,
			&item.Title,
			&item.Brief,
			&item.ContentType,
			&item.Channel,
			&publishWindowStartAt,
			&publishWindowEndAt,
			&status,
			&dependencyItemIDs,
			&item.ApprovalRequired,
			&item.ApprovalStatus,
			&metadata,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.Status = model.CampaignCalendarItemStatus(status)
		item.DependencyItemIDs = decodeStringSlice(dependencyItemIDs)
		item.Metadata = decodeJSONMap(metadata)
		if publishWindowStartAt.Valid {
			item.PublishWindowStartAt = &publishWindowStartAt.Time
		}
		if publishWindowEndAt.Valid {
			item.PublishWindowEndAt = &publishWindowEndAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadCampaignMetrics(ctx context.Context) ([]model.CampaignMetric, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			campaign_id,
			COALESCE(calendar_item_id, ''),
			COALESCE(content_id, ''),
			COALESCE(publish_job_id, ''),
			COALESCE(media_account_id, ''),
			metric_name,
			metric_value,
			metric_unit,
			source,
			collected_at,
			metadata::text,
			created_at
		FROM campaign_metrics
		ORDER BY collected_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CampaignMetric{}
	for rows.Next() {
		var item model.CampaignMetric
		var metadata string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.CampaignID,
			&item.CalendarItemID,
			&item.ContentID,
			&item.PublishJobID,
			&item.MediaAccountID,
			&item.MetricName,
			&item.MetricValue,
			&item.MetricUnit,
			&item.Source,
			&item.CollectedAt,
			&metadata,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = decodeJSONMap(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadCampaignRollups(ctx context.Context) ([]model.CampaignRollup, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			campaign_id,
			period_start,
			period_end,
			content_count,
			scheduled_count,
			published_count,
			failed_count,
			impression_count,
			engagement_count,
			click_count,
			conversion_count,
			spend_cents,
			revenue_cents,
			metadata::text,
			created_at
		FROM campaign_rollups
		ORDER BY period_start DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CampaignRollup{}
	for rows.Next() {
		var item model.CampaignRollup
		var metadata string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.CampaignID,
			&item.PeriodStart,
			&item.PeriodEnd,
			&item.ContentCount,
			&item.ScheduledCount,
			&item.PublishedCount,
			&item.FailedCount,
			&item.ImpressionCount,
			&item.EngagementCount,
			&item.ClickCount,
			&item.ConversionCount,
			&item.SpendCents,
			&item.RevenueCents,
			&metadata,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Metadata = decodeJSONMap(metadata)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveCampaign(ctx context.Context, item model.Campaign) error {
	if db == nil || db.conn == nil {
		return nil
	}

	metadata, err := json.Marshal(defaultJSONMap(item.Metadata))
	if err != nil {
		return err
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO campaigns (
			id,
			workspace_id,
			name,
			description,
			status,
			goal,
			products,
			target_audiences,
			channels,
			media_account_ids,
			start_at,
			end_at,
			budget_cents,
			currency,
			content_quota,
			approval_policy,
			success_metrics,
			metadata,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::text[], $8::text[], $9::text[], $10::text[], $11, $12, $13, $14, $15, $16, $17::text[], $18::jsonb, $19, $20)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			status = EXCLUDED.status,
			goal = EXCLUDED.goal,
			products = EXCLUDED.products,
			target_audiences = EXCLUDED.target_audiences,
			channels = EXCLUDED.channels,
			media_account_ids = EXCLUDED.media_account_ids,
			start_at = EXCLUDED.start_at,
			end_at = EXCLUDED.end_at,
			budget_cents = EXCLUDED.budget_cents,
			currency = EXCLUDED.currency,
			content_quota = EXCLUDED.content_quota,
			approval_policy = EXCLUDED.approval_policy,
			success_metrics = EXCLUDED.success_metrics,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.Name,
		item.Description,
		item.Status,
		item.Goal,
		pgTextArray(item.Products),
		pgTextArray(item.TargetAudiences),
		pgTextArray(item.Channels),
		pgTextArray(item.MediaAccountIDs),
		timePtrSQLValue(item.StartAt),
		timePtrSQLValue(item.EndAt),
		item.BudgetCents,
		item.Currency,
		item.ContentQuota,
		item.ApprovalPolicy,
		pgTextArray(item.SuccessMetrics),
		string(metadata),
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveCampaignTopic(ctx context.Context, item model.CampaignTopic) error {
	if db == nil || db.conn == nil {
		return nil
	}

	metadata, err := json.Marshal(defaultJSONMap(item.Metadata))
	if err != nil {
		return err
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO campaign_topics (
			id,
			workspace_id,
			campaign_id,
			title,
			brief,
			content_type,
			target_audience,
			funnel_stage,
			keywords,
			status,
			content_id,
			metadata,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::text[], $10, nullif($11, ''), $12::jsonb, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			brief = EXCLUDED.brief,
			content_type = EXCLUDED.content_type,
			target_audience = EXCLUDED.target_audience,
			funnel_stage = EXCLUDED.funnel_stage,
			keywords = EXCLUDED.keywords,
			status = EXCLUDED.status,
			content_id = EXCLUDED.content_id,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.CampaignID,
		item.Title,
		item.Brief,
		item.ContentType,
		item.TargetAudience,
		item.FunnelStage,
		pgTextArray(item.Keywords),
		item.Status,
		item.ContentID,
		string(metadata),
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveCampaignCalendarItem(ctx context.Context, item model.CampaignCalendarItem) error {
	if db == nil || db.conn == nil {
		return nil
	}

	metadata, err := json.Marshal(defaultJSONMap(item.Metadata))
	if err != nil {
		return err
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO campaign_calendar_items (
			id,
			workspace_id,
			campaign_id,
			topic_id,
			content_id,
			publish_schedule_id,
			publish_job_id,
			media_account_id,
			assigned_user_id,
			title,
			brief,
			content_type,
			channel,
			publish_window_start_at,
			publish_window_end_at,
			status,
			dependency_item_ids,
			approval_required,
			approval_status,
			metadata,
			created_at,
			updated_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, ''), nullif($6, ''), nullif($7, ''), nullif($8, ''), nullif($9, ''), $10, $11, $12, $13, $14, $15, $16, $17::text[], $18, $19, $20::jsonb, $21, $22)
		ON CONFLICT (id) DO UPDATE SET
			topic_id = EXCLUDED.topic_id,
			content_id = EXCLUDED.content_id,
			publish_schedule_id = EXCLUDED.publish_schedule_id,
			publish_job_id = EXCLUDED.publish_job_id,
			media_account_id = EXCLUDED.media_account_id,
			assigned_user_id = EXCLUDED.assigned_user_id,
			title = EXCLUDED.title,
			brief = EXCLUDED.brief,
			content_type = EXCLUDED.content_type,
			channel = EXCLUDED.channel,
			publish_window_start_at = EXCLUDED.publish_window_start_at,
			publish_window_end_at = EXCLUDED.publish_window_end_at,
			status = EXCLUDED.status,
			dependency_item_ids = EXCLUDED.dependency_item_ids,
			approval_required = EXCLUDED.approval_required,
			approval_status = EXCLUDED.approval_status,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.CampaignID,
		item.TopicID,
		item.ContentID,
		item.PublishScheduleID,
		item.PublishJobID,
		item.MediaAccountID,
		item.AssignedUserID,
		item.Title,
		item.Brief,
		item.ContentType,
		item.Channel,
		timePtrSQLValue(item.PublishWindowStartAt),
		timePtrSQLValue(item.PublishWindowEndAt),
		item.Status,
		pgTextArray(item.DependencyItemIDs),
		item.ApprovalRequired,
		item.ApprovalStatus,
		string(metadata),
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveCampaignMetric(ctx context.Context, item model.CampaignMetric) error {
	if db == nil || db.conn == nil {
		return nil
	}

	metadata, err := json.Marshal(defaultJSONMap(item.Metadata))
	if err != nil {
		return err
	}
	collectedAt := item.CollectedAt
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO campaign_metrics (
			id,
			workspace_id,
			campaign_id,
			calendar_item_id,
			content_id,
			publish_job_id,
			media_account_id,
			metric_name,
			metric_value,
			metric_unit,
			source,
			collected_at,
			metadata,
			created_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), nullif($5, ''), nullif($6, ''), nullif($7, ''), $8, $9, $10, $11, $12, $13::jsonb, $14)
		ON CONFLICT (id) DO UPDATE SET
			calendar_item_id = EXCLUDED.calendar_item_id,
			content_id = EXCLUDED.content_id,
			publish_job_id = EXCLUDED.publish_job_id,
			media_account_id = EXCLUDED.media_account_id,
			metric_name = EXCLUDED.metric_name,
			metric_value = EXCLUDED.metric_value,
			metric_unit = EXCLUDED.metric_unit,
			source = EXCLUDED.source,
			collected_at = EXCLUDED.collected_at,
			metadata = EXCLUDED.metadata
	`, item.ID,
		item.WorkspaceID,
		item.CampaignID,
		item.CalendarItemID,
		item.ContentID,
		item.PublishJobID,
		item.MediaAccountID,
		item.MetricName,
		item.MetricValue,
		item.MetricUnit,
		item.Source,
		collectedAt,
		string(metadata),
		createdAt,
	)
	return err
}

func (db *DB) SaveCampaignRollup(ctx context.Context, item model.CampaignRollup) error {
	if db == nil || db.conn == nil {
		return nil
	}

	metadata, err := json.Marshal(defaultJSONMap(item.Metadata))
	if err != nil {
		return err
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO campaign_rollups (
			id,
			workspace_id,
			campaign_id,
			period_start,
			period_end,
			content_count,
			scheduled_count,
			published_count,
			failed_count,
			impression_count,
			engagement_count,
			click_count,
			conversion_count,
			spend_cents,
			revenue_cents,
			metadata,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16::jsonb, $17)
		ON CONFLICT (id) DO UPDATE SET
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			content_count = EXCLUDED.content_count,
			scheduled_count = EXCLUDED.scheduled_count,
			published_count = EXCLUDED.published_count,
			failed_count = EXCLUDED.failed_count,
			impression_count = EXCLUDED.impression_count,
			engagement_count = EXCLUDED.engagement_count,
			click_count = EXCLUDED.click_count,
			conversion_count = EXCLUDED.conversion_count,
			spend_cents = EXCLUDED.spend_cents,
			revenue_cents = EXCLUDED.revenue_cents,
			metadata = EXCLUDED.metadata
	`, item.ID,
		item.WorkspaceID,
		item.CampaignID,
		item.PeriodStart,
		item.PeriodEnd,
		item.ContentCount,
		item.ScheduledCount,
		item.PublishedCount,
		item.FailedCount,
		item.ImpressionCount,
		item.EngagementCount,
		item.ClickCount,
		item.ConversionCount,
		item.SpendCents,
		item.RevenueCents,
		string(metadata),
		createdAt,
	)
	return err
}

func decodeJSONMap(value string) map[string]any {
	var result map[string]any
	if err := json.Unmarshal([]byte(value), &result); err == nil && result != nil {
		return result
	}
	return map[string]any{}
}

func defaultJSONMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func timePtrSQLValue(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}
