// Package messaging defines the messaging port for publishing pipeline events to Pub/Sub.
// The Publisher interface is consumed by app-layer use cases; the GCP Pub/Sub implementation
// lives in infra/messaging.
package messaging

import "context"

// Message is a Pub/Sub message payload. Data contains the raw bytes; Attributes
// carries optional string metadata (e.g. content type, source listing id).
type Message struct {
	// Data is the raw message payload.
	Data []byte
	// Attributes are optional metadata key-value pairs attached to the message.
	Attributes map[string]string
}

// Publisher publishes a Message to a pre-configured topic.
// Implementations must be safe for concurrent use.
type Publisher interface {
	Publish(ctx context.Context, msg Message) error
}
