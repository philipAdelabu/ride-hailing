package ridehistory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepository implements RepositoryInterface for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) GetRiderHistory(ctx context.Context, riderID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	args := m.Called(ctx, riderID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]RideHistoryEntry), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetDriverHistory(ctx context.Context, driverID uuid.UUID, filters *HistoryFilters, limit, offset int) ([]RideHistoryEntry, int, error) {
	args := m.Called(ctx, driverID, filters, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]RideHistoryEntry), args.Int(1), args.Error(2)
}

func (m *MockRepository) GetRideByID(ctx context.Context, rideID uuid.UUID) (*RideHistoryEntry, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideHistoryEntry), args.Error(1)
}

func (m *MockRepository) GetRiderStats(ctx context.Context, riderID uuid.UUID, from, to time.Time) (*RideStats, error) {
	args := m.Called(ctx, riderID, from, to)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RideStats), args.Error(1)
}

func (m *MockRepository) GetFrequentRoutes(ctx context.Context, riderID uuid.UUID, limit int) ([]FrequentRoute, error) {
	args := m.Called(ctx, riderID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]FrequentRoute), args.Error(1)
}

// ========================================
// GET RIDER HISTORY TESTS
// ========================================

func TestGetRiderHistory_Success(t *testing.T) {
	tests := []struct {
		name             string
		page             int
		pageSize         int
		filters          *HistoryFilters
		mockRides        []RideHistoryEntry
		mockTotal        int
		expectedPage     int
		expectedPageSize int
		expectedOffset   int
	}{
		{
			name:     "successful retrieval with default pagination",
			page:     1,
			pageSize: 20,
			filters:  nil,
			mockRides: []RideHistoryEntry{
				{ID: uuid.New(), Status: "completed", TotalFare: 25.50},
				{ID: uuid.New(), Status: "completed", TotalFare: 32.00},
			},
			mockTotal:        2,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:     "page 2 with custom page size",
			page:     2,
			pageSize: 10,
			filters:  nil,
			mockRides: []RideHistoryEntry{
				{ID: uuid.New(), Status: "completed", TotalFare: 15.00},
			},
			mockTotal:        15,
			expectedPage:     2,
			expectedPageSize: 10,
			expectedOffset:   10,
		},
		{
			name:             "invalid page defaults to 1",
			page:             0,
			pageSize:         20,
			filters:          nil,
			mockRides:        []RideHistoryEntry{},
			mockTotal:        0,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "negative page defaults to 1",
			page:             -5,
			pageSize:         20,
			filters:          nil,
			mockRides:        []RideHistoryEntry{},
			mockTotal:        0,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "page size 0 defaults to 20",
			page:             1,
			pageSize:         0,
			filters:          nil,
			mockRides:        []RideHistoryEntry{},
			mockTotal:        0,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "page size exceeds max defaults to 20",
			page:             1,
			pageSize:         100,
			filters:          nil,
			mockRides:        []RideHistoryEntry{},
			mockTotal:        0,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo)
			ctx := context.Background()
			riderID := uuid.New()

			mockRepo.On("GetRiderHistory", ctx, riderID, tt.filters, tt.expectedPageSize, tt.expectedOffset).
				Return(tt.mockRides, tt.mockTotal, nil)

			resp, err := svc.GetRiderHistory(ctx, riderID, tt.filters, tt.page, tt.pageSize)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedPage, resp.Page)
			assert.Equal(t, tt.expectedPageSize, resp.PageSize)
			assert.Equal(t, tt.mockTotal, resp.Total)
			assert.Len(t, resp.Rides, len(tt.mockRides))
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRiderHistory_WithFilters(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	status := "completed"
	fromDate := time.Now().AddDate(0, -1, 0)
	toDate := time.Now()
	minFare := 10.0
	maxFare := 100.0

	filters := &HistoryFilters{
		Status:   &status,
		FromDate: &fromDate,
		ToDate:   &toDate,
		MinFare:  &minFare,
		MaxFare:  &maxFare,
	}

	mockRides := []RideHistoryEntry{
		{ID: uuid.New(), Status: "completed", TotalFare: 50.00},
	}

	mockRepo.On("GetRiderHistory", ctx, riderID, filters, 20, 0).
		Return(mockRides, 1, nil)

	resp, err := svc.GetRiderHistory(ctx, riderID, filters, 1, 20)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Rides, 1)
	mockRepo.AssertExpectations(t)
}

func TestGetRiderHistory_EmptyResult(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	// Return nil slice from repository
	mockRepo.On("GetRiderHistory", ctx, riderID, (*HistoryFilters)(nil), 20, 0).
		Return(nil, 0, nil)

	resp, err := svc.GetRiderHistory(ctx, riderID, nil, 1, 20)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Rides)
	assert.Len(t, resp.Rides, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetRiderHistory_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	expectedErr := errors.New("database connection failed")
	mockRepo.On("GetRiderHistory", ctx, riderID, (*HistoryFilters)(nil), 20, 0).
		Return(nil, 0, expectedErr)

	resp, err := svc.GetRiderHistory(ctx, riderID, nil, 1, 20)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resp)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET DRIVER HISTORY TESTS
// ========================================

func TestGetDriverHistory_Success(t *testing.T) {
	tests := []struct {
		name             string
		page             int
		pageSize         int
		expectedPage     int
		expectedPageSize int
		expectedOffset   int
	}{
		{
			name:             "default pagination",
			page:             1,
			pageSize:         20,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
		{
			name:             "page 3 with 15 items",
			page:             3,
			pageSize:         15,
			expectedPage:     3,
			expectedPageSize: 15,
			expectedOffset:   30,
		},
		{
			name:             "invalid page size capped",
			page:             1,
			pageSize:         -1,
			expectedPage:     1,
			expectedPageSize: 20,
			expectedOffset:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo)
			ctx := context.Background()
			driverID := uuid.New()

			mockRides := []RideHistoryEntry{
				{ID: uuid.New(), Status: "completed", TotalFare: 45.00},
			}

			mockRepo.On("GetDriverHistory", ctx, driverID, (*HistoryFilters)(nil), tt.expectedPageSize, tt.expectedOffset).
				Return(mockRides, 50, nil)

			resp, err := svc.GetDriverHistory(ctx, driverID, nil, tt.page, tt.pageSize)

			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.expectedPage, resp.Page)
			assert.Equal(t, tt.expectedPageSize, resp.PageSize)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetDriverHistory_EmptyResult(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	driverID := uuid.New()

	mockRepo.On("GetDriverHistory", ctx, driverID, (*HistoryFilters)(nil), 20, 0).
		Return(nil, 0, nil)

	resp, err := svc.GetDriverHistory(ctx, driverID, nil, 1, 20)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotNil(t, resp.Rides)
	assert.Len(t, resp.Rides, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetDriverHistory_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	driverID := uuid.New()

	expectedErr := errors.New("query timeout")
	mockRepo.On("GetDriverHistory", ctx, driverID, (*HistoryFilters)(nil), 20, 0).
		Return(nil, 0, expectedErr)

	resp, err := svc.GetDriverHistory(ctx, driverID, nil, 1, 20)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, resp)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET RIDE DETAILS TESTS
// ========================================

func TestGetRideDetails_Success_AsRider(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()

	mockRide := &RideHistoryEntry{
		ID:        rideID,
		RiderID:   riderID,
		Status:    "completed",
		TotalFare: 35.50,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	ride, err := svc.GetRideDetails(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, ride)
	assert.Equal(t, rideID, ride.ID)
	assert.Equal(t, riderID, ride.RiderID)
	mockRepo.AssertExpectations(t)
}

func TestGetRideDetails_Success_AsDriver(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()

	mockRide := &RideHistoryEntry{
		ID:        rideID,
		RiderID:   riderID,
		DriverID:  &driverID,
		Status:    "completed",
		TotalFare: 42.00,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	ride, err := svc.GetRideDetails(ctx, rideID, driverID)

	assert.NoError(t, err)
	assert.NotNil(t, ride)
	assert.Equal(t, rideID, ride.ID)
	mockRepo.AssertExpectations(t)
}

func TestGetRideDetails_NotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	rideID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetRideByID", ctx, rideID).Return(nil, pgx.ErrNoRows)

	ride, err := svc.GetRideDetails(ctx, rideID, userID)

	assert.Error(t, err)
	assert.Nil(t, ride)

	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, 404, appErr.Code)
	assert.Contains(t, appErr.Message, "ride not found")
	mockRepo.AssertExpectations(t)
}

func TestGetRideDetails_Forbidden_UnauthorizedUser(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	driverID := uuid.New()
	unauthorizedUserID := uuid.New()
	rideID := uuid.New()

	mockRide := &RideHistoryEntry{
		ID:        rideID,
		RiderID:   riderID,
		DriverID:  &driverID,
		Status:    "completed",
		TotalFare: 28.00,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	ride, err := svc.GetRideDetails(ctx, rideID, unauthorizedUserID)

	assert.Error(t, err)
	assert.Nil(t, ride)

	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, 403, appErr.Code)
	assert.Contains(t, appErr.Message, "don't have access")
	mockRepo.AssertExpectations(t)
}

func TestGetRideDetails_Forbidden_NilDriver(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	unauthorizedUserID := uuid.New()
	rideID := uuid.New()

	// Ride has no driver assigned
	mockRide := &RideHistoryEntry{
		ID:        rideID,
		RiderID:   riderID,
		DriverID:  nil,
		Status:    "requested",
		TotalFare: 0,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	ride, err := svc.GetRideDetails(ctx, rideID, unauthorizedUserID)

	assert.Error(t, err)
	assert.Nil(t, ride)

	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, 403, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetRideDetails_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	rideID := uuid.New()
	userID := uuid.New()

	expectedErr := errors.New("database error")
	mockRepo.On("GetRideByID", ctx, rideID).Return(nil, expectedErr)

	ride, err := svc.GetRideDetails(ctx, rideID, userID)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, ride)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET RECEIPT TESTS
// ========================================

func TestGetReceipt_Success_FullFareBreakdown(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()
	completedAt := time.Now()
	promoCode := "SAVE20"
	driverName := "John Driver"
	licensePlate := "ABC123"
	vehicleMake := "Toyota"
	vehicleModel := "Camry"
	vehicleColor := "Silver"

	mockRide := &RideHistoryEntry{
		ID:              rideID,
		RiderID:         riderID,
		Status:          "completed",
		PickupAddress:   "123 Main St",
		DropoffAddress:  "456 Oak Ave",
		Distance:        8.5,
		Duration:        22,
		BaseFare:        5.00,
		DistanceFare:    12.75,
		TimeFare:        4.40,
		SurgeMultiplier: 1.5,
		SurgeAmount:     3.30,
		TollFees:        2.50,
		WaitTimeCharge:  1.00,
		TipAmount:       5.00,
		DiscountAmount:  4.00,
		PromoCode:       &promoCode,
		TotalFare:       29.95,
		Currency:        "USD",
		PaymentMethod:   "card",
		DriverName:      &driverName,
		LicensePlate:    &licensePlate,
		VehicleMake:     &vehicleMake,
		VehicleModel:    &vehicleModel,
		VehicleColor:    &vehicleColor,
		RequestedAt:     time.Now().Add(-30 * time.Minute),
		CompletedAt:     &completedAt,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)

	// Verify receipt metadata
	assert.NotEmpty(t, receipt.ReceiptID)
	assert.True(t, len(receipt.ReceiptID) > 4)
	assert.Equal(t, "RCP-", receipt.ReceiptID[:4])
	assert.Equal(t, rideID, receipt.RideID)

	// Verify trip details
	assert.Equal(t, "123 Main St", receipt.PickupAddress)
	assert.Equal(t, "456 Oak Ave", receipt.DropoffAddress)
	assert.Equal(t, 8.5, receipt.Distance)
	assert.Equal(t, 22, receipt.Duration)

	// Verify fare breakdown count
	assert.Len(t, receipt.FareBreakdown, 8) // base, distance, time, surge, wait, toll, discount, tip

	// Verify fare breakdown items
	fareLabels := make(map[string]FareLineItem)
	for _, item := range receipt.FareBreakdown {
		fareLabels[item.Label] = item
	}

	assert.Contains(t, fareLabels, "Base fare")
	assert.Equal(t, 5.00, fareLabels["Base fare"].Amount)
	assert.Equal(t, "charge", fareLabels["Base fare"].Type)

	assert.Contains(t, fareLabels, "Distance (8.5 km)")
	assert.Equal(t, 12.75, fareLabels["Distance (8.5 km)"].Amount)

	assert.Contains(t, fareLabels, "Time (22 min)")
	assert.Equal(t, 4.40, fareLabels["Time (22 min)"].Amount)

	assert.Contains(t, fareLabels, "Surge (1.5x)")
	assert.Equal(t, 3.30, fareLabels["Surge (1.5x)"].Amount)

	assert.Contains(t, fareLabels, "Wait time")
	assert.Equal(t, 1.00, fareLabels["Wait time"].Amount)
	assert.Equal(t, "fee", fareLabels["Wait time"].Type)

	assert.Contains(t, fareLabels, "Tolls")
	assert.Equal(t, 2.50, fareLabels["Tolls"].Amount)
	assert.Equal(t, "fee", fareLabels["Tolls"].Type)

	assert.Contains(t, fareLabels, "Promo (SAVE20)")
	assert.Equal(t, -4.00, fareLabels["Promo (SAVE20)"].Amount)
	assert.Equal(t, "discount", fareLabels["Promo (SAVE20)"].Type)

	assert.Contains(t, fareLabels, "Tip")
	assert.Equal(t, 5.00, fareLabels["Tip"].Amount)
	assert.Equal(t, "tip", fareLabels["Tip"].Type)

	// Verify totals
	assert.InDelta(t, 25.45, receipt.Subtotal, 0.01) // base + distance + time + surge
	assert.Equal(t, 3.50, receipt.Fees)              // tolls + wait time
	assert.Equal(t, 4.00, receipt.Discounts)
	assert.Equal(t, 5.00, receipt.Tip)
	assert.Equal(t, 29.95, receipt.Total)

	// Verify payment
	assert.Equal(t, "USD", receipt.Currency)
	assert.Equal(t, "card", receipt.PaymentMethod)

	// Verify driver/vehicle info
	assert.NotNil(t, receipt.DriverName)
	assert.Equal(t, "John Driver", *receipt.DriverName)
	assert.NotNil(t, receipt.LicensePlate)
	assert.Equal(t, "ABC123", *receipt.LicensePlate)
	assert.NotNil(t, receipt.VehicleInfo)
	assert.Equal(t, "Silver Toyota Camry", *receipt.VehicleInfo)

	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_Success_MinimalFare(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()
	completedAt := time.Now()

	mockRide := &RideHistoryEntry{
		ID:             rideID,
		RiderID:        riderID,
		Status:         "completed",
		PickupAddress:  "A",
		DropoffAddress: "B",
		Distance:       2.0,
		Duration:       5,
		BaseFare:       5.00,
		DistanceFare:   0,
		TimeFare:       0,
		TotalFare:      5.00,
		Currency:       "USD",
		PaymentMethod:  "cash",
		RequestedAt:    time.Now().Add(-10 * time.Minute),
		CompletedAt:    &completedAt,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Len(t, receipt.FareBreakdown, 1) // Only base fare
	assert.Equal(t, 5.00, receipt.Subtotal)
	assert.Equal(t, 0.0, receipt.Fees)
	assert.Equal(t, 0.0, receipt.Discounts)
	assert.Equal(t, 0.0, receipt.Tip)
	assert.Equal(t, 5.00, receipt.Total)
	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_Success_DiscountWithoutPromoCode(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()
	completedAt := time.Now()

	mockRide := &RideHistoryEntry{
		ID:             rideID,
		RiderID:        riderID,
		Status:         "completed",
		PickupAddress:  "A",
		DropoffAddress: "B",
		Distance:       5.0,
		Duration:       15,
		BaseFare:       5.00,
		DistanceFare:   7.50,
		DiscountAmount: 2.50,
		PromoCode:      nil, // No promo code
		TotalFare:      10.00,
		Currency:       "USD",
		PaymentMethod:  "card",
		RequestedAt:    time.Now().Add(-20 * time.Minute),
		CompletedAt:    &completedAt,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)

	// Find discount item
	var discountItem *FareLineItem
	for i, item := range receipt.FareBreakdown {
		if item.Type == "discount" {
			discountItem = &receipt.FareBreakdown[i]
			break
		}
	}

	assert.NotNil(t, discountItem)
	assert.Equal(t, "Discount", discountItem.Label)
	assert.Equal(t, -2.50, discountItem.Amount)
	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_NotCompleted(t *testing.T) {
	statuses := []string{"requested", "accepted", "in_progress", "cancelled"}

	for _, status := range statuses {
		t.Run("status_"+status, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo)
			ctx := context.Background()

			riderID := uuid.New()
			rideID := uuid.New()

			mockRide := &RideHistoryEntry{
				ID:        rideID,
				RiderID:   riderID,
				Status:    status,
				TotalFare: 0,
			}

			mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

			receipt, err := svc.GetReceipt(ctx, rideID, riderID)

			assert.Error(t, err)
			assert.Nil(t, receipt)

			appErr, ok := err.(*common.AppError)
			assert.True(t, ok)
			assert.Equal(t, 400, appErr.Code)
			assert.Contains(t, appErr.Message, "completed rides")
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetReceipt_RideNotFound(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	rideID := uuid.New()
	userID := uuid.New()

	mockRepo.On("GetRideByID", ctx, rideID).Return(nil, pgx.ErrNoRows)

	receipt, err := svc.GetReceipt(ctx, rideID, userID)

	assert.Error(t, err)
	assert.Nil(t, receipt)

	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, 404, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_Forbidden(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	unauthorizedUserID := uuid.New()
	rideID := uuid.New()
	completedAt := time.Now()

	mockRide := &RideHistoryEntry{
		ID:          rideID,
		RiderID:     riderID,
		Status:      "completed",
		TotalFare:   25.00,
		CompletedAt: &completedAt,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, unauthorizedUserID)

	assert.Error(t, err)
	assert.Nil(t, receipt)

	appErr, ok := err.(*common.AppError)
	assert.True(t, ok)
	assert.Equal(t, 403, appErr.Code)
	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_VehicleInfoFormatting(t *testing.T) {
	tests := []struct {
		name         string
		vehicleMake  *string
		vehicleModel *string
		vehicleColor *string
		expected     *string
	}{
		{
			name:         "all fields present",
			vehicleMake:  strPtr("Honda"),
			vehicleModel: strPtr("Accord"),
			vehicleColor: strPtr("Black"),
			expected:     strPtr("Black Honda Accord"),
		},
		{
			name:         "no color",
			vehicleMake:  strPtr("Tesla"),
			vehicleModel: strPtr("Model 3"),
			vehicleColor: nil,
			expected:     strPtr("Tesla Model 3"),
		},
		{
			name:         "no make",
			vehicleMake:  nil,
			vehicleModel: strPtr("Civic"),
			vehicleColor: strPtr("White"),
			expected:     nil,
		},
		{
			name:         "no model",
			vehicleMake:  strPtr("BMW"),
			vehicleModel: nil,
			vehicleColor: strPtr("Blue"),
			expected:     nil,
		},
		{
			name:         "all nil",
			vehicleMake:  nil,
			vehicleModel: nil,
			vehicleColor: nil,
			expected:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo)
			ctx := context.Background()

			riderID := uuid.New()
			rideID := uuid.New()
			completedAt := time.Now()

			mockRide := &RideHistoryEntry{
				ID:           rideID,
				RiderID:      riderID,
				Status:       "completed",
				BaseFare:     10.00,
				TotalFare:    10.00,
				Currency:     "USD",
				VehicleMake:  tt.vehicleMake,
				VehicleModel: tt.vehicleModel,
				VehicleColor: tt.vehicleColor,
				RequestedAt:  time.Now().Add(-15 * time.Minute),
				CompletedAt:  &completedAt,
			}

			mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

			receipt, err := svc.GetReceipt(ctx, rideID, riderID)

			assert.NoError(t, err)
			assert.NotNil(t, receipt)

			if tt.expected == nil {
				assert.Nil(t, receipt.VehicleInfo)
			} else {
				assert.NotNil(t, receipt.VehicleInfo)
				assert.Equal(t, *tt.expected, *receipt.VehicleInfo)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetReceipt_TimeFormatting(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()

	requestedAt := time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC)
	completedAt := time.Date(2024, 6, 15, 15, 5, 0, 0, time.UTC)

	mockRide := &RideHistoryEntry{
		ID:          rideID,
		RiderID:     riderID,
		Status:      "completed",
		BaseFare:    20.00,
		TotalFare:   20.00,
		Currency:    "USD",
		RequestedAt: requestedAt,
		CompletedAt: &completedAt,
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Equal(t, "June 15, 2024", receipt.TripDate)
	assert.Equal(t, "2:30 PM", receipt.TripStartTime)
	assert.Equal(t, "3:05 PM", receipt.TripEndTime)
	mockRepo.AssertExpectations(t)
}

func TestGetReceipt_NoCompletedAt(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	rideID := uuid.New()

	mockRide := &RideHistoryEntry{
		ID:          rideID,
		RiderID:     riderID,
		Status:      "completed",
		BaseFare:    15.00,
		TotalFare:   15.00,
		Currency:    "USD",
		RequestedAt: time.Now().Add(-30 * time.Minute),
		CompletedAt: nil, // Edge case: completed but no timestamp
	}

	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRide, nil)

	receipt, err := svc.GetReceipt(ctx, rideID, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, receipt)
	assert.Empty(t, receipt.TripEndTime)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET RIDER STATS TESTS
// ========================================

func TestGetRiderStats_Success(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"this week", "this_week"},
		{"this month", "this_month"},
		{"last month", "last_month"},
		{"this year", "this_year"},
		{"all time", "all_time"},
		{"unknown period defaults to all_time", "invalid_period"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(MockRepository)
			svc := NewService(mockRepo)
			ctx := context.Background()
			riderID := uuid.New()

			mockStats := &RideStats{
				TotalRides:     100,
				CompletedRides: 95,
				CancelledRides: 5,
				TotalSpent:     2500.00,
				TotalDistance:  750.5,
				TotalDuration:  1800,
				AverageFare:    26.32,
				AverageDistance: 7.9,
				AverageRating:  4.5,
				Currency:       "USD",
			}

			mockRepo.On("GetRiderStats", ctx, riderID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
				Return(mockStats, nil)

			stats, err := svc.GetRiderStats(ctx, riderID, tt.period)

			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, tt.period, stats.Period)
			assert.Equal(t, 100, stats.TotalRides)
			assert.Equal(t, 95, stats.CompletedRides)
			assert.Equal(t, 5, stats.CancelledRides)
			assert.Equal(t, 2500.00, stats.TotalSpent)
			assert.Equal(t, 750.5, stats.TotalDistance)
			assert.InDelta(t, 26.32, stats.AverageFare, 0.01)
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestGetRiderStats_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	expectedErr := errors.New("aggregation failed")
	mockRepo.On("GetRiderStats", ctx, riderID, mock.AnythingOfType("time.Time"), mock.AnythingOfType("time.Time")).
		Return(nil, expectedErr)

	stats, err := svc.GetRiderStats(ctx, riderID, "this_month")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, stats)
	mockRepo.AssertExpectations(t)
}

// ========================================
// GET FREQUENT ROUTES TESTS
// ========================================

func TestGetFrequentRoutes_Success(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	mockRoutes := []FrequentRoute{
		{
			PickupAddress:  "Home",
			DropoffAddress: "Office",
			RideCount:      45,
			AverageFare:    22.50,
			LastRideAt:     "2024-06-15 08:30:00",
		},
		{
			PickupAddress:  "Office",
			DropoffAddress: "Home",
			RideCount:      42,
			AverageFare:    21.75,
			LastRideAt:     "2024-06-14 18:00:00",
		},
		{
			PickupAddress:  "Home",
			DropoffAddress: "Gym",
			RideCount:      15,
			AverageFare:    12.00,
			LastRideAt:     "2024-06-13 06:00:00",
		},
	}

	mockRepo.On("GetFrequentRoutes", ctx, riderID, 10).Return(mockRoutes, nil)

	routes, err := svc.GetFrequentRoutes(ctx, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, routes)
	assert.Len(t, routes, 3)

	// Verify ordering (should be by ride_count desc)
	assert.Equal(t, 45, routes[0].RideCount)
	assert.Equal(t, 42, routes[1].RideCount)
	assert.Equal(t, 15, routes[2].RideCount)

	// Verify first route details
	assert.Equal(t, "Home", routes[0].PickupAddress)
	assert.Equal(t, "Office", routes[0].DropoffAddress)
	assert.Equal(t, 22.50, routes[0].AverageFare)
	mockRepo.AssertExpectations(t)
}

func TestGetFrequentRoutes_Empty(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	mockRepo.On("GetFrequentRoutes", ctx, riderID, 10).Return(nil, nil)

	routes, err := svc.GetFrequentRoutes(ctx, riderID)

	assert.NoError(t, err)
	assert.NotNil(t, routes)
	assert.Len(t, routes, 0)
	mockRepo.AssertExpectations(t)
}

func TestGetFrequentRoutes_RepositoryError(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()
	riderID := uuid.New()

	expectedErr := errors.New("query failed")
	mockRepo.On("GetFrequentRoutes", ctx, riderID, 10).Return(nil, expectedErr)

	routes, err := svc.GetFrequentRoutes(ctx, riderID)

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Nil(t, routes)
	mockRepo.AssertExpectations(t)
}

// ========================================
// PERIOD TO TIME RANGE TESTS
// ========================================

func TestPeriodToTimeRange(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name   string
		period string
	}{
		{"this_week", "this_week"},
		{"this_month", "this_month"},
		{"last_month", "last_month"},
		{"this_year", "this_year"},
		{"all_time", "all_time"},
		{"unknown defaults to all_time", "random"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := svc.periodToTimeRange(tt.period)
			assert.True(t, from.Before(to), "from (%v) should be before to (%v)", from, to)
		})
	}
}

func TestPeriodToTimeRange_ThisWeek(t *testing.T) {
	svc := &Service{}
	from, to := svc.periodToTimeRange("this_week")

	assert.True(t, from.Before(to))
	// Should start from Monday
	assert.True(t, from.Weekday() == time.Monday || from.Weekday() == time.Sunday)
}

func TestPeriodToTimeRange_ThisMonth(t *testing.T) {
	svc := &Service{}
	from, to := svc.periodToTimeRange("this_month")

	now := time.Now()
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, now.Month(), from.Month())
	assert.Equal(t, now.Year(), from.Year())
	assert.True(t, from.Before(to))
}

func TestPeriodToTimeRange_LastMonth(t *testing.T) {
	svc := &Service{}
	from, to := svc.periodToTimeRange("last_month")

	now := time.Now()
	expectedMonth := now.Month() - 1
	if expectedMonth == 0 {
		expectedMonth = 12
	}

	assert.Equal(t, 1, from.Day())
	assert.Equal(t, 1, to.Day())
	assert.True(t, from.Before(to))
}

func TestPeriodToTimeRange_ThisYear(t *testing.T) {
	svc := &Service{}
	from, to := svc.periodToTimeRange("this_year")

	now := time.Now()
	assert.Equal(t, time.January, from.Month())
	assert.Equal(t, 1, from.Day())
	assert.Equal(t, now.Year(), from.Year())
	assert.True(t, from.Before(to))
}

func TestPeriodToTimeRange_AllTime(t *testing.T) {
	svc := &Service{}
	from, to := svc.periodToTimeRange("all_time")

	assert.Equal(t, 2020, from.Year())
	assert.Equal(t, time.January, from.Month())
	assert.Equal(t, 1, from.Day())
	assert.True(t, to.After(from))
}

// ========================================
// GENERATE RECEIPT ID TESTS
// ========================================

func TestGenerateReceiptID(t *testing.T) {
	id := generateReceiptID()

	assert.NotEmpty(t, id)
	assert.Equal(t, "RCP-", id[:4])
	assert.Len(t, id, 17) // RCP-XXXXXX-XXXXXX

	// Check format: RCP-XXXXXX-XXXXXX
	assert.Equal(t, '-', rune(id[10]))
}

func TestGenerateReceiptID_Uniqueness(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := generateReceiptID()
		assert.False(t, ids[id], "duplicate receipt ID generated: %s", id)
		ids[id] = true
	}
}

func TestGenerateReceiptID_ValidCharacters(t *testing.T) {
	validChars := "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

	for i := 0; i < 50; i++ {
		id := generateReceiptID()
		// Skip "RCP-" prefix and the middle dash
		chars := id[4:10] + id[11:]
		for _, c := range chars {
			found := false
			for _, valid := range validChars {
				if c == valid {
					found = true
					break
				}
			}
			assert.True(t, found, "invalid character '%c' in receipt ID", c)
		}
	}
}

// ========================================
// FARE CALCULATION ACCURACY TESTS
// ========================================

func TestFareBreakdown_Accuracy(t *testing.T) {
	tests := []struct {
		name             string
		baseFare         float64
		distanceFare     float64
		timeFare         float64
		surgeAmount      float64
		expectedSubtotal float64
	}{
		{
			name:             "basic fare",
			baseFare:         5.00,
			distanceFare:     10.00,
			timeFare:         3.00,
			surgeAmount:      0.00,
			expectedSubtotal: 18.00,
		},
		{
			name:             "with surge",
			baseFare:         5.00,
			distanceFare:     12.50,
			timeFare:         4.25,
			surgeAmount:      5.44,
			expectedSubtotal: 27.19,
		},
		{
			name:             "decimal precision",
			baseFare:         3.33,
			distanceFare:     7.77,
			timeFare:         2.22,
			surgeAmount:      1.11,
			expectedSubtotal: 14.43,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subtotal := tt.baseFare + tt.distanceFare + tt.timeFare + tt.surgeAmount
			assert.InDelta(t, tt.expectedSubtotal, subtotal, 0.01)
		})
	}
}

func TestFeeCalculation_Accuracy(t *testing.T) {
	tests := []struct {
		name         string
		tollFees     float64
		waitCharge   float64
		expectedFees float64
	}{
		{
			name:         "no fees",
			tollFees:     0.00,
			waitCharge:   0.00,
			expectedFees: 0.00,
		},
		{
			name:         "toll only",
			tollFees:     3.50,
			waitCharge:   0.00,
			expectedFees: 3.50,
		},
		{
			name:         "wait only",
			tollFees:     0.00,
			waitCharge:   2.25,
			expectedFees: 2.25,
		},
		{
			name:         "both fees",
			tollFees:     4.75,
			waitCharge:   1.50,
			expectedFees: 6.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fees := tt.tollFees + tt.waitCharge
			assert.InDelta(t, tt.expectedFees, fees, 0.01)
		})
	}
}

// ========================================
// INTEGRATION-LIKE TESTS
// ========================================

func TestService_FullRideHistoryFlow(t *testing.T) {
	mockRepo := new(MockRepository)
	svc := NewService(mockRepo)
	ctx := context.Background()

	riderID := uuid.New()
	driverID := uuid.New()
	rideID := uuid.New()
	completedAt := time.Now()

	// Step 1: Get rider history
	mockRides := []RideHistoryEntry{
		{
			ID:        rideID,
			RiderID:   riderID,
			DriverID:  &driverID,
			Status:    "completed",
			TotalFare: 35.00,
		},
	}
	mockRepo.On("GetRiderHistory", ctx, riderID, (*HistoryFilters)(nil), 20, 0).
		Return(mockRides, 1, nil).Once()

	historyResp, err := svc.GetRiderHistory(ctx, riderID, nil, 1, 20)
	assert.NoError(t, err)
	assert.Len(t, historyResp.Rides, 1)

	// Step 2: Get ride details
	mockRideDetail := &RideHistoryEntry{
		ID:             rideID,
		RiderID:        riderID,
		DriverID:       &driverID,
		Status:         "completed",
		PickupAddress:  "Start",
		DropoffAddress: "End",
		BaseFare:       10.00,
		DistanceFare:   20.00,
		TimeFare:       5.00,
		TotalFare:      35.00,
		Currency:       "USD",
		PaymentMethod:  "card",
		RequestedAt:    time.Now().Add(-1 * time.Hour),
		CompletedAt:    &completedAt,
	}
	mockRepo.On("GetRideByID", ctx, rideID).Return(mockRideDetail, nil)

	rideDetail, err := svc.GetRideDetails(ctx, rideID, riderID)
	assert.NoError(t, err)
	assert.Equal(t, rideID, rideDetail.ID)

	// Step 3: Get receipt (uses same mock from GetRideDetails)
	receipt, err := svc.GetReceipt(ctx, rideID, riderID)
	assert.NoError(t, err)
	assert.Equal(t, 35.00, receipt.Total)

	mockRepo.AssertExpectations(t)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func strPtr(s string) *string {
	return &s
}
