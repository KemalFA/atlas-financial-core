package account

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/text/currency"
)

var (
	ErrOwnerName  = fmt.Errorf("Name Cant Be Empty")
	ErrCurrencies = fmt.Errorf("Currencies is not Valid")
	ErrBalance    = fmt.Errorf("Balance Cant Be Negative")
)

type InputAccount struct {
	OwnerName string `json:"owner_name"`
	Currency  string `json:"currency"`
	Balance   int64  `json:"balance"`
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func IsValidCurrency(code string) bool {

	_, err := currency.ParseISO(code)
	return err == nil
}

func GenerateAccountNumber() (string, error) {
	const AccountPrefix = "8801"
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("failed to generate random account number: %w", err)
	}
	accNum := fmt.Sprintf("%s%06d", AccountPrefix, n.Int64())
	if len(accNum) != 10 {
		return "", fmt.Errorf("invalid generated account number length: %d", len(accNum))
	}

	return accNum, nil
}

func (s *Service) CreateAccount(ctx context.Context, input InputAccount) (Account, error) {
	var err error
	trimmed := strings.ToUpper(strings.TrimSpace(input.OwnerName))
	currencies := IsValidCurrency(input.Currency)
	input.OwnerName = trimmed

	if trimmed == "" {
		return Account{}, ErrOwnerName
	}

	if input.Currency == "" {
		return Account{}, fmt.Errorf("Currency Cant Be Empty")
	}

	if !currencies {
		return Account{}, ErrCurrencies
	}

	if input.Balance < 0 {
		return Account{}, ErrBalance
	}

	AccNum, err := GenerateAccountNumber()
	if AccNum == "" {
		return Account{}, err
	}

	NewAccount := Account{
		AccountNumber: AccNum,
		OwnerName:     trimmed,
		Currency:      input.Currency,
		Balance:       input.Balance,
	}

	result, err := s.repo.CreateAccount(ctx, NewAccount)
	if err != nil {
		return Account{}, fmt.Errorf("Failed to CreateAccount: %w", err)
	}

	return result, nil
}
