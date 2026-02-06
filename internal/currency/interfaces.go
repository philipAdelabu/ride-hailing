package currency

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for currency repository operations
type RepositoryInterface interface {
	GetActiveCurrencies(ctx context.Context) ([]*Currency, error)
	GetCurrencyByCode(ctx context.Context, code string) (*Currency, error)
	GetLatestExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRate, error)
	GetExchangeRateByID(ctx context.Context, id uuid.UUID) (*ExchangeRate, error)
	CreateExchangeRate(ctx context.Context, rate *ExchangeRate) error
	BulkCreateExchangeRates(ctx context.Context, rates []*ExchangeRate) error
	GetAllExchangeRatesFromBase(ctx context.Context, baseCurrency string) ([]*ExchangeRate, error)
	InvalidateExchangeRates(ctx context.Context, fromCurrency string) error
	CreateCurrency(ctx context.Context, currency *Currency) error
	UpdateCurrency(ctx context.Context, currency *Currency) error
	CleanupExpiredRates(ctx context.Context, olderThan time.Duration) (int64, error)
}
