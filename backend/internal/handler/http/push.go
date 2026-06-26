package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"

	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// ScrapingDispatcher dispatches a decoded scrape.tick Pub/Sub message to the
// scraping application service. Defined here (consumer) rather than in the app
// package so that the handler package owns the contract.
type ScrapingDispatcher interface {
	HandleScrapeTick(ctx context.Context, msg messaging.Message) error
}

// ExtractionDispatcher dispatches a decoded listing.extract Pub/Sub message to
// the extraction application service.
type ExtractionDispatcher interface {
	HandleListingExtract(ctx context.Context, msg messaging.Message) error
}

// PushScrapeTick handles POST /push/scrape-tick. The OIDC middleware runs before
// this handler and has already verified the bearer token. This handler:
//
//  1. Reads and parses the Pub/Sub push envelope.
//  2. Base64-decodes the message data.
//  3. Dispatches to the scraping application service.
//  4. Returns 204 on success so Pub/Sub considers the message acknowledged.
//
// Any non-2xx response causes Pub/Sub to redeliver with exponential backoff.
func PushScrapeTick(dispatcher ScrapingDispatcher, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.ErrorContext(r.Context(), "reading push body", "err", err)
			http.Error(w, "bad request: cannot read body", http.StatusBadRequest)
			return
		}

		env, err := messaging.DecodePushEnvelope(body)
		if err != nil {
			logger.ErrorContext(r.Context(), "decoding push envelope", "err", err)
			http.Error(w, "bad request: invalid push envelope", http.StatusBadRequest)
			return
		}

		data, err := env.Message.DecodeData()
		if err != nil {
			logger.ErrorContext(r.Context(), "decoding push message data", "err", err)
			http.Error(w, "bad request: invalid message data", http.StatusBadRequest)
			return
		}

		msg := messaging.Message{
			Data:       data,
			Attributes: env.Message.Attributes,
		}

		if err := dispatcher.HandleScrapeTick(r.Context(), msg); err != nil {
			logger.ErrorContext(r.Context(), "handling scrape tick", "err", err, "msg_id", env.Message.MessageID)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// PushListingExtract handles POST /push/listing-extract. Operates identically to
// PushScrapeTick but delegates to the extraction application service.
func PushListingExtract(dispatcher ExtractionDispatcher, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			logger.ErrorContext(r.Context(), "reading push body", "err", err)
			http.Error(w, "bad request: cannot read body", http.StatusBadRequest)
			return
		}

		env, err := messaging.DecodePushEnvelope(body)
		if err != nil {
			logger.ErrorContext(r.Context(), "decoding push envelope", "err", err)
			http.Error(w, "bad request: invalid push envelope", http.StatusBadRequest)
			return
		}

		data, err := env.Message.DecodeData()
		if err != nil {
			logger.ErrorContext(r.Context(), "decoding push message data", "err", err)
			http.Error(w, "bad request: invalid message data", http.StatusBadRequest)
			return
		}

		msg := messaging.Message{
			Data:       data,
			Attributes: env.Message.Attributes,
		}

		if err := dispatcher.HandleListingExtract(r.Context(), msg); err != nil {
			logger.ErrorContext(r.Context(), "handling listing extract", "err", err, "msg_id", env.Message.MessageID)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
