package mq

import (
	"bytes"
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
	defer resp.Body.Close()
	
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
	// For HTTP-based MQ, we need to implement polling or use a different approach
	// For now, return an error to indicate this needs a different implementation
	return nil, nil, fmt.Errorf("SubscribeWithAck not supported via HTTP - requires direct broker connection")
}

// Close closes the client
func (d *DirectBrokerClient) Close() {
	// HTTP client doesn't need explicit closing
}

// Ensure DirectBrokerClient implements BrokerInterface
var _ BrokerInterface = (*DirectBrokerClient)(nil)