package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/A7med-noureldin/wallet-ledger/internal/ledger"
	"github.com/A7med-noureldin/wallet-ledger/internal/money"
)

type LedgerService interface {
	CreateAccount(ctx context.Context) (int64, error)
	Deposit(ctx context.Context, accountID int64, req ledger.DepositReq) error
	Transfer(ctx context.Context, fromID, toID int64, req ledger.TransferReq) error
	GetBalance(ctx context.Context, accountID int64) (map[string]uint64, error)
	GetTransactions(ctx context.Context, accountID int64) ([]ledger.Transaction, error)
}

type Handler struct {
	service LedgerService
}

func New(service LedgerService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /accounts", h.handleCreateAccount)
	mux.HandleFunc("POST /accounts/{id}/deposits", h.handleDeposit)
	mux.HandleFunc("POST /transfers", h.handleTransfer)
	mux.HandleFunc("GET /accounts/{id}/balance", h.handleGetBalance)
	mux.HandleFunc("GET /accounts/{id}/transactions", h.handleGetTransactions)
}

func (h *Handler) handleCreateAccount(w http.ResponseWriter, r *http.Request) {
	id, err := h.service.CreateAccount(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create account")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"account_id": id})
}

func (h *Handler) handleDeposit(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	var req ledger.DepositReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	err = h.service.Deposit(r.Context(), accountID, req)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req ledger.TransferReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	err := h.service.Transfer(r.Context(), req.FromAccount, req.ToAccount, req)
	if err != nil {
		handleDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	balances, err := h.service.GetBalance(r.Context(), accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve balance")
		return
	}

	writeJSON(w, http.StatusOK, balances)
}

func (h *Handler) handleGetTransactions(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	accountID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid account id")
		return
	}

	transactions, err := h.service.GetTransactions(r.Context(), accountID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve transaction history")
		return
	}

	if transactions == nil {
		transactions = []ledger.Transaction{}
	}
	
	writeJSON(w, http.StatusOK, transactions)
}

func handleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, money.ErrInsufficientFunds):
		writeError(w, http.StatusUnprocessableEntity, "insufficient funds for transfer")
	case errors.Is(err, money.ErrUnsupportedCurrency):
		writeError(w, http.StatusBadRequest, "unsupported currency provided")
	default:
		writeError(w, http.StatusBadRequest, err.Error())
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
