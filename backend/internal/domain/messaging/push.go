package messaging

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// PushEnvelope is the outer JSON envelope delivered by a GCP Pub/Sub push subscription.
// Cloud Run receives this on the worker's HTTP endpoint; the worker passes it to
// DecodePushEnvelope to extract and decode the message payload.
//
// Reference: https://cloud.google.com/pubsub/docs/push#receiving_messages
type PushEnvelope struct {
	Message      PushMessage `json:"message"`
	Subscription string      `json:"subscription"`
}

// PushMessage is the Pub/Sub message embedded in a push delivery.
type PushMessage struct {
	// Data is the base64-encoded message payload. Use DecodeData to get raw bytes.
	Data        string            `json:"data"`
	Attributes  map[string]string `json:"attributes"`
	MessageID   string            `json:"messageId"`
	PublishTime string            `json:"publishTime"`
}

// DecodeData returns the raw (base64-decoded) message payload.
func (m PushMessage) DecodeData() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(m.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding push message data: %w", err)
	}
	return b, nil
}

// DecodePushEnvelope parses raw JSON bytes into a PushEnvelope.
// Use this at the HTTP handler boundary to extract the Pub/Sub push payload.
func DecodePushEnvelope(body []byte) (PushEnvelope, error) {
	var env PushEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return PushEnvelope{}, fmt.Errorf("decoding push envelope: %w", err)
	}
	return env, nil
}
