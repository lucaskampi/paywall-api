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
	conn   *websocket.Conn
	mu     sync.Mutex
	closed bool
}

func (c *client) Close(status websocket.StatusCode, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	return c.conn.Close(status, reason)
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

	log.Printf("ws broadcast to %d clients; payload type=%T", len(clients), v)

	for _, c := range clients {
		go func(c *client) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := wsjson.Write(ctx, c.conn, v); err != nil {
				log.Printf("ws write error: %v; closing client %p", err, c)
				h.unregister(c)
				_ = c.Close(websocket.StatusInternalError, "write error")
			}
		}(c)
	}
}

func (h *Hub) register(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws register %p; clients=%d", c, count)
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws unregister %p; clients=%d", c, count)
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

	// run the read loop synchronously so the handler stays alive for the lifetime of the connection
	defer func() {
		h.unregister(c)
		_ = c.Close(websocket.StatusNormalClosure, "closing")
	}()

	for {
		var v interface{}
		// use request context so Read returns when the request is cancelled
		if err := wsjson.Read(r.Context(), conn, &v); err != nil {
			// treat normal client disconnects as non-errors
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway ||
				err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			log.Printf("ws read closed: %v", err)
			return
		}
		// you can handle incoming messages here if needed
	}
}
