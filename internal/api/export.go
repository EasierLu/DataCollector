package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/model"
	"github.com/datacollector/datacollector/internal/storage"
	"github.com/gin-gonic/gin"
)

// ExportHandler 数据导出处理器
type ExportHandler struct {
	store storage.DataStore
}

// NewExportHandler 创建新的数据导出处理器
func NewExportHandler(store storage.DataStore) *ExportHandler {
	return &ExportHandler{
		store: store,
	}
}

// ExportData 导出数据
// GET /api/v1/admin/data/export
func (h *ExportHandler) ExportData(c *gin.Context) {
	// 绑定查询参数
	var filter model.RecordFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, err.Error())
		return
	}

	// 获取导出格式
	format := c.DefaultQuery("format", "csv")
	if format != "csv" && format != "json" {
		model.SendError(c, http.StatusBadRequest, model.CodeQueryParamError, "format must be csv or json")
		return
	}

	const maxExportRows = 100000
	filter.ExportLimit = maxExportRows

	// 获取所有匹配记录（受限）
	records, err := h.store.ExportRecords(c.Request.Context(), filter)
	if err != nil {
		model.SendError(c, http.StatusInternalServerError, model.CodeExportFailed, err.Error())
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("export_%s.%s", time.Now().Format("20060102"), format)

	switch format {
	case "csv":
		h.exportCSV(c, records, filename)
	case "json":
		h.exportJSON(c, records, filename)
	}
}

// exportCSV 导出为 CSV 格式
func (h *ExportHandler) exportCSV(c *gin.Context, records []*model.DataRecord, filename string) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	writer := csv.NewWriter(c.Writer)
	defer writer.Flush()

	// 写入表头
	headers := []string{"id", "source_id", "data", "ip_address", "user_agent", "created_at"}
	if err := writer.Write(headers); err != nil {
		return
	}

	// 写入数据行
	for _, record := range records {
		// 将 data 转换为 JSON 字符串
		dataStr := ""
		if record.Data != nil {
			dataStr = string(record.Data)
		}

		row := []string{
			strconv.FormatInt(record.ID, 10),
			strconv.FormatInt(record.SourceID, 10),
			dataStr,
			record.IPAddress,
			record.UserAgent,
			record.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return
		}
	}
}

// exportJSON 导出为 JSON 格式
func (h *ExportHandler) exportJSON(c *gin.Context, records []*model.DataRecord, filename string) {
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))

	// 直接序列化记录数组
	encoder := json.NewEncoder(c.Writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(records); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode JSON"})
	}
}
