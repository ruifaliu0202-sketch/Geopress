package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"geopress/backend/internal/model"
)

func (db *DB) loadBrandAssets(ctx context.Context) ([]model.BrandAsset, error) {
	if db == nil || db.conn == nil {
		return []model.BrandAsset{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, type, name, description, content, channels::text, tags::text, source, status, metadata::text, created_at, updated_at
		FROM brand_assets
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.BrandAsset{}
	for rows.Next() {
		item, err := scanBrandAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListBrandAssets(ctx context.Context, workspaceID string) ([]model.BrandAsset, error) {
	if db == nil || db.conn == nil {
		return []model.BrandAsset{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, type, name, description, content, channels::text, tags::text, source, status, metadata::text, created_at, updated_at
		FROM brand_assets
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.BrandAsset{}
	for rows.Next() {
		item, err := scanBrandAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) BrandAssetByID(ctx context.Context, workspaceID string, assetID string) (model.BrandAsset, bool, error) {
	if db == nil || db.conn == nil {
		return model.BrandAsset{}, false, nil
	}

	row := db.conn.QueryRowContext(ctx, `
		SELECT id, workspace_id, type, name, description, content, channels::text, tags::text, source, status, metadata::text, created_at, updated_at
		FROM brand_assets
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, assetID)
	item, err := scanBrandAsset(row)
	if err == sql.ErrNoRows {
		return model.BrandAsset{}, false, nil
	}
	if err != nil {
		return model.BrandAsset{}, false, err
	}
	return item, true, nil
}

func (db *DB) SaveBrandAsset(ctx context.Context, item model.BrandAsset) error {
	if db == nil || db.conn == nil {
		return nil
	}

	channels, err := jsonText(item.Channels, []string{})
	if err != nil {
		return err
	}
	tags, err := jsonText(item.Tags, []string{})
	if err != nil {
		return err
	}
	metadata, err := jsonText(item.Metadata, map[string]string{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	status := item.Status
	if status == "" {
		status = model.BrandAssetActive
	}

	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO brand_assets (
			id, workspace_id, type, name, description, content, channels, tags, source, status, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9, $10, $11::jsonb, $12, $13)
		ON CONFLICT (id) DO UPDATE SET
			type = EXCLUDED.type,
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			channels = EXCLUDED.channels,
			tags = EXCLUDED.tags,
			source = EXCLUDED.source,
			status = EXCLUDED.status,
			metadata = EXCLUDED.metadata,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Type, item.Name, item.Description, item.Content, channels, tags, item.Source, status, metadata, createdAt, updatedAt)
	return err
}

func (db *DB) ArchiveBrandAsset(ctx context.Context, workspaceID string, assetID string, updatedAt time.Time) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		UPDATE brand_assets
		SET status = 'archived', updated_at = $3
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, assetID, defaultTime(updatedAt))
	return err
}

func (db *DB) loadBrandGuardrails(ctx context.Context) ([]model.BrandGuardrail, error) {
	if db == nil || db.conn == nil {
		return []model.BrandGuardrail{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(asset_id, ''), name, category, channel, source_type, source_id, severity, rules::text, action, enabled, created_at, updated_at
		FROM brand_guardrails
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.BrandGuardrail{}
	for rows.Next() {
		item, err := scanBrandGuardrail(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListBrandGuardrails(ctx context.Context, workspaceID string) ([]model.BrandGuardrail, error) {
	if db == nil || db.conn == nil {
		return []model.BrandGuardrail{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, COALESCE(asset_id, ''), name, category, channel, source_type, source_id, severity, rules::text, action, enabled, created_at, updated_at
		FROM brand_guardrails
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.BrandGuardrail{}
	for rows.Next() {
		item, err := scanBrandGuardrail(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveBrandGuardrail(ctx context.Context, item model.BrandGuardrail) error {
	if db == nil || db.conn == nil {
		return nil
	}

	rules, err := jsonText(item.Rules, []string{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO brand_guardrails (
			id, workspace_id, asset_id, name, category, channel, source_type, source_id, severity, rules, action, enabled, created_at, updated_at
		)
		VALUES ($1, $2, nullif($3, ''), $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			asset_id = EXCLUDED.asset_id,
			name = EXCLUDED.name,
			category = EXCLUDED.category,
			channel = EXCLUDED.channel,
			source_type = EXCLUDED.source_type,
			source_id = EXCLUDED.source_id,
			severity = EXCLUDED.severity,
			rules = EXCLUDED.rules,
			action = EXCLUDED.action,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.AssetID, item.Name, item.Category, item.Channel, item.SourceType, item.SourceID, item.Severity, rules, item.Action, item.Enabled, createdAt, updatedAt)
	return err
}

func (db *DB) loadApprovalWorkflows(ctx context.Context) ([]model.ApprovalWorkflow, error) {
	if db == nil || db.conn == nil {
		return []model.ApprovalWorkflow{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, resource_type, resource_id, name, status, stages::text, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM approval_workflows
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalWorkflow{}
	for rows.Next() {
		item, err := scanApprovalWorkflow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListApprovalWorkflows(ctx context.Context, workspaceID string) ([]model.ApprovalWorkflow, error) {
	if db == nil || db.conn == nil {
		return []model.ApprovalWorkflow{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, resource_type, resource_id, name, status, stages::text, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM approval_workflows
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalWorkflow{}
	for rows.Next() {
		item, err := scanApprovalWorkflow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveApprovalWorkflow(ctx context.Context, item model.ApprovalWorkflow) error {
	if db == nil || db.conn == nil {
		return nil
	}
	return saveApprovalWorkflowExec(ctx, db.conn, item)
}

func (db *DB) SaveApprovalWorkflowWithTasks(ctx context.Context, workflow model.ApprovalWorkflow, tasks []model.ApprovalTask) error {
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

	if err := saveApprovalWorkflowExec(ctx, tx, workflow); err != nil {
		return err
	}
	for _, task := range tasks {
		if err := saveApprovalTaskExec(ctx, tx, task); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func saveApprovalWorkflowExec(ctx context.Context, exec sqlExecutor, item model.ApprovalWorkflow) error {
	stages, err := jsonText(item.Stages, []model.ApprovalStage{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err = exec.ExecContext(ctx, `
		INSERT INTO approval_workflows (
			id, workspace_id, resource_type, resource_id, name, status, stages, created_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, nullif($8, ''), $9, $10)
		ON CONFLICT (id) DO UPDATE SET
			resource_type = EXCLUDED.resource_type,
			resource_id = EXCLUDED.resource_id,
			name = EXCLUDED.name,
			status = EXCLUDED.status,
			stages = EXCLUDED.stages,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.ResourceType, item.ResourceID, item.Name, item.Status, stages, item.CreatedByUserID, createdAt, updatedAt)
	return err
}

func (db *DB) loadApprovalTasks(ctx context.Context) ([]model.ApprovalTask, error) {
	if db == nil || db.conn == nil {
		return []model.ApprovalTask{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, workflow_id, resource_type, resource_id, stage_name, COALESCE(assignee_user_id, ''), assignee_role, status, decision, comment, COALESCE(processed_by_user_id, ''), due_at, processed_at, created_at, updated_at
		FROM approval_tasks
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalTask{}
	for rows.Next() {
		item, err := scanApprovalTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListApprovalTasks(ctx context.Context, workspaceID string) ([]model.ApprovalTask, error) {
	if db == nil || db.conn == nil {
		return []model.ApprovalTask{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, workflow_id, resource_type, resource_id, stage_name, COALESCE(assignee_user_id, ''), assignee_role, status, decision, comment, COALESCE(processed_by_user_id, ''), due_at, processed_at, created_at, updated_at
		FROM approval_tasks
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ApprovalTask{}
	for rows.Next() {
		item, err := scanApprovalTask(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ApprovalTaskByID(ctx context.Context, workspaceID string, taskID string) (model.ApprovalTask, bool, error) {
	if db == nil || db.conn == nil {
		return model.ApprovalTask{}, false, nil
	}

	row := db.conn.QueryRowContext(ctx, `
		SELECT id, workspace_id, workflow_id, resource_type, resource_id, stage_name, COALESCE(assignee_user_id, ''), assignee_role, status, decision, comment, COALESCE(processed_by_user_id, ''), due_at, processed_at, created_at, updated_at
		FROM approval_tasks
		WHERE workspace_id = $1 AND id = $2
	`, workspaceID, taskID)
	item, err := scanApprovalTask(row)
	if err == sql.ErrNoRows {
		return model.ApprovalTask{}, false, nil
	}
	if err != nil {
		return model.ApprovalTask{}, false, err
	}
	return item, true, nil
}

func (db *DB) SaveApprovalTask(ctx context.Context, item model.ApprovalTask) error {
	if db == nil || db.conn == nil {
		return nil
	}
	return saveApprovalTaskExec(ctx, db.conn, item)
}

func saveApprovalTaskExec(ctx context.Context, exec sqlExecutor, item model.ApprovalTask) error {
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO approval_tasks (
			id, workspace_id, workflow_id, resource_type, resource_id, stage_name, assignee_user_id, assignee_role,
			status, decision, comment, processed_by_user_id, due_at, processed_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, nullif($7, ''), $8, $9, $10, $11, nullif($12, ''), $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			decision = EXCLUDED.decision,
			comment = EXCLUDED.comment,
			processed_by_user_id = EXCLUDED.processed_by_user_id,
			processed_at = EXCLUDED.processed_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.WorkflowID, item.ResourceType, item.ResourceID, item.StageName, item.AssigneeUserID, item.AssigneeRole, item.Status, item.Decision, item.Comment, item.ProcessedByUserID, item.DueAt, item.ProcessedAt, createdAt, updatedAt)
	return err
}

func (db *DB) loadComplianceChecks(ctx context.Context) ([]model.ComplianceCheck, error) {
	if db == nil || db.conn == nil {
		return []model.ComplianceCheck{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, resource_type, resource_id, channel, status, risk_level, summary, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM compliance_checks
		ORDER BY created_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ComplianceCheck{}
	for rows.Next() {
		var item model.ComplianceCheck
		var status string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ResourceType, &item.ResourceID, &item.Channel, &status, &item.RiskLevel, &item.Summary, &item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Status = model.ComplianceCheckStatus(status)
		findings, err := db.ListComplianceFindings(ctx, item.WorkspaceID, item.ID)
		if err != nil {
			return nil, err
		}
		item.Findings = findings
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListComplianceChecks(ctx context.Context, workspaceID string) ([]model.ComplianceCheck, error) {
	if db == nil || db.conn == nil {
		return []model.ComplianceCheck{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, resource_type, resource_id, channel, status, risk_level, summary, COALESCE(created_by_user_id, ''), created_at, updated_at
		FROM compliance_checks
		WHERE workspace_id = $1
		ORDER BY created_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ComplianceCheck{}
	for rows.Next() {
		var item model.ComplianceCheck
		var status string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.ResourceType, &item.ResourceID, &item.Channel, &status, &item.RiskLevel, &item.Summary, &item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Status = model.ComplianceCheckStatus(status)
		findings, err := db.ListComplianceFindings(ctx, item.WorkspaceID, item.ID)
		if err != nil {
			return nil, err
		}
		item.Findings = findings
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveComplianceCheckWithFindings(ctx context.Context, check model.ComplianceCheck) error {
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

	createdAt := defaultTime(check.CreatedAt)
	updatedAt := defaultTime(check.UpdatedAt)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO compliance_checks (
			id, workspace_id, resource_type, resource_id, channel, status, risk_level, summary, created_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, nullif($9, ''), $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			risk_level = EXCLUDED.risk_level,
			summary = EXCLUDED.summary,
			updated_at = EXCLUDED.updated_at
	`, check.ID, check.WorkspaceID, check.ResourceType, check.ResourceID, check.Channel, check.Status, check.RiskLevel, check.Summary, check.CreatedByUserID, createdAt, updatedAt)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM compliance_findings WHERE check_id = $1`, check.ID); err != nil {
		return err
	}
	for _, finding := range check.Findings {
		if err = saveComplianceFindingTx(ctx, tx, finding); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (db *DB) ListComplianceFindings(ctx context.Context, workspaceID string, checkID string) ([]model.ComplianceFinding, error) {
	if db == nil || db.conn == nil {
		return []model.ComplianceFinding{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, check_id, workspace_id, severity, category, evidence, finding, action, source_type, source_id, created_at
		FROM compliance_findings
		WHERE workspace_id = $1 AND check_id = $2
		ORDER BY created_at ASC, id ASC
	`, workspaceID, checkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ComplianceFinding{}
	for rows.Next() {
		var item model.ComplianceFinding
		if err := rows.Scan(&item.ID, &item.CheckID, &item.WorkspaceID, &item.Severity, &item.Category, &item.Evidence, &item.Finding, &item.Action, &item.SourceType, &item.SourceID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) loadAgencyClientRelations(ctx context.Context) ([]model.AgencyClientRelation, error) {
	if db == nil || db.conn == nil {
		return []model.AgencyClientRelation{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, agency_workspace_id, client_workspace_id, client_name, status, scopes::text, notes, created_at, updated_at
		FROM agency_client_relations
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.AgencyClientRelation{}
	for rows.Next() {
		var item model.AgencyClientRelation
		var scopes string
		if err := rows.Scan(&item.ID, &item.AgencyWorkspaceID, &item.ClientWorkspaceID, &item.ClientName, &item.Status, &scopes, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Scopes = decodeStringSlice(scopes)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListAgencyClientRelations(ctx context.Context, workspaceID string) ([]model.AgencyClientRelation, error) {
	if db == nil || db.conn == nil {
		return []model.AgencyClientRelation{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, agency_workspace_id, client_workspace_id, client_name, status, scopes::text, notes, created_at, updated_at
		FROM agency_client_relations
		WHERE agency_workspace_id = $1 OR client_workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.AgencyClientRelation{}
	for rows.Next() {
		var item model.AgencyClientRelation
		var scopes string
		if err := rows.Scan(&item.ID, &item.AgencyWorkspaceID, &item.ClientWorkspaceID, &item.ClientName, &item.Status, &scopes, &item.Notes, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Scopes = decodeStringSlice(scopes)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveAgencyClientRelation(ctx context.Context, item model.AgencyClientRelation) error {
	if db == nil || db.conn == nil {
		return nil
	}
	scopes, err := jsonText(item.Scopes, []string{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO agency_client_relations (
			id, agency_workspace_id, client_workspace_id, client_name, status, scopes, notes, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9)
		ON CONFLICT (agency_workspace_id, client_workspace_id) DO UPDATE SET
			client_name = EXCLUDED.client_name,
			status = EXCLUDED.status,
			scopes = EXCLUDED.scopes,
			notes = EXCLUDED.notes,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.AgencyWorkspaceID, item.ClientWorkspaceID, item.ClientName, item.Status, scopes, item.Notes, createdAt, updatedAt)
	return err
}

func (db *DB) loadReportPackages(ctx context.Context) ([]model.ReportPackage, error) {
	if db == nil || db.conn == nil {
		return []model.ReportPackage{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, name, report_type, audience, period_start, period_end, status, sections::text, metrics::text, summary, COALESCE(generated_by_user_id, ''), created_at, updated_at
		FROM report_packages
		ORDER BY period_end DESC, created_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ReportPackage{}
	for rows.Next() {
		var item model.ReportPackage
		var sections string
		var metrics string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.ReportType, &item.Audience, &item.PeriodStart, &item.PeriodEnd, &item.Status, &sections, &metrics, &item.Summary, &item.GeneratedByUserID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Sections = decodeStringSlice(sections)
		item.Metrics = decodeAnyMap(metrics)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListReportPackages(ctx context.Context, workspaceID string) ([]model.ReportPackage, error) {
	if db == nil || db.conn == nil {
		return []model.ReportPackage{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, name, report_type, audience, period_start, period_end, status, sections::text, metrics::text, summary, COALESCE(generated_by_user_id, ''), created_at, updated_at
		FROM report_packages
		WHERE workspace_id = $1
		ORDER BY period_end DESC, created_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.ReportPackage{}
	for rows.Next() {
		var item model.ReportPackage
		var sections string
		var metrics string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.Name, &item.ReportType, &item.Audience, &item.PeriodStart, &item.PeriodEnd, &item.Status, &sections, &metrics, &item.Summary, &item.GeneratedByUserID, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Sections = decodeStringSlice(sections)
		item.Metrics = decodeAnyMap(metrics)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveReportPackage(ctx context.Context, item model.ReportPackage) error {
	if db == nil || db.conn == nil {
		return nil
	}
	sections, err := jsonText(item.Sections, []string{})
	if err != nil {
		return err
	}
	metrics, err := jsonText(item.Metrics, map[string]any{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO report_packages (
			id, workspace_id, name, report_type, audience, period_start, period_end, status, sections, metrics, summary, generated_by_user_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10::jsonb, $11, nullif($12, ''), $13, $14)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			report_type = EXCLUDED.report_type,
			audience = EXCLUDED.audience,
			period_start = EXCLUDED.period_start,
			period_end = EXCLUDED.period_end,
			status = EXCLUDED.status,
			sections = EXCLUDED.sections,
			metrics = EXCLUDED.metrics,
			summary = EXCLUDED.summary,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.Name, item.ReportType, item.Audience, item.PeriodStart, item.PeriodEnd, item.Status, sections, metrics, item.Summary, item.GeneratedByUserID, createdAt, updatedAt)
	return err
}

func (db *DB) loadStrategyRecommendations(ctx context.Context) ([]model.StrategyRecommendation, error) {
	if db == nil || db.conn == nil {
		return []model.StrategyRecommendation{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, source_type, recommendation_type, title, rationale, evidence::text, action, confidence, status, created_at, updated_at
		FROM strategy_recommendations
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.StrategyRecommendation{}
	for rows.Next() {
		var item model.StrategyRecommendation
		var evidence string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.SourceType, &item.RecommendationType, &item.Title, &item.Rationale, &evidence, &item.Action, &item.Confidence, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Evidence = decodeStringSlice(evidence)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) ListStrategyRecommendations(ctx context.Context, workspaceID string) ([]model.StrategyRecommendation, error) {
	if db == nil || db.conn == nil {
		return []model.StrategyRecommendation{}, nil
	}

	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, workspace_id, source_type, recommendation_type, title, rationale, evidence::text, action, confidence, status, created_at, updated_at
		FROM strategy_recommendations
		WHERE workspace_id = $1
		ORDER BY updated_at DESC, id ASC
	`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []model.StrategyRecommendation{}
	for rows.Next() {
		var item model.StrategyRecommendation
		var evidence string
		if err := rows.Scan(&item.ID, &item.WorkspaceID, &item.SourceType, &item.RecommendationType, &item.Title, &item.Rationale, &evidence, &item.Action, &item.Confidence, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		item.Evidence = decodeStringSlice(evidence)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (db *DB) SaveStrategyRecommendation(ctx context.Context, item model.StrategyRecommendation) error {
	if db == nil || db.conn == nil {
		return nil
	}
	evidence, err := jsonText(item.Evidence, []string{})
	if err != nil {
		return err
	}
	createdAt := defaultTime(item.CreatedAt)
	updatedAt := defaultTime(item.UpdatedAt)
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO strategy_recommendations (
			id, workspace_id, source_type, recommendation_type, title, rationale, evidence, action, confidence, status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
			source_type = EXCLUDED.source_type,
			recommendation_type = EXCLUDED.recommendation_type,
			title = EXCLUDED.title,
			rationale = EXCLUDED.rationale,
			evidence = EXCLUDED.evidence,
			action = EXCLUDED.action,
			confidence = EXCLUDED.confidence,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.SourceType, item.RecommendationType, item.Title, item.Rationale, evidence, item.Action, item.Confidence, item.Status, createdAt, updatedAt)
	return err
}

type rowScanner interface {
	Scan(dest ...any) error
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func scanBrandAsset(row rowScanner) (model.BrandAsset, error) {
	var item model.BrandAsset
	var channels string
	var tags string
	var metadata string
	var status string
	if err := row.Scan(&item.ID, &item.WorkspaceID, &item.Type, &item.Name, &item.Description, &item.Content, &channels, &tags, &item.Source, &status, &metadata, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.BrandAsset{}, err
	}
	item.Channels = decodeStringSlice(channels)
	item.Tags = decodeStringSlice(tags)
	item.Metadata = decodeStringMap(metadata)
	item.Status = model.BrandAssetStatus(status)
	return item, nil
}

func scanBrandGuardrail(row rowScanner) (model.BrandGuardrail, error) {
	var item model.BrandGuardrail
	var rules string
	if err := row.Scan(&item.ID, &item.WorkspaceID, &item.AssetID, &item.Name, &item.Category, &item.Channel, &item.SourceType, &item.SourceID, &item.Severity, &rules, &item.Action, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.BrandGuardrail{}, err
	}
	item.Rules = decodeStringSlice(rules)
	return item, nil
}

func scanApprovalWorkflow(row rowScanner) (model.ApprovalWorkflow, error) {
	var item model.ApprovalWorkflow
	var status string
	var stages string
	if err := row.Scan(&item.ID, &item.WorkspaceID, &item.ResourceType, &item.ResourceID, &item.Name, &status, &stages, &item.CreatedByUserID, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.ApprovalWorkflow{}, err
	}
	item.Status = model.ApprovalWorkflowStatus(status)
	item.Stages = decodeApprovalStages(stages)
	return item, nil
}

func scanApprovalTask(row rowScanner) (model.ApprovalTask, error) {
	var item model.ApprovalTask
	var status string
	var dueAt sql.NullTime
	var processedAt sql.NullTime
	if err := row.Scan(&item.ID, &item.WorkspaceID, &item.WorkflowID, &item.ResourceType, &item.ResourceID, &item.StageName, &item.AssigneeUserID, &item.AssigneeRole, &status, &item.Decision, &item.Comment, &item.ProcessedByUserID, &dueAt, &processedAt, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return model.ApprovalTask{}, err
	}
	item.Status = model.ApprovalTaskStatus(status)
	if dueAt.Valid {
		item.DueAt = &dueAt.Time
	}
	if processedAt.Valid {
		item.ProcessedAt = &processedAt.Time
	}
	return item, nil
}

func saveComplianceFindingTx(ctx context.Context, tx *sql.Tx, item model.ComplianceFinding) error {
	createdAt := defaultTime(item.CreatedAt)
	_, err := tx.ExecContext(ctx, `
		INSERT INTO compliance_findings (
			id, check_id, workspace_id, severity, category, evidence, finding, action, source_type, source_id, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, item.ID, item.CheckID, item.WorkspaceID, item.Severity, item.Category, item.Evidence, item.Finding, item.Action, item.SourceType, item.SourceID, createdAt)
	return err
}

func jsonText(value any, fallback any) (string, error) {
	if value == nil {
		value = fallback
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeAnyMap(value string) map[string]any {
	if value == "" {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(value), &result); err != nil || result == nil {
		return map[string]any{}
	}
	return result
}

func decodeApprovalStages(value string) []model.ApprovalStage {
	value = strings.TrimSpace(value)
	if value == "" {
		return []model.ApprovalStage{}
	}
	var result []model.ApprovalStage
	if err := json.Unmarshal([]byte(value), &result); err != nil || result == nil {
		return []model.ApprovalStage{}
	}
	return result
}
