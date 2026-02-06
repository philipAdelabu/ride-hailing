package paymentmethods

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the interface for payment methods repository operations
type RepositoryInterface interface {
	// Payment Methods
	CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error
	GetPaymentMethodsByUser(ctx context.Context, userID uuid.UUID) ([]PaymentMethod, error)
	GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error)
	GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error)
	SetDefault(ctx context.Context, userID, methodID uuid.UUID) error
	DeactivatePaymentMethod(ctx context.Context, id, userID uuid.UUID) error

	// Wallet
	GetWallet(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error)
	EnsureWalletExists(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error)
	UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, delta float64) (float64, error)
	GetWalletBalance(ctx context.Context, userID uuid.UUID) (float64, error)
	CreateWalletTransaction(ctx context.Context, tx *WalletTransaction) error
	GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]WalletTransaction, error)
}
