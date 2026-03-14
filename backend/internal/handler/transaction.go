package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jayeshinusshinde/banking-backend/internal/service"
)

type TransactionHandler struct {
	svc *service.TransactionService
}

func NewTransactionHandler(svc *service.TransactionService) *TransactionHandler {
	return &TransactionHandler{svc: svc}
}

func (h *TransactionHandler) Deposit(w http.ResponseWriter, r *http.Request) {
	var req service.DepositRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.AccountID == "" || req.Amount <= 0 || req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_id, amount (>0), and idempotency_key are required"})
		return
	}

	account, err := h.svc.Deposit(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, account)
}

func (h *TransactionHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	var req service.WithdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.AccountID == "" || req.Amount <= 0 || req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "account_id, amount (>0), and idempotency_key are required"})
		return
	}

	account, err := h.svc.Withdraw(r.Context(), req)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, account)
}

func (h *TransactionHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req service.TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.FromAccountID == "" || req.ToAccountID == "" || req.Amount <= 0 || req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from_account_id, to_account_id, amount (>0), and idempotency_key are required"})
		return
	}

	if err := h.svc.Transfer(r.Context(), req); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}
