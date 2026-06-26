package kernel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ErrInvalidMoney is returned when a money value cannot be parsed or constructed.
var ErrInvalidMoney = errors.New("invalid money value")

// Currency represents an ISO 4217 currency code (e.g. "EUR").
type Currency string

const (
	// CurrencyEUR is the Euro, the default currency for salary values.
	CurrencyEUR Currency = "EUR"
)

// Money is an immutable value object representing a monetary amount stored in the
// smallest currency unit (centimes for EUR). Salary ranges use two Money values (min/max).
// The zero value is 0 EUR.
//
// Example:
//
//	m, err := ParseMoney("60000 EUR")
//	fmt.Println(m) // "60000 EUR"
type Money struct {
	amountCents int64
	currency    Currency
}

// NewMoney constructs a Money value. amountCents is the amount in the smallest unit
// (centimes for EUR). currency must be a non-empty ISO 4217 code.
func NewMoney(amountCents int64, currency Currency) (Money, error) {
	if currency == "" {
		return Money{}, fmt.Errorf("%w: currency is required", ErrInvalidMoney)
	}
	return Money{amountCents: amountCents, currency: currency}, nil
}

// ParseMoney parses a money value from a string of the form "<amount>" or "<amount> <CURRENCY>".
// The amount is treated as whole units (euros, not centimes). Thousands separators (commas)
// are stripped. When no currency is specified, EUR is assumed.
//
// Examples:
//
//	ParseMoney("60000")       → 60000 EUR
//	ParseMoney("60,000 EUR")  → 60000 EUR
//	ParseMoney("80000 EUR")   → 80000 EUR
func ParseMoney(s string) (Money, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Money{}, fmt.Errorf("%w: empty string", ErrInvalidMoney)
	}

	parts := strings.Fields(s)
	amountStr := parts[0]
	currency := CurrencyEUR

	if len(parts) == 2 {
		currency = Currency(strings.ToUpper(parts[1]))
	}

	// Strip thousands separators.
	amountStr = strings.ReplaceAll(amountStr, ",", "")

	amount, err := strconv.ParseInt(amountStr, 10, 64)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %q is not a valid integer", ErrInvalidMoney, amountStr)
	}

	return Money{amountCents: amount * 100, currency: currency}, nil
}

// AmountCents returns the monetary amount in the smallest currency unit (centimes for EUR).
func (m Money) AmountCents() int64 { return m.amountCents }

// AmountUnits returns the monetary amount as whole units (euros for EUR).
func (m Money) AmountUnits() int64 { return m.amountCents / 100 }

// Currency returns the ISO 4217 currency code.
func (m Money) Currency() Currency { return m.currency }

// String formats the money as "<whole-units> <CURRENCY>" (e.g. "60000 EUR").
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.AmountUnits(), m.currency)
}

// Equal reports whether m and o represent exactly the same monetary value.
func (m Money) Equal(o Money) bool {
	return m.amountCents == o.amountCents && m.currency == o.currency
}
