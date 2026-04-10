package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/middleware"
	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// CollectorHandler 数据采集 API Handler
type CollectorHandler struct {
	store       storage.DataStore
	processor   *collector.Processor
	rateLimiter *middleware.RateLimiter
}

// NewCollectorHandler 创建新的采集处理器
func NewCollectorHandler(store storage.DataStore, processor *collector.Processor, rateLimiter *middleware.RateLimiter) *CollectorHandler {
	return &CollectorHandler{
		store:       store,
		processor:   processor,
		rateLimiter: rateLimiter,
	}
}

// CollectData 处理单条数据提交
// POST /api/v1/collect/:collect_id
func (h *CollectorHandler) CollectData(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 从 X-Data-Token 头获取 token
	token := c.GetHeader("X-Data-Token")
	if token == "" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 2-4. 验证 Token 和数据源
	source, tokenRecord, err := h.validateCollectRequest(ctx, c, token)
	if err != nil {
		return
	}

	// 6. 解析请求体为 map[string]interface{}
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid request body")
		return
	}

	// 7. 解析数据源的 schema_config 为 model.SchemaConfig
	var schemaConfig model.SchemaConfig
	if len(source.SchemaConfig) > 0 {
		if err := json.Unmarshal(source.SchemaConfig, &schemaConfig); err != nil {
			schemaConfig = model.SchemaConfig{}
		}
	}

	// 8. 调用 collector.ValidateData 验证数据
	validationErrors := collector.ValidateData(data, &schemaConfig)
	if validationErrors != nil {
		model.SendValidationError(c, validationErrors)
		return
	}

	// 9. 构建 DataRecord（包含 IP、User-Agent）
	dataJSON, err := json.Marshal(data)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to marshal data")
		return
	}

	record := &model.DataRecord{
		SourceID:  source.ID,
		TokenID:   tokenRecord.ID,
		Data:      dataJSON,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	// 10. 调用 processor.ProcessRecord 持久化
	_, err = h.processor.ProcessRecord(ctx, source, record)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to save record")
		return
	}

	model.SendSuccess(c, gin.H{})
}

// CollectBatchData 处理批量数据提交
// POST /api/v1/collect/:collect_id/batch
func (h *CollectorHandler) CollectBatchData(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 从 X-Data-Token 头获取 token
	token := c.GetHeader("X-Data-Token")
	if token == "" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 2-4. 验证 Token 和数据源
	source, _, err := h.validateCollectRequest(ctx, c, token)
	if err != nil {
		return
	}

	// 5. 解析请求体：{"records": [...]}
	var batchRequest struct {
		Records []map[string]interface{} `json:"records" binding:"required"`
	}
	if err := c.ShouldBindJSON(&batchRequest); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid request body: records array required")
		return
	}

	if len(batchRequest.Records) == 0 {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "records array is empty")
		return
	}

	const maxBatchSize = 100
	if len(batchRequest.Records) > maxBatchSize {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "batch size exceeds maximum of 100 records")
		return
	}

	// 6. 解析数据源的 schema_config
	var schemaConfig model.SchemaConfig
	if len(source.SchemaConfig) > 0 {
		if err := json.Unmarshal(source.SchemaConfig, &schemaConfig); err != nil {
			schemaConfig = model.SchemaConfig{}
		}
	}

	// 逐条验证和处理
	records := make([]*model.DataRecord, 0, len(batchRequest.Records))
	validationErrors := make(map[int]map[string]string)

	for i, data := range batchRequest.Records {
		errors := collector.ValidateData(data, &schemaConfig)
		if errors != nil {
			validationErrors[i] = errors
			continue
		}

		dataJSON, err := json.Marshal(data)
		if err != nil {
			validationErrors[i] = map[string]string{"_error": "failed to marshal data"}
			continue
		}

		record := &model.DataRecord{
			SourceID:  source.ID,
			Data:      dataJSON,
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
		}
		records = append(records, record)
	}

	// 如果有验证错误，返回 400
	if len(validationErrors) > 0 {
		model.SendValidationError(c, validationErrors)
		return
	}

	// 9. 批量处理记录
	succeeded, failed, _, err := h.processor.ProcessBatch(ctx, source, records)
	if err != nil && succeeded == 0 {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to process batch")
		return
	}

	// 返回结果
	model.SendSuccess(c, gin.H{
		"total":     len(batchRequest.Records),
		"succeeded": succeeded,
		"failed":    failed,
	})
}

// validateCollectRequest validates the data token and source for a collect request.
// Returns the source, token record, or an error (the error response is already sent to the client).
func (h *CollectorHandler) validateCollectRequest(ctx context.Context, c *gin.Context, rawToken string) (*model.DataSource, *model.DataToken, error) {
	// 使用 HMAC-SHA256 哈希查找 Token
	tokenHash := hmacSHA256(rawToken)
	tokenRecord, err := h.store.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return nil, nil, err
	}

	if tokenRecord.Status == 0 {
		model.SendError(c, http.StatusForbidden, model.CodeTokenDisabled, "")
		return nil, nil, errRequestHandled
	}

	if tokenRecord.ExpiresAt != nil && time.Now().After(*tokenRecord.ExpiresAt) {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return nil, nil, errRequestHandled
	}

	collectID := c.Param("collect_id")
	if collectID == "" {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid collect_id")
		return nil, nil, errRequestHandled
	}

	source, err := h.store.GetSourceByCollectID(ctx, collectID)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return nil, nil, err
	}

	if tokenRecord.SourceID != source.ID {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return nil, nil, errRequestHandled
	}

	if h.rateLimiter != nil {
		rps, burst := h.getSourceRateLimit(ctx, source)
		if rps > 0 {
			key := "token:" + rawToken
			if !h.rateLimiter.Allow(key, rps, burst) {
				model.SendError(c, http.StatusTooManyRequests, model.CodeRateLimitExceeded, "请求频率超限，请稍后再试")
				return nil, nil, errRequestHandled
			}
		}
	}

	_ = h.store.UpdateTokenLastUsed(ctx, tokenRecord.ID)

	return source, tokenRecord, nil
}

// errRequestHandled is a sentinel indicating the HTTP response was already sent.
var errRequestHandled = errors.New("request already handled")

// getSourceRateLimit 获取数据源的限流参数（每秒请求数和突发量）
// 优先使用数据源级别配置，若未配置则回退全局默认值
func (h *CollectorHandler) getSourceRateLimit(ctx context.Context, source *model.DataSource) (rps float64, burst int) {
	limitPerMin := source.RateLimit
	burst = source.RateLimitBurst

	// 回退全局默认值
	if limitPerMin <= 0 || burst <= 0 {
		globalSettings := LoadRateLimitSettings(ctx, h.store)
		if limitPerMin <= 0 {
			limitPerMin = globalSettings.RateLimitPerToken
		}
		if burst <= 0 {
			burst = globalSettings.RateLimitPerTokenBurst
		}
	}

	// 将每分钟请求数转换为每秒请求数
	rps = float64(limitPerMin) / 60.0
	return
}
