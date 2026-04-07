package sqlite

import (
	"context"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// IncrementStatCount 增加统计计数（UPSERT）
func (s *SQLiteStore) IncrementStatCount(ctx context.Context, sourceID int64, date string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
		INSERT INTO statistics (source_id, stat_date, count, created_at, updated_at)
		VALUES (?, ?, 1, ?, ?)
		ON CONFLICT(source_id, stat_date) DO UPDATE SET
			count = count + 1,
			updated_at = excluded.updated_at
	`
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, sourceID, date, now, now)
	return err
}

// GetStatsBySourceAndDateRange 获取指定数据源在日期范围内的统计
func (s *SQLiteStore) GetStatsBySourceAndDateRange(ctx context.Context, sourceID int64, startDate, endDate string) ([]*model.Statistics, error) {
	query := `
		SELECT id, source_id, stat_date, count, created_at, updated_at
		FROM statistics
		WHERE source_id = ? AND stat_date >= ? AND stat_date <= ?
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
func (s *SQLiteStore) GetTotalCountByDateRange(ctx context.Context, startDate, endDate string) (int64, error) {
	query := `
		SELECT COALESCE(SUM(count), 0)
		FROM statistics
		WHERE stat_date >= ? AND stat_date <= ?
	`
	var total int64
	err := s.db.QueryRowContext(ctx, query, startDate, endDate).Scan(&total)
	return total, err
}

// GetCountBySourceID 获取指定数据源的总记录数
func (s *SQLiteStore) GetCountBySourceID(ctx context.Context, sourceID int64) (int64, error) {
	query := `
		SELECT COALESCE(SUM(count), 0)
		FROM statistics
		WHERE source_id = ?
	`
	var total int64
	err := s.db.QueryRowContext(ctx, query, sourceID).Scan(&total)
	return total, err
}
