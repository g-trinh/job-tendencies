package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// errorResponse is the JSON body returned for all error responses.
type errorResponse struct {
	Error string `json:"error"`
}

// RespondError maps a domain error to the appropriate HTTP status code and writes a
// JSON error body. Domain error types map as follows:
//
//   - kernel.ErrNotFound / *kernel.NotFoundError → 404
//   - kernel.ErrInvalidInput / *kernel.ValidationError → 400
//   - kernel.ErrConflict → 409
//   - kernel.ErrUnauthorized → 401
//   - anything else → 500 (internal error detail is not leaked)
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	status, msg := mapDomainError(err)
	if status == http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "unhandled error", "err", err)
	}
	respond(w, status, errorResponse{Error: msg})
}

// mapDomainError returns the HTTP status code and user-facing message for err.
func mapDomainError(err error) (int, string) {
	switch {
	case errors.Is(err, kernel.ErrNotFound):
		var nfe *kernel.NotFoundError
		if errors.As(err, &nfe) {
			return http.StatusNotFound, nfe.Error()
		}
		return http.StatusNotFound, "resource not found"

	case errors.Is(err, kernel.ErrInvalidInput):
		var ve *kernel.ValidationError
		if errors.As(err, &ve) {
			return http.StatusBadRequest, ve.Error()
		}
		return http.StatusBadRequest, "invalid input"

	case errors.Is(err, kernel.ErrConflict):
		return http.StatusConflict, "conflict"

	case errors.Is(err, kernel.ErrUnauthorized):
		return http.StatusUnauthorized, "unauthorized"

	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// respond writes status and a JSON-encoded body to w.
func respond(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("encoding response", "err", err)
	}
}
