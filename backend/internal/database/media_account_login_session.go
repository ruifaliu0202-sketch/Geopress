package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"geopress/backend/internal/model"
)

func (db *DB) SaveMediaAccountLoginSession(ctx context.Context, item model.MediaAccountLoginSession) error {
	if db == nil || db.conn == nil {
		return nil
	}
	status := item.Status
	if status == "" {
		status = "active"
	}
	createdAt := item.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := item.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO media_account_login_sessions (
			id, workspace_id, account_id, platform, profile_dir, login_url, state_file, status, expires_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			workspace_id = EXCLUDED.workspace_id,
			account_id = EXCLUDED.account_id,
			platform = EXCLUDED.platform,
			profile_dir = EXCLUDED.profile_dir,
			login_url = EXCLUDED.login_url,
			state_file = EXCLUDED.state_file,
			status = EXCLUDED.status,
			expires_at = EXCLUDED.expires_at,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.WorkspaceID, item.AccountID, item.Platform, item.ProfileDir, item.LoginURL, item.StateFile, status, item.ExpiresAt, createdAt, updatedAt)
	return err
}

func (db *DB) LatestMediaAccountLoginSession(ctx context.Context, workspaceID string, accountID string) (model.MediaAccountLoginSession, bool, error) {
	if db == nil || db.conn == nil {
		return model.MediaAccountLoginSession{}, false, nil
	}
	var item model.MediaAccountLoginSession
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, workspace_id, account_id, platform, profile_dir, login_url, state_file, status, expires_at, created_at, updated_at
		FROM media_account_login_sessions
		WHERE workspace_id = $1
		  AND account_id = $2
		  AND status = 'active'
		ORDER BY updated_at DESC
		LIMIT 1
	`, workspaceID, accountID).Scan(
		&item.ID,
		&item.WorkspaceID,
		&item.AccountID,
		&item.Platform,
		&item.ProfileDir,
		&item.LoginURL,
		&item.StateFile,
		&item.Status,
		&item.ExpiresAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return model.MediaAccountLoginSession{}, false, nil
	}
	if err != nil {
		return model.MediaAccountLoginSession{}, false, err
	}
	return item, true, nil
}

func (db *DB) CompleteMediaAccountLoginSession(ctx context.Context, id string) error {
	return db.updateMediaAccountLoginSessionStatus(ctx, id, "completed")
}

func (db *DB) ExpireMediaAccountLoginSession(ctx context.Context, id string) error {
	return db.updateMediaAccountLoginSessionStatus(ctx, id, "expired")
}

func (db *DB) updateMediaAccountLoginSessionStatus(ctx context.Context, id string, status string) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `
		UPDATE media_account_login_sessions
		SET status = $2,
		    updated_at = $3
		WHERE id = $1
	`, id, status, time.Now().UTC())
	return err
}
