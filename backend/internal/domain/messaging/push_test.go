package messaging_test

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// AC: push decoder parses a Pub/Sub push envelope.

func TestDecodePushEnvelope(t *testing.T) {
	t.Parallel()

	payload := []byte(`{"listing_id":"raw-123"}`)
	encoded := base64.StdEncoding.EncodeToString(payload)

	cases := []struct {
		name        string
		body        []byte
		wantSub     string
		wantMsgID   string
		wantDecoded []byte
		wantErr     bool
	}{
		{
			name: "parses a well-formed push envelope",
			body: mustMarshal(t, map[string]interface{}{
				"subscription": "projects/my-project/subscriptions/listing-extract-sub",
				"message": map[string]interface{}{
					"data":        encoded,
					"messageId":   "msg-abc",
					"publishTime": "2026-01-01T00:00:00Z",
					"attributes":  map[string]string{"source": "scrape-worker"},
				},
			}),
			wantSub:     "projects/my-project/subscriptions/listing-extract-sub",
			wantMsgID:   "msg-abc",
			wantDecoded: payload,
		},
		{
			name:    "returns error on invalid JSON",
			body:    []byte(`{not valid json`),
			wantErr: true,
		},
		{
			name: "returns error when message data is not valid base64",
			body: mustMarshal(t, map[string]interface{}{
				"message": map[string]interface{}{
					"data": "!!!not-base64!!!",
				},
			}),
			// DecodePushEnvelope succeeds; DecodeData fails.
			wantErr: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			env, err := messaging.DecodePushEnvelope(tc.body)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("DecodePushEnvelope() returned nil error; want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodePushEnvelope() unexpected error: %v", err)
			}

			if tc.wantSub != "" && env.Subscription != tc.wantSub {
				t.Errorf("Subscription = %q; want %q", env.Subscription, tc.wantSub)
			}
			if tc.wantMsgID != "" && env.Message.MessageID != tc.wantMsgID {
				t.Errorf("MessageID = %q; want %q", env.Message.MessageID, tc.wantMsgID)
			}
			if tc.wantDecoded != nil {
				got, err := env.Message.DecodeData()
				if err != nil {
					t.Fatalf("DecodeData() unexpected error: %v", err)
				}
				if string(got) != string(tc.wantDecoded) {
					t.Errorf("DecodeData() = %q; want %q", got, tc.wantDecoded)
				}
			}
		})
	}
}

func TestPushMessage_DecodeData_InvalidBase64(t *testing.T) {
	t.Parallel()

	msg := messaging.PushMessage{Data: "!!!not-base64!!!"}
	_, err := msg.DecodeData()
	if err == nil {
		t.Fatal("DecodeData() returned nil error for invalid base64; want non-nil")
	}
}

func mustMarshal(t *testing.T, v interface{}) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return b
}
