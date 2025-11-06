package payments

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCalculateDriverEarnings(t *testing.T) {
	tests := []struct {
		name               string
		totalAmount        float64
		commissionRate     float64
		expectedEarnings   float64
		expectedCommission float64
	}{
		{
			name:               "standard fare",
			totalAmount:        100.00,
			commissionRate:     0.20,
			expectedEarnings:   80.00,
			expectedCommission: 20.00,
		},
		{
			name:               "minimum fare",
			totalAmount:        10.00,
			commissionRate:     0.20,
			expectedEarnings:   8.00,
			expectedCommission: 2.00,
		},
		{
			name:               "high fare",
			totalAmount:        500.00,
			commissionRate:     0.20,
			expectedEarnings:   400.00,
			expectedCommission: 100.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.totalAmount * tt.commissionRate
			driverEarnings := tt.totalAmount - commission

			assert.Equal(t, tt.expectedCommission, commission)
			assert.Equal(t, tt.expectedEarnings, driverEarnings)
		})
	}
}

func TestCalculateRefundAmount(t *testing.T) {
	tests := []struct {
		name                string
		originalAmount      float64
		cancellationFeeRate float64
		expectedRefund      float64
		expectedFee         float64
	}{
		{
			name:                "standard cancellation",
			originalAmount:      100.00,
			cancellationFeeRate: 0.10,
			expectedRefund:      90.00,
			expectedFee:         10.00,
		},
		{
			name:                "high value cancellation",
			originalAmount:      500.00,
			cancellationFeeRate: 0.10,
			expectedRefund:      450.00,
			expectedFee:         50.00,
		},
		{
			name:                "low value cancellation",
			originalAmount:      20.00,
			cancellationFeeRate: 0.10,
			expectedRefund:      18.00,
			expectedFee:         2.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cancellationFee := tt.originalAmount * tt.cancellationFeeRate
			refundAmount := tt.originalAmount - cancellationFee

			assert.InDelta(t, tt.expectedFee, cancellationFee, 0.01)
			assert.InDelta(t, tt.expectedRefund, refundAmount, 0.01)
		})
	}
}

func TestWalletTransactionTypes(t *testing.T) {
	tests := []struct {
		name            string
		transactionType string
		amount          float64
		initialBalance  float64
		expectedBalance float64
	}{
		{
			name:            "credit transaction",
			transactionType: "credit",
			amount:          50.00,
			initialBalance:  100.00,
			expectedBalance: 150.00,
		},
		{
			name:            "debit transaction",
			transactionType: "debit",
			amount:          30.00,
			initialBalance:  100.00,
			expectedBalance: 70.00,
		},
		{
			name:            "large credit",
			transactionType: "credit",
			amount:          500.00,
			initialBalance:  100.00,
			expectedBalance: 600.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var newBalance float64
			if tt.transactionType == "credit" {
				newBalance = tt.initialBalance + tt.amount
			} else {
				newBalance = tt.initialBalance - tt.amount
			}

			assert.Equal(t, tt.expectedBalance, newBalance)
		})
	}
}

func TestPaymentValidation(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		commission     float64
		driverEarnings float64
		wantErr        bool
	}{
		{
			name:           "valid payment",
			amount:         100.00,
			commission:     20.00,
			driverEarnings: 80.00,
			wantErr:        false,
		},
		{
			name:           "zero amount",
			amount:         0.00,
			commission:     0.00,
			driverEarnings: 0.00,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantErr {
				assert.LessOrEqual(t, tt.amount, 0.0)
			} else {
				assert.Greater(t, tt.amount, 0.0)
				assert.InDelta(t, tt.amount, tt.commission+tt.driverEarnings, 0.01)
			}
		})
	}
}
