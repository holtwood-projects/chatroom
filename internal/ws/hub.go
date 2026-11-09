package ws

import (
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"
)

// Hub 管理房间级别的子 Hub，实现延迟创建与并发安全。
type Hub struct {
	mu           sync.RWMutex
	rooms        map[uint]*RoomHub
	cleanupAfter time.Duration
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[uint]*RoomHub), cleanupAfter: 3 * time.Minute}
}

// GetRoom 若房间未初始化则懒加载一个 RoomHub。
func (h *Hub) GetRoom(roomID uint) *RoomHub {
	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()
	if room != nil {
		return room
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	room = h.rooms[roomID]
	if room != nil {
		return room
	}
	room = NewRoomHub(roomID)
	room.parent = h
	h.rooms[roomID] = room
	go room.run()
	return room
}

func (h *Hub) Online(roomID uint) int {
	h.mu.RLock()
	room := h.rooms[roomID]
	h.mu.RUnlock()
	if room == nil {
		return 0
	}
	return room.Online()
}

func (h *Hub) cleanupRoomLater(room *RoomHub) {
	if h.cleanupAfter <= 0 {
		return
	}
	time.AfterFunc(h.cleanupAfter, func() {
		// 先在写锁内确认空房并摘除 map 条目，再 Stop。顺序反过来会留下
		// 窗口：GetRoom 仍能返回已停止的 RoomHub，新连接的 register
		// 在无人接收的 channel 上永久阻塞（handler goroutine 泄漏）。
		h.mu.Lock()
		if h.rooms[room.roomID] != room || room.Online() != 0 {
			h.mu.Unlock()
			return
		}
		delete(h.rooms, room.roomID)
		h.mu.Unlock()

		room.Stop()
	})
}

// Shutdown 关闭所有 RoomHub goroutine，用于优雅停服。
func (h *Hub) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for id, room := range h.rooms {
		room.Stop()
		delete(h.rooms, id)
	}
}

type RoomHub struct {
	parent     *Hub
	roomID     uint
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	stop       chan struct{}
	online     int32
}

func NewRoomHub(roomID uint) *RoomHub {
	return &RoomHub{
		roomID:     roomID,
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, 256),
		stop:       make(chan struct{}),
	}
}

// broadcastToClients 向房间内所有客户端发送数据，慢客户端会被清理。
// 注意：此函数会关闭慢客户端的 send 通道，但会先从 clients map 中删除，
// 这样后续的 unregister case 可以通过检查 clients map 来避免重复关闭。
func (rh *RoomHub) broadcastToClients(data []byte) {
	dropped := false
	for cli := range rh.clients {
		select {
		case cli.send <- data:
		default:
			// 先从 map 中删除，再关闭通道，防止 unregister case 重复关闭。
			// 这里也必须同步在线人数，否则后续 unregister 因 map miss 提前
			// 返回，在线数会永久偏高、空房也不会触发延迟清理。
			delete(rh.clients, cli)
			close(cli.send)
			dropped = true
		}
	}
	if dropped {
		rh.updateOnline()
		if len(rh.clients) == 0 && rh.parent != nil {
			rh.parent.cleanupRoomLater(rh)
		}
	}
}

// wsPresenceEvent 用于 join/leave 广播的类型安全结构体。
type wsPresenceEvent struct {
	Type     string `json:"type"`
	RoomID   uint   `json:"room_id"`
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Online   int    `json:"online"`
}

// broadcastEvent 序列化事件并广播给房间内所有客户端。
func (rh *RoomHub) broadcastEvent(evt interface{}) {
	b, err := json.Marshal(evt)
	if err != nil {
		return
	}
	rh.broadcastToClients(b)
}

// updateOnline 更新当前在线人数。
func (rh *RoomHub) updateOnline() {
	// Gosec G115: Safe conversion - room client count is bounded and won't exceed int32 range in practice.
	// The nolint directive is placed per Gosec's official guidance for intentional conversions.
	n := int32(len(rh.clients)) //nolint:gosec // G115 - bounded client count, safe to convert
	atomic.StoreInt32(&rh.online, n)
}

func (rh *RoomHub) run() {
	for {
		select {
		case <-rh.stop:
			for c := range rh.clients {
				close(c.send)
				delete(rh.clients, c)
			}
			atomic.StoreInt32(&rh.online, 0)
			return

		case c := <-rh.register:
			rh.clients[c] = true
			rh.updateOnline()
			evt := wsPresenceEvent{Type: "join", RoomID: rh.roomID, UserID: c.userID, Username: c.uname, Online: rh.Online()}
			rh.broadcastEvent(evt)

		case c := <-rh.unregister:
			if _, ok := rh.clients[c]; !ok {
				break
			}
			delete(rh.clients, c)
			close(c.send)
			rh.updateOnline()
			evt := wsPresenceEvent{Type: "leave", RoomID: rh.roomID, UserID: c.userID, Username: c.uname, Online: rh.Online()}
			rh.broadcastEvent(evt)
			if rh.parent != nil && len(rh.clients) == 0 {
				rh.parent.cleanupRoomLater(rh)
			}

		case msg := <-rh.broadcast:
			rh.broadcastToClients(msg)
		}
	}
}

// Stop 停止 RoomHub 的 run goroutine。
func (rh *RoomHub) Stop() {
	select {
	case <-rh.stop:
	default:
		close(rh.stop)
	}
}

// Online 返回房间在线客户端数量，供 REST 接口复用。
func (rh *RoomHub) Online() int { return int(atomic.LoadInt32(&rh.online)) }
