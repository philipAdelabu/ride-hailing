package fraud

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockFraudRepository struct {
	mock.Mock
}

func (m *mockFraudRepository) GetUserRiskProfile(ctx context.Context, userID uuid.UUID) (*UserRiskProfile, error) {
	args := m.Called(ctx, userID)
	profile, _ := args.Get(0).(*UserRiskProfile)
	return profile, args.Error(1)
}

func (m *mockFraudRepository) UpdateUserRiskProfile(ctx context.Context, profile *UserRiskProfile) error {
	args := m.Called(ctx, profile)
	return args.Error(0)
}

func (m *mockFraudRepository) GetPaymentFraudIndicators(ctx context.Context, userID uuid.UUID) (*PaymentFraudIndicators, error) {
	args := m.Called(ctx, userID)
	indicators, _ := args.Get(0).(*PaymentFraudIndicators)
	return indicators, args.Error(1)
}

func (m *mockFraudRepository) GetRideFraudIndicators(ctx context.Context, userID uuid.UUID) (*RideFraudIndicators, error) {
	args := m.Called(ctx, userID)
	indicators, _ := args.Get(0).(*RideFraudIndicators)
	return indicators, args.Error(1)
}

func (m *mockFraudRepository) CreateFraudAlert(ctx context.Context, alert *FraudAlert) error {
	args := m.Called(ctx, alert)
	return args.Error(0)
}

func (m *mockFraudRepository) GetFraudAlertByID(ctx context.Context, alertID uuid.UUID) (*FraudAlert, error) {
	args := m.Called(ctx, alertID)
	alert, _ := args.Get(0).(*FraudAlert)
	return alert, args.Error(1)
}

func (m *mockFraudRepository) GetAlertsByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*FraudAlert, error) {
	args := m.Called(ctx, userID, limit, offset)
	alerts, _ := args.Get(0).([]*FraudAlert)
	return alerts, args.Error(1)
}

func (m *mockFraudRepository) GetPendingAlerts(ctx context.Context, limit, offset int) ([]*FraudAlert, error) {
	args := m.Called(ctx, limit, offset)
	alerts, _ := args.Get(0).([]*FraudAlert)
	return alerts, args.Error(1)
}

func (m *mockFraudRepository) UpdateAlertStatus(ctx context.Context, alertID uuid.UUID, status FraudAlertStatus, investigatorID *uuid.UUID, notes, actionTaken string) error {
	args := m.Called(ctx, alertID, status, investigatorID, notes, actionTaken)
	return args.Error(0)
}

func TestAnalyzeUserCalculatesRiskAndGeneratesAlerts(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()
	profile := &UserRiskProfile{
		UserID:      userID,
		RiskScore:   10,
		TotalAlerts: 0,
		LastUpdated: time.Now(),
	}

	paymentIndicators := &PaymentFraudIndicators{
		UserID:                 userID,
		FailedPaymentAttempts:  3,
		ChargebackCount:        2,
		MultiplePaymentMethods: 4,
		RapidPaymentChanges:    true,
		RiskScore:              80,
	}

	rideIndicators := &RideFraudIndicators{
		UserID:                 userID,
		ExcessiveCancellations: 5,
		UnusualRidePatterns:    true,
		PromoAbuse:             true,
		RiskScore:              70,
	}

	repo.On("GetUserRiskProfile", ctx, userID).Return(profile, nil).Once()
	repo.On("GetPaymentFraudIndicators", ctx, userID).Return(paymentIndicators, nil).Once()
	repo.On("GetRideFraudIndicators", ctx, userID).Return(rideIndicators, nil).Once()
	repo.On("CreateFraudAlert", ctx, mock.MatchedBy(func(alert *FraudAlert) bool {
		return alert.AlertType == AlertTypePaymentFraud &&
			alert.AlertLevel == AlertLevelHigh &&
			alert.Details["failed_attempts"] == paymentIndicators.FailedPaymentAttempts &&
			alert.RiskScore == paymentIndicators.RiskScore
	})).Return(nil).Once()
	repo.On("CreateFraudAlert", ctx, mock.MatchedBy(func(alert *FraudAlert) bool {
		return alert.AlertType == AlertTypeRideFraud &&
			alert.AlertLevel == AlertLevelHigh &&
			alert.Details["excessive_cancellations"] == rideIndicators.ExcessiveCancellations
	})).Return(nil).Once()

	expectedRisk := paymentIndicators.RiskScore*0.6 + rideIndicators.RiskScore*0.4
	repo.On("UpdateUserRiskProfile", ctx, mock.MatchedBy(func(updated *UserRiskProfile) bool {
		return math.Abs(updated.RiskScore-expectedRisk) < 0.001 &&
			updated.TotalAlerts == 2 &&
			!updated.LastUpdated.IsZero()
	})).Return(nil).Once()

	result, err := service.AnalyzeUser(ctx, userID)
	require.NoError(t, err)
	assert.InDelta(t, expectedRisk, result.RiskScore, 0.001)
	assert.Equal(t, 2, result.TotalAlerts)
	assert.Equal(t, 0, result.CriticalAlerts)
	assert.False(t, result.AccountSuspended)
	repo.AssertExpectations(t)
}

func TestAnalyzeUserSuspendsAccountOnCriticalRisk(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()
	profile := &UserRiskProfile{
		UserID:      userID,
		RiskScore:   20,
		LastUpdated: time.Now(),
	}

	paymentIndicators := &PaymentFraudIndicators{
		UserID:    userID,
		RiskScore: 95,
	}
	rideIndicators := &RideFraudIndicators{
		UserID:    userID,
		RiskScore: 92,
	}

	repo.On("GetUserRiskProfile", ctx, userID).Return(profile, nil).Once()
	repo.On("GetPaymentFraudIndicators", ctx, userID).Return(paymentIndicators, nil).Once()
	repo.On("GetRideFraudIndicators", ctx, userID).Return(rideIndicators, nil).Once()
	repo.On("CreateFraudAlert", ctx, mock.AnythingOfType("*fraud.FraudAlert")).Return(nil).Twice()
	repo.On("UpdateUserRiskProfile", ctx, mock.MatchedBy(func(updated *UserRiskProfile) bool {
		return updated.AccountSuspended &&
			updated.CriticalAlerts == 2 &&
			updated.TotalAlerts == 2 &&
			updated.RiskScore >= 90
	})).Return(nil).Once()

	result, err := service.AnalyzeUser(ctx, userID)
	require.NoError(t, err)
	assert.True(t, result.AccountSuspended)
	assert.Equal(t, 2, result.CriticalAlerts)
	repo.AssertExpectations(t)
}

func TestCreateAlertAppliesDefaultsAndUpdatesProfile(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()
	profile := &UserRiskProfile{
		UserID:      userID,
		RiskScore:   40,
		TotalAlerts: 1,
		LastUpdated: time.Now().Add(-time.Hour),
	}

	alert := &FraudAlert{
		UserID:      userID,
		AlertType:   AlertTypePaymentFraud,
		AlertLevel:  AlertLevelHigh,
		RiskScore:   80,
		Description: "manual alert",
	}

	repo.On("CreateFraudAlert", ctx, mock.MatchedBy(func(saved *FraudAlert) bool {
		return saved.ID != uuid.Nil &&
			saved.Status == AlertStatusPending &&
			!saved.DetectedAt.IsZero()
	})).Return(nil).Once()
	repo.On("GetUserRiskProfile", ctx, userID).Return(profile, nil).Once()
	repo.On("UpdateUserRiskProfile", ctx, mock.MatchedBy(func(updated *UserRiskProfile) bool {
		return updated.TotalAlerts == 2 &&
			updated.RiskScore == 48 &&
			updated.LastAlertAt != nil
	})).Return(nil).Once()

	err := service.CreateAlert(ctx, alert)
	require.NoError(t, err)
	assert.Equal(t, AlertStatusPending, alert.Status)
	assert.NotZero(t, alert.DetectedAt)
	assert.Equal(t, 2, profile.TotalAlerts)
	assert.Equal(t, 48.0, profile.RiskScore)
	assert.NotNil(t, profile.LastAlertAt)
	repo.AssertExpectations(t)
}

func TestDetectPaymentFraudCreatesAlertAboveThreshold(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()
	profile := &UserRiskProfile{
		UserID: userID,
	}

	indicators := &PaymentFraudIndicators{
		UserID:                 userID,
		FailedPaymentAttempts:  2,
		ChargebackCount:        1,
		MultiplePaymentMethods: 3,
		SuspiciousTransactions: 4,
		RapidPaymentChanges:    true,
		RiskScore:              75,
	}

	repo.On("GetPaymentFraudIndicators", ctx, userID).Return(indicators, nil).Once()
	repo.On("CreateFraudAlert", ctx, mock.MatchedBy(func(alert *FraudAlert) bool {
		return alert.AlertType == AlertTypePaymentFraud &&
			alert.AlertLevel == AlertLevelHigh &&
			alert.Details["chargebacks"] == indicators.ChargebackCount &&
			alert.Description == "Payment fraud risk score: 75.0"
	})).Return(nil).Once()
	repo.On("GetUserRiskProfile", ctx, userID).Return(profile, nil).Once()
	repo.On("UpdateUserRiskProfile", ctx, mock.AnythingOfType("*fraud.UserRiskProfile")).Return(nil).Once()

	err := service.DetectPaymentFraud(ctx, userID)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDetectPaymentFraudNoAlertBelowThreshold(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()

	indicators := &PaymentFraudIndicators{
		UserID:    userID,
		RiskScore: 50,
	}

	repo.On("GetPaymentFraudIndicators", ctx, userID).Return(indicators, nil).Once()

	err := service.DetectPaymentFraud(ctx, userID)
	require.NoError(t, err)
	repo.AssertNotCalled(t, "CreateFraudAlert", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestDetectRideFraudCreatesAlertAboveThreshold(t *testing.T) {
	ctx := context.Background()
	repo := new(mockFraudRepository)
	service := NewService(repo)
	userID := uuid.New()
	profile := &UserRiskProfile{UserID: userID}

	indicators := &RideFraudIndicators{
		UserID:                 userID,
		ExcessiveCancellations: 8,
		UnusualRidePatterns:    true,
		FakeGPSDetected:        true,
		PromoAbuse:             true,
		RiskScore:              82,
	}

	repo.On("GetRideFraudIndicators", ctx, userID).Return(indicators, nil).Once()
	repo.On("CreateFraudAlert", ctx, mock.MatchedBy(func(alert *FraudAlert) bool {
		return alert.AlertType == AlertTypeRideFraud &&
			alert.AlertLevel == AlertLevelHigh &&
			alert.Details["fake_gps"] == indicators.FakeGPSDetected
	})).Return(nil).Once()
	repo.On("GetUserRiskProfile", ctx, userID).Return(profile, nil).Once()
	repo.On("UpdateUserRiskProfile", ctx, mock.AnythingOfType("*fraud.UserRiskProfile")).Return(nil).Once()

	err := service.DetectRideFraud(ctx, userID)
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
