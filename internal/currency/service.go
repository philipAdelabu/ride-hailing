package currency

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Service handles currency business logic
type Service struct {
	repo         *Repository
	converter    *Converter
	baseCurrency string
	cache        *rateCache
}

// rateCache provides in-memory caching for exchange rates
type rateCache struct {
	mu    sync.RWMutex
	rates map[string]*ExchangeRate
	ttl   time.Duration
}

// NewService creates a new currency service
func NewService(repo *Repository, baseCurrency string) *Service {
	if baseCurrency == "" {
		baseCurrency = CurrencyUSD
	}

	return &Service{
		repo:         repo,
		converter:    NewConverter(baseCurrency),
		baseCurrency: baseCurrency,
		cache: &rateCache{
			rates: make(map[string]*ExchangeRate),
			ttl:   5 * time.Minute,
		},
	}
}

// GetActiveCurrencies returns all active currencies
func (s *Service) GetActiveCurrencies(ctx context.Context) ([]*Currency, error) {
	return s.repo.GetActiveCurrencies(ctx)
}

// GetCurrency returns a currency by code
func (s *Service) GetCurrency(ctx context.Context, code string) (*Currency, error) {
	return s.repo.GetCurrencyByCode(ctx, code)
}

// GetExchangeRate returns the latest exchange rate between two currencies
func (s *Service) GetExchangeRate(ctx context.Context, from, to string) (*ExchangeRate, error) {
	// Same currency - return 1:1 rate
	if from == to {
		return &ExchangeRate{
			ID:           uuid.Nil,
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         1.0,
			InverseRate:  1.0,
			Source:       "identity",
			FetchedAt:    time.Now(),
			ValidUntil:   time.Now().Add(24 * time.Hour),
		}, nil
	}

	// Check cache first
	cacheKey := fmt.Sprintf("%s-%s", from, to)
	s.cache.mu.RLock()
	if cached, ok := s.cache.rates[cacheKey]; ok {
		if cached.ValidUntil.After(time.Now()) {
			s.cache.mu.RUnlock()
			return cached, nil
		}
	}
	s.cache.mu.RUnlock()

	// Try direct rate
	rate, err := s.repo.GetLatestExchangeRate(ctx, from, to)
	if err == nil {
		s.cacheRate(rate)
		return rate, nil
	}

	// Try inverse rate
	inverseRate, err := s.repo.GetLatestExchangeRate(ctx, to, from)
	if err == nil {
		// Create a rate from the inverse
		rate = &ExchangeRate{
			ID:           inverseRate.ID,
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         inverseRate.InverseRate,
			InverseRate:  inverseRate.Rate,
			Source:       inverseRate.Source,
			FetchedAt:    inverseRate.FetchedAt,
			ValidUntil:   inverseRate.ValidUntil,
			CreatedAt:    inverseRate.CreatedAt,
		}
		s.cacheRate(rate)
		return rate, nil
	}

	// Try triangulation via base currency
	if from != s.baseCurrency && to != s.baseCurrency {
		fromToBase, err := s.GetExchangeRate(ctx, from, s.baseCurrency)
		if err != nil {
			return nil, fmt.Errorf("no rate path found from %s to %s", from, to)
		}

		baseToTarget, err := s.GetExchangeRate(ctx, s.baseCurrency, to)
		if err != nil {
			return nil, fmt.Errorf("no rate path found from %s to %s", from, to)
		}

		// Calculate triangulated rate
		triangulatedRate := fromToBase.Rate * baseToTarget.Rate

		rate = &ExchangeRate{
			ID:           uuid.Nil,
			FromCurrency: from,
			ToCurrency:   to,
			Rate:         triangulatedRate,
			InverseRate:  1 / triangulatedRate,
			Source:       "triangulated",
			FetchedAt:    time.Now(),
			ValidUntil:   minTime(fromToBase.ValidUntil, baseToTarget.ValidUntil),
		}
		s.cacheRate(rate)
		return rate, nil
	}

	return nil, fmt.Errorf("no exchange rate found for %s to %s", from, to)
}

// Convert converts an amount from one currency to another
func (s *Service) Convert(ctx context.Context, amount float64, from, to string) (*ConversionResult, error) {
	if from == to {
		return &ConversionResult{
			Original:     Money{Amount: amount, Currency: from},
			Converted:    Money{Amount: amount, Currency: to},
			ExchangeRate: 1.0,
			ConvertedAt:  time.Now(),
		}, nil
	}

	rate, err := s.GetExchangeRate(ctx, from, to)
	if err != nil {
		return nil, err
	}

	// Get target currency for proper rounding
	toCurrency, err := s.repo.GetCurrencyByCode(ctx, to)
	if err != nil {
		// Default to 2 decimal places if currency not found
		toCurrency = &Currency{Code: to, DecimalPlaces: 2}
	}

	convertedAmount := s.converter.Convert(amount, rate, RoundingModeStandard, toCurrency.DecimalPlaces)

	return &ConversionResult{
		Original:       Money{Amount: amount, Currency: from},
		Converted:      Money{Amount: convertedAmount, Currency: to},
		ExchangeRate:   rate.Rate,
		ExchangeRateID: rate.ID,
		ConvertedAt:    time.Now(),
	}, nil
}

// ConvertToBase converts an amount to the base currency
func (s *Service) ConvertToBase(ctx context.Context, amount float64, from string) (*ConversionResult, error) {
	return s.Convert(ctx, amount, from, s.baseCurrency)
}

// ConvertFromBase converts an amount from the base currency
func (s *Service) ConvertFromBase(ctx context.Context, amount float64, to string) (*ConversionResult, error) {
	return s.Convert(ctx, amount, s.baseCurrency, to)
}

// FormatMoney formats a money amount with currency symbol
func (s *Service) FormatMoney(ctx context.Context, money Money) (string, error) {
	currency, err := s.repo.GetCurrencyByCode(ctx, money.Currency)
	if err != nil {
		return fmt.Sprintf("%.2f %s", money.Amount, money.Currency), nil
	}

	return s.converter.FormatAmount(money.Amount, currency), nil
}

// SetExchangeRate manually sets an exchange rate
func (s *Service) SetExchangeRate(ctx context.Context, from, to string, rate float64, validFor time.Duration) error {
	if rate <= 0 {
		return fmt.Errorf("rate must be positive")
	}

	// Verify both currencies exist
	_, err := s.repo.GetCurrencyByCode(ctx, from)
	if err != nil {
		return fmt.Errorf("from currency %s not found", from)
	}

	_, err = s.repo.GetCurrencyByCode(ctx, to)
	if err != nil {
		return fmt.Errorf("to currency %s not found", to)
	}

	exchangeRate := &ExchangeRate{
		FromCurrency: from,
		ToCurrency:   to,
		Rate:         rate,
		InverseRate:  1 / rate,
		Source:       string(SourceManual),
		FetchedAt:    time.Now(),
		ValidUntil:   time.Now().Add(validFor),
	}

	err = s.repo.CreateExchangeRate(ctx, exchangeRate)
	if err != nil {
		return err
	}

	// Clear cache for this pair
	s.invalidateCache(from, to)

	return nil
}

// BulkSetExchangeRates sets multiple exchange rates from a base currency
func (s *Service) BulkSetExchangeRates(ctx context.Context, baseCurrency string, rates map[string]float64, validFor time.Duration) error {
	var exchangeRates []*ExchangeRate

	now := time.Now()
	validUntil := now.Add(validFor)

	for toCurrency, rate := range rates {
		if toCurrency == baseCurrency {
			continue
		}

		exchangeRates = append(exchangeRates, &ExchangeRate{
			FromCurrency: baseCurrency,
			ToCurrency:   toCurrency,
			Rate:         rate,
			InverseRate:  1 / rate,
			Source:       string(SourceManual),
			FetchedAt:    now,
			ValidUntil:   validUntil,
		})
	}

	err := s.repo.BulkCreateExchangeRates(ctx, exchangeRates)
	if err != nil {
		return err
	}

	// Clear all cache entries for base currency
	s.invalidateCacheForBase(baseCurrency)

	return nil
}

// GetBaseCurrency returns the configured base currency
func (s *Service) GetBaseCurrency() string {
	return s.baseCurrency
}

// GetAllRatesFromBase returns all exchange rates from the base currency
func (s *Service) GetAllRatesFromBase(ctx context.Context) ([]*ExchangeRate, error) {
	return s.repo.GetAllExchangeRatesFromBase(ctx, s.baseCurrency)
}

// CreateCurrency creates a new currency
func (s *Service) CreateCurrency(ctx context.Context, currency *Currency) error {
	return s.repo.CreateCurrency(ctx, currency)
}

// UpdateCurrency updates a currency
func (s *Service) UpdateCurrency(ctx context.Context, currency *Currency) error {
	return s.repo.UpdateCurrency(ctx, currency)
}

// ValidateConversion validates that a conversion can be performed
func (s *Service) ValidateConversion(ctx context.Context, from, to string) error {
	_, err := s.GetExchangeRate(ctx, from, to)
	return err
}

// cacheRate adds a rate to the cache
func (s *Service) cacheRate(rate *ExchangeRate) {
	cacheKey := fmt.Sprintf("%s-%s", rate.FromCurrency, rate.ToCurrency)
	s.cache.mu.Lock()
	s.cache.rates[cacheKey] = rate
	s.cache.mu.Unlock()
}

// invalidateCache removes a specific pair from cache
func (s *Service) invalidateCache(from, to string) {
	s.cache.mu.Lock()
	delete(s.cache.rates, fmt.Sprintf("%s-%s", from, to))
	delete(s.cache.rates, fmt.Sprintf("%s-%s", to, from))
	s.cache.mu.Unlock()
}

// invalidateCacheForBase removes all cache entries involving a base currency
func (s *Service) invalidateCacheForBase(base string) {
	s.cache.mu.Lock()
	for key := range s.cache.rates {
		// Key format is "FROM-TO"
		if len(key) >= 3 && (key[:3] == base || key[len(key)-3:] == base) {
			delete(s.cache.rates, key)
		}
	}
	s.cache.mu.Unlock()
}

// ToCurrencyResponse converts Currency to API response
func ToCurrencyResponse(c *Currency) *CurrencyResponse {
	if c == nil {
		return nil
	}
	return &CurrencyResponse{
		Code:          c.Code,
		Name:          c.Name,
		Symbol:        c.Symbol,
		DecimalPlaces: c.DecimalPlaces,
	}
}

// ToExchangeRateResponse converts ExchangeRate to API response
func ToExchangeRateResponse(r *ExchangeRate) *ExchangeRateResponse {
	if r == nil {
		return nil
	}
	return &ExchangeRateResponse{
		FromCurrency: r.FromCurrency,
		ToCurrency:   r.ToCurrency,
		Rate:         r.Rate,
		ValidUntil:   r.ValidUntil,
	}
}

// helper to get minimum of two times
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
