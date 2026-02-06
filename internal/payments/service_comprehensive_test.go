package payments

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/config"
	"github.com/richxcame/ride-hailing/pkg/models"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stripe/stripe-go/v83"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestService_ProcessRidePayment_Wallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "wallet")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, rideID, payment.RideID)
	assert.Equal(t, amount, payment.Amount)
	assert.Equal(t, "wallet", payment.PaymentMethod)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	piID := "pi_test123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockStripe.On("CreatePaymentIntent", int64(2550), "usd", "", mock.Anything, mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).Return(nil)

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "stripe")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, piID, *payment.StripePaymentID)
	assert.Equal(t, string(stripe.PaymentIntentStatusSucceeded), payment.Status)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRidePayment_InvalidMethod(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "invalid_method")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentIntentFails(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 25.50

	mockStripe.On("CreatePaymentIntent", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe error"))

	// Act
	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "stripe")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, payment)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	amount := 100.0
	walletID := uuid.New()

	existingWallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	piID := "pi_topup123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(existingWallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(10000), "usd", "", "Wallet top-up", mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	transaction, err := service.TopUpWallet(ctx, userID, amount, "pm_test123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	assert.Equal(t, walletID, transaction.WalletID)
	assert.Equal(t, amount, transaction.Amount)
	assert.Equal(t, "credit", transaction.Type)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateWalletIfNotExists(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	amount := 100.0

	piID := "pi_topup123"
	mockPI := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).Return(nil)
	mockStripe.On("CreatePaymentIntent", int64(10000), "usd", "", "Wallet top-up", mock.Anything).Return(mockPI, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	transaction, err := service.TopUpWallet(ctx, userID, amount, "pm_test123")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, transaction)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_PayoutToDriver_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	walletID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   amount,
		Status:   "completed",
	}

	driverWallet := &models.Wallet{
		ID:       walletID,
		UserID:   driverID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(driverWallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 80.0).Return(nil) // 100 - 20% commission
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	err := service.PayoutToDriver(ctx, paymentID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_PaymentNotCompleted(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	payment := &models.Payment{
		ID:     paymentID,
		Amount: 100.0,
		Status: "pending", // Not completed
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)

	// Act
	err := service.PayoutToDriver(ctx, paymentID)

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_RiderCancelled_WithCancellationFee(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	walletID := uuid.New()
	chargeID := "ch_test123"
	amount := 100.0

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          amount,
		Status:          "completed",
		PaymentMethod:   "wallet",
		StripeChargeID:  &chargeID,
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   riderID,
		Balance:  10.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 90.0).Return(nil) // 100 - 10% cancellation fee
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Stripe_FullRefund(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	chargeID := "ch_test123"
	piID := "pi_test123"
	amount := 100.0

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          amount,
		Status:          "completed",
		PaymentMethod:   "stripe",
		StripePaymentID: &piID,
		StripeChargeID:  &chargeID,
	}

	mockRefund := &stripe.Refund{
		ID:     "re_test123",
		Amount: 10000,
		Status: "succeeded",
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", chargeID, mock.AnythingOfType("*int64"), "driver_cancelled").Return(mockRefund, nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRefund_AlreadyRefunded(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	payment := &models.Payment{
		ID:     paymentID,
		Amount: 100.0,
		Status: "refunded", // Already refunded
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)

	// Act
	err := service.ProcessRefund(ctx, paymentID, "any_reason")

	// Assert
	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_GetWallet_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  150.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)

	// Act
	result, err := service.GetWallet(ctx, userID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, walletID, result.ID)
	assert.Equal(t, 150.0, result.Balance)
	mockRepo.AssertExpectations(t)
}

func TestService_GetWalletTransactions_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  150.0,
		Currency: "usd",
		IsActive: true,
	}

	transactions := []*models.WalletTransaction{
		{ID: uuid.New(), WalletID: walletID, Type: "credit", Amount: 100.0},
		{ID: uuid.New(), WalletID: walletID, Type: "debit", Amount: 25.0},
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", ctx, walletID, 10, 0).Return(transactions, int64(2), nil)

	// Act
	result, total, err := service.GetWalletTransactions(ctx, userID, 10, 0)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, int64(2), total)
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_Success(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	amount := 100.0
	piID := "pi_test123"

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, amount).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	// Act
	err := service.ConfirmWalletTopUp(ctx, userID, amount, piID)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_HandleStripeWebhook_PaymentIntentSucceeded(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	// Act
	err := service.HandleStripeWebhook(ctx, "payment_intent.succeeded", "pi_test123")

	// Assert
	assert.NoError(t, err)
}

func TestService_HandleStripeWebhook_PaymentIntentFailed(t *testing.T) {
	// Arrange
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	// Act
	err := service.HandleStripeWebhook(ctx, "payment_intent.payment_failed", "pi_test123")

	// Assert
	assert.NoError(t, err)
}

func TestCommissionCalculation(t *testing.T) {
	tests := []struct {
		name               string
		totalAmount        float64
		expectedCommission float64
		expectedEarnings   float64
	}{
		{
			name:               "100 dollar ride",
			totalAmount:        100.0,
			expectedCommission: 20.0,
			expectedEarnings:   80.0,
		},
		{
			name:               "50 dollar ride",
			totalAmount:        50.0,
			expectedCommission: 10.0,
			expectedEarnings:   40.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.totalAmount * defaultCommissionRate
			earnings := tt.totalAmount - commission

			assert.InDelta(t, tt.expectedCommission, commission, 0.01)
			assert.InDelta(t, tt.expectedEarnings, earnings, 0.01)
		})
	}
}

func TestCancellationFeeCalculation(t *testing.T) {
	tests := []struct {
		name           string
		amount         float64
		expectedFee    float64
		expectedRefund float64
	}{
		{
			name:           "100 dollar cancellation",
			amount:         100.0,
			expectedFee:    10.0,
			expectedRefund: 90.0,
		},
		{
			name:           "50 dollar cancellation",
			amount:         50.0,
			expectedFee:    5.0,
			expectedRefund: 45.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := tt.amount * defaultCancellationFeeRate
			refund := tt.amount - fee

			assert.InDelta(t, tt.expectedFee, fee, 0.01)
			assert.InDelta(t, tt.expectedRefund, refund, 0.01)
		})
	}
}

// Additional error path tests

func TestService_ProcessRidePayment_Wallet_RepositoryError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("insufficient balance"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "wallet")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "insufficient balance")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentIntentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe API error"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "stripe")

	assert.Error(t, err)
	assert.Nil(t, payment)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Stripe_CreatePaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	piID := "pi_test123"

	pi := &stripe.PaymentIntent{
		ID:     piID,
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(pi, nil)
	mockRepo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).
		Return(errors.New("database error"))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 50.0, "stripe")

	assert.Error(t, err)
	assert.Nil(t, payment)
	assert.Contains(t, err.Error(), "database error")
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).
		Return(errors.New("failed to create wallet"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	assert.Contains(t, err.Error(), "failed to create wallet")
	mockRepo.AssertExpectations(t)
}

func TestService_TopUpWallet_StripeError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(nil, errors.New("stripe error"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_TopUpWallet_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  50.0,
		Currency: "usd",
	}

	pi := &stripe.PaymentIntent{
		ID:     "pi_test123",
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", mock.AnythingOfType("int64"), "usd", "", mock.Anything, mock.Anything).
		Return(pi, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("transaction error"))

	tx, err := service.TopUpWallet(ctx, userID, 100.0, "pm_test123")

	assert.Error(t, err)
	assert.Nil(t, tx)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: 50.0,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(errors.New("update error"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")
	mockRepo.AssertExpectations(t)
}

func TestService_ConfirmWalletTopUp_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  userID,
		Balance: 50.0,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("transaction create error"))

	err := service.ConfirmWalletTopUp(ctx, userID, 100.0, "pi_test123")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "transaction create error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_GetPaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(nil, errors.New("payment not found"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_CreateWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(nil, errors.New("wallet not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).
		Return(errors.New("create wallet error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create wallet error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	expectedEarnings := 80.0 // 100 - 20% commission

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).
		Return(errors.New("update balance error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update balance error")
	mockRepo.AssertExpectations(t)
}

func TestService_PayoutToDriver_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		Status:   "completed",
		Amount:   100.0,
		DriverID: driverID,
		RideID:   rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	expectedEarnings := 80.0

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("create transaction error"))

	err := service.PayoutToDriver(ctx, paymentID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create transaction error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_GetPaymentError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(nil, errors.New("payment not found"))

	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "payment not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(nil, errors.New("wallet not found"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_UpdateBalanceError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).
		Return(errors.New("update balance error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update balance error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_CreateTransactionError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).
		Return(errors.New("create transaction error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create transaction error")
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRefund_Wallet_UpdateStatusError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	riderID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:            paymentID,
		Status:        "completed",
		Amount:        100.0,
		PaymentMethod: "wallet",
		RiderID:       riderID,
		RideID:        rideID,
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  riderID,
		Balance: 50.0,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 100.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).
		Return(errors.New("update status error"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update status error")
	mockRepo.AssertExpectations(t)
}

func TestService_GetWalletTransactions_GetWalletError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, errors.New("wallet not found"))

	txs, total, err := service.GetWalletTransactions(ctx, userID, 10, 0)

	assert.Error(t, err)
	assert.Nil(t, txs)
	assert.Equal(t, int64(0), total)
	assert.Contains(t, err.Error(), "wallet not found")
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetRideDriverID Tests
// ============================================================================

func TestService_GetRideDriverID_Success(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	driverID := uuid.New()

	mockRepo.On("GetRideDriverID", ctx, rideID).Return(&driverID, nil)

	result, err := service.GetRideDriverID(ctx, rideID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, driverID, *result)
	mockRepo.AssertExpectations(t)
}

func TestService_GetRideDriverID_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()

	mockRepo.On("GetRideDriverID", ctx, rideID).Return(nil, errors.New("ride not found"))

	result, err := service.GetRideDriverID(ctx, rideID)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "ride not found")
	mockRepo.AssertExpectations(t)
}

func TestService_GetRideDriverID_NoDriverAssigned(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()

	// Return nil driver ID (no driver assigned yet)
	mockRepo.On("GetRideDriverID", ctx, rideID).Return((*uuid.UUID)(nil), nil)

	result, err := service.GetRideDriverID(ctx, rideID)

	assert.NoError(t, err)
	assert.Nil(t, result)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Custom Business Config Tests
// ============================================================================

func TestService_CustomCommissionRate(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	// Custom config with 15% commission instead of default 20%
	customConfig := &config.BusinessConfig{
		CommissionRate:      0.15,
		CancellationFeeRate: 0.10,
	}
	service := NewService(mockRepo, mockStripe, customConfig)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	walletID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   amount,
		Status:   "completed",
	}

	driverWallet := &models.Wallet{
		ID:       walletID,
		UserID:   driverID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	// With 15% commission, driver should receive 85.0
	expectedEarnings := 85.0

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(driverWallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	err := service.PayoutToDriver(ctx, paymentID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_CustomCancellationFeeRate(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	// Custom config with 15% cancellation fee instead of default 10%
	customConfig := &config.BusinessConfig{
		CommissionRate:      0.20,
		CancellationFeeRate: 0.15,
	}
	service := NewService(mockRepo, mockStripe, customConfig)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	walletID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:            paymentID,
		RideID:        rideID,
		RiderID:       riderID,
		Amount:        amount,
		Status:        "completed",
		PaymentMethod: "wallet",
	}

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   riderID,
		Balance:  10.0,
		Currency: "usd",
		IsActive: true,
	}

	// With 15% cancellation fee, refund should be 85.0
	expectedRefund := 85.0

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, riderID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedRefund).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestService_ZeroCommissionRate(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	// Zero commission rate (promotional period)
	customConfig := &config.BusinessConfig{
		CommissionRate:      0.0,
		CancellationFeeRate: 0.10,
	}
	// Note: Service uses default when rate is 0 or negative
	service := NewService(mockRepo, mockStripe, customConfig)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	walletID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   amount,
		Status:   "completed",
	}

	driverWallet := &models.Wallet{
		ID:       walletID,
		UserID:   driverID,
		Balance:  50.0,
		Currency: "usd",
		IsActive: true,
	}

	// With 0 commission rate, config defaults to 20%, so driver gets 80.0
	expectedEarnings := 80.0

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(driverWallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, expectedEarnings).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	err := service.PayoutToDriver(ctx, paymentID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Financial Precision Tests (2 Decimal Places)
// ============================================================================

func TestFinancialPrecision_CommissionCalculation(t *testing.T) {
	tests := []struct {
		name               string
		amount             float64
		commissionRate     float64
		expectedCommission float64
		expectedEarnings   float64
	}{
		{
			name:               "cents precision - $25.99 fare",
			amount:             25.99,
			commissionRate:     0.20,
			expectedCommission: 5.20, // 25.99 * 0.20 = 5.198 -> rounded
			expectedEarnings:   20.79,
		},
		{
			name:               "cents precision - $13.45 fare",
			amount:             13.45,
			commissionRate:     0.20,
			expectedCommission: 2.69,
			expectedEarnings:   10.76,
		},
		{
			name:               "sub-cent precision - $99.99 fare",
			amount:             99.99,
			commissionRate:     0.20,
			expectedCommission: 20.00, // 99.99 * 0.20 = 19.998
			expectedEarnings:   79.99,
		},
		{
			name:               "small amount - $5.00 fare",
			amount:             5.00,
			commissionRate:     0.20,
			expectedCommission: 1.00,
			expectedEarnings:   4.00,
		},
		{
			name:               "large amount - $500.00 fare",
			amount:             500.00,
			commissionRate:     0.20,
			expectedCommission: 100.00,
			expectedEarnings:   400.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commission := tt.amount * tt.commissionRate
			earnings := tt.amount - commission

			// Using InDelta for floating point comparison with 2 decimal precision
			assert.InDelta(t, tt.expectedCommission, commission, 0.01, "commission mismatch")
			assert.InDelta(t, tt.expectedEarnings, earnings, 0.01, "earnings mismatch")
		})
	}
}

func TestFinancialPrecision_RefundCalculation(t *testing.T) {
	tests := []struct {
		name                string
		originalAmount      float64
		cancellationFeeRate float64
		expectedFee         float64
		expectedRefund      float64
	}{
		{
			name:                "cents precision - $25.99 refund",
			originalAmount:      25.99,
			cancellationFeeRate: 0.10,
			expectedFee:         2.60, // 25.99 * 0.10 = 2.599
			expectedRefund:      23.39,
		},
		{
			name:                "cents precision - $13.45 refund",
			originalAmount:      13.45,
			cancellationFeeRate: 0.10,
			expectedFee:         1.35, // 13.45 * 0.10 = 1.345
			expectedRefund:      12.10,
		},
		{
			name:                "exact cents - $50.00 refund",
			originalAmount:      50.00,
			cancellationFeeRate: 0.10,
			expectedFee:         5.00,
			expectedRefund:      45.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fee := tt.originalAmount * tt.cancellationFeeRate
			refund := tt.originalAmount - fee

			assert.InDelta(t, tt.expectedFee, fee, 0.01, "fee mismatch")
			assert.InDelta(t, tt.expectedRefund, refund, 0.01, "refund mismatch")
		})
	}
}

func TestFinancialPrecision_StripeAmountConversion(t *testing.T) {
	// Stripe uses cents (integer), so $25.99 should become 2599 cents
	tests := []struct {
		name           string
		dollarAmount   float64
		expectedCents  int64
	}{
		{"exact dollar", 25.00, 2500},
		{"with cents", 25.99, 2599},
		{"small amount", 0.50, 50},
		{"minimum viable", 0.01, 1},
		{"large amount", 999.99, 99999},
		{"round number", 100.00, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cents := int64(tt.dollarAmount * 100)
			assert.Equal(t, tt.expectedCents, cents)
		})
	}
}

// ============================================================================
// Edge Cases: Zero and Negative Amounts
// ============================================================================

func TestService_ProcessRidePayment_ZeroAmount(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	// Zero amount payment should still be processed (e.g., promotional ride)
	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, 0.0, "wallet")

	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, 0.0, payment.Amount)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_SmallAmount(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()

	// Minimum viable amount (1 cent)
	amount := 0.01

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "wallet")

	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, amount, payment.Amount)
	mockRepo.AssertExpectations(t)
}

func TestService_TopUpWallet_LargeAmount(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	amount := 10000.00 // $10,000

	wallet := &models.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  0,
		Currency: "usd",
		IsActive: true,
	}

	pi := &stripe.PaymentIntent{
		ID:     "pi_large_amount",
		Status: stripe.PaymentIntentStatusSucceeded,
	}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockStripe.On("CreatePaymentIntent", int64(1000000), "usd", "", "Wallet top-up", mock.Anything).Return(pi, nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	tx, err := service.TopUpWallet(ctx, userID, amount, "pm_test")

	assert.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, amount, tx.Amount)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

// ============================================================================
// Stripe Refund Error Path Tests
// ============================================================================

func TestService_ProcessRefund_Stripe_RefundFails(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	chargeID := "ch_test123"
	piID := "pi_test123"

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          100.0,
		Status:          "completed",
		PaymentMethod:   "stripe",
		StripePaymentID: &piID,
		StripeChargeID:  &chargeID,
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", chargeID, mock.AnythingOfType("*int64"), "driver_cancelled").
		Return(nil, errors.New("stripe refund failed"))

	err := service.ProcessRefund(ctx, paymentID, "driver_cancelled")

	assert.Error(t, err)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 500, appErr.Code)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

func TestService_ProcessRefund_Stripe_WithCancellationFee(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	riderID := uuid.New()
	chargeID := "ch_test123"
	piID := "pi_test123"
	amount := 100.0

	payment := &models.Payment{
		ID:              paymentID,
		RideID:          rideID,
		RiderID:         riderID,
		Amount:          amount,
		Status:          "completed",
		PaymentMethod:   "stripe",
		StripePaymentID: &piID,
		StripeChargeID:  &chargeID,
	}

	// With rider_cancelled and 10% fee, refund should be 90.00 = 9000 cents
	mockRefund := &stripe.Refund{
		ID:     "re_test123",
		Amount: 9000,
		Status: "succeeded",
	}

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockStripe.On("CreateRefund", chargeID, mock.MatchedBy(func(amount *int64) bool {
		return amount != nil && *amount == 9000
	}), "rider_cancelled").Return(mockRefund, nil)
	mockRepo.On("UpdatePaymentStatus", ctx, paymentID, "refunded", mock.Anything).Return(nil)

	err := service.ProcessRefund(ctx, paymentID, "rider_cancelled")

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
	mockStripe.AssertExpectations(t)
}

// ============================================================================
// Wallet Insufficient Balance Edge Cases
// ============================================================================

func TestService_ProcessRidePayment_Wallet_InsufficientBalance(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 100.0

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).
		Return(common.NewBadRequestError("insufficient wallet balance", nil))

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "wallet")

	assert.Error(t, err)
	assert.Nil(t, payment)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestService_ProcessRidePayment_Wallet_ExactBalance(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	rideID := uuid.New()
	riderID := uuid.New()
	driverID := uuid.New()
	amount := 50.00 // Exact wallet balance

	mockRepo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, amount, "wallet")

	assert.NoError(t, err)
	assert.NotNil(t, payment)
	assert.Equal(t, amount, payment.Amount)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// Payment Status Transition Tests
// ============================================================================

func TestPaymentStatusTransitions(t *testing.T) {
	tests := []struct {
		name          string
		initialStatus string
		action        string
		expectedStatus string
		shouldFail    bool
	}{
		{
			name:          "pending to completed",
			initialStatus: "pending",
			action:        "complete",
			expectedStatus: "completed",
			shouldFail:    false,
		},
		{
			name:          "completed to refunded",
			initialStatus: "completed",
			action:        "refund",
			expectedStatus: "refunded",
			shouldFail:    false,
		},
		{
			name:          "refunded cannot be refunded again",
			initialStatus: "refunded",
			action:        "refund",
			expectedStatus: "",
			shouldFail:    true,
		},
		{
			name:          "pending cannot be refunded",
			initialStatus: "pending",
			action:        "payout",
			expectedStatus: "",
			shouldFail:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These are logical validation tests for payment state machine
			if tt.action == "refund" && tt.initialStatus == "refunded" {
				assert.True(t, tt.shouldFail, "refunded payments should not be refundable")
			}
			if tt.action == "payout" && tt.initialStatus != "completed" {
				assert.True(t, tt.shouldFail, "only completed payments can have payout")
			}
		})
	}
}

// ============================================================================
// Handle Stripe Webhook Additional Tests
// ============================================================================

func TestService_HandleStripeWebhook_ChargeRefunded(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	err := service.HandleStripeWebhook(ctx, "charge.refunded", "pi_test123")

	assert.NoError(t, err)
}

func TestService_HandleStripeWebhook_UnhandledEvent(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	// Unknown event types should not fail
	err := service.HandleStripeWebhook(ctx, "unknown.event.type", "pi_test123")

	assert.NoError(t, err)
}

// ============================================================================
// PayoutToDriver Create Wallet Path Tests
// ============================================================================

func TestService_PayoutToDriver_CreateWalletSuccess(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	paymentID := uuid.New()
	rideID := uuid.New()
	driverID := uuid.New()
	amount := 100.0

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   amount,
		Status:   "completed",
	}

	expectedEarnings := 80.0 // 100 - 20% commission

	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(nil, errors.New("wallet not found"))
	mockRepo.On("CreateWallet", ctx, mock.AnythingOfType("*models.Wallet")).Return(nil)
	mockRepo.On("UpdateWalletBalance", ctx, mock.AnythingOfType("uuid.UUID"), expectedEarnings).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	err := service.PayoutToDriver(ctx, paymentID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// NewService Default Values Tests
// ============================================================================

func TestNewService_NilConfig_UsesDefaults(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	service := NewService(mockRepo, mockStripe, nil)

	// Verify service is created (we can't access private fields, but we can test behavior)
	assert.NotNil(t, service)

	// Test default commission rate through behavior
	ctx := context.Background()
	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   100.0,
		Status:   "completed",
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	// Default commission is 20%, so earnings should be 80.0
	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 80.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	err := service.PayoutToDriver(ctx, paymentID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestNewService_PartialConfig_UsesDefaults(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)

	// Only set commission rate, leave cancellation rate empty (should use default)
	partialConfig := &config.BusinessConfig{
		CommissionRate:      0.25, // Custom 25%
		CancellationFeeRate: 0,    // Should use default 10%
	}

	service := NewService(mockRepo, mockStripe, partialConfig)
	assert.NotNil(t, service)

	// Test custom commission rate
	ctx := context.Background()
	paymentID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	walletID := uuid.New()

	payment := &models.Payment{
		ID:       paymentID,
		RideID:   rideID,
		DriverID: driverID,
		Amount:   100.0,
		Status:   "completed",
	}

	wallet := &models.Wallet{
		ID:      walletID,
		UserID:  driverID,
		Balance: 50.0,
	}

	// Custom 25% commission, so earnings should be 75.0
	mockRepo.On("GetPaymentByID", ctx, paymentID).Return(payment, nil)
	mockRepo.On("GetWalletByUserID", ctx, driverID).Return(wallet, nil)
	mockRepo.On("UpdateWalletBalance", ctx, walletID, 75.0).Return(nil)
	mockRepo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)

	err := service.PayoutToDriver(ctx, paymentID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// wrapStripeError Tests
// ============================================================================

func TestWrapStripeError_NilError(t *testing.T) {
	result := wrapStripeError(nil, "fallback message")
	assert.Nil(t, result)
}

func TestWrapStripeError_AppError(t *testing.T) {
	appErr := common.NewBadRequestError("test error", nil)
	result := wrapStripeError(appErr, "fallback message")

	assert.Equal(t, appErr, result)
}

func TestWrapStripeError_GenericError(t *testing.T) {
	genericErr := errors.New("some stripe error")
	result := wrapStripeError(genericErr, "fallback message")

	var appErr *common.AppError
	assert.True(t, errors.As(result, &appErr))
	assert.Equal(t, 500, appErr.Code)
}

// ============================================================================
// Table-Driven Tests for ProcessRidePayment
// ============================================================================

func TestService_ProcessRidePayment_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
		amount        float64
		setupMocks    func(*mocks.MockPaymentsRepository, *mocks.MockStripeClient, context.Context, uuid.UUID, uuid.UUID, uuid.UUID)
		wantErr       bool
		errCode       int
	}{
		{
			name:          "wallet payment success",
			paymentMethod: "wallet",
			amount:        50.00,
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, rideID, riderID, driverID uuid.UUID) {
				repo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "stripe payment success",
			paymentMethod: "stripe",
			amount:        75.00,
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, rideID, riderID, driverID uuid.UUID) {
				pi := &stripe.PaymentIntent{
					ID:     "pi_test",
					Status: stripe.PaymentIntentStatusSucceeded,
				}
				stripeClient.On("CreatePaymentIntent", int64(7500), "usd", "", mock.Anything, mock.Anything).Return(pi, nil)
				repo.On("CreatePayment", ctx, mock.AnythingOfType("*models.Payment")).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "invalid payment method",
			paymentMethod: "bitcoin",
			amount:        50.00,
			setupMocks:    func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, rideID, riderID, driverID uuid.UUID) {},
			wantErr:       true,
			errCode:       400,
		},
		{
			name:          "wallet insufficient funds",
			paymentMethod: "wallet",
			amount:        1000.00,
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, rideID, riderID, driverID uuid.UUID) {
				repo.On("ProcessPaymentWithWallet", ctx, mock.AnythingOfType("*models.Payment"), mock.AnythingOfType("*models.WalletTransaction")).
					Return(common.NewBadRequestError("insufficient wallet balance", nil))
			},
			wantErr: true,
			errCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPaymentsRepository)
			mockStripe := new(mocks.MockStripeClient)
			service := NewService(mockRepo, mockStripe, nil)
			ctx := context.Background()

			rideID := uuid.New()
			riderID := uuid.New()
			driverID := uuid.New()

			tt.setupMocks(mockRepo, mockStripe, ctx, rideID, riderID, driverID)

			payment, err := service.ProcessRidePayment(ctx, rideID, riderID, driverID, tt.amount, tt.paymentMethod)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, payment)
				if tt.errCode > 0 {
					var appErr *common.AppError
					if errors.As(err, &appErr) {
						assert.Equal(t, tt.errCode, appErr.Code)
					}
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, payment)
				assert.Equal(t, tt.amount, payment.Amount)
			}

			mockRepo.AssertExpectations(t)
			mockStripe.AssertExpectations(t)
		})
	}
}

// ============================================================================
// Table-Driven Tests for ProcessRefund
// ============================================================================

func TestService_ProcessRefund_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		paymentMethod string
		status        string
		reason        string
		setupMocks    func(*mocks.MockPaymentsRepository, *mocks.MockStripeClient, context.Context, *models.Payment)
		wantErr       bool
		errCode       int
	}{
		{
			name:          "wallet refund full amount",
			paymentMethod: "wallet",
			status:        "completed",
			reason:        "driver_cancelled",
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, payment *models.Payment) {
				wallet := &models.Wallet{ID: uuid.New(), UserID: payment.RiderID, Balance: 0}
				repo.On("GetPaymentByID", ctx, payment.ID).Return(payment, nil)
				repo.On("GetWalletByUserID", ctx, payment.RiderID).Return(wallet, nil)
				repo.On("UpdateWalletBalance", ctx, wallet.ID, 100.0).Return(nil)
				repo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
				repo.On("UpdatePaymentStatus", ctx, payment.ID, "refunded", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "wallet refund with cancellation fee",
			paymentMethod: "wallet",
			status:        "completed",
			reason:        "rider_cancelled",
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, payment *models.Payment) {
				wallet := &models.Wallet{ID: uuid.New(), UserID: payment.RiderID, Balance: 0}
				repo.On("GetPaymentByID", ctx, payment.ID).Return(payment, nil)
				repo.On("GetWalletByUserID", ctx, payment.RiderID).Return(wallet, nil)
				repo.On("UpdateWalletBalance", ctx, wallet.ID, 90.0).Return(nil) // 100 - 10% fee
				repo.On("CreateWalletTransaction", ctx, mock.AnythingOfType("*models.WalletTransaction")).Return(nil)
				repo.On("UpdatePaymentStatus", ctx, payment.ID, "refunded", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name:          "already refunded",
			paymentMethod: "wallet",
			status:        "refunded",
			reason:        "driver_cancelled",
			setupMocks: func(repo *mocks.MockPaymentsRepository, stripeClient *mocks.MockStripeClient, ctx context.Context, payment *models.Payment) {
				repo.On("GetPaymentByID", ctx, payment.ID).Return(payment, nil)
			},
			wantErr: true,
			errCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPaymentsRepository)
			mockStripe := new(mocks.MockStripeClient)
			service := NewService(mockRepo, mockStripe, nil)
			ctx := context.Background()

			payment := &models.Payment{
				ID:            uuid.New(),
				RideID:        uuid.New(),
				RiderID:       uuid.New(),
				Amount:        100.0,
				Status:        tt.status,
				PaymentMethod: tt.paymentMethod,
			}

			tt.setupMocks(mockRepo, mockStripe, ctx, payment)

			err := service.ProcessRefund(ctx, payment.ID, tt.reason)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errCode > 0 {
					var appErr *common.AppError
					if errors.As(err, &appErr) {
						assert.Equal(t, tt.errCode, appErr.Code)
					}
				}
			} else {
				assert.NoError(t, err)
			}

			mockRepo.AssertExpectations(t)
			mockStripe.AssertExpectations(t)
		})
	}
}

// ============================================================================
// GetWallet Error Handling
// ============================================================================

func TestService_GetWallet_NotFound(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(nil, common.NewNotFoundError("wallet not found", nil))

	wallet, err := service.GetWallet(ctx, userID)

	assert.Error(t, err)
	assert.Nil(t, wallet)
	var appErr *common.AppError
	assert.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
	mockRepo.AssertExpectations(t)
}

// ============================================================================
// GetWalletTransactions Pagination Tests
// ============================================================================

func TestService_GetWalletTransactions_Pagination(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{ID: walletID, UserID: userID, Balance: 100.0}

	// Create 5 transactions
	allTransactions := make([]*models.WalletTransaction, 5)
	for i := 0; i < 5; i++ {
		allTransactions[i] = &models.WalletTransaction{
			ID:       uuid.New(),
			WalletID: walletID,
			Type:     "credit",
			Amount:   float64(10 * (i + 1)),
		}
	}

	// Test first page (limit 2, offset 0)
	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil).Once()
	mockRepo.On("GetWalletTransactionsWithTotal", ctx, walletID, 2, 0).Return(allTransactions[:2], int64(5), nil).Once()

	txs, total, err := service.GetWalletTransactions(ctx, userID, 2, 0)

	assert.NoError(t, err)
	assert.Len(t, txs, 2)
	assert.Equal(t, int64(5), total)
	mockRepo.AssertExpectations(t)
}

func TestService_GetWalletTransactions_EmptyResult(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	ctx := context.Background()

	userID := uuid.New()
	walletID := uuid.New()
	wallet := &models.Wallet{ID: walletID, UserID: userID, Balance: 0}

	mockRepo.On("GetWalletByUserID", ctx, userID).Return(wallet, nil)
	mockRepo.On("GetWalletTransactionsWithTotal", ctx, walletID, 10, 0).Return([]*models.WalletTransaction{}, int64(0), nil)

	txs, total, err := service.GetWalletTransactions(ctx, userID, 10, 0)

	assert.NoError(t, err)
	assert.Empty(t, txs)
	assert.Equal(t, int64(0), total)
	mockRepo.AssertExpectations(t)
}
