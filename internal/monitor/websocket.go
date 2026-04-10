package monitor

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHub 管理所有 WebSocket 连接
type WebSocketHub struct {
	clients        map[*Client]bool
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	mu             sync.RWMutex
	logger         *slog.Logger
	allowedOrigins []string
	stopCh         chan struct{}
}

// Client 代表一个 WebSocket 客户端连接
type Client struct {
	hub       *WebSocketHub
	conn      *websocket.Conn
	send      chan []byte
	closeOnce sync.Once
}

// WSMessage WebSocket 消息格式
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// StatsUpdateData 统计更新数据
type StatsUpdateData struct {
	SourceID   int64     `json:"source_id"`
	TodayCount int64     `json:"today_count"`
	Timestamp  time.Time `json:"timestamp"`
}

// closeSend 安全关闭 client 的 send channel，保证只关闭一次
func (c *Client) closeSend() {
	c.closeOnce.Do(func() {
		close(c.send)
	})
}

func newUpgrader(allowedOrigins []string) websocket.Upgrader {
	return websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			for _, o := range allowedOrigins {
				if o == "*" || o == origin {
					return true
				}
			}
			return false
		},
	}
}

// NewWebSocketHub 创建新的 WebSocket Hub
func NewWebSocketHub(logger *slog.Logger, allowedOrigins []string) *WebSocketHub {
	return &WebSocketHub{
		clients:        make(map[*Client]bool),
		broadcast:      make(chan []byte),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		logger:         logger,
		allowedOrigins: allowedOrigins,
		stopCh:         make(chan struct{}),
	}
}

// Run 启动 Hub 主循环
func (h *WebSocketHub) Run() {
	for {
		select {
		case <-h.stopCh:
			h.mu.Lock()
			for client := range h.clients {
				client.closeSend()
				client.conn.Close()
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Debug("client registered", "client_count", len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.closeSend()
			}
			h.mu.Unlock()
			h.logger.Debug("client unregistered", "client_count", len(h.clients))

		case message := <-h.broadcast:
			h.mu.RLock()
			clients := make([]*Client, 0, len(h.clients))
			for client := range h.clients {
				clients = append(clients, client)
			}
			h.mu.RUnlock()

			for _, client := range clients {
				select {
				case client.send <- message:
				default:
					// 客户端发送缓冲区已满，关闭连接
					h.mu.Lock()
					if _, ok := h.clients[client]; ok {
						delete(h.clients, client)
						client.closeSend()
						client.conn.Close()
					}
					h.mu.Unlock()
				}
			}
		}
	}
}

// Stop gracefully shuts down the Hub's Run loop.
func (h *WebSocketHub) Stop() {
	close(h.stopCh)
}

// BroadcastStatsUpdate 广播统计数据更新通知
// 通知所有连接的客户端重新获取统计数据
func (h *WebSocketHub) BroadcastStatsUpdate() {
	msg := WSMessage{
		Type: "stats_update",
		Data: map[string]string{"action": "refresh"},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("failed to marshal stats update message", "error", err)
		return
	}

	select {
	case h.broadcast <- jsonData:
	default:
		h.logger.Warn("broadcast channel full, message dropped")
	}
}

// HandleWebSocket 处理 WebSocket 连接升级
func (h *WebSocketHub) HandleWebSocket(c *gin.Context) {
	up := newUpgrader(h.allowedOrigins)

	// gorilla/websocket skips manually set Sec-WebSocket-Protocol response
	// headers, so we must use the Upgrader.Subprotocols field for negotiation.
	// Echo back the client's full sub-protocol so the browser accepts the upgrade.
	if proto := c.GetHeader("Sec-WebSocket-Protocol"); proto != "" {
		for _, p := range strings.Split(proto, ",") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "access_token.") {
				up.Subprotocols = []string{p}
				break
			}
		}
	}

	conn, err := up.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", "error", err)
		return
	}

	client := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, 256),
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

// writePump 处理向客户端发送消息
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// 批量发送等待中的消息
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// readPump 处理从客户端读取消息
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.logger.Warn("websocket unexpected close", "error", err)
			}
			break
		}
	}
}
