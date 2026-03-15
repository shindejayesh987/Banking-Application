package handler

import (
	"encoding/json"
	"net/http"

	"github.com/jayeshinusshinde/banking-backend/internal/saga"
)

type SagaTransferHandler struct {
	transferSaga sagaExecutor
}

func NewSagaTransferHandler(transferSaga sagaExecutor) *SagaTransferHandler {
	return &SagaTransferHandler{transferSaga: transferSaga}
}

func (h *SagaTransferHandler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req saga.TransferPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.FromAccountID == "" || req.ToAccountID == "" || req.Amount <= 0 || req.IdempotencyKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "from_account_id, to_account_id, amount (>0), and idempotency_key are required",
		})
		return
	}
	if req.FromAccountID == req.ToAccountID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot transfer to same account"})
		return
	}
	if req.Currency == "" {
		req.Currency = "USD"
	}

	payload, _ := json.Marshal(req)

	sagaID, err := h.transferSaga.Execute(r.Context(), payload)
	if err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error":   err.Error(),
			"saga_id": sagaID,
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "completed",
		"saga_id": sagaID,
	})
}
