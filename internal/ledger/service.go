package ledger

import (
	"context"
	"errors"
	"fmt"

	"github.com/A7med-noureldin/wallet-ledger/internal/money"
)

var ErrAccountNotFound = errors.New("account not found")

type Service struct {
	repo Repository
}

type Repository interface {
	CreateAccount(ctx context.Context) (int64, error)
	Deposit(ctx context.Context, accountID int64, amount uint64, currency money.Currency) error
	Transfer(ctx context.Context, fromID, toID int64, amount uint64, currency money.Currency) error
	GetBalance(ctx context.Context, accountID int64) (map[string]uint64, error)
	AccountExists(ctx context.Context, accountID int64) (bool, error)
	GetTransactions(ctx context.Context, accountID int64) ([]Transaction, error)
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateAccount(ctx context.Context) (int64, error) {
	return s.repo.CreateAccount(ctx)
}

func (s *Service) Deposit(ctx context.Context, accountID int64, req DepositReq) error {
	if req.Amount == 0 {
		return errors.New("amount must be greater than zero")
	}

	targetCurrency := req.Currency

	_, err := money.New(req.Amount, targetCurrency)
	if err != nil {
		return fmt.Errorf("failed deposit: %w", err)
	}

	return s.repo.Deposit(ctx, accountID, req.Amount, targetCurrency)
}

func (s *Service) Transfer(ctx context.Context, fromID, toID int64, req TransferReq) error {
	if req.Amount == 0 {
		return errors.New("amount must be greater than zero")
	}

	if fromID == toID {
		return errors.New("cannot transfer money to the same account")
	}

	targetCurrency := req.Currency

	_, err := money.New(req.Amount, targetCurrency)
	if err != nil {
		return fmt.Errorf("failed transfer: %w", err)
	}

	return s.repo.Transfer(ctx, fromID, toID, req.Amount, targetCurrency)
}

func (s *Service) GetBalance(ctx context.Context, accountID int64) (map[string]uint64, error) {
	return s.repo.GetBalance(ctx, accountID)
}

func (s *Service) GetTransactions(ctx context.Context, accountID int64) ([]Transaction, error) {
	exists, err := s.repo.AccountExists(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify account existence: %w", err)
	}
	if !exists {
		return nil, ErrAccountNotFound
	}

	transactions, err := s.repo.GetTransactions(ctx, accountID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}

	return transactions, nil
}
