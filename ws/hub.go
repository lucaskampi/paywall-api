package ws

import (
	"context"
	"encoding/json"
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

func (c *client) WriteJSON(ctx context.Context, v interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return context.Canceled
	}
	return wsjson.Write(ctx, c.conn, v)
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
	// subs maps a Stripe Checkout Session ID to subscribed clients.
	subs map[string]map[*client]struct{}
	// clientSubs is the reverse index to allow cleanup on disconnect.
	clientSubs map[*client]map[string]struct{}
}

var h = &Hub{
	clients:    map[*client]struct{}{},
	subs:       map[string]map[*client]struct{}{},
	clientSubs: map[*client]map[string]struct{}{},
}

// Event is the outbound WS payload envelope.
// Frontend can switch on Type and read Data.
type Event struct {
	Type string      `json:"type"`
	Data interface{} `json:"data,omitempty"`
	At   string      `json:"at"`
}

type inboundMessage struct {
	Type      string `json:"type"`
	SessionID string `json:"session_id,omitempty"`
}

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
			if err := c.WriteJSON(ctx, v); err != nil {
				log.Printf("ws write error: %v; closing client %p", err, c)
				h.unregister(c)
				_ = c.Close(websocket.StatusInternalError, "write error")
			}
		}(c)
	}
}

// BroadcastToSession sends a JSON-serializable message to clients subscribed to a Stripe Checkout Session ID.
func BroadcastToSession(sessionID string, v interface{}) {
	if sessionID == "" {
		return
	}

	h.mu.Lock()
	set := h.subs[sessionID]
	clients := make([]*client, 0, len(set))
	for c := range set {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	log.Printf("ws broadcast session_id=%s to %d clients; payload type=%T", sessionID, len(clients), v)

	for _, c := range clients {
		go func(c *client) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			if err := c.WriteJSON(ctx, v); err != nil {
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
	if _, ok := h.clientSubs[c]; !ok {
		h.clientSubs[c] = map[string]struct{}{}
	}
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws register %p; clients=%d", c, count)
}

func (h *Hub) unregister(c *client) {
	h.mu.Lock()
	delete(h.clients, c)
	// remove subscriptions for this client
	if subs, ok := h.clientSubs[c]; ok {
		for sessionID := range subs {
			if set, ok := h.subs[sessionID]; ok {
				delete(set, c)
				if len(set) == 0 {
					delete(h.subs, sessionID)
				}
			}
		}
		delete(h.clientSubs, c)
	}
	count := len(h.clients)
	h.mu.Unlock()
	log.Printf("ws unregister %p; clients=%d", c, count)
}

func (h *Hub) subscribe(c *client, sessionID string) {
	if sessionID == "" {
		return
	}
	h.mu.Lock()
	set := h.subs[sessionID]
	if set == nil {
		set = map[*client]struct{}{}
		h.subs[sessionID] = set
	}
	set[c] = struct{}{}
	if _, ok := h.clientSubs[c]; !ok {
		h.clientSubs[c] = map[string]struct{}{}
	}
	h.clientSubs[c][sessionID] = struct{}{}
	h.mu.Unlock()
	log.Printf("ws subscribe %p session_id=%s", c, sessionID)
}

func (h *Hub) unsubscribe(c *client, sessionID string) {
	if sessionID == "" {
		return
	}
	h.mu.Lock()
	if set, ok := h.subs[sessionID]; ok {
		delete(set, c)
		if len(set) == 0 {
			delete(h.subs, sessionID)
		}
	}
	if subs, ok := h.clientSubs[c]; ok {
		delete(subs, sessionID)
	}
	h.mu.Unlock()
	log.Printf("ws unsubscribe %p session_id=%s", c, sessionID)
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
		var raw json.RawMessage
		// use request context so Read returns when the request is cancelled
		if err := wsjson.Read(r.Context(), conn, &raw); err != nil {
			// treat normal client disconnects as non-errors
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway ||
				err == context.Canceled || err == context.DeadlineExceeded {
				return
			}
			log.Printf("ws read closed: %v", err)
			return
		}

		var msg inboundMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = c.WriteJSON(ctx, Event{Type: "error", Data: map[string]string{"message": "invalid JSON"}, At: time.Now().UTC().Format(time.RFC3339Nano)})
			cancel()
			continue
		}

		switch msg.Type {
		case "subscribe":
			h.subscribe(c, msg.SessionID)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = c.WriteJSON(ctx, Event{Type: "subscribed", Data: map[string]string{"session_id": msg.SessionID}, At: time.Now().UTC().Format(time.RFC3339Nano)})
			cancel()
		case "unsubscribe":
			h.unsubscribe(c, msg.SessionID)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = c.WriteJSON(ctx, Event{Type: "unsubscribed", Data: map[string]string{"session_id": msg.SessionID}, At: time.Now().UTC().Format(time.RFC3339Nano)})
			cancel()
		case "ping":
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = c.WriteJSON(ctx, Event{Type: "pong", At: time.Now().UTC().Format(time.RFC3339Nano)})
			cancel()
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = c.WriteJSON(ctx, Event{Type: "error", Data: map[string]string{"message": "unknown message type"}, At: time.Now().UTC().Format(time.RFC3339Nano)})
			cancel()
		}
	}
}
