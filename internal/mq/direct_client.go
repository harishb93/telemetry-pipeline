package mq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// DirectBrokerClient connects directly to the MQ service's broker
type DirectBrokerClient struct {
	baseURL string
	client  *http.Client
}

// NewDirectBrokerClient creates a new direct broker client
func NewDirectBrokerClient(baseURL string) *DirectBrokerClient {
	return &DirectBrokerClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Publish publishes a message to a topic
func (d *DirectBrokerClient) Publish(topic string, msg Message) error {
	url := fmt.Sprintf("%s/publish/%s", d.baseURL, topic)

	resp, err := d.client.Post(url, "application/json", bytes.NewBuffer(msg.Payload))
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed with status %d", resp.StatusCode)
	}

	return nil
}

// Subscribe creates a basic subscription (not implemented for HTTP)
func (d *DirectBrokerClient) Subscribe(topic string) (chan []byte, func(), error) {
	return nil, nil, fmt.Errorf("Subscribe not supported via HTTP - use direct broker connection")
}

// SubscribeWithAck creates a subscription with acknowledgment using polling
func (d *DirectBrokerClient) SubscribeWithAck(topic string) (chan Message, func(), error) {
	msgCh := make(chan Message, 100)
	stopCh := make(chan struct{})

	// Start polling goroutine
	go d.pollMessages(topic, msgCh, stopCh)

	unsubscribe := func() {
		close(stopCh)
		close(msgCh)
	}

	return msgCh, unsubscribe, nil
}

// pollMessages continuously polls for messages from the MQ service
func (d *DirectBrokerClient) pollMessages(topic string, msgCh chan Message, stopCh chan struct{}) {
	ticker := time.NewTicker(1 * time.Second) // Poll every second
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			// Poll for messages
			url := fmt.Sprintf("%s/consume/%s?timeout=2&max_messages=10", d.baseURL, topic)
			resp, err := d.client.Get(url)
			if err != nil {
				// Log error but continue polling
				continue
			}

			if resp.StatusCode == http.StatusOK {
				var result struct {
					Messages []struct {
						Payload   string `json:"payload"`
						Timestamp string `json:"timestamp"`
					} `json:"messages"`
				}

				if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
					// Send messages to channel
					for _, msg := range result.Messages {
						select {
						case msgCh <- Message{
							Payload: []byte(msg.Payload),
							Ack:     func() {}, // No-op ack since we already consumed from HTTP
						}:
						case <-stopCh:
							_ = resp.Body.Close()
							return
						default:
							// Channel full, skip message
						}
					}
				}
			}
			_ = resp.Body.Close()
		}
	}
}

// Close closes the client
func (d *DirectBrokerClient) Close() {
	// HTTP client doesn't need explicit closing
}

// Ensure DirectBrokerClient implements BrokerInterface
var _ BrokerInterface = (*DirectBrokerClient)(nil)
