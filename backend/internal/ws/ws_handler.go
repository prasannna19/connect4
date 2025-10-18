package ws

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// global upgrader for HTTP → WebSocket
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for testing
	},
}

// global hub (shared between all players)
var hub = NewHub()

// Handle incoming WebSocket connection
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	username := r.URL.Query().Get("username")
	if username == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("❌ WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		Conn:     conn,
		Username: username,
		LastSeen: time.Now(),
	}

	log.Printf("🔌 %s connected", username)
	hub.Join(client)

	// start read loop for this connection
	go handleRead(client)
}

// Read messages from WebSocket (loop)
func handleRead(c *Client) {
	defer func() {
		log.Printf("⚠️ %s disconnected", c.Username)
		hub.MarkDisconnected(c.Username)
		c.Conn.Close()
	}()

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			log.Printf("read error from %s: %v", c.Username, err)
			break
		}

		c.LastSeen = time.Now()
		fmt.Printf("📩 %s → %s\n", c.Username, string(msg))
		// here you can decode JSON messages for moves, etc.
	}
}
