package examples

import (
	"fmt"
	"log"
	"time"

	"github.com/harishb93/telemetry-pipeline/internal/mq"
)

func main() {
	// Create broker with persistence enabled
	config := mq.DefaultBrokerConfig()
	config.PersistenceEnabled = true
	config.PersistenceDir = "/tmp/mq-data"
	config.AckTimeout = 10 * time.Second

	broker := mq.NewBroker(config)
	defer broker.Close()

	// Start admin server in background
	go func() {
		log.Println("Starting admin server on port 8080...")
		log.Println("Available endpoints:")
		log.Println("  GET /health - Health check")
		log.Println("  GET /stats - Overall broker statistics")
		log.Println("  GET /stats/{topic} - Topic-specific statistics")
		
		if err := broker.StartAdminServer("8080"); err != nil {
			log.Printf("Admin server error: %v", err)
		}
	}()

	// Example usage
	topic := "gpu-telemetry"

	// Subscribe to the topic
	ch, unsubscribe, err := broker.Subscribe(topic)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	defer unsubscribe()

	// Start message consumer
	go func() {
		for payload := range ch {
			fmt.Printf("Received message: %s\n", string(payload))
		}
	}()

	// Subscribe with acknowledgment
	ackCh, unsubscribeAck, err := broker.SubscribeWithAck(topic)
	if err != nil {
		log.Fatalf("Failed to subscribe with ack: %v", err)
	}
	defer unsubscribeAck()

	// Start acknowledgment-aware consumer
	go func() {
		for msg := range ackCh {
			fmt.Printf("Received message with ack: %s\n", string(msg.Payload))
			// Acknowledge the message
			msg.Ack()
		}
	}()

	// Publish some messages
	for i := 0; i < 5; i++ {
		msg := mq.Message{
			Payload: []byte(fmt.Sprintf("GPU telemetry message %d", i)),
			Ack:     func() {}, // Will be overridden by Publish
		}

		err := broker.Publish(topic, msg)
		if err != nil {
			log.Printf("Failed to publish message %d: %v", i, err)
		}

		time.Sleep(1 * time.Second)
	}

	// Print statistics
	stats := broker.GetStats()
	fmt.Printf("Broker statistics: %+v\n", stats)

	// Keep the server running
	fmt.Println("Broker is running. Check admin endpoints at:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl http://localhost:8080/stats")
	fmt.Println("  curl http://localhost:8080/stats/gpu-telemetry")
	
	time.Sleep(30 * time.Second)
}