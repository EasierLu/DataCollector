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
		panic("HMAC key not set: call SetHMACKey before hashing")
	}
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}
