package api

import (
	"net/http"
	"strconv"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// DataHandler 数据管理处理器
type DataHandler struct {
	store storage.DataStore
}

// NewDataHandler 创建新的数据管理处理器
func NewDataHandler(store storage.DataStore) *DataHandler {
	return &DataHandler{
		store: store,
	}
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []int64 `json:"ids" binding:"required"`
}

// QueryData 查询数据记录
// GET /api/v1/admin/data
func (h *DataHandler) QueryData(c *gin.Context) {
	var filter model.RecordFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, err.Error())
		return
	}

	// 设置默认值
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 || filter.Size > 100 {
		filter.Size = 20
	}

	result, err := h.store.QueryRecords(c.Request.Context(), filter)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, result)
}

// DeleteRecord 删除单条记录
// DELETE /api/v1/admin/data/:id
func (h *DataHandler) DeleteRecord(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "invalid record id")
		return
	}

	if err := h.store.DeleteRecord(c.Request.Context(), id); err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{"message": "record deleted successfully"})
}

// BatchDeleteRecords 批量删除记录
// POST /api/v1/admin/data/batch-delete
func (h *DataHandler) BatchDeleteRecords(c *gin.Context) {
	var req BatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, err.Error())
		return
	}

	if len(req.IDs) == 0 {
		model.SendError(c, http.StatusBadRequest, model.CodeParamMissing, "ids cannot be empty")
		return
	}

	count, err := h.store.DeleteRecordsByIDs(c.Request.Context(), req.IDs)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeInternalError, err.Error())
		return
	}

	model.SendSuccess(c, gin.H{
		"message": "records deleted successfully",
		"count":   count,
	})
}
