# Component: Messaging Port

Domain port for publishing pipeline events and decoding Pub/Sub push deliveries. GCP Pub/Sub implementation in `infra/messaging`.

## Interfaces

### Publisher

```go
Publish(ctx context.Context, msg Message) error
```

Publishes a `Message{Data []byte, Attributes map[string]string}` to a pre-configured topic. Implementations are safe for concurrent use.

## Data Types

### PushEnvelope

Outer JSON envelope from a GCP Pub/Sub push delivery. Parsed at the worker HTTP handler via `DecodePushEnvelope(body []byte) (PushEnvelope, error)`.

### PushMessage

Embedded message in a push delivery. `DecodeData() ([]byte, error)` base64-decodes the payload.

## Notes

- Package path: `internal/domain/messaging`
- Implementation: `internal/infra/messaging.PubSubPublisher` (uses `cloud.google.com/go/pubsub/v2`).
- OIDC push token verification (P1-BE-8) is separate from push envelope decoding.
