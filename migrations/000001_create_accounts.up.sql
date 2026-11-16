CREATE TABLE accounts (
    id UUID PRIMARY KEY,
    account_number VARCHAR(20) NOT NULL UNIQUE,
    owner_name VARCHAR(100),
    currency VARCHAR(10),
    balance BIGINT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    CONSTRAINT balance_positive CHECK (balance >= 0)
);