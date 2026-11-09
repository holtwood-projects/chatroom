package ws

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"chatroom/internal/auth"
	"chatroom/internal/config"
	"chatroom/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

const (
	// pongWait 等待客户端 Pong 的超时时间。
	pongWait = 60 * time.Second
	// pingInterval 向客户端发送 Ping 的间隔，必须小于 pongWait。
	pingInterval = 30 * time.Second
	// writeWait 每次写操作的截止时间。
	writeWait = 10 * time.Second
	// sendBufSize 每个客户端发送缓冲区大小。
	sendBufSize = 256
)

type Client struct {
	room      *RoomHub
	conn      *websocket.Conn
	send      chan []byte
	processor MessageProcessor
	userID    uint
	uname     string
	cfg       config.Config
}

func newUpgrader(cfg config.Config) websocket.Upgrader {
	return websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return false
			}
			return cfg.AllowsOrigin(origin, r)
		},
		Subprotocols: []string{"chatroom.v1"},
	}
}

type InboundMessage struct {
	Type     string `json:"type"`
	Content  string `json:"content"`
	IsTyping bool   `json:"is_typing"`
}

type OutboundMessage struct {
	Type      string    `json:"type"`
	ID        uint      `json:"id"`
	RoomID    uint      `json:"room_id"`
	UserID    uint      `json:"user_id"`
	Username  string    `json:"username"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type wsTypingEvent struct {
	Type     string `json:"type"`
	RoomID   uint   `json:"room_id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
}

type wsSimpleMsg struct {
	Type    string `json:"type"`
	Content string `json:"content,omitempty"`
}

func extractWSTicket(r *http.Request) string {
	for _, proto := range websocket.Subprotocols(r) {
		if strings.HasPrefix(proto, "ticket.") {
			return strings.TrimPrefix(proto, "ticket.")
		}
	}
	return ""
}

// Serve 返回 Gin 处理函数，用于校验用户、加入房间并启动读写循环。
func Serve(h *Hub, db *gorm.DB, cfg config.Config) gin.HandlerFunc {
	upgrader := newUpgrader(cfg)
	return func(c *gin.Context) {
		roomIDStr := c.Query("room_id")
		rid64, err := strconv.ParseUint(roomIDStr, 10, 64)
		if err != nil || rid64 == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
			return
		}
		var room models.Room
		if err := db.First(&room, uint(rid64)).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
			return
		}

		ticket := extractWSTicket(c.Request)
		if ticket == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "missing ws ticket"})
			return
		}
		claims, err := auth.ValidateAndConsumeWSTicket(db, ticket, cfg.JWTSecret, uint(rid64))
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid ws ticket"})
			return
		}

		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
			return
		}

		// 协议协商由 upgrader 的 Subprotocols 字段完成，Upgrade 内部会
		// 把选中的子协议写入响应头；不要再用 responseHeader 重复写
		// Sec-WebSocket-Protocol，否则会产生两个相同的响应头，导致
		// 浏览器解析异常并中断握手（close 1006）。
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error().Err(err).Uint64("room_id", rid64).Str("remote", c.Request.RemoteAddr).Msg("ws upgrade")
			return
		}
		rh := h.GetRoom(uint(rid64))
		processor := NewDefaultMessageProcessor(db, rh, MessageProcessorConfig{
			RoomID:     uint(rid64),
			UserID:     user.ID,
			Username:   user.Username,
			MaxSize:    cfg.WsMaxMessageSize,
			MaxContent: cfg.WsMaxContentSize,
		})
		client := &Client{
			room:      rh,
			conn:      conn,
			send:      make(chan []byte, sendBufSize),
			processor: processor,
			userID:    user.ID,
			uname:     user.Username,
			cfg:       cfg,
		}
		// GetRoom 之后房间可能被延迟清理：若 run goroutine 已退出，
		// 直接发送会永久阻塞。stop 已关闭时关闭连接，让客户端重连。
		select {
		case rh.register <- client:
		case <-rh.stop:
			_ = conn.Close()
			return
		}

		go client.writePump()
		client.readPump()
	}
}

// readPump 负责读取客户端信息、校验输入并推送到房间广播。
func (c *Client) readPump() {
	defer func() {
		// 房间可能已停止；此时 unregister 无人接收，必须走 stop 分支。
		select {
		case c.room.unregister <- c:
		case <-c.room.stop:
		}
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(c.cfg.WsMaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait)) //nolint:errcheck // deadline only affects subsequent reads
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Warn().Err(err).Uint("user_id", c.userID).Uint("room_id", c.room.roomID).Msg("ws unexpected close")
			}
			break
		}
		var in InboundMessage
		if err := json.Unmarshal(data, &in); err != nil {
			continue
		}

		switch in.Type {
		case "ping":
			if b, err := json.Marshal(wsSimpleMsg{Type: "pong"}); err == nil {
				select {
				case c.send <- b:
				default:
				}
			}

		case "typing":
			evt := wsTypingEvent{Type: "typing", RoomID: c.room.roomID, UserID: c.userID, Username: c.uname, IsTyping: in.IsTyping}
			if b, err := json.Marshal(evt); err == nil {
				c.room.broadcast <- b
			}

		case "message":
			c.handleMessage(in.Content)

		default:
			c.handleMessage(in.Content)
		}
	}
}

// handleMessage 校验并持久化聊天消息，然后广播给房间内的所有客户端。
func (c *Client) handleMessage(content string) {
	result := c.processor.Process(content)
	if result == nil {
		return
	}
	b, err := json.Marshal(result.Message)
	if err != nil {
		return
	}
	if result.Broadcast {
		c.room.broadcast <- b
	} else {
		select {
		case c.send <- b:
		default:
		}
	}
}

// writePump 周期性发送服务端数据与心跳，防止浏览器断线。
func (c *Client) writePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck // deadline only affects subsequent writes
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{}) //nolint:errcheck // connection closing anyway
				return
			}
			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message) //nolint:errcheck // NextWriter error is sufficient
			_ = w.Close()

			n := len(c.send)
			for range n {
				msg, ok := <-c.send
				if !ok {
					return
				}
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck // deadline only affects subsequent writes
				if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait)) //nolint:errcheck // deadline only affects subsequent writes
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
