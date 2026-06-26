package kernel_test

import (
	"errors"
	"testing"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
)

func TestParseMoney(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		input       string
		wantUnits   int64
		wantCents   int64
		wantCurr    kernel.Currency
		wantStr     string
		wantErr     bool
		errContains string
	}{
		{
			name:      "parses whole amount with explicit EUR",
			input:     "60000 EUR",
			wantUnits: 60000,
			wantCents: 6_000_000,
			wantCurr:  kernel.CurrencyEUR,
			wantStr:   "60000 EUR",
		},
		{
			name:      "parses whole amount without currency — defaults to EUR",
			input:     "80000",
			wantUnits: 80000,
			wantCents: 8_000_000,
			wantCurr:  kernel.CurrencyEUR,
			wantStr:   "80000 EUR",
		},
		{
			name:      "strips thousands separator comma",
			input:     "60,000 EUR",
			wantUnits: 60000,
			wantCents: 6_000_000,
			wantCurr:  kernel.CurrencyEUR,
			wantStr:   "60000 EUR",
		},
		{
			name:      "trims surrounding whitespace",
			input:     "  45000 EUR  ",
			wantUnits: 45000,
			wantCents: 4_500_000,
			wantCurr:  kernel.CurrencyEUR,
			wantStr:   "45000 EUR",
		},
		{
			name:    "empty string returns error",
			input:   "",
			wantErr: true,
		},
		{
			name:        "non-numeric amount returns error",
			input:       "not-a-number EUR",
			wantErr:     true,
			errContains: "not a valid integer",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.ParseMoney(tc.input)

			if tc.wantErr {
				if err == nil {
					t.Fatalf("ParseMoney(%q) returned nil error; want non-nil", tc.input)
				}
				if !errors.Is(err, kernel.ErrInvalidMoney) {
					t.Errorf("ParseMoney(%q) error does not wrap ErrInvalidMoney: %v", tc.input, err)
				}
				if tc.errContains != "" {
					if msg := err.Error(); !contains(msg, tc.errContains) {
						t.Errorf("ParseMoney(%q) error = %q; want to contain %q", tc.input, msg, tc.errContains)
					}
				}
				return
			}

			if err != nil {
				t.Fatalf("ParseMoney(%q) unexpected error: %v", tc.input, err)
			}
			if got.AmountUnits() != tc.wantUnits {
				t.Errorf("AmountUnits() = %d; want %d", got.AmountUnits(), tc.wantUnits)
			}
			if got.AmountCents() != tc.wantCents {
				t.Errorf("AmountCents() = %d; want %d", got.AmountCents(), tc.wantCents)
			}
			if got.Currency() != tc.wantCurr {
				t.Errorf("Currency() = %q; want %q", got.Currency(), tc.wantCurr)
			}
			if got.String() != tc.wantStr {
				t.Errorf("String() = %q; want %q", got.String(), tc.wantStr)
			}
		})
	}
}

func TestMoney_Equal(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		a     string
		b     string
		equal bool
	}{
		{
			name:  "same amount and currency are equal",
			a:     "60000 EUR",
			b:     "60000 EUR",
			equal: true,
		},
		{
			name:  "different amounts are not equal",
			a:     "60000 EUR",
			b:     "70000 EUR",
			equal: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			a, err := kernel.ParseMoney(tc.a)
			if err != nil {
				t.Fatalf("ParseMoney(%q): %v", tc.a, err)
			}
			b, err := kernel.ParseMoney(tc.b)
			if err != nil {
				t.Fatalf("ParseMoney(%q): %v", tc.b, err)
			}

			if got := a.Equal(b); got != tc.equal {
				t.Errorf("Equal() = %v; want %v", got, tc.equal)
			}
		})
	}
}

func TestNewMoney(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		cents   int64
		curr    kernel.Currency
		wantErr bool
	}{
		{
			name:  "valid amount and currency",
			cents: 6_000_000,
			curr:  kernel.CurrencyEUR,
		},
		{
			name:    "empty currency returns error",
			cents:   1000,
			curr:    "",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := kernel.NewMoney(tc.cents, tc.curr)

			if tc.wantErr {
				if err == nil {
					t.Fatal("NewMoney() returned nil error; want non-nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("NewMoney() unexpected error: %v", err)
			}
			if got.AmountCents() != tc.cents {
				t.Errorf("AmountCents() = %d; want %d", got.AmountCents(), tc.cents)
			}
		})
	}
}

// contains is a helper since strings package would add an import to a test-only file.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (len(s) > 0 && stringContains(s, sub)))
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
