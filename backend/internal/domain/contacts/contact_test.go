package contacts_test

import (
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/contacts"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// AC: Upserting a contact with an existing email|linkedin merges, not duplicates.
// This test covers the dedup_key computation which drives the upsert at the DB level.

func TestNewContact_DedupKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		email       string
		linkedInURL string
		wantKey     string
		wantErr     bool
	}{
		{
			name:        "returns error when both email and linkedin are empty",
			email:       "",
			linkedInURL: "",
			wantErr:     true,
		},
		{
			name:    "uses email as dedup key when email is present",
			email:   "alice@example.com",
			wantKey: "email:alice@example.com",
		},
		{
			name:        "uses linkedin as dedup key when only linkedin present",
			linkedInURL: "https://linkedin.com/in/alice",
			wantKey:     "linkedin:https://linkedin.com/in/alice",
		},
		{
			name:        "email takes priority over linkedin for dedup key",
			email:       "alice@example.com",
			linkedInURL: "https://linkedin.com/in/alice",
			wantKey:     "email:alice@example.com",
		},
		{
			name:    "normalises email to lowercase in dedup key",
			email:   "Alice@Example.COM",
			wantKey: "email:alice@example.com",
		},
		{
			name:    "strips whitespace from email before computing dedup key",
			email:   "  alice@example.com  ",
			wantKey: "email:alice@example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c, err := contacts.NewContact("Alice", "Acme", tc.email, tc.linkedInURL, "", "", nil)

			if tc.wantErr {
				if err == nil {
					t.Fatal("want error; got nil")
				}
				if !errors.Is(err, kernel.ErrInvalidInput) {
					t.Errorf("error type = %T; want *kernel.ValidationError", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.DedupKey != tc.wantKey {
				t.Errorf("DedupKey = %q; want %q", c.DedupKey, tc.wantKey)
			}
		})
	}
}

func TestComputeDedupKey(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		email       string
		linkedInURL string
		want        string
	}{
		{
			name:  "email prefix when email provided",
			email: "bob@example.com",
			want:  "email:bob@example.com",
		},
		{
			name:        "linkedin prefix when only linkedin provided",
			linkedInURL: "https://linkedin.com/in/bob",
			want:        "linkedin:https://linkedin.com/in/bob",
		},
		{
			name:        "email wins when both provided",
			email:       "bob@example.com",
			linkedInURL: "https://linkedin.com/in/bob",
			want:        "email:bob@example.com",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := contacts.ComputeDedupKey(tc.email, tc.linkedInURL)
			if got != tc.want {
				t.Errorf("ComputeDedupKey = %q; want %q", got, tc.want)
			}
		})
	}
}
