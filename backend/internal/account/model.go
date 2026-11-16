package account

import (
	"time"
)

type Account struct {
	ID            string     `json:"id"`
	AccountNumber string     `json:"account_number"`
	OwnerName     *string    `json:"owner_name"`
	Currency      *string    `json:"currency"`
	Balance       int64      `json:"balance"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     *time.Time `json:"updated_at"`
}
