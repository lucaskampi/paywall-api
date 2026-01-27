package ws

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

type client struct {
	conn *websocket.Conn
}

type Hub struct {
	mu      sync.Mutex
	clients map[*client]struct{}
}

var h = &Hub{clients: map[*client]struct{}{}}

// Broadcast sends a JSON-serializable message to all connected clients.
func Broadcast(v interface{}) {
	h.mu.Lock()
	clients := make([]*client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		go func(c *client) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := wsjson.Write(ctx, c.conn, v); err != nil {
				// write error; close connection
				c.conn.Close(websocket.StatusInternalError, "write error")
				h.unregister(c)
			}
		}(c)
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
}

// ServeWS upgrades the HTTP connection to a WebSocket and registers the client.
func ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}
	c := &client{conn: conn}
	h.register(c)

	// read loop to keep connection open and detect close from client
	go func() {
		defer func() {
			conn.Close(websocket.StatusNormalClosure, "closing")
			h.unregister(c)
		}()
		for {
			var v interface{}
			ctx := r.Context()
			if err := wsjson.Read(ctx, conn, &v); err != nil {
				return
			}
		}
	}()
}
