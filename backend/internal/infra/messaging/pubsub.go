// Package messaging provides the GCP Pub/Sub implementation of the domain messaging port.
// Construct a PubSubPublisher per topic at the composition root and pass it to
// application services that need to publish events.
package messaging

import (
	"context"
	"fmt"

	"cloud.google.com/go/pubsub/v2"

	domainmsg "github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// PubSubPublisher implements domain/messaging.Publisher using GCP Pub/Sub.
// It holds a reference to a single Pub/Sub publisher (one topic); construct one
// instance per topic at the composition root.
//
// Stop must be called when the publisher is no longer needed to flush and release
// background goroutines started by the Pub/Sub client.
type PubSubPublisher struct {
	publisher *pubsub.Publisher
}

// NewPubSubPublisher constructs a PubSubPublisher for the given GCP project and topic.
// The Pub/Sub client authenticates using Application Default Credentials; in Cloud Run
// this is the service account; locally it uses gcloud credentials.
func NewPubSubPublisher(ctx context.Context, projectID, topicID string) (*PubSubPublisher, error) {
	if projectID == "" {
		return nil, fmt.Errorf("pubsub publisher: projectID is required")
	}
	if topicID == "" {
		return nil, fmt.Errorf("pubsub publisher: topicID is required")
	}

	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("creating pubsub client for project %q: %w", projectID, err)
	}

	return &PubSubPublisher{publisher: client.Publisher(topicID)}, nil
}

// Publish publishes a single message to the configured Pub/Sub topic.
// It blocks until Pub/Sub acknowledges the message or ctx is cancelled.
func (p *PubSubPublisher) Publish(ctx context.Context, msg domainmsg.Message) error {
	result := p.publisher.Publish(ctx, &pubsub.Message{
		Data:       msg.Data,
		Attributes: msg.Attributes,
	})

	if _, err := result.Get(ctx); err != nil {
		return fmt.Errorf("publishing message to pubsub: %w", err)
	}
	return nil
}

// Stop flushes pending messages and releases background goroutines. Call this during
// graceful shutdown.
func (p *PubSubPublisher) Stop() {
	p.publisher.Stop()
}
