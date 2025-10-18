package store

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// PlayerStats represents a player's game record
type PlayerStats struct {
	Player  string    `json:"Player"`
	Wins    int       `json:"Wins"`
	Total   int       `json:"Total"`
	LastWin time.Time `json:"LastWin"`
}

var (
	Leaderboard = make(map[string]*PlayerStats)
	mu          sync.Mutex
	DB          *sql.DB
	filePath    = "leaderboard.json"
)

// ✅ Initialize PostgreSQL connection
func InitDB() {
	connStr := "postgres://postgres:post@localhost:5432/connect4?sslmode=disable"
	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("❌ Database connection failed:", err)
	}
	if err = DB.Ping(); err != nil {
		log.Fatal("❌ Cannot reach PostgreSQL:", err)
	}
	log.Println("✅ PostgreSQL initialized and ready (leaderboard)")
}

// ✅ Load leaderboard data from file (call once on app startup)
func LoadLeaderboard() error {
	mu.Lock()
	defer mu.Unlock()

	f, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("ℹ️ No leaderboard file found — starting fresh")
			return nil
		}
		return err
	}
	defer f.Close()

	var data []*PlayerStats
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return err
	}

	Leaderboard = make(map[string]*PlayerStats)
	for _, p := range data {
		Leaderboard[p.Player] = p
	}
	log.Println("🏆 Leaderboard loaded successfully")
	return nil
}

// ✅ Save leaderboard data to file
func saveLeaderboard() error {
	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	var data []*PlayerStats
	for _, v := range Leaderboard {
		data = append(data, v)
	}
	return json.NewEncoder(f).Encode(data)
}

// ✅ Record game result (update memory + persist JSON)
func RecordResult(player string, won bool) {
	if player == "" {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	stats, exists := Leaderboard[player]
	if !exists {
		stats = &PlayerStats{Player: player}
		Leaderboard[player] = stats
	}

	stats.Total++
	if won {
		stats.Wins++
		stats.LastWin = time.Now()
	}

	// Save to file
	if err := saveLeaderboard(); err != nil {
		log.Println("⚠️ Failed to save leaderboard:", err)
	}
}

// ✅ GetLeaderboard returns the in-memory leaderboard
func GetLeaderboard() []*PlayerStats {
	mu.Lock()
	defer mu.Unlock()

	var res []*PlayerStats
	for _, v := range Leaderboard {
		res = append(res, v)
	}
	return res
}
