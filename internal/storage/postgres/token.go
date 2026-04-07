package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateToken 创建 Token
func (s *PostgresStore) CreateToken(ctx context.Context, token *model.DataToken) (int64, error) {
	query := `
		INSERT INTO data_tokens (source_id, token_hash, name, status, expires_at, created_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id int64
	err := s.db.QueryRowContext(ctx, query,
		token.SourceID,
		token.TokenHash,
		token.Name,
		token.Status,
		token.ExpiresAt,
		token.CreatedBy,
		time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetTokenByHash 根据哈希获取 Token
func (s *PostgresStore) GetTokenByHash(ctx context.Context, hash string) (*model.DataToken, error) {
	query := `
		SELECT id, source_id, token_hash, name, status, expires_at, last_used_at, created_by, created_at
		FROM data_tokens
		WHERE token_hash = $1
	`
	row := s.db.QueryRowContext(ctx, query, hash)

	var token model.DataToken
	err := row.Scan(
		&token.ID,
		&token.SourceID,
		&token.TokenHash,
		&token.Name,
		&token.Status,
		&token.ExpiresAt,
		&token.LastUsedAt,
		&token.CreatedBy,
		&token.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &token, nil
}

// ListTokensBySourceID 获取数据源的所有 Token
func (s *PostgresStore) ListTokensBySourceID(ctx context.Context, sourceID int64) ([]*model.DataToken, error) {
	query := `
		SELECT id, source_id, token_hash, name, status, expires_at, last_used_at, created_by, created_at
		FROM data_tokens
		WHERE source_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.QueryContext(ctx, query, sourceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*model.DataToken
	for rows.Next() {
		var token model.DataToken
		err := rows.Scan(
			&token.ID,
			&token.SourceID,
			&token.TokenHash,
			&token.Name,
			&token.Status,
			&token.ExpiresAt,
			&token.LastUsedAt,
			&token.CreatedBy,
			&token.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}

// UpdateTokenStatus 更新 Token 状态
func (s *PostgresStore) UpdateTokenStatus(ctx context.Context, id int64, status int) error {
	query := `UPDATE data_tokens SET status = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateTokenLastUsed 更新 Token 最后使用时间
func (s *PostgresStore) UpdateTokenLastUsed(ctx context.Context, id int64) error {
	query := `UPDATE data_tokens SET last_used_at = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// DeleteToken 删除 Token
func (s *PostgresStore) DeleteToken(ctx context.Context, id int64) error {
	query := `DELETE FROM data_tokens WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
