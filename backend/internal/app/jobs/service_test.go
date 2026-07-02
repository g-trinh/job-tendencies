package jobs

import (
	"context"
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/messaging"
)

// fakeRawListingSource is a minimal JobRawListingSource fake for ReextractJob tests.
type fakeRawListingSource struct {
	ids []kernel.RawListingID
	err error
}

func (f *fakeRawListingSource) RawListingIDsByJob(context.Context, kernel.ProfileID, kernel.JobID) ([]kernel.RawListingID, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.ids, nil
}

// fakePublisher records every published message.
type fakePublisher struct {
	published []messaging.Message
	err       error
}

func (p *fakePublisher) Publish(_ context.Context, msg messaging.Message) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, msg)
	return nil
}

// TestReextractJob_PublishesPerRawListing verifies P5-4: one listing.extract message is
// republished per raw listing retained for the job, each carrying the raw listing id.
func TestReextractJob_PublishesPerRawListing(t *testing.T) {
	t.Parallel()

	rawListings := &fakeRawListingSource{ids: []kernel.RawListingID{"raw-1", "raw-2"}}
	pub := &fakePublisher{}
	svc := NewWithWriter(nil, nil).WithReextraction(rawListings, pub)

	if err := svc.ReextractJob(context.Background(), "profile-1", "job-1"); err != nil {
		t.Fatalf("ReextractJob error = %v", err)
	}
	if len(pub.published) != 2 {
		t.Fatalf("published = %d messages, want 2", len(pub.published))
	}
	if string(pub.published[0].Data) != "raw-1" || string(pub.published[1].Data) != "raw-2" {
		t.Fatalf("unexpected published payloads: %+v", pub.published)
	}
}

// TestReextractJob_NoSourcesIsNotFound verifies that a job with no raw listings visible
// to the profile (wrong profile, or unknown job) surfaces as kernel.NotFoundError rather
// than silently publishing nothing.
func TestReextractJob_NoSourcesIsNotFound(t *testing.T) {
	t.Parallel()

	rawListings := &fakeRawListingSource{ids: nil}
	pub := &fakePublisher{}
	svc := NewWithWriter(nil, nil).WithReextraction(rawListings, pub)

	err := svc.ReextractJob(context.Background(), "profile-1", "job-missing")
	var nfe *kernel.NotFoundError
	if !errors.As(err, &nfe) {
		t.Fatalf("ReextractJob error = %v, want *kernel.NotFoundError", err)
	}
	if len(pub.published) != 0 {
		t.Fatalf("published = %d messages, want 0", len(pub.published))
	}
}

// TestReextractJob_NotConfiguredErrors verifies the fail-fast guard when a Service is
// constructed without WithReextraction (e.g. read-only services via New).
func TestReextractJob_NotConfiguredErrors(t *testing.T) {
	t.Parallel()

	svc := New(nil)
	if err := svc.ReextractJob(context.Background(), "profile-1", "job-1"); err == nil {
		t.Fatal("expected error when reextraction is not configured")
	}
}
