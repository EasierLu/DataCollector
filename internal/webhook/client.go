package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/datacollector/datacollector/internal/model"
)

// Client Webhook HTTP客户端
type Client struct{}

// NewClient 创建Webhook客户端
func NewClient() *Client {
	return &Client{}
}

// Send 执行单次Webhook HTTP调用
// - 根据Config.Method发送请求（默认POST）
// - 设置Content-Type: application/json
// - 添加Config.Headers中的自定义请求头
// - 如果Config.Secret不为空，计算HMAC-SHA256签名并添加 X-Webhook-Signature 头
// - 使用Config.Timeout设置超时（默认10秒）
// - 返回HTTP状态码和error
func (c *Client) Send(url string, body []byte, config *model.WebhookConfig) (int, error) {
	method := config.Method
	if method == "" {
		method = "POST"
	}

	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// 添加自定义请求头
	for k, v := range config.Headers {
		req.Header.Set(k, v)
	}

	// HMAC-SHA256 签名
	if config.Secret != "" {
		mac := hmac.New(sha256.New, []byte(config.Secret))
		mac.Write(body)
		signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Webhook-Signature", signature)
	}

	httpClient := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to send webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	return resp.StatusCode, nil
}
