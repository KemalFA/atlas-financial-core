package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


type Repository interface {
	CreateAccount(ctx context.Context, account Account) (Account, error)
	FindByAccountNumber(ctx context.Context, accounNumber string) (Account, error)
	ListAccounts(ctx context.Context) ([]Account, error)
}

type DataRepository struct {
	db *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *DataRepository {
	return &DataRepository{db: pool}
}

// repository to create an account
func (r *DataRepository) CreateAccount(ctx context.Context, account Account) (Account, error) {
	query := `INSERT INTO accounts (account_number, owner_name, currency, balance) VALUES ($1, $2, $3, $4) RETURNING id, account_number, owner_name, currency, balance, created_at, updated_at`
	rows, err := r.db.Query(ctx, query, account.AccountNumber, account.OwnerName, account.Currency, account.Balance)
	if err != nil {
		return Account{}, fmt.Errorf("unable to insert account: %w", err)
	}

	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Account])
	if err != nil {
		return Account{}, fmt.Errorf("unable to scan account: %w", err)
	}

	return data, nil
}

func (r *DataRepository) FindByAccountNumber(ctx context.Context, accountNumber string) (Account, error) {
	query := `SELECT id, account_number, owner_name, currency, balance, created_at, updated_at FROM accounts WHERE account_number = $1`
	rows, err := r.db.Query(ctx, query, accountNumber)
	if err != nil {
		return Account{}, fmt.Errorf("failed to query account: %w", err)
	}
	data, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[Account])
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrAccountNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("unable to scan account: %w", err)
	}
	return data, nil
}

func (r *DataRepository) ListAccounts(ctx context.Context) ([]Account, error) {
	query := `SELECT id, account_number, owner_name, currency, balance, created_at, updated_at FROM accounts ORDER BY created_at, id`
	rows, err := r.db.Query(ctx, query)

	if err != nil {
		return nil, fmt.Errorf("failed to query accounts: %w", err)
	}

	data, err := pgx.CollectRows(rows, pgx.RowToStructByName[Account])
	if err != nil {
		return nil, fmt.Errorf("unable to scan accounts: %w", err)
	}

	return data, nil
}
