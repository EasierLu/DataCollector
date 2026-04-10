package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// QueryHandler 数据查询 API Handler（供外部通过 API Key 调用）
type QueryHandler struct {
	store storage.DataStore
}

// NewQueryHandler 创建新的查询处理器
func NewQueryHandler(store storage.DataStore) *QueryHandler {
	return &QueryHandler{
		store: store,
	}
}

// BatchQueryRequest 批量查询请求
type BatchQueryRequest struct {
	StartID int64 `json:"start_id" binding:"required"`
	EndID   int64 `json:"end_id" binding:"required"`
}

// APIKeyAuthMiddleware API Key 鉴权中间件
// 从 X-API-Key 请求头获取独立 API Key，验证其有效性和权限
// requiredPerm 指定该中间件要求的权限
func APIKeyAuthMiddleware(store storage.DataStore, requiredPerm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		if apiKey == "" {
			model.SendError(c, http.StatusUnauthorized, model.CodeInvalidAPIKey, "缺少 X-API-Key 请求头")
			c.Abort()
			return
		}

		ctx := c.Request.Context()

		// 计算 API Key 哈希并查找（兼容新旧哈希）
		keyHash := hmacSHA256(apiKey)
		keyRecord, err := store.GetApiKeyByHash(ctx, keyHash)
		if err != nil {
			// 回退到 plain SHA-256 兼容旧 API Key
			legacyHash := plainSHA256(apiKey)
			keyRecord, err = store.GetApiKeyByHash(ctx, legacyHash)
			if err != nil {
				model.SendError(c, http.StatusUnauthorized, model.CodeInvalidAPIKey, "")
				c.Abort()
				return
			}
		}

		// 检查过期
		if keyRecord.ExpiresAt != nil && time.Now().After(*keyRecord.ExpiresAt) {
			model.SendError(c, http.StatusUnauthorized, model.CodeInvalidAPIKey, "API Key 已过期")
			c.Abort()
			return
		}

		// 检查权限
		if !keyRecord.HasPermission(requiredPerm) {
			model.SendError(c, http.StatusForbidden, model.CodeInvalidAPIKey, "该 API Key 无 "+requiredPerm+" 权限")
			c.Abort()
			return
		}

		// 将 API Key ID 存入 context
		c.Set("api_key_id", keyRecord.ID)

		// 更新最后使用时间
		_ = store.UpdateApiKeyLastUsed(ctx, keyRecord.ID)

		c.Next()
	}
}

// validateQueryCollectID 验证 collect_id 并获取对应的数据源
func (h *QueryHandler) validateQueryCollectID(ctx context.Context, c *gin.Context) (*model.DataSource, error) {
	collectID := c.Param("collect_id")
	if collectID == "" {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "缺少 collect_id")
		return nil, errRequestHandled
	}

	source, err := h.store.GetSourceByCollectID(ctx, collectID)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return nil, err
	}

	return source, nil
}

// GetLastRecord 查询最后一条数据
// GET /api/v1/query/:collect_id/last
func (h *QueryHandler) GetLastRecord(c *gin.Context) {
	ctx := c.Request.Context()

	source, err := h.validateQueryCollectID(ctx, c)
	if err != nil {
		return
	}

	record, err := h.store.GetLastRecordBySourceID(ctx, source.ID)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "查询失败")
		return
	}

	if record == nil {
		model.SendError(c, http.StatusNotFound, model.CodeRecordNotFound, "")
		return
	}

	model.SendSuccess(c, record)
}

// GetRecord 查询单条数据
// GET /api/v1/query/:collect_id/record/:record_id
func (h *QueryHandler) GetRecord(c *gin.Context) {
	ctx := c.Request.Context()

	source, err := h.validateQueryCollectID(ctx, c)
	if err != nil {
		return
	}

	recordID, err := strconv.ParseInt(c.Param("record_id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "无效的 record_id")
		return
	}

	record, err := h.store.GetRecordByID(ctx, recordID)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "查询失败")
		return
	}

	if record == nil || record.SourceID != source.ID {
		model.SendError(c, http.StatusNotFound, model.CodeRecordNotFound, "")
		return
	}

	model.SendSuccess(c, record)
}

// BatchQueryRecords 批量查询数据（按 ID 范围）
// POST /api/v1/query/:collect_id/records
func (h *QueryHandler) BatchQueryRecords(c *gin.Context) {
	ctx := c.Request.Context()

	source, err := h.validateQueryCollectID(ctx, c)
	if err != nil {
		return
	}

	var req BatchQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "请提供 start_id 和 end_id")
		return
	}

	if req.StartID > req.EndID {
		model.SendError(c, http.StatusBadRequest, model.CodeInvalidIDRange, "start_id 不能大于 end_id")
		return
	}

	// 限制单次查询范围，防止一次查询过多数据
	const maxRange = 1000
	if req.EndID-req.StartID+1 > maxRange {
		model.SendError(c, http.StatusBadRequest, model.CodeInvalidIDRange, "单次查询范围不能超过 1000 条")
		return
	}

	records, err := h.store.GetRecordsByIDRange(ctx, source.ID, req.StartID, req.EndID)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "查询失败")
		return
	}

	if records == nil {
		records = make([]*model.DataRecord, 0)
	}

	model.SendSuccess(c, gin.H{
		"total":   len(records),
		"records": records,
	})
}

// errRequestHandled is reused from collector.go (same package)
var _ = errors.New // ensure import
