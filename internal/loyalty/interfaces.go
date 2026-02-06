package loyalty

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for loyalty repository operations
type RepositoryInterface interface {
	// Rider Loyalty Account
	GetRiderLoyalty(ctx context.Context, riderID uuid.UUID) (*RiderLoyalty, error)
	CreateRiderLoyalty(ctx context.Context, account *RiderLoyalty) error
	UpdatePoints(ctx context.Context, riderID uuid.UUID, earnedPoints, tierPoints int) error
	DeductPoints(ctx context.Context, riderID uuid.UUID, points int) error
	UpdateTier(ctx context.Context, riderID uuid.UUID, tierID uuid.UUID) error
	UpdateStreak(ctx context.Context, riderID uuid.UUID, streakDays int) error

	// Loyalty Tiers
	GetTier(ctx context.Context, tierID uuid.UUID) (*LoyaltyTier, error)
	GetTierByName(ctx context.Context, name TierName) (*LoyaltyTier, error)
	GetAllTiers(ctx context.Context) ([]*LoyaltyTier, error)

	// Points Transactions
	CreatePointsTransaction(ctx context.Context, tx *PointsTransaction) error
	GetPointsHistory(ctx context.Context, riderID uuid.UUID, limit, offset int) ([]*PointsTransaction, int, error)

	// Rewards
	GetReward(ctx context.Context, rewardID uuid.UUID) (*RewardCatalogItem, error)
	GetAvailableRewards(ctx context.Context, tierID *uuid.UUID) ([]*RewardCatalogItem, error)
	GetUserRedemptionCount(ctx context.Context, riderID, rewardID uuid.UUID) (int, error)
	CreateRedemption(ctx context.Context, redemption *Redemption) error
	IncrementRewardRedemptionCount(ctx context.Context, rewardID uuid.UUID) error

	// Challenges
	GetActiveChallenges(ctx context.Context, tierID *uuid.UUID) ([]*RiderChallenge, error)
	GetActiveChallengesByType(ctx context.Context, challengeType string, tierID *uuid.UUID) ([]*RiderChallenge, error)
	GetChallengeProgress(ctx context.Context, riderID, challengeID uuid.UUID) (*ChallengeProgress, error)
	CreateChallengeProgress(ctx context.Context, progress *ChallengeProgress) error
	UpdateChallengeProgress(ctx context.Context, progressID uuid.UUID, currentValue int, completed bool) error

	// Admin
	GetLoyaltyStats(ctx context.Context) (*LoyaltyStats, error)
}
