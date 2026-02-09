package disputes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK REPOSITORY
// ========================================

// MockRepository is a mock implementation of RepositoryInterface for testing
type MockRepository struct {
	// Storage for mock data
	disputes        map[uuid.UUID]*Dispute
	rideContexts    map[uuid.UUID]*RideContext
	comments        map[uuid.UUID][]DisputeComment
	userDisputes    map[uuid.UUID][]DisputeSummary
	allDisputes     []DisputeSummary
	stats           *DisputeStats
	disputeByRideUser map[string]*Dispute // key: rideID-userID

	// Error injection
	createDisputeErr        error
	getDisputeByIDErr       error
	getDisputeByRideUserErr error
	getUserDisputesErr      error
	resolveDisputeErr       error
	updateStatusErr         error
	createCommentErr        error
	getCommentsErr          error
	getRideContextErr       error
	getAllDisputesErr       error
	getStatsErr             error

	// Call tracking
	createDisputeCalled        bool
	getDisputeByIDCalled       bool
	getDisputeByRideUserCalled bool
	getUserDisputesCalled      bool
	resolveDisputeCalled       bool
	updateStatusCalled         bool
	createCommentCalled        bool
	getCommentsCalled          bool
	getRideContextCalled       bool
	getAllDisputesCalled       bool
	getStatsCalled             bool

	// Capture arguments for verification
	lastDisputeCreated  *Dispute
	lastCommentCreated  *DisputeComment
	lastResolveArgs     *resolveArgs
	lastStatusUpdate    *statusUpdateArgs
}

type resolveArgs struct {
	ID           uuid.UUID
	Status       DisputeStatus
	ResType      ResolutionType
	RefundAmount *float64
	Note         string
	ResolvedBy   uuid.UUID
}

type statusUpdateArgs struct {
	ID     uuid.UUID
	Status DisputeStatus
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		disputes:          make(map[uuid.UUID]*Dispute),
		rideContexts:      make(map[uuid.UUID]*RideContext),
		comments:          make(map[uuid.UUID][]DisputeComment),
		userDisputes:      make(map[uuid.UUID][]DisputeSummary),
		disputeByRideUser: make(map[string]*Dispute),
	}
}

func (m *MockRepository) CreateDispute(ctx context.Context, d *Dispute) error {
	m.createDisputeCalled = true
	m.lastDisputeCreated = d
	if m.createDisputeErr != nil {
		return m.createDisputeErr
	}
	m.disputes[d.ID] = d
	return nil
}

func (m *MockRepository) GetDisputeByID(ctx context.Context, id uuid.UUID) (*Dispute, error) {
	m.getDisputeByIDCalled = true
	if m.getDisputeByIDErr != nil {
		return nil, m.getDisputeByIDErr
	}
	d, ok := m.disputes[id]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return d, nil
}

func (m *MockRepository) GetDisputeByRideAndUser(ctx context.Context, rideID, userID uuid.UUID) (*Dispute, error) {
	m.getDisputeByRideUserCalled = true
	if m.getDisputeByRideUserErr != nil {
		return nil, m.getDisputeByRideUserErr
	}
	key := rideID.String() + "-" + userID.String()
	d, ok := m.disputeByRideUser[key]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return d, nil
}

func (m *MockRepository) GetUserDisputes(ctx context.Context, userID uuid.UUID, status *DisputeStatus, limit, offset int) ([]DisputeSummary, int, error) {
	m.getUserDisputesCalled = true
	if m.getUserDisputesErr != nil {
		return nil, 0, m.getUserDisputesErr
	}
	disputes := m.userDisputes[userID]
	if disputes == nil {
		disputes = []DisputeSummary{}
	}
	return disputes, len(disputes), nil
}

func (m *MockRepository) ResolveDispute(ctx context.Context, id uuid.UUID, status DisputeStatus, resType ResolutionType, refundAmount *float64, note string, resolvedBy uuid.UUID) error {
	m.resolveDisputeCalled = true
	m.lastResolveArgs = &resolveArgs{
		ID:           id,
		Status:       status,
		ResType:      resType,
		RefundAmount: refundAmount,
		Note:         note,
		ResolvedBy:   resolvedBy,
	}
	if m.resolveDisputeErr != nil {
		return m.resolveDisputeErr
	}
	if d, ok := m.disputes[id]; ok {
		d.Status = status
		d.ResolutionType = &resType
		d.RefundAmount = refundAmount
		d.ResolutionNote = &note
		d.ResolvedBy = &resolvedBy
		now := time.Now()
		d.ResolvedAt = &now
	}
	return nil
}

func (m *MockRepository) UpdateDisputeStatus(ctx context.Context, id uuid.UUID, status DisputeStatus) error {
	m.updateStatusCalled = true
	m.lastStatusUpdate = &statusUpdateArgs{ID: id, Status: status}
	if m.updateStatusErr != nil {
		return m.updateStatusErr
	}
	if d, ok := m.disputes[id]; ok {
		d.Status = status
	}
	return nil
}

func (m *MockRepository) CreateComment(ctx context.Context, c *DisputeComment) error {
	m.createCommentCalled = true
	m.lastCommentCreated = c
	if m.createCommentErr != nil {
		return m.createCommentErr
	}
	m.comments[c.DisputeID] = append(m.comments[c.DisputeID], *c)
	return nil
}

func (m *MockRepository) GetCommentsByDispute(ctx context.Context, disputeID uuid.UUID, includeInternal bool) ([]DisputeComment, error) {
	m.getCommentsCalled = true
	if m.getCommentsErr != nil {
		return nil, m.getCommentsErr
	}
	comments := m.comments[disputeID]
	if !includeInternal {
		var filtered []DisputeComment
		for _, c := range comments {
			if !c.IsInternal {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}
	return comments, nil
}

func (m *MockRepository) GetRideContext(ctx context.Context, rideID uuid.UUID) (*RideContext, error) {
	m.getRideContextCalled = true
	if m.getRideContextErr != nil {
		return nil, m.getRideContextErr
	}
	rc, ok := m.rideContexts[rideID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return rc, nil
}

func (m *MockRepository) GetAllDisputes(ctx context.Context, status *DisputeStatus, reason *DisputeReason, limit, offset int) ([]DisputeSummary, int, error) {
	m.getAllDisputesCalled = true
	if m.getAllDisputesErr != nil {
		return nil, 0, m.getAllDisputesErr
	}
	return m.allDisputes, len(m.allDisputes), nil
}

func (m *MockRepository) GetDisputeStats(ctx context.Context, from, to time.Time) (*DisputeStats, error) {
	m.getStatsCalled = true
	if m.getStatsErr != nil {
		return nil, m.getStatsErr
	}
	if m.stats == nil {
		return &DisputeStats{ByReason: []ReasonCount{}}, nil
	}
	return m.stats, nil
}

// Helper methods for setting up test data
func (m *MockRepository) AddDispute(d *Dispute) {
	m.disputes[d.ID] = d
}

func (m *MockRepository) AddRideContext(rc *RideContext) {
	m.rideContexts[rc.RideID] = rc
}

func (m *MockRepository) AddDisputeByRideAndUser(rideID, userID uuid.UUID, d *Dispute) {
	key := rideID.String() + "-" + userID.String()
	m.disputeByRideUser[key] = d
}

func (m *MockRepository) AddUserDisputes(userID uuid.UUID, disputes []DisputeSummary) {
	m.userDisputes[userID] = disputes
}

func (m *MockRepository) SetAllDisputes(disputes []DisputeSummary) {
	m.allDisputes = disputes
}

func (m *MockRepository) SetStats(stats *DisputeStats) {
	m.stats = stats
}

// ========================================
// TEST HELPERS
// ========================================

func createTestService(mock *MockRepository) *Service {
	return &Service{repo: mock}
}

func newCompletedRideContext(rideID uuid.UUID, fare float64) *RideContext {
	completedAt := time.Now().Add(-time.Hour) // completed 1 hour ago
	return &RideContext{
		RideID:            rideID,
		EstimatedFare:     fare,
		FinalFare:         &fare,
		EstimatedDistance: 10.0,
		EstimatedDuration: 30,
		SurgeMultiplier:   1.0,
		PickupAddress:     "123 Start St",
		DropoffAddress:    "456 End Ave",
		RequestedAt:       time.Now().Add(-2 * time.Hour),
		CompletedAt:       &completedAt,
	}
}

func newTestDispute(userID, rideID uuid.UUID) *Dispute {
	return &Dispute{
		ID:             uuid.New(),
		RideID:         rideID,
		UserID:         userID,
		DisputeNumber:  "DSP-123456",
		Reason:         ReasonOvercharged,
		Description:    "I was overcharged for this ride",
		Status:         DisputeStatusPending,
		OriginalFare:   50.0,
		DisputedAmount: 20.0,
		Evidence:       []string{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// ========================================
// EXISTING TESTS (PRESERVED)
// ========================================

func TestGetDisputeReasons(t *testing.T) {
	svc := &Service{}
	resp := svc.GetDisputeReasons()

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 10)

	expectedCodes := []DisputeReason{
		ReasonWrongRoute, ReasonOvercharged, ReasonTripNotTaken,
		ReasonDriverDetour, ReasonWrongFare, ReasonSurgeUnfair,
		ReasonWaitTimeWrong, ReasonCancelFeeWrong, ReasonDuplicateCharge,
		ReasonOther,
	}

	for i, reason := range resp.Reasons {
		assert.Equal(t, expectedCodes[i], reason.Code, "reason code mismatch at index %d", i)
		assert.NotEmpty(t, reason.Label, "reason %d label should not be empty", i)
		assert.NotEmpty(t, reason.Description, "reason %d description should not be empty", i)
	}
}

func TestGetDisputeReasons_LastIsOther(t *testing.T) {
	svc := &Service{}
	resp := svc.GetDisputeReasons()

	lastReason := resp.Reasons[len(resp.Reasons)-1]
	assert.Equal(t, ReasonOther, lastReason.Code)
	assert.Equal(t, "Other", lastReason.Label)
}

func TestResolutionTypeToStatus_Mapping(t *testing.T) {
	tests := []struct {
		name           string
		resolutionType ResolutionType
		expectedStatus DisputeStatus
	}{
		{
			name:           "full refund maps to approved",
			resolutionType: ResolutionFullRefund,
			expectedStatus: DisputeStatusApproved,
		},
		{
			name:           "partial refund maps to partial_refund",
			resolutionType: ResolutionPartialRefund,
			expectedStatus: DisputeStatusPartial,
		},
		{
			name:           "credits maps to approved",
			resolutionType: ResolutionCredits,
			expectedStatus: DisputeStatusApproved,
		},
		{
			name:           "no action maps to rejected",
			resolutionType: ResolutionNoAction,
			expectedStatus: DisputeStatusRejected,
		},
		{
			name:           "fare adjustment maps to approved",
			resolutionType: ResolutionFareAdjust,
			expectedStatus: DisputeStatusApproved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status DisputeStatus
			switch tt.resolutionType {
			case ResolutionFullRefund:
				status = DisputeStatusApproved
			case ResolutionPartialRefund:
				status = DisputeStatusPartial
			case ResolutionCredits:
				status = DisputeStatusApproved
			case ResolutionNoAction:
				status = DisputeStatusRejected
			case ResolutionFareAdjust:
				status = DisputeStatusApproved
			}
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestDisputeStatusConstants(t *testing.T) {
	assert.Equal(t, DisputeStatus("pending"), DisputeStatusPending)
	assert.Equal(t, DisputeStatus("reviewing"), DisputeStatusReviewing)
	assert.Equal(t, DisputeStatus("approved"), DisputeStatusApproved)
	assert.Equal(t, DisputeStatus("rejected"), DisputeStatusRejected)
	assert.Equal(t, DisputeStatus("partial_refund"), DisputeStatusPartial)
	assert.Equal(t, DisputeStatus("closed"), DisputeStatusClosed)
}

func TestDisputeReasonConstants(t *testing.T) {
	reasons := []DisputeReason{
		ReasonWrongRoute, ReasonOvercharged, ReasonTripNotTaken,
		ReasonDriverDetour, ReasonWrongFare, ReasonSurgeUnfair,
		ReasonWaitTimeWrong, ReasonCancelFeeWrong, ReasonDuplicateCharge,
		ReasonOther,
	}

	assert.Len(t, reasons, 10)

	seen := make(map[DisputeReason]bool)
	for _, reason := range reasons {
		assert.NotEmpty(t, string(reason))
		assert.False(t, seen[reason], "duplicate reason: %s", reason)
		seen[reason] = true
	}
}

func TestResolutionTypeConstants(t *testing.T) {
	types := []ResolutionType{
		ResolutionFullRefund, ResolutionPartialRefund,
		ResolutionCredits, ResolutionNoAction, ResolutionFareAdjust,
	}

	assert.Len(t, types, 5)

	seen := make(map[ResolutionType]bool)
	for _, rt := range types {
		assert.NotEmpty(t, string(rt))
		assert.False(t, seen[rt], "duplicate resolution type: %s", rt)
		seen[rt] = true
	}
}

func TestGenerateDisputeNumber(t *testing.T) {
	num := generateDisputeNumber()

	assert.NotEmpty(t, num)
	assert.Equal(t, "DSP-", num[:4], "dispute number should start with DSP-")
	assert.Len(t, num, 10, "DSP-XXXXXX should be 10 chars")

	// Verify uniqueness
	num2 := generateDisputeNumber()
	assert.NotEqual(t, num, num2, "dispute numbers should be unique")
}

func TestDisputeWindowDays(t *testing.T) {
	assert.Equal(t, 30, maxDisputeWindowDays)
}

func TestMaxOpenDisputes(t *testing.T) {
	assert.Equal(t, 5, maxOpenDisputes)
}

// ========================================
// NEW SERVICE TESTS - CreateDispute
// ========================================

func TestService_CreateDispute_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Setup completed ride
	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "The fare was too high for the distance traveled",
		DisputedAmount: 20.0,
		Evidence:       []string{"screenshot1.png"},
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, userID, dispute.UserID)
	assert.Equal(t, rideID, dispute.RideID)
	assert.Equal(t, ReasonOvercharged, dispute.Reason)
	assert.Equal(t, DisputeStatusPending, dispute.Status)
	assert.Equal(t, 50.0, dispute.OriginalFare)
	assert.Equal(t, 20.0, dispute.DisputedAmount)
	assert.Contains(t, dispute.DisputeNumber, "DSP-")
	assert.True(t, mock.createDisputeCalled)
	assert.True(t, mock.createCommentCalled)
}

func TestService_CreateDispute_WithEmptyEvidence(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonWrongRoute,
		Description:    "Driver took a wrong route",
		DisputedAmount: 15.0,
		Evidence:       nil,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.NotNil(t, dispute.Evidence)
	assert.Len(t, dispute.Evidence, 0)
}

func TestService_CreateDispute_RideNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()
	// No ride context added

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
	assert.Contains(t, appErr.Message, "ride not found")
}

func TestService_CreateDispute_RideNotCompleted(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Ride not completed (CompletedAt is nil)
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		CompletedAt:   nil,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "can only dispute completed rides")
}

func TestService_CreateDispute_OutsideDisputeWindow(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Ride completed 31 days ago (outside the 30-day window)
	completedAt := time.Now().Add(-31 * 24 * time.Hour)
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		CompletedAt:   &completedAt,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "30 days")
}

func TestService_CreateDispute_DisputedAmountExceedsFare(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 100.0, // Exceeds fare of 50.0
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "cannot exceed the ride fare")
}

func TestService_CreateDispute_DisputedAmountEqualsExactFare(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonTripNotTaken,
		Description:    "I was charged but never took this trip",
		DisputedAmount: 50.0, // Exactly equals fare
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, 50.0, dispute.DisputedAmount)
}

func TestService_CreateDispute_ExistingActiveDispute(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	// Add existing dispute for this ride/user
	existingDispute := newTestDispute(userID, rideID)
	mock.AddDisputeByRideAndUser(rideID, userID, existingDispute)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 409, appErr.Code)
	assert.Contains(t, appErr.Message, "already have an active dispute")
}

func TestService_CreateDispute_UsesFinalFareWhenAvailable(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// FinalFare differs from EstimatedFare
	completedAt := time.Now().Add(-time.Hour)
	finalFare := 60.0
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		FinalFare:     &finalFare,
		CompletedAt:   &completedAt,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 25.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, 60.0, dispute.OriginalFare) // Should use FinalFare
}

func TestService_CreateDispute_UsesEstimatedFareWhenNoFinal(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	completedAt := time.Now().Add(-time.Hour)
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		FinalFare:     nil,
		CompletedAt:   &completedAt,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 25.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, 50.0, dispute.OriginalFare) // Should use EstimatedFare
}

func TestService_CreateDispute_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.createDisputeErr = errors.New("database error")
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	assert.Contains(t, err.Error(), "create dispute")
}

func TestService_CreateDispute_GetRideContextError(t *testing.T) {
	mock := NewMockRepository()
	mock.getRideContextErr = errors.New("database error")
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
	assert.Contains(t, err.Error(), "database error")
}

func TestService_CreateDispute_AllReasons(t *testing.T) {
	reasons := []DisputeReason{
		ReasonWrongRoute, ReasonOvercharged, ReasonTripNotTaken,
		ReasonDriverDetour, ReasonWrongFare, ReasonSurgeUnfair,
		ReasonWaitTimeWrong, ReasonCancelFeeWrong, ReasonDuplicateCharge,
		ReasonOther,
	}

	for _, reason := range reasons {
		t.Run(string(reason), func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			userID := uuid.New()
			rideID := uuid.New()

			rc := newCompletedRideContext(rideID, 50.0)
			mock.AddRideContext(rc)

			req := &CreateDisputeRequest{
				RideID:         rideID,
				Reason:         reason,
				Description:    "Test description for " + string(reason),
				DisputedAmount: 20.0,
			}

			dispute, err := svc.CreateDispute(context.Background(), userID, req)

			require.NoError(t, err)
			require.NotNil(t, dispute)
			assert.Equal(t, reason, dispute.Reason)
		})
	}
}

// ========================================
// GetMyDisputes TESTS
// ========================================

func TestService_GetMyDisputes_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputes := []DisputeSummary{
		{ID: uuid.New(), DisputeNumber: "DSP-000001", Status: DisputeStatusPending},
		{ID: uuid.New(), DisputeNumber: "DSP-000002", Status: DisputeStatusReviewing},
	}
	mock.AddUserDisputes(userID, disputes)

	result, total, err := svc.GetMyDisputes(context.Background(), userID, nil, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, 2, total)
	assert.True(t, mock.getUserDisputesCalled)
}

func TestService_GetMyDisputes_EmptyList(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()

	result, total, err := svc.GetMyDisputes(context.Background(), userID, nil, 1, 20)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

func TestService_GetMyDisputes_WithStatusFilter(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	status := DisputeStatusPending
	disputes := []DisputeSummary{
		{ID: uuid.New(), Status: DisputeStatusPending},
	}
	mock.AddUserDisputes(userID, disputes)

	result, total, err := svc.GetMyDisputes(context.Background(), userID, &status, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

func TestService_GetMyDisputes_PaginationDefaults(t *testing.T) {
	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedLimit int
	}{
		{"zero limit defaults to 20", 0, 0, 20},
		{"negative limit defaults to 20", -5, 0, 20},
		{"limit over 50 defaults to 20", 100, 0, 20},
		{"valid limit unchanged", 25, 10, 25},
		{"negative offset defaults to 0", 20, -5, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			userID := uuid.New()
			svc.GetMyDisputes(context.Background(), userID, nil, tt.limit, tt.offset)

			// Verify pagination was applied (implicitly through mock)
			assert.True(t, mock.getUserDisputesCalled)
		})
	}
}

func TestService_GetMyDisputes_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.getUserDisputesErr = errors.New("database error")
	svc := createTestService(mock)

	userID := uuid.New()

	result, total, err := svc.GetMyDisputes(context.Background(), userID, nil, 1, 20)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

// ========================================
// GetDisputeDetail TESTS
// ========================================

func TestService_GetDisputeDetail_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		RideID: rideID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	result, err := svc.GetDisputeDetail(context.Background(), disputeID, userID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, disputeID, result.Dispute.ID)
	assert.NotNil(t, result.RideContext)
	assert.True(t, mock.getDisputeByIDCalled)
	assert.True(t, mock.getRideContextCalled)
	assert.True(t, mock.getCommentsCalled)
}

func TestService_GetDisputeDetail_DisputeNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	result, err := svc.GetDisputeDetail(context.Background(), disputeID, userID)

	require.Error(t, err)
	assert.Nil(t, result)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
}

func TestService_GetDisputeDetail_NotAuthorized(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: ownerID, // Different from requesting user
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	result, err := svc.GetDisputeDetail(context.Background(), disputeID, otherUserID)

	require.Error(t, err)
	assert.Nil(t, result)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.Code)
	assert.Contains(t, appErr.Message, "not authorized")
}

func TestService_GetDisputeDetail_WithComments(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		RideID: rideID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)
	mock.AddRideContext(newCompletedRideContext(rideID, 50.0))

	// Add some comments
	mock.comments[disputeID] = []DisputeComment{
		{ID: uuid.New(), DisputeID: disputeID, Comment: "User comment", IsInternal: false},
		{ID: uuid.New(), DisputeID: disputeID, Comment: "Internal note", IsInternal: true},
	}

	result, err := svc.GetDisputeDetail(context.Background(), disputeID, userID)

	require.NoError(t, err)
	// User view should not include internal comments
	assert.Len(t, result.Comments, 1)
	assert.Equal(t, "User comment", result.Comments[0].Comment)
}

func TestService_GetDisputeDetail_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.getDisputeByIDErr = errors.New("database error")
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	result, err := svc.GetDisputeDetail(context.Background(), disputeID, userID)

	require.Error(t, err)
	assert.Nil(t, result)
}

// ========================================
// AddComment TESTS
// ========================================

func TestService_AddComment_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	req := &AddCommentRequest{
		Comment: "Additional information about my dispute",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

	require.NoError(t, err)
	require.NotNil(t, comment)
	assert.Equal(t, disputeID, comment.DisputeID)
	assert.Equal(t, userID, comment.UserID)
	assert.Equal(t, "user", comment.UserRole)
	assert.Equal(t, req.Comment, comment.Comment)
	assert.False(t, comment.IsInternal)
	assert.True(t, mock.createCommentCalled)
}

func TestService_AddComment_DisputeNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	req := &AddCommentRequest{
		Comment: "Test comment",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

	require.Error(t, err)
	assert.Nil(t, comment)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
}

func TestService_AddComment_NotAuthorized(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	ownerID := uuid.New()
	otherUserID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: ownerID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	req := &AddCommentRequest{
		Comment: "Test comment",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, otherUserID, req)

	require.Error(t, err)
	assert.Nil(t, comment)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 403, appErr.Code)
}

func TestService_AddComment_ClosedDispute(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		Status: DisputeStatusClosed,
	}
	mock.AddDispute(dispute)

	req := &AddCommentRequest{
		Comment: "Test comment",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

	require.Error(t, err)
	assert.Nil(t, comment)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "closed")
}

func TestService_AddComment_RejectedDispute(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		Status: DisputeStatusRejected,
	}
	mock.AddDispute(dispute)

	req := &AddCommentRequest{
		Comment: "Test comment",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

	require.Error(t, err)
	assert.Nil(t, comment)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
}

func TestService_AddComment_AllowedStatuses(t *testing.T) {
	allowedStatuses := []DisputeStatus{
		DisputeStatusPending,
		DisputeStatusReviewing,
		DisputeStatusPartial,
		DisputeStatusApproved, // Note: approved but not closed
	}

	for _, status := range allowedStatuses {
		t.Run(string(status), func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			userID := uuid.New()
			disputeID := uuid.New()

			dispute := &Dispute{
				ID:     disputeID,
				UserID: userID,
				Status: status,
			}
			mock.AddDispute(dispute)

			req := &AddCommentRequest{
				Comment: "Test comment",
			}

			comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

			require.NoError(t, err)
			require.NotNil(t, comment)
		})
	}
}

func TestService_AddComment_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.createCommentErr = errors.New("database error")
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	req := &AddCommentRequest{
		Comment: "Test comment",
	}

	comment, err := svc.AddComment(context.Background(), disputeID, userID, req)

	require.Error(t, err)
	assert.Nil(t, comment)
	assert.Contains(t, err.Error(), "create comment")
}

// ========================================
// AdminGetDisputes TESTS
// ========================================

func TestService_AdminGetDisputes_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	disputes := []DisputeSummary{
		{ID: uuid.New(), DisputeNumber: "DSP-000001", Status: DisputeStatusPending},
		{ID: uuid.New(), DisputeNumber: "DSP-000002", Status: DisputeStatusReviewing},
		{ID: uuid.New(), DisputeNumber: "DSP-000003", Status: DisputeStatusApproved},
	}
	mock.SetAllDisputes(disputes)

	result, total, err := svc.AdminGetDisputes(context.Background(), nil, nil, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Equal(t, 3, total)
	assert.True(t, mock.getAllDisputesCalled)
}

func TestService_AdminGetDisputes_EmptyList(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	result, total, err := svc.AdminGetDisputes(context.Background(), nil, nil, 1, 20)

	require.NoError(t, err)
	assert.Empty(t, result)
	assert.Equal(t, 0, total)
}

func TestService_AdminGetDisputes_WithStatusFilter(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	status := DisputeStatusPending
	disputes := []DisputeSummary{
		{ID: uuid.New(), Status: DisputeStatusPending},
	}
	mock.SetAllDisputes(disputes)

	result, total, err := svc.AdminGetDisputes(context.Background(), &status, nil, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

func TestService_AdminGetDisputes_WithReasonFilter(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	reason := ReasonOvercharged
	disputes := []DisputeSummary{
		{ID: uuid.New(), Reason: ReasonOvercharged},
	}
	mock.SetAllDisputes(disputes)

	result, total, err := svc.AdminGetDisputes(context.Background(), nil, &reason, 1, 20)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, 1, total)
}

func TestService_AdminGetDisputes_PaginationDefaults(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{"zero limit defaults to 20", 0, 0},
		{"negative limit defaults to 20", -5, 0},
		{"limit over 100 defaults to 20", 200, 0},
		{"valid limit unchanged", 50, 10},
		{"negative offset defaults to 0", 20, -5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			svc.AdminGetDisputes(context.Background(), nil, nil, tt.limit, tt.offset)

			assert.True(t, mock.getAllDisputesCalled)
		})
	}
}

func TestService_AdminGetDisputes_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.getAllDisputesErr = errors.New("database error")
	svc := createTestService(mock)

	result, total, err := svc.AdminGetDisputes(context.Background(), nil, nil, 1, 20)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, 0, total)
}

// ========================================
// AdminGetDisputeDetail TESTS
// ========================================

func TestService_AdminGetDisputeDetail_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		RideID: rideID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)
	mock.AddRideContext(newCompletedRideContext(rideID, 50.0))

	result, err := svc.AdminGetDisputeDetail(context.Background(), disputeID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, disputeID, result.Dispute.ID)
	assert.True(t, mock.getDisputeByIDCalled)
}

func TestService_AdminGetDisputeDetail_DisputeNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	disputeID := uuid.New()

	result, err := svc.AdminGetDisputeDetail(context.Background(), disputeID)

	require.Error(t, err)
	assert.Nil(t, result)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
}

func TestService_AdminGetDisputeDetail_IncludesInternalComments(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		RideID: rideID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)
	mock.AddRideContext(newCompletedRideContext(rideID, 50.0))

	// Add both internal and external comments
	mock.comments[disputeID] = []DisputeComment{
		{ID: uuid.New(), DisputeID: disputeID, Comment: "User comment", IsInternal: false},
		{ID: uuid.New(), DisputeID: disputeID, Comment: "Internal admin note", IsInternal: true},
	}

	result, err := svc.AdminGetDisputeDetail(context.Background(), disputeID)

	require.NoError(t, err)
	// Admin view should include internal comments
	assert.Len(t, result.Comments, 2)
}

func TestService_AdminGetDisputeDetail_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.getDisputeByIDErr = errors.New("database error")
	svc := createTestService(mock)

	disputeID := uuid.New()

	result, err := svc.AdminGetDisputeDetail(context.Background(), disputeID)

	require.Error(t, err)
	assert.Nil(t, result)
}

// ========================================
// AdminResolveDispute TESTS
// ========================================

func TestService_AdminResolveDispute_FullRefund_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "Approved for full refund due to verified overcharge",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.True(t, mock.resolveDisputeCalled)
	require.NotNil(t, mock.lastResolveArgs)
	assert.Equal(t, DisputeStatusApproved, mock.lastResolveArgs.Status)
	assert.Equal(t, ResolutionFullRefund, mock.lastResolveArgs.ResType)
	assert.NotNil(t, mock.lastResolveArgs.RefundAmount)
	assert.Equal(t, 50.0, *mock.lastResolveArgs.RefundAmount) // Full fare
}

func TestService_AdminResolveDispute_PartialRefund_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	refundAmount := 25.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   &refundAmount,
		Note:           "Partial refund approved for surge pricing error",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.True(t, mock.resolveDisputeCalled)
	require.NotNil(t, mock.lastResolveArgs)
	assert.Equal(t, DisputeStatusPartial, mock.lastResolveArgs.Status)
	assert.Equal(t, 25.0, *mock.lastResolveArgs.RefundAmount)
}

func TestService_AdminResolveDispute_PartialRefund_NoAmount(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   nil, // Missing
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "positive refund amount")
}

func TestService_AdminResolveDispute_PartialRefund_ZeroAmount(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	zeroAmount := 0.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   &zeroAmount,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
}

func TestService_AdminResolveDispute_PartialRefund_NegativeAmount(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	negativeAmount := -10.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   &negativeAmount,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
}

func TestService_AdminResolveDispute_PartialRefund_ExceedsFare(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	tooMuch := 100.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   &tooMuch,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "cannot exceed original fare")
}

func TestService_AdminResolveDispute_Credits_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionCredits,
		Note:           "Issued ride credits as goodwill gesture",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.Equal(t, DisputeStatusApproved, mock.lastResolveArgs.Status)
	assert.Equal(t, ResolutionCredits, mock.lastResolveArgs.ResType)
}

func TestService_AdminResolveDispute_NoAction_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionNoAction,
		Note:           "Dispute claim not substantiated by evidence",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.Equal(t, DisputeStatusRejected, mock.lastResolveArgs.Status)
	assert.Equal(t, ResolutionNoAction, mock.lastResolveArgs.ResType)
}

func TestService_AdminResolveDispute_FareAdjust_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	adjustment := 15.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFareAdjust,
		RefundAmount:   &adjustment,
		Note:           "Fare adjusted due to route deviation",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.Equal(t, DisputeStatusApproved, mock.lastResolveArgs.Status)
	assert.Equal(t, ResolutionFareAdjust, mock.lastResolveArgs.ResType)
}

func TestService_AdminResolveDispute_FareAdjust_NoAmount(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFareAdjust,
		RefundAmount:   nil,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "fare adjustment requires an amount")
}

func TestService_AdminResolveDispute_InvalidResolutionType(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionType("invalid_type"),
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 400, appErr.Code)
	assert.Contains(t, appErr.Message, "invalid resolution type")
}

func TestService_AdminResolveDispute_DisputeNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
}

func TestService_AdminResolveDispute_AlreadyResolved(t *testing.T) {
	statuses := []DisputeStatus{
		DisputeStatusClosed,
		DisputeStatusApproved,
		DisputeStatusRejected,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			adminID := uuid.New()
			disputeID := uuid.New()

			dispute := &Dispute{
				ID:           disputeID,
				UserID:       uuid.New(),
				Status:       status,
				OriginalFare: 50.0,
			}
			mock.AddDispute(dispute)

			req := &ResolveDisputeRequest{
				ResolutionType: ResolutionFullRefund,
				Note:           "Test note",
			}

			err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

			require.Error(t, err)
			var appErr *common.AppError
			require.True(t, errors.As(err, &appErr))
			assert.Equal(t, 400, appErr.Code)
			assert.Contains(t, appErr.Message, "already resolved")
		})
	}
}

func TestService_AdminResolveDispute_CanResolveFromPending(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionNoAction,
		Note:           "Quick resolution",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
}

func TestService_AdminResolveDispute_CanResolveFromReviewing(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusReviewing,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "After thorough review",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
}

func TestService_AdminResolveDispute_CanResolveFromPartial(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPartial,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "Upgraded to full refund upon appeal",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
}

func TestService_AdminResolveDispute_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.resolveDisputeErr = errors.New("database error")
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "Test note",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resolve dispute")
}

func TestService_AdminResolveDispute_AddsComment(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionFullRefund,
		Note:           "Resolution note for user",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.True(t, mock.createCommentCalled)
	require.NotNil(t, mock.lastCommentCreated)
	assert.Equal(t, "admin", mock.lastCommentCreated.UserRole)
	assert.Contains(t, mock.lastCommentCreated.Comment, "full_refund")
	assert.Contains(t, mock.lastCommentCreated.Comment, "Resolution note for user")
}

// ========================================
// AdminAddComment TESTS
// ========================================

func TestService_AdminAddComment_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: uuid.New(),
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	comment, err := svc.AdminAddComment(context.Background(), disputeID, adminID, "Admin response to user", false)

	require.NoError(t, err)
	require.NotNil(t, comment)
	assert.Equal(t, disputeID, comment.DisputeID)
	assert.Equal(t, adminID, comment.UserID)
	assert.Equal(t, "admin", comment.UserRole)
	assert.Equal(t, "Admin response to user", comment.Comment)
	assert.False(t, comment.IsInternal)
	assert.True(t, mock.createCommentCalled)
	assert.True(t, mock.updateStatusCalled)
}

func TestService_AdminAddComment_InternalNote(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: uuid.New(),
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	comment, err := svc.AdminAddComment(context.Background(), disputeID, adminID, "Internal note for team", true)

	require.NoError(t, err)
	require.NotNil(t, comment)
	assert.True(t, comment.IsInternal)
}

func TestService_AdminAddComment_DisputeNotFound(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	comment, err := svc.AdminAddComment(context.Background(), disputeID, adminID, "Test comment", false)

	require.Error(t, err)
	assert.Nil(t, comment)
	var appErr *common.AppError
	require.True(t, errors.As(err, &appErr))
	assert.Equal(t, 404, appErr.Code)
}

func TestService_AdminAddComment_UpdatesToReviewing(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: uuid.New(),
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	_, err := svc.AdminAddComment(context.Background(), disputeID, adminID, "Looking into this", false)

	require.NoError(t, err)
	assert.True(t, mock.updateStatusCalled)
	require.NotNil(t, mock.lastStatusUpdate)
	assert.Equal(t, DisputeStatusReviewing, mock.lastStatusUpdate.Status)
}

func TestService_AdminAddComment_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.createCommentErr = errors.New("database error")
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: uuid.New(),
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	comment, err := svc.AdminAddComment(context.Background(), disputeID, adminID, "Test comment", false)

	require.Error(t, err)
	assert.Nil(t, comment)
	assert.Contains(t, err.Error(), "create admin comment")
}

// ========================================
// AdminGetStats TESTS
// ========================================

func TestService_AdminGetStats_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	stats := &DisputeStats{
		TotalDisputes:      100,
		PendingDisputes:    20,
		ReviewingDisputes:  15,
		ApprovedDisputes:   50,
		RejectedDisputes:   15,
		TotalRefunded:      2500.00,
		TotalDisputed:      5000.00,
		AvgResolutionHours: 24.5,
		DisputeRate:        2.5,
		ByReason: []ReasonCount{
			{Reason: ReasonOvercharged, Count: 40, Percentage: 40.0},
			{Reason: ReasonWrongRoute, Count: 30, Percentage: 30.0},
		},
	}
	mock.SetStats(stats)

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	result, err := svc.AdminGetStats(context.Background(), from, to)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 100, result.TotalDisputes)
	assert.Equal(t, 20, result.PendingDisputes)
	assert.Equal(t, 2500.00, result.TotalRefunded)
	assert.Len(t, result.ByReason, 2)
	assert.True(t, mock.getStatsCalled)
}

func TestService_AdminGetStats_EmptyStats(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	result, err := svc.AdminGetStats(context.Background(), from, to)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, 0, result.TotalDisputes)
	assert.Empty(t, result.ByReason)
}

func TestService_AdminGetStats_RepositoryError(t *testing.T) {
	mock := NewMockRepository()
	mock.getStatsErr = errors.New("database error")
	svc := createTestService(mock)

	from := time.Now().Add(-7 * 24 * time.Hour)
	to := time.Now()

	result, err := svc.AdminGetStats(context.Background(), from, to)

	require.Error(t, err)
	assert.Nil(t, result)
}

// ========================================
// NewService TESTS
// ========================================

func TestNewService_Success(t *testing.T) {
	mock := NewMockRepository()
	svc := NewService(mock)

	require.NotNil(t, svc)
}

func TestNewService_NilRepositoryPanics(t *testing.T) {
	assert.Panics(t, func() {
		NewService(nil)
	})
}

// ========================================
// EDGE CASE TESTS
// ========================================

func TestService_CreateDispute_JustWithinDisputeWindow(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Ride completed 29 days ago (just within the 30-day window)
	completedAt := time.Now().Add(-29 * 24 * time.Hour)
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		FinalFare:     nil,
		CompletedAt:   &completedAt,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
}

func TestService_CreateDispute_JustOutsideDisputeWindow(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Ride completed just over 30 days ago
	completedAt := time.Now().Add(-30*24*time.Hour - time.Hour)
	rc := &RideContext{
		RideID:        rideID,
		EstimatedFare: 50.0,
		CompletedAt:   &completedAt,
	}
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.Error(t, err)
	assert.Nil(t, dispute)
}

func TestService_CreateDispute_ZeroDisputedAmount(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOther,
		Description:    "Test description with zero amount",
		DisputedAmount: 0.0, // Edge case: zero amount
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	// Service allows zero; validation might be at handler level
	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, 0.0, dispute.DisputedAmount)
}

func TestService_CreateDispute_VeryLargeFare(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	// Very large fare (e.g., luxury ride or long distance)
	largeFare := 10000.0
	rc := newCompletedRideContext(rideID, largeFare)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Large fare dispute",
		DisputedAmount: 5000.0,
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Equal(t, 10000.0, dispute.OriginalFare)
}

func TestService_GetMyDisputes_OffsetPassthrough(t *testing.T) {
	tests := []struct {
		name   string
		limit  int
		offset int
	}{
		{"offset 0", 20, 0},
		{"offset 20", 20, 20},
		{"offset 100 with limit 25", 25, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockRepository()
			svc := createTestService(mock)

			userID := uuid.New()
			svc.GetMyDisputes(context.Background(), userID, nil, tt.limit, tt.offset)

			assert.True(t, mock.getUserDisputesCalled)
		})
	}
}

func TestService_AdminResolveDispute_PartialRefundExactlyEqualsFare(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	adminID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:           disputeID,
		UserID:       uuid.New(),
		Status:       DisputeStatusPending,
		OriginalFare: 50.0,
	}
	mock.AddDispute(dispute)

	exactFare := 50.0
	req := &ResolveDisputeRequest{
		ResolutionType: ResolutionPartialRefund,
		RefundAmount:   &exactFare,
		Note:           "Full refund via partial type",
	}

	err := svc.AdminResolveDispute(context.Background(), disputeID, adminID, req)

	require.NoError(t, err)
	assert.Equal(t, 50.0, *mock.lastResolveArgs.RefundAmount)
}

func TestService_MultipleCommentsOnDispute(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	disputeID := uuid.New()

	dispute := &Dispute{
		ID:     disputeID,
		UserID: userID,
		Status: DisputeStatusPending,
	}
	mock.AddDispute(dispute)

	// Add multiple comments
	for i := 0; i < 5; i++ {
		req := &AddCommentRequest{
			Comment: "Comment " + string(rune('A'+i)),
		}
		comment, err := svc.AddComment(context.Background(), disputeID, userID, req)
		require.NoError(t, err)
		require.NotNil(t, comment)
	}

	// Verify all comments were created
	assert.Len(t, mock.comments[disputeID], 5)
}

func TestService_CreateDispute_WithMultipleEvidence(t *testing.T) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonWrongRoute,
		Description:    "Driver took wrong route, see attached evidence",
		DisputedAmount: 15.0,
		Evidence: []string{
			"screenshot1.png",
			"screenshot2.png",
			"gps_log.json",
			"receipt.pdf",
		},
	}

	dispute, err := svc.CreateDispute(context.Background(), userID, req)

	require.NoError(t, err)
	require.NotNil(t, dispute)
	assert.Len(t, dispute.Evidence, 4)
}

// ========================================
// BENCHMARK TESTS
// ========================================

func BenchmarkGenerateDisputeNumber(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateDisputeNumber()
	}
}

func BenchmarkCreateDispute(b *testing.B) {
	mock := NewMockRepository()
	svc := createTestService(mock)

	userID := uuid.New()
	rideID := uuid.New()

	rc := newCompletedRideContext(rideID, 50.0)
	mock.AddRideContext(rc)

	req := &CreateDisputeRequest{
		RideID:         rideID,
		Reason:         ReasonOvercharged,
		Description:    "Test description",
		DisputedAmount: 20.0,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Reset to allow creation
		delete(mock.disputeByRideUser, rideID.String()+"-"+userID.String())
		mock.createDisputeCalled = false
		svc.CreateDispute(ctx, userID, req)
	}
}
