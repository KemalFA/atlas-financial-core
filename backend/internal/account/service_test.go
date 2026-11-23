package account

import (
	"context"
	"errors"
	"testing"
	"time"
)

type FakeRepository struct {
	createCalled    bool
	accountReceived Account
	accountSent     Account
	createErr       error

	//Find By Account Number Properties
	findCalled            bool
	accountNumberReceived string
	findErr               error

	//List Accounts Properties
	listCalled bool
	dummyList  []Account
	listErr    error
}

func (f *FakeRepository) CreateAccount(ctx context.Context, account Account) (Account, error) {
	f.createCalled = true
	f.accountReceived = account

	f.accountSent.ID = "fake-id"
	f.accountSent.AccountNumber = f.accountReceived.AccountNumber

	return f.accountSent, f.createErr
}

func (f *FakeRepository) FindByAccountNumber(ctx context.Context, accountNumber string) (Account, error) {
	f.findCalled = true
	f.accountNumberReceived = accountNumber
	return f.accountSent, f.findErr
}

func (f *FakeRepository) ListAccounts(ctx context.Context) ([]Account, error) {
	f.listCalled = true
	return f.dummyList, f.listErr
}

func TestCreateAccountService_Happy(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "idr ",
		InitialDeposit: 150000,
	}

	data, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err != nil {
		t.Fatalf("CreateAccount() error = %v", err)
	}

	if !dataDummy.createCalled {
		t.Fatal("expected repository CreateAccount to be called")
	}

	if dataDummy.accountReceived.OwnerName != "Test" {
		t.Errorf("repository received owner name %q, want %q",
			dataDummy.accountReceived.OwnerName, "Test")
	}

	if dataDummy.accountReceived.Currency != "IDR" {
		t.Errorf("repository received currency %q, want %q",
			dataDummy.accountReceived.Currency, "IDR")
	}

	if dataDummy.accountReceived.Balance != data.Balance {
		t.Errorf("repository received balance %d, want %d",
			dataDummy.accountReceived.Balance, data.Balance)
	}

	if len(dataDummy.accountReceived.AccountNumber) != 10 {
		t.Errorf("account number length = %d, want 10",
			len(dataDummy.accountReceived.AccountNumber))
	}

	if dataDummy.accountReceived.ID != "" {
		t.Errorf("expected ID: nil, got: %s", dataDummy.accountReceived.ID)
	}

	if data.ID != "fake-id" {
		t.Errorf("returned ID = %q, want %q", data.ID, "fake-id")
	}

	if data.AccountNumber != dataDummy.accountReceived.AccountNumber {
		t.Errorf("expected account number: %s, got %s", dataDummy.accountReceived.AccountNumber, data.AccountNumber)
	}
}

func TestCreateAccountService_EmptyOwnerName(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "",
		Currency:       "IDR",
		InitialDeposit: 150000,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err == nil {
		t.Fatalf("CreateAccount() success = %v", err)
	}

	if errors.Is(err, ErrOwnerName) != true {
		t.Errorf("expected: %v", ErrOwnerName)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: False")
	}
}

func TestCreateAccountService_InvalidCurrencies(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "idrs",
		InitialDeposit: 150000,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err == nil {
		t.Fatalf("CreateAccount() success = %v", err)
	}

	if errors.Is(err, ErrInvalidCurrencies) != true {
		t.Errorf("expected: %v", ErrInvalidCurrencies)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: False")
	}
}

func TestCreateAccountService_EmptyCurrencies(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "",
		InitialDeposit: 150000,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err == nil {
		t.Fatalf("CreateAccount() success = %v", err)
	}

	if errors.Is(err, ErrEmptyCurrencies) != true {
		t.Errorf("expected: %v", ErrEmptyCurrencies)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: false")
	}
}

func TestCreateAccountService_NotIDR(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "USD",
		InitialDeposit: 150000,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err == nil {
		t.Fatalf("CreateAccount() success = %v", err)
	}

	if errors.Is(err, ErrNotIDR) != true {
		t.Errorf("expected: %v", ErrNotIDR)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: False")
	}
}

func TestCreateAccountService_InitialDepositBelowMinimum(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "IDR",
		InitialDeposit: 99999,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if err == nil {
		t.Fatalf("CreateAccount() success = %v", err)
	}

	if errors.Is(err, ErrInitialDeposit) != true {
		t.Errorf("expected: %v", ErrInitialDeposit)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: False")
	}
}

func TestCreateAccountService_RepositoryError(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}
	errRepository := errors.New("repository create failed")
	dataDummy.createErr = errRepository

	serviceTest := NewService(&dataDummy)
	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "IDR",
		InitialDeposit: 150000,
	}
	_, err := serviceTest.CreateAccount(context.Background(), inputTestAccount)
	if errors.Is(err, errRepository) != true {
		t.Errorf("CreateAccount() error = %v", err)
	}

	if dataDummy.createCalled != true {
		t.Errorf("expected: true")
	}
}

func TestFindByAccountNumberService(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			AccountNumber: "8801123456",
			OwnerName:     "Test",
			Currency:      "IDR",
			Balance:       150000,
			CreatedAt:     time.Now(),
		},
	}

	serviceTest := NewService(&dataDummy)
	accountNumber := "8801123456 "

	data, err := serviceTest.FindByAccountNumber(context.Background(), accountNumber)
	if err != nil {
		t.Fatalf("FindByAccountNumber(), unsucceed")
	}

	if dataDummy.findCalled != true {
		t.Fatal("expected repository FindByAccountNumber to be called")
	}

	if dataDummy.accountNumberReceived != "8801123456" {
		t.Errorf("number is untrimmed")
	}

	if data.AccountNumber != "8801123456" {
		t.Errorf("account number mismatch")
	}
}

func TestFindByAccountNumberService_EmptyAccountNumber(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			AccountNumber: "8801123456",
			OwnerName:     "Test",
			Currency:      "IDR",
			Balance:       150000,
			CreatedAt:     time.Now(),
		},
	}

	serviceTest := NewService(&dataDummy)
	accountNumber := " "

	_, err := serviceTest.FindByAccountNumber(context.Background(), accountNumber)
	if err == nil {
		t.Fatalf("FindByAccountNumber(), succeed")
	}

	if errors.Is(err, ErrEmptyAccountNumber) != true {
		t.Errorf("expected: %v", err)
	}

	if dataDummy.findCalled != false {
		t.Errorf("find repository call expected false")
	}
}

func TestFindByAccountNumberService_AccountNotFound(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			AccountNumber: "8801123456",
			OwnerName:     "Test",
			Currency:      "IDR",
			Balance:       150000,
			CreatedAt:     time.Now(),
		},
	}

	serviceTest := NewService(&dataDummy)
	accountNumber := "8801123452"
	dataDummy.findErr = ErrAccountNotFound

	_, err := serviceTest.FindByAccountNumber(context.Background(), accountNumber)
	if err == nil {
		t.Fatalf("FindByAccountNumber(), succeed")
	}

	if errors.Is(err, ErrAccountNotFound) != true {
		t.Errorf("expected: %v", err)
	}

	if dataDummy.findCalled != true {
		t.Errorf("find repository call expected false")
	}
}

func TestFindByAccountNumberService_RepositoryError(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			AccountNumber: "8801123456",
			OwnerName:     "Test",
			Currency:      "IDR",
			Balance:       150000,
			CreatedAt:     time.Now(),
		},
	}

	serviceTest := NewService(&dataDummy)
	errRepository := errors.New("Repository Failed")
	dataDummy.findErr = errRepository
	accountNumber := "8801123456"

	_, err := serviceTest.FindByAccountNumber(context.Background(), accountNumber)
	if errors.Is(err, errRepository) != true {
		t.Errorf("FindByAccountNumber() error = %v", err)
	}

	if dataDummy.findCalled != true {
		t.Errorf("expected: true")
	}
}

func TestListAccounts_Happy(t *testing.T) {
	dataDummy := FakeRepository{
		dummyList: []Account{
			{
				AccountNumber: "8801123456",
				OwnerName:     "Test",
				Currency:      "IDR",
				Balance:       150000,
				CreatedAt:     time.Now(),
			},
			{
				AccountNumber: "8801654321",
				OwnerName:     "Test2",
				Currency:      "IDR",
				Balance:       150000,
				CreatedAt:     time.Now(),
			},
		},
	}

	serviceTest := NewService(&dataDummy)
	data, err := serviceTest.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts() repository failed")
	}

	if len(data) != len(dataDummy.dummyList) {
		t.Fatalf("got %d accounts, want %d", len(data), len(dataDummy.dummyList))
	}

	for i, want := range dataDummy.dummyList {
		got := data[i]
		if got.AccountNumber != want.AccountNumber {
			t.Errorf("index %d: got AccountNumber %q, want %q", i, got.AccountNumber, want.AccountNumber)
		}
		if got.OwnerName != want.OwnerName {
			t.Errorf("index %d: got OwnerName %q, want %q", i, got.OwnerName, want.OwnerName)
		}
		if got.Balance != want.Balance {
			t.Errorf("index %d: got Balance %d, want %d", i, got.Balance, want.Balance)
		}
	}

	if dataDummy.listCalled != true {
		t.Errorf("list expected called true")
	}

}

func TestListAccounts_RepositoryError(t *testing.T) {
	dataDummy := FakeRepository{
		dummyList: []Account{
			{
				AccountNumber: "8801123456",
				OwnerName:     "Test",
				Currency:      "IDR",
				Balance:       150000,
				CreatedAt:     time.Now(),
			},
			{
				AccountNumber: "8801654321",
				OwnerName:     "Test2",
				Currency:      "IDR",
				Balance:       150000,
				CreatedAt:     time.Now(),
			},
		},
	}

	serviceTest := NewService(&dataDummy)
	ErrRepositoryFailed := errors.New("list accounts repository failed")
	dataDummy.listErr = ErrRepositoryFailed
	_, err := serviceTest.ListAccounts(context.Background())
	if errors.Is(err, ErrRepositoryFailed) != true {
		t.Errorf("ListAccounts() error: %v", err)
	}
	if dataDummy.listCalled != true {
		t.Errorf("list expected called true")
	}
}
