package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateUser 创建用户
func (s *PostgresStore) CreateUser(ctx context.Context, user *model.User) (int64, error) {
	query := `
		INSERT INTO users (username, password_hash, role, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	now := time.Now()
	var id int64
	err := s.db.QueryRowContext(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Role,
		user.Status,
		now,
		now,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE username = $1
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
func (s *PostgresStore) GetUserByID(ctx context.Context, id int64) (*model.User, error) {
	query := `
		SELECT id, username, password_hash, role, status, created_at, updated_at
		FROM users
		WHERE id = $1
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
func (s *PostgresStore) UpdateUser(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET username = $1, password_hash = $2, role = $3, status = $4, updated_at = $5
		WHERE id = $6
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
