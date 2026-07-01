package profiles_test

import (
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	"github.com/g-trinh/job-tendencies/internal/domain/profiles"
)

func TestNewProfile(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		inName   string
		location string
		keywords []string
		wantErr  bool
	}{
		{
			name:     "returns error when name is empty",
			inName:   "",
			location: "Paris",
			keywords: []string{"golang"},
			wantErr:  true,
		},
		{
			name:     "returns error when name is only whitespace",
			inName:   "   ",
			location: "Paris",
			keywords: []string{"golang"},
			wantErr:  true,
		},
		{
			name:     "creates profile with valid name",
			inName:   "Go Backend Paris",
			location: "Paris",
			keywords: []string{"golang", "backend"},
		},
		{
			name:     "nil keywords defaults to empty slice",
			inName:   "Remote Go",
			location: "Remote",
			keywords: nil,
		},
		{
			name:     "trims name whitespace",
			inName:   "  My Profile  ",
			location: "",
			keywords: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p, err := profiles.NewProfile(tc.inName, tc.location, tc.keywords)

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
			if p.Name == "" {
				t.Error("profile name is empty after construction")
			}
			if p.SearchKeywords == nil {
				t.Error("SearchKeywords should never be nil")
			}
		})
	}
}
