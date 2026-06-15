package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"geopress/backend/internal/model"
)

type Snapshot struct {
	Users                  []model.User
	SubscriptionPlans      []model.SubscriptionPlan
	Workspaces             []model.Workspace
	Members                []model.WorkspaceMember
	KnowledgeBases         []model.KnowledgeBase
	KnowledgeItems         []model.KnowledgeItem
	PlatformKnowledgeBases []model.PlatformKnowledgeBase
	PlatformKnowledgeItems []model.PlatformKnowledgeItem
	Platforms              []model.MediaPlatform
	Accounts               []model.MediaAccount
	Contents               []model.Content
	Schedules              []model.PublishSchedule
	Jobs                   []model.PublishJob
	Generations            []model.GenerationRequest
	TokenUsageEvents       []model.AITokenUsageEvent
}

type UserCredentials struct {
	User         model.User
	PasswordHash string
}

func (db *DB) LoadSnapshot(ctx context.Context) (Snapshot, error) {
	if db == nil || db.conn == nil {
		return Snapshot{}, nil
	}

	var snapshot Snapshot
	var err error
	if snapshot.Users, err = db.loadUsers(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.SubscriptionPlans, err = db.loadSubscriptionPlans(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Workspaces, err = db.loadWorkspaces(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Members, err = db.loadWorkspaceMembers(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.KnowledgeBases, err = db.loadKnowledgeBases(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.KnowledgeItems, err = db.loadKnowledgeItems(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.PlatformKnowledgeBases, err = db.loadPlatformKnowledgeBases(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.PlatformKnowledgeItems, err = db.loadPlatformKnowledgeItems(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Platforms, err = db.loadMediaPlatforms(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Accounts, err = db.loadMediaAccounts(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Contents, err = db.loadContents(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Schedules, err = db.loadPublishSchedules(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Jobs, err = db.loadPublishJobs(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.Generations, err = db.loadGenerationRequests(ctx); err != nil {
		return Snapshot{}, err
	}
	if snapshot.TokenUsageEvents, err = db.loadAITokenUsageEvents(ctx); err != nil {
		return Snapshot{}, err
	}

	return snapshot, nil
}

func (db *DB) UserCredentialsByEmail(ctx context.Context, email string) (UserCredentials, bool, error) {
	if db == nil || db.conn == nil {
		return UserCredentials{}, false, nil
	}

	var item UserCredentials
	var tier string
	var status string
	var planID string
	var expiresAt sql.NullTime
	var onboardingCompletedAt sql.NullTime
	err := db.conn.QueryRowContext(ctx, `
		SELECT
			id, name, email, is_platform_admin,
			subscription_tier, subscription_plan_id, subscription_status, subscription_expires_at,
			monthly_token_budget_cents, monthly_token_used_cents, monthly_token_input_used, monthly_token_output_used,
			subscription_current_period, onboarding_completed, onboarding_completed_at,
			created_at, password_hash
		FROM users
		WHERE lower(email) = lower($1)
		  AND status = 'active'
	`, email).Scan(
		&item.User.ID,
		&item.User.Name,
		&item.User.Email,
		&item.User.IsPlatformAdmin,
		&tier,
		&planID,
		&status,
		&expiresAt,
		&item.User.MonthlyTokenBudgetCents,
		&item.User.MonthlyTokenUsedCents,
		&item.User.MonthlyTokenInputUsed,
		&item.User.MonthlyTokenOutputUsed,
		&item.User.SubscriptionCurrentPeriod,
		&item.User.OnboardingCompleted,
		&onboardingCompletedAt,
		&item.User.CreatedAt,
		&item.PasswordHash,
	)
	if err == sql.ErrNoRows {
		return UserCredentials{}, false, nil
	}
	if err != nil {
		return UserCredentials{}, false, err
	}
	item.User.SubscriptionTier = model.SubscriptionTier(tier)
	item.User.SubscriptionPlanID = model.SubscriptionPlanID(planID)
	item.User.SubscriptionStatus = model.SubscriptionStatus(status)
	if expiresAt.Valid {
		item.User.SubscriptionExpiresAt = &expiresAt.Time
	}
	if onboardingCompletedAt.Valid {
		item.User.OnboardingCompletedAt = &onboardingCompletedAt.Time
	}
	return item, true, nil
}

func (db *DB) CreateRegisteredUser(ctx context.Context, user model.User, passwordHash string, workspace model.Workspace, member model.WorkspaceMember) error {
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

	now := time.Now().UTC()
	createdAt := user.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (
			id, name, email, password_hash, status, is_platform_admin,
			subscription_tier, subscription_plan_id, subscription_status, subscription_expires_at,
			monthly_token_budget_cents, monthly_token_used_cents, monthly_token_input_used, monthly_token_output_used,
			subscription_current_period, onboarding_completed, onboarding_completed_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, user.ID,
		user.Name,
		user.Email,
		passwordHash,
		user.IsPlatformAdmin,
		defaultSubscriptionTier(user.SubscriptionTier),
		defaultSubscriptionPlanID(user.SubscriptionPlanID, user.SubscriptionTier),
		defaultSubscriptionStatus(user.SubscriptionStatus),
		user.SubscriptionExpiresAt,
		user.MonthlyTokenBudgetCents,
		user.MonthlyTokenUsedCents,
		user.MonthlyTokenInputUsed,
		user.MonthlyTokenOutputUsed,
		defaultBillingPeriod(user.SubscriptionCurrentPeriod, now),
		user.OnboardingCompleted,
		user.OnboardingCompletedAt,
		createdAt,
		now,
	)
	if err != nil {
		return err
	}

	workspaceCreatedAt := workspace.CreatedAt
	if workspaceCreatedAt.IsZero() {
		workspaceCreatedAt = now
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, type, plan, status, industry, language, tone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, workspace.ID,
		workspace.Name,
		workspace.Type,
		workspace.Plan,
		workspace.Status,
		workspace.Industry,
		workspace.Language,
		workspace.Tone,
		workspaceCreatedAt,
		now,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1, $2, $3)
	`, member.WorkspaceID, member.UserID, member.Role)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) CreateUserSession(ctx context.Context, token string, userID string, expiresAt *time.Time) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO user_sessions (token, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, token, userID, expiresAt)
	return err
}

func (db *DB) UserIDBySessionToken(ctx context.Context, token string) (string, bool, error) {
	if db == nil || db.conn == nil {
		return "", false, nil
	}
	var userID string
	err := db.conn.QueryRowContext(ctx, `
		SELECT user_id
		FROM user_sessions
		WHERE token = $1
		  AND (expires_at IS NULL OR expires_at > now())
	`, token).Scan(&userID)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return userID, true, nil
}

func (db *DB) loadUsers(ctx context.Context) ([]model.User, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, name, email, is_platform_admin,
			subscription_tier, subscription_plan_id, subscription_status, subscription_expires_at,
			monthly_token_budget_cents, monthly_token_used_cents, monthly_token_input_used, monthly_token_output_used,
			subscription_current_period, onboarding_completed, onboarding_completed_at,
			created_at
		FROM users
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.User{}
	for rows.Next() {
		var item model.User
		var expiresAt sql.NullTime
		var onboardingCompletedAt sql.NullTime
		var tier string
		var planID string
		var status string
		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Email,
			&item.IsPlatformAdmin,
			&tier,
			&planID,
			&status,
			&expiresAt,
			&item.MonthlyTokenBudgetCents,
			&item.MonthlyTokenUsedCents,
			&item.MonthlyTokenInputUsed,
			&item.MonthlyTokenOutputUsed,
			&item.SubscriptionCurrentPeriod,
			&item.OnboardingCompleted,
			&onboardingCompletedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.SubscriptionTier = model.SubscriptionTier(tier)
		item.SubscriptionPlanID = model.SubscriptionPlanID(planID)
		item.SubscriptionStatus = model.SubscriptionStatus(status)
		if expiresAt.Valid {
			item.SubscriptionExpiresAt = &expiresAt.Time
		}
		if onboardingCompletedAt.Valid {
			item.OnboardingCompletedAt = &onboardingCompletedAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadSubscriptionPlans(ctx context.Context) ([]model.SubscriptionPlan, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, name, tier, price_cents, currency, monthly_token_budget_cents,
			input_token_price_per_1k, output_token_price_per_1k, enabled
		FROM subscription_plans
		ORDER BY price_cents ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.SubscriptionPlan{}
	for rows.Next() {
		var item model.SubscriptionPlan
		var id string
		var tier string
		if err := rows.Scan(
			&id,
			&item.Name,
			&tier,
			&item.PriceCents,
			&item.Currency,
			&item.MonthlyTokenBudgetCents,
			&item.InputTokenPricePer1K,
			&item.OutputTokenPricePer1K,
			&item.Enabled,
		); err != nil {
			return nil, err
		}
		item.ID = model.SubscriptionPlanID(id)
		item.Tier = model.SubscriptionTier(tier)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadWorkspaces(ctx context.Context) ([]model.Workspace, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, name, type, plan, status, industry, language, tone, created_at
		FROM workspaces
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Workspace{}
	for rows.Next() {
		var item model.Workspace
		var workspaceType string
		if err := rows.Scan(&item.ID, &item.Name, &workspaceType, &item.Plan, &item.Status, &item.Industry, &item.Language, &item.Tone, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Type = model.WorkspaceType(workspaceType)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadWorkspaceMembers(ctx context.Context) ([]model.WorkspaceMember, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT user_id, workspace_id, role
		FROM workspace_members
		ORDER BY workspace_id ASC, user_id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.WorkspaceMember{}
	for rows.Next() {
		var item model.WorkspaceMember
		if err := rows.Scan(&item.UserID, &item.WorkspaceID, &item.Role); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadKnowledgeBases(ctx context.Context) ([]model.KnowledgeBase, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			kb.id,
			kb.workspace_id,
			kb.name,
			kb.description,
			COUNT(kib.knowledge_item_id)::int AS item_count,
			kb.updated_at
		FROM knowledge_bases kb
		LEFT JOIN knowledge_item_bases kib ON kib.knowledge_base_id = kb.id
		GROUP BY kb.id, kb.workspace_id, kb.name, kb.description, kb.updated_at
		ORDER BY kb.updated_at DESC, kb.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.KnowledgeBase{}
	for rows.Next() {
		var item model.KnowledgeBase
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.Description, &item.ItemCount, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadKnowledgeItems(ctx context.Context) ([]model.KnowledgeItem, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			ki.id,
			ki.workspace_id,
			ki.type,
			ki.title,
			ki.content,
			ki.enabled,
			ki.updated_at,
			COALESCE(jsonb_agg(kib.knowledge_base_id ORDER BY kib.knowledge_base_id) FILTER (WHERE kib.knowledge_base_id IS NOT NULL), '[]'::jsonb)::text AS knowledge_base_ids
		FROM knowledge_items ki
		LEFT JOIN knowledge_item_bases kib ON kib.knowledge_item_id = ki.id
		GROUP BY ki.id, ki.workspace_id, ki.type, ki.title, ki.content, ki.enabled, ki.updated_at
		ORDER BY ki.updated_at DESC, ki.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.KnowledgeItem{}
	for rows.Next() {
		var item model.KnowledgeItem
		var baseIDs string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Type, &item.Title, &item.Content, &item.Enabled, &item.UpdatedAt, &baseIDs); err != nil {
			return nil, err
		}
		item.KnowledgeBaseIDs = decodeStringSlice(baseIDs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadPlatformKnowledgeBases(ctx context.Context) ([]model.PlatformKnowledgeBase, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			pkb.id,
			pkb.name,
			pkb.description,
			pkb.category,
			pkb.price_cents,
			pkb.currency,
			pkb.marketplace_listed,
			COUNT(pkib.platform_knowledge_item_id)::int AS item_count,
			pkb.updated_at
		FROM platform_knowledge_bases pkb
		LEFT JOIN platform_knowledge_item_bases pkib ON pkib.platform_knowledge_base_id = pkb.id
		GROUP BY pkb.id, pkb.name, pkb.description, pkb.category, pkb.price_cents, pkb.currency, pkb.marketplace_listed, pkb.updated_at
		ORDER BY pkb.updated_at DESC, pkb.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PlatformKnowledgeBase{}
	for rows.Next() {
		var item model.PlatformKnowledgeBase
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Category, &item.PriceCents, &item.Currency, &item.MarketplaceListed, &item.ItemCount, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadPlatformKnowledgeItems(ctx context.Context) ([]model.PlatformKnowledgeItem, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			pki.id,
			pki.type,
			pki.title,
			pki.content,
			pki.enabled,
			pki.updated_at,
			COALESCE(jsonb_agg(pkib.platform_knowledge_base_id ORDER BY pkib.platform_knowledge_base_id) FILTER (WHERE pkib.platform_knowledge_base_id IS NOT NULL), '[]'::jsonb)::text AS knowledge_base_ids
		FROM platform_knowledge_items pki
		LEFT JOIN platform_knowledge_item_bases pkib ON pkib.platform_knowledge_item_id = pki.id
		GROUP BY pki.id, pki.type, pki.title, pki.content, pki.enabled, pki.updated_at
		ORDER BY pki.updated_at DESC, pki.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PlatformKnowledgeItem{}
	for rows.Next() {
		var item model.PlatformKnowledgeItem
		var baseIDs string
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Content, &item.Enabled, &item.UpdatedAt, &baseIDs); err != nil {
			return nil, err
		}
		item.KnowledgeBaseIDs = decodeStringSlice(baseIDs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadMediaPlatforms(ctx context.Context) ([]model.MediaPlatform, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields::text
		FROM media_platforms
		ORDER BY name ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.MediaPlatform{}
	for rows.Next() {
		var item model.MediaPlatform
		var credentialFields string
		if err := rows.Scan(&item.ID, &item.Name, &item.Type, &item.Enabled, &item.SupportsArticle, &item.SupportsImage, &item.SupportsScheduling, &credentialFields); err != nil {
			return nil, err
		}
		item.CredentialFields = decodeStringSlice(credentialFields)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadMediaAccounts(ctx context.Context) ([]model.MediaAccount, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, platform_id, name, external_id, status, credentials::text, expires_at, COALESCE(last_checked_at, updated_at)
		FROM media_accounts
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.MediaAccount{}
	for rows.Next() {
		var item model.MediaAccount
		var credentials string
		var expiresAt sql.NullTime
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.PlatformID, &item.Name, &item.ExternalID, &item.Status, &credentials, &expiresAt, &item.LastCheckedAt); err != nil {
			return nil, err
		}
		item.CredentialMeta = decodeStringMap(credentials)
		item.LoginMethod = item.CredentialMeta["loginMethod"]
		if item.LoginMethod == "" {
			item.LoginMethod = "manual"
		}
		delete(item.CredentialMeta, "loginMethod")
		if expiresAt.Valid {
			item.ExpiresAt = &expiresAt.Time
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadContents(ctx context.Context) ([]model.Content, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(knowledge_base_id, ''), title, summary, body, to_json(keywords)::text, status, author_name, source, updated_at
		FROM contents
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Content{}
	for rows.Next() {
		var item model.Content
		var keywords string
		var status string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.KnowledgeBaseID, &item.Title, &item.Summary, &item.Body, &keywords, &status, &item.Author, &item.Source, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Keywords = decodeStringSlice(keywords)
		item.Status = model.ContentStatus(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadPublishSchedules(ctx context.Context) ([]model.PublishSchedule, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, name, COALESCE(content_id, ''), media_account_id, frequency, next_run_at, enabled, created_at
		FROM publish_schedules
		ORDER BY next_run_at ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PublishSchedule{}
	for rows.Next() {
		var item model.PublishSchedule
		var frequency string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.ContentID, &item.MediaAccountID, &frequency, &item.NextRunAt, &item.Enabled, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.Frequency = model.PublishScheduleFrequency(frequency)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadPublishJobs(ctx context.Context) ([]model.PublishJob, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(schedule_id, ''), COALESCE(content_id, ''), media_account_id, status, scheduled_at, external_url, last_message
		FROM publish_jobs
		ORDER BY scheduled_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.PublishJob{}
	for rows.Next() {
		var item model.PublishJob
		var status string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ScheduleID, &item.ContentID, &item.MediaAccountID, &status, &item.ScheduledAt, &item.ExternalURL, &item.LastMessage); err != nil {
			return nil, err
		}
		item.Status = model.PublishJobStatus(status)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadGenerationRequests(ctx context.Context) ([]model.GenerationRequest, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id,
			workspace_id,
			COALESCE(user_id, ''),
			COALESCE(knowledge_base_id, ''),
			COALESCE(content_id, ''),
			provider,
			model,
			content_type,
			to_json(keywords)::text,
			prompt::text,
			prompt_version,
			skill_id,
			skill_version,
			retrieved_knowledge_ids::text,
			raw_output::text,
			parsed_output::text,
			input_tokens,
			output_tokens,
			total_tokens,
			status,
			error_message,
			created_at
		FROM generation_requests
		ORDER BY created_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.GenerationRequest{}
	for rows.Next() {
		var item model.GenerationRequest
		var keywords string
		var retrievedKnowledgeIDs string
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.UserID,
			&item.KnowledgeBaseID,
			&item.ContentID,
			&item.Provider,
			&item.Model,
			&item.ContentType,
			&keywords,
			&item.Prompt,
			&item.PromptVersion,
			&item.SkillID,
			&item.SkillVersion,
			&retrievedKnowledgeIDs,
			&item.RawOutput,
			&item.ParsedOutput,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.Status,
			&item.ErrorMessage,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.Keywords = decodeStringSlice(keywords)
		item.RetrievedKnowledgeIDs = decodeStringSlice(retrievedKnowledgeIDs)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadAITokenUsageEvents(ctx context.Context) ([]model.AITokenUsageEvent, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, user_id, workspace_id, generation_request_id, provider, model,
			subscription_plan_id, input_tokens, output_tokens, total_tokens,
			input_cost_cents, output_cost_cents, total_cost_cents,
			billing_period, created_at
		FROM ai_token_usage_events
		ORDER BY created_at DESC, id ASC
		LIMIT 500
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.AITokenUsageEvent{}
	for rows.Next() {
		var item model.AITokenUsageEvent
		var planID string
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.WorkspaceID,
			&item.GenerationRequestID,
			&item.Provider,
			&item.Model,
			&planID,
			&item.InputTokens,
			&item.OutputTokens,
			&item.TotalTokens,
			&item.InputCostCents,
			&item.OutputCostCents,
			&item.TotalCostCents,
			&item.BillingPeriod,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.SubscriptionPlanID = model.SubscriptionPlanID(planID)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) UpdateWorkspaceProfile(ctx context.Context, workspace model.Workspace) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		UPDATE workspaces
		SET
			industry = $2,
			tone = $3,
			updated_at = $4
		WHERE id = $1
	`, workspace.ID, workspace.Industry, workspace.Tone, time.Now().UTC())
	return err
}

func (db *DB) CompleteUserOnboarding(ctx context.Context, user model.User, workspace model.Workspace) error {
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

	now := time.Now().UTC()
	_, err = tx.ExecContext(ctx, `
		UPDATE workspaces
		SET
			industry = $2,
			tone = $3,
			updated_at = $4
		WHERE id = $1
	`, workspace.ID, workspace.Industry, workspace.Tone, now)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET
			subscription_tier = $2,
			subscription_plan_id = $3,
			subscription_status = $4,
			subscription_expires_at = $5,
			monthly_token_budget_cents = $6,
			monthly_token_used_cents = $7,
			monthly_token_input_used = $8,
			monthly_token_output_used = $9,
			subscription_current_period = $10,
			onboarding_completed = $11,
			onboarding_completed_at = $12,
			updated_at = $13
		WHERE id = $1
	`, user.ID,
		defaultSubscriptionTier(user.SubscriptionTier),
		defaultSubscriptionPlanID(user.SubscriptionPlanID, user.SubscriptionTier),
		defaultSubscriptionStatus(user.SubscriptionStatus),
		user.SubscriptionExpiresAt,
		user.MonthlyTokenBudgetCents,
		user.MonthlyTokenUsedCents,
		user.MonthlyTokenInputUsed,
		user.MonthlyTokenOutputUsed,
		defaultBillingPeriod(user.SubscriptionCurrentPeriod, now),
		user.OnboardingCompleted,
		user.OnboardingCompletedAt,
		now,
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) RecordAITokenUsageEvent(ctx context.Context, event model.AITokenUsageEvent) error {
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

	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO ai_token_usage_events (
			id, user_id, workspace_id, generation_request_id, provider, model,
			subscription_plan_id, input_tokens, output_tokens, total_tokens,
			input_cost_cents, output_cost_cents, total_cost_cents,
			billing_period, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (id) DO NOTHING
	`, event.ID,
		event.UserID,
		event.WorkspaceID,
		event.GenerationRequestID,
		event.Provider,
		event.Model,
		defaultSubscriptionPlanID(event.SubscriptionPlanID, ""),
		event.InputTokens,
		event.OutputTokens,
		event.TotalTokens,
		event.InputCostCents,
		event.OutputCostCents,
		event.TotalCostCents,
		defaultBillingPeriod(event.BillingPeriod, createdAt),
		createdAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		UPDATE users
		SET
			monthly_token_used_cents = monthly_token_used_cents + $2,
			monthly_token_input_used = monthly_token_input_used + $3,
			monthly_token_output_used = monthly_token_output_used + $4,
			subscription_current_period = $5,
			updated_at = $6
		WHERE id = $1
	`, event.UserID,
		event.TotalCostCents,
		event.InputTokens,
		event.OutputTokens,
		defaultBillingPeriod(event.BillingPeriod, createdAt),
		time.Now().UTC(),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) SaveKnowledgeBase(ctx context.Context, item model.KnowledgeBase) error {
	if db == nil || db.conn == nil {
		return nil
	}
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO knowledge_bases (id, workspace_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			workspace_id = EXCLUDED.workspace_id,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Name, item.Description, updatedAt, updatedAt)
	return err
}

func (db *DB) SaveKnowledgeItem(ctx context.Context, item model.KnowledgeItem) error {
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

	updatedAt := defaultTime(item.UpdatedAt)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO knowledge_items (id, workspace_id, type, title, content, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (id) DO UPDATE SET
			workspace_id = EXCLUDED.workspace_id,
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Type, item.Title, item.Content, item.Enabled, updatedAt, updatedAt)
	if err != nil {
		return err
	}
	if err = replaceKnowledgeItemBases(ctx, tx, item.ID, item.WorkspaceID, item.KnowledgeBaseIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) AssignKnowledgeItemsToBases(ctx context.Context, workspaceID string, knowledgeItemIDs []string, knowledgeBaseIDs []string) error {
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

	for _, itemID := range knowledgeItemIDs {
		for _, baseID := range knowledgeBaseIDs {
			_, err = tx.ExecContext(ctx, `
				INSERT INTO knowledge_item_bases (knowledge_item_id, knowledge_base_id, workspace_id)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING
			`, itemID, baseID, workspaceID)
			if err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `
			UPDATE knowledge_items
			SET updated_at = $3
			WHERE id = $1 AND workspace_id = $2
		`, itemID, workspaceID, time.Now().UTC())
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) SavePlatformKnowledgeBase(ctx context.Context, item model.PlatformKnowledgeBase) error {
	if db == nil || db.conn == nil {
		return nil
	}
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO platform_knowledge_bases (
			id, name, description, category, price_cents, currency, marketplace_listed, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			category = EXCLUDED.category,
			price_cents = EXCLUDED.price_cents,
			currency = EXCLUDED.currency,
			marketplace_listed = EXCLUDED.marketplace_listed,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.Name, item.Description, item.Category, item.PriceCents, item.Currency, item.MarketplaceListed, updatedAt, updatedAt)
	return err
}

func (db *DB) SavePlatformKnowledgeItem(ctx context.Context, item model.PlatformKnowledgeItem) error {
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

	updatedAt := defaultTime(item.UpdatedAt)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO platform_knowledge_items (id, type, title, content, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.Type, item.Title, item.Content, item.Enabled, updatedAt, updatedAt)
	if err != nil {
		return err
	}
	if err = replacePlatformKnowledgeItemBases(ctx, tx, item.ID, item.KnowledgeBaseIDs); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) SaveMediaPlatform(ctx context.Context, item model.MediaPlatform) error {
	if db == nil || db.conn == nil {
		return nil
	}
	credentialFields, err := json.Marshal(item.CredentialFields)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO media_platforms (
			id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			enabled = EXCLUDED.enabled,
			supports_article = EXCLUDED.supports_article,
			supports_image = EXCLUDED.supports_image,
			supports_scheduling = EXCLUDED.supports_scheduling,
			credential_fields = EXCLUDED.credential_fields,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.Name, item.Type, item.Enabled, item.SupportsArticle, item.SupportsImage, item.SupportsScheduling, string(credentialFields), time.Now().UTC(), time.Now().UTC())
	return err
}

func (db *DB) SaveMediaAccount(ctx context.Context, item model.MediaAccount) error {
	if db == nil || db.conn == nil {
		return nil
	}
	credentials := map[string]string{}
	for key, value := range item.CredentialMeta {
		credentials[key] = value
	}
	credentials["loginMethod"] = item.LoginMethod
	credentialsJSON, err := json.Marshal(credentials)
	if err != nil {
		return err
	}
	lastCheckedAt := item.LastCheckedAt
	if lastCheckedAt.IsZero() {
		lastCheckedAt = time.Now().UTC()
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO media_accounts (
			id, workspace_id, platform_id, name, external_id, status, credentials, expires_at, last_checked_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			workspace_id = EXCLUDED.workspace_id,
			platform_id = EXCLUDED.platform_id,
			name = EXCLUDED.name,
			external_id = EXCLUDED.external_id,
			status = EXCLUDED.status,
			credentials = EXCLUDED.credentials,
			expires_at = EXCLUDED.expires_at,
			last_checked_at = EXCLUDED.last_checked_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.PlatformID, item.Name, item.ExternalID, item.Status, string(credentialsJSON), item.ExpiresAt, lastCheckedAt, lastCheckedAt, time.Now().UTC())
	return err
}

func (db *DB) SavePublishSchedule(ctx context.Context, item model.PublishSchedule) error {
	if db == nil || db.conn == nil {
		return nil
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO publish_schedules (
			id, workspace_id, name, content_id, media_account_id, frequency, rule, next_run_at, enabled, created_at, updated_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, '{}'::jsonb, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			content_id = EXCLUDED.content_id,
			media_account_id = EXCLUDED.media_account_id,
			frequency = EXCLUDED.frequency,
			next_run_at = EXCLUDED.next_run_at,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Name, item.ContentID, item.MediaAccountID, item.Frequency, item.NextRunAt, item.Enabled, createdAt, time.Now().UTC())
	return err
}

func (db *DB) SavePublishJob(ctx context.Context, item model.PublishJob) error {
	if db == nil || db.conn == nil {
		return nil
	}
	scheduledAt := item.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now().UTC()
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO publish_jobs (
			id, workspace_id, schedule_id, content_id, media_account_id, status, scheduled_at, external_url, idempotency_key, last_message, created_at, updated_at
		)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			schedule_id = EXCLUDED.schedule_id,
			content_id = EXCLUDED.content_id,
			media_account_id = EXCLUDED.media_account_id,
			status = EXCLUDED.status,
			scheduled_at = EXCLUDED.scheduled_at,
			external_url = EXCLUDED.external_url,
			last_message = EXCLUDED.last_message,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.ScheduleID, item.ContentID, item.MediaAccountID, item.Status, scheduledAt, item.ExternalURL, item.ID, item.LastMessage, scheduledAt, time.Now().UTC())
	return err
}

func (db *DB) SavePublishScheduleWithJob(ctx context.Context, schedule model.PublishSchedule, job model.PublishJob) error {
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
	if err = savePublishScheduleTx(ctx, tx, schedule); err != nil {
		return err
	}
	if err = savePublishJobTx(ctx, tx, job); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) SavePublishJobWithContent(ctx context.Context, job model.PublishJob, content model.Content) error {
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
	if err = savePublishJobTx(ctx, tx, job); err != nil {
		return err
	}
	if err = saveContentTx(ctx, tx, content); err != nil {
		return err
	}
	return tx.Commit()
}

func saveContentTx(ctx context.Context, tx *sql.Tx, item model.Content) error {
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO contents (
			id,
			workspace_id,
			knowledge_base_id,
			title,
			summary,
			body,
			keywords,
			status,
			author_name,
			source,
			created_at,
			updated_at
		)
		VALUES ($1, $2, nullif($3, ''), $4, $5, $6, $7::text[], $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			knowledge_base_id = EXCLUDED.knowledge_base_id,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			body = EXCLUDED.body,
			keywords = EXCLUDED.keywords,
			status = EXCLUDED.status,
			author_name = EXCLUDED.author_name,
			source = EXCLUDED.source,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.KnowledgeBaseID,
		item.Title,
		item.Summary,
		item.Body,
		pgTextArray(item.Keywords),
		item.Status,
		item.Author,
		item.Source,
		updatedAt,
		updatedAt,
	)
	return err
}

func replaceKnowledgeItemBases(ctx context.Context, tx *sql.Tx, itemID string, workspaceID string, baseIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_item_bases WHERE knowledge_item_id = $1`, itemID); err != nil {
		return err
	}
	for _, baseID := range baseIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO knowledge_item_bases (knowledge_item_id, knowledge_base_id, workspace_id)
			VALUES ($1, $2, $3)
			ON CONFLICT DO NOTHING
		`, itemID, baseID, workspaceID); err != nil {
			return err
		}
	}
	return nil
}

func replacePlatformKnowledgeItemBases(ctx context.Context, tx *sql.Tx, itemID string, baseIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM platform_knowledge_item_bases WHERE platform_knowledge_item_id = $1`, itemID); err != nil {
		return err
	}
	for _, baseID := range baseIDs {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO platform_knowledge_item_bases (platform_knowledge_item_id, platform_knowledge_base_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, itemID, baseID); err != nil {
			return err
		}
	}
	return nil
}

func savePublishScheduleTx(ctx context.Context, tx *sql.Tx, item model.PublishSchedule) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO publish_schedules (
			id, workspace_id, name, content_id, media_account_id, frequency, rule, next_run_at, enabled, created_at, updated_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, '{}'::jsonb, $7, $8, $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			content_id = EXCLUDED.content_id,
			media_account_id = EXCLUDED.media_account_id,
			frequency = EXCLUDED.frequency,
			next_run_at = EXCLUDED.next_run_at,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Name, item.ContentID, item.MediaAccountID, item.Frequency, item.NextRunAt, item.Enabled, createdAt, time.Now().UTC())
	return err
}

func savePublishJobTx(ctx context.Context, tx *sql.Tx, item model.PublishJob) error {
	scheduledAt := item.ScheduledAt
	if scheduledAt.IsZero() {
		scheduledAt = time.Now().UTC()
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO publish_jobs (
			id, workspace_id, schedule_id, content_id, media_account_id, status, scheduled_at, external_url, idempotency_key, last_message, created_at, updated_at
		)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			schedule_id = EXCLUDED.schedule_id,
			content_id = EXCLUDED.content_id,
			media_account_id = EXCLUDED.media_account_id,
			status = EXCLUDED.status,
			scheduled_at = EXCLUDED.scheduled_at,
			external_url = EXCLUDED.external_url,
			last_message = EXCLUDED.last_message,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.ScheduleID, item.ContentID, item.MediaAccountID, item.Status, scheduledAt, item.ExternalURL, item.ID, item.LastMessage, scheduledAt, time.Now().UTC())
	return err
}

func defaultTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value
}

func decodeStringSlice(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(value), &result); err == nil {
		return result
	}
	return []string{}
}

func decodeStringMap(value string) map[string]string {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]string{}
	}
	var result map[string]string
	if err := json.Unmarshal([]byte(value), &result); err == nil && result != nil {
		return result
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(value), &raw); err != nil {
		return map[string]string{}
	}
	result = map[string]string{}
	for key, rawValue := range raw {
		result[key] = fmt.Sprint(rawValue)
	}
	return result
}

func defaultSubscriptionPlanID(value model.SubscriptionPlanID, tier model.SubscriptionTier) string {
	if value != "" {
		return string(value)
	}
	if tier == model.SubscriptionTierVIP {
		return string(model.SubscriptionPlanVIP)
	}
	return string(model.SubscriptionPlanFree)
}

func defaultBillingPeriod(value string, now time.Time) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return now.Format("2006-01")
}

func pgTextArray(values []string) string {
	if len(values) == 0 {
		return "{}"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Quote(value))
	}
	return "{" + strings.Join(parts, ",") + "}"
}

func placeholders(count int) string {
	if count <= 0 {
		return ""
	}
	items := make([]string, 0, count)
	for index := 1; index <= count; index++ {
		items = append(items, fmt.Sprintf("$%d", index))
	}
	return strings.Join(items, ", ")
}

func anySlice(values []string) []any {
	items := make([]any, 0, len(values))
	for _, value := range values {
		items = append(items, value)
	}
	return items
}
