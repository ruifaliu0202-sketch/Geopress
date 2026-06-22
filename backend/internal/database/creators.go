package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"geopress/backend/internal/model"
)

func (db *DB) LoadCreatorSnapshot(ctx context.Context, workspaceID string) (CreatorSnapshot, error) {
	if db == nil || db.conn == nil {
		return CreatorSnapshot{}, nil
	}

	var snapshot CreatorSnapshot
	var err error
	if snapshot.Creators, err = db.ListCreators(ctx); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.MediaAccounts, err = db.ListCreatorMediaAccounts(ctx); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.Shortlists, err = db.ListCreatorShortlists(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.Briefs, err = db.ListCreatorCampaignBriefs(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.Orders, err = db.ListCreatorOrders(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.Deliverables, err = db.ListCreatorDeliverables(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.Settlements, err = db.ListCreatorSettlements(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	if snapshot.ComplianceEvidence, err = db.ListCreatorComplianceEvidence(ctx, workspaceID); err != nil {
		return CreatorSnapshot{}, err
	}
	return snapshot, nil
}

func (db *DB) ListCreators(ctx context.Context) ([]model.Creator, error) {
	if db == nil || db.conn == nil {
		return []model.Creator{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, display_name, legal_name, bio, avatar_url, contact_email,
			to_json(verticals)::text, audience_attributes::text,
			base_price_cents, currency, availability_status, collaboration_policy,
			verification_state, brand_safety_level, created_at, updated_at
		FROM creators
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.Creator{}
	for rows.Next() {
		item, err := scanCreator(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) CreatorByID(ctx context.Context, creatorID string) (model.Creator, bool, error) {
	if db == nil || db.conn == nil {
		return model.Creator{}, false, nil
	}
	row := db.conn.QueryRowContext(ctx, `
		SELECT
			id, display_name, legal_name, bio, avatar_url, contact_email,
			to_json(verticals)::text, audience_attributes::text,
			base_price_cents, currency, availability_status, collaboration_policy,
			verification_state, brand_safety_level, created_at, updated_at
		FROM creators
		WHERE id = $1
	`, creatorID)
	item, err := scanCreator(row)
	if err == sql.ErrNoRows {
		return model.Creator{}, false, nil
	}
	if err != nil {
		return model.Creator{}, false, err
	}
	return item, true, nil
}

func (db *DB) ListCreatorMediaAccounts(ctx context.Context) ([]model.CreatorMediaAccount, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorMediaAccount{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			cma.id, cma.creator_id, COALESCE(cma.platform_id, ''), cma.platform_name,
			cma.handle, cma.profile_url, cma.follower_count, cma.average_engagement_rate::float8,
			to_json(cma.verticals)::text, cma.audience_attributes::text,
			cma.account_access_mode, cma.verified, cma.created_at, cma.updated_at
		FROM creator_media_accounts cma
		ORDER BY cma.updated_at DESC, cma.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorMediaAccount{}
	for rows.Next() {
		item, err := scanCreatorMediaAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListCreatorMediaAccountsByCreator(ctx context.Context, creatorID string) ([]model.CreatorMediaAccount, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorMediaAccount{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			cma.id, cma.creator_id, COALESCE(cma.platform_id, ''), cma.platform_name,
			cma.handle, cma.profile_url, cma.follower_count, cma.average_engagement_rate::float8,
			to_json(cma.verticals)::text, cma.audience_attributes::text,
			cma.account_access_mode, cma.verified, cma.created_at, cma.updated_at
		FROM creator_media_accounts cma
		WHERE cma.creator_id = $1
		ORDER BY cma.updated_at DESC, cma.id ASC
	`, creatorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorMediaAccount{}
	for rows.Next() {
		item, err := scanCreatorMediaAccount(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListCreatorShortlists(ctx context.Context, workspaceID string) ([]model.CreatorShortlist, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorShortlist{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, creator_id, name, fit_score, qualification_status,
			brand_safety_level, brand_safety_notes, operator_notes,
			COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM creator_shortlists
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorShortlist{}
	for rows.Next() {
		var item model.CreatorShortlist
		if err := rows.Scan(
			&item.ID,
			&item.WorkspaceID,
			&item.CreatorID,
			&item.Name,
			&item.FitScore,
			&item.QualificationStatus,
			&item.BrandSafetyLevel,
			&item.BrandSafetyNotes,
			&item.OperatorNotes,
			&item.CreatedByUserID,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveCreatorShortlist(ctx context.Context, item model.CreatorShortlist) error {
	if db == nil || db.conn == nil {
		return nil
	}
	now := defaultTime(item.UpdatedAt)
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO creator_shortlists (
			id, workspace_id, creator_id, name, fit_score, qualification_status,
			brand_safety_level, brand_safety_notes, operator_notes,
			created_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, nullif($10, ''), $11, $12)
		ON CONFLICT (workspace_id, creator_id, name) DO UPDATE SET
			fit_score = EXCLUDED.fit_score,
			qualification_status = EXCLUDED.qualification_status,
			brand_safety_level = EXCLUDED.brand_safety_level,
			brand_safety_notes = EXCLUDED.brand_safety_notes,
			operator_notes = EXCLUDED.operator_notes,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.CreatorID,
		item.Name,
		item.FitScore,
		item.QualificationStatus,
		item.BrandSafetyLevel,
		item.BrandSafetyNotes,
		item.OperatorNotes,
		item.CreatedByUserID,
		createdAt,
		now,
	)
	return err
}

func (db *DB) ListCreatorCampaignBriefs(ctx context.Context, workspaceID string) ([]model.CreatorCampaignBrief, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorCampaignBrief{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, title, objective, product_name, target_audience,
			to_json(platform_targets)::text, to_json(deliverable_requirements)::text,
			to_json(disclosure_requirements)::text, to_json(prohibited_claims)::text,
			authorization_scope, content_usage_rights, review_window_hours,
			deadline_at, budget_cents, currency, status, COALESCE(created_by_user_id, ''),
			created_at, updated_at
		FROM creator_campaign_briefs
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorCampaignBrief{}
	for rows.Next() {
		item, err := scanCreatorCampaignBrief(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) CreatorCampaignBriefByID(ctx context.Context, workspaceID string, briefID string) (model.CreatorCampaignBrief, bool, error) {
	if db == nil || db.conn == nil {
		return model.CreatorCampaignBrief{}, false, nil
	}
	row := db.conn.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, title, objective, product_name, target_audience,
			to_json(platform_targets)::text, to_json(deliverable_requirements)::text,
			to_json(disclosure_requirements)::text, to_json(prohibited_claims)::text,
			authorization_scope, content_usage_rights, review_window_hours,
			deadline_at, budget_cents, currency, status, COALESCE(created_by_user_id, ''),
			created_at, updated_at
		FROM creator_campaign_briefs
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, briefID)
	item, err := scanCreatorCampaignBrief(row)
	if err == sql.ErrNoRows {
		return model.CreatorCampaignBrief{}, false, nil
	}
	if err != nil {
		return model.CreatorCampaignBrief{}, false, err
	}
	return item, true, nil
}

func (db *DB) SaveCreatorCampaignBrief(ctx context.Context, item model.CreatorCampaignBrief) error {
	if db == nil || db.conn == nil {
		return nil
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = defaultTime(item.UpdatedAt)
	}
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO creator_campaign_briefs (
			id, workspace_id, title, objective, product_name, target_audience,
			platform_targets, deliverable_requirements, disclosure_requirements,
			prohibited_claims, authorization_scope, content_usage_rights,
			review_window_hours, deadline_at, budget_cents, currency, status,
			created_by_user_id, created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7::text[], $8::text[], $9::text[], $10::text[],
			$11, $12, $13, $14, $15, $16, $17, nullif($18, ''), $19, $20
		)
		ON CONFLICT (id) DO UPDATE SET
			title = EXCLUDED.title,
			objective = EXCLUDED.objective,
			product_name = EXCLUDED.product_name,
			target_audience = EXCLUDED.target_audience,
			platform_targets = EXCLUDED.platform_targets,
			deliverable_requirements = EXCLUDED.deliverable_requirements,
			disclosure_requirements = EXCLUDED.disclosure_requirements,
			prohibited_claims = EXCLUDED.prohibited_claims,
			authorization_scope = EXCLUDED.authorization_scope,
			content_usage_rights = EXCLUDED.content_usage_rights,
			review_window_hours = EXCLUDED.review_window_hours,
			deadline_at = EXCLUDED.deadline_at,
			budget_cents = EXCLUDED.budget_cents,
			currency = EXCLUDED.currency,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.Title,
		item.Objective,
		item.ProductName,
		item.TargetAudience,
		pgTextArray(item.PlatformTargets),
		pgTextArray(item.DeliverableRequirements),
		pgTextArray(item.DisclosureRequirements),
		pgTextArray(item.ProhibitedClaims),
		item.AuthorizationScope,
		item.ContentUsageRights,
		item.ReviewWindowHours,
		item.DeadlineAt,
		item.BudgetCents,
		item.Currency,
		item.Status,
		item.CreatedByUserID,
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) ListCreatorOrders(ctx context.Context, workspaceID string) ([]model.CreatorOrder, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorOrder{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, brief_id, creator_id, status,
			price_cents, deposit_cents, service_fee_cents, currency,
			to_json(disclosure_requirements)::text, to_json(deliverable_requirements)::text,
			authorization_scope, content_usage_rights, due_at, accepted_at, completed_at,
			last_message, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM creator_orders
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorOrder{}
	for rows.Next() {
		item, err := scanCreatorOrder(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) CreatorOrderByID(ctx context.Context, workspaceID string, orderID string) (model.CreatorOrder, bool, error) {
	if db == nil || db.conn == nil {
		return model.CreatorOrder{}, false, nil
	}
	row := db.conn.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, brief_id, creator_id, status,
			price_cents, deposit_cents, service_fee_cents, currency,
			to_json(disclosure_requirements)::text, to_json(deliverable_requirements)::text,
			authorization_scope, content_usage_rights, due_at, accepted_at, completed_at,
			last_message, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM creator_orders
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, orderID)
	item, err := scanCreatorOrder(row)
	if err == sql.ErrNoRows {
		return model.CreatorOrder{}, false, nil
	}
	if err != nil {
		return model.CreatorOrder{}, false, err
	}
	return item, true, nil
}

func (db *DB) SaveCreatorOrderWithSettlement(ctx context.Context, order model.CreatorOrder, settlement model.CreatorSettlement) error {
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
	if err = saveCreatorOrderTx(ctx, tx, order); err != nil {
		return err
	}
	if err = saveCreatorSettlementTx(ctx, tx, settlement); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) SaveCreatorOrder(ctx context.Context, item model.CreatorOrder) error {
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
	if err = saveCreatorOrderTx(ctx, tx, item); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) ListCreatorDeliverables(ctx context.Context, workspaceID string) ([]model.CreatorDeliverable, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorDeliverable{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, order_id, creator_id, type, title, content,
			to_json(asset_urls)::text, status, external_url, publication_proof_url,
			publication_proof_note, review_feedback, revision,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		FROM creator_deliverables
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorDeliverable{}
	for rows.Next() {
		item, err := scanCreatorDeliverable(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) CreatorDeliverableByID(ctx context.Context, workspaceID string, deliverableID string) (model.CreatorDeliverable, bool, error) {
	if db == nil || db.conn == nil {
		return model.CreatorDeliverable{}, false, nil
	}
	row := db.conn.QueryRowContext(ctx, `
		SELECT
			id, workspace_id, order_id, creator_id, type, title, content,
			to_json(asset_urls)::text, status, external_url, publication_proof_url,
			publication_proof_note, review_feedback, revision,
			submitted_at, reviewed_at, published_at, created_at, updated_at
		FROM creator_deliverables
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, deliverableID)
	item, err := scanCreatorDeliverable(row)
	if err == sql.ErrNoRows {
		return model.CreatorDeliverable{}, false, nil
	}
	if err != nil {
		return model.CreatorDeliverable{}, false, err
	}
	return item, true, nil
}

func (db *DB) SaveCreatorDeliverable(ctx context.Context, item model.CreatorDeliverable) error {
	if db == nil || db.conn == nil {
		return nil
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = defaultTime(item.UpdatedAt)
	}
	updatedAt := defaultTime(item.UpdatedAt)
	submittedAt := item.SubmittedAt
	if submittedAt.IsZero() {
		submittedAt = createdAt
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO creator_deliverables (
			id, workspace_id, order_id, creator_id, type, title, content, asset_urls,
			status, external_url, publication_proof_url, publication_proof_note,
			review_feedback, revision, submitted_at, reviewed_at, published_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::text[], $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			asset_urls = EXCLUDED.asset_urls,
			status = EXCLUDED.status,
			external_url = EXCLUDED.external_url,
			publication_proof_url = EXCLUDED.publication_proof_url,
			publication_proof_note = EXCLUDED.publication_proof_note,
			review_feedback = EXCLUDED.review_feedback,
			revision = EXCLUDED.revision,
			reviewed_at = EXCLUDED.reviewed_at,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.OrderID,
		item.CreatorID,
		item.Type,
		item.Title,
		item.Content,
		pgTextArray(item.AssetURLs),
		item.Status,
		item.ExternalURL,
		item.PublicationProofURL,
		item.PublicationProofNote,
		item.ReviewFeedback,
		item.Revision,
		submittedAt,
		item.ReviewedAt,
		item.PublishedAt,
		createdAt,
		updatedAt,
	)
	return err
}

func (db *DB) SaveCreatorDeliverableWithOrderAndEvidence(ctx context.Context, deliverable model.CreatorDeliverable, order model.CreatorOrder, evidence model.CreatorComplianceEvidence) error {
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
	if err = saveCreatorDeliverableTx(ctx, tx, deliverable); err != nil {
		return err
	}
	if err = saveCreatorOrderTx(ctx, tx, order); err != nil {
		return err
	}
	if evidence.ID != "" {
		if err = saveCreatorComplianceEvidenceTx(ctx, tx, evidence); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) SaveCreatorPublicationProof(ctx context.Context, deliverable model.CreatorDeliverable, order model.CreatorOrder, settlement model.CreatorSettlement, evidence model.CreatorComplianceEvidence) error {
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
	if err = saveCreatorDeliverableTx(ctx, tx, deliverable); err != nil {
		return err
	}
	if err = saveCreatorOrderTx(ctx, tx, order); err != nil {
		return err
	}
	if settlement.ID != "" {
		if err = saveCreatorSettlementTx(ctx, tx, settlement); err != nil {
			return err
		}
	}
	if evidence.ID != "" {
		if err = saveCreatorComplianceEvidenceTx(ctx, tx, evidence); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListCreatorSettlements(ctx context.Context, workspaceID string) ([]model.CreatorSettlement, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorSettlement{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, order_id, creator_id, status,
			price_cents, deposit_cents, service_fee_cents, creator_payout_cents,
			currency, invoice_id, due_at, paid_at, created_at, updated_at
		FROM creator_settlements
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorSettlement{}
	for rows.Next() {
		item, err := scanCreatorSettlement(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListCreatorComplianceEvidence(ctx context.Context, workspaceID string) ([]model.CreatorComplianceEvidence, error) {
	if db == nil || db.conn == nil {
		return []model.CreatorComplianceEvidence{}, nil
	}
	rows, err := db.conn.QueryContext(ctx, `
		SELECT
			id, workspace_id, order_id, COALESCE(deliverable_id, ''), creator_id,
			evidence_type, disclosure_text, authorization_scope, content_usage_rights,
			external_url, file_url, notes, COALESCE(created_by_user_id, ''), created_at
		FROM creator_compliance_evidence
		WHERE workspace_id = $1
		ORDER BY created_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.CreatorComplianceEvidence{}
	for rows.Next() {
		item, err := scanCreatorComplianceEvidence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveCreatorComplianceEvidence(ctx context.Context, item model.CreatorComplianceEvidence) error {
	if db == nil || db.conn == nil {
		return nil
	}
	return saveCreatorComplianceEvidenceExec(ctx, db.conn, item)
}

func saveCreatorOrderTx(ctx context.Context, tx *sql.Tx, item model.CreatorOrder) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = defaultTime(item.UpdatedAt)
	}
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO creator_orders (
			id, workspace_id, brief_id, creator_id, status, price_cents,
			deposit_cents, service_fee_cents, currency, disclosure_requirements,
			deliverable_requirements, authorization_scope, content_usage_rights,
			due_at, accepted_at, completed_at, last_message, created_by_user_id,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10::text[], $11::text[],
			$12, $13, $14, $15, $16, $17, nullif($18, ''), $19, $20
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			price_cents = EXCLUDED.price_cents,
			deposit_cents = EXCLUDED.deposit_cents,
			service_fee_cents = EXCLUDED.service_fee_cents,
			currency = EXCLUDED.currency,
			disclosure_requirements = EXCLUDED.disclosure_requirements,
			deliverable_requirements = EXCLUDED.deliverable_requirements,
			authorization_scope = EXCLUDED.authorization_scope,
			content_usage_rights = EXCLUDED.content_usage_rights,
			due_at = EXCLUDED.due_at,
			accepted_at = EXCLUDED.accepted_at,
			completed_at = EXCLUDED.completed_at,
			last_message = EXCLUDED.last_message,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.BriefID,
		item.CreatorID,
		item.Status,
		item.PriceCents,
		item.DepositCents,
		item.ServiceFeeCents,
		item.Currency,
		pgTextArray(item.DisclosureRequirements),
		pgTextArray(item.DeliverableRequirements),
		item.AuthorizationScope,
		item.ContentUsageRights,
		item.DueAt,
		item.AcceptedAt,
		item.CompletedAt,
		item.LastMessage,
		item.CreatedByUserID,
		createdAt,
		updatedAt,
	)
	return err
}

func saveCreatorSettlementTx(ctx context.Context, tx *sql.Tx, item model.CreatorSettlement) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = defaultTime(item.UpdatedAt)
	}
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO creator_settlements (
			id, workspace_id, order_id, creator_id, status, price_cents,
			deposit_cents, service_fee_cents, creator_payout_cents, currency,
			invoice_id, due_at, paid_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (order_id) DO UPDATE SET
			status = EXCLUDED.status,
			price_cents = EXCLUDED.price_cents,
			deposit_cents = EXCLUDED.deposit_cents,
			service_fee_cents = EXCLUDED.service_fee_cents,
			creator_payout_cents = EXCLUDED.creator_payout_cents,
			currency = EXCLUDED.currency,
			invoice_id = EXCLUDED.invoice_id,
			due_at = EXCLUDED.due_at,
			paid_at = EXCLUDED.paid_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.OrderID,
		item.CreatorID,
		item.Status,
		item.PriceCents,
		item.DepositCents,
		item.ServiceFeeCents,
		item.CreatorPayoutCents,
		item.Currency,
		item.InvoiceID,
		item.DueAt,
		item.PaidAt,
		createdAt,
		updatedAt,
	)
	return err
}

func saveCreatorDeliverableTx(ctx context.Context, tx *sql.Tx, item model.CreatorDeliverable) error {
	if tx == nil {
		return nil
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = defaultTime(item.UpdatedAt)
	}
	updatedAt := defaultTime(item.UpdatedAt)
	submittedAt := item.SubmittedAt
	if submittedAt.IsZero() {
		submittedAt = createdAt
	}
	_, err := tx.ExecContext(ctx, `
		INSERT INTO creator_deliverables (
			id, workspace_id, order_id, creator_id, type, title, content, asset_urls,
			status, external_url, publication_proof_url, publication_proof_note,
			review_feedback, revision, submitted_at, reviewed_at, published_at,
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8::text[], $9, $10, $11, $12,
			$13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			title = EXCLUDED.title,
			content = EXCLUDED.content,
			asset_urls = EXCLUDED.asset_urls,
			status = EXCLUDED.status,
			external_url = EXCLUDED.external_url,
			publication_proof_url = EXCLUDED.publication_proof_url,
			publication_proof_note = EXCLUDED.publication_proof_note,
			review_feedback = EXCLUDED.review_feedback,
			revision = EXCLUDED.revision,
			reviewed_at = EXCLUDED.reviewed_at,
			published_at = EXCLUDED.published_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID,
		item.WorkspaceID,
		item.OrderID,
		item.CreatorID,
		item.Type,
		item.Title,
		item.Content,
		pgTextArray(item.AssetURLs),
		item.Status,
		item.ExternalURL,
		item.PublicationProofURL,
		item.PublicationProofNote,
		item.ReviewFeedback,
		item.Revision,
		submittedAt,
		item.ReviewedAt,
		item.PublishedAt,
		createdAt,
		updatedAt,
	)
	return err
}

func saveCreatorComplianceEvidenceTx(ctx context.Context, tx *sql.Tx, item model.CreatorComplianceEvidence) error {
	return saveCreatorComplianceEvidenceExec(ctx, tx, item)
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func saveCreatorComplianceEvidenceExec(ctx context.Context, exec execer, item model.CreatorComplianceEvidence) error {
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	query := `
		INSERT INTO creator_compliance_evidence (
			id, workspace_id, order_id, deliverable_id, creator_id, evidence_type,
			disclosure_text, authorization_scope, content_usage_rights, external_url,
			file_url, notes, created_by_user_id, created_at
		)
		VALUES ($1, $2, $3, nullif($4, ''), $5, $6, $7, $8, $9, $10, $11, $12, nullif($13, ''), $14)
		ON CONFLICT (id) DO NOTHING
	`
	args := []any{
		item.ID,
		item.WorkspaceID,
		item.OrderID,
		item.DeliverableID,
		item.CreatorID,
		item.EvidenceType,
		item.DisclosureText,
		item.AuthorizationScope,
		item.ContentUsageRights,
		item.ExternalURL,
		item.FileURL,
		item.Notes,
		item.CreatedByUserID,
		createdAt,
	}
	_, err := exec.ExecContext(ctx, query, args...)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCreator(row scanner) (model.Creator, error) {
	var item model.Creator
	var verticals string
	var audienceAttributes string
	var availabilityStatus string
	var verificationState string
	if err := row.Scan(
		&item.ID,
		&item.DisplayName,
		&item.LegalName,
		&item.Bio,
		&item.AvatarURL,
		&item.ContactEmail,
		&verticals,
		&audienceAttributes,
		&item.BasePriceCents,
		&item.Currency,
		&availabilityStatus,
		&item.CollaborationPolicy,
		&verificationState,
		&item.BrandSafetyLevel,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.Creator{}, err
	}
	item.Verticals = decodeStringSlice(verticals)
	item.AudienceAttributes = decodeStringMap(audienceAttributes)
	item.AvailabilityStatus = model.CreatorAvailabilityStatus(availabilityStatus)
	item.VerificationState = model.CreatorVerificationState(verificationState)
	return item, nil
}

func scanCreatorMediaAccount(row scanner) (model.CreatorMediaAccount, error) {
	var item model.CreatorMediaAccount
	var verticals string
	var audienceAttributes string
	if err := row.Scan(
		&item.ID,
		&item.CreatorID,
		&item.PlatformID,
		&item.PlatformName,
		&item.Handle,
		&item.ProfileURL,
		&item.FollowerCount,
		&item.AverageEngagementRate,
		&verticals,
		&audienceAttributes,
		&item.AccountAccessMode,
		&item.Verified,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.CreatorMediaAccount{}, err
	}
	item.Verticals = decodeStringSlice(verticals)
	item.AudienceAttributes = decodeStringMap(audienceAttributes)
	return item, nil
}

func scanCreatorCampaignBrief(row scanner) (model.CreatorCampaignBrief, error) {
	var item model.CreatorCampaignBrief
	var platformTargets string
	var deliverableRequirements string
	var disclosureRequirements string
	var prohibitedClaims string
	var status string
	var deadlineAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.Title,
		&item.Objective,
		&item.ProductName,
		&item.TargetAudience,
		&platformTargets,
		&deliverableRequirements,
		&disclosureRequirements,
		&prohibitedClaims,
		&item.AuthorizationScope,
		&item.ContentUsageRights,
		&item.ReviewWindowHours,
		&deadlineAt,
		&item.BudgetCents,
		&item.Currency,
		&status,
		&item.CreatedByUserID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.CreatorCampaignBrief{}, err
	}
	item.PlatformTargets = decodeStringSlice(platformTargets)
	item.DeliverableRequirements = decodeStringSlice(deliverableRequirements)
	item.DisclosureRequirements = decodeStringSlice(disclosureRequirements)
	item.ProhibitedClaims = decodeStringSlice(prohibitedClaims)
	item.Status = model.CreatorCampaignBriefStatus(status)
	if deadlineAt.Valid {
		item.DeadlineAt = &deadlineAt.Time
	}
	return item, nil
}

func scanCreatorOrder(row scanner) (model.CreatorOrder, error) {
	var item model.CreatorOrder
	var disclosureRequirements string
	var deliverableRequirements string
	var status string
	var dueAt sql.NullTime
	var acceptedAt sql.NullTime
	var completedAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.BriefID,
		&item.CreatorID,
		&status,
		&item.PriceCents,
		&item.DepositCents,
		&item.ServiceFeeCents,
		&item.Currency,
		&disclosureRequirements,
		&deliverableRequirements,
		&item.AuthorizationScope,
		&item.ContentUsageRights,
		&dueAt,
		&acceptedAt,
		&completedAt,
		&item.LastMessage,
		&item.CreatedByUserID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.CreatorOrder{}, err
	}
	item.Status = model.CreatorOrderStatus(status)
	item.DisclosureRequirements = decodeStringSlice(disclosureRequirements)
	item.DeliverableRequirements = decodeStringSlice(deliverableRequirements)
	if dueAt.Valid {
		item.DueAt = &dueAt.Time
	}
	if acceptedAt.Valid {
		item.AcceptedAt = &acceptedAt.Time
	}
	if completedAt.Valid {
		item.CompletedAt = &completedAt.Time
	}
	return item, nil
}

func scanCreatorDeliverable(row scanner) (model.CreatorDeliverable, error) {
	var item model.CreatorDeliverable
	var assetURLs string
	var status string
	var reviewedAt sql.NullTime
	var publishedAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.OrderID,
		&item.CreatorID,
		&item.Type,
		&item.Title,
		&item.Content,
		&assetURLs,
		&status,
		&item.ExternalURL,
		&item.PublicationProofURL,
		&item.PublicationProofNote,
		&item.ReviewFeedback,
		&item.Revision,
		&item.SubmittedAt,
		&reviewedAt,
		&publishedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.CreatorDeliverable{}, err
	}
	item.AssetURLs = decodeStringSlice(assetURLs)
	item.Status = model.CreatorDeliverableStatus(status)
	if reviewedAt.Valid {
		item.ReviewedAt = &reviewedAt.Time
	}
	if publishedAt.Valid {
		item.PublishedAt = &publishedAt.Time
	}
	return item, nil
}

func scanCreatorSettlement(row scanner) (model.CreatorSettlement, error) {
	var item model.CreatorSettlement
	var status string
	var dueAt sql.NullTime
	var paidAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.OrderID,
		&item.CreatorID,
		&status,
		&item.PriceCents,
		&item.DepositCents,
		&item.ServiceFeeCents,
		&item.CreatorPayoutCents,
		&item.Currency,
		&item.InvoiceID,
		&dueAt,
		&paidAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.CreatorSettlement{}, err
	}
	item.Status = model.CreatorSettlementStatus(status)
	if dueAt.Valid {
		item.DueAt = &dueAt.Time
	}
	if paidAt.Valid {
		item.PaidAt = &paidAt.Time
	}
	return item, nil
}

func scanCreatorComplianceEvidence(row scanner) (model.CreatorComplianceEvidence, error) {
	var item model.CreatorComplianceEvidence
	var evidenceType string
	if err := row.Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.OrderID,
		&item.DeliverableID,
		&item.CreatorID,
		&evidenceType,
		&item.DisclosureText,
		&item.AuthorizationScope,
		&item.ContentUsageRights,
		&item.ExternalURL,
		&item.FileURL,
		&item.Notes,
		&item.CreatedByUserID,
		&item.CreatedAt,
	); err != nil {
		return model.CreatorComplianceEvidence{}, err
	}
	item.EvidenceType = model.CreatorComplianceEvidenceType(evidenceType)
	return item, nil
}

func encodeStringMap(value map[string]string) (string, error) {
	if value == nil {
		value = map[string]string{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
