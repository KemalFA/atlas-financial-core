package transfer

import (
	"errors"
	"time"
)

var (
	ErrAccountNotFound         = errors.New("sender or receiver account not found")
	ErrCurrenciesNotMatch      = errors.New("currencies not match")
	ErrInsufficientBalance     = errors.New("insufficient balance")
	ErrIdempotencyKeyMissing   = errors.New("idempotency key is missing")
	ErrSameAccountID           = errors.New("cant transfer to the same account")
	ErrInvalidAmount           = errors.New("cant transfer negative or 0 value")
	ErrSenderIdNotMatch        = errors.New("account sender id not match")
	ErrReceiverIdNotMatch      = errors.New("account receiver id not match")
	ErrAmountNotMatch          = errors.New("amount not match")
	ErrPurposeNotMatch         = errors.New("purpose not match")
	ErrDuplicateIdempotencyKey = errors.New("duplicate idempotency key")
)

type Transfer struct {
	ID                string    `json:"id"`
	AccountSenderID   string    `json:"account_sender_id"`
	AccountReceiverID string    `json:"account_receiver_id"`
	Amount            int64     `json:"amount"`
	Purpose           string    `json:"purpose"`
	IdempotencyKey    string    `json:"-"`
	CreatedAt         time.Time `json:"created_at"`
}
