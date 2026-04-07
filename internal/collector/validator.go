package collector

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// 预编译正则表达式
var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateData 根据 SchemaConfig 验证提交的数据
// data: 用户提交的数据 map
// schema: 数据源的字段定义
// 返回: 验证错误 map[field_name]error_message，如果验证通过返回 nil
func ValidateData(data map[string]interface{}, schema *model.SchemaConfig) map[string]string {
	// 如果 schema 为 nil 或没有 fields，则跳过验证
	if schema == nil || len(schema.Fields) == 0 {
		return nil
	}

	errors := make(map[string]string)

	for _, field := range schema.Fields {
		value, exists := data[field.Name]

		// 检查必填字段
		if field.Required {
			if !exists || value == nil || isEmptyValue(value) {
				errors[field.Name] = "required"
				continue
			}
		}

		// 如果字段不存在且不是必填的，跳过验证
		if !exists || value == nil {
			continue
		}

		// 类型验证
		if field.Type != "" {
			if errMsg := validateType(value, field.Type); errMsg != "" {
				errors[field.Name] = errMsg
				continue
			}
		}

		// 字符串特有验证
		if strValue, ok := value.(string); ok {
			// 最大长度验证
			if field.MaxLength > 0 && len(strValue) > field.MaxLength {
				errors[field.Name] = fmt.Sprintf("max_length exceeded (max: %d)", field.MaxLength)
				continue
			}

			// 最小长度验证
			if field.MinLength > 0 && len(strValue) < field.MinLength {
				errors[field.Name] = fmt.Sprintf("min_length not met (min: %d)", field.MinLength)
				continue
			}

			// 正则匹配验证
			if field.Pattern != "" {
				matched, err := regexp.MatchString(field.Pattern, strValue)
				if err != nil || !matched {
					errors[field.Name] = "pattern mismatch"
					continue
				}
			}
		}
	}

	if len(errors) > 0 {
		return errors
	}
	return nil
}

// isEmptyValue 检查值是否为空
func isEmptyValue(value interface{}) bool {
	switch v := value.(type) {
	case nil:
		return true
	case string:
		return v == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}

// validateType 验证值是否符合指定类型
func validateType(value interface{}, fieldType string) string {
	switch fieldType {
	case "string":
		if _, ok := value.(string); !ok {
			return "type must be string"
		}

	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return ""
		case uint, uint8, uint16, uint32, uint64:
			return ""
		case float32, float64:
			return ""
		case json.Number:
			return ""
		default:
			return "type must be number"
		}

	case "email":
		str, ok := value.(string)
		if !ok {
			return "type must be string"
		}
		if !emailRegex.MatchString(str) {
			return "invalid email format"
		}

	case "url":
		str, ok := value.(string)
		if !ok {
			return "type must be string"
		}
		u, err := url.Parse(str)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return "invalid url format"
		}

	case "boolean":
		switch value.(type) {
		case bool:
			return ""
		default:
			return "type must be boolean"
		}

	case "date":
		str, ok := value.(string)
		if !ok {
			return "type must be string"
		}
		// 尝试解析日期格式
		if _, err := time.Parse("2006-01-02", str); err != nil {
			return "invalid date format (expected: YYYY-MM-DD)"
		}

	case "datetime":
		str, ok := value.(string)
		if !ok {
			return "type must be string"
		}
		// 尝试解析日期时间格式
		if _, err := time.Parse(time.RFC3339, str); err != nil {
			if _, err := time.Parse("2006-01-02 15:04:05", str); err != nil {
				return "invalid datetime format (expected: RFC3339 or YYYY-MM-DD HH:MM:SS)"
			}
		}

	case "integer":
		switch value.(type) {
		case int, int8, int16, int32, int64:
			return ""
		case uint, uint8, uint16, uint32, uint64:
			return ""
		case json.Number:
			return ""
		case string:
			// 尝试解析为整数
			if _, err := strconv.ParseInt(value.(string), 10, 64); err != nil {
				return "type must be integer"
			}
		default:
			return "type must be integer"
		}

	case "float":
		switch value.(type) {
		case float32, float64:
			return ""
		case int, int8, int16, int32, int64:
			return ""
		case uint, uint8, uint16, uint32, uint64:
			return ""
		case json.Number:
			return ""
		case string:
			// 尝试解析为浮点数
			if _, err := strconv.ParseFloat(value.(string), 64); err != nil {
				return "type must be float"
			}
		default:
			return "type must be float"
		}

	case "array":
		if _, ok := value.([]interface{}); !ok {
			return "type must be array"
		}

	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return "type must be object"
		}
	}

	return ""
}
