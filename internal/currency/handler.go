package currency

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/richxcame/ride-hailing/pkg/common"
)

// Handler handles HTTP requests for currency
type Handler struct {
	service *Service
}

// NewHandler creates a new currency handler
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// GetCurrencies returns all active currencies
func (h *Handler) GetCurrencies(c *gin.Context) {
	currencies, err := h.service.GetActiveCurrencies(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get currencies")
		return
	}

	responses := make([]*CurrencyResponse, len(currencies))
	for i, currency := range currencies {
		responses[i] = ToCurrencyResponse(currency)
	}

	common.SuccessResponse(c, responses)
}

// GetCurrency returns a currency by code
func (h *Handler) GetCurrency(c *gin.Context) {
	code := c.Param("code")
	if len(code) != 3 {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid currency code")
		return
	}

	currency, err := h.service.GetCurrency(c.Request.Context(), code)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "currency not found")
		return
	}

	common.SuccessResponse(c, ToCurrencyResponse(currency))
}

// GetExchangeRate returns the exchange rate between two currencies
func (h *Handler) GetExchangeRate(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")

	if len(from) != 3 || len(to) != 3 {
		common.ErrorResponse(c, http.StatusBadRequest, "invalid currency codes")
		return
	}

	rate, err := h.service.GetExchangeRate(c.Request.Context(), from, to)
	if err != nil {
		common.ErrorResponse(c, http.StatusNotFound, "exchange rate not found")
		return
	}

	common.SuccessResponse(c, ToExchangeRateResponse(rate))
}

// GetAllRates returns all exchange rates from base currency
func (h *Handler) GetAllRates(c *gin.Context) {
	rates, err := h.service.GetAllRatesFromBase(c.Request.Context())
	if err != nil {
		common.ErrorResponse(c, http.StatusInternalServerError, "failed to get exchange rates")
		return
	}

	responses := make([]*ExchangeRateResponse, len(rates))
	for i, rate := range rates {
		responses[i] = ToExchangeRateResponse(rate)
	}

	common.SuccessResponse(c, gin.H{
		"base_currency": h.service.GetBaseCurrency(),
		"rates":         responses,
	})
}

// Convert converts an amount between currencies
func (h *Handler) Convert(c *gin.Context) {
	var req ConvertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.Convert(c.Request.Context(), req.Amount, req.FromCurrency, req.ToCurrency)
	if err != nil {
		common.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Format the amounts
	formattedOriginal, _ := h.service.FormatMoney(c.Request.Context(), result.Original)
	formattedConverted, _ := h.service.FormatMoney(c.Request.Context(), result.Converted)

	common.SuccessResponse(c, ConvertResponse{
		OriginalAmount:     result.Original.Amount,
		OriginalCurrency:   result.Original.Currency,
		ConvertedAmount:    result.Converted.Amount,
		ConvertedCurrency:  result.Converted.Currency,
		ExchangeRate:       result.ExchangeRate,
		FormattedOriginal:  formattedOriginal,
		FormattedConverted: formattedConverted,
	})
}

// RegisterRoutes registers currency routes
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	curr := rg.Group("/currency")
	{
		curr.GET("/currencies", h.GetCurrencies)
		curr.GET("/currencies/:code", h.GetCurrency)
		curr.GET("/rates", h.GetAllRates)
		curr.GET("/rate", h.GetExchangeRate)
		curr.POST("/convert", h.Convert)
	}
}
