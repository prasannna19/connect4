package ws

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type GameSession struct {
	GameID       string
	Player1      string
	Player2      string
	Grid         [6][7]int
	LastSeen     map[string]time.Time
	Disconnected map[string]time.Time
	IsOver       bool
	Winner       string
}

var (
	activeGames = make(map[string]*GameSession)
	mu          sync.Mutex
)

// Create new game with unique ID
func NewGame(p1, p2 string) *GameSession {
	mu.Lock()
	defer mu.Unlock()

	id := uuid.NewString()
	game := &GameSession{
		GameID:       id,
		Player1:      p1,
		Player2:      p2,
		LastSeen:     map[string]time.Time{p1: time.Now(), p2: time.Now()},
		Disconnected: make(map[string]time.Time),
	}
	activeGames[id] = game
	log.Printf("🎮 Game created: %s (%s vs %s)", id, p1, p2)
	return game
}

// Mark player as disconnected
func MarkDisconnected(gameID, username string) {
	mu.Lock()
	defer mu.Unlock()
	if g, ok := activeGames[gameID]; ok && !g.IsOver {
		g.Disconnected[username] = time.Now()
		log.Printf("⚠️ %s disconnected from game %s", username, gameID)
	}
}

// Mark player as reconnected
func MarkReconnected(gameID, username string) bool {
	mu.Lock()
	defer mu.Unlock()
	if g, ok := activeGames[gameID]; ok && !g.IsOver {
		delete(g.Disconnected, username)
		g.LastSeen[username] = time.Now()
		log.Printf("🔄 %s reconnected to game %s", username, gameID)
		return true
	}
	return false
}

// Background monitor: auto-forfeit after 30s
func StartDisconnectMonitor() {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			now := time.Now()

			mu.Lock()
			for id, g := range activeGames {
				if g.IsOver {
					continue
				}
				for player, t := range g.Disconnected {
					if now.Sub(t) > 30*time.Second {
						winner := g.Player1
						if player == g.Player1 {
							winner = g.Player2
						}
						g.IsOver = true
						g.Winner = winner
						delete(activeGames, id)

						log.Printf("🏁 Game %s: %s forfeited after 30s — %s wins", id, player, winner)

						// You can also record in DB here
						// store.RecordResult(winner, true)
					}
				}
			}
			mu.Unlock()
		}
	}()
}
