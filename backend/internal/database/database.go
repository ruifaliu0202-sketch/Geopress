package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"geopress/backend/internal/model"
	"geopress/backend/internal/repository"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type DB struct {
	conn *sql.DB
}

func Open(databaseURL string) (*DB, error) {
	if databaseURL == "" {
		return nil, nil
	}

	conn, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := conn.PingContext(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &DB{conn: conn}, nil
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	if db == nil || db.conn == nil {
		return errors.New("database is not configured")
	}
	return db.conn.PingContext(ctx)
}

func (db *DB) SQL() *sql.DB {
	if db == nil {
		return nil
	}
	return db.conn
}

func (db *DB) SeedWorkspaceData(
	ctx context.Context,
	users []model.User,
	workspaces []model.Workspace,
	members []model.WorkspaceMember,
	knowledgeBases []model.KnowledgeBase,
	knowledgeItems []model.KnowledgeItem,
	platformKnowledgeBases []model.PlatformKnowledgeBase,
	platformKnowledgeItems []model.PlatformKnowledgeItem,
	platforms []model.MediaPlatform,
	accounts []model.MediaAccount,
	contents []model.Content,
	schedules []model.PublishSchedule,
	jobs []model.PublishJob,
) error {
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
	for _, user := range users {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (
				id, name, email, password_hash, status, is_platform_admin,
				subscription_tier, subscription_status, subscription_expires_at,
				created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, 'active', $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				email = EXCLUDED.email,
				is_platform_admin = EXCLUDED.is_platform_admin,
				subscription_tier = EXCLUDED.subscription_tier,
				subscription_status = EXCLUDED.subscription_status,
				subscription_expires_at = EXCLUDED.subscription_expires_at,
				updated_at = EXCLUDED.updated_at
		`,
			user.ID,
			user.Name,
			user.Email,
			"demo-password-disabled",
			user.IsPlatformAdmin,
			defaultSubscriptionTier(user.SubscriptionTier),
			defaultSubscriptionStatus(user.SubscriptionStatus),
			user.SubscriptionExpiresAt,
			user.CreatedAt,
			now,
		)
		if err != nil {
			return err
		}
	}

	for _, workspace := range workspaces {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO workspaces (id, name, type, plan, status, industry, language, tone, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				type = EXCLUDED.type,
				plan = EXCLUDED.plan,
				status = EXCLUDED.status,
				industry = EXCLUDED.industry,
				language = EXCLUDED.language,
				tone = EXCLUDED.tone,
				updated_at = EXCLUDED.updated_at
		`, workspace.ID, workspace.Name, workspace.Type, workspace.Plan, workspace.Status, workspace.Industry, workspace.Language, workspace.Tone, workspace.CreatedAt, now)
		if err != nil {
			return err
		}
	}

	for _, member := range members {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO workspace_members (workspace_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = EXCLUDED.role
		`, member.WorkspaceID, member.UserID, member.Role)
		if err != nil {
			return err
		}
	}

	for _, base := range knowledgeBases {
		updatedAt := base.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO knowledge_bases (id, workspace_id, name, description, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				workspace_id = EXCLUDED.workspace_id,
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				updated_at = EXCLUDED.updated_at
		`, base.ID, base.WorkspaceID, base.Name, base.Description, updatedAt, updatedAt)
		if err != nil {
			return err
		}
	}

	for _, item := range knowledgeItems {
		updatedAt := item.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
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
	}

	for _, base := range platformKnowledgeBases {
		updatedAt := base.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		_, err = tx.ExecContext(ctx, `
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
		`, base.ID, base.Name, base.Description, base.Category, base.PriceCents, base.Currency, base.MarketplaceListed, updatedAt, updatedAt)
		if err != nil {
			return err
		}
	}

	for _, item := range platformKnowledgeItems {
		updatedAt := item.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
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
	}

	allowedPlatformIDs := make([]string, 0, len(platforms))
	for _, platform := range platforms {
		allowedPlatformIDs = append(allowedPlatformIDs, platform.ID)
	}
	if len(allowedPlatformIDs) > 0 {
		_, err = tx.ExecContext(ctx, `
			DELETE FROM media_accounts
			WHERE platform_id NOT IN (`+placeholders(len(allowedPlatformIDs))+`)
		`, anySlice(allowedPlatformIDs)...)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `
			DELETE FROM media_platforms
			WHERE id NOT IN (`+placeholders(len(allowedPlatformIDs))+`)
		`, anySlice(allowedPlatformIDs)...)
		if err != nil {
			return err
		}
	}
	for _, platform := range platforms {
		if err = repository.SaveMediaPlatform(ctx, tx, platform); err != nil {
			return err
		}
	}

	for _, account := range accounts {
		credentials := map[string]string{}
		for key, value := range account.CredentialMeta {
			credentials[key] = value
		}
		credentials["loginMethod"] = account.LoginMethod
		credentialsJSON, marshalErr := json.Marshal(credentials)
		if marshalErr != nil {
			return marshalErr
		}
		matrixMetadata, marshalErr := jsonText(account.MatrixMetadata)
		if marshalErr != nil {
			return marshalErr
		}
		lastCheckedAt := account.LastCheckedAt
		if lastCheckedAt.IsZero() {
			lastCheckedAt = now
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO media_accounts (
				id, workspace_id, platform_id, name, external_id, status, credentials, expires_at, last_checked_at,
				account_group, ownership_type, operating_role, persona, positioning, target_audience,
				content_categories, health_status, health_notes, authorization_scopes, sync_enabled,
				last_sync_job_id, last_sync_status, last_sync_message, last_profile_synced_at, last_metrics_synced_at,
				next_sync_at, matrix_metadata, created_at, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12, $13, $14, $15, $16::text[], $17, $18, $19::text[], $20, $21, $22, $23, $24, $25, $26, $27::jsonb, $28, $29)
			ON CONFLICT (id) DO UPDATE SET
				workspace_id = EXCLUDED.workspace_id,
				platform_id = EXCLUDED.platform_id,
				name = EXCLUDED.name,
				external_id = EXCLUDED.external_id,
				status = EXCLUDED.status,
				credentials = EXCLUDED.credentials,
				expires_at = EXCLUDED.expires_at,
				last_checked_at = EXCLUDED.last_checked_at,
				account_group = EXCLUDED.account_group,
				ownership_type = EXCLUDED.ownership_type,
				operating_role = EXCLUDED.operating_role,
				persona = EXCLUDED.persona,
				positioning = EXCLUDED.positioning,
				target_audience = EXCLUDED.target_audience,
				content_categories = EXCLUDED.content_categories,
				health_status = EXCLUDED.health_status,
				health_notes = EXCLUDED.health_notes,
				authorization_scopes = EXCLUDED.authorization_scopes,
				sync_enabled = EXCLUDED.sync_enabled,
				last_sync_job_id = EXCLUDED.last_sync_job_id,
				last_sync_status = EXCLUDED.last_sync_status,
				last_sync_message = EXCLUDED.last_sync_message,
				last_profile_synced_at = EXCLUDED.last_profile_synced_at,
				last_metrics_synced_at = EXCLUDED.last_metrics_synced_at,
				next_sync_at = EXCLUDED.next_sync_at,
				matrix_metadata = EXCLUDED.matrix_metadata,
				updated_at = EXCLUDED.updated_at
		`, account.ID,
			account.WorkspaceID,
			account.PlatformID,
			account.Name,
			account.ExternalID,
			account.Status,
			string(credentialsJSON),
			account.ExpiresAt,
			lastCheckedAt,
			account.AccountGroup,
			defaultString(account.OwnershipType, "owned"),
			defaultString(account.OperatingRole, "primary"),
			account.Persona,
			account.Positioning,
			account.TargetAudience,
			pgTextArray(account.ContentCategories),
			defaultString(account.HealthStatus, mediaAccountHealthFromStatus(account.Status)),
			account.HealthNotes,
			pgTextArray(account.AuthorizationScopes),
			account.SyncEnabled,
			account.LastSyncJobID,
			account.LastSyncStatus,
			account.LastSyncMessage,
			account.LastProfileSyncedAt,
			account.LastMetricsSyncedAt,
			account.NextSyncAt,
			matrixMetadata,
			lastCheckedAt,
			now,
		)
		if err != nil {
			return err
		}
	}

	for _, content := range contents {
		updatedAt := content.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = now
		}
		attributionMetadata, marshalErr := jsonText(content.AttributionMetadata)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO contents (
				id, workspace_id, knowledge_base_id, attributed_media_account_id, title, summary, body, keywords, status, author_name, source, metadata, created_at, updated_at
			)
			VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, $8::text[], $9, $10, $11, $12::jsonb, $13, $14)
			ON CONFLICT (id) DO UPDATE SET
				knowledge_base_id = EXCLUDED.knowledge_base_id,
				attributed_media_account_id = EXCLUDED.attributed_media_account_id,
				title = EXCLUDED.title,
				summary = EXCLUDED.summary,
				body = EXCLUDED.body,
				keywords = EXCLUDED.keywords,
				status = EXCLUDED.status,
				author_name = EXCLUDED.author_name,
				source = EXCLUDED.source,
				metadata = EXCLUDED.metadata,
				updated_at = EXCLUDED.updated_at
		`, content.ID, content.WorkspaceID, content.KnowledgeBaseID, content.AttributedMediaAccountID, content.Title, content.Summary, content.Body, pgTextArray(content.Keywords), content.Status, content.Author, content.Source, attributionMetadata, updatedAt, updatedAt)
		if err != nil {
			return err
		}
	}

	for _, schedule := range schedules {
		if err = savePublishScheduleTx(ctx, tx, schedule); err != nil {
			return err
		}
	}

	for _, job := range jobs {
		if err = savePublishJobTx(ctx, tx, job); err != nil {
			return err
		}
	}

	err = tx.Commit()
	return err
}

func defaultSubscriptionTier(value model.SubscriptionTier) string {
	if value == "" {
		return string(model.SubscriptionTierFree)
	}
	return string(value)
}

func defaultSubscriptionStatus(value model.SubscriptionStatus) string {
	if value == "" {
		return string(model.SubscriptionStatusActive)
	}
	return string(value)
}

func (db *DB) UpdateUserSubscription(ctx context.Context, user model.User) error {
	if db == nil || db.conn == nil {
		return nil
	}

	_, err := db.conn.ExecContext(ctx, `
		UPDATE users
		SET
			subscription_tier = $2,
			subscription_status = $3,
			subscription_expires_at = $4,
			subscription_plan_id = $5,
			monthly_token_budget_cents = $6,
			subscription_current_period = $7,
			updated_at = $8
		WHERE id = $1
	`, user.ID,
		defaultSubscriptionTier(user.SubscriptionTier),
		defaultSubscriptionStatus(user.SubscriptionStatus),
		user.SubscriptionExpiresAt,
		defaultSubscriptionPlanID(user.SubscriptionPlanID, user.SubscriptionTier),
		user.MonthlyTokenBudgetCents,
		defaultBillingPeriod(user.SubscriptionCurrentPeriod, time.Now().UTC()),
		time.Now().UTC(),
	)
	return err
}

func (db *DB) SaveContent(ctx context.Context, item model.Content) error {
	if db == nil || db.conn == nil {
		return nil
	}

	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	attributionMetadata, err := jsonText(item.AttributionMetadata)
	if err != nil {
		return err
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO contents (
			id,
			workspace_id,
			knowledge_base_id,
			attributed_media_account_id,
			title,
			summary,
			body,
			keywords,
			status,
			author_name,
			source,
			metadata,
			created_at,
			updated_at
		)
		VALUES ($1, $2, nullif($3, ''), nullif($4, ''), $5, $6, $7, $8::text[], $9, $10, $11, $12::jsonb, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			knowledge_base_id = EXCLUDED.knowledge_base_id,
			attributed_media_account_id = EXCLUDED.attributed_media_account_id,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			body = EXCLUDED.body,
			keywords = EXCLUDED.keywords,
			status = EXCLUDED.status,
			author_name = EXCLUDED.author_name,
			source = EXCLUDED.source,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.KnowledgeBaseID,
		item.AttributedMediaAccountID,
		item.Title,
		item.Summary,
		item.Body,
		pgTextArray(item.Keywords),
		item.Status,
		item.Author,
		item.Source,
		attributionMetadata,
		updatedAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveGenerationRequest(ctx context.Context, item model.GenerationRequest) error {
	if db == nil || db.conn == nil {
		return nil
	}

	prompt := jsonString(item.Prompt)
	rawOutput := jsonString(item.RawOutput)
	parsedOutput := jsonString(item.ParsedOutput)
	retrievedKnowledgeIDs, err := json.Marshal(item.RetrievedKnowledgeIDs)
	if err != nil {
		return err
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO generation_requests (
			id,
			workspace_id,
			user_id,
			knowledge_base_id,
			content_id,
			skill_package_version_id,
			provider,
			model,
			content_type,
			keywords,
			prompt,
			prompt_version,
			skill_id,
			skill_version,
			retrieved_knowledge_ids,
			raw_output,
			parsed_output,
			input_tokens,
			output_tokens,
			total_tokens,
			status,
			error_message,
			created_at
		)
		VALUES (
			$1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), $6,
			$7, $8, $9, $10::text[],
			$11::jsonb, $12, $13, $14, $15::jsonb, $16::jsonb, $17::jsonb,
			$18, $19, $20, $21, $22, $23
		)
	`, item.ID,
		item.WorkspaceID,
		item.UserID,
		item.KnowledgeBaseID,
		item.ContentID,
		item.SkillPackageVersionID,
		item.Provider,
		item.Model,
		item.ContentType,
		pgTextArray(item.Keywords),
		prompt,
		item.PromptVersion,
		item.SkillID,
		item.SkillVersion,
		string(retrievedKnowledgeIDs),
		rawOutput,
		parsedOutput,
		item.InputTokens,
		item.OutputTokens,
		item.TotalTokens,
		item.Status,
		item.ErrorMessage,
		item.CreatedAt,
	)
	return err
}

func jsonString(value string) string {
	if value == "" {
		return "{}"
	}
	return value
}
