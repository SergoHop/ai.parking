package ws

import (
	"encoding/json"
	"testing"
	"time"
)

func decode(t *testing.T, raw []byte) Message {
	t.Helper()
	var msg Message
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message: %v", err)
	}
	return msg
}

func expectMessage(t *testing.T, c *Client) Message {
	t.Helper()
	select {
	case raw := <-c.Send():
		return decode(t, raw)
	case <-time.After(time.Second):
		t.Fatal("expected a message")
		return Message{}
	}
}

func expectNoMessage(t *testing.T, c *Client) {
	t.Helper()
	select {
	case raw := <-c.Send():
		t.Fatalf("unexpected message: %s", raw)
	case <-time.After(20 * time.Millisecond):
	}
}

func TestBroadcastFiltersByLot(t *testing.T) {
	hub := NewHub()
	lot3 := NewClient(hub, 3)
	lot6 := NewClient(hub, 6)
	all := NewClient(hub, 0)

	hub.Broadcast(Message{Type: "spot_status", LotID: 3, SpotID: 21, Status: "free"})

	if msg := expectMessage(t, lot3); msg.LotID != 3 || msg.SpotID != 21 {
		t.Fatalf("unexpected message for lot 3: %+v", msg)
	}
	if msg := expectMessage(t, all); msg.LotID != 3 {
		t.Fatalf("unexpected message for all-lots client: %+v", msg)
	}
	expectNoMessage(t, lot6)
}

func TestBroadcastDropsSlowClient(t *testing.T) {
	hub := NewHub()
	slow := NewClient(hub, 0)

	// Nobody drains the channel, so the hub must drop the client.
	for i := 0; i <= sendBuffer; i++ {
		hub.Broadcast(Message{Type: "spot_status", LotID: 1, SpotID: int64(i)})
	}

	if hub.Count() != 0 {
		t.Fatalf("expected slow client to be dropped, count=%d", hub.Count())
	}
	// Closed channel is drained; a later broadcast must not panic.
	hub.Broadcast(Message{Type: "spot_status", LotID: 1})
	remaining := 0
	for range slow.Send() {
		remaining++
	}
	if remaining != sendBuffer {
		t.Fatalf("expected %d queued messages, got %d", sendBuffer, remaining)
	}
}

func TestRemoveIsIdempotent(t *testing.T) {
	hub := NewHub()
	client := NewClient(hub, 3)

	hub.Remove(client)
	hub.Remove(client)

	if hub.Count() != 0 {
		t.Fatalf("expected no clients, got %d", hub.Count())
	}
}
