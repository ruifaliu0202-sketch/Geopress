package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"geopress/backend/internal/domain"
	"geopress/backend/internal/model"
)

type MediaPlatformRepository struct {
	db *sql.DB
}

func NewMediaPlatformRepository(db *sql.DB) MediaPlatformRepository {
	return MediaPlatformRepository{db: db}
}

func (repo MediaPlatformRepository) List(ctx context.Context) ([]model.MediaPlatform, error) {
	if repo.db == nil {
		return []model.MediaPlatform{}, nil
	}
	rows, err := repo.db.QueryContext(ctx, `
		SELECT id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields::text, capabilities::text
		FROM media_platforms
		ORDER BY name ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list media platforms: %w", err)
	}
	defer rows.Close()

	items := []model.MediaPlatform{}
	for rows.Next() {
		item, err := scanMediaPlatform(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list media platforms rows: %w", err)
	}
	return items, nil
}

func (repo MediaPlatformRepository) Save(ctx context.Context, item model.MediaPlatform) error {
	if repo.db == nil {
		return nil
	}
	return SaveMediaPlatform(ctx, repo.db, item)
}

func SaveMediaPlatform(ctx context.Context, q Queryer, item model.MediaPlatform) error {
	if q == nil {
		return errors.New("save media platform requires queryer")
	}

	item.EnsureCapabilities()
	credentialFields, err := json.Marshal(item.CredentialFields)
	if err != nil {
		return fmt.Errorf("marshal media platform credential fields: %w", err)
	}
	capabilities, err := json.Marshal(item.Capabilities)
	if err != nil {
		return fmt.Errorf("marshal media platform capabilities: %w", err)
	}

	now := time.Now().UTC()
	_, err = q.ExecContext(ctx, `
		INSERT INTO media_platforms (
			id, name, type, enabled, supports_article, supports_image, supports_scheduling, credential_fields, capabilities, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10, $11)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			type = EXCLUDED.type,
			enabled = EXCLUDED.enabled,
			supports_article = EXCLUDED.supports_article,
			supports_image = EXCLUDED.supports_image,
			supports_scheduling = EXCLUDED.supports_scheduling,
			credential_fields = EXCLUDED.credential_fields,
			capabilities = EXCLUDED.capabilities,
			updated_at = EXCLUDED.updated_at
	`, item.ID, item.Name, item.Type, item.Enabled, item.SupportsArticle, item.SupportsImage, item.SupportsScheduling, string(credentialFields), string(capabilities), now, now)
	if err != nil {
		return fmt.Errorf("save media platform: %w", err)
	}
	return nil
}

type mediaPlatformScanner interface {
	Scan(dest ...any) error
}

func scanMediaPlatform(scanner mediaPlatformScanner) (model.MediaPlatform, error) {
	var item model.MediaPlatform
	var credentialFields string
	var capabilities string
	if err := scanner.Scan(&item.ID, &item.Name, &item.Type, &item.Enabled, &item.SupportsArticle, &item.SupportsImage, &item.SupportsScheduling, &credentialFields, &capabilities); err != nil {
		return model.MediaPlatform{}, fmt.Errorf("scan media platform: %w", err)
	}
	item.CredentialFields = decodeStringSlice(credentialFields)
	item.Capabilities = decodeMediaPlatformCapabilities(capabilities, item.SupportsArticle, item.SupportsImage, item.SupportsScheduling, item.CredentialFields)
	return item, nil
}

func decodeMediaPlatformCapabilities(value string, supportsArticle, supportsImage, supportsScheduling bool, credentialFields []string) domain.MediaPlatformCapabilities {
	value = strings.TrimSpace(value)
	if value == "" {
		return domain.LegacyCapabilities(supportsArticle, supportsImage, supportsScheduling, credentialFields)
	}
	var result domain.MediaPlatformCapabilities
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return domain.LegacyCapabilities(supportsArticle, supportsImage, supportsScheduling, credentialFields)
	}
	if result.IsZero() {
		return domain.LegacyCapabilities(supportsArticle, supportsImage, supportsScheduling, credentialFields)
	}
	return result.WithDefaults()
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
