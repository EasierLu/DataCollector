package api

import (
	"net/http"
	"strconv"
	"sync"
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

	todayStr := now.Format("2006-01-02")
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	weekStartStr := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")
	monthStartStr := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	var (
		todayCount    int64
		weekCount     int64
		monthCount    int64
		totalSources  int64
		recentRecords []*model.DataRecord
		wg            sync.WaitGroup
	)

	wg.Add(5)

	go func() {
		defer wg.Done()
		if v, err := h.store.GetTotalCountByDateRange(ctx, todayStr, todayStr); err == nil {
			todayCount = v
		}
	}()

	go func() {
		defer wg.Done()
		if v, err := h.store.GetTotalCountByDateRange(ctx, weekStartStr, todayStr); err == nil {
			weekCount = v
		}
	}()

	go func() {
		defer wg.Done()
		if v, err := h.store.GetTotalCountByDateRange(ctx, monthStartStr, todayStr); err == nil {
			monthCount = v
		}
	}()

	go func() {
		defer wg.Done()
		if result, err := h.store.ListSources(ctx, 1, 1); err == nil && result != nil {
			totalSources = result.Total
		}
	}()

	go func() {
		defer wg.Done()
		recentFilter := model.RecordFilter{Page: 1, Size: 10}
		if result, err := h.store.QueryRecords(ctx, recentFilter); err == nil && result != nil {
			if list, ok := result.List.([]*model.DataRecord); ok {
				recentRecords = list
			}
		}
	}()

	wg.Wait()

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
