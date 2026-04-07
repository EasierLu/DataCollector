package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// SourceHandler 数据源处理器
type SourceHandler struct {
	store storage.DataStore
}

// NewSourceHandler 创建新的数据源处理器
func NewSourceHandler(store storage.DataStore) *SourceHandler {
	return &SourceHandler{
		store: store,
	}
}

// CreateSourceRequest 创建数据源请求
type CreateSourceRequest struct {
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	SchemaConfig json.RawMessage `json:"schema_config"`
}

// UpdateSourceRequest 更新数据源请求
type UpdateSourceRequest struct {
	Name         string          `json:"name" binding:"required"`
	Description  string          `json:"description"`
	SchemaConfig json.RawMessage `json:"schema_config"`
}

// ListSources 获取数据源列表
// GET /api/v1/admin/sources
func (h *SourceHandler) ListSources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}

	result, err := h.store.ListSources(c.Request.Context(), page, size)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, result)
}

// CreateSource 创建数据源
// POST /api/v1/admin/sources
func (h *SourceHandler) CreateSource(c *gin.Context) {
	var req CreateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 从 context 获取 user_id
	userID, exists := c.Get("user_id")
	if !exists {
		model.SendError(c, http.StatusUnauthorized, model.CodeInvalidJWT, "user_id not found in context")
		return
	}

	// 处理 schema_config
	var schemaConfig json.RawMessage
	if req.SchemaConfig != nil && len(req.SchemaConfig) > 0 {
		schemaConfig = req.SchemaConfig
	} else {
		schemaConfig = json.RawMessage("{}")
	}

	source := &model.DataSource{
		Name:         req.Name,
		Description:  req.Description,
		SchemaConfig: schemaConfig,
		Status:       1,
		CreatedBy:    userID.(int64),
	}

	id, err := h.store.CreateSource(c.Request.Context(), source)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeSourceCreateFailed, err.Error())
		return
	}

	source.ID = id
	model.SendSuccess(c, source)
}

// UpdateSource 更新数据源
// PUT /api/v1/admin/sources/:id
func (h *SourceHandler) UpdateSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source id")
		return
	}

	var req UpdateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	// 检查数据源是否存在
	existing, err := h.store.GetSourceByID(c.Request.Context(), id)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return
	}

	// 处理 schema_config
	var schemaConfig json.RawMessage
	if req.SchemaConfig != nil && len(req.SchemaConfig) > 0 {
		schemaConfig = req.SchemaConfig
	} else {
		schemaConfig = json.RawMessage("{}")
	}

	// 更新字段
	existing.Name = req.Name
	existing.Description = req.Description
	existing.SchemaConfig = schemaConfig

	if err := h.store.UpdateSource(c.Request.Context(), existing); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeSourceUpdateFailed, err.Error())
		return
	}

	model.SendSuccess(c, existing)
}

// DeleteSource 删除数据源（软删除）
// DELETE /api/v1/admin/sources/:id
func (h *SourceHandler) DeleteSource(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid source id")
		return
	}

	// 检查数据源是否存在
	_, err = h.store.GetSourceByID(c.Request.Context(), id)
	if err != nil {
		model.SendError(c, http.StatusNotFound, model.CodeSourceNotFound, "")
		return
	}

	if err := h.store.DeleteSource(c.Request.Context(), id); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeSourceDeleteFailed, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "source deleted successfully"})
}
