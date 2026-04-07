package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateSource 创建数据源
func (s *SQLiteStore) CreateSource(ctx context.Context, source *model.DataSource) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO data_sources (name, description, schema_config, status, created_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	now := time.Now()
	result, err := s.db.ExecContext(ctx, query,
		source.Name,
		source.Description,
		source.SchemaConfig,
		source.Status,
		source.CreatedBy,
		now,
		now,
	)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

// GetSourceByID 根据 ID 获取数据源
func (s *SQLiteStore) GetSourceByID(ctx context.Context, id int64) (*model.DataSource, error) {
	query := `
		SELECT id, name, description, schema_config, status, created_by, created_at, updated_at
		FROM data_sources
		WHERE id = ? AND status = 1
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var source model.DataSource
	err := row.Scan(
		&source.ID,
		&source.Name,
		&source.Description,
		&source.SchemaConfig,
		&source.Status,
		&source.CreatedBy,
		&source.CreatedAt,
		&source.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &source, nil
}

// ListSources 分页查询数据源列表
func (s *SQLiteStore) ListSources(ctx context.Context, page, size int) (*model.PageResult, error) {
	// 处理默认值
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}

	// 查询总数
	var total int64
	countQuery := `SELECT COUNT(*) FROM data_sources WHERE status = 1`
	if err := s.db.QueryRowContext(ctx, countQuery).Scan(&total); err != nil {
		return nil, err
	}

	// 查询列表（包含 token_count）
	query := `
		SELECT s.id, s.name, s.description, s.schema_config, s.status, s.created_by, 
		       s.created_at, s.updated_at, COUNT(t.id) as token_count
		FROM data_sources s
		LEFT JOIN data_tokens t ON s.id = t.source_id AND t.status = 1
		WHERE s.status = 1
		GROUP BY s.id
		ORDER BY s.created_at DESC
		LIMIT ? OFFSET ?
	`
	offset := (page - 1) * size
	rows, err := s.db.QueryContext(ctx, query, size, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*model.DataSource
	for rows.Next() {
		var source model.DataSource
		err := rows.Scan(
			&source.ID,
			&source.Name,
			&source.Description,
			&source.SchemaConfig,
			&source.Status,
			&source.CreatedBy,
			&source.CreatedAt,
			&source.UpdatedAt,
			&source.TokenCount,
		)
		if err != nil {
			return nil, err
		}
		sources = append(sources, &source)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.PageResult{
		Total: total,
		List:  sources,
	}, nil
}

// UpdateSource 更新数据源
func (s *SQLiteStore) UpdateSource(ctx context.Context, source *model.DataSource) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		UPDATE data_sources
		SET name = ?, description = ?, schema_config = ?, status = ?, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query,
		source.Name,
		source.Description,
		source.SchemaConfig,
		source.Status,
		time.Now(),
		source.ID,
	)
	return err
}

// DeleteSource 软删除数据源
func (s *SQLiteStore) DeleteSource(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		UPDATE data_sources
		SET status = 0, updated_at = ?
		WHERE id = ?
	`
	_, err := s.db.ExecContext(ctx, query, time.Now(), id)
	return err
}
