package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateUser 创建用户
func (s *SQLiteStore) CreateUser(ctx context.Context, user *model.User) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO users (username, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := s.db.ExecContext(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Status,
		now,
		now,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetUserByUsername 根据用户名获取用户
func (s *SQLiteStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE username = ?
	`
	row := s.db.QueryRowContext(ctx, query, username)

	var user model.User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByID 根据 ID 获取用户
func (s *SQLiteStore) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE id = ?
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var user model.User
	err := row.Scan(
		&user.ID,
		&user.Username,
		&user.PasswordHash,
		&user.Role,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateUser 更新用户信息
func (s *SQLiteStore) UpdateUser(ctx context.Context, user *model.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		UPDATE users
		SET username = ?, password_hash = ?, role = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Status,
		time.Now(),
		user.ID,
	)
	return err
}
