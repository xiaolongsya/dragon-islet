package logic

import (
	"dragon-islet/internal/model"
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Client 每一个在线的游侠
type Client struct {
	ID   uint
	Conn *websocket.Conn
	Send chan []byte
}

// Hub 龙屿中央通讯枢纽
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var GlobalHub = &Hub{
	broadcast:  make(chan []byte),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	clients:    make(map[*Client]bool),
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Register(c *Client) {
	h.register <- c
}

func (h *Hub) Unregister(c *Client) {
	h.unregister <- c
}

// ReadPump 监听客户端发来的消息 (目前主要用于检测断开)
func (c *Client) ReadPump() {
	defer func() {
		GlobalHub.Unregister(c)
		c.Conn.Close()
	}()
	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump 将 Hub 广播的消息写入 WebSocket 连接
func (c *Client) WritePump() {
	defer func() {
		c.Conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}

// BroadcastMessage 将消息广播给所有在线游侠
func (h *Hub) BroadcastMessage(msg *model.Message) {
	data, _ := json.Marshal(msg)
	h.broadcast <- data
}
