package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/datacollector/datacollector/internal/collector"
	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
)

// CollectorHandler 数据采集 API Handler
type CollectorHandler struct {
	store     storage.DataStore
	processor *collector.Processor
}

// NewCollectorHandler 创建新的采集处理器
func NewCollectorHandler(store storage.DataStore, processor *collector.Processor) *CollectorHandler {
	return &CollectorHandler{
		store:     store,
		processor: processor,
	}
}

// CollectData 处理单条数据提交
// POST /api/v1/collect/:source_id
func (h *CollectorHandler) CollectData(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. 从 X-Data-Token 头获取 token
	token := c.GetHeader("X-Data-Token")
	if token == "" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 2. 计算 SHA-256 哈希
	tokenHash := hashToken(token)

	// 3. 通过 store.GetTokenByHash 查找 token 记录
	tokenRecord, err := h.store.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 4. 验证 token
	// status=0：返回 403, CodeTokenDisabled
	if tokenRecord.Status == 0 {
		model.SendError(c, http.StatusForbidden, model.CodeTokenDisabled, "")
		return
	}

	// 已过期（expires_at 不为 nil 且已过期）：返回 401, CodeInvalidToken
	if tokenRecord.ExpiresAt != nil && time.Now().After(*tokenRecord.ExpiresAt) {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// source_id 不匹配 URL 参数：返回 401, CodeInvalidToken
	sourceIDParam := c.Param("source_id")
	sourceID, err := strconv.ParseInt(sourceIDParam, 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source_id")
		return
	}

	if tokenRecord.SourceID != sourceID {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 5. 更新 token 的最后使用时间
	if err := h.store.UpdateTokenLastUsed(ctx, tokenRecord.ID); err != nil {
		// 记录日志但不中断流程
		// log.Printf("failed to update token last used: %v", err)
	}

	// 6. 获取数据源配置
	source, err := h.store.GetSourceByID(ctx, sourceID)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return
	}

	// 7. 解析请求体为 map[string]interface{}
	var data map[string]interface{}
	if err := c.ShouldBindJSON(&data); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid request body")
		return
	}

	// 8. 解析数据源的 schema_config 为 model.SchemaConfig
	var schemaConfig model.SchemaConfig
	if len(source.SchemaConfig) > 0 {
		if err := json.Unmarshal(source.SchemaConfig, &schemaConfig); err != nil {
			// schema 解析失败，使用空配置（允许自由格式数据）
			schemaConfig = model.SchemaConfig{}
		}
	}

	// 9. 调用 collector.ValidateData 验证数据
	validationErrors := collector.ValidateData(data, &schemaConfig)
	if validationErrors != nil {
		// 10. 验证失败：返回 400, CodeValidationFailed，errors 字段包含验证错误
		model.SendValidationError(c, validationErrors)
		return
	}

	// 11. 构建 DataRecord（包含 IP、User-Agent）
	dataJSON, err := json.Marshal(data)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to marshal data")
		return
	}

	record := &model.DataRecord{
		SourceID:  sourceID,
		TokenID:   tokenRecord.ID,
		Data:      dataJSON,
		IPAddress: c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}

	// 12. 调用 processor.ProcessRecord 持久化
	recordID, err := h.processor.ProcessRecord(ctx, record)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to save record")
		return
	}

	// 13. 返回成功，data 包含 record_id
	model.SendSuccess(c, gin.H{
		"record_id": recordID,
	})
}

// CollectBatchData 处理批量数据提交
// POST /api/v1/collect/:source_id/batch
func (h *CollectorHandler) CollectBatchData(c *gin.Context) {
	ctx := c.Request.Context()

	// 1-6 步骤与单条提交相同
	// 1. 从 X-Data-Token 头获取 token
	token := c.GetHeader("X-Data-Token")
	if token == "" {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 2. 计算 SHA-256 哈希
	tokenHash := hashToken(token)

	// 3. 通过 store.GetTokenByHash 查找 token 记录
	tokenRecord, err := h.store.GetTokenByHash(ctx, tokenHash)
	if err != nil {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 4. 验证 token
	if tokenRecord.Status == 0 {
		model.SendError(c, http.StatusForbidden, model.CodeTokenDisabled, "")
		return
	}

	if tokenRecord.ExpiresAt != nil && time.Now().After(*tokenRecord.ExpiresAt) {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	sourceIDParam := c.Param("source_id")
	sourceID, err := strconv.ParseInt(sourceIDParam, 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source_id")
		return
	}

	if tokenRecord.SourceID != sourceID {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidToken, "")
		return
	}

	// 5. 更新 token 的最后使用时间
	if err := h.store.UpdateTokenLastUsed(ctx, tokenRecord.ID); err != nil {
		// 记录日志但不中断流程
	}

	// 6. 获取数据源配置
	source, err := h.store.GetSourceByID(ctx, sourceID)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return
	}

	// 7. 解析请求体：{"records": [...]}
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

	// 8. 解析数据源的 schema_config
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
		// 验证数据
		errors := collector.ValidateData(data, &schemaConfig)
		if errors != nil {
			validationErrors[i] = errors
			continue
		}

		// 构建记录
		dataJSON, err := json.Marshal(data)
		if err != nil {
			validationErrors[i] = map[string]string{"_error": "failed to marshal data"}
			continue
		}

		record := &model.DataRecord{
			SourceID:  sourceID,
			TokenID:   tokenRecord.ID,
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
	succeeded, failed, recordIDs, err := h.processor.ProcessBatch(ctx, records)
	if err != nil && succeeded == 0 {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, "failed to process batch")
		return
	}

	// 过滤出有效的 record IDs（排除 0 值）
	validRecordIDs := make([]int64, 0, len(recordIDs))
	for _, id := range recordIDs {
		if id > 0 {
			validRecordIDs = append(validRecordIDs, id)
		}
	}

	// 返回结果
	model.SendSuccess(c, gin.H{
		"total":      len(batchRequest.Records),
		"succeeded":  succeeded,
		"failed":     failed,
		"record_ids": validRecordIDs,
	})
}

// RegisterCollectorRoutes 注册采集路由
func (h *CollectorHandler) RegisterRoutes(r *gin.RouterGroup) {
	collect := r.Group("/collect")
	{
		collect.POST("/:source_id", h.CollectData)
		collect.POST("/:source_id/batch", h.CollectBatchData)
	}
}

// hashToken 计算 token 的 SHA-256 哈希
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
