package currency

import (
	"time"

	"github.com/google/uuid"
)

// Currency represents a supported currency
type Currency struct {
	Code          string    `json:"code" db:"code"`
	Name          string    `json:"name" db:"name"`
	Symbol        string    `json:"symbol" db:"symbol"`
	DecimalPlaces int       `json:"decimal_places" db:"decimal_places"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ExchangeRate represents an exchange rate between two currencies
type ExchangeRate struct {
	ID           uuid.UUID `json:"id" db:"id"`
	FromCurrency string    `json:"from_currency" db:"from_currency"`
	ToCurrency   string    `json:"to_currency" db:"to_currency"`
	Rate         float64   `json:"rate" db:"rate"`                   // How many ToCurrency units per 1 FromCurrency
	InverseRate  float64   `json:"inverse_rate" db:"inverse_rate"`   // How many FromCurrency units per 1 ToCurrency
	Source       string    `json:"source" db:"source"`               // manual, openexchange, etc.
	FetchedAt    time.Time `json:"fetched_at" db:"fetched_at"`
	ValidUntil   time.Time `json:"valid_until" db:"valid_until"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// Money represents an amount with currency
type Money struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// ConversionResult represents the result of a currency conversion
type ConversionResult struct {
	Original       Money     `json:"original"`
	Converted      Money     `json:"converted"`
	ExchangeRate   float64   `json:"exchange_rate"`
	ExchangeRateID uuid.UUID `json:"exchange_rate_id,omitempty"`
	ConvertedAt    time.Time `json:"converted_at"`
}

// CurrencyResponse is the API response for currency
type CurrencyResponse struct {
	Code          string `json:"code"`
	Name          string `json:"name"`
	Symbol        string `json:"symbol"`
	DecimalPlaces int    `json:"decimal_places"`
}

// ExchangeRateResponse is the API response for exchange rate
type ExchangeRateResponse struct {
	FromCurrency string    `json:"from_currency"`
	ToCurrency   string    `json:"to_currency"`
	Rate         float64   `json:"rate"`
	ValidUntil   time.Time `json:"valid_until"`
}

// ConvertRequest is the API request for conversion
type ConvertRequest struct {
	Amount       float64 `json:"amount" binding:"required,gt=0"`
	FromCurrency string  `json:"from_currency" binding:"required,len=3"`
	ToCurrency   string  `json:"to_currency" binding:"required,len=3"`
}

// ConvertResponse is the API response for conversion
type ConvertResponse struct {
	OriginalAmount   float64 `json:"original_amount"`
	OriginalCurrency string  `json:"original_currency"`
	ConvertedAmount  float64 `json:"converted_amount"`
	ConvertedCurrency string `json:"converted_currency"`
	ExchangeRate     float64 `json:"exchange_rate"`
	FormattedOriginal  string `json:"formatted_original"`
	FormattedConverted string `json:"formatted_converted"`
}

// RoundingMode defines how amounts are rounded
type RoundingMode int

const (
	RoundingModeNone       RoundingMode = iota // No rounding
	RoundingModeStandard                       // Standard rounding
	RoundingModeCeiling                        // Always round up
	RoundingModeFloor                          // Always round down
	RoundingModeBankers                        // Banker's rounding (round to even)
)

// Common currency codes
const (
	CurrencyUSD = "USD"
	CurrencyEUR = "EUR"
	CurrencyGBP = "GBP"
	CurrencyTMT = "TMT"
	CurrencyUZS = "UZS"
	CurrencyKZT = "KZT"
	CurrencyRUB = "RUB"
	CurrencyTRY = "TRY"
	CurrencyAED = "AED"
	CurrencyINR = "INR"
)

// ExchangeRateSource defines the source of exchange rates
type ExchangeRateSource string

const (
	SourceManual       ExchangeRateSource = "manual"
	SourceOpenExchange ExchangeRateSource = "openexchange"
	SourceFixer        ExchangeRateSource = "fixer"
	SourceCurrencyAPI  ExchangeRateSource = "currencyapi"
)
