package account

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"atlas-financial-core/internal/response"
)

func TestCreateHandler_Happy(t *testing.T) {
	var account Account
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}

	inputTestAccount := InputCreateAccount{
		OwnerName:      "Test ",
		Currency:       "idr ",
		InitialDeposit: 150000,
	}

	b, err := json.Marshal(inputTestAccount)
	if err != nil {
		t.Fatalf("Failed to Serialize to JSON from native Go struct type: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.CreateHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		t.Errorf("expected: 201, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	err = json.NewDecoder(res.Body).Decode(&account)
	if err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if account.OwnerName != dataDummy.accountSent.OwnerName {
		t.Errorf("owner name is not match")
	}

	if account.Currency != dataDummy.accountSent.Currency {
		t.Errorf("currencies is not match")
	}

	if account.Balance != dataDummy.accountSent.Balance {
		t.Errorf("balance is not match")
	}

	if dataDummy.createCalled != true {
		t.Errorf("expected to be called")
	}
}

func TestCreateHandler_JSONError(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}

	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBufferString("{invalid"))
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.CreateHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected: 400, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var response response.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error.Status != http.StatusBadRequest {
		t.Errorf("expected error status 400, got %d", response.Error.Status)
	}

	if response.Error.Code != "JSON_ERROR_DECODE" {
		t.Errorf("expected error code JSON_ERROR_DECODE, got %s", response.Error.Code)
	}

	if response.Error.Message != "JSON decoding error" {
		t.Errorf("expected error message JSON decoding error, got %s", response.Error.Message)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: false")
	}
}

func TestCreateHandler_EmptyOwnerName(t *testing.T) {
	dataDummy := FakeRepository{
		accountSent: Account{
			OwnerName: "Test",
			Currency:  "IDR",
			Balance:   150000,
			CreatedAt: time.Now(),
		},
	}

	inputTestAccount := InputCreateAccount{
		OwnerName:      "",
		Currency:       "idr ",
		InitialDeposit: 150000,
	}

	b, err := json.Marshal(inputTestAccount)
	if err != nil {
		t.Fatalf("Failed to Serialize to JSON from native Go struct type: %v", err)
	}

	r := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewBuffer(b))
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.CreateHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected: 400, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var response response.ErrorResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error.Status != http.StatusBadRequest {
		t.Errorf("expected error status 400, got %d", response.Error.Status)
	}

	if response.Error.Code != "VALIDATION_ERROR" {
		t.Errorf("expected error code VALIDATION_ERROR, got %s", response.Error.Code)
	}

	if response.Error.Message != ErrOwnerName.Error() {
		t.Errorf("expected error message %s, got %s", ErrOwnerName.Error(), response.Error.Message)
	}

	if dataDummy.createCalled != false {
		t.Errorf("expected: false")
	}
}

func TestFindByAccountNumberHandler_Happy(t *testing.T) {
	var account Account
	dataDummy := FakeRepository{
		accountSent: Account{
			AccountNumber: "8801123456",
			OwnerName:     "Test",
			Currency:      "IDR",
			Balance:       150000,
			CreatedAt:     time.Now(),
		},
	}

	pathURL := "/accounts/8801123456"

	r := httptest.NewRequest(http.MethodGet, pathURL, nil)
	w := httptest.NewRecorder()

	r.SetPathValue("account_number", "8801123456")

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.FindByAccountNumberHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected: 200, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	err := json.NewDecoder(res.Body).Decode(&account)
	if err != nil {
		t.Fatalf("unable to decode json %v", err)
	}

	if dataDummy.accountNumberReceived != "8801123456" {
		t.Errorf("expected: 8801123456, got: %v", dataDummy.accountNumberReceived)
	}

	if dataDummy.accountSent.AccountNumber != account.AccountNumber {
		t.Errorf("expected: %v, got: %v", dataDummy.accountSent.AccountNumber, account.AccountNumber)
	}

	if dataDummy.accountSent.OwnerName != account.OwnerName {
		t.Errorf("expected: %v, got: %v", dataDummy.accountSent.OwnerName, account.OwnerName)
	}

	if dataDummy.findCalled != true {
		t.Errorf("expected to be called")
	}
}

func TestFindByAccountNumberHandler_AccountNotFound(t *testing.T) {
	dataDummy := FakeRepository{
		findErr: ErrAccountNotFound,
	}

	pathURL := "/accounts/8801123456"

	r := httptest.NewRequest(http.MethodGet, pathURL, nil)
	w := httptest.NewRecorder()

	r.SetPathValue("account_number", "8801123456")

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.FindByAccountNumberHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Errorf("expected: %v, got: %v", http.StatusNotFound, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var response response.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error.Status != http.StatusNotFound {
		t.Errorf("expected error status 404, got %d", response.Error.Status)
	}

	if response.Error.Code != "ACCOUNT_NOT_FOUND" {
		t.Errorf("expected error code ACCOUNT_NOT_FOUND, got %s", response.Error.Code)
	}

	if response.Error.Message != "failed to search account: account not found" {
		t.Errorf("expected error message failed to search account: account not found, got %s", response.Error.Message)
	}

	if dataDummy.findCalled != true {
		t.Errorf("expected to be called")
	}
}

func TestListAccountsHandler_Happy(t *testing.T) {
	var account []Account
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
	r := httptest.NewRequest(http.MethodGet, "/accounts", nil)
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ListAccountsHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	err := json.NewDecoder(res.Body).Decode(&account)
	if err != nil {
		t.Fatalf("unable to decode json %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected: 200, got: %v", res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}
	if len(account) != len(dataDummy.dummyList) {
		t.Fatalf("got %d accounts, want %d", len(account), len(dataDummy.dummyList))
	}

	for i, want := range dataDummy.dummyList {
		got := account[i]
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
		t.Errorf("expected to be called")
	}
}

func TestListAccounts_Err(t *testing.T) {
	errListAccount := errors.New("list account error")
	dataDummy := FakeRepository{
		listErr: errListAccount,
	}

	pathURL := "/accounts"

	r := httptest.NewRequest(http.MethodGet, pathURL, nil)
	w := httptest.NewRecorder()

	serviceTest := NewService(&dataDummy)
	handlerTest := NewHandler(serviceTest)

	handlerTest.ListAccountsHandler(w, r)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("expected: %v, got: %v", http.StatusInternalServerError, res.StatusCode)
	}

	if res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("content type header is not application/json")
	}

	var response response.ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if response.Error.Status != http.StatusInternalServerError {
		t.Errorf("expected error status 500, got %d", response.Error.Status)
	}

	if response.Error.Code != "INTERNAL_SERVER_ERROR" {
		t.Errorf("expected error code INTERNAL_SERVER_ERROR, got %s", response.Error.Code)
	}

	if response.Error.Message != "an unexpected error occurred" {
		t.Errorf("expected error message an unexpected error occurred, got %s", response.Error.Message)
	}

	if dataDummy.listCalled != true {
		t.Errorf("expected to be called")
	}
}
