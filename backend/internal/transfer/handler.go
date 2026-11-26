package transfer

import (
	"atlas-financial-core/internal/response"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type Handler struct {
	s *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		s: s,
	}
}

func (h *Handler) ExecuteTransfer(w http.ResponseWriter, r *http.Request) {
	var transfer Transfer
	transfer.IdempotencyKey = r.Header.Get("idempotency-key")
	err := json.NewDecoder(r.Body).Decode(&transfer)
	if err != nil {
		response.ErrorWriter(w, http.StatusBadRequest, "JSON_ERROR_DECODE", "JSON decoding error")
		return
	}

	data, err := h.s.ExecuteTransfer(r.Context(), transfer)
	if err != nil {
		switch {
		case errors.Is(err, ErrIdempotencyKeyMissing),
			errors.Is(err, ErrInsufficientBalance),
			errors.Is(err, ErrCurrenciesNotMatch),
			errors.Is(err, ErrSameAccountID),
			errors.Is(err, ErrInvalidAmount):
			response.ErrorWriter(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())

		case errors.Is(err, ErrSenderIdNotMatch),
			errors.Is(err, ErrReceiverIdNotMatch),
			errors.Is(err, ErrAmountNotMatch),
			errors.Is(err, ErrPurposeNotMatch):
			response.ErrorWriter(w, http.StatusConflict, "CONFLICT_ERROR", "conflict account value mismatch")
		case errors.Is(err, ErrAccountNotFound):
			response.ErrorWriter(w, http.StatusNotFound, "NOT_FOUND_ERROR", "account not found")
		default:
			slog.Error("internal server error", slog.String("error", err.Error()))
			response.ErrorWriter(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an unexpected error occurred")
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.Error("failed to encode json response", slog.String("error", err.Error()))
	}
}

func (h *Handler) RegisterRoute(mux *http.ServeMux) {
	mux.HandleFunc("POST /transfers", h.ExecuteTransfer)
}
