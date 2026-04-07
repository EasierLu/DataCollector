package postgres

import (
	"context"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// IncrementStatCount 增加统计计数（UPSERT）
func (s *PostgresStore) IncrementStatCount(ctx context.Context, sourceID int64, date string) error {
	query := `
		INSERT INTO statistics (source_id, stat_date, count, created_at, updated_at)
		VALUES ($1, $2, 1, $3, $3)
		ON CONFLICT (source_id, stat_date) DO UPDATE SET
			count = statistics.count + 1,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, sourceID, date, now)
	return err
}

// GetStatsBySourceAndDateRange 获取指定数据源在日期范围内的统计
func (s *PostgresStore) GetStatsBySourceAndDateRange(ctx context.Context, sourceID int64, startDate, endDate string) ([]*model.Statistics, error) {
	query := `
		SELECT id, source_id, stat_date, count, created_at, updated_at
		FROM statistics
		WHERE source_id = $1 AND stat_date >= $2 AND stat_date <= $3
		ORDER BY stat_date ASC
	`
	rows, err := s.db.QueryContext(ctx, query, sourceID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*model.Statistics
	for rows.Next() {
		var stat model.Statistics
		err := rows.Scan(
			&stat.ID,
			&stat.SourceID,
			&stat.StatDate,
			&stat.Count,
			&stat.CreatedAt,
			&stat.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		stats = append(stats, &stat)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return stats, nil
}

// GetTotalCountByDateRange 获取日期范围内的总记录数
func (s *PostgresStore) GetTotalCountByDateRange(ctx context.Context, startDate, endDate string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(count), 0)
		FROM statistics
		WHERE stat_date >= $1 AND stat_date <= $2
	`
	var total int64
	err := s.db.QueryRowContext(ctx, query, startDate, endDate).Scan(&total)
	return total, err
}

// GetCountBySourceID 获取指定数据源的总记录数
func (s *PostgresStore) GetCountBySourceID(ctx context.Context, sourceID int64) (int64, error) {
	query := `
		SELECT COALESCE(SUM(count), 0)
		FROM statistics
		WHERE source_id = $1
	`
	var total int64
	err := s.db.QueryRowContext(ctx, query, sourceID).Scan(&total)
	return total, err
}
