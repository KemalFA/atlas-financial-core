package transfer

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

type FakeRepository struct {
	// FindByIdempotencyKey properties
	findCalled       bool
	findCallCount    int
	findReceived     Transfer
	transferSent     Transfer
	findErr          error
	findErrSequence  []error

	// ExecuteTransfer properties
	executeCalled   bool
	executeReceived Transfer
	executeErr      error
}

func (f *FakeRepository) FindByIdempotencyKey(ctx context.Context, transfer Transfer) (Transfer, error) {
	f.findCalled = true
	callIdx := f.findCallCount
	f.findCallCount++
	f.findReceived = transfer

	if callIdx < len(f.findErrSequence) {
		if f.findErrSequence[callIdx] != nil {
			return Transfer{}, f.findErrSequence[callIdx]
		}
		return f.transferSent, nil
	}

	if f.findErr != nil {
		return Transfer{}, f.findErr
	}

	return f.transferSent, nil
}

func (f *FakeRepository) ExecuteTransfer(ctx context.Context, transfer Transfer) (Transfer, error) {
	f.executeCalled = true
	f.executeReceived = transfer

	if f.executeErr != nil {
		return Transfer{}, f.executeErr
	}

	f.transferSent.ID = "fake-transfer-id"
	return f.transferSent, nil
}

func TestExecuteTransferService_Happy(t *testing.T) {
	now := time.Now()
	dataDummy := FakeRepository{
		findErr: pgx.ErrNoRows,
		transferSent: Transfer{
			AccountSenderID:   "sender-1",
			AccountReceiverID: "receiver-1",
			Amount:            50000,
			Purpose:           "payment",
			IdempotencyKey:    "idem-key-1",
			CreatedAt:         now,
		},
	}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	data, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err != nil {
		t.Fatalf("ExecuteTransfer() error = %v", err)
	}

	if !dataDummy.findCalled {
		t.Fatal("expected repository FindByIdempotencyKey to be called")
	}

	if !dataDummy.executeCalled {
		t.Fatal("expected repository ExecuteTransfer to be called")
	}

	if dataDummy.executeReceived.AccountSenderID != inputTransfer.AccountSenderID {
		t.Errorf("repository received sender ID %q, want %q", dataDummy.executeReceived.AccountSenderID, inputTransfer.AccountSenderID)
	}

	if dataDummy.executeReceived.AccountReceiverID != inputTransfer.AccountReceiverID {
		t.Errorf("repository received receiver ID %q, want %q", dataDummy.executeReceived.AccountReceiverID, inputTransfer.AccountReceiverID)
	}

	if dataDummy.executeReceived.Amount != inputTransfer.Amount {
		t.Errorf("repository received amount %d, want %d", dataDummy.executeReceived.Amount, inputTransfer.Amount)
	}

	if data.ID != "fake-transfer-id" {
		t.Errorf("returned ID = %q, want %q", data.ID, "fake-transfer-id")
	}
}

func TestExecuteTransferService_IdempotentReplay(t *testing.T) {
	now := time.Now()
	dataDummy := FakeRepository{
		transferSent: Transfer{
			ID:                "existing-transfer-id",
			AccountSenderID:   "sender-1",
			AccountReceiverID: "receiver-1",
			Amount:            50000,
			Purpose:           "payment",
			IdempotencyKey:    "idem-key-1",
			CreatedAt:         now,
		},
	}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	data, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err != nil {
		t.Fatalf("ExecuteTransfer() error = %v", err)
	}

	if !dataDummy.findCalled {
		t.Fatal("expected repository FindByIdempotencyKey to be called")
	}

	if dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer not to be called on idempotent replay")
	}

	if data.ID != "existing-transfer-id" {
		t.Errorf("returned ID = %q, want %q", data.ID, "existing-transfer-id")
	}
}

func TestExecuteTransferService_ConcurrentDuplicateKey(t *testing.T) {
	now := time.Now()
	dataDummy := FakeRepository{
		findErrSequence: []error{pgx.ErrNoRows, nil},
		executeErr:      ErrDuplicateIdempotencyKey,
		transferSent: Transfer{
			ID:                "concurrent-transfer-id",
			AccountSenderID:   "sender-1",
			AccountReceiverID: "receiver-1",
			Amount:            50000,
			Purpose:           "payment",
			IdempotencyKey:    "idem-key-1",
			CreatedAt:         now,
		},
	}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	data, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err != nil {
		t.Fatalf("ExecuteTransfer() error = %v", err)
	}

	if dataDummy.findCallCount != 2 {
		t.Errorf("expected FindByIdempotencyKey to be called 2 times, got %d", dataDummy.findCallCount)
	}

	if !dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer to be called")
	}

	if data.ID != "concurrent-transfer-id" {
		t.Errorf("returned ID = %q, want %q", data.ID, "concurrent-transfer-id")
	}
}

func TestExecuteTransferService_ConflictPayload(t *testing.T) {
	tests := []struct {
		name    string
		input   Transfer
		wantErr error
	}{
		{
			name: "sender id mismatch",
			input: Transfer{
				AccountSenderID:   "sender-other",
				AccountReceiverID: "receiver-1",
				Amount:            50000,
				Purpose:           "payment",
				IdempotencyKey:    "idem-key-1",
			},
			wantErr: ErrSenderIdNotMatch,
		},
		{
			name: "receiver id mismatch",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-other",
				Amount:            50000,
				Purpose:           "payment",
				IdempotencyKey:    "idem-key-1",
			},
			wantErr: ErrReceiverIdNotMatch,
		},
		{
			name: "amount mismatch",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-1",
				Amount:            75000,
				Purpose:           "payment",
				IdempotencyKey:    "idem-key-1",
			},
			wantErr: ErrAmountNotMatch,
		},
		{
			name: "purpose mismatch",
			input: Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-1",
				Amount:            50000,
				Purpose:           "different-purpose",
				IdempotencyKey:    "idem-key-1",
			},
			wantErr: ErrPurposeNotMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			serviceTest := NewService(&dataDummy)
			_, err := serviceTest.ExecuteTransfer(context.Background(), tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("expected error %v, got %v", tt.wantErr, err)
			}

			if !dataDummy.findCalled {
				t.Error("expected repository FindByIdempotencyKey to be called")
			}

			if dataDummy.executeCalled {
				t.Error("expected repository ExecuteTransfer not to be called")
			}
		})
	}
}

func TestExecuteTransferService_InsufficientBalance(t *testing.T) {
	dataDummy := FakeRepository{
		findErr:    pgx.ErrNoRows,
		executeErr: ErrInsufficientBalance,
	}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            500000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	_, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrInsufficientBalance) {
		t.Errorf("expected error %v, got %v", ErrInsufficientBalance, err)
	}

	if !dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer to be called")
	}
}

func TestExecuteTransferService_SelfTransfer(t *testing.T) {
	dataDummy := FakeRepository{}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "sender-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	_, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrSameAccountID) {
		t.Errorf("expected error %v, got %v", ErrSameAccountID, err)
	}

	if dataDummy.findCalled {
		t.Error("expected repository FindByIdempotencyKey not to be called")
	}

	if dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer not to be called")
	}
}

func TestExecuteTransferService_InvalidAmount(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{name: "zero amount", amount: 0},
		{name: "negative amount", amount: -1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDummy := FakeRepository{}

			serviceTest := NewService(&dataDummy)
			inputTransfer := Transfer{
				AccountSenderID:   "sender-1",
				AccountReceiverID: "receiver-1",
				Amount:            tt.amount,
				Purpose:           "payment",
				IdempotencyKey:    "idem-key-1",
			}

			_, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, ErrInvalidAmount) {
				t.Errorf("expected error %v, got %v", ErrInvalidAmount, err)
			}

			if dataDummy.findCalled {
				t.Error("expected repository FindByIdempotencyKey not to be called")
			}

			if dataDummy.executeCalled {
				t.Error("expected repository ExecuteTransfer not to be called")
			}
		})
	}
}

func TestExecuteTransferService_MissingIdempotencyKey(t *testing.T) {
	dataDummy := FakeRepository{}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "",
	}

	_, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrIdempotencyKeyMissing) {
		t.Errorf("expected error %v, got %v", ErrIdempotencyKeyMissing, err)
	}

	if dataDummy.findCalled {
		t.Error("expected repository FindByIdempotencyKey not to be called")
	}

	if dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer not to be called")
	}
}

func TestExecuteTransferService_RepositoryError(t *testing.T) {
	errRepository := errors.New("repository execute failed")
	dataDummy := FakeRepository{
		findErr:    pgx.ErrNoRows,
		executeErr: errRepository,
	}

	serviceTest := NewService(&dataDummy)
	inputTransfer := Transfer{
		AccountSenderID:   "sender-1",
		AccountReceiverID: "receiver-1",
		Amount:            50000,
		Purpose:           "payment",
		IdempotencyKey:    "idem-key-1",
	}

	_, err := serviceTest.ExecuteTransfer(context.Background(), inputTransfer)
	if !errors.Is(err, errRepository) {
		t.Errorf("ExecuteTransfer() error = %v, want %v", err, errRepository)
	}

	if !dataDummy.executeCalled {
		t.Error("expected repository ExecuteTransfer to be called")
	}
}
