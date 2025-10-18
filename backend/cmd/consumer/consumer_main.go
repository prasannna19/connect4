package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	_ "github.com/lib/pq"
	"github.com/segmentio/kafka-go"
)

type AnalyticsEvent struct {
	Type         string `json:"type"`
	Player       string `json:"player"`
	Opponent     string `json:"opponent"`
	ColumnPlayed int    `json:"column_played"`
	Winner       string `json:"winner"`
	ResultText   string `json:"result_text"`
	Source       string `json:"source"`
}

func main() {
	// ✅ Connect to PostgreSQL
	connStr := "postgres://postgres:post@localhost:5432/connect4?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ DB connection error: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("❌ DB ping error: %v", err)
	}
	log.Println("✅ Connected to PostgreSQL (analytics)")

	// Ensure table exists
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS analytics_events (
			id SERIAL PRIMARY KEY,
			event_type TEXT,
			player TEXT,
			opponent TEXT,
			column_played INT,
			winner TEXT,
			result_text TEXT,
			source TEXT,
			created_at TIMESTAMPTZ DEFAULT now()
		);
	`)
	if err != nil {
		log.Fatalf("❌ Could not create analytics_events: %v", err)
	}

	// ✅ Kafka Consumer
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "game-events",
		GroupID: "analytics-service",
	})
	defer r.Close()

	log.Println("📡 Kafka consumer started — listening for game events...")

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			log.Printf("❌ Kafka read error: %v", err)
			continue
		}

		log.Printf("📊 Event received → %s", string(m.Value))

		var e AnalyticsEvent
		if err := json.Unmarshal(m.Value, &e); err != nil {
			log.Println("❌ JSON parse error:", err)
			continue
		}

		_, err = db.Exec(`
			INSERT INTO analytics_events (event_type, player, opponent, column_played, winner, result_text, source)
			VALUES ($1,$2,$3,$4,$5,$6,$7)
		`, e.Type, e.Player, e.Opponent, e.ColumnPlayed, e.Winner, e.ResultText, e.Source)

		if err != nil {
			log.Println("❌ DB insert error:", err)
		} else {
			log.Printf("💾 Saved event: %+v\n", e)
		}
	}
}
