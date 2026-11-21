package account

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

func (h *Handler) CreateHandler(w http.ResponseWriter, r *http.Request) {
	input := InputCreateAccount{}
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		response.ErrorWriter(w, 400, "JSON_ERROR_DECODE", "JSON decoding error")
		return
	}
	data, err := h.s.CreateAccount(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, ErrOwnerName),
			errors.Is(err, ErrEmptyCurrencies),
			errors.Is(err, ErrInvalidCurrencies),
			errors.Is(err, ErrNotIDR),
			errors.Is(err, ErrInitialDeposit):
			response.ErrorWriter(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		default:
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

func (h *Handler) FindByAccountNumberHandler(w http.ResponseWriter, r *http.Request) {
	accNum := r.PathValue("account_number")
	data, err := h.s.FindByAccountNumber(r.Context(), accNum)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			response.ErrorWriter(w, http.StatusNotFound, "ACCOUNT_NOT_FOUND", err.Error())
			return
		}
		if errors.Is(err, ErrEmptyAccountNumber) {
			response.ErrorWriter(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		response.ErrorWriter(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an unexpected error occurred")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.Error("failed to encode json response", slog.String("error", err.Error()))
	}
}

func (h *Handler) ListAccountsHandler(w http.ResponseWriter, r *http.Request) {
	data, err := h.s.ListAccounts(r.Context())
	if err != nil {
		response.ErrorWriter(w, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "an unexpected error occurred")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		slog.Error("failed to encode json response", slog.String("error", err.Error()))
	}
}

func (h *Handler) RegisterRoute(mux *http.ServeMux) {
	mux.HandleFunc("POST /accounts", h.CreateHandler)
	mux.HandleFunc("GET /accounts/{account_number}", h.FindByAccountNumberHandler)
	mux.HandleFunc("GET /accounts", h.ListAccountsHandler)
}
