package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTranslate_English(t *testing.T) {
	result := Translate("notification.ride.accepted.title", "en")
	assert.Equal(t, "Driver Found!", result)
}

func TestTranslate_Russian(t *testing.T) {
	result := Translate("notification.ride.accepted.title", "ru")
	assert.Equal(t, "Водитель найден!", result)
}

func TestTranslate_Turkish(t *testing.T) {
	result := Translate("notification.ride.started.title", "tr")
	assert.Equal(t, "Yolculuk Başladı", result)
}

func TestTranslate_Turkmen(t *testing.T) {
	result := Translate("notification.ride.started.title", "tk")
	assert.Equal(t, "Ýol Başlandy", result)
}

func TestTranslate_FallsBackToEnglish_UnknownLang(t *testing.T) {
	result := Translate("notification.ride.accepted.title", "zh")
	assert.Equal(t, "Driver Found!", result)
}

func TestTranslate_EmptyLang_UsesEnglish(t *testing.T) {
	result := Translate("notification.ride.accepted.title", "")
	assert.Equal(t, "Driver Found!", result)
}

func TestTranslate_UnknownKey_ReturnsKey(t *testing.T) {
	result := Translate("does.not.exist", "en")
	assert.Equal(t, "does.not.exist", result)
}

func TestTranslate_WithArgs(t *testing.T) {
	result := Translate("notification.ride.accepted.body", "en", "Ali", 5)
	assert.Equal(t, "Ali will pick you up in 5 minutes", result)
}

func TestTranslate_WithArgs_Russian(t *testing.T) {
	result := Translate("notification.ride.completed.body.rider", "ru", "150.00 TMT")
	assert.Equal(t, "Ваша поездка завершена. Итоговая стоимость: 150.00 TMT", result)
}

func TestFormatAmount_USD(t *testing.T) {
	assert.Equal(t, "$15.50", FormatAmount(15.5, "USD"))
}

func TestFormatAmount_EUR(t *testing.T) {
	assert.Equal(t, "€9.99", FormatAmount(9.99, "EUR"))
}

func TestFormatAmount_TMT(t *testing.T) {
	assert.Equal(t, "150.00 TMT", FormatAmount(150.0, "TMT"))
}

func TestFormatAmount_AED(t *testing.T) {
	assert.Equal(t, "25.00 د.إ", FormatAmount(25.0, "AED"))
}

func TestFormatAmount_UnknownCurrency(t *testing.T) {
	assert.Equal(t, "10.00 XYZ", FormatAmount(10.0, "XYZ"))
}
