package extraction

import (
	"testing"
)

func TestComputeFingerprint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		title    string
		company  string
		location string
		// wantSame pairs share the same fingerprint; wantDiff pairs differ.
		wantSameAs *[3]string // when non-nil, fingerprint must equal computeFingerprint of this triplet
	}{
		{
			name:     "basic normalization lowercases and trims",
			title:    "  Go Engineer ",
			company:  " Acme Corp ",
			location: "Paris",
		},
		{
			name:     "same as lowercased input",
			title:    "go engineer",
			company:  "acme corp",
			location: "paris",
			wantSameAs: &[3]string{"  Go Engineer ", " Acme Corp ", "Paris"},
		},
		{
			name:     "location comma stripping paris france",
			title:    "Backend Developer",
			company:  "Beta Inc",
			location: "Paris, France",
			wantSameAs: &[3]string{"Backend Developer", "Beta Inc", "Paris"},
		},
		{
			name:     "location comma stripping with region",
			title:    "Backend Developer",
			company:  "Beta Inc",
			location: "Paris, Île-de-France, France",
			wantSameAs: &[3]string{"Backend Developer", "Beta Inc", "Paris, France"},
		},
		{
			name:     "whitespace collapse in title",
			title:    "Go   Engineer",
			company:  "Acme Corp",
			location: "Paris",
			wantSameAs: &[3]string{"Go Engineer", "Acme Corp", "Paris"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := computeFingerprint(tc.title, tc.company, tc.location)
			if got == "" {
				t.Fatal("expected non-empty fingerprint")
			}
			if tc.wantSameAs != nil {
				want := computeFingerprint(tc.wantSameAs[0], tc.wantSameAs[1], tc.wantSameAs[2])
				if got != want {
					t.Fatalf("fingerprint mismatch: got %q, want %q (same as %v)", got, want, tc.wantSameAs)
				}
			}
		})
	}
}

func TestComputeFingerprintDiffers(t *testing.T) {
	t.Parallel()

	// Listings that look similar but must NOT deduplicate.
	pairs := [][2][3]string{
		{
			{"Go Engineer", "Acme", "Paris"},
			{"Go Engineer", "Beta Corp", "Paris"}, // different company
		},
		{
			{"Go Engineer", "Acme", "Paris"},
			{"Python Engineer", "Acme", "Paris"}, // different title
		},
		{
			{"Go Engineer", "Acme", "Paris"},
			{"Go Engineer", "Acme", "Lyon"}, // different city
		},
	}

	for _, pair := range pairs {
		a := computeFingerprint(pair[0][0], pair[0][1], pair[0][2])
		b := computeFingerprint(pair[1][0], pair[1][1], pair[1][2])
		if a == b {
			t.Errorf("expected different fingerprints for %v and %v", pair[0], pair[1])
		}
	}
}
