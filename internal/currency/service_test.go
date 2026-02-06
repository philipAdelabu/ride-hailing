package currency

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is an in-package mock for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetActiveCurrencies(ctx context.Context) ([]*Currency, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*Currency), args.Error(1)
}

func (m *MockRepository) GetCurrencyByCode(ctx context.Context, code string) (*Currency, error) {
	args := m.Called(ctx, code)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Currency), args.Error(1)
}

func (m *MockRepository) GetLatestExchangeRate(ctx context.Context, fromCurrency, toCurrency string) (*ExchangeRate, error) {
	args := m.Called(ctx, fromCurrency, toCurrency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExchangeRate), args.Error(1)
}

func (m *MockRepository) GetExchangeRateByID(ctx context.Context, id uuid.UUID) (*ExchangeRate, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ExchangeRate), args.Error(1)
}

func (m *MockRepository) CreateExchangeRate(ctx context.Context, rate *ExchangeRate) error {
	args := m.Called(ctx, rate)
	return args.Error(0)
}

func (m *MockRepository) BulkCreateExchangeRates(ctx context.Context, rates []*ExchangeRate) error {
	args := m.Called(ctx, rates)
	return args.Error(0)
}

func (m *MockRepository) GetAllExchangeRatesFromBase(ctx context.Context, baseCurrency string) ([]*ExchangeRate, error) {
	args := m.Called(ctx, baseCurrency)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*ExchangeRate), args.Error(1)
}

func (m *MockRepository) InvalidateExchangeRates(ctx context.Context, fromCurrency string) error {
	args := m.Called(ctx, fromCurrency)
	return args.Error(0)
}

func (m *MockRepository) CreateCurrency(ctx context.Context, curr *Currency) error {
	args := m.Called(ctx, curr)
	return args.Error(0)
}

func (m *MockRepository) UpdateCurrency(ctx context.Context, curr *Currency) error {
	args := m.Called(ctx, curr)
	return args.Error(0)
}

func (m *MockRepository) CleanupExpiredRates(ctx context.Context, olderThan time.Duration) (int64, error) {
	args := m.Called(ctx, olderThan)
	return args.Get(0).(int64), args.Error(1)
}

// =============================================================================
// Test NewService
// =============================================================================

func TestNewService(t *testing.T) {
	t.Run("with custom base currency", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo, CurrencyEUR)

		assert.NotNil(t, service)
		assert.Equal(t, CurrencyEUR, service.GetBaseCurrency())
	})

	t.Run("with empty base currency defaults to USD", func(t *testing.T) {
		mockRepo := new(MockRepository)
		service := NewService(mockRepo, "")

		assert.NotNil(t, service)
		assert.Equal(t, CurrencyUSD, service.GetBaseCurrency())
	})
}

// =============================================================================
// Test GetExchangeRate - 3-level fallback (direct, inverse, triangulation)
// =============================================================================

func TestGetExchangeRate_SameCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 1.0, rate.Rate)
	assert.Equal(t, 1.0, rate.InverseRate)
	assert.Equal(t, CurrencyEUR, rate.FromCurrency)
	assert.Equal(t, CurrencyEUR, rate.ToCurrency)
	assert.Equal(t, "identity", rate.Source)
}

func TestGetExchangeRate_DirectRate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	expectedRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(expectedRate, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 0.85, rate.Rate)
	assert.Equal(t, CurrencyUSD, rate.FromCurrency)
	assert.Equal(t, CurrencyEUR, rate.ToCurrency)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_InverseRate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Direct rate not found, but inverse exists
	inverseRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.18,
		InverseRate:  1.0 / 1.18,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(inverseRate, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	// The returned rate should use the inverse rate's InverseRate as its Rate
	assert.InDelta(t, 1.0/1.18, rate.Rate, 0.0001)
	assert.Equal(t, CurrencyUSD, rate.FromCurrency)
	assert.Equal(t, CurrencyEUR, rate.ToCurrency)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_Triangulation(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// EUR -> GBP (neither direct nor inverse available, triangulate via USD)
	// EUR -> USD = 1.10, USD -> GBP = 0.75
	// Expected: EUR -> GBP = 1.10 * 0.75 = 0.825

	eurToUsd := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.10,
		InverseRate:  1.0 / 1.10,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	usdToGbp := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		InverseRate:  1.0 / 0.75,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	// EUR -> GBP: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	// GBP -> EUR: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	// EUR -> USD: found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(eurToUsd, nil)
	// USD -> GBP: found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(usdToGbp, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	require.NoError(t, err)
	// Triangulated: 1.10 * 0.75 = 0.825
	assert.InDelta(t, 0.825, rate.Rate, 0.0001)
	assert.Equal(t, CurrencyEUR, rate.FromCurrency)
	assert.Equal(t, CurrencyGBP, rate.ToCurrency)
	assert.Equal(t, "triangulated", rate.Source)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_NoRateFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// No direct or inverse rate, triangulation fails
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	assert.Error(t, err)
	assert.Nil(t, rate)
	assert.Contains(t, err.Error(), "no rate path found")
}

func TestGetExchangeRate_CacheHit(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	expectedRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(expectedRate, nil).Once()

	// First call - should hit the database
	rate1, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
	require.NoError(t, err)
	assert.Equal(t, 0.85, rate1.Rate)

	// Second call - should hit the cache (no additional mock call)
	rate2, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
	require.NoError(t, err)
	assert.Equal(t, 0.85, rate2.Rate)

	// Verify only one database call was made
	mockRepo.AssertNumberOfCalls(t, "GetLatestExchangeRate", 1)
}

func TestGetExchangeRate_CacheExpiry(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// First rate with short expiry (already expired)
	expiredRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(-1 * time.Second), // Already expired
	}

	// Fresh rate
	freshRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.86,
		InverseRate:  1.0 / 0.86,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	// Manually cache the expired rate
	service.cacheRate(expiredRate)

	// When cache is expired, should fetch from DB
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(freshRate, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
	require.NoError(t, err)
	assert.Equal(t, 0.86, rate.Rate)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// Test Convert - Currency Conversion with Rounding
// =============================================================================

func TestConvert_SameCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	result, err := service.Convert(ctx, 100.50, CurrencyUSD, CurrencyUSD)

	require.NoError(t, err)
	assert.Equal(t, 100.50, result.Original.Amount)
	assert.Equal(t, 100.50, result.Converted.Amount)
	assert.Equal(t, 1.0, result.ExchangeRate)
	assert.Equal(t, CurrencyUSD, result.Original.Currency)
	assert.Equal(t, CurrencyUSD, result.Converted.Currency)
}

func TestConvert_DifferentCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	result, err := service.Convert(ctx, 100.00, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 100.00, result.Original.Amount)
	assert.Equal(t, 85.00, result.Converted.Amount) // 100 * 0.85 = 85
	assert.Equal(t, 0.85, result.ExchangeRate)
	mockRepo.AssertExpectations(t)
}

func TestConvert_PrecisionRounding(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		rate           float64
		decimalPlaces  int
		expectedAmount float64
	}{
		{
			name:           "standard 2 decimal places - round down",
			amount:         100.00,
			rate:           0.333333,
			decimalPlaces:  2,
			expectedAmount: 33.33,
		},
		{
			name:           "standard 2 decimal places - round up",
			amount:         100.00,
			rate:           0.666666,
			decimalPlaces:  2,
			expectedAmount: 66.67,
		},
		{
			name:           "zero decimal places (JPY-like)",
			amount:         100.50,
			rate:           110.5,
			decimalPlaces:  0,
			expectedAmount: 11105.0, // 100.50 * 110.5 = 11110.25, rounded to 11110
		},
		{
			name:           "3 decimal places",
			amount:         100.00,
			rate:           0.12345,
			decimalPlaces:  3,
			expectedAmount: 12.345,
		},
		{
			name:           "boundary value - exactly 0.5",
			amount:         100.00,
			rate:           0.125, // 100 * 0.125 = 12.5
			decimalPlaces:  0,
			expectedAmount: 13.0, // Standard rounding
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, CurrencyUSD)
			ctx := context.Background()

			rate := &ExchangeRate{
				ID:           uuid.New(),
				FromCurrency: CurrencyUSD,
				ToCurrency:   CurrencyEUR,
				Rate:         tt.rate,
				InverseRate:  1.0 / tt.rate,
				ValidUntil:   time.Now().Add(1 * time.Hour),
			}

			toCurrency := &Currency{
				Code:          CurrencyEUR,
				DecimalPlaces: tt.decimalPlaces,
			}

			mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
			mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

			result, err := service.Convert(ctx, tt.amount, CurrencyUSD, CurrencyEUR)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedAmount, result.Converted.Amount, 0.001)
		})
	}
}

func TestConvert_CurrencyNotFound_DefaultsTo2DecimalPlaces(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   "XYZ",
		Rate:         1.5,
		InverseRate:  1.0 / 1.5,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, "XYZ").Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, "XYZ").Return(nil, errors.New("not found"))

	result, err := service.Convert(ctx, 100.00, CurrencyUSD, "XYZ")

	require.NoError(t, err)
	assert.Equal(t, 150.00, result.Converted.Amount) // 100 * 1.5 = 150, with 2 decimal places
	mockRepo.AssertExpectations(t)
}

func TestConvert_RateNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))

	result, err := service.Convert(ctx, 100.00, CurrencyUSD, CurrencyEUR)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestConvert_ZeroAmount(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	result, err := service.Convert(ctx, 0.00, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 0.00, result.Converted.Amount)
}

func TestConvert_NegativeAmount(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	result, err := service.Convert(ctx, -100.00, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, -85.00, result.Converted.Amount) // -100 * 0.85 = -85
}

// =============================================================================
// Test ConvertToBase and ConvertFromBase
// =============================================================================

func TestConvertToBase(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.18,
		InverseRate:  1.0 / 1.18,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyUSD,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(toCurrency, nil)

	result, err := service.ConvertToBase(ctx, 100.00, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 118.00, result.Converted.Amount)
	assert.Equal(t, CurrencyUSD, result.Converted.Currency)
	mockRepo.AssertExpectations(t)
}

func TestConvertFromBase(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	result, err := service.ConvertFromBase(ctx, 100.00, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 85.00, result.Converted.Amount)
	assert.Equal(t, CurrencyEUR, result.Converted.Currency)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// Test SetExchangeRate
// =============================================================================

func TestSetExchangeRate_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	fromCurrency := &Currency{Code: CurrencyUSD}
	toCurrency := &Currency{Code: CurrencyEUR}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)
	mockRepo.On("CreateExchangeRate", ctx, mock.AnythingOfType("*currency.ExchangeRate")).Return(nil)

	err := service.SetExchangeRate(ctx, CurrencyUSD, CurrencyEUR, 0.85, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetExchangeRate_InvalidRate(t *testing.T) {
	tests := []struct {
		name string
		rate float64
	}{
		{"zero rate", 0},
		{"negative rate", -0.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, CurrencyUSD)
			ctx := context.Background()

			err := service.SetExchangeRate(ctx, CurrencyUSD, CurrencyEUR, tt.rate, 24*time.Hour)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "rate must be positive")
		})
	}
}

func TestSetExchangeRate_FromCurrencyNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetCurrencyByCode", ctx, "XXX").Return(nil, errors.New("not found"))

	err := service.SetExchangeRate(ctx, "XXX", CurrencyEUR, 0.85, 24*time.Hour)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "from currency XXX not found")
}

func TestSetExchangeRate_ToCurrencyNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	fromCurrency := &Currency{Code: CurrencyUSD}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, "XXX").Return(nil, errors.New("not found"))

	err := service.SetExchangeRate(ctx, CurrencyUSD, "XXX", 0.85, 24*time.Hour)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "to currency XXX not found")
}

func TestSetExchangeRate_CreateError(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	fromCurrency := &Currency{Code: CurrencyUSD}
	toCurrency := &Currency{Code: CurrencyEUR}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)
	mockRepo.On("CreateExchangeRate", ctx, mock.AnythingOfType("*currency.ExchangeRate")).Return(errors.New("db error"))

	err := service.SetExchangeRate(ctx, CurrencyUSD, CurrencyEUR, 0.85, 24*time.Hour)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetExchangeRate_InvalidatesCache(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Pre-populate cache
	cachedRate := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.80,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}
	service.cacheRate(cachedRate)

	fromCurrency := &Currency{Code: CurrencyUSD}
	toCurrency := &Currency{Code: CurrencyEUR}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)
	mockRepo.On("CreateExchangeRate", ctx, mock.AnythingOfType("*currency.ExchangeRate")).Return(nil)

	err := service.SetExchangeRate(ctx, CurrencyUSD, CurrencyEUR, 0.85, 24*time.Hour)
	require.NoError(t, err)

	// Verify cache was invalidated by checking the cache is empty for this pair
	service.cache.mu.RLock()
	_, exists := service.cache.rates["USD-EUR"]
	service.cache.mu.RUnlock()
	assert.False(t, exists, "Cache should be invalidated after SetExchangeRate")
}

// =============================================================================
// Test BulkSetExchangeRates
// =============================================================================

func TestBulkSetExchangeRates_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := map[string]float64{
		CurrencyEUR: 0.85,
		CurrencyGBP: 0.75,
		CurrencyTMT: 3.50,
	}

	mockRepo.On("BulkCreateExchangeRates", ctx, mock.AnythingOfType("[]*currency.ExchangeRate")).Return(nil)

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBulkSetExchangeRates_SkipsBaseCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := map[string]float64{
		CurrencyUSD: 1.00, // Should be skipped
		CurrencyEUR: 0.85,
	}

	mockRepo.On("BulkCreateExchangeRates", ctx, mock.MatchedBy(func(rates []*ExchangeRate) bool {
		// Should only have 1 rate (EUR), USD should be skipped
		return len(rates) == 1 && rates[0].ToCurrency == CurrencyEUR
	})).Return(nil)

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBulkSetExchangeRates_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := map[string]float64{
		CurrencyEUR: 0.85,
	}

	mockRepo.On("BulkCreateExchangeRates", ctx, mock.AnythingOfType("[]*currency.ExchangeRate")).Return(errors.New("db error"))

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, 24*time.Hour)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBulkSetExchangeRates_InvalidatesCacheForBase(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Pre-populate cache with USD-based rates
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.80,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})

	rates := map[string]float64{
		CurrencyEUR: 0.85,
	}

	mockRepo.On("BulkCreateExchangeRates", ctx, mock.AnythingOfType("[]*currency.ExchangeRate")).Return(nil)

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, 24*time.Hour)
	require.NoError(t, err)

	// Verify cache was invalidated
	service.cache.mu.RLock()
	_, exists := service.cache.rates["USD-EUR"]
	service.cache.mu.RUnlock()
	assert.False(t, exists, "Cache should be invalidated after BulkSetExchangeRates")
}

// =============================================================================
// Test FormatMoney
// =============================================================================

func TestFormatMoney_Success(t *testing.T) {
	tests := []struct {
		name     string
		money    Money
		currency *Currency
		expected string
	}{
		{
			name:  "USD format",
			money: Money{Amount: 100.50, Currency: CurrencyUSD},
			currency: &Currency{
				Code:          CurrencyUSD,
				Symbol:        "$",
				DecimalPlaces: 2,
			},
			expected: "$100.50",
		},
		{
			name:  "EUR format",
			money: Money{Amount: 99.99, Currency: CurrencyEUR},
			currency: &Currency{
				Code:          CurrencyEUR,
				Symbol:        "E",
				DecimalPlaces: 2,
			},
			expected: "E99.99",
		},
		{
			name:  "JPY format (zero decimals)",
			money: Money{Amount: 1000, Currency: "JPY"},
			currency: &Currency{
				Code:          "JPY",
				Symbol:        "Y",
				DecimalPlaces: 0,
			},
			expected: "Y1000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, CurrencyUSD)
			ctx := context.Background()

			mockRepo.On("GetCurrencyByCode", ctx, tt.money.Currency).Return(tt.currency, nil)

			formatted, err := service.FormatMoney(ctx, tt.money)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, formatted)
		})
	}
}

func TestFormatMoney_CurrencyNotFound_FallbackFormat(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	money := Money{Amount: 100.50, Currency: "XYZ"}

	mockRepo.On("GetCurrencyByCode", ctx, "XYZ").Return(nil, errors.New("not found"))

	formatted, err := service.FormatMoney(ctx, money)

	require.NoError(t, err)
	assert.Equal(t, "100.50 XYZ", formatted)
}

// =============================================================================
// Test Cache Management
// =============================================================================

func TestCacheRate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	rate := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	service.cacheRate(rate)

	service.cache.mu.RLock()
	cachedRate, exists := service.cache.rates["USD-EUR"]
	service.cache.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, 0.85, cachedRate.Rate)
}

func TestInvalidateCache(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	// Add rates to cache
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.18,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})

	service.invalidateCache(CurrencyUSD, CurrencyEUR)

	service.cache.mu.RLock()
	_, existsForward := service.cache.rates["USD-EUR"]
	_, existsReverse := service.cache.rates["EUR-USD"]
	service.cache.mu.RUnlock()

	assert.False(t, existsForward, "Forward cache entry should be invalidated")
	assert.False(t, existsReverse, "Reverse cache entry should be invalidated")
}

func TestInvalidateCacheForBase(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	// Add various rates to cache
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyGBP,
		ToCurrency:   CurrencyUSD,
		Rate:         1.25,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyGBP,
		Rate:         0.88,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})

	service.invalidateCacheForBase(CurrencyUSD)

	service.cache.mu.RLock()
	_, existsUsdEur := service.cache.rates["USD-EUR"]
	_, existsGbpUsd := service.cache.rates["GBP-USD"]
	_, existsEurGbp := service.cache.rates["EUR-GBP"]
	service.cache.mu.RUnlock()

	assert.False(t, existsUsdEur, "USD-EUR should be invalidated")
	assert.False(t, existsGbpUsd, "GBP-USD should be invalidated")
	assert.True(t, existsEurGbp, "EUR-GBP should NOT be invalidated")
}

// =============================================================================
// Test Other Service Methods
// =============================================================================

func TestGetActiveCurrencies(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currencies := []*Currency{
		{Code: CurrencyUSD, Name: "US Dollar", IsActive: true},
		{Code: CurrencyEUR, Name: "Euro", IsActive: true},
	}

	mockRepo.On("GetActiveCurrencies", ctx).Return(currencies, nil)

	result, err := service.GetActiveCurrencies(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestGetActiveCurrencies_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetActiveCurrencies", ctx).Return(nil, errors.New("db error"))

	result, err := service.GetActiveCurrencies(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestGetCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currency := &Currency{
		Code:   CurrencyUSD,
		Name:   "US Dollar",
		Symbol: "$",
	}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(currency, nil)

	result, err := service.GetCurrency(ctx, CurrencyUSD)

	require.NoError(t, err)
	assert.Equal(t, CurrencyUSD, result.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetCurrency_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetCurrencyByCode", ctx, "XXX").Return(nil, errors.New("not found"))

	result, err := service.GetCurrency(ctx, "XXX")

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

func TestGetBaseCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyEUR)

	assert.Equal(t, CurrencyEUR, service.GetBaseCurrency())
}

func TestGetAllRatesFromBase(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := []*ExchangeRate{
		{FromCurrency: CurrencyUSD, ToCurrency: CurrencyEUR, Rate: 0.85},
		{FromCurrency: CurrencyUSD, ToCurrency: CurrencyGBP, Rate: 0.75},
	}

	mockRepo.On("GetAllExchangeRatesFromBase", ctx, CurrencyUSD).Return(rates, nil)

	result, err := service.GetAllRatesFromBase(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	mockRepo.AssertExpectations(t)
}

func TestCreateCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currency := &Currency{
		Code:          "TMT",
		Name:          "Turkmen Manat",
		Symbol:        "m",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	mockRepo.On("CreateCurrency", ctx, currency).Return(nil)

	err := service.CreateCurrency(ctx, currency)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currency := &Currency{
		Code:          "TMT",
		Name:          "Turkmen Manat Updated",
		Symbol:        "M",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	mockRepo.On("UpdateCurrency", ctx, currency).Return(nil)

	err := service.UpdateCurrency(ctx, currency)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestValidateConversion_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)

	err := service.ValidateConversion(ctx, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestValidateConversion_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))

	err := service.ValidateConversion(ctx, CurrencyUSD, CurrencyEUR)

	assert.Error(t, err)
}

// =============================================================================
// Test Response Converters
// =============================================================================

func TestToCurrencyResponse(t *testing.T) {
	t.Run("valid currency", func(t *testing.T) {
		currency := &Currency{
			Code:          CurrencyUSD,
			Name:          "US Dollar",
			Symbol:        "$",
			DecimalPlaces: 2,
		}

		response := ToCurrencyResponse(currency)

		assert.NotNil(t, response)
		assert.Equal(t, CurrencyUSD, response.Code)
		assert.Equal(t, "US Dollar", response.Name)
		assert.Equal(t, "$", response.Symbol)
		assert.Equal(t, 2, response.DecimalPlaces)
	})

	t.Run("nil currency", func(t *testing.T) {
		response := ToCurrencyResponse(nil)
		assert.Nil(t, response)
	})
}

func TestToExchangeRateResponse(t *testing.T) {
	t.Run("valid exchange rate", func(t *testing.T) {
		validUntil := time.Now().Add(1 * time.Hour)
		rate := &ExchangeRate{
			FromCurrency: CurrencyUSD,
			ToCurrency:   CurrencyEUR,
			Rate:         0.85,
			ValidUntil:   validUntil,
		}

		response := ToExchangeRateResponse(rate)

		assert.NotNil(t, response)
		assert.Equal(t, CurrencyUSD, response.FromCurrency)
		assert.Equal(t, CurrencyEUR, response.ToCurrency)
		assert.Equal(t, 0.85, response.Rate)
		assert.Equal(t, validUntil, response.ValidUntil)
	})

	t.Run("nil exchange rate", func(t *testing.T) {
		response := ToExchangeRateResponse(nil)
		assert.Nil(t, response)
	})
}

// =============================================================================
// Test Helper Function
// =============================================================================

func TestMinTime(t *testing.T) {
	now := time.Now()
	earlier := now.Add(-1 * time.Hour)
	later := now.Add(1 * time.Hour)

	assert.Equal(t, earlier, minTime(earlier, later))
	assert.Equal(t, earlier, minTime(later, earlier))
	assert.Equal(t, now, minTime(now, now))
}

// =============================================================================
// Financial Precision Tests - Boundary Values
// =============================================================================

func TestConvert_FinancialPrecision_BoundaryValues(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		rate           float64
		decimalPlaces  int
		expectedAmount float64
		description    string
	}{
		{
			name:           "very small amount",
			amount:         0.01,
			rate:           0.85,
			decimalPlaces:  2,
			expectedAmount: 0.01, // 0.01 * 0.85 = 0.0085, rounded to 0.01
			description:    "smallest unit conversion",
		},
		{
			name:           "large amount",
			amount:         1000000.00,
			rate:           0.85,
			decimalPlaces:  2,
			expectedAmount: 850000.00,
			description:    "million dollar conversion",
		},
		{
			name:           "rate less than 1",
			amount:         100.00,
			rate:           0.001,
			decimalPlaces:  2,
			expectedAmount: 0.10, // 100 * 0.001 = 0.1
			description:    "conversion with very small rate",
		},
		{
			name:           "rate greater than 1000",
			amount:         100.00,
			rate:           3500.0,
			decimalPlaces:  0,
			expectedAmount: 350000.0, // 100 * 3500
			description:    "conversion to currency with high rate (like TMT)",
		},
		{
			name:           "rounding boundary - exactly half",
			amount:         1.00,
			rate:           0.005, // 1 * 0.005 = 0.005
			decimalPlaces:  2,
			expectedAmount: 0.01, // Standard rounding: 0.005 -> 0.01
			description:    "edge case for banker's vs standard rounding",
		},
		{
			name:           "rounding boundary - just below half",
			amount:         1.00,
			rate:           0.004, // 1 * 0.004 = 0.004
			decimalPlaces:  2,
			expectedAmount: 0.00, // Rounds down
			description:    "edge case just below 0.5",
		},
		{
			name:           "rounding boundary - just above half",
			amount:         1.00,
			rate:           0.006, // 1 * 0.006 = 0.006
			decimalPlaces:  2,
			expectedAmount: 0.01, // Rounds up
			description:    "edge case just above 0.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, CurrencyUSD)
			ctx := context.Background()

			rate := &ExchangeRate{
				ID:           uuid.New(),
				FromCurrency: CurrencyUSD,
				ToCurrency:   CurrencyEUR,
				Rate:         tt.rate,
				InverseRate:  1.0 / tt.rate,
				ValidUntil:   time.Now().Add(1 * time.Hour),
			}

			toCurrency := &Currency{
				Code:          CurrencyEUR,
				DecimalPlaces: tt.decimalPlaces,
			}

			mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
			mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

			result, err := service.Convert(ctx, tt.amount, CurrencyUSD, CurrencyEUR)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedAmount, result.Converted.Amount, 0.001,
				"Failed: %s", tt.description)
		})
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentCacheAccess(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)

	// Run concurrent cache operations
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

// =============================================================================
// Rate Staleness Detection Tests
// =============================================================================

func TestGetExchangeRate_StaleRate_TriggersRefresh(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Rate that expired 1 hour ago - considered stale
	staleRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.80,
		InverseRate:  1.0 / 0.80,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(-1 * time.Hour), // Already expired
	}

	// Fresh rate from DB
	freshRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	// Cache the stale rate
	service.cacheRate(staleRate)

	// Should fetch fresh rate from DB since cached is expired
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(freshRate, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 0.85, rate.Rate, "Should return fresh rate, not stale cached rate")
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_ValidCachedRate_NoDBCall(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Valid cached rate (not expired)
	validRate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(30 * time.Minute),
	}

	// Cache the valid rate
	service.cacheRate(validRate)

	// Should NOT call DB since cache is valid
	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 0.85, rate.Rate)
	// Verify NO database call was made
	mockRepo.AssertNotCalled(t, "GetLatestExchangeRate")
}

// =============================================================================
// Advanced Triangulation Tests
// =============================================================================

func TestGetExchangeRate_TriangulationWithInverseRates(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Scenario: EUR -> GBP needs triangulation
	// EUR -> USD (direct not found, inverse found: USD -> EUR = 0.85)
	// USD -> GBP (direct found)

	usdToEur := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85, // ~1.176
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	usdToGbp := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		InverseRate:  1.0 / 0.75,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	// EUR -> GBP: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	// GBP -> EUR: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	// EUR -> USD: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))
	// USD -> EUR: found (for inverse)
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(usdToEur, nil)
	// USD -> GBP: found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(usdToGbp, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	require.NoError(t, err)
	// EUR -> USD = 1/0.85 = 1.176, USD -> GBP = 0.75
	// EUR -> GBP = 1.176 * 0.75 = 0.882
	expectedRate := (1.0 / 0.85) * 0.75
	assert.InDelta(t, expectedRate, rate.Rate, 0.001)
	assert.Equal(t, "triangulated", rate.Source)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_TriangulationBothInverse(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Both legs need inverse rates
	// EUR -> USD: not found direct, but USD -> EUR exists
	// USD -> GBP: not found direct, but GBP -> USD exists

	usdToEur := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	gbpToUsd := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyGBP,
		ToCurrency:   CurrencyUSD,
		Rate:         1.35,
		InverseRate:  1.0 / 1.35,
		Source:       string(SourceManual),
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	// EUR -> GBP: not found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	// EUR -> USD: not found direct
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))
	// USD -> EUR: found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(usdToEur, nil)
	// USD -> GBP: not found direct
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(nil, errors.New("not found"))
	// GBP -> USD: found
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyUSD).Return(gbpToUsd, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	require.NoError(t, err)
	// EUR -> USD = 1/0.85, USD -> GBP = 1/1.35
	expectedRate := (1.0 / 0.85) * (1.0 / 1.35)
	assert.InDelta(t, expectedRate, rate.Rate, 0.001)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_TriangulationValidUntilUsesMin(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	earlierExpiry := time.Now().Add(30 * time.Minute)
	laterExpiry := time.Now().Add(2 * time.Hour)

	eurToUsd := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.10,
		InverseRate:  1.0 / 1.10,
		Source:       string(SourceManual),
		ValidUntil:   earlierExpiry,
	}

	usdToGbp := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		InverseRate:  1.0 / 0.75,
		Source:       string(SourceManual),
		ValidUntil:   laterExpiry,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(eurToUsd, nil)
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(usdToGbp, nil)

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	require.NoError(t, err)
	// Triangulated rate should use the earlier expiry (min of the two)
	assert.Equal(t, earlierExpiry, rate.ValidUntil)
	mockRepo.AssertExpectations(t)
}

func TestGetExchangeRate_DirectToBase_NoTriangulation(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// EUR -> USD should NOT try triangulation (USD is base)
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))

	rate, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyUSD)

	// Should fail without trying triangulation since one currency is the base
	assert.Error(t, err)
	assert.Nil(t, rate)
	assert.Contains(t, err.Error(), "no exchange rate found")
}

func TestGetExchangeRate_BaseToTarget_NoTriangulation(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// USD -> EUR should NOT try triangulation (USD is base)
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(nil, errors.New("not found"))

	rate, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)

	// Should fail without trying triangulation since one currency is the base
	assert.Error(t, err)
	assert.Nil(t, rate)
	assert.Contains(t, err.Error(), "no exchange rate found")
}

// =============================================================================
// Financial Precision - Edge Cases for Money Handling
// =============================================================================

func TestConvert_VeryLargeAmount(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	// Test with billion dollar amount
	result, err := service.Convert(ctx, 1000000000.00, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	assert.Equal(t, 850000000.00, result.Converted.Amount)
}

func TestConvert_VerySmallAmount(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	// Smallest practical amount
	result, err := service.Convert(ctx, 0.01, CurrencyUSD, CurrencyEUR)

	require.NoError(t, err)
	// 0.01 * 0.85 = 0.0085, rounded to 0.01
	assert.Equal(t, 0.01, result.Converted.Amount)
}

func TestConvert_PrecisionWithDifferentDecimalPlaces(t *testing.T) {
	tests := []struct {
		name           string
		targetCurrency string
		decimalPlaces  int
		amount         float64
		rate           float64
		expectedAmount float64
	}{
		{
			name:           "JPY (0 decimals) - round up",
			targetCurrency: "JPY",
			decimalPlaces:  0,
			amount:         100.00,
			rate:           145.67,
			expectedAmount: 14567.0, // 100 * 145.67 = 14567
		},
		{
			name:           "JPY (0 decimals) - round down",
			targetCurrency: "JPY",
			decimalPlaces:  0,
			amount:         100.00,
			rate:           145.23,
			expectedAmount: 14523.0,
		},
		{
			name:           "BHD (3 decimals)",
			targetCurrency: "BHD",
			decimalPlaces:  3,
			amount:         100.00,
			rate:           0.37678,
			expectedAmount: 37.678,
		},
		{
			name:           "EUR (2 decimals) - standard",
			targetCurrency: CurrencyEUR,
			decimalPlaces:  2,
			amount:         100.00,
			rate:           0.85123,
			expectedAmount: 85.12, // Rounded to 2 decimal places
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, CurrencyUSD)
			ctx := context.Background()

			rate := &ExchangeRate{
				ID:           uuid.New(),
				FromCurrency: CurrencyUSD,
				ToCurrency:   tt.targetCurrency,
				Rate:         tt.rate,
				InverseRate:  1.0 / tt.rate,
				ValidUntil:   time.Now().Add(1 * time.Hour),
			}

			toCurrency := &Currency{
				Code:          tt.targetCurrency,
				DecimalPlaces: tt.decimalPlaces,
			}

			mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, tt.targetCurrency).Return(rate, nil)
			mockRepo.On("GetCurrencyByCode", ctx, tt.targetCurrency).Return(toCurrency, nil)

			result, err := service.Convert(ctx, tt.amount, CurrencyUSD, tt.targetCurrency)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedAmount, result.Converted.Amount, 0.0001)
		})
	}
}

func TestConvert_RoundingConsistency(t *testing.T) {
	// Test that repeated conversions produce consistent results
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rate := &ExchangeRate{
		ID:           uuid.New(),
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85333333,
		InverseRate:  1.0 / 0.85333333,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	// Run same conversion multiple times
	var results []float64
	for i := 0; i < 5; i++ {
		result, err := service.Convert(ctx, 123.45, CurrencyUSD, CurrencyEUR)
		require.NoError(t, err)
		results = append(results, result.Converted.Amount)
	}

	// All results should be identical
	for i := 1; i < len(results); i++ {
		assert.Equal(t, results[0], results[i], "Conversion should be deterministic")
	}
}

// =============================================================================
// Cache Edge Cases
// =============================================================================

func TestCache_MultipleCurrencyPairs(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	usdEur := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	usdGbp := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	eurGbp := &ExchangeRate{
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyGBP,
		Rate:         0.88,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(usdEur, nil).Once()
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(usdGbp, nil).Once()
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(eurGbp, nil).Once()

	// Fetch different pairs
	rate1, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
	require.NoError(t, err)

	rate2, err := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyGBP)
	require.NoError(t, err)

	rate3, err := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)
	require.NoError(t, err)

	// Verify rates
	assert.Equal(t, 0.85, rate1.Rate)
	assert.Equal(t, 0.75, rate2.Rate)
	assert.Equal(t, 0.88, rate3.Rate)

	// Fetch again - should use cache
	rate1Again, _ := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyEUR)
	rate2Again, _ := service.GetExchangeRate(ctx, CurrencyUSD, CurrencyGBP)
	rate3Again, _ := service.GetExchangeRate(ctx, CurrencyEUR, CurrencyGBP)

	assert.Equal(t, rate1.Rate, rate1Again.Rate)
	assert.Equal(t, rate2.Rate, rate2Again.Rate)
	assert.Equal(t, rate3.Rate, rate3Again.Rate)

	// Verify only 3 DB calls total
	mockRepo.AssertNumberOfCalls(t, "GetLatestExchangeRate", 3)
}

func TestCache_InvalidateOnlyAffectedPairs(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	// Cache multiple pairs
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyGBP,
		Rate:         0.88,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})

	// Invalidate only USD-EUR pair
	service.invalidateCache(CurrencyUSD, CurrencyEUR)

	service.cache.mu.RLock()
	_, existsUsdEur := service.cache.rates["USD-EUR"]
	_, existsEurUsd := service.cache.rates["EUR-USD"]
	_, existsUsdGbp := service.cache.rates["USD-GBP"]
	_, existsEurGbp := service.cache.rates["EUR-GBP"]
	service.cache.mu.RUnlock()

	assert.False(t, existsUsdEur, "USD-EUR should be invalidated")
	assert.False(t, existsEurUsd, "EUR-USD should be invalidated (reverse)")
	assert.True(t, existsUsdGbp, "USD-GBP should remain in cache")
	assert.True(t, existsEurGbp, "EUR-GBP should remain in cache")
}

// =============================================================================
// ConversionResult Validation Tests
// =============================================================================

func TestConvert_ResultContainsCorrectMetadata(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rateID := uuid.New()
	rate := &ExchangeRate{
		ID:           rateID,
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		InverseRate:  1.0 / 0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	toCurrency := &Currency{
		Code:          CurrencyEUR,
		DecimalPlaces: 2,
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyEUR).Return(rate, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)

	beforeConversion := time.Now()
	result, err := service.Convert(ctx, 100.00, CurrencyUSD, CurrencyEUR)
	afterConversion := time.Now()

	require.NoError(t, err)

	// Verify original money
	assert.Equal(t, 100.00, result.Original.Amount)
	assert.Equal(t, CurrencyUSD, result.Original.Currency)

	// Verify converted money
	assert.Equal(t, 85.00, result.Converted.Amount)
	assert.Equal(t, CurrencyEUR, result.Converted.Currency)

	// Verify exchange rate info
	assert.Equal(t, 0.85, result.ExchangeRate)
	assert.Equal(t, rateID, result.ExchangeRateID)

	// Verify timestamp
	assert.True(t, result.ConvertedAt.After(beforeConversion) || result.ConvertedAt.Equal(beforeConversion))
	assert.True(t, result.ConvertedAt.Before(afterConversion) || result.ConvertedAt.Equal(afterConversion))
}

// =============================================================================
// BulkSetExchangeRates - Comprehensive Tests
// =============================================================================

func TestBulkSetExchangeRates_VerifiesRateValues(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := map[string]float64{
		CurrencyEUR: 0.85,
		CurrencyGBP: 0.75,
		CurrencyTMT: 3.50,
	}

	// Capture the rates passed to BulkCreateExchangeRates
	mockRepo.On("BulkCreateExchangeRates", ctx, mock.MatchedBy(func(exchangeRates []*ExchangeRate) bool {
		if len(exchangeRates) != 3 {
			return false
		}

		for _, r := range exchangeRates {
			if r.FromCurrency != CurrencyUSD {
				return false
			}
			if r.Source != string(SourceManual) {
				return false
			}
			if r.InverseRate != 1/r.Rate {
				return false
			}
		}
		return true
	})).Return(nil)

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestBulkSetExchangeRates_ValidUntilSet(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	rates := map[string]float64{
		CurrencyEUR: 0.85,
	}

	validFor := 12 * time.Hour
	beforeCall := time.Now()

	mockRepo.On("BulkCreateExchangeRates", ctx, mock.MatchedBy(func(exchangeRates []*ExchangeRate) bool {
		if len(exchangeRates) != 1 {
			return false
		}
		r := exchangeRates[0]
		expectedValidUntil := beforeCall.Add(validFor)
		// Allow 1 second tolerance
		return r.ValidUntil.After(expectedValidUntil.Add(-1*time.Second)) &&
			r.ValidUntil.Before(expectedValidUntil.Add(1*time.Second))
	})).Return(nil)

	err := service.BulkSetExchangeRates(ctx, CurrencyUSD, rates, validFor)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// SetExchangeRate - Edge Cases
// =============================================================================

func TestSetExchangeRate_VerySmallRate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	fromCurrency := &Currency{Code: CurrencyUSD}
	toCurrency := &Currency{Code: CurrencyEUR}

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, CurrencyEUR).Return(toCurrency, nil)
	mockRepo.On("CreateExchangeRate", ctx, mock.MatchedBy(func(r *ExchangeRate) bool {
		return r.Rate == 0.000001 && r.InverseRate == 1.0/0.000001
	})).Return(nil)

	err := service.SetExchangeRate(ctx, CurrencyUSD, CurrencyEUR, 0.000001, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestSetExchangeRate_VeryLargeRate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	fromCurrency := &Currency{Code: CurrencyUSD}
	toCurrency := &Currency{Code: "UZS"} // Uzbek Som has high rate vs USD

	mockRepo.On("GetCurrencyByCode", ctx, CurrencyUSD).Return(fromCurrency, nil)
	mockRepo.On("GetCurrencyByCode", ctx, "UZS").Return(toCurrency, nil)
	mockRepo.On("CreateExchangeRate", ctx, mock.MatchedBy(func(r *ExchangeRate) bool {
		return r.Rate == 12500.00 && r.InverseRate == 1.0/12500.00
	})).Return(nil)

	err := service.SetExchangeRate(ctx, CurrencyUSD, "UZS", 12500.00, 24*time.Hour)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// GetAllRatesFromBase Tests
// =============================================================================

func TestGetAllRatesFromBase_EmptyResult(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetAllExchangeRatesFromBase", ctx, CurrencyUSD).Return([]*ExchangeRate{}, nil)

	result, err := service.GetAllRatesFromBase(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetAllRatesFromBase_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	mockRepo.On("GetAllExchangeRatesFromBase", ctx, CurrencyUSD).Return(nil, errors.New("db error"))

	result, err := service.GetAllRatesFromBase(ctx)

	assert.Error(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// CreateCurrency and UpdateCurrency Error Cases
// =============================================================================

func TestCreateCurrency_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currency := &Currency{
		Code:          "NEW",
		Name:          "New Currency",
		Symbol:        "N",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	mockRepo.On("CreateCurrency", ctx, currency).Return(errors.New("db error"))

	err := service.CreateCurrency(ctx, currency)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateCurrency_Error(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	currency := &Currency{
		Code:          CurrencyUSD,
		Name:          "Updated Name",
		Symbol:        "$",
		DecimalPlaces: 2,
		IsActive:      true,
	}

	mockRepo.On("UpdateCurrency", ctx, currency).Return(errors.New("db error"))

	err := service.UpdateCurrency(ctx, currency)

	assert.Error(t, err)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// ValidateConversion - Additional Tests
// =============================================================================

func TestValidateConversion_SameCurrency(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	// Same currency should always be valid (no DB call needed)
	err := service.ValidateConversion(ctx, CurrencyUSD, CurrencyUSD)

	require.NoError(t, err)
	mockRepo.AssertNotCalled(t, "GetLatestExchangeRate")
}

func TestValidateConversion_ViaTriangulation(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)
	ctx := context.Background()

	eurToUsd := &ExchangeRate{
		FromCurrency: CurrencyEUR,
		ToCurrency:   CurrencyUSD,
		Rate:         1.10,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	usdToGbp := &ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyGBP,
		Rate:         0.75,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	}

	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyGBP).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyGBP, CurrencyEUR).Return(nil, errors.New("not found"))
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyEUR, CurrencyUSD).Return(eurToUsd, nil)
	mockRepo.On("GetLatestExchangeRate", ctx, CurrencyUSD, CurrencyGBP).Return(usdToGbp, nil)

	err := service.ValidateConversion(ctx, CurrencyEUR, CurrencyGBP)

	require.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// =============================================================================
// Concurrent Write Operations
// =============================================================================

func TestConcurrentCacheWrite(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	// Run concurrent cache writes
	done := make(chan bool, 50)
	for i := 0; i < 50; i++ {
		go func(idx int) {
			rate := &ExchangeRate{
				FromCurrency: CurrencyUSD,
				ToCurrency:   CurrencyEUR,
				Rate:         0.85 + float64(idx)/100,
				ValidUntil:   time.Now().Add(1 * time.Hour),
			}
			service.cacheRate(rate)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 50; i++ {
		<-done
	}

	// Verify cache has a rate (any rate is fine)
	service.cache.mu.RLock()
	_, exists := service.cache.rates["USD-EUR"]
	service.cache.mu.RUnlock()
	assert.True(t, exists)
}

func TestConcurrentCacheInvalidate(t *testing.T) {
	mockRepo := new(MockRepository)
	service := NewService(mockRepo, CurrencyUSD)

	// Pre-populate cache
	service.cacheRate(&ExchangeRate{
		FromCurrency: CurrencyUSD,
		ToCurrency:   CurrencyEUR,
		Rate:         0.85,
		ValidUntil:   time.Now().Add(1 * time.Hour),
	})

	// Run concurrent invalidations and writes
	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func() {
			service.invalidateCache(CurrencyUSD, CurrencyEUR)
			done <- true
		}()
		go func(idx int) {
			rate := &ExchangeRate{
				FromCurrency: CurrencyUSD,
				ToCurrency:   CurrencyEUR,
				Rate:         0.85 + float64(idx)/100,
				ValidUntil:   time.Now().Add(1 * time.Hour),
			}
			service.cacheRate(rate)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// No assertion needed - test passes if no race condition/panic occurs
}

// =============================================================================
// Table-Driven Tests for GetExchangeRate Fallback Logic
// =============================================================================

func TestGetExchangeRate_FallbackScenarios(t *testing.T) {
	tests := []struct {
		name           string
		from           string
		to             string
		directRate     *ExchangeRate
		directErr      error
		inverseRate    *ExchangeRate
		inverseErr     error
		baseCurrency   string
		expectError    bool
		expectedRate   float64
		expectedSource string
	}{
		{
			name: "direct rate available",
			from: CurrencyUSD,
			to:   CurrencyEUR,
			directRate: &ExchangeRate{
				FromCurrency: CurrencyUSD,
				ToCurrency:   CurrencyEUR,
				Rate:         0.85,
				InverseRate:  1.0 / 0.85,
				ValidUntil:   time.Now().Add(1 * time.Hour),
				Source:       string(SourceManual),
			},
			directErr:      nil,
			baseCurrency:   CurrencyUSD,
			expectError:    false,
			expectedRate:   0.85,
			expectedSource: string(SourceManual),
		},
		{
			name:       "only inverse rate available",
			from:       CurrencyUSD,
			to:         CurrencyEUR,
			directRate: nil,
			directErr:  errors.New("not found"),
			inverseRate: &ExchangeRate{
				FromCurrency: CurrencyEUR,
				ToCurrency:   CurrencyUSD,
				Rate:         1.18,
				InverseRate:  1.0 / 1.18,
				ValidUntil:   time.Now().Add(1 * time.Hour),
				Source:       string(SourceManual),
			},
			inverseErr:     nil,
			baseCurrency:   CurrencyUSD,
			expectError:    false,
			expectedRate:   1.0 / 1.18,
			expectedSource: string(SourceManual),
		},
		{
			name:         "same currency (identity)",
			from:         CurrencyEUR,
			to:           CurrencyEUR,
			baseCurrency: CurrencyUSD,
			expectError:  false,
			expectedRate: 1.0,
		},
		{
			name:         "no rate path found - direct to base",
			from:         CurrencyUSD,
			to:           CurrencyEUR,
			directRate:   nil,
			directErr:    errors.New("not found"),
			inverseRate:  nil,
			inverseErr:   errors.New("not found"),
			baseCurrency: CurrencyUSD,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			service := NewService(mockRepo, tt.baseCurrency)
			ctx := context.Background()

			if tt.from != tt.to {
				if tt.directRate != nil || tt.directErr != nil {
					mockRepo.On("GetLatestExchangeRate", ctx, tt.from, tt.to).Return(tt.directRate, tt.directErr)
				}
				if tt.directErr != nil && (tt.inverseRate != nil || tt.inverseErr != nil) {
					mockRepo.On("GetLatestExchangeRate", ctx, tt.to, tt.from).Return(tt.inverseRate, tt.inverseErr)
				}
			}

			rate, err := service.GetExchangeRate(ctx, tt.from, tt.to)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, rate)
			} else {
				require.NoError(t, err)
				assert.InDelta(t, tt.expectedRate, rate.Rate, 0.0001)
				if tt.expectedSource != "" {
					assert.Equal(t, tt.expectedSource, rate.Source)
				}
			}
		})
	}
}
