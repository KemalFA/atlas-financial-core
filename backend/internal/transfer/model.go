package transfer

import "time"

type Transfer struct {
	ID                string `json:"id"`
	AccountSenderID   string `json:"account_sender_id"`
	AccountReceiverID string `json:"account_receiver_id"`
	Amount            int64  `json:"amount"`
	Purpose           string `json:"purpose"`
	IdempotencyKey    string
	CreatedAt         time.Time `json:"created_at"`
}
