package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/A7med-noureldin/wallet-ledger/internal/ledger"
	"github.com/A7med-noureldin/wallet-ledger/internal/money"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS accounts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ledger_entries (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		transaction_id INTEGER NOT NULL,
		account_id INTEGER NOT NULL,
		amount INTEGER NOT NULL CHECK(amount > 0),
		currency TEXT NOT NULL CHECK(currency IN ('EGP', 'USD')),
		direction TEXT NOT NULL CHECK(direction IN ('CREDIT', 'DEBIT')),
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(transaction_id) REFERENCES transactions(id),
		FOREIGN KEY(account_id) REFERENCES accounts(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ledger_balance ON ledger_entries(account_id, currency, direction, amount);
	CREATE INDEX IF NOT EXISTS idx_ledger_history ON ledger_entries(account_id, created_at DESC);
	`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (r *Repository) CreateAccount(ctx context.Context) (int64, error) {
	res, err := r.db.ExecContext(ctx, "INSERT INTO accounts DEFAULT VALUES")
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *Repository) Deposit(ctx context.Context, accountID int64, amount uint64, currency money.Currency) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	res, err := tx.ExecContext(ctx, "INSERT INTO transactions DEFAULT VALUES")
	if err != nil {
		return err
	}

	txID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO ledger_entries (transaction_id, account_id, amount, currency, direction)
		VALUES (?, ?, ?, ?, 'CREDIT')
	`, txID, accountID, amount, currency)

	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) Transfer(ctx context.Context, fromID, toID int64, amount uint64, currency money.Currency) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	var currentBalance uint64
	q := `
          SELECT COALESCE(SUM(CASE WHEN direction = 'CREDIT' THEN amount ELSE -amount END), 0) FROM ledger_entries 
          WHERE account_id = ? AND currency = ?
       `

	err = tx.QueryRowContext(ctx, q, fromID, currency).Scan(&currentBalance)
	if err != nil {
		return err
	}

	balMoney, err := money.New(currentBalance, currency)
	if err != nil {
		return fmt.Errorf("failed to initialize balance domain object: %w", err)
	}

	txMoney, err := money.New(amount, currency)
	if err != nil {
		return fmt.Errorf("failed to initialize transfer domain object: %w", err)
	}

	_, err = balMoney.Subtract(txMoney)
	if err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `INSERT INTO transactions DEFAULT VALUES`)
	if err != nil {
		return err
	}

	txID, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO ledger_entries (transaction_id, account_id, amount, currency, direction)
                               VALUES (?, ?, ?, ?, 'DEBIT')`, txID, fromID, amount, currency)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO ledger_entries (transaction_id, account_id, amount, currency, direction)
                               VALUES (?, ?, ?, ?, 'CREDIT')`, txID, toID, amount, currency)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repository) GetBalance(ctx context.Context, accountID int64) (map[string]uint64, error) {
	q := `
			SELECT currency, SUM(CASE WHEN direction = 'CREDIT' THEN amount ELSE -amount END) as balance
			FROM ledger_entries WHERE account_id = ?
			GROUP BY currency;
		`

	rows, err := r.db.QueryContext(ctx, q, accountID)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	balances := make(map[string]uint64)

	for rows.Next() {
		var currency string
		var balance uint64

		if err := rows.Scan(&currency, &balance); err != nil {
			return nil, err
		}
		balances[currency] = balance
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during row iteration: %w", err)
	}

	return balances, nil
}

func (r *Repository) GetTransactions(ctx context.Context, accountID int64) ([]ledger.Transaction, error) {
	query := `
       SELECT id, account_id, amount, currency, direction as type, created_at 
       FROM ledger_entries 
       WHERE account_id = ?
       ORDER BY created_at DESC;
    `

	rows, err := r.db.QueryContext(ctx, query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []ledger.Transaction

	for rows.Next() {
		var tx ledger.Transaction
		if err := rows.Scan(
			&tx.ID,
			&tx.AccountID,
			&tx.Amount,
			&tx.Currency,
			&tx.Type,
			&tx.CreatedAt,
		); err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if transactions == nil {
		transactions = make([]ledger.Transaction, 0)
	}

	return transactions, nil
}

func (r *Repository) AccountExists(ctx context.Context, accountID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM accounts WHERE id = ?)`

	err := r.db.QueryRowContext(ctx, query, accountID).Scan(&exists)
	if err != nil {
		return false, err
	}

	return exists, nil
}
