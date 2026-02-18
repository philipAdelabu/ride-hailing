package i18n

import "fmt"

// currencySymbols maps ISO 4217 currency codes to their display symbol.
// Currencies where the symbol precedes the amount use format "$%.2f".
// Others use "%.2f CURRENCY" format.
var currencySymbols = map[string]struct {
	symbol string
	prefix bool // true = "$12.50", false = "12.50 TMT"
}{
	"USD": {"$", true},
	"EUR": {"€", true},
	"GBP": {"£", true},
	"JPY": {"¥", true},
	"CNY": {"¥", true},
	"KRW": {"₩", true},
	"INR": {"₹", true},
	"TRY": {"₺", true},
	"RUB": {"₽", true},
	"BRL": {"R$", true},
	"MXN": {"$", true},
	"AED": {"د.إ", false},
	"KZT": {"₸", false},
	"UZS": {"сум", false},
	"TMT": {"TMT", false},
	"NGN": {"₦", true},
	"KES": {"KSh", true},
	"ZAR": {"R", true},
	"EGP": {"E£", true},
	"PKR": {"₨", true},
}

// FormatAmount returns a human-readable amount string with the currency symbol.
// Examples:
//
//	FormatAmount(15.5, "USD")  → "$15.50"
//	FormatAmount(15.5, "EUR")  → "€15.50"
//	FormatAmount(150.0, "TMT") → "150.00 TMT"
//	FormatAmount(150.0, "XYZ") → "150.00 XYZ"
func FormatAmount(amount float64, currencyCode string) string {
	info, ok := currencySymbols[currencyCode]
	if !ok {
		// Unknown currency — fall back to "amount CODE"
		return fmt.Sprintf("%.2f %s", amount, currencyCode)
	}
	if info.prefix {
		return fmt.Sprintf("%s%.2f", info.symbol, amount)
	}
	return fmt.Sprintf("%.2f %s", amount, info.symbol)
}
