package hub

import (
	"context"
	"encoding/json"
	"log"
	"realtime-dashboard/internal/metrics"
)

const broadcastBuffer = 4

type Hub struct {
	clients    map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	done       chan struct{}
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte, broadcastBuffer),
		done:       make(chan struct{}),
	}
}

func (h *Hub) Run(ctx context.Context) {
	defer func() {
		close(h.done)
		for c := range h.clients {
			close(c.send)
		}
	}()
	for {
		select {
		case client := <-h.register:
			h.clients[client] = struct{}{}
		case client := <-h.unregister:
			_, ok := h.clients[client]
			if ok {
				h.removeClient(client)
			}
		case msg := <-h.broadcast:
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					h.removeClient(c)
				}
			}
		case <-ctx.Done():
			return
		}
	}
}

func (h *Hub) Register(c *Client) {
	select {
	case h.register <- c:
	case <-h.done:
		return
	}
}

func (h *Hub) Unregister(c *Client) {
	select {
	case h.unregister <- c:
	case <-h.done:
		return
	}
}

func (h *Hub) Broadcast(s metrics.Snapshot) {
	data, err := json.Marshal(s)
	if err != nil {
		log.Printf("snapshot not marshalled: %v", err)
		return
	}
	select {
	case h.broadcast <- data:
	default:
	}
}

func (h *Hub) removeClient(c *Client) {
	delete(h.clients, c)
	close(c.send)
}
