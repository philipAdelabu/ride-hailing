package giftcards

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockRepository implements RepositoryInterface for testing
type mockRepository struct {
	mock.Mock
}

func (m *mockRepository) CreateCard(ctx context.Context, card *GiftCard) error {
	args := m.Called(ctx, card)
	return args.Error(0)
}

func (m *mockRepository) GetCardByCode(ctx context.Context, code string) (*GiftCard, error) {
	args := m.Called(ctx, code)
	card, _ := args.Get(0).(*GiftCard)
	return card, args.Error(1)
}

func (m *mockRepository) GetCardByID(ctx context.Context, id uuid.UUID) (*GiftCard, error) {
	args := m.Called(ctx, id)
	card, _ := args.Get(0).(*GiftCard)
	return card, args.Error(1)
}

func (m *mockRepository) RedeemCard(ctx context.Context, cardID, userID uuid.UUID) error {
	args := m.Called(ctx, cardID, userID)
	return args.Error(0)
}

func (m *mockRepository) DeductBalance(ctx context.Context, cardID uuid.UUID, amount float64) (bool, error) {
	args := m.Called(ctx, cardID, amount)
	return args.Bool(0), args.Error(1)
}

func (m *mockRepository) GetActiveCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	args := m.Called(ctx, userID)
	cards, _ := args.Get(0).([]GiftCard)
	return cards, args.Error(1)
}

func (m *mockRepository) GetPurchasedCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error) {
	args := m.Called(ctx, userID)
	cards, _ := args.Get(0).([]GiftCard)
	return cards, args.Error(1)
}

func (m *mockRepository) CreateTransaction(ctx context.Context, tx *GiftCardTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *mockRepository) GetTransactionsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]GiftCardTransaction, error) {
	args := m.Called(ctx, userID, limit)
	txns, _ := args.Get(0).([]GiftCardTransaction)
	return txns, args.Error(1)
}

func (m *mockRepository) GetTotalBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepository) ExpireCards(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

// ============================================================
// PurchaseCard Tests - FINANCIAL CRITICAL
// ============================================================

func TestPurchaseCard_ValidAmounts(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		currency string
		wantErr  bool
	}{
		{
			name:     "minimum valid amount $5",
			amount:   5.0,
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "maximum valid amount $500",
			amount:   500.0,
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "typical amount $50",
			amount:   50.0,
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "decimal amount $25.50",
			amount:   25.50,
			currency: "USD",
			wantErr:  false,
		},
		{
			name:     "default currency when empty",
			amount:   100.0,
			currency: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)
			purchaserID := uuid.New()

			repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
				return card.OriginalAmount == tt.amount &&
					card.RemainingAmount == tt.amount &&
					card.PurchaserID != nil &&
					*card.PurchaserID == purchaserID &&
					card.Status == CardStatusActive &&
					card.CardType == CardTypePurchased
			})).Return(nil).Once()

			req := &PurchaseGiftCardRequest{
				Amount:   tt.amount,
				Currency: tt.currency,
			}

			card, err := service.PurchaseCard(ctx, purchaserID, req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, card)
			} else {
				require.NoError(t, err)
				require.NotNil(t, card)
				assert.Equal(t, tt.amount, card.OriginalAmount)
				assert.Equal(t, tt.amount, card.RemainingAmount)
				assert.Equal(t, purchaserID, *card.PurchaserID)
				assert.NotEmpty(t, card.Code)
				assert.NotNil(t, card.ExpiresAt)
				// Verify expiry is approximately 1 year from now
				assert.WithinDuration(t, time.Now().AddDate(1, 0, 0), *card.ExpiresAt, time.Hour)
			}

			repo.AssertExpectations(t)
		})
	}
}

func TestPurchaseCard_InvalidAmounts(t *testing.T) {
	tests := []struct {
		name    string
		amount  float64
		wantErr string
	}{
		{
			name:    "below minimum $4.99",
			amount:  4.99,
			wantErr: "amount must be between 5 and 500",
		},
		{
			name:    "zero amount",
			amount:  0.0,
			wantErr: "amount must be between 5 and 500",
		},
		{
			name:    "negative amount",
			amount:  -10.0,
			wantErr: "amount must be between 5 and 500",
		},
		{
			name:    "above maximum $500.01",
			amount:  500.01,
			wantErr: "amount must be between 5 and 500",
		},
		{
			name:    "way above maximum $1000",
			amount:  1000.0,
			wantErr: "amount must be between 5 and 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)
			purchaserID := uuid.New()

			req := &PurchaseGiftCardRequest{
				Amount: tt.amount,
			}

			card, err := service.PurchaseCard(ctx, purchaserID, req)

			require.Error(t, err)
			assert.Nil(t, card)
			assert.Contains(t, err.Error(), tt.wantErr)

			// Repository should NOT be called for invalid amounts
			repo.AssertNotCalled(t, "CreateCard", mock.Anything, mock.Anything)
		})
	}
}

func TestPurchaseCard_BoundaryValues(t *testing.T) {
	// Test exact boundary values (FINANCIAL CRITICAL)
	tests := []struct {
		name    string
		amount  float64
		wantErr bool
	}{
		{"exactly $5 (min)", 5.0, false},
		{"just below min $4.999999", 4.999999, true},
		{"exactly $500 (max)", 500.0, false},
		{"just above max $500.000001", 500.000001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)
			purchaserID := uuid.New()

			if !tt.wantErr {
				repo.On("CreateCard", ctx, mock.Anything).Return(nil).Once()
			}

			req := &PurchaseGiftCardRequest{Amount: tt.amount}
			card, err := service.PurchaseCard(ctx, purchaserID, req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, card)
			} else {
				require.NoError(t, err)
				require.NotNil(t, card)
			}
		})
	}
}

func TestPurchaseCard_WithRecipientDetails(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	purchaserID := uuid.New()

	email := "friend@example.com"
	name := "Best Friend"
	message := "Happy Birthday!"
	design := "birthday_celebration"

	repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
		return card.RecipientEmail != nil && *card.RecipientEmail == email &&
			card.RecipientName != nil && *card.RecipientName == name &&
			card.PersonalMessage != nil && *card.PersonalMessage == message &&
			card.DesignTemplate != nil && *card.DesignTemplate == design
	})).Return(nil).Once()

	req := &PurchaseGiftCardRequest{
		Amount:          100.0,
		RecipientEmail:  &email,
		RecipientName:   &name,
		PersonalMessage: &message,
		DesignTemplate:  &design,
	}

	card, err := service.PurchaseCard(ctx, purchaserID, req)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, &email, card.RecipientEmail)
	assert.Equal(t, &name, card.RecipientName)
	assert.Equal(t, &message, card.PersonalMessage)
	assert.Equal(t, &design, card.DesignTemplate)
	repo.AssertExpectations(t)
}

func TestPurchaseCard_RepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	purchaserID := uuid.New()

	repo.On("CreateCard", ctx, mock.Anything).Return(errors.New("database connection failed")).Once()

	req := &PurchaseGiftCardRequest{Amount: 50.0}
	card, err := service.PurchaseCard(ctx, purchaserID, req)

	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "create card")
	repo.AssertExpectations(t)
}

// ============================================================
// RedeemCard Tests - FINANCIAL CRITICAL
// ============================================================

func TestRedeemCard_ValidCode(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	expiresAt := time.Now().AddDate(0, 6, 0) // Expires in 6 months
	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "TEST-CODE-1234-5678",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 100.0,
		ExpiresAt:       &expiresAt,
	}

	repo.On("GetCardByCode", ctx, "TEST-CODE-1234-5678").Return(existingCard, nil).Once()
	repo.On("RedeemCard", ctx, existingCard.ID, userID).Return(nil).Once()

	req := &RedeemGiftCardRequest{Code: "TEST-CODE-1234-5678"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, userID, *card.RecipientID)
	assert.NotNil(t, card.RedeemedAt)
	repo.AssertExpectations(t)
}

func TestRedeemCard_AlreadyRedeemedBySameUser(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	expiresAt := time.Now().AddDate(0, 6, 0)
	redeemedAt := time.Now().Add(-time.Hour)
	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "ALREADY-REDEEMED",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 75.0, // Partially used
		RecipientID:     &userID,
		RedeemedAt:      &redeemedAt,
		ExpiresAt:       &expiresAt,
	}

	repo.On("GetCardByCode", ctx, "ALREADY-REDEEMED").Return(existingCard, nil).Once()
	// RedeemCard should NOT be called since already redeemed by same user

	req := &RedeemGiftCardRequest{Code: "ALREADY-REDEEMED"}
	card, err := service.RedeemCard(ctx, userID, req)

	// Should succeed and return the existing card
	require.NoError(t, err)
	require.NotNil(t, card)
	assert.Equal(t, existingCard.ID, card.ID)
	repo.AssertExpectations(t)
}

func TestRedeemCard_AlreadyRedeemedByDifferentUser(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	otherUserID := uuid.New()

	expiresAt := time.Now().AddDate(0, 6, 0)
	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "OTHER-USER-CARD",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 50.0,
		RecipientID:     &otherUserID, // Already redeemed by someone else
		ExpiresAt:       &expiresAt,
	}

	repo.On("GetCardByCode", ctx, "OTHER-USER-CARD").Return(existingCard, nil).Once()

	req := &RedeemGiftCardRequest{Code: "OTHER-USER-CARD"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "already redeemed by another user")
	repo.AssertExpectations(t)
}

func TestRedeemCard_ExpiredCard(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	expiredAt := time.Now().Add(-24 * time.Hour) // Expired yesterday
	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "EXPIRED-CARD",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 100.0,
		ExpiresAt:       &expiredAt,
	}

	repo.On("GetCardByCode", ctx, "EXPIRED-CARD").Return(existingCard, nil).Once()

	req := &RedeemGiftCardRequest{Code: "EXPIRED-CARD"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "expired")
	repo.AssertExpectations(t)
}

func TestRedeemCard_InactiveCard(t *testing.T) {
	tests := []struct {
		name   string
		status CardStatus
	}{
		{"redeemed status", CardStatusRedeemed},
		{"expired status", CardStatusExpired},
		{"disabled status", CardStatusDisabled},
		{"pending status", CardStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)
			userID := uuid.New()

			existingCard := &GiftCard{
				ID:              uuid.New(),
				Code:            "INACTIVE-CARD",
				Status:          tt.status,
				OriginalAmount:  100.0,
				RemainingAmount: 100.0,
			}

			repo.On("GetCardByCode", ctx, "INACTIVE-CARD").Return(existingCard, nil).Once()

			req := &RedeemGiftCardRequest{Code: "INACTIVE-CARD"}
			card, err := service.RedeemCard(ctx, userID, req)

			require.Error(t, err)
			assert.Nil(t, card)
			assert.Contains(t, err.Error(), "not active")
			repo.AssertExpectations(t)
		})
	}
}

func TestRedeemCard_ZeroBalance(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "ZERO-BALANCE",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 0.0, // Fully used
	}

	repo.On("GetCardByCode", ctx, "ZERO-BALANCE").Return(existingCard, nil).Once()

	req := &RedeemGiftCardRequest{Code: "ZERO-BALANCE"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "no remaining balance")
	repo.AssertExpectations(t)
}

func TestRedeemCard_CardNotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetCardByCode", ctx, "NONEXISTENT").Return((*GiftCard)(nil), pgx.ErrNoRows).Once()

	req := &RedeemGiftCardRequest{Code: "NONEXISTENT"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.Error(t, err)
	assert.Nil(t, card)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

func TestRedeemCard_NoExpiryDate(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	existingCard := &GiftCard{
		ID:              uuid.New(),
		Code:            "NO-EXPIRY",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 100.0,
		ExpiresAt:       nil, // No expiry
	}

	repo.On("GetCardByCode", ctx, "NO-EXPIRY").Return(existingCard, nil).Once()
	repo.On("RedeemCard", ctx, existingCard.ID, userID).Return(nil).Once()

	req := &RedeemGiftCardRequest{Code: "NO-EXPIRY"}
	card, err := service.RedeemCard(ctx, userID, req)

	require.NoError(t, err)
	require.NotNil(t, card)
	repo.AssertExpectations(t)
}

// ============================================================
// UseBalance Tests - FINANCIAL CRITICAL (FIFO Logic)
// ============================================================

func TestUseBalance_SufficientSingleCard(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	cards := []GiftCard{
		{
			ID:              uuid.New(),
			RemainingAmount: 50.0,
			Status:          CardStatusActive,
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("DeductBalance", ctx, cards[0].ID, 25.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == cards[0].ID &&
			tx.UserID == userID &&
			tx.RideID != nil && *tx.RideID == rideID &&
			tx.Amount == 25.0 &&
			tx.BalanceBefore == 50.0 &&
			tx.BalanceAfter == 25.0
	})).Return(nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 25.0)

	require.NoError(t, err)
	assert.Equal(t, 25.0, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_InsufficientBalance(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	cards := []GiftCard{
		{
			ID:              uuid.New(),
			RemainingAmount: 30.0,
			Status:          CardStatusActive,
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("DeductBalance", ctx, cards[0].ID, 30.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.Amount == 30.0 && tx.BalanceAfter == 0.0
	})).Return(nil).Once()

	// Request $50 but only $30 available
	deducted, err := service.UseBalance(ctx, userID, rideID, 50.0)

	require.NoError(t, err)
	assert.Equal(t, 30.0, deducted) // Should only deduct available amount
	repo.AssertExpectations(t)
}

func TestUseBalance_FIFOOrder(t *testing.T) {
	// CRITICAL: Oldest cards should be used first
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	oldCardID := uuid.New()
	newCardID := uuid.New()

	// Cards are returned in FIFO order (oldest first)
	cards := []GiftCard{
		{
			ID:              oldCardID,
			RemainingAmount: 30.0,
			Status:          CardStatusActive,
			CreatedAt:       time.Now().Add(-48 * time.Hour), // 2 days ago
		},
		{
			ID:              newCardID,
			RemainingAmount: 50.0,
			Status:          CardStatusActive,
			CreatedAt:       time.Now().Add(-24 * time.Hour), // 1 day ago
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	// Should use old card first (FIFO)
	repo.On("DeductBalance", ctx, oldCardID, 30.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == oldCardID && tx.Amount == 30.0
	})).Return(nil).Once()
	// Then use new card for remaining
	repo.On("DeductBalance", ctx, newCardID, 20.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == newCardID && tx.Amount == 20.0
	})).Return(nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 50.0)

	require.NoError(t, err)
	assert.Equal(t, 50.0, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_MultipleCardsPartialUse(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	card1ID := uuid.New()
	card2ID := uuid.New()
	card3ID := uuid.New()

	cards := []GiftCard{
		{ID: card1ID, RemainingAmount: 10.0, Status: CardStatusActive},
		{ID: card2ID, RemainingAmount: 15.0, Status: CardStatusActive},
		{ID: card3ID, RemainingAmount: 25.0, Status: CardStatusActive},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("DeductBalance", ctx, card1ID, 10.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == card1ID && tx.Amount == 10.0
	})).Return(nil).Once()
	repo.On("DeductBalance", ctx, card2ID, 15.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == card2ID && tx.Amount == 15.0
	})).Return(nil).Once()
	repo.On("DeductBalance", ctx, card3ID, 5.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == card3ID && tx.Amount == 5.0
	})).Return(nil).Once()

	// Total: $10 + $15 + $5 = $30
	deducted, err := service.UseBalance(ctx, userID, rideID, 30.0)

	require.NoError(t, err)
	assert.Equal(t, 30.0, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_NoCardsAvailable(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveCardsByUser", ctx, userID).Return([]GiftCard{}, nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 25.0)

	require.NoError(t, err)
	assert.Equal(t, 0.0, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_RoundingPrecision(t *testing.T) {
	// FINANCIAL CRITICAL: Test rounding to 2 decimal places
	tests := []struct {
		name           string
		cardBalance    float64
		requestAmount  float64
		expectedDeduct float64
	}{
		{
			name:           "round down small fraction",
			cardBalance:    10.999,
			requestAmount:  10.999,
			expectedDeduct: 11.0, // math.Round(10.999*100)/100 = 11.0
		},
		{
			name:           "round properly half cent",
			cardBalance:    50.0,
			requestAmount:  10.005,
			expectedDeduct: 10.01, // math.Round(10.005*100)/100 = 10.01 (banker's rounding)
		},
		{
			name:           "exact two decimals",
			cardBalance:    50.0,
			requestAmount:  25.75,
			expectedDeduct: 25.75,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)
			userID := uuid.New()
			rideID := uuid.New()

			cards := []GiftCard{
				{
					ID:              uuid.New(),
					RemainingAmount: tt.cardBalance,
					Status:          CardStatusActive,
				},
			}

			repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
			repo.On("DeductBalance", ctx, cards[0].ID, tt.expectedDeduct).Return(true, nil).Once()
			repo.On("CreateTransaction", ctx, mock.Anything).Return(nil).Once()

			deducted, err := service.UseBalance(ctx, userID, rideID, tt.requestAmount)

			require.NoError(t, err)
			assert.InDelta(t, tt.expectedDeduct, deducted, 0.001)
			repo.AssertExpectations(t)
		})
	}
}

func TestUseBalance_DeductFailureSkipsCard(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	card1ID := uuid.New()
	card2ID := uuid.New()

	cards := []GiftCard{
		{ID: card1ID, RemainingAmount: 20.0, Status: CardStatusActive},
		{ID: card2ID, RemainingAmount: 30.0, Status: CardStatusActive},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	// First card fails to deduct (race condition or other issue)
	repo.On("DeductBalance", ctx, card1ID, 20.0).Return(false, nil).Once()
	// Service should continue with second card
	repo.On("DeductBalance", ctx, card2ID, 25.0).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.MatchedBy(func(tx *GiftCardTransaction) bool {
		return tx.CardID == card2ID && tx.Amount == 25.0
	})).Return(nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 25.0)

	require.NoError(t, err)
	assert.Equal(t, 25.0, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_ZeroAmount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	cards := []GiftCard{
		{
			ID:              uuid.New(),
			RemainingAmount: 50.0,
			Status:          CardStatusActive,
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 0.0)

	require.NoError(t, err)
	assert.Equal(t, 0.0, deducted)
	// No deductions should be made
	repo.AssertNotCalled(t, "DeductBalance", mock.Anything, mock.Anything, mock.Anything)
}

// ============================================================
// CreateBulk Tests - CORPORATE/ADMIN
// ============================================================

func TestCreateBulk_ValidRequest(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	expiresInDays := 90

	repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
		return card.OriginalAmount == 50.0 &&
			card.RemainingAmount == 50.0 &&
			card.CardType == CardTypeCorporate &&
			card.Status == CardStatusActive &&
			card.Currency == "USD" &&
			card.ExpiresAt != nil
	})).Return(nil).Times(5)

	req := &CreateBulkRequest{
		Count:         5,
		Amount:        50.0,
		Currency:      "USD",
		CardType:      CardTypeCorporate,
		ExpiresInDays: &expiresInDays,
	}

	result, err := service.CreateBulk(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Cards, 5)
	assert.Equal(t, 5, result.Count)
	assert.Equal(t, 250.0, result.Total) // 5 * $50
	repo.AssertExpectations(t)
}

func TestCreateBulk_UniqueCodesGenerated(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("CreateCard", ctx, mock.Anything).Return(nil).Times(10)

	req := &CreateBulkRequest{
		Count:    10,
		Amount:   25.0,
		CardType: CardTypePromotional,
	}

	result, err := service.CreateBulk(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify all codes are unique
	codes := make(map[string]bool)
	for _, card := range result.Cards {
		assert.NotEmpty(t, card.Code)
		assert.False(t, codes[card.Code], "duplicate code found: %s", card.Code)
		codes[card.Code] = true
	}
	repo.AssertExpectations(t)
}

func TestCreateBulk_NoExpiry(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
		return card.ExpiresAt == nil // No expiry
	})).Return(nil).Times(3)

	req := &CreateBulkRequest{
		Count:         3,
		Amount:        100.0,
		CardType:      CardTypeCorporate,
		ExpiresInDays: nil, // No expiry specified
	}

	result, err := service.CreateBulk(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	for _, card := range result.Cards {
		assert.Nil(t, card.ExpiresAt)
	}
	repo.AssertExpectations(t)
}

func TestCreateBulk_DefaultCurrency(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
		return card.Currency == "USD" // Default currency
	})).Return(nil).Times(2)

	req := &CreateBulkRequest{
		Count:    2,
		Amount:   50.0,
		Currency: "", // Empty should default to USD
		CardType: CardTypePromotional,
	}

	result, err := service.CreateBulk(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	for _, card := range result.Cards {
		assert.Equal(t, "USD", card.Currency)
	}
	repo.AssertExpectations(t)
}

func TestCreateBulk_RepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	// First card succeeds, second fails
	repo.On("CreateCard", ctx, mock.Anything).Return(nil).Once()
	repo.On("CreateCard", ctx, mock.Anything).Return(errors.New("database error")).Once()

	req := &CreateBulkRequest{
		Count:    5,
		Amount:   50.0,
		CardType: CardTypeCorporate,
	}

	result, err := service.CreateBulk(ctx, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create card 2") // Failed on second card
	repo.AssertExpectations(t)
}

func TestCreateBulk_CardTypes(t *testing.T) {
	cardTypes := []CardType{
		CardTypePurchased,
		CardTypePromotional,
		CardTypeCorporate,
		CardTypeRefund,
	}

	for _, cardType := range cardTypes {
		t.Run(string(cardType), func(t *testing.T) {
			ctx := context.Background()
			repo := new(mockRepository)
			service := NewService(repo)

			repo.On("CreateCard", ctx, mock.MatchedBy(func(card *GiftCard) bool {
				return card.CardType == cardType
			})).Return(nil).Once()

			req := &CreateBulkRequest{
				Count:    1,
				Amount:   25.0,
				CardType: cardType,
			}

			result, err := service.CreateBulk(ctx, req)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, cardType, result.Cards[0].CardType)
			repo.AssertExpectations(t)
		})
	}
}

// ============================================================
// CheckBalance Tests
// ============================================================

func TestCheckBalance_ValidActiveCard(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	expiresAt := time.Now().AddDate(0, 6, 0)
	card := &GiftCard{
		ID:              uuid.New(),
		Code:            "VALID-CODE",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 75.0,
		Currency:        "USD",
		ExpiresAt:       &expiresAt,
	}

	repo.On("GetCardByCode", ctx, "VALID-CODE").Return(card, nil).Once()

	result, err := service.CheckBalance(ctx, "VALID-CODE")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "VALID-CODE", result.Code)
	assert.Equal(t, CardStatusActive, result.Status)
	assert.Equal(t, 100.0, result.OriginalAmount)
	assert.Equal(t, 75.0, result.RemainingAmount)
	assert.Equal(t, "USD", result.Currency)
	assert.True(t, result.IsValid)
	repo.AssertExpectations(t)
}

func TestCheckBalance_ExpiredCard(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	expiredAt := time.Now().Add(-time.Hour)
	card := &GiftCard{
		ID:              uuid.New(),
		Code:            "EXPIRED-CODE",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 100.0,
		ExpiresAt:       &expiredAt,
	}

	repo.On("GetCardByCode", ctx, "EXPIRED-CODE").Return(card, nil).Once()

	result, err := service.CheckBalance(ctx, "EXPIRED-CODE")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsValid) // Expired card should be invalid
	repo.AssertExpectations(t)
}

func TestCheckBalance_ZeroBalance(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	card := &GiftCard{
		ID:              uuid.New(),
		Code:            "ZERO-BALANCE",
		Status:          CardStatusActive,
		OriginalAmount:  100.0,
		RemainingAmount: 0.0,
	}

	repo.On("GetCardByCode", ctx, "ZERO-BALANCE").Return(card, nil).Once()

	result, err := service.CheckBalance(ctx, "ZERO-BALANCE")

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.IsValid) // No balance = invalid
	repo.AssertExpectations(t)
}

func TestCheckBalance_CardNotFound(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("GetCardByCode", ctx, "NONEXISTENT").Return((*GiftCard)(nil), pgx.ErrNoRows).Once()

	result, err := service.CheckBalance(ctx, "NONEXISTENT")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
	repo.AssertExpectations(t)
}

// ============================================================
// GetMySummary Tests
// ============================================================

func TestGetMySummary_WithActiveCards(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	cards := []GiftCard{
		{ID: uuid.New(), RemainingAmount: 50.0},
		{ID: uuid.New(), RemainingAmount: 25.0},
	}
	txns := []GiftCardTransaction{
		{ID: uuid.New(), Amount: 10.0},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("GetTotalBalance", ctx, userID).Return(75.0, nil).Once()
	repo.On("GetTransactionsByUser", ctx, userID, 20).Return(txns, nil).Once()

	summary, err := service.GetMySummary(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 75.0, summary.TotalBalance)
	assert.Equal(t, 2, summary.ActiveCards)
	assert.Len(t, summary.Cards, 2)
	assert.Len(t, summary.RecentTransactions, 1)
	repo.AssertExpectations(t)
}

func TestGetMySummary_NoCards(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetActiveCardsByUser", ctx, userID).Return([]GiftCard{}, nil).Once()
	repo.On("GetTotalBalance", ctx, userID).Return(0.0, nil).Once()
	repo.On("GetTransactionsByUser", ctx, userID, 20).Return([]GiftCardTransaction{}, nil).Once()

	summary, err := service.GetMySummary(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, summary)
	assert.Equal(t, 0.0, summary.TotalBalance)
	assert.Equal(t, 0, summary.ActiveCards)
	assert.NotNil(t, summary.Cards) // Should be empty slice, not nil
	assert.NotNil(t, summary.RecentTransactions)
	repo.AssertExpectations(t)
}

func TestGetMySummary_ErrorsReturnEmptyData(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	// All repository calls return errors
	repo.On("GetActiveCardsByUser", ctx, userID).Return(([]GiftCard)(nil), errors.New("db error")).Once()
	repo.On("GetTotalBalance", ctx, userID).Return(0.0, errors.New("db error")).Once()
	repo.On("GetTransactionsByUser", ctx, userID, 20).Return(([]GiftCardTransaction)(nil), errors.New("db error")).Once()

	summary, err := service.GetMySummary(ctx, userID)

	require.NoError(t, err) // Should not fail, just return empty data
	require.NotNil(t, summary)
	assert.Equal(t, 0.0, summary.TotalBalance)
	assert.NotNil(t, summary.Cards)
	assert.NotNil(t, summary.RecentTransactions)
	repo.AssertExpectations(t)
}

// ============================================================
// GetPurchasedCards Tests
// ============================================================

func TestGetPurchasedCards_WithCards(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	cards := []GiftCard{
		{ID: uuid.New(), OriginalAmount: 100.0, CardType: CardTypePurchased},
		{ID: uuid.New(), OriginalAmount: 50.0, CardType: CardTypePurchased},
	}

	repo.On("GetPurchasedCardsByUser", ctx, userID).Return(cards, nil).Once()

	result, err := service.GetPurchasedCards(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestGetPurchasedCards_NilReturnsEmptySlice(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetPurchasedCardsByUser", ctx, userID).Return(([]GiftCard)(nil), nil).Once()

	result, err := service.GetPurchasedCards(ctx, userID)

	require.NoError(t, err)
	assert.NotNil(t, result) // Should be empty slice, not nil
	assert.Len(t, result, 0)
	repo.AssertExpectations(t)
}

// ============================================================
// GetTotalBalance Tests
// ============================================================

func TestGetTotalBalance_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetTotalBalance", ctx, userID).Return(150.75, nil).Once()

	balance, err := service.GetTotalBalance(ctx, userID)

	require.NoError(t, err)
	assert.Equal(t, 150.75, balance)
	repo.AssertExpectations(t)
}

func TestGetTotalBalance_Error(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()

	repo.On("GetTotalBalance", ctx, userID).Return(0.0, errors.New("db error")).Once()

	balance, err := service.GetTotalBalance(ctx, userID)

	require.Error(t, err)
	assert.Equal(t, 0.0, balance)
	repo.AssertExpectations(t)
}

// ============================================================
// ExpireCards Tests
// ============================================================

func TestExpireCards_Success(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("ExpireCards", ctx).Return(int64(5), nil).Once()

	count, err := service.ExpireCards(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(5), count)
	repo.AssertExpectations(t)
}

func TestExpireCards_NoCardsExpired(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)

	repo.On("ExpireCards", ctx).Return(int64(0), nil).Once()

	count, err := service.ExpireCards(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(0), count)
	repo.AssertExpectations(t)
}

// ============================================================
// Code Generation Tests
// ============================================================

func TestGenerateGiftCode_Format(t *testing.T) {
	// Generate multiple codes and verify format
	for i := 0; i < 100; i++ {
		code := generateGiftCode()

		// Should be in format XXXX-XXXX-XXXX-XXXX
		assert.Len(t, code, 19) // 16 chars + 3 dashes
		assert.Regexp(t, `^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`, code)

		// Should not contain confusing characters that are excluded:
		// 0 (zero) - looks like O
		// O (letter O) - looks like 0
		// 1 (one) - looks like I or l
		// I (letter I) - looks like 1 or l
		// Note: L is included in the charset (capital L is distinguishable)
		assert.NotContains(t, code, "0")
		assert.NotContains(t, code, "O")
		assert.NotContains(t, code, "1")
		assert.NotContains(t, code, "I")
	}
}

func TestGenerateGiftCode_Uniqueness(t *testing.T) {
	codes := make(map[string]bool)

	// Generate many codes and verify uniqueness
	for i := 0; i < 1000; i++ {
		code := generateGiftCode()
		assert.False(t, codes[code], "duplicate code generated: %s", code)
		codes[code] = true
	}
}

// ============================================================
// Edge Cases and Error Handling
// ============================================================

func TestUseBalance_VerySmallAmount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	cards := []GiftCard{
		{
			ID:              uuid.New(),
			RemainingAmount: 100.0,
			Status:          CardStatusActive,
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("DeductBalance", ctx, cards[0].ID, 0.01).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.Anything).Return(nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 0.01)

	require.NoError(t, err)
	assert.Equal(t, 0.01, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_LargeAmount(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	cards := []GiftCard{
		{
			ID:              uuid.New(),
			RemainingAmount: 500.0,
			Status:          CardStatusActive,
		},
	}

	repo.On("GetActiveCardsByUser", ctx, userID).Return(cards, nil).Once()
	repo.On("DeductBalance", ctx, cards[0].ID, 499.99).Return(true, nil).Once()
	repo.On("CreateTransaction", ctx, mock.Anything).Return(nil).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 499.99)

	require.NoError(t, err)
	assert.Equal(t, 499.99, deducted)
	repo.AssertExpectations(t)
}

func TestUseBalance_RepositoryError(t *testing.T) {
	ctx := context.Background()
	repo := new(mockRepository)
	service := NewService(repo)
	userID := uuid.New()
	rideID := uuid.New()

	repo.On("GetActiveCardsByUser", ctx, userID).Return(([]GiftCard)(nil), errors.New("connection failed")).Once()

	deducted, err := service.UseBalance(ctx, userID, rideID, 25.0)

	require.Error(t, err)
	assert.Equal(t, 0.0, deducted)
	repo.AssertExpectations(t)
}
