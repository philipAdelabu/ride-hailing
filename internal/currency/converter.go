package currency

import (
	"fmt"
	"math"
)

// Converter handles currency conversion calculations
type Converter struct {
	baseCurrency string
}

// NewConverter creates a new currency converter
func NewConverter(baseCurrency string) *Converter {
	if baseCurrency == "" {
		baseCurrency = CurrencyUSD
	}
	return &Converter{baseCurrency: baseCurrency}
}

// Convert converts an amount from one currency to another using the given rate
func (c *Converter) Convert(amount float64, rate *ExchangeRate, roundingMode RoundingMode, decimalPlaces int) float64 {
	converted := amount * rate.Rate
	return c.Round(converted, roundingMode, decimalPlaces)
}

// ConvertInverse converts an amount using the inverse rate
func (c *Converter) ConvertInverse(amount float64, rate *ExchangeRate, roundingMode RoundingMode, decimalPlaces int) float64 {
	converted := amount * rate.InverseRate
	return c.Round(converted, roundingMode, decimalPlaces)
}

// Round rounds an amount according to the specified mode and decimal places
func (c *Converter) Round(amount float64, mode RoundingMode, decimalPlaces int) float64 {
	if decimalPlaces < 0 {
		decimalPlaces = 0
	}

	multiplier := math.Pow(10, float64(decimalPlaces))

	switch mode {
	case RoundingModeNone:
		return amount
	case RoundingModeCeiling:
		return math.Ceil(amount*multiplier) / multiplier
	case RoundingModeFloor:
		return math.Floor(amount*multiplier) / multiplier
	case RoundingModeBankers:
		return c.bankersRound(amount, decimalPlaces)
	default: // RoundingModeStandard
		return math.Round(amount*multiplier) / multiplier
	}
}

// bankersRound implements banker's rounding (round half to even)
func (c *Converter) bankersRound(amount float64, decimalPlaces int) float64 {
	multiplier := math.Pow(10, float64(decimalPlaces))
	shifted := amount * multiplier
	truncated := math.Trunc(shifted)
	fraction := shifted - truncated

	if fraction > 0.5 {
		return (truncated + 1) / multiplier
	} else if fraction < 0.5 {
		return truncated / multiplier
	}

	// Exactly 0.5 - round to even
	if int64(truncated)%2 == 0 {
		return truncated / multiplier
	}
	return (truncated + 1) / multiplier
}

// FormatAmount formats an amount with the currency symbol and appropriate decimal places
func (c *Converter) FormatAmount(amount float64, currency *Currency) string {
	roundedAmount := c.Round(amount, RoundingModeStandard, currency.DecimalPlaces)

	format := fmt.Sprintf("%%.%df", currency.DecimalPlaces)
	amountStr := fmt.Sprintf(format, roundedAmount)

	return fmt.Sprintf("%s%s", currency.Symbol, amountStr)
}

// ParseAmount parses a formatted amount string (removes currency symbol)
func (c *Converter) ParseAmount(formatted string, currency *Currency) (float64, error) {
	// Remove currency symbol
	clean := formatted
	if len(currency.Symbol) > 0 && len(formatted) > len(currency.Symbol) {
		if formatted[:len(currency.Symbol)] == currency.Symbol {
			clean = formatted[len(currency.Symbol):]
		}
	}

	var amount float64
	_, err := fmt.Sscanf(clean, "%f", &amount)
	if err != nil {
		return 0, fmt.Errorf("failed to parse amount: %w", err)
	}

	return amount, nil
}

// CalculateRate calculates the exchange rate between two currencies via a base currency
func (c *Converter) CalculateRate(fromRate, toRate *ExchangeRate) float64 {
	// fromRate: base -> from currency (so inverse gives us from -> base)
	// toRate: base -> to currency
	// Combined: from -> base -> to = fromRate.InverseRate * toRate.Rate
	return fromRate.InverseRate * toRate.Rate
}

// IsZeroCurreny returns true if the currency uses zero decimal places
func (c *Converter) IsZeroCurrency(currencyCode string) bool {
	zeroCurrencies := map[string]bool{
		"JPY": true,
		"KRW": true,
		"VND": true,
		"UZS": true,
		"KZT": true,
		"PKR": true,
	}
	return zeroCurrencies[currencyCode]
}

// ToSmallestUnit converts an amount to the smallest currency unit (e.g., cents)
func (c *Converter) ToSmallestUnit(amount float64, decimalPlaces int) int64 {
	multiplier := math.Pow(10, float64(decimalPlaces))
	return int64(math.Round(amount * multiplier))
}

// FromSmallestUnit converts from the smallest currency unit to the standard amount
func (c *Converter) FromSmallestUnit(units int64, decimalPlaces int) float64 {
	divisor := math.Pow(10, float64(decimalPlaces))
	return float64(units) / divisor
}

// ValidateAmount checks if an amount is valid for a given currency
func (c *Converter) ValidateAmount(amount float64, currency *Currency) error {
	if amount < 0 {
		return fmt.Errorf("amount cannot be negative")
	}

	// Check that the amount doesn't have more decimal places than allowed
	multiplier := math.Pow(10, float64(currency.DecimalPlaces))
	shifted := amount * multiplier
	rounded := math.Round(shifted)

	if math.Abs(shifted-rounded) > 0.0001 {
		return fmt.Errorf("amount has too many decimal places for %s (max %d)",
			currency.Code, currency.DecimalPlaces)
	}

	return nil
}

// NormalizeAmount normalizes an amount to the correct decimal places for a currency
func (c *Converter) NormalizeAmount(amount float64, currency *Currency) float64 {
	return c.Round(amount, RoundingModeStandard, currency.DecimalPlaces)
}
