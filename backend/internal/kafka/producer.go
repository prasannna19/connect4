package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

var writer *kafka.Writer

// Initialize Kafka producer
func InitProducer() {
	writer = &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "game-events",
		Balancer: &kafka.LeastBytes{},
	}
	log.Println("✅ Kafka producer initialized")
}

// Publish an event to Kafka
func Publish(event interface{}) {
	msg, err := json.Marshal(event)
	if err != nil {
		log.Println("❌ Error marshaling event:", err)
		return
	}
	if writer == nil {
		log.Println("⚠️ Kafka writer not initialized")
		return
	}
	err = writer.WriteMessages(context.Background(), kafka.Message{Value: msg})
	if err != nil {
		log.Println("❌ Failed to send message:", err)
	} else {
		log.Println("📤 Sent event to Kafka:", string(msg))
	}
}

// CloseProducer safely closes the Kafka writer on shutdown
func CloseProducer() {
	if writer != nil {
		_ = writer.Close()
		log.Println("🧹 Kafka producer closed")
	}
}
