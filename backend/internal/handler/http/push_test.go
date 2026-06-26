package handler_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
	handler "github.com/g-trinh/job-tendencies/internal/handler/http"
)

// fakeScrapingDispatcher records the last message it received.
type fakeScrapingDispatcher struct {
	got    *messaging.Message
	errOut error
}

func (f *fakeScrapingDispatcher) HandleScrapeTick(_ context.Context, msg messaging.Message) error {
	if f.errOut != nil {
		return f.errOut
	}
	f.got = &msg
	return nil
}

// fakeExtractionDispatcher records the last message it received.
type fakeExtractionDispatcher struct {
	got    *messaging.Message
	errOut error
}

func (f *fakeExtractionDispatcher) HandleListingExtract(_ context.Context, msg messaging.Message) error {
	if f.errOut != nil {
		return f.errOut
	}
	f.got = &msg
	return nil
}

func buildPushBody(t *testing.T, data []byte) []byte {
	t.Helper()
	envelope := map[string]interface{}{
		"subscription": "projects/p/subscriptions/s",
		"message": map[string]interface{}{
			"data":        base64.StdEncoding.EncodeToString(data),
			"messageId":   "msg-1",
			"publishTime": "2026-01-01T00:00:00Z",
		},
	}
	b, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("building push body: %v", err)
	}
	return b
}

func TestPushScrapeTick(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name       string
		body       []byte
		dispatcher *fakeScrapingDispatcher
		wantStatus int
		wantData   []byte
	}{
		{
			name:       "valid push envelope dispatches and returns 204",
			body:       buildPushBody(t, []byte(`{"run_id":"run-1"}`)),
			dispatcher: &fakeScrapingDispatcher{},
			wantStatus: http.StatusNoContent,
			wantData:   []byte(`{"run_id":"run-1"}`),
		},
		{
			name:       "invalid JSON body returns 400",
			body:       []byte(`{not json`),
			dispatcher: &fakeScrapingDispatcher{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Post("/push/scrape-tick", handler.PushScrapeTick(tc.dispatcher, logger))

			req := httptest.NewRequest(http.MethodPost, "/push/scrape-tick", bytes.NewReader(tc.body))
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}

			if tc.wantData != nil {
				if tc.dispatcher.got == nil {
					t.Fatal("dispatcher.got is nil; want dispatched message")
				}
				if string(tc.dispatcher.got.Data) != string(tc.wantData) {
					t.Errorf("dispatched data = %q; want %q", tc.dispatcher.got.Data, tc.wantData)
				}
			}
		})
	}
}

func TestPushListingExtract(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	cases := []struct {
		name       string
		body       []byte
		dispatcher *fakeExtractionDispatcher
		wantStatus int
		wantData   []byte
	}{
		{
			name:       "valid push envelope dispatches and returns 204",
			body:       buildPushBody(t, []byte(`{"raw_listing_id":"rl-1"}`)),
			dispatcher: &fakeExtractionDispatcher{},
			wantStatus: http.StatusNoContent,
			wantData:   []byte(`{"raw_listing_id":"rl-1"}`),
		},
		{
			name:       "invalid JSON body returns 400",
			body:       []byte(`not json`),
			dispatcher: &fakeExtractionDispatcher{},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			r := chi.NewRouter()
			r.Post("/push/listing-extract", handler.PushListingExtract(tc.dispatcher, logger))

			req := httptest.NewRequest(http.MethodPost, "/push/listing-extract", bytes.NewReader(tc.body))
			rec := httptest.NewRecorder()

			r.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Errorf("status = %d; want %d (body: %s)", rec.Code, tc.wantStatus, rec.Body.String())
			}

			if tc.wantData != nil {
				if tc.dispatcher.got == nil {
					t.Fatal("dispatcher.got is nil; want dispatched message")
				}
				if string(tc.dispatcher.got.Data) != string(tc.wantData) {
					t.Errorf("dispatched data = %q; want %q", tc.dispatcher.got.Data, tc.wantData)
				}
			}
		})
	}
}
