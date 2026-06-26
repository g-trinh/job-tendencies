package handler

import (
	"context"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// contextWithProfileID stores a ProfileID in the given context.
// Retrieved via ActiveProfileID on a request derived from that context.
func contextWithProfileID(ctx context.Context, id kernel.ProfileID) context.Context {
	return context.WithValue(ctx, activeProfileKey, id)
}
