package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

var (
	hmacKeyMu sync.RWMutex
	hmacKey   []byte
)

// SetHMACKey 设置用于 HMAC-SHA256 的密钥（应在服务启动时调用）
func SetHMACKey(key string) {
	hmacKeyMu.Lock()
	defer hmacKeyMu.Unlock()
	hmacKey = []byte(key)
}

// hmacSHA256 使用 HMAC-SHA256 计算哈希
func hmacSHA256(data string) string {
	hmacKeyMu.RLock()
	key := hmacKey
	hmacKeyMu.RUnlock()

	if len(key) == 0 {
		// 未设置 HMAC key 时回退到 SHA-256
		return plainSHA256(data)
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

// plainSHA256 纯 SHA-256 哈希（用于兼容旧数据）
func plainSHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// verifyHash 验证哈希值，优先尝试 HMAC-SHA256，回退到 plain SHA-256
func verifyHash(raw, storedHash string) bool {
	// 优先用 HMAC-SHA256 校验
	if hmacSHA256(raw) == storedHash {
		return true
	}
	// 兼容旧的 plain SHA-256 哈希
	if plainSHA256(raw) == storedHash {
		return true
	}
	return false
}
