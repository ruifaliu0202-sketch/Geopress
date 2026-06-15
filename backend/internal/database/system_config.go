package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
)

var ErrSystemConfigNotFound = errors.New("system config not found")

func (db *DB) SystemSetting(ctx context.Context, key string, target any) (bool, error) {
	if db == nil || db.conn == nil {
		return false, nil
	}

	var raw []byte
	err := db.conn.QueryRowContext(ctx, `SELECT value FROM system_settings WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return false, err
	}
	return true, nil
}

func (db *DB) UpsertSystemSetting(ctx context.Context, key string, value any, valueType string, description string, updatedBy string) error {
	if db == nil || db.conn == nil {
		return nil
	}
	if valueType == "" {
		valueType = "json"
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO system_settings (key, value, value_type, is_secret, description, updated_by, created_at, updated_at)
		VALUES ($1, $2::jsonb, $3, FALSE, $4, nullif($5, ''), now(), now())
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			value_type = EXCLUDED.value_type,
			is_secret = FALSE,
			description = EXCLUDED.description,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
	`, key, string(raw), valueType, description, updatedBy)
	return err
}

func (db *DB) SystemSecret(ctx context.Context, key string) (string, bool, error) {
	if db == nil || db.conn == nil {
		return "", false, nil
	}

	var value string
	err := db.conn.QueryRowContext(ctx, `SELECT encrypted_value FROM system_secrets WHERE key = $1`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (db *DB) UpsertSystemSecret(ctx context.Context, key string, value string, provider string, description string, updatedBy string) error {
	if db == nil || db.conn == nil {
		return nil
	}
	if provider == "" {
		provider = "db"
	}
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO system_secrets (key, encrypted_value, provider, description, updated_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, nullif($5, ''), now(), now())
		ON CONFLICT (key) DO UPDATE SET
			encrypted_value = EXCLUDED.encrypted_value,
			provider = EXCLUDED.provider,
			description = EXCLUDED.description,
			updated_by = EXCLUDED.updated_by,
			updated_at = now()
	`, key, value, provider, description, updatedBy)
	return err
}

func (db *DB) DeleteSystemSecret(ctx context.Context, key string) error {
	if db == nil || db.conn == nil {
		return nil
	}
	_, err := db.conn.ExecContext(ctx, `DELETE FROM system_secrets WHERE key = $1`, key)
	return err
}
