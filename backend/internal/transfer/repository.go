package transfer

import (
	"atlas-financial-core/internal/account"
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	FindByIdempotencyKey(ctx context.Context, transfer Transfer) (Transfer, error)
	ExecuteTransfer(ctx context.Context, transfer Transfer) (Transfer, error)
}

type DataRepository struct {
	db *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *DataRepository {
	return &DataRepository{db: pool}
}

func (r *DataRepository) FindByIdempotencyKey(ctx context.Context, transfer Transfer) (Transfer, error) {
	query := `SELECT id, account_sender_id, account_receiver_id, amount, purpose, idempotency_key, created_at FROM transfers WHERE idempotency_key = $1`
	rows, err := r.db.Query(ctx, query, transfer.IdempotencyKey)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to query %w", err)
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Transfer])

	if err != nil {
		return Transfer{}, fmt.Errorf("unexpected error %w", err)
	}

	return data, nil
}

// repository to execute transfer transactions
func (r *DataRepository) ExecuteTransfer(ctx context.Context, transfer Transfer) (Transfer, error) {
	var senderBalance int64
	var receiverBalance int64

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to begin a transaction %w", err)
	}
	defer tx.Rollback(ctx)

	//where transactions happened
	//lock the row of sender dan receiver
	query := `SELECT id, account_number, owner_name, currency, balance, created_at, updated_at FROM accounts WHERE id IN ($1, $2) ORDER BY id ASC FOR UPDATE`
	rows, err := tx.Query(ctx, query, transfer.AccountSenderID, transfer.AccountReceiverID)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to query %w", err)
	}

	dataTx, err := pgx.CollectRows(rows, pgx.RowToStructByName[account.Account])

	if err != nil {
		return Transfer{}, fmt.Errorf("db error %w", err)
	}

	if len(dataTx) != 2 {
		return Transfer{}, ErrAccountNotFound
	}

	for _, validate := range dataTx {
		if validate.ID == transfer.AccountSenderID {
			if validate.Currency != "IDR" {
				return Transfer{}, ErrCurrenciesNotMatch
			}
			if transfer.Amount > validate.Balance {
				return Transfer{}, ErrInsufficientBalance
			}

			senderBalance = validate.Balance - transfer.Amount
		}

		if validate.ID == transfer.AccountReceiverID {
			if validate.Currency != "IDR" {
				return Transfer{}, ErrCurrenciesNotMatch
			}

			receiverBalance = validate.Balance + transfer.Amount
		}
	}

	query = `UPDATE accounts SET balance = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err = tx.Exec(ctx, query, transfer.AccountSenderID, senderBalance)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to query %w", err)
	}

	query = `UPDATE accounts SET balance = $2, updated_at = CURRENT_TIMESTAMP WHERE id = $1`
	_, err = tx.Exec(ctx, query, transfer.AccountReceiverID, receiverBalance)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to query %w", err)
	}

	queryInsert := `INSERT INTO transfers (account_sender_id, account_receiver_id, amount, purpose, idempotency_key) VALUES ($1, $2, $3, $4, $5) RETURNING id, account_sender_id, account_receiver_id, amount, purpose, idempotency_key, created_at`
	rows, err = tx.Query(ctx, queryInsert, transfer.AccountSenderID, transfer.AccountReceiverID, transfer.Amount, transfer.Purpose, transfer.IdempotencyKey)
	if err != nil {
		return Transfer{}, fmt.Errorf("unable to insert transfer: %w", err)
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Transfer])
	if err != nil {
		return Transfer{}, fmt.Errorf("unable to scan transfer: %w", err)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to commit %w", err)
	}

	return data, nil
}
