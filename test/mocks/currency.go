package mocks

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/internal/currency"
	"github.com/stretchr/testify/mock"
)

// MockCurrencyRepository is a mock implementation of the currency repository
type MockCurrencyRepository struct {
	mock.Mock
}

func (m *MockCurrencyRepository) GetActiveCurrencies(ctx context.Context) ([]*currency.Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*currency.Currency), args.Error(1)
}

func (m *MockCurrencyRepository) GetCurrencyByCode(ctx context.Context, code string) (*currency.Currency, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*currency.Currency), args.Error(1)
}

func (m *MockCurrencyRepository) GetLatestExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*currency.ExchangeRate, error) {
	args := m.Called(ctx, fromCurrency, toCurrency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*currency.ExchangeRate), args.Error(1)
}

func (m *MockCurrencyRepository) GetExchangeRateByID(ctx context.Context, id uuid.UUID) (*currency.ExchangeRate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*currency.ExchangeRate), args.Error(1)
}

func (m *MockCurrencyRepository) CreateExchangeRate(ctx context.Context, rate *currency.ExchangeRate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockCurrencyRepository) BulkCreateExchangeRates(ctx context.Context, rates []*currency.ExchangeRate) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

func (m *MockCurrencyRepository) GetAllExchangeRatesFromBase(ctx context.Context, baseCurrency string) ([]*currency.ExchangeRate, error) {
	args := m.Called(ctx, baseCurrency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*currency.ExchangeRate), args.Error(1)
}

func (m *MockCurrencyRepository) InvalidateExchangeRates(ctx context.Context, fromCurrency string) error {
	args := m.Called(ctx, fromCurrency)
	return args.Error(0)
}

func (m *MockCurrencyRepository) CreateCurrency(ctx context.Context, curr *currency.Currency) error {
	args := m.Called(ctx, curr)
	return args.Error(0)
}

func (m *MockCurrencyRepository) UpdateCurrency(ctx context.Context, curr *currency.Currency) error {
	args := m.Called(ctx, curr)
	return args.Error(0)
}

func (m *MockCurrencyRepository) CleanupExpiredRates(ctx context.Context, olderThan time.Duration) (int64, error) {
	args := m.Called(ctx, olderThan)
	return args.Get(0).(int64), args.Error(1)
}
