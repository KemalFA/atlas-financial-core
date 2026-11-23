CREATE TABLE transfers (
  id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
  account_sender_id UUID NOT NULL REFERENCES accounts(id),
  account_receiver_id UUID NOT NULL REFERENCES accounts(id),
  idempotency_key UUID NOT NULL UNIQUE,
  amount BIGINT NOT NULL,
  purpose varchar(100) NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CONSTRAINT amount_positive CHECK (amount > 0),
  CONSTRAINT transfer_check CHECK (account_sender_id != account_receiver_id)
);