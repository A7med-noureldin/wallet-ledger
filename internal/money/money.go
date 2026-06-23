package money

import (
	"errors"
)

var (
	ErrUnsupportedCurrency = errors.New("unsupported currency")
	ErrInsufficientFunds   = errors.New("insufficient funds")
)

type Currency string

const (
	EGP Currency = "EGP"
	USD Currency = "USD"
)

type Money struct {
	amount   uint64
	currency Currency
}

func New(amount uint64, currency Currency) (*Money, error) {
	if currency != EGP && currency != USD {
		return nil, ErrUnsupportedCurrency
	}
	return &Money{amount: amount, currency: currency}, nil
}

func (m *Money) Amount() uint64 {
	return m.amount
}

func (m *Money) Currency() Currency {
	return m.currency
}

func (m *Money) Add(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot add different currencies")
	}
	return &Money{amount: m.amount + other.amount, currency: m.currency}, nil
}

func (m *Money) Subtract(other *Money) (*Money, error) {
	if m.currency != other.currency {
		return nil, errors.New("cannot sub different currencies")
	}
	if m.amount < other.amount {
		return nil, ErrInsufficientFunds
	}
	return &Money{amount: m.amount - other.amount, currency: m.currency}, nil
}
