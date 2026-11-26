package transfer

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) FindByIdempotencyKey(ctx context.Context, transfer Transfer) (Transfer, error) {
	if transfer.IdempotencyKey == "" {
		return Transfer{}, ErrIdempotencyKeyMissing
	}

	if transfer.AccountSenderID == transfer.AccountReceiverID {
		return Transfer{}, ErrSameAccountID
	}

	if transfer.Amount <= 0 {
		return Transfer{}, ErrInvalidAmount
	}

	data, err := s.repo.FindByIdempotencyKey(ctx, transfer)
	if err != nil {
		return data, err
	}

	if data.AccountSenderID != transfer.AccountSenderID {
		return Transfer{}, ErrSenderIdNotMatch
	}

	if data.AccountReceiverID != transfer.AccountReceiverID {
		return Transfer{}, ErrReceiverIdNotMatch
	}

	if data.Amount != transfer.Amount {
		return Transfer{}, ErrAmountNotMatch
	}

	if data.Purpose != transfer.Purpose {
		return Transfer{}, ErrPurposeNotMatch
	}
	return data, nil
}

func (s *Service) ExecuteTransfer(ctx context.Context, transfer Transfer) (Transfer, error) {
	data, err := s.FindByIdempotencyKey(ctx, transfer)
	if err == nil {
		return data, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return s.repo.ExecuteTransfer(ctx, transfer)
	}

	return Transfer{}, err

}
