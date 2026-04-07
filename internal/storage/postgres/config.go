package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// GetConfig 获取配置值
func (s *PostgresStore) GetConfig(ctx context.Context, key string) (string, error) {
	query := `SELECT config_value FROM system_configs WHERE config_key = $1`
	row := s.db.QueryRowContext(ctx, query, key)

	var value string
	err := row.Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

// SetConfig 设置配置值（UPSERT）
func (s *PostgresStore) SetConfig(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO system_configs (config_key, config_value, created_at, updated_at)
		VALUES ($1, $2, $3, $3)
		ON CONFLICT (config_key) DO UPDATE SET
			config_value = EXCLUDED.config_value,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, key, value, now)
	return err
}

// GetAllConfigs 获取所有配置
func (s *PostgresStore) GetAllConfigs(ctx context.Context) ([]*model.SystemConfig, error) {
	query := `
		SELECT id, config_key, config_value, created_at, updated_at
		FROM system_configs
		ORDER BY config_key ASC
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*model.SystemConfig
	for rows.Next() {
		var cfg model.SystemConfig
		err := rows.Scan(
			&cfg.ID,
			&cfg.ConfigKey,
			&cfg.ConfigValue,
			&cfg.CreatedAt,
			&cfg.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		configs = append(configs, &cfg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return configs, nil
}
