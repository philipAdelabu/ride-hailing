package family

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

// ========================================
// INTERNAL MOCK (same package for unexported access)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) CreateFamily(ctx context.Context, family *FamilyAccount) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *mockRepo) GetFamilyByID(ctx context.Context, id uuid.UUID) (*FamilyAccount, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyAccount), args.Error(1)
}

func (m *mockRepo) GetFamilyByOwner(ctx context.Context, ownerID uuid.UUID) (*FamilyAccount, error) {
	args := m.Called(ctx, ownerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyAccount), args.Error(1)
}

func (m *mockRepo) UpdateFamily(ctx context.Context, family *FamilyAccount) error {
	args := m.Called(ctx, family)
	return args.Error(0)
}

func (m *mockRepo) IncrementSpend(ctx context.Context, familyID uuid.UUID, amount float64) error {
	args := m.Called(ctx, familyID, amount)
	return args.Error(0)
}

func (m *mockRepo) ResetMonthlySpend(ctx context.Context, dayOfMonth int) error {
	args := m.Called(ctx, dayOfMonth)
	return args.Error(0)
}

func (m *mockRepo) AddMember(ctx context.Context, member *FamilyMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *mockRepo) GetMembersByFamily(ctx context.Context, familyID uuid.UUID) ([]FamilyMember, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FamilyMember), args.Error(1)
}

func (m *mockRepo) GetMemberByUserID(ctx context.Context, familyID, userID uuid.UUID) (*FamilyMember, error) {
	args := m.Called(ctx, familyID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyMember), args.Error(1)
}

func (m *mockRepo) GetFamilyForUser(ctx context.Context, userID uuid.UUID) (*FamilyAccount, *FamilyMember, error) {
	args := m.Called(ctx, userID)
	var family *FamilyAccount
	var member *FamilyMember
	if args.Get(0) != nil {
		family = args.Get(0).(*FamilyAccount)
	}
	if args.Get(1) != nil {
		member = args.Get(1).(*FamilyMember)
	}
	return family, member, args.Error(2)
}

func (m *mockRepo) UpdateMember(ctx context.Context, member *FamilyMember) error {
	args := m.Called(ctx, member)
	return args.Error(0)
}

func (m *mockRepo) RemoveMember(ctx context.Context, memberID uuid.UUID) error {
	args := m.Called(ctx, memberID)
	return args.Error(0)
}

func (m *mockRepo) IncrementMemberSpend(ctx context.Context, memberID uuid.UUID, amount float64) error {
	args := m.Called(ctx, memberID, amount)
	return args.Error(0)
}

func (m *mockRepo) CreateInvite(ctx context.Context, invite *FamilyInvite) error {
	args := m.Called(ctx, invite)
	return args.Error(0)
}

func (m *mockRepo) GetInviteByID(ctx context.Context, id uuid.UUID) (*FamilyInvite, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*FamilyInvite), args.Error(1)
}

func (m *mockRepo) GetPendingInvitesForUser(ctx context.Context, userID uuid.UUID, email *string) ([]FamilyInvite, error) {
	args := m.Called(ctx, userID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FamilyInvite), args.Error(1)
}

func (m *mockRepo) UpdateInviteStatus(ctx context.Context, inviteID uuid.UUID, status InviteStatus) error {
	args := m.Called(ctx, inviteID, status)
	return args.Error(0)
}

func (m *mockRepo) LogRide(ctx context.Context, log *FamilyRideLog) error {
	args := m.Called(ctx, log)
	return args.Error(0)
}

func (m *mockRepo) GetSpendingReport(ctx context.Context, familyID uuid.UUID) ([]MemberSpend, int, float64, error) {
	args := m.Called(ctx, familyID)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Get(2).(float64), args.Error(3)
	}
	return args.Get(0).([]MemberSpend), args.Int(1), args.Get(2).(float64), args.Error(3)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func ptrFloat64(v float64) *float64 { return &v }
func ptrInt(v int) *int             { return &v }
func ptrString(v string) *string    { return &v }

// ========================================
// TESTS: CreateFamily
// ========================================

func TestCreateFamily_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := uuid.New()
	req := &CreateFamilyRequest{
		Name:          "The Smiths",
		MonthlyBudget: ptrFloat64(500),
		Currency:      "USD",
	}

	repo.On("GetFamilyByOwner", ctx, ownerID).Return(nil, pgx.ErrNoRows)
	repo.On("GetFamilyForUser", ctx, ownerID).Return(nil, nil, nil)
	repo.On("CreateFamily", ctx, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)
	repo.On("AddMember", ctx, mock.AnythingOfType("*family.FamilyMember")).Return(nil)

	result, err := svc.CreateFamily(ctx, ownerID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "The Smiths", result.Name)
	assert.Equal(t, ptrFloat64(500), result.MonthlyBudget)
	assert.Equal(t, "USD", result.Currency)
	assert.Equal(t, ownerID, result.OwnerID)
	assert.True(t, result.IsActive)
	assert.Equal(t, float64(0), result.CurrentSpend)
	require.Len(t, result.Members, 1)
	assert.Equal(t, MemberRoleOwner, result.Members[0].Role)
	assert.Equal(t, ownerID, result.Members[0].UserID)
	repo.AssertExpectations(t)
}

func TestCreateFamily_DefaultCurrency(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := uuid.New()
	req := &CreateFamilyRequest{
		Name: "The Johnsons",
		// No currency specified
	}

	repo.On("GetFamilyByOwner", ctx, ownerID).Return(nil, pgx.ErrNoRows)
	repo.On("GetFamilyForUser", ctx, ownerID).Return(nil, nil, nil)
	repo.On("CreateFamily", ctx, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)
	repo.On("AddMember", ctx, mock.AnythingOfType("*family.FamilyMember")).Return(nil)

	result, err := svc.CreateFamily(ctx, ownerID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "USD", result.Currency) // Default currency
	repo.AssertExpectations(t)
}

func TestCreateFamily_UserAlreadyOwnsFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := uuid.New()
	existingFamily := &FamilyAccount{
		ID:      uuid.New(),
		OwnerID: ownerID,
		Name:    "Existing Family",
	}
	req := &CreateFamilyRequest{
		Name: "New Family",
	}

	repo.On("GetFamilyByOwner", ctx, ownerID).Return(existingFamily, nil)

	result, err := svc.CreateFamily(ctx, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "conflict")
	repo.AssertExpectations(t)
}

func TestCreateFamily_UserAlreadyMemberOfFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := uuid.New()
	existingFamily := &FamilyAccount{
		ID:      uuid.New(),
		OwnerID: uuid.New(), // Different owner
		Name:    "Existing Family",
	}
	existingMember := &FamilyMember{
		ID:     uuid.New(),
		UserID: ownerID,
	}
	req := &CreateFamilyRequest{
		Name: "New Family",
	}

	repo.On("GetFamilyByOwner", ctx, ownerID).Return(nil, pgx.ErrNoRows)
	repo.On("GetFamilyForUser", ctx, ownerID).Return(existingFamily, existingMember, nil)

	result, err := svc.CreateFamily(ctx, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "conflict")
	repo.AssertExpectations(t)
}

func TestCreateFamily_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	ownerID := uuid.New()
	req := &CreateFamilyRequest{
		Name: "The Smiths",
	}

	repo.On("GetFamilyByOwner", ctx, ownerID).Return(nil, pgx.ErrNoRows)
	repo.On("GetFamilyForUser", ctx, ownerID).Return(nil, nil, nil)
	repo.On("CreateFamily", ctx, mock.AnythingOfType("*family.FamilyAccount")).Return(errors.New("db error"))

	result, err := svc.CreateFamily(ctx, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create family")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: AuthorizeRide (CRITICAL - Budget Enforcement)
// ========================================

func TestAuthorizeRide_UserNotInFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Authorized)
	assert.False(t, result.UseSharedPayment)
	assert.Contains(t, result.Reason, "not in a family account")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_PersonalPayment(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(500),
		CurrentSpend:  100,
	}
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: false, // Uses personal payment
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Authorized)
	assert.False(t, result.UseSharedPayment)
	assert.Contains(t, result.Reason, "personal payment")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_WithinFamilyBudget(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	memberID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(500),
		CurrentSpend:  100,
	}
	member := &FamilyMember{
		ID:               memberID,
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		RideApprovalReq:  false,
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	assert.Equal(t, &familyID, result.FamilyID)
	assert.Equal(t, &memberID, result.MemberID)
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_FamilyBudgetExceeded(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(500),
		CurrentSpend:  480, // Already spent 480
	}
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	// Request 50, but only 20 remaining (500 - 480 = 20)
	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	assert.Contains(t, result.Reason, "monthly budget would be exceeded")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_FamilyBudgetExactlyAtLimit(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	memberID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(500),
		CurrentSpend:  450,
	}
	member := &FamilyMember{
		ID:               memberID,
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		RideApprovalReq:  false,
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	// Exactly at limit: 450 + 50 = 500
	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_NoBudgetSet(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	memberID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: nil, // No budget limit
		CurrentSpend:  1000,
	}
	member := &FamilyMember{
		ID:               memberID,
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		RideApprovalReq:  false,
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 500.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_MaxFarePerRideExceeded(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(1000),
		CurrentSpend:  0,
	}
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		MaxFarePerRide:   ptrFloat64(30), // Max 30 per ride
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 50.0) // Requesting 50

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	assert.Contains(t, result.Reason, "fare exceeds per-ride limit")
	assert.Contains(t, result.Reason, "30.00")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_MemberMonthlyLimitExceeded(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(1000),
		CurrentSpend:  200,
	}
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		MonthlyLimit:     ptrFloat64(100), // Member limit 100
		MonthlySpend:     80,              // Already spent 80
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	// 80 + 50 = 130 > 100 limit
	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	assert.Contains(t, result.Reason, "member monthly limit would be exceeded")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_RequiresApproval(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(1000),
		CurrentSpend:  0,
	}
	member := &FamilyMember{
		ID:               uuid.New(),
		FamilyID:         familyID,
		UserID:           userID,
		UseSharedPayment: true,
		RideApprovalReq:  true, // Requires approval
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	result, err := svc.AuthorizeRide(ctx, userID, 30.0)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Authorized)
	assert.True(t, result.UseSharedPayment)
	assert.True(t, result.NeedsApproval)
	assert.Contains(t, result.Reason, "requires owner approval")
	repo.AssertExpectations(t)
}

func TestAuthorizeRide_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, errors.New("db error"))

	result, err := svc.AuthorizeRide(ctx, userID, 50.0)

	require.Error(t, err)
	assert.Nil(t, result)
	repo.AssertExpectations(t)
}

// Table-driven test for budget edge cases
func TestAuthorizeRide_BudgetEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		monthlyBudget  *float64
		currentSpend   float64
		fareAmount     float64
		wantAuthorized bool
	}{
		{
			name:           "exactly at budget limit",
			monthlyBudget:  ptrFloat64(100),
			currentSpend:   50,
			fareAmount:     50,
			wantAuthorized: true,
		},
		{
			name:           "one cent over budget",
			monthlyBudget:  ptrFloat64(100),
			currentSpend:   50,
			fareAmount:     50.01,
			wantAuthorized: false,
		},
		{
			name:           "zero budget remaining",
			monthlyBudget:  ptrFloat64(100),
			currentSpend:   100,
			fareAmount:     0.01,
			wantAuthorized: false,
		},
		{
			name:           "zero fare amount",
			monthlyBudget:  ptrFloat64(100),
			currentSpend:   100,
			fareAmount:     0,
			wantAuthorized: true,
		},
		{
			name:           "no budget set - large fare",
			monthlyBudget:  nil,
			currentSpend:   0,
			fareAmount:     10000,
			wantAuthorized: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := NewService(repo)
			ctx := context.Background()

			userID := uuid.New()
			familyID := uuid.New()
			memberID := uuid.New()
			family := &FamilyAccount{
				ID:            familyID,
				MonthlyBudget: tt.monthlyBudget,
				CurrentSpend:  tt.currentSpend,
			}
			member := &FamilyMember{
				ID:               memberID,
				FamilyID:         familyID,
				UserID:           userID,
				UseSharedPayment: true,
				RideApprovalReq:  false,
			}

			repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

			result, err := svc.AuthorizeRide(ctx, userID, tt.fareAmount)

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.Equal(t, tt.wantAuthorized, result.Authorized)
		})
	}
}

// ========================================
// TESTS: RecordFamilyRide
// ========================================

func TestRecordFamilyRide_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	rideID := uuid.New()
	fareAmount := 45.50

	repo.On("LogRide", ctx, mock.AnythingOfType("*family.FamilyRideLog")).Return(nil)
	repo.On("IncrementSpend", ctx, familyID, fareAmount).Return(nil)
	repo.On("IncrementMemberSpend", ctx, memberID, fareAmount).Return(nil)

	err := svc.RecordFamilyRide(ctx, familyID, memberID, rideID, fareAmount)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRecordFamilyRide_LogRideError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	rideID := uuid.New()
	fareAmount := 45.50

	repo.On("LogRide", ctx, mock.AnythingOfType("*family.FamilyRideLog")).Return(errors.New("db error"))

	err := svc.RecordFamilyRide(ctx, familyID, memberID, rideID, fareAmount)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "log ride")
	repo.AssertExpectations(t)
}

func TestRecordFamilyRide_IncrementSpendError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	rideID := uuid.New()
	fareAmount := 45.50

	repo.On("LogRide", ctx, mock.AnythingOfType("*family.FamilyRideLog")).Return(nil)
	repo.On("IncrementSpend", ctx, familyID, fareAmount).Return(errors.New("db error"))

	err := svc.RecordFamilyRide(ctx, familyID, memberID, rideID, fareAmount)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "increment family spend")
	repo.AssertExpectations(t)
}

func TestRecordFamilyRide_IncrementMemberSpendError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	rideID := uuid.New()
	fareAmount := 45.50

	repo.On("LogRide", ctx, mock.AnythingOfType("*family.FamilyRideLog")).Return(nil)
	repo.On("IncrementSpend", ctx, familyID, fareAmount).Return(nil)
	repo.On("IncrementMemberSpend", ctx, memberID, fareAmount).Return(errors.New("db error"))

	err := svc.RecordFamilyRide(ctx, familyID, memberID, rideID, fareAmount)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "increment member spend")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: RespondToInvite (State Transitions)
// ========================================

func TestRespondToInvite_Accept_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  familyID,
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil) // User not in any family
	repo.On("GetMembersByFamily", ctx, familyID).Return([]FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}, nil) // 1 member (owner)
	repo.On("AddMember", ctx, mock.AnythingOfType("*family.FamilyMember")).Return(nil)
	repo.On("UpdateInviteStatus", ctx, inviteID, InviteStatusAccepted).Return(nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRespondToInvite_Decline_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  uuid.New(),
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: false}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("UpdateInviteStatus", ctx, inviteID, InviteStatusDeclined).Return(nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRespondToInvite_InviteNotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(nil, pgx.ErrNoRows)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invite not found")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_InviteAlreadyAccepted(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  uuid.New(),
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusAccepted, // Already accepted
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer pending")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_InviteExpired(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  uuid.New(),
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(-24 * time.Hour), // Expired
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("UpdateInviteStatus", ctx, inviteID, InviteStatusExpired).Return(nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invite has expired")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_InviteNotForUser(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  uuid.New(),
		InviterID: uuid.New(),
		InviteeID: &otherUserID, // Different user
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_UserAlreadyInFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  familyID,
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	existingFamily := &FamilyAccount{
		ID:   uuid.New(),
		Name: "Existing Family",
	}
	existingMember := &FamilyMember{
		ID:     uuid.New(),
		UserID: userID,
	}

	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("GetFamilyForUser", ctx, userID).Return(existingFamily, existingMember, nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflict")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_FamilyAtMaxMembers(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  familyID,
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleAdult,
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	// Create 10 members (max)
	members := make([]FamilyMember, maxFamilyMembers)
	for i := 0; i < maxFamilyMembers; i++ {
		members[i] = FamilyMember{ID: uuid.New()}
	}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "maximum members")
	repo.AssertExpectations(t)
}

func TestRespondToInvite_TeenRoleRequiresApproval(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  familyID,
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleTeen, // Teen role
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return([]FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}, nil)
	repo.On("AddMember", ctx, mock.MatchedBy(func(m *FamilyMember) bool {
		return m.Role == MemberRoleTeen && m.RideApprovalReq == true
	})).Return(nil)
	repo.On("UpdateInviteStatus", ctx, inviteID, InviteStatusAccepted).Return(nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRespondToInvite_ChildRoleRequiresApproval(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	inviteID := uuid.New()
	userID := uuid.New()
	familyID := uuid.New()

	invite := &FamilyInvite{
		ID:        inviteID,
		FamilyID:  familyID,
		InviterID: uuid.New(),
		InviteeID: &userID,
		Role:      MemberRoleChild, // Child role
		Status:    InviteStatusPending,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	req := &RespondToInviteRequest{Accept: true}

	repo.On("GetInviteByID", ctx, inviteID).Return(invite, nil)
	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return([]FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}, nil)
	repo.On("AddMember", ctx, mock.MatchedBy(func(m *FamilyMember) bool {
		return m.Role == MemberRoleChild && m.RideApprovalReq == true
	})).Return(nil)
	repo.On("UpdateInviteStatus", ctx, inviteID, InviteStatusAccepted).Return(nil)

	err := svc.RespondToInvite(ctx, inviteID, userID, req)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: UpdateMember (Access Control)
// ========================================

func TestUpdateMember_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	ownerID := uuid.New()
	memberUserID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), UserID: ownerID, Role: MemberRoleOwner},
		{ID: memberID, UserID: memberUserID, Role: MemberRoleAdult, UseSharedPayment: true},
	}

	falseVal := false
	req := &UpdateMemberRequest{
		UseSharedPayment: &falseVal,
		MaxFarePerRide:   ptrFloat64(50),
		MonthlyLimit:     ptrFloat64(200),
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)
	repo.On("UpdateMember", ctx, mock.AnythingOfType("*family.FamilyMember")).Return(nil)

	result, err := svc.UpdateMember(ctx, familyID, memberID, ownerID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, false, result.UseSharedPayment)
	assert.Equal(t, ptrFloat64(50), result.MaxFarePerRide)
	assert.Equal(t, ptrFloat64(200), result.MonthlyLimit)
	repo.AssertExpectations(t)
}

func TestUpdateMember_NotOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	ownerID := uuid.New()
	notOwnerID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	req := &UpdateMemberRequest{}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)

	result, err := svc.UpdateMember(ctx, familyID, memberID, notOwnerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

func TestUpdateMember_FamilyNotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	ownerID := uuid.New()

	req := &UpdateMemberRequest{}

	repo.On("GetFamilyByID", ctx, familyID).Return(nil, pgx.ErrNoRows)

	result, err := svc.UpdateMember(ctx, familyID, memberID, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "family not found")
	repo.AssertExpectations(t)
}

func TestUpdateMember_MemberNotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	ownerID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), UserID: ownerID, Role: MemberRoleOwner},
	}

	req := &UpdateMemberRequest{}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	result, err := svc.UpdateMember(ctx, familyID, memberID, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "member not found")
	repo.AssertExpectations(t)
}

func TestUpdateMember_CannotModifyOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerMemberID := uuid.New()
	ownerID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: ownerMemberID, UserID: ownerID, Role: MemberRoleOwner},
	}

	req := &UpdateMemberRequest{
		MonthlyLimit: ptrFloat64(100),
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	result, err := svc.UpdateMember(ctx, familyID, ownerMemberID, ownerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "cannot modify the owner")
	repo.AssertExpectations(t)
}

func TestUpdateMember_CannotSetRoleToOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	memberID := uuid.New()
	ownerID := uuid.New()
	memberUserID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), UserID: ownerID, Role: MemberRoleOwner},
		{ID: memberID, UserID: memberUserID, Role: MemberRoleAdult},
	}

	ownerRole := MemberRoleOwner
	req := &UpdateMemberRequest{
		Role: &ownerRole, // Trying to set to owner
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)
	repo.On("UpdateMember", ctx, mock.MatchedBy(func(m *FamilyMember) bool {
		return m.Role == MemberRoleAdult // Role should NOT change to owner
	})).Return(nil)

	result, err := svc.UpdateMember(ctx, familyID, memberID, ownerID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, MemberRoleAdult, result.Role) // Remains adult
	repo.AssertExpectations(t)
}

func TestUpdateMember_AllowedHoursValidation(t *testing.T) {
	tests := []struct {
		name        string
		hoursStart  *int
		hoursEnd    *int
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid hours",
			hoursStart:  ptrInt(8),
			hoursEnd:    ptrInt(18),
			expectError: false,
		},
		{
			name:        "start hour too high",
			hoursStart:  ptrInt(24),
			hoursEnd:    ptrInt(18),
			expectError: true,
			errorMsg:    "allowed_hours_start must be 0-23",
		},
		{
			name:        "start hour negative",
			hoursStart:  ptrInt(-1),
			hoursEnd:    ptrInt(18),
			expectError: true,
			errorMsg:    "allowed_hours_start must be 0-23",
		},
		{
			name:        "end hour too high",
			hoursStart:  ptrInt(8),
			hoursEnd:    ptrInt(24),
			expectError: true,
			errorMsg:    "allowed_hours_end must be 0-23",
		},
		{
			name:        "end hour negative",
			hoursStart:  ptrInt(8),
			hoursEnd:    ptrInt(-1),
			expectError: true,
			errorMsg:    "allowed_hours_end must be 0-23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockRepo)
			svc := NewService(repo)
			ctx := context.Background()

			familyID := uuid.New()
			memberID := uuid.New()
			ownerID := uuid.New()

			family := &FamilyAccount{
				ID:      familyID,
				OwnerID: ownerID,
			}

			members := []FamilyMember{
				{ID: uuid.New(), UserID: ownerID, Role: MemberRoleOwner},
				{ID: memberID, UserID: uuid.New(), Role: MemberRoleAdult},
			}

			req := &UpdateMemberRequest{
				AllowedHoursStart: tt.hoursStart,
				AllowedHoursEnd:   tt.hoursEnd,
			}

			repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
			repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

			if !tt.expectError {
				repo.On("UpdateMember", ctx, mock.AnythingOfType("*family.FamilyMember")).Return(nil)
			}

			result, err := svc.UpdateMember(ctx, familyID, memberID, ownerID, req)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, result)
			}
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetSpendingReport (Financial Reporting)
// ========================================

func TestGetSpendingReport_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: ptrFloat64(500),
		CurrentSpend:  250,
	}

	member := &FamilyMember{
		ID:       uuid.New(),
		FamilyID: familyID,
		UserID:   requesterID,
		Role:     MemberRoleOwner,
	}

	memberSpending := []MemberSpend{
		{MemberID: uuid.New(), DisplayName: "Owner", Amount: 150, RideCount: 5},
		{MemberID: uuid.New(), DisplayName: "Spouse", Amount: 100, RideCount: 3},
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMemberByUserID", ctx, familyID, requesterID).Return(member, nil)
	repo.On("GetSpendingReport", ctx, familyID).Return(memberSpending, 8, 250.0, nil)

	report, err := svc.GetSpendingReport(ctx, familyID, requesterID)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, familyID, report.FamilyID)
	assert.Equal(t, 250.0, report.TotalSpend)
	assert.Equal(t, ptrFloat64(500), report.Budget)
	assert.Equal(t, 8, report.RideCount)
	assert.Len(t, report.MemberSpending, 2)
	require.NotNil(t, report.BudgetUsedPct)
	assert.Equal(t, 50.0, *report.BudgetUsedPct) // 250/500 * 100
	repo.AssertExpectations(t)
}

func TestGetSpendingReport_NoBudget(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	family := &FamilyAccount{
		ID:            familyID,
		MonthlyBudget: nil, // No budget set
		CurrentSpend:  250,
	}

	member := &FamilyMember{
		ID:       uuid.New(),
		FamilyID: familyID,
		UserID:   requesterID,
	}

	memberSpending := []MemberSpend{}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMemberByUserID", ctx, familyID, requesterID).Return(member, nil)
	repo.On("GetSpendingReport", ctx, familyID).Return(memberSpending, 0, 250.0, nil)

	report, err := svc.GetSpendingReport(ctx, familyID, requesterID)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Nil(t, report.BudgetUsedPct) // No percentage when no budget
	repo.AssertExpectations(t)
}

func TestGetSpendingReport_NotAMember(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: uuid.New(),
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMemberByUserID", ctx, familyID, requesterID).Return(nil, pgx.ErrNoRows)

	report, err := svc.GetSpendingReport(ctx, familyID, requesterID)

	require.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

func TestGetSpendingReport_FamilyNotFound(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	repo.On("GetFamilyByID", ctx, familyID).Return(nil, pgx.ErrNoRows)

	report, err := svc.GetSpendingReport(ctx, familyID, requesterID)

	require.Error(t, err)
	assert.Nil(t, report)
	assert.Contains(t, err.Error(), "family not found")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: LeaveFamily (State Management)
// ========================================

func TestLeaveFamily_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	memberID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: uuid.New(), // Different owner
	}

	member := &FamilyMember{
		ID:       memberID,
		FamilyID: familyID,
		UserID:   userID,
		Role:     MemberRoleAdult, // Not owner
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)
	repo.On("RemoveMember", ctx, memberID).Return(nil)

	err := svc.LeaveFamily(ctx, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestLeaveFamily_NotInFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)

	err := svc.LeaveFamily(ctx, userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not in any family")
	repo.AssertExpectations(t)
}

func TestLeaveFamily_OwnerCannotLeave(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()
	memberID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: userID, // User is the owner
	}

	member := &FamilyMember{
		ID:       memberID,
		FamilyID: familyID,
		UserID:   userID,
		Role:     MemberRoleOwner, // Owner role
	}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)

	err := svc.LeaveFamily(ctx, userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "owner cannot leave")
	repo.AssertExpectations(t)
}

func TestLeaveFamily_RepoError(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, errors.New("db error"))

	err := svc.LeaveFamily(ctx, userID)

	require.Error(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: InviteMember
// ========================================

func TestInviteMember_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	inviterID := uuid.New()
	inviteeID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: inviterID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}

	req := &InviteMemberRequest{
		UserID: &inviteeID,
		Role:   MemberRoleAdult,
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)
	repo.On("GetMemberByUserID", ctx, familyID, inviteeID).Return(nil, pgx.ErrNoRows)
	repo.On("CreateInvite", ctx, mock.AnythingOfType("*family.FamilyInvite")).Return(nil)

	invite, err := svc.InviteMember(ctx, familyID, inviterID, req)

	require.NoError(t, err)
	require.NotNil(t, invite)
	assert.Equal(t, familyID, invite.FamilyID)
	assert.Equal(t, inviterID, invite.InviterID)
	assert.Equal(t, &inviteeID, invite.InviteeID)
	assert.Equal(t, MemberRoleAdult, invite.Role)
	assert.Equal(t, InviteStatusPending, invite.Status)
	repo.AssertExpectations(t)
}

func TestInviteMember_NotOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerID := uuid.New()
	notOwnerID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	req := &InviteMemberRequest{
		Email: ptrString("test@example.com"),
		Role:  MemberRoleAdult,
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)

	invite, err := svc.InviteMember(ctx, familyID, notOwnerID, req)

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

func TestInviteMember_FamilyAtMaxMembers(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	inviterID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: inviterID,
	}

	// Create 10 members (max)
	members := make([]FamilyMember, maxFamilyMembers)
	for i := 0; i < maxFamilyMembers; i++ {
		members[i] = FamilyMember{ID: uuid.New()}
	}

	req := &InviteMemberRequest{
		Email: ptrString("test@example.com"),
		Role:  MemberRoleAdult,
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	invite, err := svc.InviteMember(ctx, familyID, inviterID, req)

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.Contains(t, err.Error(), "at most")
	repo.AssertExpectations(t)
}

func TestInviteMember_NoContactInfo(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	inviterID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: inviterID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}

	req := &InviteMemberRequest{
		Role: MemberRoleAdult,
		// No email, phone, or user_id
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	invite, err := svc.InviteMember(ctx, familyID, inviterID, req)

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.Contains(t, err.Error(), "must provide email, phone, or user_id")
	repo.AssertExpectations(t)
}

func TestInviteMember_CannotInviteAsOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	inviterID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: inviterID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}

	req := &InviteMemberRequest{
		Email: ptrString("test@example.com"),
		Role:  MemberRoleOwner, // Trying to invite as owner
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	invite, err := svc.InviteMember(ctx, familyID, inviterID, req)

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.Contains(t, err.Error(), "cannot invite someone as owner")
	repo.AssertExpectations(t)
}

func TestInviteMember_UserAlreadyMember(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	inviterID := uuid.New()
	existingMemberID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: inviterID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), Role: MemberRoleOwner},
	}

	existingMember := &FamilyMember{
		ID:     uuid.New(),
		UserID: existingMemberID,
	}

	req := &InviteMemberRequest{
		UserID: &existingMemberID,
		Role:   MemberRoleAdult,
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)
	repo.On("GetMemberByUserID", ctx, familyID, existingMemberID).Return(existingMember, nil)

	invite, err := svc.InviteMember(ctx, familyID, inviterID, req)

	require.Error(t, err)
	assert.Nil(t, invite)
	assert.Contains(t, err.Error(), "conflict")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: GetFamily
// ========================================

func TestGetFamily_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	family := &FamilyAccount{
		ID:            familyID,
		OwnerID:       requesterID,
		Name:          "The Smiths",
		MonthlyBudget: ptrFloat64(500),
		Currency:      "USD",
	}

	member := &FamilyMember{
		ID:       uuid.New(),
		FamilyID: familyID,
		UserID:   requesterID,
		Role:     MemberRoleOwner,
	}

	members := []FamilyMember{*member}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMemberByUserID", ctx, familyID, requesterID).Return(member, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	result, err := svc.GetFamily(ctx, familyID, requesterID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "The Smiths", result.Name)
	assert.Len(t, result.Members, 1)
	repo.AssertExpectations(t)
}

func TestGetFamily_NotAMember(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	requesterID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: uuid.New(),
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMemberByUserID", ctx, familyID, requesterID).Return(nil, pgx.ErrNoRows)

	result, err := svc.GetFamily(ctx, familyID, requesterID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: GetMyFamily
// ========================================

func TestGetMyFamily_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()
	familyID := uuid.New()

	family := &FamilyAccount{
		ID:   familyID,
		Name: "My Family",
	}

	member := &FamilyMember{
		ID:       uuid.New(),
		FamilyID: familyID,
		UserID:   userID,
	}

	members := []FamilyMember{*member}

	repo.On("GetFamilyForUser", ctx, userID).Return(family, member, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	result, err := svc.GetMyFamily(ctx, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "My Family", result.Name)
	repo.AssertExpectations(t)
}

func TestGetMyFamily_NotInFamily(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetFamilyForUser", ctx, userID).Return(nil, nil, nil)

	result, err := svc.GetMyFamily(ctx, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not in any family")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: UpdateFamily
// ========================================

func TestUpdateFamily_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerID := uuid.New()

	family := &FamilyAccount{
		ID:            familyID,
		OwnerID:       ownerID,
		Name:          "Old Name",
		MonthlyBudget: ptrFloat64(500),
	}

	newName := "New Name"
	newBudget := 1000.0
	req := &UpdateFamilyRequest{
		Name:          &newName,
		MonthlyBudget: &newBudget,
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("UpdateFamily", ctx, mock.AnythingOfType("*family.FamilyAccount")).Return(nil)

	result, err := svc.UpdateFamily(ctx, familyID, ownerID, req)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "New Name", result.Name)
	assert.Equal(t, ptrFloat64(1000), result.MonthlyBudget)
	repo.AssertExpectations(t)
}

func TestUpdateFamily_NotOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerID := uuid.New()
	notOwnerID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	req := &UpdateFamilyRequest{}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)

	result, err := svc.UpdateFamily(ctx, familyID, notOwnerID, req)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "forbidden")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: RemoveMember
// ========================================

func TestRemoveMember_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerID := uuid.New()
	memberID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: uuid.New(), UserID: ownerID, Role: MemberRoleOwner},
		{ID: memberID, UserID: uuid.New(), Role: MemberRoleAdult},
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)
	repo.On("RemoveMember", ctx, memberID).Return(nil)

	err := svc.RemoveMember(ctx, familyID, memberID, ownerID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	familyID := uuid.New()
	ownerID := uuid.New()
	ownerMemberID := uuid.New()

	family := &FamilyAccount{
		ID:      familyID,
		OwnerID: ownerID,
	}

	members := []FamilyMember{
		{ID: ownerMemberID, UserID: ownerID, Role: MemberRoleOwner},
	}

	repo.On("GetFamilyByID", ctx, familyID).Return(family, nil)
	repo.On("GetMembersByFamily", ctx, familyID).Return(members, nil)

	err := svc.RemoveMember(ctx, familyID, ownerMemberID, ownerID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot remove the family owner")
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: GetPendingInvites
// ========================================

func TestGetPendingInvites_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	invites := []FamilyInvite{
		{ID: uuid.New(), FamilyID: uuid.New(), Status: InviteStatusPending},
		{ID: uuid.New(), FamilyID: uuid.New(), Status: InviteStatusPending},
	}

	repo.On("GetPendingInvitesForUser", ctx, userID, (*string)(nil)).Return(invites, nil)

	result, err := svc.GetPendingInvites(ctx, userID)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

func TestGetPendingInvites_Empty(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	userID := uuid.New()

	repo.On("GetPendingInvitesForUser", ctx, userID, (*string)(nil)).Return(nil, nil)

	result, err := svc.GetPendingInvites(ctx, userID)

	require.NoError(t, err)
	assert.Empty(t, result)
	repo.AssertExpectations(t)
}

// ========================================
// TESTS: ResetMonthlySpend
// ========================================

func TestResetMonthlySpend_Success(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	dayOfMonth := 1

	repo.On("ResetMonthlySpend", ctx, dayOfMonth).Return(nil)

	err := svc.ResetMonthlySpend(ctx, dayOfMonth)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestResetMonthlySpend_Error(t *testing.T) {
	repo := new(mockRepo)
	svc := NewService(repo)
	ctx := context.Background()

	dayOfMonth := 1

	repo.On("ResetMonthlySpend", ctx, dayOfMonth).Return(errors.New("db error"))

	err := svc.ResetMonthlySpend(ctx, dayOfMonth)

	require.Error(t, err)
	repo.AssertExpectations(t)
}
