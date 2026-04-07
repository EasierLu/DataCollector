package model

// 错误码常量定义
const (
	CodeSuccess = 0

	// 数据采集 1000-1099
	CodeInvalidToken      = 1000
	CodeTokenDisabled     = 1001
	CodeValidationFailed  = 1002
	CodeRateLimitExceeded = 1003

	// 用户认证 2000-2099
	CodeLoginFailed        = 2000
	CodeTokenExpired       = 2001
	CodePermissionDenied   = 2002
	CodeInvalidJWT         = 2003
	CodeOldPasswordWrong   = 2004
	CodePasswordChangeFail = 2005

	// 数据源管理 3000-3099
	CodeSourceNotFound     = 3000
	CodeSourceCreateFailed = 3001
	CodeSourceUpdateFailed = 3002
	CodeSourceDeleteFailed = 3003

	// 数据查询 4000-4099
	CodeQueryParamError = 4000
	CodeExportFailed    = 4001

	// 系统运维 5000-5099
	CodeSystemUnhealthy    = 5000
	CodeInitFailed         = 5001
	CodeAlreadyInitialized = 5002

	// 通用错误 9000-9099
	CodeParamMissing  = 9000
	CodeInternalError = 9001
	CodeUnknownError  = 9002
)

// ErrorMessages 错误码对应的默认消息
var ErrorMessages = map[int]string{
	CodeSuccess: "成功",

	// 数据采集
	CodeInvalidToken:      "无效的Token",
	CodeTokenDisabled:     "Token已禁用",
	CodeValidationFailed:  "数据验证失败",
	CodeRateLimitExceeded: "请求频率超限",

	// 用户认证
	CodeLoginFailed:        "登录失败",
	CodeTokenExpired:       "Token已过期",
	CodePermissionDenied:   "权限不足",
	CodeInvalidJWT:         "无效的JWT",
	CodeOldPasswordWrong:   "旧密码错误",
	CodePasswordChangeFail: "修改密码失败",

	// 数据源管理
	CodeSourceNotFound:     "数据源不存在",
	CodeSourceCreateFailed: "创建数据源失败",
	CodeSourceUpdateFailed: "更新数据源失败",
	CodeSourceDeleteFailed: "删除数据源失败",

	// 数据查询
	CodeQueryParamError: "查询参数错误",
	CodeExportFailed:    "导出失败",

	// 系统运维
	CodeSystemUnhealthy:    "系统状态异常",
	CodeInitFailed:         "系统初始化失败",
	CodeAlreadyInitialized: "系统已初始化",

	// 通用错误
	CodeParamMissing:  "缺少必要参数",
	CodeInternalError: "内部错误",
	CodeUnknownError:  "未知错误",
}

// GetErrorMessage 获取错误码对应的默认消息
func GetErrorMessage(code int) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "未知错误"
}
