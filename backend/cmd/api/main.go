package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/cors"

	gm "backend/internal/game"
	"backend/internal/kafka"
	"backend/internal/store"
)

type Game struct {
	Grid         [][]gm.Cell
	Player1      string          // Red
	Player2      string          // Yellow (or "BOT")
	Conn1        *websocket.Conn // P1 socket
	Conn2        *websocket.Conn // P2 socket (nil in bot game)
	CurrentTurn  gm.Cell         // gm.Red or gm.Yellow
	LastSeen     map[string]time.Time
	Disconnected map[string]time.Time
	IsOver       bool
	Winner       string
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

var (
	mu            sync.Mutex
	waitingPlayer *Game
	activeGames   = make(map[string]*Game) // gameID -> *Game
)

// ------------------- helpers -------------------

func send(conn *websocket.Conn, msg any) {
	if conn == nil {
		return
	}
	data, _ := json.Marshal(msg)
	_ = conn.WriteMessage(websocket.TextMessage, data)
}

func cellToStr(c gm.Cell) string {
	switch c {
	case gm.Red:
		return "red"
	case gm.Yellow:
		return "yellow"
	default:
		return ""
	}
}

// push full state to this connection (used for reconnect)
func sendGameState(g *Game, conn *websocket.Conn, username string) {
	myColor := "red"
	if username == g.Player2 || (g.Conn2 == conn && g.Player2 != "") {
		myColor = "yellow"
	}
	send(conn, map[string]any{
		"type":   "grid",
		"grid":   g.Grid,
		"turn":   cellToStr(g.CurrentTurn), // whose turn globally
		"color":  myColor,                  // this client's color
		"winner": g.Winner,
	})
}

// tell both players whose turn it is (and each player’s color)
func sendTurn(g *Game) {
	turn := cellToStr(g.CurrentTurn)
	if g.Conn1 != nil {
		send(g.Conn1, map[string]any{"type": "turn", "turn": turn, "color": "red"})
	}
	if g.Conn2 != nil {
		send(g.Conn2, map[string]any{"type": "turn", "turn": turn, "color": "yellow"})
	}
}

// ------------------- websocket entry -------------------

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}

	// keepalive pings
	go keepAlive(conn)

	// read join (first message)
	var join struct {
		Type     string `json:"type"`
		Username string `json:"username"`
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		log.Println("read join failed:", err)
		return
	}
	_ = json.Unmarshal(msg, &join)
	if join.Username == "" {
		join.Username = "Player-" + time.Now().Format("150405")
	}

	// Try to treat as reconnect first
	mu.Lock()
	for id, g := range activeGames {
		if g.IsOver {
			continue
		}
		if g.Player1 == join.Username {
			g.Conn1 = conn
			delete(g.Disconnected, g.Player1)
			g.LastSeen[g.Player1] = time.Now()
			mu.Unlock()
			sendGameState(g, conn, join.Username)
			send(conn, map[string]any{"type": "status", "message": "🔄 Reconnected to game " + id})
			sendTurn(g)
			return
		}
		if g.Player2 == join.Username {
			g.Conn2 = conn
			delete(g.Disconnected, g.Player2)
			g.LastSeen[g.Player2] = time.Now()
			mu.Unlock()
			sendGameState(g, conn, join.Username)
			send(conn, map[string]any{"type": "status", "message": "🔄 Reconnected to game " + id})
			sendTurn(g)
			return
		}
	}
	mu.Unlock()

	// else normal matchmaking
	matchmaking(join.Username, conn)
}

func keepAlive(c *websocket.Conn) {
	for {
		time.Sleep(10 * time.Second)
		deadline := time.Now().Add(2 * time.Second)
		if err := c.WriteControl(websocket.PingMessage, []byte("keepalive"), deadline); err != nil {
			if websocket.IsUnexpectedCloseError(err) ||
				strings.Contains(err.Error(), "use of closed network connection") ||
				strings.Contains(err.Error(), "wsasend") {
				return
			}
			return
		}
	}
}

// ------------------- matchmaking -------------------

func matchmaking(username string, conn *websocket.Conn) {
	mu.Lock()
	defer mu.Unlock()

	// nobody waiting → queue this user
	if waitingPlayer == nil {
		waitingPlayer = &Game{
			Grid:         gm.NewGrid(),
			Player1:      username, // Red
			Conn1:        conn,
			CurrentTurn:  gm.Red,
			LastSeen:     map[string]time.Time{username: time.Now()},
			Disconnected: make(map[string]time.Time),
		}
		send(conn, map[string]any{"type": "status", "message": "Waiting for opponent..."})
		send(conn, map[string]any{"type": "grid", "grid": waitingPlayer.Grid})
		send(conn, map[string]any{"type": "turn", "turn": "red", "color": "red"})

		// 10s fallback → bot
		go func(g *Game) {
			time.Sleep(10 * time.Second)
			mu.Lock()
			defer mu.Unlock()
			if waitingPlayer == g {
				waitingPlayer = nil
				gameID := "bot-" + time.Now().Format("150405")
				activeGames[gameID] = g
				send(g.Conn1, map[string]any{"type": "status", "message": "No opponent found. Starting vs Competitive Bot!"})
				go startBotGame(gameID, g)
			}
		}(waitingPlayer)
		return
	}

	// match found
	opponent := waitingPlayer
	waitingPlayer = nil

	gameID := "match-" + time.Now().Format("150405")
	opponent.Player2 = username       // Yellow
	opponent.Conn2 = conn
	opponent.LastSeen[username] = time.Now()
	opponent.CurrentTurn = gm.Red
	activeGames[gameID] = opponent

	send(opponent.Conn1, map[string]any{"type": "status", "message": "Opponent found: " + username + " — your turn!"})
	send(opponent.Conn2, map[string]any{"type": "status", "message": "Match vs " + opponent.Player1 + " — wait for your turn."})
	send(opponent.Conn1, map[string]any{"type": "grid", "grid": opponent.Grid})
	send(opponent.Conn2, map[string]any{"type": "grid", "grid": opponent.Grid})
	sendTurn(opponent)

	go startPlayerVsPlayer(gameID, opponent)
}

// ------------------- gameplay: vs bot -------------------

func startBotGame(id string, g *Game) {
	// human is Red, bot is Yellow
	g.CurrentTurn = gm.Red
	sendTurn(g)

	for {
		// HUMAN MOVE
		_, raw, err := g.Conn1.ReadMessage()
		if err != nil {
			markDisconnected(id, g.Player1)
			return
		}
		var move struct {
			Type   string `json:"type"`
			Column int    `json:"column"`
		}
		if json.Unmarshal(raw, &move) != nil || move.Type != "drop" {
			continue
		}
		if g.CurrentTurn != gm.Red {
			continue // not your turn
		}
		if !gm.DropDisc(g.Grid, move.Column, gm.Red) {
			continue
		}
		send(g.Conn1, map[string]any{"type": "grid", "grid": g.Grid})
		if w, ok := gm.CheckWin(g.Grid); ok {
			finishGameSingle(g, w)
			return
		}

		// BOT MOVE
		g.CurrentTurn = gm.Yellow
		sendTurn(g)

		time.Sleep(700 * time.Millisecond)
		col := gm.BotMove(g.Grid)
		_ = gm.DropDisc(g.Grid, col, gm.Yellow)
		send(g.Conn1, map[string]any{"type": "grid", "grid": g.Grid})
		if w, ok := gm.CheckWin(g.Grid); ok {
			finishGameSingle(g, w)
			return
		}

		// back to human
		g.CurrentTurn = gm.Red
		sendTurn(g)
	}
}

// ------------------- gameplay: PvP -------------------

func startPlayerVsPlayer(id string, g *Game) {
	// start with Red (P1)
	g.CurrentTurn = gm.Red
	sendTurn(g)

	currentConn := g.Conn1
	currentColor := gm.Red

	for {
		_, raw, err := currentConn.ReadMessage()
		if err != nil {
			if currentConn == g.Conn1 {
				markDisconnected(id, g.Player1)
			} else {
				markDisconnected(id, g.Player2)
			}
			return
		}
		var move struct {
			Type   string `json:"type"`
			Column int    `json:"column"`
		}
		if json.Unmarshal(raw, &move) != nil || move.Type != "drop" {
			continue
		}
		// turn guard
		if g.CurrentTurn != currentColor {
			continue
		}
		if !gm.DropDisc(g.Grid, move.Column, currentColor) {
			continue
		}

		// push board
		if g.Conn1 != nil {
			send(g.Conn1, map[string]any{"type": "grid", "grid": g.Grid})
		}
		if g.Conn2 != nil {
			send(g.Conn2, map[string]any{"type": "grid", "grid": g.Grid})
		}

		// check win
		if w, ok := gm.CheckWin(g.Grid); ok {
			finishGamePVP(g, w)
			return
		}

		// switch turn
		if currentConn == g.Conn1 {
			currentConn = g.Conn2
			currentColor = gm.Yellow
			g.CurrentTurn = gm.Yellow
		} else {
			currentConn = g.Conn1
			currentColor = gm.Red
			g.CurrentTurn = gm.Red
		}
		sendTurn(g)
	}
}

// ------------------- disconnect + auto-forfeit -------------------

func markDisconnected(gameID, username string) {
	mu.Lock()
	defer mu.Unlock()

	g, ok := activeGames[gameID]
	if !ok || g.IsOver {
		return
	}
	g.Disconnected[username] = time.Now()
	log.Printf("⚠️ %s disconnected from game %s", username, gameID)
}

func startDisconnectMonitor() {
	go func() {
		for {
			time.Sleep(5 * time.Second)
			now := time.Now()
			mu.Lock()
			for id, g := range activeGames {
				if g.IsOver {
					continue
				}
				for user, t := range g.Disconnected {
					if now.Sub(t) > 30*time.Second {
						var winner string
						if user == g.Player1 {
							if g.Player2 != "" {
								winner = g.Player2
							} else {
								winner = "BOT"
							}
						} else {
							winner = g.Player1
						}
						g.IsOver = true
						g.Winner = winner
						delete(activeGames, id)
						log.Printf("🏁 Game %s: %s forfeited — %s wins", id, user, winner)
						// Optionally: notify the still-connected side
						if g.Conn1 != nil && g.Player1 == winner {
							send(g.Conn1, map[string]any{"type": "result", "grid": g.Grid, "result": "Opponent forfeited — You win!"})
						}
						if g.Conn2 != nil && g.Player2 == winner {
							send(g.Conn2, map[string]any{"type": "result", "grid": g.Grid, "result": "Opponent forfeited — You win!"})
						}
					}
				}
			}
			mu.Unlock()
		}
	}()
}

// ------------------- finish + record -------------------

func finishGameSingle(g *Game, w gm.Cell) {
	g.IsOver = true
	var result string
	winFlag := false
	switch w {
	case gm.Red:
		result = "You win!"
		g.Winner = g.Player1
		winFlag = true
		store.RecordResultPG(g.Player1, true)
	case gm.Yellow:
		result = "Opponent wins!"
		g.Winner = "BOT"
		store.RecordResultPG(g.Player1, false)
	default:
		result = "Draw"
		g.Winner = ""
	}

	send(g.Conn1, map[string]any{"type": "result", "grid": g.Grid, "result": result})

	// Kafka emit
	kafka.Publish(map[string]any{
		"type":     "result",
		"player":   g.Player1,
		"opponent": "BOT",
		"winner":   g.Winner,
		"win":      winFlag,
	})
}

func finishGamePVP(g *Game, w gm.Cell) {
	g.IsOver = true

	var res1, res2 string
	switch w {
	case gm.Red:
		res1, res2 = "You win!", "You lose!"
		g.Winner = g.Player1
		store.RecordResultPG(g.Player1, true)
		store.RecordResultPG(g.Player2, false)
	case gm.Yellow:
		res1, res2 = "You lose!", "You win!"
		g.Winner = g.Player2
		store.RecordResultPG(g.Player1, false)
		store.RecordResultPG(g.Player2, true)
	default:
		res1, res2 = "Draw", "Draw"
		g.Winner = ""
	}

	if g.Conn1 != nil {
		send(g.Conn1, map[string]any{"type": "result", "grid": g.Grid, "result": res1})
	}
	if g.Conn2 != nil {
		send(g.Conn2, map[string]any{"type": "result", "grid": g.Grid, "result": res2})
	}

	// Kafka emit
	kafka.Publish(map[string]any{
		"type":     "result",
		"player":   g.Player1,
		"opponent": g.Player2,
		"winner":   g.Winner,
	})
}

// ------------------- main -------------------

func main() {
	rand.Seed(time.Now().UnixNano())

	// init Kafka + Postgres + JSON fallback
	kafka.InitProducer()
	defer kafka.CloseProducer()

	store.InitPostgres("postgres://postgres:post@localhost:5432/connect4?sslmode=disable")
	if err := store.LoadLeaderboard(); err != nil {
		log.Println("⚠️ Failed to load JSON leaderboard:", err)
	}
	log.Println("🏆 Leaderboard loaded successfully")

	startDisconnectMonitor()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", handleWS)
	mux.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(store.GetLeaderboardPG())
	})

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowCredentials: true,
	}).Handler(mux)

	log.Println("✅ Running on :8080")
	_ = http.ListenAndServe(":8080", handler)
	_ = context.Background()
}
