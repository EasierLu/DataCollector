package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// CreateRecord 创建数据记录
func (s *PostgresStore) CreateRecord(ctx context.Context, record *model.DataRecord) (int64, error) {
	query := `
		INSERT INTO data_records (source_id, token_id, data, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int64
	err := s.db.QueryRowContext(ctx, query,
		record.SourceID,
		record.TokenID,
		record.Data,
		record.IPAddress,
		record.UserAgent,
		time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// GetRecordByID 根据 ID 获取数据记录
func (s *PostgresStore) GetRecordByID(ctx context.Context, id int64) (*model.DataRecord, error) {
	query := `
		SELECT id, source_id, token_id, data, ip_address, user_agent, created_at
		FROM data_records
		WHERE id = $1
	`
	row := s.db.QueryRowContext(ctx, query, id)

	var record model.DataRecord
	err := row.Scan(
		&record.ID,
		&record.SourceID,
		&record.TokenID,
		&record.Data,
		&record.IPAddress,
		&record.UserAgent,
		&record.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &record, nil
}

// QueryRecords 分页查询数据记录
func (s *PostgresStore) QueryRecords(ctx context.Context, filter model.RecordFilter) (*model.PageResult, error) {
	// 处理默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 {
		filter.Size = 20
	}

	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.SourceID > 0 {
		conditions = append(conditions, fmt.Sprintf("source_id = $%d", argIdx))
		args = append(args, filter.SourceID)
		argIdx++
	}
	if filter.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("DATE(created_at) >= $%d", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("DATE(created_at) <= $%d", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// 查询总数
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM data_records %s", whereClause)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	// 查询列表
	limitIdx := argIdx
	offsetIdx := argIdx + 1
	query := fmt.Sprintf(`
		SELECT id, source_id, token_id, data, ip_address, user_agent, created_at
		FROM data_records
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, limitIdx, offsetIdx)

	limitArgs := append(args, filter.Size, (filter.Page-1)*filter.Size)
	rows, err := s.db.QueryContext(ctx, query, limitArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.DataRecord
	for rows.Next() {
		var record model.DataRecord
		err := rows.Scan(
			&record.ID,
			&record.SourceID,
			&record.TokenID,
			&record.Data,
			&record.IPAddress,
			&record.UserAgent,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &model.PageResult{
		Total: total,
		List:  records,
	}, nil
}

// DeleteRecord 删除单条记录
func (s *PostgresStore) DeleteRecord(ctx context.Context, id int64) error {
	query := `DELETE FROM data_records WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, id)
	return err
}

// DeleteRecordsByIDs 批量删除记录
func (s *PostgresStore) DeleteRecordsByIDs(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	// 构建占位符
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf("DELETE FROM data_records WHERE id IN (%s)", strings.Join(placeholders, ","))
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// ExportRecords 导出记录（不分页）
func (s *PostgresStore) ExportRecords(ctx context.Context, filter model.RecordFilter) ([]*model.DataRecord, error) {
	// 构建查询条件
	var conditions []string
	var args []interface{}
	argIdx := 1

	if filter.SourceID > 0 {
		conditions = append(conditions, fmt.Sprintf("source_id = $%d", argIdx))
		args = append(args, filter.SourceID)
		argIdx++
	}
	if filter.StartDate != "" {
		conditions = append(conditions, fmt.Sprintf("DATE(created_at) >= $%d", argIdx))
		args = append(args, filter.StartDate)
		argIdx++
	}
	if filter.EndDate != "" {
		conditions = append(conditions, fmt.Sprintf("DATE(created_at) <= $%d", argIdx))
		args = append(args, filter.EndDate)
		argIdx++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limitClause := ""
	if filter.ExportLimit > 0 {
		limitClause = fmt.Sprintf("LIMIT %d", filter.ExportLimit)
	}

	query := fmt.Sprintf(`
		SELECT id, source_id, token_id, data, ip_address, user_agent, created_at
		FROM data_records
		%s
		ORDER BY created_at DESC
		%s
	`, whereClause, limitClause)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.DataRecord
	for rows.Next() {
		var record model.DataRecord
		err := rows.Scan(
			&record.ID,
			&record.SourceID,
			&record.TokenID,
			&record.Data,
			&record.IPAddress,
			&record.UserAgent,
			&record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		records = append(records, &record)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}
