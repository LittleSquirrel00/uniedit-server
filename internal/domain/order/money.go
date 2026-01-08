package order

import "fmt"

// Money represents a monetary value with currency.
// It is an immutable value object.
type Money struct {
	amount   int64  // Amount in smallest currency unit (e.g., cents)
	currency string // ISO 4217 currency code (e.g., "usd")
}

// NewMoney creates a new Money value object.
func NewMoney(amount int64, currency string) Money {
	if currency == "" {
		currency = "usd"
	}
	return Money{amount: amount, currency: currency}
}

// Amount returns the amount in smallest currency unit.
func (m Money) Amount() int64 {
	return m.amount
}

// Currency returns the ISO 4217 currency code.
func (m Money) Currency() string {
	return m.currency
}

// IsZero returns true if the amount is zero.
func (m Money) IsZero() bool {
	return m.amount == 0
}

// IsPositive returns true if the amount is positive.
func (m Money) IsPositive() bool {
	return m.amount > 0
}

// IsNegative returns true if the amount is negative.
func (m Money) IsNegative() bool {
	return m.amount < 0
}

// Add returns a new Money with the sum of two Money values.
// Returns an error if currencies don't match.
func (m Money) Add(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	return NewMoney(m.amount+other.amount, m.currency), nil
}

// Subtract returns a new Money with the difference of two Money values.
// Returns an error if currencies don't match.
func (m Money) Subtract(other Money) (Money, error) {
	if m.currency != other.currency {
		return Money{}, fmt.Errorf("currency mismatch: %s vs %s", m.currency, other.currency)
	}
	return NewMoney(m.amount-other.amount, m.currency), nil
}

// Multiply returns a new Money with the amount multiplied by the factor.
func (m Money) Multiply(factor int) Money {
	return NewMoney(m.amount*int64(factor), m.currency)
}

// Equals checks if two Money values are equal.
func (m Money) Equals(other Money) bool {
	return m.amount == other.amount && m.currency == other.currency
}

// String returns a string representation of the Money value.
func (m Money) String() string {
	return fmt.Sprintf("%d %s", m.amount, m.currency)
}
