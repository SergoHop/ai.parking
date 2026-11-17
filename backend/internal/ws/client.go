package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	// sendBuffer is intentionally small: a browser that cannot keep up with
	// parking events is disconnected instead of growing an unbounded queue.
	sendBuffer = 32
)

// Client is one WebSocket subscriber. It is registered in the hub as soon as
// it is created, and unregistered on disconnect.
type Client struct {
	hub   *Hub
	lotID int64
	send  chan []byte

	conn *websocket.Conn
}

// NewClient registers a subscriber for a lot (0 means all lots).
func NewClient(h *Hub, lotID int64) *Client {
	c := &Client{
		hub:   h,
		lotID: lotID,
		send:  make(chan []byte, sendBuffer),
	}
	h.add(c)
	return c
}

// LotID returns the subscribed lot, 0 for all lots.
func (c *Client) LotID() int64 { return c.lotID }

// Send exposes the outbound channel for tests.
func (c *Client) Send() <-chan []byte { return c.send }

// Serve pumps messages for an upgraded connection. It blocks until the client
// disconnects.
func (c *Client) Serve(conn *websocket.Conn) {
	c.conn = conn
	go c.writePump()
	c.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.hub.Remove(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.hub.Remove(c)
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
