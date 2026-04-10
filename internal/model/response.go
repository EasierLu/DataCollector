package model

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一API响应结构体
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

// Success 返回成功响应
func Success(data interface{}) Response {
	return Response{
		Code:    CodeSuccess,
		Message: GetErrorMessage(CodeSuccess),
		Data:    data,
	}
}

// SuccessWithMessage 返回带自定义消息的成功响应
func SuccessWithMessage(message string, data interface{}) Response {
	return Response{
		Code:    CodeSuccess,
		Message: message,
		Data:    data,
	}
}

// Error 返回错误响应
func Error(code int, message string) Response {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return Response{
		Code:    code,
		Message: message,
	}
}

// ErrorWithErrors 返回带详细错误信息的响应
func ErrorWithErrors(code int, message string, errors interface{}) Response {
	if message == "" {
		message = GetErrorMessage(code)
	}
	return Response{
		Code:    code,
		Message: message,
		Errors:  errors,
	}
}

// SendSuccess 发送成功响应（Gin上下文）
func SendSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Success(data))
}

// SendError 发送错误响应（Gin上下文）
func SendError(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Error(code, message))
}

// SendValidationError 发送验证错误响应（Gin上下文）
func SendValidationError(c *gin.Context, errors interface{}) {
	c.JSON(http.StatusBadRequest, ErrorWithErrors(CodeValidationFailed, "", errors))
}

// PagedData 统一分页响应数据
type PagedData struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
	Page  int         `json:"page"`
	Size  int         `json:"size"`
}

// SendPagedSuccess 发送分页成功响应
func SendPagedSuccess(c *gin.Context, items interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, Success(PagedData{
		Items: items,
		Total: total,
		Page:  page,
		Size:  size,
	}))
}
