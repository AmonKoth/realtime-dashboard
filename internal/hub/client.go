package hub

import (
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = time.Second * 10
	pongWait   = time.Second * 60
	pingPeriod = pongWait * 9 / 10
	sendBuffer = 16
	readLimit  = 512
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, sendBuffer),
	}
}

// Start registers the client and launches its two pumps.
func (c *Client) Start() {
	c.hub.Register(c)
	go c.read()
	go c.write()
}

func (c *Client) read() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close()
	}()
	c.conn.SetReadLimit(readLimit)
	err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
	if err != nil {
		log.Printf("extension error occurred %v", err)
		return
	}
	c.conn.SetPongHandler(func(_ string) error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		if err != nil {
			log.Printf("extension error occurred %v", err)
			return err
		}
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
	}
}

func (c *Client) write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				err := c.writeWithDeadline(websocket.CloseMessage, []byte{})
				if err != nil {
					log.Printf("connection is broken: %v", err)
				}
				return
			}
			err := c.writeWithDeadline(websocket.TextMessage, msg)
			if err != nil {
				log.Printf("connection is broken: %v", err)
				return
			}
		case <-ticker.C:
			err := c.writeWithDeadline(websocket.PingMessage, []byte{})
			if err != nil {
				log.Printf("connection is broken: %v", err)
				return
			}
		}
	}
}

func (c *Client) writeWithDeadline(messageType int, msg []byte) error {
	err := c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(messageType, msg)
}
