package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"geopress/backend/internal/model"

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
) error {
	if db == nil || db.conn == nil {
		return nil
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	now := time.Now().UTC()
	for _, user := range users {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO users (id, name, email, password_hash, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, 'active', $5, $6)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				email = EXCLUDED.email,
				updated_at = EXCLUDED.updated_at
		`, user.ID, user.Name, user.Email, "demo-password-disabled", user.CreatedAt, now)
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

	err = tx.Commit()
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

	_, err := db.conn.ExecContext(ctx, `
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
		VALUES ($1, $2, nullif($3, ''), $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (id) DO UPDATE SET
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
		item.Keywords,
		item.Status,
		item.Author,
		item.Source,
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
			$1, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''),
			$6, $7, $8, $9,
			$10::jsonb, $11, $12, $13, $14::jsonb, $15::jsonb, $16::jsonb,
			$17, $18, $19, $20, $21, $22
		)
	`, item.ID,
		item.WorkspaceID,
		item.UserID,
		item.KnowledgeBaseID,
		item.ContentID,
		item.Provider,
		item.Model,
		item.ContentType,
		item.Keywords,
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
