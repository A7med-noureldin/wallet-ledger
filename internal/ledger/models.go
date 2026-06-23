package ledger

import (
	"time"

	"github.com/A7med-noureldin/wallet-ledger/internal/money"
)

type DepositReq struct {
	Amount   uint64         `json:"amount"`
	Currency money.Currency `json:"currency"`
}

type TransferReq struct {
	FromAccount int64          `json:"from_account"`
	ToAccount   int64          `json:"to_account"`
	Amount      uint64         `json:"amount"`
	Currency    money.Currency `json:"currency"`
}

type LedgerEntry struct {
	ID            int64          `json:"id"`
	TransactionID int64          `json:"transaction_id"`
	AccountID     int64          `json:"account_id"`
	Amount        uint64         `json:"amount"`
	Currency      money.Currency `json:"currency"`
	Direction     string         `json:"direction"`
	CreatedAt     time.Time      `json:"created_at"`
}

type Transaction struct {
	ID        int64     `json:"id"`
	AccountID int64     `json:"account_id"`
	Amount    uint64    `json:"amount"`
	Currency  string    `json:"currency"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}
