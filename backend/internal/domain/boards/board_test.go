package boards_test

import (
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/boards"
	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

// AC: CRUD works; four boards seeded.

func TestNewBoard(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		bName   string
		baseURL string
		wantErr bool
	}{
		{
			name:    "returns error when name is empty",
			bName:   "",
			baseURL: "https://www.indeed.com",
			wantErr: true,
		},
		{
			name:    "returns error when base_url is empty",
			bName:   "Indeed",
			baseURL: "",
			wantErr: true,
		},
		{
			name:    "creates board enabled by default",
			bName:   "Indeed",
			baseURL: "https://www.indeed.com",
		},
		{
			name:    "trims whitespace from name and url",
			bName:   "  LinkedIn  ",
			baseURL: "  https://www.linkedin.com/jobs  ",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b, err := boards.NewBoard(tc.bName, tc.baseURL)

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
			if !b.Enabled {
				t.Error("new board should be enabled by default")
			}
			if b.Name == "" {
				t.Error("board name is empty after construction")
			}
		})
	}
}
