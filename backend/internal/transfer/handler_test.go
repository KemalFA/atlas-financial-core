package transfer

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atlas-financial-core/internal/response"

	"github.com/jackc/pgx/v5"
)

func TestExecuteTransferHandler_Happy(t *testing.T) {
	var transferResp Transfer
	dataDummy := FakeRepository{
		findErr: pgx.ErrNoRows,
		transferSent: Transfer{
			AccountSenderID:   "sender-1",
			AccountReceiverID: "receiver-1",
			Amount:            50000,
			Purpose:           "payment",
			IdempotencyKey:    "idem-key-1",
			CreatedAt:         time.Now(),
		},
	}

	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
	}

	b, err := json.Marshal(inputTransfer)
	if err != nil {
		t.Fatalf("failed to serialize JSON: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(b))
	r.Header.Set("Idempotency-Key", "idem-key-1")
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ExecuteTransfer(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected: 201, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	err = json.NewDecoder(res.Body).Decode(&transferResp)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if transferResp.AccountSenderID != dataDummy.transferSent.AccountSenderID {
		t.Errorf("sender ID mismatch: got %q, want %q", transferResp.AccountSenderID, dataDummy.transferSent.AccountSenderID)
	}

	if transferResp.AccountReceiverID != dataDummy.transferSent.AccountReceiverID {
		t.Errorf("receiver ID mismatch: got %q, want %q", transferResp.AccountReceiverID, dataDummy.transferSent.AccountReceiverID)
	}

	if transferResp.Amount != dataDummy.transferSent.Amount {
		t.Errorf("amount mismatch: got %d, want %d", transferResp.Amount, dataDummy.transferSent.Amount)
	}

	if transferResp.Purpose != dataDummy.transferSent.Purpose {
		t.Errorf("purpose mismatch: got %q, want %q", transferResp.Purpose, dataDummy.transferSent.Purpose)
	}

	if !dataDummy.executeCalled {
		t.Errorf("expected repository ExecuteTransfer to be called")
	}
}

func TestExecuteTransferHandler_JSONError(t *testing.T) {
	dataDummy := FakeRepository{}

	r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBufferString("{invalid"))
	r.Header.Set("Idempotency-Key", "idem-key-1")
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ExecuteTransfer(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected: 400, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var errResp response.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errResp)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Status != http.StatusBadRequest {
		t.Errorf("expected error status 400, got %d", errResp.Error.Status)
	}

	if errResp.Error.Code != "JSON_ERROR_DECODE" {
		t.Errorf("expected error code JSON_ERROR_DECODE, got %s", errResp.Error.Code)
	}

	if errResp.Error.Message != "JSON decoding error" {
		t.Errorf("expected error message JSON decoding error, got %s", errResp.Error.Message)
	}

	if dataDummy.executeCalled {
		t.Errorf("expected ExecuteTransfer not to be called")
	}
}

func TestExecuteTransferHandler_ValidationError(t *testing.T) {
	tests := []struct {
		name           string
		idempotencyKey string
		input          Transfer
		repo           FakeRepository
		wantMsg        string
	}{
		{
			name:           "missing idempotency key",
			idempotencyKey: "",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-1",
				Amount:            50000,
				Purpose:           "payment",
			},
			repo:    FakeRepository{},
			wantMsg: ErrIdempotencyKeyMissing.Error(),
		},
		{
			name:           "self transfer",
			idempotencyKey: "idem-key-1",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "sender-1",
				Amount:            50000,
				Purpose:           "payment",
			},
			repo:    FakeRepository{},
			wantMsg: ErrSameAccountID.Error(),
		},
		{
			name:           "invalid amount",
			idempotencyKey: "idem-key-1",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-1",
				Amount:            0,
				Purpose:           "payment",
			},
			repo:    FakeRepository{},
			wantMsg: ErrInvalidAmount.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.input)
			if err != nil {
				t.Fatalf("failed to serialize JSON: %v", err)
			}

			r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(b))
			if tt.idempotencyKey != "" {
				r.Header.Set("Idempotency-Key", tt.idempotencyKey)
			}
			w := httptest.NewRecorder()

			serviceTest := NewService(&tt.repo)
			handlerTest := NewHandler(serviceTest)

			handlerTest.ExecuteTransfer(w, r)
			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("expected: 400, got: %v", res.StatusCode)
			}

			if res.Header.Get("Content-Type") != "application/json" {
				t.Errorf("content type header is not application/json")
			}

			var errResp response.ErrorResponse
			err = json.NewDecoder(res.Body).Decode(&errResp)
			if err != nil {
				t.Fatalf("failed to decode error response: %v", err)
			}

			if errResp.Error.Status != http.StatusBadRequest {
				t.Errorf("expected error status 400, got %d", errResp.Error.Status)
			}

			if errResp.Error.Code != "VALIDATION_ERROR" {
				t.Errorf("expected error code VALIDATION_ERROR, got %s", errResp.Error.Code)
			}

			if errResp.Error.Message != tt.wantMsg {
				t.Errorf("expected error message %q, got %q", tt.wantMsg, errResp.Error.Message)
			}
		})
	}
}

func TestExecuteTransferHandler_AccountNotFound(t *testing.T) {
	dataDummy := FakeRepository{
		findErr:    pgx.ErrNoRows,
		executeErr: ErrAccountNotFound,
	}

	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-missing",
		Amount:            50000,
		Purpose:           "payment",
	}

	b, err := json.Marshal(inputTransfer)
	if err != nil {
		t.Fatalf("failed to serialize JSON: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(b))
	r.Header.Set("Idempotency-Key", "idem-key-1")
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ExecuteTransfer(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected: %v, got: %v", http.StatusNotFound, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var errResp response.ErrorResponse
	err = json.NewDecoder(res.Body).Decode(&errResp)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Status != http.StatusNotFound {
		t.Errorf("expected error status 404, got %d", errResp.Error.Status)
	}

	if errResp.Error.Code != "NOT_FOUND_ERROR" {
		t.Errorf("expected error code NOT_FOUND_ERROR, got %s", errResp.Error.Code)
	}

	if errResp.Error.Message != "sender or receiver account not found" {
		t.Errorf("expected error message sender or receiver account not found, got %s", errResp.Error.Message)
	}

	if !dataDummy.executeCalled {
		t.Errorf("expected repository ExecuteTransfer to be called")
	}
}

func TestExecuteTransferHandler_Conflict(t *testing.T) {
	dataDummy := FakeRepository{
		transferSent: Transfer{
			ID:                "existing-transfer-id",
			AccountSenderID:   "sender-1",
			AccountReceiverID: "receiver-1",
			Amount:            50000,
			Purpose:           "payment",
			IdempotencyKey:    "idem-key-1",
			CreatedAt:         time.Now(),
		},
	}

	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            99000,
		Purpose:           "payment",
	}

	b, err := json.Marshal(inputTransfer)
	if err != nil {
		t.Fatalf("failed to serialize JSON: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(b))
	r.Header.Set("Idempotency-Key", "idem-key-1")
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ExecuteTransfer(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusConflict {
		t.Errorf("expected: %v, got: %v", http.StatusConflict, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var errResp response.ErrorResponse
	err = json.NewDecoder(res.Body).Decode(&errResp)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Status != http.StatusConflict {
		t.Errorf("expected error status 409, got %d", errResp.Error.Status)
	}

	if errResp.Error.Code != "CONFLICT_ERROR" {
		t.Errorf("expected error code CONFLICT_ERROR, got %s", errResp.Error.Code)
	}

	if errResp.Error.Message != "amount not match" {
		t.Errorf("expected error message amount not match, got %s", errResp.Error.Message)
	}

	if !dataDummy.findCalled {
		t.Errorf("expected repository FindByIdempotencyKey to be called")
	}

	if dataDummy.executeCalled {
		t.Errorf("expected repository ExecuteTransfer not to be called")
	}
}

func TestExecuteTransferHandler_InternalServerError(t *testing.T) {
	errUnexpected := errors.New("unexpected database error")
	dataDummy := FakeRepository{
		findErr:    pgx.ErrNoRows,
		executeErr: errUnexpected,
	}

	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
	}

	b, err := json.Marshal(inputTransfer)
	if err != nil {
		t.Fatalf("failed to serialize JSON: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/transfers", bytes.NewBuffer(b))
	r.Header.Set("Idempotency-Key", "idem-key-1")
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ExecuteTransfer(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected: %v, got: %v", http.StatusInternalServerError, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var errResp response.ErrorResponse
	err = json.NewDecoder(res.Body).Decode(&errResp)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Status != http.StatusInternalServerError {
		t.Errorf("expected error status 500, got %d", errResp.Error.Status)
	}

	if errResp.Error.Code != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected error code INTERNAL_SERVER_ERROR, got %s", errResp.Error.Code)
	}

	if errResp.Error.Message != "an unexpected error occurred" {
		t.Errorf("expected error message an unexpected error occurred, got %s", errResp.Error.Message)
	}

	if !dataDummy.executeCalled {
		t.Errorf("expected repository ExecuteTransfer to be called")
	}
}
