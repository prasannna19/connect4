package ws

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"game/backend/internal/game"
)

// 🧠 Represents a connected player
type Client struct {
	Conn     *websocket.Conn
	Username string
	GameID   string
	Color    game.Cell // Red or Yellow
	LastSeen time.Time
}

// 🧩 Represents the game manager (Hub)
type Hub struct {
	waiting *Client
	active  map[string]*Match
	mu      sync.Mutex
}

// 🎯 Represents a single match
type Match struct {
	ID          string
	Player1     *Client
	Player2     *Client // may be nil if playing bot
	Board       *game.Board
	LastSeen    map[string]time.Time
	Disconnected map[string]time.Time
	IsOver      bool
	Winner      string
}

// 🏁 Create a new Hub instance
func NewHub() *Hub {
	h := &Hub{active: make(map[string]*Match)}
	go h.disconnectMonitor() // 🔁 start background thread
	return h
}

// 🧩 Handle a new connection request
func (h *Hub) Join(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 🧠 Check if this player is reconnecting to an active game
	for _, match := range h.active {
		if match.Player1 != nil && match.Player1.Username == client.Username {
			match.Player1.Conn = client.Conn
			delete(match.Disconnected, client.Username)
			match.LastSeen[client.Username] = time.Now()
			log.Printf("🔄 %s reconnected to game %s", client.Username, match.ID)
			return
		}
		if match.Player2 != nil && match.Player2.Username == client.Username {
			match.Player2.Conn = client.Conn
			delete(match.Disconnected, client.Username)
			match.LastSeen[client.Username] = time.Now()
			log.Printf("🔄 %s reconnected to game %s", client.Username, match.ID)
			return
		}
	}

	// 🧍 If someone is waiting, start a match
	if h.waiting != nil {
		matchID := fmt.Sprintf("match-%d", time.Now().UnixNano())
		match := &Match{
			ID:          matchID,
			Player1:     h.waiting,
			Player2:     client,
			Board:       game.NewBoard(),
			LastSeen:    map[string]time.Time{h.waiting.Username: time.Now(), client.Username: time.Now()},
			Disconnected: make(map[string]time.Time),
		}
		h.active[matchID] = match
		h.waiting = nil
		fmt.Printf("🎮 Started match %s: %s vs %s\n", matchID, match.Player1.Username, match.Player2.Username)
		return
	}

	// ⏳ No one waiting → queue player
	h.waiting = client
	fmt.Printf("⏳ %s waiting for opponent...\n", client.Username)

	// 🤖 After 10s, auto-start bot game
	go func(c *Client) {
		time.Sleep(10 * time.Second)
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.waiting == c {
			h.waiting = nil
			matchID := fmt.Sprintf("bot-%d", time.Now().UnixNano())
			match := &Match{
				ID:          matchID,
				Player1:     c,
				Player2:     nil, // bot
				Board:       game.NewBoard(),
				LastSeen:    map[string]time.Time{c.Username: time.Now()},
				Disconnected: make(map[string]time.Time),
			}
			h.active[matchID] = match
			fmt.Printf("🤖 Started match %s: %s vs BOT\n", matchID, c.Username)
		}
	}(client)
}

// ⚙️ Bot move logic
func (m *Match) BotTurn() {
	if m.Player2 == nil { // if it's a bot game
		bot := game.NewBot(game.Yellow) // bot is Yellow by default
		move := bot.ChooseMove(m.Board)
		if move == -1 {
			return
		}
		m.Board.Drop(move)
	}
}

// ⚠️ Mark player disconnected
func (h *Hub) MarkDisconnected(username string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, m := range h.active {
		if m.IsOver {
			continue
		}
		if m.Player1 != nil && m.Player1.Username == username {
			m.Disconnected[username] = time.Now()
			log.Printf("⚠️ %s disconnected from game %s", username, m.ID)
		}
		if m.Player2 != nil && m.Player2.Username == username {
			m.Disconnected[username] = time.Now()
			log.Printf("⚠️ %s disconnected from game %s", username, m.ID)
		}
	}
}

// 🔁 Background thread: forfeits after 30s
func (h *Hub) disconnectMonitor() {
	for {
		time.Sleep(5 * time.Second)
		h.mu.Lock()
		now := time.Now()
		for id, m := range h.active {
			if m.IsOver {
				continue
			}
			for user, t := range m.Disconnected {
				if now.Sub(t) > 30*time.Second {
					// find opponent
					var winner string
					if m.Player1 != nil && m.Player1.Username == user {
						if m.Player2 != nil {
							winner = m.Player2.Username
						} else {
							winner = "BOT"
						}
					} else {
						winner = m.Player1.Username
					}
					m.IsOver = true
					m.Winner = winner
					delete(h.active, id)
					log.Printf("🏁 Game %s: %s forfeited — %s wins", id, user, winner)
				}
			}
		}
		h.mu.Unlock()
	}
}
