package ws

import (
	"sync"
	"testing"
	"time"
)

func startTestRoomHub(t *testing.T, _ uint) *RoomHub {
	t.Helper()
	hub := NewHub()
	rh := hub.GetRoom(1)
	t.Cleanup(hub.Shutdown)
	return rh
}

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub() returned nil")
	}
	if hub.rooms == nil {
		t.Error("NewHub() rooms map is nil")
	}
}

func TestHub_Online_EmptyRoom(t *testing.T) {
	hub := NewHub()
	online := hub.Online(1)
	if online != 0 {
		t.Errorf("Online() for empty room = %d, want 0", online)
	}
}

func TestHub_Online_NonExistentRoom(t *testing.T) {
	hub := NewHub()
	online := hub.Online(999)
	if online != 0 {
		t.Errorf("Online() for non-existent room = %d, want 0", online)
	}
}

func TestRoomHub_Register(t *testing.T) {
	rh := startTestRoomHub(t, 1)

	// Create a fake client
	client := &Client{
		room:   rh,
		userID: 1,
		uname:  "testuser",
		send:   make(chan []byte, 256),
	}

	// Register client
	rh.register <- client

	// Give it time to process
	time.Sleep(10 * time.Millisecond)

	if rh.Online() != 1 {
		t.Errorf("Online() after register = %d, want 1", rh.Online())
	}
}

func TestRoomHub_Unregister(t *testing.T) {
	rh := startTestRoomHub(t, 1)

	client := &Client{
		room:   rh,
		userID: 1,
		uname:  "testuser",
		send:   make(chan []byte, 256),
	}

	// Register then unregister
	rh.register <- client
	time.Sleep(10 * time.Millisecond)

	rh.unregister <- client
	time.Sleep(10 * time.Millisecond)

	if rh.Online() != 0 {
		t.Errorf("Online() after unregister = %d, want 0", rh.Online())
	}
}

func TestRoomHub_Broadcast(t *testing.T) {
	rh := startTestRoomHub(t, 1)

	// Create multiple clients
	clients := make([]*Client, 3)
	for i := range 3 {
		clients[i] = &Client{
			room:   rh,
			userID: uint(i + 1),
			uname:  "user" + string(rune('0'+i)),
			send:   make(chan []byte, 256),
		}
	}

	// Register all clients
	for _, c := range clients {
		rh.register <- c
	}
	time.Sleep(20 * time.Millisecond)

	// Broadcast a message
	testMsg := []byte(`{"type":"message","content":"hello"}`)
	rh.broadcast <- testMsg

	// Check all clients received the message
	var wg sync.WaitGroup
	received := make([]bool, 3)

	for i, c := range clients {
		wg.Add(1)
		go func(idx int, client *Client) {
			defer wg.Done()
			deadline := time.After(100 * time.Millisecond)
			for {
				select {
				case msg := <-client.send:
					if string(msg) == string(testMsg) {
						received[idx] = true
						return
					}
				case <-deadline:
					return
				}
			}
		}(i, c)
	}

	wg.Wait()

	for i, r := range received {
		if !r {
			t.Errorf("Client %d did not receive broadcast message", i)
		}
	}
}

func TestRoomHub_MultipleRooms(t *testing.T) {
	hub := NewHub()
	t.Cleanup(hub.Shutdown)

	// Create clients for different rooms
	rh1 := hub.GetRoom(1)
	rh2 := hub.GetRoom(2)

	client1 := &Client{
		room:   rh1,
		userID: 1,
		uname:  "user1",
		send:   make(chan []byte, 256),
	}
	client2 := &Client{
		room:   rh2,
		userID: 2,
		uname:  "user2",
		send:   make(chan []byte, 256),
	}

	rh1.register <- client1
	rh2.register <- client2

	time.Sleep(20 * time.Millisecond)

	if hub.Online(1) != 1 {
		t.Errorf("Online(1) = %d, want 1", hub.Online(1))
	}
	if hub.Online(2) != 1 {
		t.Errorf("Online(2) = %d, want 1", hub.Online(2))
	}
}

func TestClient_Send(t *testing.T) {
	_ = startTestRoomHub(t, 1)
	client := &Client{
		send: make(chan []byte, 256),
	}

	testMsg := []byte("test message")

	// Send should not block
	select {
	case client.send <- testMsg:
		// OK
	default:
		t.Error("Send channel blocked unexpectedly")
	}

	// Verify message received
	select {
	case msg := <-client.send:
		if string(msg) != string(testMsg) {
			t.Errorf("Received message = %s, want %s", msg, testMsg)
		}
	default:
		t.Error("No message in send channel")
	}
}

func TestRoomHub_Concurrent(t *testing.T) {
	rh := startTestRoomHub(t, 1)

	var wg sync.WaitGroup
	numClients := 10

	// Concurrently register clients
	for i := range numClients {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			client := &Client{
				room:   rh,
				userID: uint(id),
				uname:  "user",
				send:   make(chan []byte, 256),
			}
			rh.register <- client
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond)

	if rh.Online() != numClients {
		t.Errorf("Online() after concurrent register = %d, want %d", rh.Online(), numClients)
	}
}

// TestHub_CleanupRemovesRoomBeforeStopping 覆盖空房清理的顺序：必须先摘除
// map 条目再 Stop，否则 GetRoom 可能返回已停止的 RoomHub。
func TestHub_CleanupRemovesRoomBeforeStopping(t *testing.T) {
	hub := NewHub()
	hub.cleanupAfter = time.Millisecond
	t.Cleanup(hub.Shutdown)

	rh := hub.GetRoom(7)
	hub.cleanupRoomLater(rh)

	deadline := time.Now().Add(2 * time.Second)
	for {
		select {
		case <-rh.stop:
			hub.mu.RLock()
			_, exists := hub.rooms[7]
			hub.mu.RUnlock()
			if exists {
				t.Fatal("room still present in map after Stop")
			}
			return
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("cleanup did not stop the empty room")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// TestRoomHub_RegisterAfterStopDoesNotBlock 覆盖 register 的 stop 分支：
// 房间已停止时发送方不能永久阻塞（原实现会导致 handler goroutine 泄漏）。
func TestRoomHub_RegisterAfterStopDoesNotBlock(t *testing.T) {
	hub := NewHub()
	t.Cleanup(hub.Shutdown)
	rh := hub.GetRoom(8)
	rh.Stop()

	client := &Client{room: rh, send: make(chan []byte, 1)}
	select {
	case rh.register <- client:
		t.Fatal("register on a stopped room must not succeed")
	case <-rh.stop:
	}
}
