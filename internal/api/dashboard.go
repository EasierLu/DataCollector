package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// DashboardHandler 仪表盘处理器
type DashboardHandler struct {
	store storage.DataStore
}

// NewDashboardHandler 创建新的仪表盘处理器
func NewDashboardHandler(store storage.DataStore) *DashboardHandler {
	return &DashboardHandler{
		store: store,
	}
}

// DashboardResponse 仪表盘数据响应
type DashboardResponse struct {
	TodayCount    int64                `json:"today_count"`
	WeekCount     int64                `json:"week_count"`
	MonthCount    int64                `json:"month_count"`
	TotalSources  int64                `json:"total_sources"`
	RecentRecords []*model.DataRecord  `json:"recent_records"`
}

// GetDashboard 获取仪表盘数据
// GET /api/v1/admin/dashboard
func (h *DashboardHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()

	// 计算今日数据量
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todayEnd := todayStart.AddDate(0, 0, 1).Add(-time.Second)
	todayCount, err := h.store.GetTotalCountByDateRange(ctx, todayStart.Format("2006-01-02"), todayEnd.Format("2006-01-02"))
	if err != nil {
		todayCount = 0
	}

	// 计算本周数据量（周一到今天）
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, -(weekday - 1))
	weekEnd := now
	weekCount, err := h.store.GetTotalCountByDateRange(ctx, weekStart.Format("2006-01-02"), weekEnd.Format("2006-01-02"))
	if err != nil {
		weekCount = 0
	}

	// 计算本月数据量（1号到今天）
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthEnd := now
	monthCount, err := h.store.GetTotalCountByDateRange(ctx, monthStart.Format("2006-01-02"), monthEnd.Format("2006-01-02"))
	if err != nil {
		monthCount = 0
	}

	// 获取数据源总数
	sourcesResult, err := h.store.ListSources(ctx, 1, 1)
	var totalSources int64
	if err == nil && sourcesResult != nil {
		totalSources = sourcesResult.Total
	}

	// 获取最近的数据记录
	recentFilter := model.RecordFilter{
		Page: 1,
		Size: 10,
	}
	recentResult, err := h.store.QueryRecords(ctx, recentFilter)
	var recentRecords []*model.DataRecord
	if err == nil && recentResult != nil {
		if list, ok := recentResult.List.([]*model.DataRecord); ok {
			recentRecords = list
		}
	}

	model.SendSuccess(c, DashboardResponse{
		TodayCount:    todayCount,
		WeekCount:     weekCount,
		MonthCount:    monthCount,
		TotalSources:  totalSources,
		RecentRecords: recentRecords,
	})
}

// GetDashboardTrend 获取仪表盘趋势数据
// GET /api/v1/admin/dashboard/trend
func (h *DashboardHandler) GetDashboardTrend(c *gin.Context) {
	ctx := c.Request.Context()

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	if startDate == "" || endDate == "" {
		model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, "start_date 和 end_date 为必填参数")
		return
	}

	var sourceID, tokenID int64
	if v := c.Query("source_id"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, "source_id 参数无效")
			return
		}
		sourceID = parsed
	}
	if v := c.Query("token_id"); v != "" {
		parsed, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, "token_id 参数无效")
			return
		}
		tokenID = parsed
	}

	points, err := h.store.GetDailyTrend(ctx, startDate, endDate, sourceID, tokenID)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "")
		return
	}

	if points == nil {
		points = make([]*model.TrendPoint, 0)
	}

	model.SendSuccess(c, points)
}
