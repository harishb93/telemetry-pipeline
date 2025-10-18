package mq

import (
	"bytes"
	"fmt"
	"net/http"
)

// HTTPBroker is a client for connecting to a remote MQ broker via HTTP
type HTTPBroker struct {
	baseURL string
	client  *http.Client
}

// NewHTTPBroker creates a new HTTP broker client
func NewHTTPBroker(baseURL string) *HTTPBroker {
	return &HTTPBroker{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// Publish publishes a message to a topic via HTTP
func (h *HTTPBroker) Publish(topic string, msg Message) error {
	url := fmt.Sprintf("%s/publish/%s", h.baseURL, topic)

	// Send the payload directly as JSON (it's already JSON from the streamer)
	resp, err := h.client.Post(url, "application/json", bytes.NewBuffer(msg.Payload))
	if err != nil {
		return fmt.Errorf("failed to publish to %s: %w", url, err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			// Can't return error from defer, just log
			fmt.Printf("Warning: failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("publish failed with status %d", resp.StatusCode)
	}

	return nil
}

// Subscribe is not implemented for HTTP broker (would require websockets or polling)
func (h *HTTPBroker) Subscribe(topic string) (chan []byte, func(), error) {
	return nil, nil, fmt.Errorf("Subscribe not supported in HTTP broker")
}

// SubscribeWithAck is not implemented for HTTP broker
func (h *HTTPBroker) SubscribeWithAck(topic string) (chan Message, func(), error) {
	return nil, nil, fmt.Errorf("SubscribeWithAck not supported in HTTP broker")
}

// Close closes the HTTP client
func (h *HTTPBroker) Close() {
	// HTTP client doesn't need explicit closing
}

// BrokerInterface defines the interface that both local and HTTP brokers implement
type BrokerInterface interface {
	Publish(topic string, msg Message) error
	Subscribe(topic string) (chan []byte, func(), error)
	SubscribeWithAck(topic string) (chan Message, func(), error)
	Close()
}

// Ensure both brokers implement the interface
var _ BrokerInterface = (*Broker)(nil)
var _ BrokerInterface = (*HTTPBroker)(nil)
