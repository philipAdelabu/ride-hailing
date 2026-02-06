package giftcards

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the contract for gift cards repository operations
type RepositoryInterface interface {
	// Gift card operations
	CreateCard(ctx context.Context, card *GiftCard) error
	GetCardByCode(ctx context.Context, code string) (*GiftCard, error)
	GetCardByID(ctx context.Context, id uuid.UUID) (*GiftCard, error)
	RedeemCard(ctx context.Context, cardID, userID uuid.UUID) error
	DeductBalance(ctx context.Context, cardID uuid.UUID, amount float64) (bool, error)
	GetActiveCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error)
	GetPurchasedCardsByUser(ctx context.Context, userID uuid.UUID) ([]GiftCard, error)

	// Transaction operations
	CreateTransaction(ctx context.Context, tx *GiftCardTransaction) error
	GetTransactionsByUser(ctx context.Context, userID uuid.UUID, limit int) ([]GiftCardTransaction, error)

	// Balance and admin operations
	GetTotalBalance(ctx context.Context, userID uuid.UUID) (float64, error)
	ExpireCards(ctx context.Context) (int64, error)
}
