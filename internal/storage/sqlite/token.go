package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateToken 创建 Token
func (s *SQLiteStore) CreateToken(ctx context.Context, token *model.DataToken) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO data_tokens (source_id, token_hash, name, status, expires_at, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := s.db.ExecContext(ctx, query,
		token.SourceID,
		token.TokenHash,
		token.Name,
		token.Status,
		token.ExpiresAt,
		token.CreatedBy,
		time.Now(),
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetTokenByHash 根据哈希获取 Token
func (s *SQLiteStore) GetTokenByHash(ctx context.Context, hash string) (*model.DataToken, error) {
	query := `
		SELECT id, source_id, token_hash, name, status, expires_at, last_used_at, created_by, created_at
		FROM data_tokens
		WHERE token_hash = ?
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
func (s *SQLiteStore) ListTokensBySourceID(ctx context.Context, sourceID int64) ([]*model.DataToken, error) {
	query := `
		SELECT id, source_id, token_hash, name, status, expires_at, last_used_at, created_by, created_at
		FROM data_tokens
		WHERE source_id = ?
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
func (s *SQLiteStore) UpdateTokenStatus(ctx context.Context, id int64, status int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE data_tokens SET status = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

// UpdateTokenLastUsed 更新 Token 最后使用时间
func (s *SQLiteStore) UpdateTokenLastUsed(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `UPDATE data_tokens SET last_used_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	return err
}

// DeleteToken 删除 Token
func (s *SQLiteStore) DeleteToken(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `DELETE FROM data_tokens WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}
