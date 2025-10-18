package store

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

// Connect to PostgreSQL and create table if needed
func InitPostgres(connStr string) {
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Println("⚠️  PostgreSQL connection failed:", err)
		db = nil
		return
	}

	if err = db.Ping(); err != nil {
		log.Println("⚠️  PostgreSQL ping failed:", err)
		db = nil
		return
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS leaderboard (
			player TEXT PRIMARY KEY,
			wins INT DEFAULT 0,
			total INT DEFAULT 0,
			last_win TIMESTAMPTZ
		);
	`)
	if err != nil {
		log.Println("⚠️  Could not create leaderboard table:", err)
		db = nil
		return
	}

	log.Println("✅ PostgreSQL initialized and ready (leaderboard)")
}

// Record result in both PostgreSQL and JSON fallback
func RecordResultPG(player string, won bool) {
	if player == "" {
		return
	}
	if db == nil {
		// fallback to JSON file if DB is not ready
		RecordResult(player, won)
		return
	}

	tx, _ := db.Begin()
	defer tx.Commit()

	_, _ = tx.Exec(`
		INSERT INTO leaderboard (player, wins, total, last_win)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (player) DO UPDATE
		SET total = leaderboard.total + 1,
			wins = leaderboard.wins + CASE WHEN $2 > 0 THEN 1 ELSE 0 END,
			last_win = CASE WHEN $2 > 0 THEN $4 ELSE leaderboard.last_win END
	`, player, boolToInt(won), 1, time.Now())

	// keep JSON mirrored as a friendly backup
	RecordResult(player, won)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func GetLeaderboardPG() []*PlayerStats {
	if db == nil {
		return GetLeaderboard()
	}
	rows, err := db.Query(`SELECT player, wins, total, last_win FROM leaderboard ORDER BY wins DESC, total DESC`)
	if err != nil {
		log.Println("⚠️  Leaderboard query failed:", err)
		return GetLeaderboard()
	}
	defer rows.Close()

	var res []*PlayerStats
	for rows.Next() {
		p := PlayerStats{}
		_ = rows.Scan(&p.Player, &p.Wins, &p.Total, &p.LastWin)
		res = append(res, &p)
	}
	return res
}
