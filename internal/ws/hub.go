package ws

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Event struct {
	Type   string          `json:"type"`
	Domain string          `json:"domain,omitempty"`
	RoomID string          `json:"roomId"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type client struct {
	conn   *websocket.Conn
	roomID string
	domain string
	send   chan []byte
}

type Hub struct {
	mu       sync.RWMutex
	rooms    map[string]map[*client]struct{}
	upgrader websocket.Upgrader
}

func NewHub(allowedOrigins []string) *Hub {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = struct{}{}
	}
	return &Hub{
		rooms: make(map[string]map[*client]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				origin := r.Header.Get("Origin")
				if origin == "" {
					return true
				}
				_, ok := allowed[origin]
				return ok
			},
		},
	}
}

func (h *Hub) Upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	return h.upgrader.Upgrade(w, r, nil)
}

func (h *Hub) Register(conn *websocket.Conn, roomID, domain string) {
	c := &client{conn: conn, roomID: roomID, domain: domain, send: make(chan []byte, 16)}
	h.mu.Lock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*client]struct{})
	}
	h.rooms[roomID][c] = struct{}{}
	h.mu.Unlock()

	go h.writePump(c)
}

func (h *Hub) Unregister(c *client) {
	h.mu.Lock()
	if clients, ok := h.rooms[c.roomID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, c.roomID)
		}
	}
	h.mu.Unlock()
	close(c.send)
}

func (h *Hub) writePump(c *client) {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
	_ = c.conn.Close()
}

func (h *Hub) Broadcast(roomID, domain, eventType string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	ev := Event{Type: eventType, Domain: domain, RoomID: roomID, Data: json.RawMessage(data)}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()

	for c := range clients {
		if domain != "" && domain != "all" && c.domain != "" && c.domain != "all" && c.domain != domain {
			continue
		}
		select {
		case c.send <- raw:
		default:
		}
	}
}

func (h *Hub) BroadcastRaw(roomID, domain, eventType string, data json.RawMessage) {
	ev := Event{Type: eventType, Domain: domain, RoomID: roomID, Data: data}
	raw, err := json.Marshal(ev)
	if err != nil {
		return
	}

	h.mu.RLock()
	clients := h.rooms[roomID]
	h.mu.RUnlock()

	for c := range clients {
		if domain != "" && domain != "all" && c.domain != "" && c.domain != "all" && c.domain != domain {
			continue
		}
		select {
		case c.send <- raw:
		default:
		}
	}
}

// Client exposes unregister for handler.
type Client = client

func (h *Hub) ClientConn(conn *websocket.Conn, roomID, domain string) *Client {
	c := &client{conn: conn, roomID: roomID, domain: domain, send: make(chan []byte, 16)}
	h.mu.Lock()
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*client]struct{})
	}
	h.rooms[roomID][c] = struct{}{}
	h.mu.Unlock()
	go h.writePump(c)
	return c
}

func (h *Hub) RemoveClient(c *Client) {
	h.Unregister(c)
}
