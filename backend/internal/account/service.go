package account

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"golang.org/x/text/currency"
)

var (
	ErrOwnerName          = errors.New("Name Cant Be Empty")
	ErrInvalidCurrencies  = errors.New("Currencies is not Valid")
	ErrEmptyCurrencies    = errors.New("Currency Cant Be Empty")
	ErrInitialDeposit     = errors.New("Minimum Deposit Must Be Rp100.000")
	ErrEmptyAccountNumber = errors.New("Account Number Cant Be Empty")
	ErrNotIDR             = errors.New("Currencies Available is Only IDR")
	ErrAccountNotFound    = errors.New("account not found")
)

type InputCreateAccount struct {
	OwnerName      string `json:"owner_name"`
	Currency       string `json:"currency"`
	InitialDeposit int64  `json:"initial_deposit"`
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

// Create Account Service
func (s *Service) CreateAccount(ctx context.Context, input InputCreateAccount) (Account, error) {
	var err error
	trimmedOwnerName := strings.TrimSpace(input.OwnerName)
	input.OwnerName = trimmedOwnerName

	fixedCurrencies := strings.ToUpper(strings.TrimSpace(input.Currency))
	currencies := IsValidCurrency(fixedCurrencies)

	if trimmedOwnerName == "" {
		return Account{}, ErrOwnerName
	}

	if fixedCurrencies == "" {
		return Account{}, ErrEmptyCurrencies
	}

	if !currencies {
		return Account{}, ErrInvalidCurrencies
	}

	if fixedCurrencies != "IDR" {
		return Account{}, ErrNotIDR
	}

	if input.InitialDeposit < 100000 {
		return Account{}, ErrInitialDeposit
	}

	accNum, err := GenerateAccountNumber()
	if err != nil {
		return Account{}, err
	}

	newAccount := Account{
		AccountNumber: accNum,
		OwnerName:     trimmedOwnerName,
		Currency:      fixedCurrencies,
		Balance:       input.InitialDeposit,
	}

	create, err := s.repo.CreateAccount(ctx, newAccount)
	if err != nil {
		return Account{}, fmt.Errorf("failed to create account: %w", err)
	}

	return create, nil
}

// FInd by Account Number Service
func (s *Service) FindByAccountNumber(ctx context.Context, accountNumber string) (Account, error) {
	trimmedAccNum := strings.TrimSpace(accountNumber)
	if trimmedAccNum == "" {
		return Account{}, ErrEmptyAccountNumber
	}

	find, err := s.repo.FindByAccountNumber(ctx, trimmedAccNum)

	if err != nil {
		return Account{}, fmt.Errorf("failed to search account: %w", err)
	}

	return find, nil
}

func (s *Service) ListAccounts(ctx context.Context) ([]Account, error) {
	list, err := s.repo.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get list account: %w", err)
	}

	return list, nil
}
