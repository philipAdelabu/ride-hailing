package loyalty

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

// ========================================
// MOCK REPOSITORY
// ========================================

type mockLoyaltyRepository struct {
	mock.Mock
}

func (m *mockLoyaltyRepository) GetRiderLoyalty(ctx context.Context, riderID uuid.UUID) (*RiderLoyalty, error) {
	args := m.Called(ctx, riderID)
	account, _ := args.Get(0).(*RiderLoyalty)
	return account, args.Error(1)
}

func (m *mockLoyaltyRepository) CreateRiderLoyalty(ctx context.Context, account *RiderLoyalty) error {
	args := m.Called(ctx, account)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) UpdatePoints(ctx context.Context, riderID uuid.UUID, earnedPoints, tierPoints int) error {
	args := m.Called(ctx, riderID, earnedPoints, tierPoints)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) DeductPoints(ctx context.Context, riderID uuid.UUID, points int) error {
	args := m.Called(ctx, riderID, points)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) UpdateTier(ctx context.Context, riderID uuid.UUID, tierID uuid.UUID) error {
	args := m.Called(ctx, riderID, tierID)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) UpdateStreak(ctx context.Context, riderID uuid.UUID, streakDays int) error {
	args := m.Called(ctx, riderID, streakDays)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) GetTier(ctx context.Context, tierID uuid.UUID) (*LoyaltyTier, error) {
	args := m.Called(ctx, tierID)
	tier, _ := args.Get(0).(*LoyaltyTier)
	return tier, args.Error(1)
}

func (m *mockLoyaltyRepository) GetTierByName(ctx context.Context, name TierName) (*LoyaltyTier, error) {
	args := m.Called(ctx, name)
	tier, _ := args.Get(0).(*LoyaltyTier)
	return tier, args.Error(1)
}

func (m *mockLoyaltyRepository) GetAllTiers(ctx context.Context) ([]*LoyaltyTier, error) {
	args := m.Called(ctx)
	tiers, _ := args.Get(0).([]*LoyaltyTier)
	return tiers, args.Error(1)
}

func (m *mockLoyaltyRepository) CreatePointsTransaction(ctx context.Context, tx *PointsTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) GetPointsHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*PointsTransaction, int, error) {
	args := m.Called(ctx, riderID, limit, offset)
	txs, _ := args.Get(0).([]*PointsTransaction)
	return txs, args.Int(1), args.Error(2)
}

func (m *mockLoyaltyRepository) GetReward(ctx context.Context, rewardID uuid.UUID) (*RewardCatalogItem, error) {
	args := m.Called(ctx, rewardID)
	reward, _ := args.Get(0).(*RewardCatalogItem)
	return reward, args.Error(1)
}

func (m *mockLoyaltyRepository) GetAvailableRewards(ctx context.Context, tierID *uuid.UUID) ([]*RewardCatalogItem, error) {
	args := m.Called(ctx, tierID)
	rewards, _ := args.Get(0).([]*RewardCatalogItem)
	return rewards, args.Error(1)
}

func (m *mockLoyaltyRepository) GetUserRedemptionCount(ctx context.Context, riderID, rewardID uuid.UUID) (int, error) {
	args := m.Called(ctx, riderID, rewardID)
	return args.Int(0), args.Error(1)
}

func (m *mockLoyaltyRepository) CreateRedemption(ctx context.Context, redemption *Redemption) error {
	args := m.Called(ctx, redemption)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) IncrementRewardRedemptionCount(ctx context.Context, rewardID uuid.UUID) error {
	args := m.Called(ctx, rewardID)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) GetActiveChallenges(ctx context.Context, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	args := m.Called(ctx, tierID)
	challenges, _ := args.Get(0).([]*RiderChallenge)
	return challenges, args.Error(1)
}

func (m *mockLoyaltyRepository) GetActiveChallengesByType(ctx context.Context, challengeType string, tierID *uuid.UUID) ([]*RiderChallenge, error) {
	args := m.Called(ctx, challengeType, tierID)
	challenges, _ := args.Get(0).([]*RiderChallenge)
	return challenges, args.Error(1)
}

func (m *mockLoyaltyRepository) GetChallengeProgress(ctx context.Context, riderID, challengeID uuid.UUID) (*ChallengeProgress, error) {
	args := m.Called(ctx, riderID, challengeID)
	progress, _ := args.Get(0).(*ChallengeProgress)
	return progress, args.Error(1)
}

func (m *mockLoyaltyRepository) CreateChallengeProgress(ctx context.Context, progress *ChallengeProgress) error {
	args := m.Called(ctx, progress)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) UpdateChallengeProgress(ctx context.Context, progressID uuid.UUID, currentValue int, completed bool) error {
	args := m.Called(ctx, progressID, currentValue, completed)
	return args.Error(0)
}

func (m *mockLoyaltyRepository) GetLoyaltyStats(ctx context.Context) (*LoyaltyStats, error) {
	args := m.Called(ctx)
	stats, _ := args.Get(0).(*LoyaltyStats)
	return stats, args.Error(1)
}

// ========================================
// TEST HELPER FUNCTIONS
// ========================================

func createBronzeTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                uuid.New(),
		Name:              TierBronze,
		DisplayName:       "Bronze",
		MinPoints:         0,
		Multiplier:        1.0,
		DiscountPercent:   0,
		PrioritySupport:   false,
		FreeCancellations: 1,
		FreeUpgrades:      0,
		Benefits:          []string{"Basic support"},
		IsActive:          true,
	}
}

func createSilverTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                uuid.New(),
		Name:              TierSilver,
		DisplayName:       "Silver",
		MinPoints:         1000,
		Multiplier:        1.25,
		DiscountPercent:   5,
		PrioritySupport:   false,
		FreeCancellations: 2,
		FreeUpgrades:      1,
		Benefits:          []string{"Priority queue", "5% discount"},
		IsActive:          true,
	}
}

func createGoldTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                uuid.New(),
		Name:              TierGold,
		DisplayName:       "Gold",
		MinPoints:         5000,
		Multiplier:        1.5,
		DiscountPercent:   10,
		PrioritySupport:   true,
		FreeCancellations: 3,
		FreeUpgrades:      2,
		Benefits:          []string{"Priority support", "10% discount", "Free upgrades"},
		IsActive:          true,
	}
}

func createPlatinumTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                uuid.New(),
		Name:              TierPlatinum,
		DisplayName:       "Platinum",
		MinPoints:         15000,
		Multiplier:        2.0,
		DiscountPercent:   15,
		PrioritySupport:   true,
		FreeCancellations: 5,
		FreeUpgrades:      3,
		Benefits:          []string{"Dedicated support", "15% discount", "Airport lounge"},
		IsActive:          true,
	}
}

func createTestAccount(riderID uuid.UUID, tier *LoyaltyTier) *RiderLoyalty {
	account := &RiderLoyalty{
		RiderID:               riderID,
		TotalPoints:           500,
		AvailablePoints:       500,
		LifetimePoints:        500,
		TierPoints:            500,
		TierPeriodStart:       time.Now().AddDate(-1, 0, 0),
		TierPeriodEnd:         time.Now().AddDate(1, 0, 0),
		FreeCancellationsUsed: 0,
		FreeUpgradesUsed:      0,
		StreakDays:            5,
		JoinedAt:              time.Now().AddDate(-1, 0, 0),
		CreatedAt:             time.Now().AddDate(-1, 0, 0),
		UpdatedAt:             time.Now(),
	}
	if tier != nil {
		account.CurrentTierID = &tier.ID
		account.CurrentTier = tier
	}
	return account
}

func createTestReward() *RewardCatalogItem {
	return &RewardCatalogItem{
		ID:             uuid.New(),
		Name:           "Free Ride",
		RewardType:     "ride_credit",
		PointsRequired: 500,
		ValidDays:      30,
		IsActive:       true,
		CreatedAt:      time.Now(),
	}
}

func createTestChallenge() *RiderChallenge {
	return &RiderChallenge{
		ID:            uuid.New(),
		Name:          "Weekly Rider",
		ChallengeType: "rides",
		TargetValue:   5,
		RewardPoints:  100,
		StartDate:     time.Now().AddDate(0, 0, -1),
		EndDate:       time.Now().AddDate(0, 0, 6),
		IsActive:      true,
		CreatedAt:     time.Now(),
	}
}

// ========================================
// GetOrCreateLoyaltyAccount TESTS
// ========================================

func TestGetOrCreateLoyaltyAccount_ExistingAccount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	existingAccount := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(existingAccount, nil).Once()

	account, err := service.GetOrCreateLoyaltyAccount(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, existingAccount, account)
	repo.AssertExpectations(t)
}

func TestGetOrCreateLoyaltyAccount_NewAccount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()
	repo.On("GetTierByName", ctx, TierBronze).Return(bronzeTier, nil).Once()
	repo.On("CreateRiderLoyalty", ctx, mock.MatchedBy(func(account *RiderLoyalty) bool {
		return account.RiderID == riderID &&
			*account.CurrentTierID == bronzeTier.ID &&
			account.TotalPoints == 0 &&
			account.AvailablePoints == 0
	})).Return(nil).Once()

	// For the goroutine that awards signup bonus - use Maybe() since it's async
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(createTestAccount(riderID, bronzeTier), nil).Maybe()
	repo.On("CreatePointsTransaction", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("UpdatePoints", mock.Anything, riderID, mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{bronzeTier}, nil).Maybe()

	account, err := service.GetOrCreateLoyaltyAccount(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, riderID, account.RiderID)
	assert.Equal(t, bronzeTier.ID, *account.CurrentTierID)
	assert.Equal(t, bronzeTier, account.CurrentTier)

	// Give goroutine time to complete
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestGetOrCreateLoyaltyAccount_GetTierError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()
	repo.On("GetTierByName", ctx, TierBronze).Return((*LoyaltyTier)(nil), errors.New("tier not found")).Once()

	account, err := service.GetOrCreateLoyaltyAccount(ctx, riderID)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

// ========================================
// EarnPoints TESTS
// ========================================

func TestEarnPoints_BasicPoints(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	account := createTestAccount(riderID, bronzeTier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.RiderID == riderID &&
			tx.TransactionType == TransactionEarn &&
			tx.Points == 100 && // Bronze multiplier is 1.0
			tx.Source == SourceRide
	})).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 100, 100).Return(nil).Once()

	// For async tier upgrade check
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{bronzeTier}, nil).Maybe()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID:     riderID,
		Points:      100,
		Source:      SourceRide,
		Description: "Completed ride",
	})

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestEarnPoints_WithTierMultipliers(t *testing.T) {
	testCases := []struct {
		name           string
		tier           *LoyaltyTier
		basePoints     int
		expectedPoints int
	}{
		{
			name:           "Bronze tier (1.0x)",
			tier:           createBronzeTier(),
			basePoints:     100,
			expectedPoints: 100,
		},
		{
			name:           "Silver tier (1.25x)",
			tier:           createSilverTier(),
			basePoints:     100,
			expectedPoints: 125,
		},
		{
			name:           "Gold tier (1.5x)",
			tier:           createGoldTier(),
			basePoints:     100,
			expectedPoints: 150,
		},
		{
			name:           "Platinum tier (2.0x)",
			tier:           createPlatinumTier(),
			basePoints:     100,
			expectedPoints: 200,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			account := createTestAccount(riderID, tc.tier)

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
				return tx.Points == tc.expectedPoints
			})).Return(nil).Once()
			repo.On("UpdatePoints", ctx, riderID, tc.expectedPoints, tc.expectedPoints).Return(nil).Once()

			// For async tier upgrade
			repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
			repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tc.tier}, nil).Maybe()

			err := service.EarnPoints(ctx, &EarnPointsRequest{
				RiderID: riderID,
				Points:  tc.basePoints,
				Source:  SourceRide,
			})

			require.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			repo.AssertExpectations(t)
		})
	}
}

func TestEarnPoints_ZeroPoints(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  0,
		Source:  SourceRide,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "points must be positive")
	repo.AssertNotCalled(t, "GetRiderLoyalty")
}

func TestEarnPoints_NegativePoints(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  -50,
		Source:  SourceRide,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "points must be positive")
}

func TestEarnPoints_TransactionCreationFails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(errors.New("database error")).Once()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  100,
		Source:  SourceRide,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

func TestEarnPoints_FromDifferentSources(t *testing.T) {
	sources := []PointSource{
		SourceRide,
		SourceReferral,
		SourcePromo,
		SourceChallenge,
		SourceBirthday,
		SourceStreak,
		SourceSignup,
	}

	for _, source := range sources {
		t.Run(string(source), func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			tier := createBronzeTier()
			account := createTestAccount(riderID, tier)

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
				return tx.Source == source
			})).Return(nil).Once()
			repo.On("UpdatePoints", ctx, riderID, mock.Anything, mock.Anything).Return(nil).Once()

			// For async tier upgrade
			repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
			repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

			err := service.EarnPoints(ctx, &EarnPointsRequest{
				RiderID: riderID,
				Points:  50,
				Source:  source,
			})

			require.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// RedeemPoints TESTS
// ========================================

func TestRedeemPoints_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	// Note: reward.MaxRedemptionsPerUser is nil so GetUserRedemptionCount won't be called

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("CreateRedemption", ctx, mock.MatchedBy(func(redemption *Redemption) bool {
		return redemption.RiderID == riderID &&
			redemption.RewardID == reward.ID &&
			redemption.PointsSpent == reward.PointsRequired &&
			redemption.Status == "active"
	})).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.TransactionType == TransactionRedeem &&
			tx.Points == -reward.PointsRequired
	})).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, reward.PointsRequired).Return(nil).Once()
	repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, reward.PointsRequired, response.PointsSpent)
	assert.Equal(t, account.AvailablePoints-reward.PointsRequired, response.BalanceAfter)
	assert.NotEmpty(t, response.RedemptionCode)
	assert.Contains(t, response.RedemptionCode, "RDM-")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 100
	reward := createTestReward()
	reward.PointsRequired = 500

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "insufficient points")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_InactiveReward(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	reward.IsActive = false

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "no longer available")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_MaxRedemptionLimitReached(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	maxRedemptions := 2
	reward.MaxRedemptionsPerUser = &maxRedemptions

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("GetUserRedemptionCount", ctx, riderID, reward.ID).Return(2, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "maximum redemptions")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_TierRestricted(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	goldTier := createGoldTier()
	account := createTestAccount(riderID, bronzeTier)
	account.AvailablePoints = 5000
	reward := createTestReward()
	reward.TierRestriction = &goldTier.ID

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("GetTier", ctx, bronzeTier.ID).Return(bronzeTier, nil).Once()
	repo.On("GetTier", ctx, goldTier.ID).Return(goldTier, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_AccountNotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	rewardID := uuid.New()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: rewardID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_RewardNotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	rewardID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, rewardID).Return((*RewardCatalogItem)(nil), errors.New("not found")).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: rewardID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

// ========================================
// checkTierUpgrade TESTS
// ========================================

func TestCheckTierUpgrade_NoUpgrade(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	account := createTestAccount(riderID, bronzeTier)
	account.TierPoints = 500 // Below silver threshold

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Once()

	err := service.checkTierUpgrade(ctx, riderID)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "UpdateTier")
	repo.AssertExpectations(t)
}

func TestCheckTierUpgrade_UpgradeToBronzeToSilver(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	account := createTestAccount(riderID, bronzeTier)
	account.TierPoints = 1500 // Above silver threshold

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Once()
	repo.On("UpdateTier", ctx, riderID, silverTier.ID).Return(nil).Once()

	err := service.checkTierUpgrade(ctx, riderID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCheckTierUpgrade_SkipToGold(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	goldTier := createGoldTier()
	account := createTestAccount(riderID, bronzeTier)
	account.TierPoints = 6000 // Above gold threshold

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier, goldTier}, nil).Once()
	repo.On("UpdateTier", ctx, riderID, goldTier.ID).Return(nil).Once()

	err := service.checkTierUpgrade(ctx, riderID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCheckTierUpgrade_AlreadyAtTier(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	account := createTestAccount(riderID, silverTier)
	account.TierPoints = 1500 // At silver threshold but already silver

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Once()

	err := service.checkTierUpgrade(ctx, riderID)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "UpdateTier")
	repo.AssertExpectations(t)
}

func TestCheckTierUpgrade_AccountNotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()

	err := service.checkTierUpgrade(ctx, riderID)

	require.Error(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// UpdateChallengeProgress TESTS
// ========================================

func TestUpdateChallengeProgress_NewProgress(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return((*ChallengeProgress)(nil), errors.New("not found")).Once()
	repo.On("CreateChallengeProgress", ctx, mock.MatchedBy(func(p *ChallengeProgress) bool {
		return p.RiderID == riderID && p.ChallengeID == challenge.ID
	})).Return(nil).Once()
	repo.On("UpdateChallengeProgress", ctx, mock.Anything, 1, false).Return(nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_ExistingProgress(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 2,
		Completed:    false,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress.ID, 3, false).Return(nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_CompletesChallenge(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	challenge.TargetValue = 5
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 4,
		Completed:    false,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress.ID, 5, true).Return(nil).Once()

	// EarnPoints for challenge completion
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.Source == SourceChallenge && tx.Points == challenge.RewardPoints
	})).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, challenge.RewardPoints, challenge.RewardPoints).Return(nil).Once()

	// For async tier upgrade in EarnPoints
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_AlreadyCompleted(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 5,
		Completed:    true,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "UpdateChallengeProgress")
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_NoAccount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err) // Returns nil when no account
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_MultipleChallenges(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge1 := createTestChallenge()
	challenge1.Name = "Weekly Rider"
	challenge2 := createTestChallenge()
	challenge2.ID = uuid.New()
	challenge2.Name = "Monthly Rider"
	challenge2.TargetValue = 20

	progress1 := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge1.ID,
		CurrentValue: 1,
		Completed:    false,
	}
	progress2 := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge2.ID,
		CurrentValue: 5,
		Completed:    false,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge1, challenge2}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge1.ID).Return(progress1, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge2.ID).Return(progress2, nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress1.ID, 2, false).Return(nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress2.ID, 6, false).Return(nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 1)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// GetPointsHistory TESTS
// ========================================

func TestGetPointsHistory_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	transactions := []*PointsTransaction{
		{
			ID:              uuid.New(),
			RiderID:         riderID,
			TransactionType: TransactionEarn,
			Points:          100,
			BalanceAfter:    500,
			Source:          SourceRide,
			CreatedAt:       time.Now(),
		},
		{
			ID:              uuid.New(),
			RiderID:         riderID,
			TransactionType: TransactionRedeem,
			Points:          -50,
			BalanceAfter:    450,
			Source:          PointSource("redemption"),
			CreatedAt:       time.Now().Add(-time.Hour),
		},
	}

	repo.On("GetPointsHistory", ctx, riderID, 20, 0).Return(transactions, 2, nil).Once()

	response, err := service.GetPointsHistory(ctx, riderID, 20, 0)

	require.NoError(t, err)
	assert.Len(t, response.Transactions, 2)
	assert.Equal(t, 2, response.Total)
	assert.Equal(t, 20, response.Limit)
	assert.Equal(t, 0, response.Offset)
	repo.AssertExpectations(t)
}

func TestGetPointsHistory_Pagination(t *testing.T) {
	testCases := []struct {
		name          string
		limit         int
		offset        int
		expectedLimit int
		expectedOffset int
	}{
		{
			name:           "Valid limit and offset",
			limit:          10,
			offset:         0,
			expectedLimit:  10,
			expectedOffset: 0,
		},
		{
			name:           "Offset skips first page",
			limit:          10,
			offset:         10,
			expectedLimit:  10,
			expectedOffset: 10,
		},
		{
			name:           "Invalid limit (zero) defaults to 20",
			limit:          0,
			offset:         0,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "Limit exceeds max defaults to 20",
			limit:          200,
			offset:         0,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "Negative limit defaults to 20",
			limit:          -1,
			offset:         0,
			expectedLimit:  20,
			expectedOffset: 0,
		},
		{
			name:           "Negative offset defaults to 0",
			limit:          10,
			offset:         -5,
			expectedLimit:  10,
			expectedOffset: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()

			repo.On("GetPointsHistory", ctx, riderID, tc.expectedLimit, tc.expectedOffset).Return([]*PointsTransaction{}, 0, nil).Once()

			_, err := service.GetPointsHistory(ctx, riderID, tc.limit, tc.offset)

			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestGetPointsHistory_RepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	repo.On("GetPointsHistory", ctx, riderID, 20, 0).Return(([]*PointsTransaction)(nil), 0, errors.New("database error")).Once()

	response, err := service.GetPointsHistory(ctx, riderID, 20, 0)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

func TestGetPointsHistory_EmptyHistory(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	repo.On("GetPointsHistory", ctx, riderID, 20, 0).Return([]*PointsTransaction{}, 0, nil).Once()

	response, err := service.GetPointsHistory(ctx, riderID, 20, 0)

	require.NoError(t, err)
	assert.Empty(t, response.Transactions)
	assert.Equal(t, 0, response.Total)
	repo.AssertExpectations(t)
}

// ========================================
// GetLoyaltyStatus TESTS
// ========================================

func TestGetLoyaltyStatus_WithNextTier(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	account := createTestAccount(riderID, bronzeTier)
	account.TierPoints = 500

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetTier", ctx, bronzeTier.ID).Return(bronzeTier, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Once()

	status, err := service.GetLoyaltyStatus(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, riderID, status.RiderID)
	assert.Equal(t, bronzeTier, status.CurrentTier)
	assert.Equal(t, silverTier, status.NextTier)
	assert.Equal(t, 500, status.PointsToNextTier) // 1000 - 500
	assert.InDelta(t, 50.0, status.TierProgress, 0.1)
	repo.AssertExpectations(t)
}

func TestGetLoyaltyStatus_AtMaxTier(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	platinumTier := createPlatinumTier()
	account := createTestAccount(riderID, platinumTier)
	account.TierPoints = 20000

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetTier", ctx, platinumTier.ID).Return(platinumTier, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, platinumTier}, nil).Once()

	status, err := service.GetLoyaltyStatus(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, platinumTier, status.CurrentTier)
	assert.Nil(t, status.NextTier)
	assert.Equal(t, 0, status.PointsToNextTier)
	assert.InDelta(t, 100.0, status.TierProgress, 0.1)
	repo.AssertExpectations(t)
}

func TestGetLoyaltyStatus_CalculatesBenefits(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	goldTier := createGoldTier()
	account := createTestAccount(riderID, goldTier)
	account.FreeCancellationsUsed = 1
	account.FreeUpgradesUsed = 1

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetTier", ctx, goldTier.ID).Return(goldTier, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{goldTier}, nil).Once()

	status, err := service.GetLoyaltyStatus(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, 2, status.FreeCancellations) // 3 - 1
	assert.Equal(t, 1, status.FreeUpgrades)      // 2 - 1
	assert.Equal(t, goldTier.Benefits, status.Benefits)
	repo.AssertExpectations(t)
}

// ========================================
// GetActiveChallenges TESTS
// ========================================

func TestGetActiveChallenges_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 3,
		Completed:    false,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallenges", ctx, account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()

	response, err := service.GetActiveChallenges(ctx, riderID)

	require.NoError(t, err)
	assert.Len(t, response.Challenges, 1)
	assert.Equal(t, challenge.Name, response.Challenges[0].Challenge.Name)
	assert.Equal(t, 3, response.Challenges[0].CurrentValue)
	assert.InDelta(t, 60.0, response.Challenges[0].ProgressPercent, 0.1)
	assert.False(t, response.Challenges[0].Completed)
	repo.AssertExpectations(t)
}

func TestGetActiveChallenges_NoProgress(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallenges", ctx, account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return((*ChallengeProgress)(nil), errors.New("not found")).Once()

	response, err := service.GetActiveChallenges(ctx, riderID)

	require.NoError(t, err)
	assert.Len(t, response.Challenges, 1)
	assert.Equal(t, 0, response.Challenges[0].CurrentValue)
	assert.False(t, response.Challenges[0].Completed)
	repo.AssertExpectations(t)
}

// ========================================
// GetRewardsCatalog TESTS
// ========================================

func TestGetRewardsCatalog_WithAccount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createGoldTier()
	account := createTestAccount(riderID, tier)
	rewards := []*RewardCatalogItem{createTestReward()}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAvailableRewards", ctx, account.CurrentTierID).Return(rewards, nil).Once()

	result, err := service.GetRewardsCatalog(ctx, riderID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

func TestGetRewardsCatalog_NoAccount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	rewards := []*RewardCatalogItem{createTestReward()}

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()
	repo.On("GetAvailableRewards", ctx, (*uuid.UUID)(nil)).Return(rewards, nil).Once()

	result, err := service.GetRewardsCatalog(ctx, riderID)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	repo.AssertExpectations(t)
}

// ========================================
// GetAllTiers TESTS
// ========================================

func TestGetAllTiers_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	tiers := []*LoyaltyTier{createBronzeTier(), createSilverTier(), createGoldTier()}

	repo.On("GetAllTiers", ctx).Return(tiers, nil).Once()

	result, err := service.GetAllTiers(ctx)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	repo.AssertExpectations(t)
}

func TestGetAllTiers_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)

	repo.On("GetAllTiers", ctx).Return(([]*LoyaltyTier)(nil), errors.New("database error")).Once()

	result, err := service.GetAllTiers(ctx)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// INTEGRATION-LIKE TESTS (Testing flows)
// ========================================

func TestEarnPointsTriggersTierUpgrade(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	account := createTestAccount(riderID, bronzeTier)
	account.TierPoints = 900 // Close to silver

	// Initial EarnPoints call
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 100, 100).Return(nil).Once()

	// Async tier upgrade check - account now has 1000 points
	accountAfterEarn := *account
	accountAfterEarn.TierPoints = 1000
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(&accountAfterEarn, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Maybe()
	repo.On("UpdateTier", mock.Anything, riderID, silverTier.ID).Return(nil).Maybe()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  100,
		Source:  SourceRide,
	})

	require.NoError(t, err)
	time.Sleep(100 * time.Millisecond) // Wait for async operations
	repo.AssertExpectations(t)
}

func TestCompleteChallengeThenRedeemReward(t *testing.T) {
	// This test demonstrates a complete user flow:
	// 1. User completes a challenge
	// 2. User receives points from challenge completion
	// 3. User redeems points for a reward

	ctx := context.Background()
	riderID := uuid.New()
	tier := createBronzeTier()
	challenge := createTestChallenge()
	challenge.TargetValue = 5
	challenge.RewardPoints = 100
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 4,
		Completed:    false,
	}
	reward := createTestReward()
	reward.PointsRequired = 500

	// Part 1: Complete challenge
	repo1 := new(mockLoyaltyRepository)
	service1 := NewService(repo1)
	account1 := createTestAccount(riderID, tier)
	account1.AvailablePoints = 400

	repo1.On("GetRiderLoyalty", ctx, riderID).Return(account1, nil).Once()
	repo1.On("GetActiveChallengesByType", ctx, "rides", account1.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo1.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()
	repo1.On("UpdateChallengeProgress", ctx, progress.ID, 5, true).Return(nil).Once()

	// EarnPoints from challenge completion
	repo1.On("GetRiderLoyalty", ctx, riderID).Return(account1, nil).Once()
	repo1.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.Source == SourceChallenge && tx.Points == 100
	})).Return(nil).Once()
	repo1.On("UpdatePoints", ctx, riderID, 100, 100).Return(nil).Once()

	// Async tier check
	repo1.On("GetRiderLoyalty", mock.Anything, riderID).Return(account1, nil).Maybe()
	repo1.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

	err := service1.UpdateChallengeProgress(ctx, riderID, "rides", 1)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo1.AssertExpectations(t)

	// Part 2: Redeem reward with new balance (simulating DB has updated)
	repo2 := new(mockLoyaltyRepository)
	service2 := NewService(repo2)
	accountWithBonus := createTestAccount(riderID, tier)
	accountWithBonus.AvailablePoints = 500 // Now has enough points

	repo2.On("GetRiderLoyalty", ctx, riderID).Return(accountWithBonus, nil).Once()
	repo2.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo2.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo2.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.TransactionType == TransactionRedeem
	})).Return(nil).Once()
	repo2.On("DeductPoints", ctx, riderID, 500).Return(nil).Once()
	repo2.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service2.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 500, response.PointsSpent)
	assert.Equal(t, 0, response.BalanceAfter)
	repo2.AssertExpectations(t)
}

// ========================================
// ADDITIONAL EDGE CASE TESTS
// ========================================

// Create Diamond tier for additional tests
func createDiamondTier() *LoyaltyTier {
	return &LoyaltyTier{
		ID:                  uuid.New(),
		Name:                TierDiamond,
		DisplayName:         "Diamond",
		MinPoints:           50000,
		Multiplier:          2.5,
		DiscountPercent:     25,
		PrioritySupport:     true,
		FreeCancellations:   10,
		FreeUpgrades:        5,
		AirportLoungeAccess: true,
		DedicatedSupport:    true,
		Benefits:            []string{"VIP support", "25% discount", "Airport lounge", "Personal concierge"},
		IsActive:            true,
	}
}

// ========================================
// TIER MULTIPLIER PRECISION TESTS
// ========================================

func TestEarnPoints_MultiplierPrecision(t *testing.T) {
	// Financial accuracy tests - points are like currency
	testCases := []struct {
		name           string
		tier           *LoyaltyTier
		basePoints     int
		expectedPoints int
		description    string
	}{
		{
			name:           "Bronze 1.0x - exact",
			tier:           createBronzeTier(),
			basePoints:     100,
			expectedPoints: 100,
			description:    "Bronze should earn exactly base points",
		},
		{
			name:           "Silver 1.25x - exact calculation",
			tier:           createSilverTier(),
			basePoints:     100,
			expectedPoints: 125,
			description:    "100 * 1.25 = 125",
		},
		{
			name:           "Silver 1.25x - truncation behavior",
			tier:           createSilverTier(),
			basePoints:     1,
			expectedPoints: 1, // 1 * 1.25 = 1.25 -> truncates to 1
			description:    "Should truncate fractional points",
		},
		{
			name:           "Silver 1.25x - odd number",
			tier:           createSilverTier(),
			basePoints:     3,
			expectedPoints: 3, // 3 * 1.25 = 3.75 -> truncates to 3
			description:    "3 * 1.25 = 3.75 truncates to 3",
		},
		{
			name:           "Gold 1.5x - exact",
			tier:           createGoldTier(),
			basePoints:     100,
			expectedPoints: 150,
			description:    "100 * 1.5 = 150",
		},
		{
			name:           "Gold 1.5x - odd base",
			tier:           createGoldTier(),
			basePoints:     1,
			expectedPoints: 1, // 1 * 1.5 = 1.5 -> truncates to 1
			description:    "1 * 1.5 = 1.5 truncates to 1",
		},
		{
			name:           "Gold 1.5x - larger odd",
			tier:           createGoldTier(),
			basePoints:     33,
			expectedPoints: 49, // 33 * 1.5 = 49.5 -> truncates to 49
			description:    "33 * 1.5 = 49.5 truncates to 49",
		},
		{
			name:           "Platinum 2.0x - exact double",
			tier:           createPlatinumTier(),
			basePoints:     100,
			expectedPoints: 200,
			description:    "100 * 2.0 = 200",
		},
		{
			name:           "Platinum 2.0x - small amount",
			tier:           createPlatinumTier(),
			basePoints:     1,
			expectedPoints: 2, // 1 * 2.0 = 2
			description:    "1 * 2.0 = 2",
		},
		{
			name:           "Diamond 2.5x - exact",
			tier:           createDiamondTier(),
			basePoints:     100,
			expectedPoints: 250,
			description:    "100 * 2.5 = 250",
		},
		{
			name:           "Diamond 2.5x - truncation",
			tier:           createDiamondTier(),
			basePoints:     1,
			expectedPoints: 2, // 1 * 2.5 = 2.5 -> truncates to 2
			description:    "1 * 2.5 = 2.5 truncates to 2",
		},
		{
			name:           "Large point amount - Silver",
			tier:           createSilverTier(),
			basePoints:     10000,
			expectedPoints: 12500,
			description:    "10000 * 1.25 = 12500",
		},
		{
			name:           "Large point amount - Diamond",
			tier:           createDiamondTier(),
			basePoints:     10000,
			expectedPoints: 25000,
			description:    "10000 * 2.5 = 25000",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			account := createTestAccount(riderID, tc.tier)

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
				// Strict validation of points calculation
				return tx.Points == tc.expectedPoints
			})).Return(nil).Once()
			repo.On("UpdatePoints", ctx, riderID, tc.expectedPoints, tc.expectedPoints).Return(nil).Once()

			// For async tier upgrade
			repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
			repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tc.tier}, nil).Maybe()

			err := service.EarnPoints(ctx, &EarnPointsRequest{
				RiderID: riderID,
				Points:  tc.basePoints,
				Source:  SourceRide,
			})

			require.NoError(t, err, tc.description)
			time.Sleep(50 * time.Millisecond)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// TIER BOUNDARY TESTS
// ========================================

func TestCheckTierUpgrade_ExactBoundaries(t *testing.T) {
	testCases := []struct {
		name           string
		tierPoints     int
		currentTier    *LoyaltyTier
		allTiers       []*LoyaltyTier
		expectedTierID *uuid.UUID
		shouldUpgrade  bool
	}{
		{
			name:       "Exactly at Silver threshold (1000 points)",
			tierPoints: 1000,
		},
		{
			name:       "One point below Silver threshold (999 points)",
			tierPoints: 999,
		},
		{
			name:       "One point above Silver threshold (1001 points)",
			tierPoints: 1001,
		},
		{
			name:       "Exactly at Gold threshold (5000 points)",
			tierPoints: 5000,
		},
		{
			name:       "Exactly at Platinum threshold (15000 points)",
			tierPoints: 15000,
		},
	}

	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()
	goldTier := createGoldTier()
	platinumTier := createPlatinumTier()
	allTiers := []*LoyaltyTier{bronzeTier, silverTier, goldTier, platinumTier}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			account := createTestAccount(riderID, bronzeTier)
			account.TierPoints = tc.tierPoints

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("GetAllTiers", ctx).Return(allTiers, nil).Once()

			// Determine expected tier based on tier points
			var expectedTier *LoyaltyTier
			for _, t := range allTiers {
				if account.TierPoints >= t.MinPoints {
					expectedTier = t
				}
			}

			// Only expect UpdateTier if tier changes
			if expectedTier != nil && expectedTier.ID != bronzeTier.ID {
				repo.On("UpdateTier", ctx, riderID, expectedTier.ID).Return(nil).Once()
			}

			err := service.checkTierUpgrade(ctx, riderID)

			require.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestCheckTierUpgrade_AllTierTransitions(t *testing.T) {
	testCases := []struct {
		name         string
		currentTier  func() *LoyaltyTier
		tierPoints   int
		expectedTier func() *LoyaltyTier
	}{
		{
			name:         "Bronze to Silver",
			currentTier:  createBronzeTier,
			tierPoints:   1000,
			expectedTier: createSilverTier,
		},
		{
			name:         "Silver to Gold",
			currentTier:  createSilverTier,
			tierPoints:   5000,
			expectedTier: createGoldTier,
		},
		{
			name:         "Gold to Platinum",
			currentTier:  createGoldTier,
			tierPoints:   15000,
			expectedTier: createPlatinumTier,
		},
		{
			name:         "Bronze directly to Gold (skip Silver)",
			currentTier:  createBronzeTier,
			tierPoints:   5000,
			expectedTier: createGoldTier,
		},
		{
			name:         "Bronze directly to Platinum (skip Silver and Gold)",
			currentTier:  createBronzeTier,
			tierPoints:   15000,
			expectedTier: createPlatinumTier,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			currentTier := tc.currentTier()
			expectedTier := tc.expectedTier()
			account := createTestAccount(riderID, currentTier)
			account.TierPoints = tc.tierPoints

			bronzeTier := createBronzeTier()
			silverTier := createSilverTier()
			goldTier := createGoldTier()
			platinumTier := createPlatinumTier()

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier, goldTier, platinumTier}, nil).Once()
			repo.On("UpdateTier", ctx, riderID, mock.MatchedBy(func(tierID uuid.UUID) bool {
				// Verify upgrading to the correct tier level
				return tierID != currentTier.ID
			})).Return(nil).Once()

			err := service.checkTierUpgrade(ctx, riderID)

			require.NoError(t, err)
			repo.AssertExpectations(t)

			// Verify expected tier is at least at the expected level
			assert.GreaterOrEqual(t, expectedTier.MinPoints, currentTier.MinPoints)
		})
	}
}

// ========================================
// REDEMPTION EDGE CASES
// ========================================

func TestRedeemPoints_ExactPointsBalance(t *testing.T) {
	// Test redeeming when user has EXACTLY enough points
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 500 // Exactly enough
	reward := createTestReward()
	reward.PointsRequired = 500

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.Points == -500 && tx.BalanceAfter == 0
	})).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, 500).Return(nil).Once()
	repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.Equal(t, 0, response.BalanceAfter)
	assert.Equal(t, 500, response.PointsSpent)
	repo.AssertExpectations(t)
}

func TestRedeemPoints_OnePointShort(t *testing.T) {
	// Test redeeming when user is exactly 1 point short
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 499 // 1 point short
	reward := createTestReward()
	reward.PointsRequired = 500

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "insufficient points")
	assert.Contains(t, err.Error(), "need 500")
	assert.Contains(t, err.Error(), "have 499")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_ZeroPointsBalance(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 0 // Zero balance
	reward := createTestReward()
	reward.PointsRequired = 100

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "insufficient points")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_TierRestriction_SameTier(t *testing.T) {
	// User at Gold tier trying to redeem Gold-restricted reward
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	goldTier := createGoldTier()
	account := createTestAccount(riderID, goldTier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	reward.TierRestriction = &goldTier.ID

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("GetTier", ctx, goldTier.ID).Return(goldTier, nil).Twice() // Called for both current and restricted tier

	// Should succeed - user is at required tier
	repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, reward.PointsRequired).Return(nil).Once()
	repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.NotNil(t, response)
	repo.AssertExpectations(t)
}

func TestRedeemPoints_TierRestriction_HigherTier(t *testing.T) {
	// User at Platinum tier trying to redeem Gold-restricted reward (should succeed)
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	goldTier := createGoldTier()
	platinumTier := createPlatinumTier()
	account := createTestAccount(riderID, platinumTier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	reward.TierRestriction = &goldTier.ID

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("GetTier", ctx, platinumTier.ID).Return(platinumTier, nil).Once()
	repo.On("GetTier", ctx, goldTier.ID).Return(goldTier, nil).Once()

	// Should succeed - Platinum (15000 min points) > Gold (5000 min points)
	repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, reward.PointsRequired).Return(nil).Once()
	repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.NotNil(t, response)
	repo.AssertExpectations(t)
}

func TestRedeemPoints_MaxRedemptionsNotReached(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()
	maxRedemptions := 3
	reward.MaxRedemptionsPerUser = &maxRedemptions

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("GetUserRedemptionCount", ctx, riderID, reward.ID).Return(2, nil).Once() // 2 < 3

	repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, reward.PointsRequired).Return(nil).Once()
	repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.NoError(t, err)
	assert.NotNil(t, response)
	repo.AssertExpectations(t)
}

func TestRedeemPoints_CreateRedemptionFails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("CreateRedemption", ctx, mock.Anything).Return(errors.New("database error")).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

func TestRedeemPoints_DeductPointsFails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	account.AvailablePoints = 1000
	reward := createTestReward()

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
	repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("DeductPoints", ctx, riderID, reward.PointsRequired).Return(errors.New("concurrent modification")).Once()

	response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
		RiderID:  riderID,
		RewardID: reward.ID,
	})

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

// ========================================
// BENEFITS CALCULATION EDGE CASES
// ========================================

func TestGetLoyaltyStatus_BenefitsExhausted(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	goldTier := createGoldTier()
	account := createTestAccount(riderID, goldTier)
	// Gold tier has 3 free cancellations and 2 free upgrades
	account.FreeCancellationsUsed = 5 // More than available
	account.FreeUpgradesUsed = 3      // More than available

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetTier", ctx, goldTier.ID).Return(goldTier, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{goldTier}, nil).Once()

	status, err := service.GetLoyaltyStatus(ctx, riderID)

	require.NoError(t, err)
	assert.Equal(t, 0, status.FreeCancellations) // Should not go negative
	assert.Equal(t, 0, status.FreeUpgrades)      // Should not go negative
	repo.AssertExpectations(t)
}

func TestGetLoyaltyStatus_NoCurrentTier(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()
	silverTier := createSilverTier()

	// Account with nil CurrentTierID and 500 tier points
	// Since Bronze min=0, Silver min=1000, with 500 points the next tier is Silver
	account := &RiderLoyalty{
		RiderID:         riderID,
		CurrentTierID:   nil,
		TotalPoints:     500,
		AvailablePoints: 500,
		LifetimePoints:  500,
		TierPoints:      500,
		TierPeriodStart: time.Now().AddDate(-1, 0, 0),
		TierPeriodEnd:   time.Now().AddDate(1, 0, 0),
		StreakDays:      0,
		JoinedAt:        time.Now(),
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{bronzeTier, silverTier}, nil).Once()

	status, err := service.GetLoyaltyStatus(ctx, riderID)

	require.NoError(t, err)
	assert.Nil(t, status.CurrentTier)
	// Next tier is Silver since tierPoints(500) < silverTier.MinPoints(1000)
	// The logic finds first tier where MinPoints > TierPoints
	assert.Equal(t, silverTier, status.NextTier)
	assert.Empty(t, status.Benefits)
	repo.AssertExpectations(t)
}

func TestGetLoyaltyStatus_TierProgressCalculation(t *testing.T) {
	testCases := []struct {
		name             string
		tierPoints       int
		currentTierMin   int
		nextTierMin      int
		expectedProgress float64
	}{
		{
			name:             "0% progress (at tier minimum)",
			tierPoints:       0,
			currentTierMin:   0,
			nextTierMin:      1000,
			expectedProgress: 0.0,
		},
		{
			name:             "50% progress",
			tierPoints:       500,
			currentTierMin:   0,
			nextTierMin:      1000,
			expectedProgress: 50.0,
		},
		{
			name:             "99% progress",
			tierPoints:       999,
			currentTierMin:   0,
			nextTierMin:      1000,
			expectedProgress: 99.9,
		},
		{
			name:             "Progress within Gold tier (5000-15000)",
			tierPoints:       10000,
			currentTierMin:   5000,
			nextTierMin:      15000,
			expectedProgress: 50.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()

			currentTier := &LoyaltyTier{
				ID:        uuid.New(),
				Name:      TierBronze,
				MinPoints: tc.currentTierMin,
				Benefits:  []string{},
			}
			nextTier := &LoyaltyTier{
				ID:        uuid.New(),
				Name:      TierSilver,
				MinPoints: tc.nextTierMin,
			}

			account := createTestAccount(riderID, currentTier)
			account.TierPoints = tc.tierPoints

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("GetTier", ctx, currentTier.ID).Return(currentTier, nil).Once()
			repo.On("GetAllTiers", ctx).Return([]*LoyaltyTier{currentTier, nextTier}, nil).Once()

			status, err := service.GetLoyaltyStatus(ctx, riderID)

			require.NoError(t, err)
			assert.InDelta(t, tc.expectedProgress, status.TierProgress, 0.5)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// CHALLENGE PROGRESS EDGE CASES
// ========================================

func TestUpdateChallengeProgress_LargeIncrement(t *testing.T) {
	// Test when increment completes challenge in one go
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	challenge.TargetValue = 10
	challenge.RewardPoints = 500

	// Starting from 0, incrementing by 10 should complete immediately
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return((*ChallengeProgress)(nil), errors.New("not found")).Once()
	repo.On("CreateChallengeProgress", ctx, mock.Anything).Return(nil).Once()
	repo.On("UpdateChallengeProgress", ctx, mock.Anything, 10, true).Return(nil).Once()

	// EarnPoints for challenge completion
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.Source == SourceChallenge && tx.Points == 500
	})).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 500, 500).Return(nil).Once()

	// Async tier check
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 10)

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_ExceedsTarget(t *testing.T) {
	// Test when increment exceeds target (should still complete)
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	challenge.TargetValue = 5
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 4,
		Completed:    false,
	}

	// Incrementing by 3 when 1 needed - should result in 7 (4+3)
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress.ID, 7, true).Return(nil).Once()

	// EarnPoints for challenge completion
	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, mock.Anything, mock.Anything).Return(nil).Once()

	// Async tier check
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 3)

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_ZeroIncrement(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	challenge.TargetValue = 5
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 3,
		Completed:    false,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "rides", account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()
	repo.On("UpdateChallengeProgress", ctx, progress.ID, 3, false).Return(nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "rides", 0)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestUpdateChallengeProgress_NoChallengesOfType(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallengesByType", ctx, "spending", account.CurrentTierID).Return([]*RiderChallenge{}, nil).Once()

	err := service.UpdateChallengeProgress(ctx, riderID, "spending", 100)

	require.NoError(t, err)
	repo.AssertNotCalled(t, "GetChallengeProgress")
	repo.AssertExpectations(t)
}

// ========================================
// EARNPOINTS ADDITIONAL EDGE CASES
// ========================================

func TestEarnPoints_NoTierAssigned(t *testing.T) {
	// Test earning points when account has no tier (multiplier defaults to 1.0)
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()

	account := &RiderLoyalty{
		RiderID:         riderID,
		CurrentTierID:   nil,
		CurrentTier:     nil,
		AvailablePoints: 0,
		TierPoints:      0,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.Points == 100 // No multiplier applied (1.0x)
	})).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 100, 100).Return(nil).Once()

	// Async tier check
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{}, nil).Maybe()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  100,
		Source:  SourceRide,
	})

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

func TestEarnPoints_UpdatePointsFails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.Anything).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 100, 100).Return(errors.New("database error")).Once()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID: riderID,
		Points:  100,
		Source:  SourceRide,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

func TestEarnPoints_WithSourceID(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	rideID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
		return tx.SourceID != nil && *tx.SourceID == rideID
	})).Return(nil).Once()
	repo.On("UpdatePoints", ctx, riderID, 100, 100).Return(nil).Once()

	// Async tier check
	repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
	repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

	err := service.EarnPoints(ctx, &EarnPointsRequest{
		RiderID:     riderID,
		Points:      100,
		Source:      SourceRide,
		SourceID:    &rideID,
		Description: "Completed ride #12345",
	})

	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	repo.AssertExpectations(t)
}

// ========================================
// ACTIVE CHALLENGES ADDITIONAL TESTS
// ========================================

func TestGetActiveChallenges_ProgressExceedsTarget(t *testing.T) {
	// Test that progress percent caps at 100%
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)
	challenge := createTestChallenge()
	challenge.TargetValue = 5
	progress := &ChallengeProgress{
		ID:           uuid.New(),
		RiderID:      riderID,
		ChallengeID:  challenge.ID,
		CurrentValue: 10, // Exceeds target
		Completed:    true,
	}

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallenges", ctx, account.CurrentTierID).Return([]*RiderChallenge{challenge}, nil).Once()
	repo.On("GetChallengeProgress", ctx, riderID, challenge.ID).Return(progress, nil).Once()

	response, err := service.GetActiveChallenges(ctx, riderID)

	require.NoError(t, err)
	assert.Len(t, response.Challenges, 1)
	assert.Equal(t, 10, response.Challenges[0].CurrentValue)
	assert.Equal(t, 100.0, response.Challenges[0].ProgressPercent) // Capped at 100%
	assert.True(t, response.Challenges[0].Completed)
	repo.AssertExpectations(t)
}

func TestGetActiveChallenges_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetActiveChallenges", ctx, account.CurrentTierID).Return(([]*RiderChallenge)(nil), errors.New("database error")).Once()

	response, err := service.GetActiveChallenges(ctx, riderID)

	require.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

// ========================================
// REWARDS CATALOG ADDITIONAL TESTS
// ========================================

func TestGetRewardsCatalog_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAvailableRewards", ctx, account.CurrentTierID).Return(([]*RewardCatalogItem)(nil), errors.New("database error")).Once()

	result, err := service.GetRewardsCatalog(ctx, riderID)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

func TestGetRewardsCatalog_EmptyList(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	tier := createBronzeTier()
	account := createTestAccount(riderID, tier)

	repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
	repo.On("GetAvailableRewards", ctx, account.CurrentTierID).Return([]*RewardCatalogItem{}, nil).Once()

	result, err := service.GetRewardsCatalog(ctx, riderID)

	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// NEW ACCOUNT CREATION EDGE CASES
// ========================================

func TestGetOrCreateLoyaltyAccount_CreateFails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockLoyaltyRepository)
	service := NewService(repo)
	riderID := uuid.New()
	bronzeTier := createBronzeTier()

	repo.On("GetRiderLoyalty", ctx, riderID).Return((*RiderLoyalty)(nil), errors.New("not found")).Once()
	repo.On("GetTierByName", ctx, TierBronze).Return(bronzeTier, nil).Once()
	repo.On("CreateRiderLoyalty", ctx, mock.Anything).Return(errors.New("database error")).Once()

	account, err := service.GetOrCreateLoyaltyAccount(ctx, riderID)

	require.Error(t, err)
	assert.Nil(t, account)
	assert.Contains(t, err.Error(), "internal server error")
	repo.AssertExpectations(t)
}

// ========================================
// FINANCIAL ACCURACY STRESS TESTS
// ========================================

func TestEarnPoints_LargePointValues(t *testing.T) {
	testCases := []struct {
		name           string
		basePoints     int
		multiplier     float64
		expectedPoints int
	}{
		{
			name:           "Max int32 safe - Bronze",
			basePoints:     1000000,
			multiplier:     1.0,
			expectedPoints: 1000000,
		},
		{
			name:           "Large points - Gold",
			basePoints:     1000000,
			multiplier:     1.5,
			expectedPoints: 1500000,
		},
		{
			name:           "Large points - Platinum",
			basePoints:     1000000,
			multiplier:     2.0,
			expectedPoints: 2000000,
		},
		{
			name:           "Large points - Diamond",
			basePoints:     1000000,
			multiplier:     2.5,
			expectedPoints: 2500000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()

			tier := &LoyaltyTier{
				ID:         uuid.New(),
				Name:       TierGold,
				Multiplier: tc.multiplier,
			}
			account := createTestAccount(riderID, tier)

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
				return tx.Points == tc.expectedPoints
			})).Return(nil).Once()
			repo.On("UpdatePoints", ctx, riderID, tc.expectedPoints, tc.expectedPoints).Return(nil).Once()

			repo.On("GetRiderLoyalty", mock.Anything, riderID).Return(account, nil).Maybe()
			repo.On("GetAllTiers", mock.Anything).Return([]*LoyaltyTier{tier}, nil).Maybe()

			err := service.EarnPoints(ctx, &EarnPointsRequest{
				RiderID: riderID,
				Points:  tc.basePoints,
				Source:  SourceRide,
			})

			require.NoError(t, err)
			time.Sleep(50 * time.Millisecond)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// CONCURRENT OPERATIONS SIMULATION
// ========================================

func TestRedeemPoints_VerifyBalanceAfterCalculation(t *testing.T) {
	// Verify that balance_after is calculated correctly for various scenarios
	testCases := []struct {
		name             string
		availablePoints  int
		pointsRequired   int
		expectedBalance  int
	}{
		{
			name:            "Standard redemption",
			availablePoints: 1000,
			pointsRequired:  300,
			expectedBalance: 700,
		},
		{
			name:            "Full balance redemption",
			availablePoints: 500,
			pointsRequired:  500,
			expectedBalance: 0,
		},
		{
			name:            "Large balance small redemption",
			availablePoints: 100000,
			pointsRequired:  100,
			expectedBalance: 99900,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockLoyaltyRepository)
			service := NewService(repo)
			riderID := uuid.New()
			tier := createBronzeTier()
			account := createTestAccount(riderID, tier)
			account.AvailablePoints = tc.availablePoints

			reward := createTestReward()
			reward.PointsRequired = tc.pointsRequired

			repo.On("GetRiderLoyalty", ctx, riderID).Return(account, nil).Once()
			repo.On("GetReward", ctx, reward.ID).Return(reward, nil).Once()
			repo.On("CreateRedemption", ctx, mock.Anything).Return(nil).Once()
			repo.On("CreatePointsTransaction", ctx, mock.MatchedBy(func(tx *PointsTransaction) bool {
				return tx.BalanceAfter == tc.expectedBalance && tx.Points == -tc.pointsRequired
			})).Return(nil).Once()
			repo.On("DeductPoints", ctx, riderID, tc.pointsRequired).Return(nil).Once()
			repo.On("IncrementRewardRedemptionCount", ctx, reward.ID).Return(nil).Once()

			response, err := service.RedeemPoints(ctx, &RedeemPointsRequest{
				RiderID:  riderID,
				RewardID: reward.ID,
			})

			require.NoError(t, err)
			assert.Equal(t, tc.expectedBalance, response.BalanceAfter)
			assert.Equal(t, tc.pointsRequired, response.PointsSpent)
			repo.AssertExpectations(t)
		})
	}
}
