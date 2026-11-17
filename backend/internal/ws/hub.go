// Package ws implements a small in-memory fan-out hub for live parking
// statuses. One process owns the hub; clients subscribe to a single lot or to
// every lot (lotID == 0).
package ws

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"
)

// Message is the envelope sent to browser clients. Unused fields are omitted
// depending on Type: spot_status or camera_status.
type Message struct {
	Type        string     `json:"type"`
	LotID       int64      `json:"lot_id"`
	SpotID      int64      `json:"spot_id,omitempty"`
	Code        string     `json:"code,omitempty"`
	Status      string     `json:"status,omitempty"`
	Confidence  *float32   `json:"confidence,omitempty"`
	CameraID    int64      `json:"camera_id,omitempty"`
	LastFrameAt *time.Time `json:"last_frame_at,omitempty"`
	At          time.Time  `json:"at"`
}

// Hub keeps connected clients and broadcasts messages without blocking
// producers. Slow clients are disconnected instead of stalling the hub.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{})}
}

func (h *Hub) add(c *Client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

// Remove unregisters a client and closes its send channel. Safe to call twice.
func (h *Hub) Remove(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.send)
}

// Broadcast fans a message out to matching clients. A client with lotID 0
// receives everything; otherwise only messages for its lot are delivered.
func (h *Hub) Broadcast(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		slog.Error("ws: marshal message", "error", err)
		return
	}

	var slow []*Client
	h.mu.RLock()
	for c := range h.clients {
		if c.lotID != 0 && c.lotID != msg.LotID {
			continue
		}
		select {
		case c.send <- data:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()

	for _, c := range slow {
		slog.Warn("ws: dropping slow client", "lot_id", c.lotID)
		h.Remove(c)
	}
}

// Count returns the number of connected clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
