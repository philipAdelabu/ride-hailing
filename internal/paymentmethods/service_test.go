package paymentmethods

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreatePaymentMethod(ctx context.Context, pm *PaymentMethod) error {
	args := m.Called(ctx, pm)
	return args.Error(0)
}

func (m *mockRepo) GetPaymentMethodsByUser(ctx context.Context, userID uuid.UUID) ([]PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]PaymentMethod), args.Error(1)
}

func (m *mockRepo) GetPaymentMethodByID(ctx context.Context, id uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *mockRepo) GetDefaultPaymentMethod(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *mockRepo) SetDefault(ctx context.Context, userID, methodID uuid.UUID) error {
	args := m.Called(ctx, userID, methodID)
	return args.Error(0)
}

func (m *mockRepo) DeactivatePaymentMethod(ctx context.Context, id, userID uuid.UUID) error {
	args := m.Called(ctx, id, userID)
	return args.Error(0)
}

func (m *mockRepo) GetWallet(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *mockRepo) EnsureWalletExists(ctx context.Context, userID uuid.UUID) (*PaymentMethod, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*PaymentMethod), args.Error(1)
}

func (m *mockRepo) UpdateWalletBalance(ctx context.Context, walletID uuid.UUID, delta float64) (float64, error) {
	args := m.Called(ctx, walletID, delta)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepo) GetWalletBalance(ctx context.Context, userID uuid.UUID) (float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(float64), args.Error(1)
}

func (m *mockRepo) CreateWalletTransaction(ctx context.Context, tx *WalletTransaction) error {
	args := m.Called(ctx, tx)
	return args.Error(0)
}

func (m *mockRepo) GetWalletTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]WalletTransaction, error) {
	args := m.Called(ctx, userID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WalletTransaction), args.Error(1)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

// ========================================
// ADD CARD TESTS
// ========================================

func TestAddCard(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		req        *AddCardRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errMessage string
		validate   func(t *testing.T, pm *PaymentMethod)
	}{
		{
			name: "success - first card becomes default",
			req: &AddCardRequest{
				Token:        "tok_visa_123",
				SetAsDefault: false,
			},
			setupMocks: func(m *mockRepo) {
				// No existing payment methods
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.Equal(t, userID, pm.UserID)
				assert.Equal(t, PaymentMethodCard, pm.Type)
				assert.True(t, pm.IsDefault, "first card should be default")
				assert.True(t, pm.IsActive)
				assert.NotNil(t, pm.CardBrand)
				assert.Equal(t, CardBrandVisa, *pm.CardBrand)
				assert.NotNil(t, pm.CardLast4)
				assert.Equal(t, "4242", *pm.CardLast4)
			},
		},
		{
			name: "success - additional card with set_as_default",
			req: &AddCardRequest{
				Token:        "tok_mastercard_456",
				SetAsDefault: true,
			},
			setupMocks: func(m *mockRepo) {
				existingMethods := []PaymentMethod{
					{ID: uuid.New(), UserID: userID, Type: PaymentMethodCard, IsDefault: true},
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(existingMethods, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.True(t, pm.IsDefault, "should be default when set_as_default is true")
			},
		},
		{
			name: "success - additional card without set_as_default",
			req: &AddCardRequest{
				Token:        "tok_amex_789",
				SetAsDefault: false,
			},
			setupMocks: func(m *mockRepo) {
				existingMethods := []PaymentMethod{
					{ID: uuid.New(), UserID: userID, Type: PaymentMethodCard, IsDefault: true},
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(existingMethods, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.False(t, pm.IsDefault, "should not be default")
			},
		},
		{
			name: "error - max payment methods reached",
			req: &AddCardRequest{
				Token: "tok_visa_123",
			},
			setupMocks: func(m *mockRepo) {
				// Return DefaultConfig().MaxPaymentMethods (10) existing methods
				methods := make([]PaymentMethod, DefaultConfig().MaxPaymentMethods)
				for i := 0; i < DefaultConfig().MaxPaymentMethods; i++ {
					methods[i] = PaymentMethod{ID: uuid.New(), UserID: userID, Type: PaymentMethodCard}
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
			},
			wantErr:    true,
			errMessage: "maximum 10 payment methods allowed",
		},
		{
			name: "error - repository error on get methods",
			req: &AddCardRequest{
				Token: "tok_visa_123",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error - repository error on create",
			req: &AddCardRequest{
				Token: "tok_visa_123",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			pm, err := svc.AddCard(context.Background(), userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, pm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pm)
				tt.validate(t, pm)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TOP UP WALLET TESTS
// ========================================

func TestTopUpWallet(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	cardID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	walletID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name       string
		req        *TopUpWalletRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errMessage string
	}{
		{
			name: "success - top up within limits",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				card := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				balance := 100.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, 50.0).Return(150.0, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(150.0, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - top up at minimum amount",
			req: &TopUpWalletRequest{
				Amount:         DefaultConfig().MinTopUpAmount, // 5.0
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				card := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				balance := 0.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, DefaultConfig().MinTopUpAmount).Return(DefaultConfig().MinTopUpAmount, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(DefaultConfig().MinTopUpAmount, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - top up at maximum amount",
			req: &TopUpWalletRequest{
				Amount:         DefaultConfig().MaxTopUpAmount, // 500.0
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				card := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				balance := 0.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, DefaultConfig().MaxTopUpAmount).Return(DefaultConfig().MaxTopUpAmount, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(DefaultConfig().MaxTopUpAmount, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error - amount too low",
			req: &TopUpWalletRequest{
				Amount:         DefaultConfig().MinTopUpAmount - 0.01, // 4.99
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errMessage: "top up amount must be between",
		},
		{
			name: "error - amount too high",
			req: &TopUpWalletRequest{
				Amount:         DefaultConfig().MaxTopUpAmount + 0.01, // 500.01
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errMessage: "top up amount must be between",
		},
		{
			name: "error - source payment method not found",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errMessage: "source payment method not found",
		},
		{
			name: "error - source payment method belongs to different user",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				otherUserID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
				card := &PaymentMethod{
					ID:     cardID,
					UserID: otherUserID, // Different user
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
			},
			wantErr:    true,
			errMessage: "forbidden",
		},
		{
			name: "error - source is not a card (wallet)",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				wallet := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodWallet, // Not a card
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(wallet, nil)
			},
			wantErr:    true,
			errMessage: "can only top up from a card",
		},
		{
			name: "error - source is not a card (cash)",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				cash := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCash,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(cash, nil)
			},
			wantErr:    true,
			errMessage: "can only top up from a card",
		},
		{
			name: "error - ensure wallet fails",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				card := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(nil, errors.New("wallet creation failed"))
			},
			wantErr: true,
		},
		{
			name: "error - update balance fails",
			req: &TopUpWalletRequest{
				Amount:         50.0,
				SourceMethodID: cardID,
			},
			setupMocks: func(m *mockRepo) {
				card := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				balance := 0.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(card, nil)
				m.On("EnsureWalletExists", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, 50.0).Return(0.0, errors.New("balance update failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.TopUpWallet(context.Background(), userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, summary)
			} else {
				require.NoError(t, err)
				require.NotNil(t, summary)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// DEDUCT FROM WALLET TESTS
// ========================================

func TestDeductFromWallet(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	walletID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	rideID := uuid.MustParse("66666666-6666-6666-6666-666666666666")

	tests := []struct {
		name           string
		amount         float64
		setupMocks     func(m *mockRepo)
		wantErr        bool
		wantDeducted   float64
	}{
		{
			name:   "success - full amount deducted",
			amount: 15.0,
			setupMocks: func(m *mockRepo) {
				balance := 50.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetWallet", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, -15.0).Return(35.0, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
			},
			wantErr:      false,
			wantDeducted: 15.0,
		},
		{
			name:   "success - partial deduction (insufficient balance)",
			amount: 100.0,
			setupMocks: func(m *mockRepo) {
				balance := 30.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetWallet", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, -30.0).Return(0.0, nil)
				m.On("CreateWalletTransaction", mock.Anything, mock.AnythingOfType("*paymentmethods.WalletTransaction")).Return(nil)
			},
			wantErr:      false,
			wantDeducted: 30.0, // Only deducts what's available
		},
		{
			name:   "success - zero balance returns zero",
			amount: 25.0,
			setupMocks: func(m *mockRepo) {
				balance := 0.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetWallet", mock.Anything, userID).Return(wallet, nil)
			},
			wantErr:      false,
			wantDeducted: 0.0,
		},
		{
			name:   "success - nil balance treated as zero",
			amount: 25.0,
			setupMocks: func(m *mockRepo) {
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: nil, // nil balance
				}
				m.On("GetWallet", mock.Anything, userID).Return(wallet, nil)
			},
			wantErr:      false,
			wantDeducted: 0.0,
		},
		{
			name:   "error - wallet not found",
			amount: 15.0,
			setupMocks: func(m *mockRepo) {
				m.On("GetWallet", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:      true,
			wantDeducted: 0.0,
		},
		{
			name:   "error - update balance fails",
			amount: 15.0,
			setupMocks: func(m *mockRepo) {
				balance := 50.0
				wallet := &PaymentMethod{
					ID:            walletID,
					UserID:        userID,
					Type:          PaymentMethodWallet,
					WalletBalance: &balance,
				}
				m.On("GetWallet", mock.Anything, userID).Return(wallet, nil)
				m.On("UpdateWalletBalance", mock.Anything, walletID, -15.0).Return(0.0, errors.New("update failed"))
			},
			wantErr:      true,
			wantDeducted: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			deducted, err := svc.DeductFromWallet(context.Background(), userID, rideID, tt.amount)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantDeducted, deducted)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// GET PAYMENT METHODS TESTS
// ========================================

func TestGetPaymentMethods(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	defaultCardID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	walletID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, resp *PaymentMethodsResponse)
	}{
		{
			name: "success - returns methods with wallet balance and default",
			setupMocks: func(m *mockRepo) {
				methods := []PaymentMethod{
					{ID: defaultCardID, UserID: userID, Type: PaymentMethodCard, IsDefault: true},
					{ID: walletID, UserID: userID, Type: PaymentMethodWallet, IsDefault: false},
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(75.50, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *PaymentMethodsResponse) {
				assert.Len(t, resp.Methods, 2)
				assert.NotNil(t, resp.DefaultMethod)
				assert.Equal(t, defaultCardID, *resp.DefaultMethod)
				assert.Equal(t, 75.50, resp.WalletBalance)
				assert.Equal(t, "USD", resp.Currency)
			},
		},
		{
			name: "success - empty methods returns empty list",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(0.0, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *PaymentMethodsResponse) {
				assert.NotNil(t, resp.Methods)
				assert.Len(t, resp.Methods, 0)
				assert.Nil(t, resp.DefaultMethod)
				assert.Equal(t, 0.0, resp.WalletBalance)
			},
		},
		{
			name: "success - no default method",
			setupMocks: func(m *mockRepo) {
				methods := []PaymentMethod{
					{ID: walletID, UserID: userID, Type: PaymentMethodWallet, IsDefault: false},
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(25.0, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *PaymentMethodsResponse) {
				assert.Len(t, resp.Methods, 1)
				assert.Nil(t, resp.DefaultMethod)
			},
		},
		{
			name: "success - wallet balance error is handled gracefully",
			setupMocks: func(m *mockRepo) {
				methods := []PaymentMethod{
					{ID: defaultCardID, UserID: userID, Type: PaymentMethodCard, IsDefault: true},
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(methods, nil)
				m.On("GetWalletBalance", mock.Anything, userID).Return(0.0, errors.New("balance error"))
			},
			wantErr: false,
			validate: func(t *testing.T, resp *PaymentMethodsResponse) {
				assert.Len(t, resp.Methods, 1)
				assert.Equal(t, 0.0, resp.WalletBalance) // Default to 0 on error
			},
		},
		{
			name: "error - repository error on get methods",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.GetPaymentMethods(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				tt.validate(t, resp)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// SET DEFAULT TESTS
// ========================================

func TestSetDefault(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	methodID := uuid.MustParse("33333333-3333-3333-3333-333333333333")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errMessage string
	}{
		{
			name: "success",
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     methodID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
				m.On("SetDefault", mock.Anything, userID, methodID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - payment method not found",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errMessage: "payment method not found",
		},
		{
			name: "error - payment method belongs to different user",
			setupMocks: func(m *mockRepo) {
				otherUserID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
				pm := &PaymentMethod{
					ID:     methodID,
					UserID: otherUserID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
			},
			wantErr:    true,
			errMessage: "forbidden",
		},
		{
			name: "error - repository error on get",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error - repository error on set default",
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     methodID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, methodID).Return(pm, nil)
				m.On("SetDefault", mock.Anything, userID, methodID).Return(errors.New("update failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.SetDefault(context.Background(), userID, methodID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// REMOVE PAYMENT METHOD TESTS
// ========================================

func TestRemovePaymentMethod(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	cardID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	walletID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name       string
		methodID   uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
		errMessage string
	}{
		{
			name:     "success - remove card",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(pm, nil)
				m.On("DeactivatePaymentMethod", mock.Anything, cardID, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "success - remove apple pay",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodApplePay,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(pm, nil)
				m.On("DeactivatePaymentMethod", mock.Anything, cardID, userID).Return(nil)
			},
			wantErr: false,
		},
		{
			name:     "error - cannot remove wallet",
			methodID: walletID,
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     walletID,
					UserID: userID,
					Type:   PaymentMethodWallet,
				}
				m.On("GetPaymentMethodByID", mock.Anything, walletID).Return(pm, nil)
			},
			wantErr:    true,
			errMessage: "wallet cannot be removed",
		},
		{
			name:     "error - payment method not found",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(nil, pgx.ErrNoRows)
			},
			wantErr:    true,
			errMessage: "payment method not found",
		},
		{
			name:     "error - payment method belongs to different user",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				otherUserID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
				pm := &PaymentMethod{
					ID:     cardID,
					UserID: otherUserID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(pm, nil)
			},
			wantErr:    true,
			errMessage: "forbidden",
		},
		{
			name:     "error - repository error on get",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name:     "error - repository error on deactivate",
			methodID: cardID,
			setupMocks: func(m *mockRepo) {
				pm := &PaymentMethod{
					ID:     cardID,
					UserID: userID,
					Type:   PaymentMethodCard,
				}
				m.On("GetPaymentMethodByID", mock.Anything, cardID).Return(pm, nil)
				m.On("DeactivatePaymentMethod", mock.Anything, cardID, userID).Return(errors.New("deactivate failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.RemovePaymentMethod(context.Background(), userID, tt.methodID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// ADD DIGITAL WALLET TESTS
// ========================================

func TestAddDigitalWallet(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		req        *AddWalletPaymentRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errMessage string
		validate   func(t *testing.T, pm *PaymentMethod)
	}{
		{
			name: "success - add Apple Pay",
			req: &AddWalletPaymentRequest{
				Type:  PaymentMethodApplePay,
				Token: "apple_pay_token_123",
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.Equal(t, PaymentMethodApplePay, pm.Type)
				assert.Equal(t, userID, pm.UserID)
				assert.True(t, pm.IsActive)
				assert.False(t, pm.IsDefault)
			},
		},
		{
			name: "success - add Google Pay",
			req: &AddWalletPaymentRequest{
				Type:  PaymentMethodGooglePay,
				Token: "google_pay_token_456",
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.Equal(t, PaymentMethodGooglePay, pm.Type)
			},
		},
		{
			name: "error - invalid type (card)",
			req: &AddWalletPaymentRequest{
				Type:  PaymentMethodCard,
				Token: "card_token",
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errMessage: "type must be apple_pay or google_pay",
		},
		{
			name: "error - invalid type (wallet)",
			req: &AddWalletPaymentRequest{
				Type:  PaymentMethodWallet,
				Token: "wallet_token",
			},
			setupMocks: func(m *mockRepo) {},
			wantErr:    true,
			errMessage: "type must be apple_pay or google_pay",
		},
		{
			name: "error - repository error on create",
			req: &AddWalletPaymentRequest{
				Type:  PaymentMethodApplePay,
				Token: "apple_pay_token",
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			pm, err := svc.AddDigitalWallet(context.Background(), userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMessage != "" {
					assert.Contains(t, err.Error(), tt.errMessage)
				}
				assert.Nil(t, pm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pm)
				tt.validate(t, pm)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// ENABLE CASH TESTS
// ========================================

func TestEnableCash(t *testing.T) {
	userID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	existingCashID := uuid.MustParse("77777777-7777-7777-7777-777777777777")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, pm *PaymentMethod)
	}{
		{
			name: "success - enable cash for first time",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.Equal(t, PaymentMethodCash, pm.Type)
				assert.Equal(t, userID, pm.UserID)
				assert.True(t, pm.IsActive)
				assert.False(t, pm.IsDefault)
			},
		},
		{
			name: "success - cash already enabled returns existing",
			setupMocks: func(m *mockRepo) {
				existingCash := PaymentMethod{
					ID:     existingCashID,
					UserID: userID,
					Type:   PaymentMethodCash,
				}
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{existingCash}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, pm *PaymentMethod) {
				assert.Equal(t, existingCashID, pm.ID)
				assert.Equal(t, PaymentMethodCash, pm.Type)
			},
		},
		{
			name: "error - repository error on get methods",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
		{
			name: "error - repository error on create",
			setupMocks: func(m *mockRepo) {
				m.On("GetPaymentMethodsByUser", mock.Anything, userID).Return([]PaymentMethod{}, nil)
				m.On("CreatePaymentMethod", mock.Anything, mock.AnythingOfType("*paymentmethods.PaymentMethod")).Return(errors.New("create failed"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			pm, err := svc.EnableCash(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, pm)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pm)
				tt.validate(t, pm)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// GET WALLET SUMMARY TESTS
// ========================================

func TestGetWalletSummary(t *testing.T) {
	userID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, summary *WalletSummary)
	}{
		{
			name: "success - returns balance and transactions",
			setupMocks: func(m *mockRepo) {
				transactions := []WalletTransaction{
					{ID: uuid.New(), UserID: userID, Type: "topup", Amount: 50.0},
					{ID: uuid.New(), UserID: userID, Type: "debit", Amount: 15.0},
				}
				m.On("GetWalletBalance", mock.Anything, userID).Return(35.0, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return(transactions, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WalletSummary) {
				assert.Equal(t, 35.0, summary.Balance)
				assert.Equal(t, "USD", summary.Currency)
				assert.Len(t, summary.RecentTransactions, 2)
			},
		},
		{
			name: "success - handles balance error gracefully",
			setupMocks: func(m *mockRepo) {
				m.On("GetWalletBalance", mock.Anything, userID).Return(0.0, errors.New("balance error"))
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return([]WalletTransaction{}, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WalletSummary) {
				assert.Equal(t, 0.0, summary.Balance)
			},
		},
		{
			name: "success - handles transactions error gracefully",
			setupMocks: func(m *mockRepo) {
				m.On("GetWalletBalance", mock.Anything, userID).Return(100.0, nil)
				m.On("GetWalletTransactions", mock.Anything, userID, 20).Return(nil, errors.New("transactions error"))
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WalletSummary) {
				assert.Equal(t, 100.0, summary.Balance)
				assert.NotNil(t, summary.RecentTransactions)
				assert.Len(t, summary.RecentTransactions, 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.GetWalletSummary(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, summary)
			} else {
				require.NoError(t, err)
				require.NotNil(t, summary)
				tt.validate(t, summary)
			}

			m.AssertExpectations(t)
		})
	}
}
