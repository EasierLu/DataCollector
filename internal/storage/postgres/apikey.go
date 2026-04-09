package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateApiKey 创建 API Key
func (s *PostgresStore) CreateApiKey(ctx context.Context, apiKey *model.ApiKey) (int64, error) {
	query := `
		INSERT INTO api_keys (key_hash, name, permissions, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int64
	err := s.db.QueryRowContext(ctx, query,
		apiKey.KeyHash,
		apiKey.Name,
		apiKey.Permissions,
		apiKey.ExpiresAt,
		apiKey.CreatedBy,
		time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetApiKeyByHash 根据哈希获取 API Key
func (s *PostgresStore) GetApiKeyByHash(ctx context.Context, hash string) (*model.ApiKey, error) {
	query := `
		SELECT id, key_hash, name, permissions, expires_at, last_used_at, created_by, created_at
		FROM api_keys
		WHERE key_hash = $1
	`
	row := s.db.QueryRowContext(ctx, query, hash)

	var apiKey model.ApiKey
	err := row.Scan(
		&apiKey.ID,
		&apiKey.KeyHash,
		&apiKey.Name,
		&apiKey.Permissions,
		&apiKey.ExpiresAt,
		&apiKey.LastUsedAt,
		&apiKey.CreatedBy,
		&apiKey.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

// ListApiKeys 获取所有 API Key
func (s *PostgresStore) ListApiKeys(ctx context.Context) ([]*model.ApiKey, error) {
	query := `
		SELECT id, key_hash, name, permissions, expires_at, last_used_at, created_by, created_at
		FROM api_keys
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []*model.ApiKey
	for rows.Next() {
		var k model.ApiKey
		err := rows.Scan(
			&k.ID,
			&k.KeyHash,
			&k.Name,
			&k.Permissions,
			&k.ExpiresAt,
			&k.LastUsedAt,
			&k.CreatedBy,
			&k.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		keys = append(keys, &k)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

// UpdateApiKeyPermissions 更新 API Key 权限
func (s *PostgresStore) UpdateApiKeyPermissions(ctx context.Context, id int64, permissions string) error {
	query := `UPDATE api_keys SET permissions = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, permissions, id)
	return err
}

// UpdateApiKeyLastUsed 更新 API Key 最后使用时间
func (s *PostgresStore) UpdateApiKeyLastUsed(ctx context.Context, id int64) error {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// DeleteApiKey 删除 API Key
func (s *PostgresStore) DeleteApiKey(ctx context.Context, id int64) error {
	query := `DELETE FROM api_keys WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
