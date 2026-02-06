package waittime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// ========================================
// MOCK REPOSITORY
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetActiveConfig(ctx context.Context) (*WaitTimeConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WaitTimeConfig), args.Error(1)
}

func (m *mockRepo) GetAllConfigs(ctx context.Context) ([]WaitTimeConfig, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WaitTimeConfig), args.Error(1)
}

func (m *mockRepo) CreateConfig(ctx context.Context, config *WaitTimeConfig) error {
	args := m.Called(ctx, config)
	return args.Error(0)
}

func (m *mockRepo) CreateRecord(ctx context.Context, rec *WaitTimeRecord) error {
	args := m.Called(ctx, rec)
	return args.Error(0)
}

func (m *mockRepo) GetActiveWaitByRide(ctx context.Context, rideID uuid.UUID) (*WaitTimeRecord, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*WaitTimeRecord), args.Error(1)
}

func (m *mockRepo) CompleteWait(ctx context.Context, recordID uuid.UUID, totalWaitMin, chargeableMin, totalCharge float64, wasCapped bool) error {
	args := m.Called(ctx, recordID, totalWaitMin, chargeableMin, totalCharge, wasCapped)
	return args.Error(0)
}

func (m *mockRepo) WaiveCharge(ctx context.Context, recordID uuid.UUID) error {
	args := m.Called(ctx, recordID)
	return args.Error(0)
}

func (m *mockRepo) GetRecordsByRide(ctx context.Context, rideID uuid.UUID) ([]WaitTimeRecord, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]WaitTimeRecord), args.Error(1)
}

func (m *mockRepo) SaveNotification(ctx context.Context, n *WaitTimeNotification) error {
	args := m.Called(ctx, n)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

// ========================================
// CONSTANTS TESTS
// ========================================

func TestDefaultConstants(t *testing.T) {
	assert.Equal(t, 3, defaultFreeMinutes)
	assert.Equal(t, 0.25, defaultChargePerMin)
	assert.Equal(t, 15, defaultMaxWaitMinutes)
	assert.Equal(t, 10.0, defaultMaxWaitCharge)
}

// ========================================
// START WAIT TESTS
// ========================================

func TestStartWait(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errType    string
		validate   func(t *testing.T, record *WaitTimeRecord)
	}{
		{
			name: "success - creates active wait record with config",
			setupMocks: func(m *mockRepo) {
				config := &WaitTimeConfig{
					ID:              configID,
					FreeWaitMinutes: 5,
					ChargePerMinute: 0.50,
					MaxWaitMinutes:  20,
					MaxWaitCharge:   15.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
				m.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, rideID, record.RideID)
				assert.Equal(t, driverID, record.DriverID)
				assert.Equal(t, configID, record.ConfigID)
				assert.Equal(t, "pickup", record.WaitType)
				assert.Equal(t, 5, record.FreeMinutes)
				assert.Equal(t, 0.50, record.ChargePerMinute)
				assert.Equal(t, "waiting", record.Status)
				assert.NotZero(t, record.ArrivedAt)
			},
		},
		{
			name: "success - uses default config when none exists",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
				m.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
				m.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, defaultFreeMinutes, record.FreeMinutes)
				assert.Equal(t, defaultChargePerMin, record.ChargePerMinute)
				assert.Equal(t, "waiting", record.Status)
			},
		},
		{
			name: "error - active wait already exists",
			setupMocks: func(m *mockRepo) {
				existingRecord := &WaitTimeRecord{
					ID:     uuid.New(),
					RideID: rideID,
					Status: "waiting",
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(existingRecord, nil)
			},
			wantErr: true,
			errType: "conflict",
		},
		{
			name: "error - repository error on check",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
		{
			name: "error - repository error on create",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
				m.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
				m.On("CreateRecord", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeRecord")).Return(errors.New("create error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			req := &StartWaitRequest{
				RideID:   rideID,
				WaitType: "pickup",
			}

			record, err := svc.StartWait(context.Background(), driverID, req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
				if tt.errType == "conflict" {
					var appErr *common.AppError
					assert.True(t, errors.As(err, &appErr))
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)
				tt.validate(t, record)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// STOP WAIT TESTS
// ========================================

func TestStopWait(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()
	otherDriverID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		driverID   uuid.UUID
		wantErr    bool
		errType    string
		validate   func(t *testing.T, record *WaitTimeRecord)
	}{
		{
			name: "success - calculate charges correctly within free minutes",
			setupMocks: func(m *mockRepo) {
				// Wait started 2 minutes ago (within 3 min free period)
				arrivedAt := time.Now().Add(-2 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 0.25,
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
				// Chargeable minutes should be 0 since 2 < 3 free minutes
				m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), 0.0, 0.0, false).Return(nil)
			},
			driverID: driverID,
			wantErr:  false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, "completed", record.Status)
				assert.Equal(t, 0.0, record.ChargeableMinutes)
				assert.Equal(t, 0.0, record.TotalCharge)
				assert.False(t, record.WasCapped)
			},
		},
		{
			name: "success - calculate charges correctly beyond free minutes",
			setupMocks: func(m *mockRepo) {
				// Wait started 8 minutes ago (5 chargeable minutes at $0.50/min)
				arrivedAt := time.Now().Add(-8 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 0.50,
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
				// ~5 chargeable minutes at $0.50/min = ~$2.50
				m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), false).Return(nil)
			},
			driverID: driverID,
			wantErr:  false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, "completed", record.Status)
				assert.Greater(t, record.ChargeableMinutes, 0.0)
				assert.Greater(t, record.TotalCharge, 0.0)
				assert.False(t, record.WasCapped)
			},
		},
		{
			name: "success - charge capped at max",
			setupMocks: func(m *mockRepo) {
				// Wait started 60 minutes ago (very long wait)
				arrivedAt := time.Now().Add(-60 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 1.00, // $1/min would be $57 for 57 chargeable minutes
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
				// Should be capped at $10
				m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), 10.0, true).Return(nil)
			},
			driverID: driverID,
			wantErr:  false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, "completed", record.Status)
				assert.Equal(t, 10.0, record.TotalCharge)
				assert.True(t, record.WasCapped)
			},
		},
		{
			name: "success - uses default max charge when no config",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-60 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 1.00,
					Status:          "waiting",
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
				// Should use default max charge of 10.0
				m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), defaultMaxWaitCharge, true).Return(nil)
			},
			driverID: driverID,
			wantErr:  false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, defaultMaxWaitCharge, record.TotalCharge)
				assert.True(t, record.WasCapped)
			},
		},
		{
			name: "error - no active wait timer",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
			},
			driverID: driverID,
			wantErr:  true,
			errType:  "not_found",
		},
		{
			name: "error - not your ride",
			setupMocks: func(m *mockRepo) {
				activeRecord := &WaitTimeRecord{
					ID:       recordID,
					RideID:   rideID,
					DriverID: otherDriverID,
					Status:   "waiting",
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			},
			driverID: driverID,
			wantErr:  true,
			errType:  "forbidden",
		},
		{
			name: "error - repository error on get",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			driverID: driverID,
			wantErr:  true,
		},
		{
			name: "error - repository error on complete",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-2 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 0.25,
					Status:          "waiting",
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
				m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("bool")).Return(errors.New("update error"))
			},
			driverID: driverID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			req := &StopWaitRequest{
				RideID: rideID,
			}

			record, err := svc.StopWait(context.Background(), tt.driverID, req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
				if tt.errType == "not_found" || tt.errType == "forbidden" {
					var appErr *common.AppError
					assert.True(t, errors.As(err, &appErr))
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)
				tt.validate(t, record)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// FREE MINUTES CALCULATION TESTS
// ========================================

func TestFreeMinutesCalculation(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name                  string
		waitMinutes           int
		freeMinutes           int
		chargePerMin          float64
		expectedChargeableMin float64
		expectedCharge        float64
	}{
		{
			name:                  "0 min wait - no charge",
			waitMinutes:           0,
			freeMinutes:           3,
			chargePerMin:          0.25,
			expectedChargeableMin: 0.0,
			expectedCharge:        0.0,
		},
		{
			name:                  "exactly at free limit - no charge",
			waitMinutes:           3,
			freeMinutes:           3,
			chargePerMin:          0.25,
			expectedChargeableMin: 0.0,
			expectedCharge:        0.0,
		},
		{
			name:                  "1 min over free limit",
			waitMinutes:           4,
			freeMinutes:           3,
			chargePerMin:          0.25,
			expectedChargeableMin: 1.0,
			expectedCharge:        0.25,
		},
		{
			name:                  "5 min over free limit",
			waitMinutes:           8,
			freeMinutes:           3,
			chargePerMin:          0.50,
			expectedChargeableMin: 5.0,
			expectedCharge:        2.50,
		},
		{
			name:                  "different free minutes config - 5 min free",
			waitMinutes:           10,
			freeMinutes:           5,
			chargePerMin:          0.30,
			expectedChargeableMin: 5.0,
			expectedCharge:        1.50,
		},
		{
			name:                  "zero free minutes",
			waitMinutes:           5,
			freeMinutes:           0,
			chargePerMin:          0.25,
			expectedChargeableMin: 5.0,
			expectedCharge:        1.25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)

			arrivedAt := time.Now().Add(-time.Duration(tt.waitMinutes) * time.Minute)
			activeRecord := &WaitTimeRecord{
				ID:              recordID,
				RideID:          rideID,
				DriverID:        driverID,
				ConfigID:        configID,
				ArrivedAt:       arrivedAt,
				FreeMinutes:     tt.freeMinutes,
				ChargePerMinute: tt.chargePerMin,
				Status:          "waiting",
			}
			config := &WaitTimeConfig{
				MaxWaitCharge: 100.0, // High cap to not interfere with test
			}

			m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), false).Return(nil)

			svc := newTestService(m)
			req := &StopWaitRequest{RideID: rideID}

			record, err := svc.StopWait(context.Background(), driverID, req)

			require.NoError(t, err)
			require.NotNil(t, record)

			// Allow for small time drift during test execution
			assert.InDelta(t, tt.expectedChargeableMin, record.ChargeableMinutes, 0.1)
			assert.InDelta(t, tt.expectedCharge, record.TotalCharge, 0.05)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// MAX CHARGE CAP TESTS
// ========================================

func TestMaxChargeCap(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name            string
		waitMinutes     int
		freeMinutes     int
		chargePerMin    float64
		maxCharge       float64
		expectedCharge  float64
		expectedCapped  bool
	}{
		{
			name:           "charge below cap",
			waitMinutes:    8,
			freeMinutes:    3,
			chargePerMin:   0.50,
			maxCharge:      10.0,
			expectedCharge: 2.50, // 5 chargeable min * $0.50
			expectedCapped: false,
		},
		{
			name:           "charge just below cap",
			waitMinutes:    22,
			freeMinutes:    3,
			chargePerMin:   0.50,
			maxCharge:      10.0,
			expectedCharge: 9.50, // 19 chargeable min * $0.50 = $9.50
			expectedCapped: false,
		},
		{
			name:           "charge exceeds cap - capped",
			waitMinutes:    30,
			freeMinutes:    3,
			chargePerMin:   1.00,
			maxCharge:      10.0,
			expectedCharge: 10.0, // Would be $27, capped at $10
			expectedCapped: true,
		},
		{
			name:           "very long wait - capped at low max",
			waitMinutes:    120,
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      5.0,
			expectedCharge: 5.0, // Would be $29.25, capped at $5
			expectedCapped: true,
		},
		{
			name:           "high rate long wait - capped",
			waitMinutes:    60,
			freeMinutes:    5,
			chargePerMin:   2.00,
			maxCharge:      15.0,
			expectedCharge: 15.0, // Would be $110, capped at $15
			expectedCapped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)

			arrivedAt := time.Now().Add(-time.Duration(tt.waitMinutes) * time.Minute)
			activeRecord := &WaitTimeRecord{
				ID:              recordID,
				RideID:          rideID,
				DriverID:        driverID,
				ConfigID:        configID,
				ArrivedAt:       arrivedAt,
				FreeMinutes:     tt.freeMinutes,
				ChargePerMinute: tt.chargePerMin,
				Status:          "waiting",
			}
			config := &WaitTimeConfig{
				MaxWaitCharge: tt.maxCharge,
			}

			m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), tt.expectedCapped).Return(nil)

			svc := newTestService(m)
			req := &StopWaitRequest{RideID: rideID}

			record, err := svc.StopWait(context.Background(), driverID, req)

			require.NoError(t, err)
			require.NotNil(t, record)

			assert.InDelta(t, tt.expectedCharge, record.TotalCharge, 0.01)
			assert.Equal(t, tt.expectedCapped, record.WasCapped)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// ROUNDING TESTS
// ========================================

func TestChargeRounding(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name           string
		waitMinutes    float64 // Use float for precise timing
		freeMinutes    int
		chargePerMin   float64
		maxCharge      float64
		expectedCharge float64
	}{
		{
			name:           "rounds to 2 decimal places - down",
			waitMinutes:    5.5,
			freeMinutes:    3,
			chargePerMin:   0.33,
			maxCharge:      100.0,
			expectedCharge: 0.83, // 2.5 * 0.33 = 0.825 -> rounds to 0.83
		},
		{
			name:           "rounds to 2 decimal places - up",
			waitMinutes:    6.0,
			freeMinutes:    3,
			chargePerMin:   0.17,
			maxCharge:      100.0,
			expectedCharge: 0.51, // 3 * 0.17 = 0.51
		},
		{
			name:           "already clean decimal",
			waitMinutes:    8.0,
			freeMinutes:    3,
			chargePerMin:   0.50,
			maxCharge:      100.0,
			expectedCharge: 2.50, // 5 * 0.50 = 2.50
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)

			arrivedAt := time.Now().Add(-time.Duration(tt.waitMinutes*60) * time.Second)
			activeRecord := &WaitTimeRecord{
				ID:              recordID,
				RideID:          rideID,
				DriverID:        driverID,
				ConfigID:        configID,
				ArrivedAt:       arrivedAt,
				FreeMinutes:     tt.freeMinutes,
				ChargePerMinute: tt.chargePerMin,
				Status:          "waiting",
			}
			config := &WaitTimeConfig{
				MaxWaitCharge: tt.maxCharge,
			}

			m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), false).Return(nil)

			svc := newTestService(m)
			req := &StopWaitRequest{RideID: rideID}

			record, err := svc.StopWait(context.Background(), driverID, req)

			require.NoError(t, err)
			require.NotNil(t, record)

			// Allow small delta for timing variations
			assert.InDelta(t, tt.expectedCharge, record.TotalCharge, 0.05)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// GET CURRENT WAIT STATUS TESTS
// ========================================

func TestGetCurrentWaitStatus(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		errType    string
		validate   func(t *testing.T, record *WaitTimeRecord)
	}{
		{
			name: "success - calculates live values within free period",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-2 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 0.25,
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.InDelta(t, 2.0, record.TotalWaitMinutes, 0.2)
				assert.Equal(t, 0.0, record.ChargeableMinutes)
				assert.Equal(t, 0.0, record.TotalCharge)
				assert.False(t, record.WasCapped)
			},
		},
		{
			name: "success - calculates live values beyond free period",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-8 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 0.50,
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.InDelta(t, 8.0, record.TotalWaitMinutes, 0.2)
				assert.InDelta(t, 5.0, record.ChargeableMinutes, 0.2)
				assert.InDelta(t, 2.50, record.TotalCharge, 0.1)
				assert.False(t, record.WasCapped)
			},
		},
		{
			name: "success - shows capped status when charge would exceed max",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-60 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 1.00,
					Status:          "waiting",
				}
				config := &WaitTimeConfig{
					MaxWaitCharge: 10.0,
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, 10.0, record.TotalCharge)
				assert.True(t, record.WasCapped)
			},
		},
		{
			name: "success - uses default max charge when no config",
			setupMocks: func(m *mockRepo) {
				arrivedAt := time.Now().Add(-60 * time.Minute)
				activeRecord := &WaitTimeRecord{
					ID:              recordID,
					RideID:          rideID,
					DriverID:        driverID,
					ConfigID:        configID,
					ArrivedAt:       arrivedAt,
					FreeMinutes:     3,
					ChargePerMinute: 1.00,
					Status:          "waiting",
				}
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
				m.On("GetActiveConfig", mock.Anything).Return(nil, errors.New("no config"))
			},
			wantErr: false,
			validate: func(t *testing.T, record *WaitTimeRecord) {
				assert.Equal(t, defaultMaxWaitCharge, record.TotalCharge)
				assert.True(t, record.WasCapped)
			},
		},
		{
			name: "error - no active wait timer",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, nil)
			},
			wantErr: true,
			errType: "not_found",
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			record, err := svc.GetCurrentWaitStatus(context.Background(), rideID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, record)
				if tt.errType == "not_found" {
					var appErr *common.AppError
					assert.True(t, errors.As(err, &appErr))
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, record)
				tt.validate(t, record)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// WAIVE CHARGE TESTS
// ========================================

func TestWaiveCharge(t *testing.T) {
	recordID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
	}{
		{
			name: "success - waives charge",
			setupMocks: func(m *mockRepo) {
				m.On("WaiveCharge", mock.Anything, recordID).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("WaiveCharge", mock.Anything, recordID).Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.WaiveCharge(context.Background(), recordID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// GET WAIT TIME SUMMARY TESTS
// ========================================

func TestGetWaitTimeSummary(t *testing.T) {
	rideID := uuid.New()
	recordID1 := uuid.New()
	recordID2 := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, summary *WaitTimeSummary)
	}{
		{
			name: "success - single completed record",
			setupMocks: func(m *mockRepo) {
				records := []WaitTimeRecord{
					{
						ID:                recordID1,
						RideID:            rideID,
						TotalWaitMinutes:  8.0,
						ChargeableMinutes: 5.0,
						TotalCharge:       2.50,
						Status:            "completed",
					},
				}
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WaitTimeSummary) {
				assert.Equal(t, rideID, summary.RideID)
				assert.Len(t, summary.Records, 1)
				assert.Equal(t, 8.0, summary.TotalWaitMinutes)
				assert.Equal(t, 5.0, summary.TotalChargeableMin)
				assert.Equal(t, 2.50, summary.TotalCharge)
				assert.False(t, summary.HasActiveWait)
			},
		},
		{
			name: "success - multiple records summed",
			setupMocks: func(m *mockRepo) {
				records := []WaitTimeRecord{
					{
						ID:                recordID1,
						RideID:            rideID,
						TotalWaitMinutes:  8.0,
						ChargeableMinutes: 5.0,
						TotalCharge:       2.50,
						Status:            "completed",
					},
					{
						ID:                recordID2,
						RideID:            rideID,
						TotalWaitMinutes:  10.0,
						ChargeableMinutes: 7.0,
						TotalCharge:       3.50,
						Status:            "completed",
					},
				}
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WaitTimeSummary) {
				assert.Len(t, summary.Records, 2)
				assert.Equal(t, 18.0, summary.TotalWaitMinutes)
				assert.Equal(t, 12.0, summary.TotalChargeableMin)
				assert.Equal(t, 6.0, summary.TotalCharge)
				assert.False(t, summary.HasActiveWait)
			},
		},
		{
			name: "success - has active wait",
			setupMocks: func(m *mockRepo) {
				records := []WaitTimeRecord{
					{
						ID:                recordID1,
						RideID:            rideID,
						TotalWaitMinutes:  8.0,
						ChargeableMinutes: 5.0,
						TotalCharge:       2.50,
						Status:            "completed",
					},
					{
						ID:     recordID2,
						RideID: rideID,
						Status: "waiting",
					},
				}
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WaitTimeSummary) {
				assert.True(t, summary.HasActiveWait)
			},
		},
		{
			name: "success - no records returns empty summary",
			setupMocks: func(m *mockRepo) {
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, summary *WaitTimeSummary) {
				assert.Equal(t, rideID, summary.RideID)
				assert.Len(t, summary.Records, 0)
				assert.Equal(t, 0.0, summary.TotalWaitMinutes)
				assert.Equal(t, 0.0, summary.TotalCharge)
				assert.False(t, summary.HasActiveWait)
			},
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.GetWaitTimeSummary(context.Background(), rideID)

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

// ========================================
// GET TOTAL WAIT CHARGE TESTS
// ========================================

func TestGetTotalWaitCharge(t *testing.T) {
	rideID := uuid.New()

	tests := []struct {
		name          string
		setupMocks    func(m *mockRepo)
		wantErr       bool
		expectedTotal float64
	}{
		{
			name: "success - sums completed records only",
			setupMocks: func(m *mockRepo) {
				records := []WaitTimeRecord{
					{TotalCharge: 2.50, Status: "completed"},
					{TotalCharge: 3.50, Status: "completed"},
					{TotalCharge: 1.00, Status: "waiting"},  // Should be ignored
					{TotalCharge: 0.00, Status: "waived"},   // Should be ignored
				}
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)
			},
			wantErr:       false,
			expectedTotal: 6.0,
		},
		{
			name: "success - no records returns 0",
			setupMocks: func(m *mockRepo) {
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, nil)
			},
			wantErr:       false,
			expectedTotal: 0.0,
		},
		{
			name: "success - all waived returns 0",
			setupMocks: func(m *mockRepo) {
				records := []WaitTimeRecord{
					{TotalCharge: 0.00, Status: "waived"},
					{TotalCharge: 0.00, Status: "waived"},
				}
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(records, nil)
			},
			wantErr:       false,
			expectedTotal: 0.0,
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetRecordsByRide", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			total, err := svc.GetTotalWaitCharge(context.Background(), rideID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTotal, total)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// ADMIN: CREATE CONFIG TESTS
// ========================================

func TestCreateConfig(t *testing.T) {
	tests := []struct {
		name       string
		req        *CreateConfigRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, config *WaitTimeConfig)
	}{
		{
			name: "success - creates config",
			req: &CreateConfigRequest{
				Name:            "Standard",
				FreeWaitMinutes: 5,
				ChargePerMinute: 0.30,
				MaxWaitMinutes:  20,
				MaxWaitCharge:   12.0,
				AppliesTo:       "pickup",
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateConfig", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeConfig")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, config *WaitTimeConfig) {
				assert.Equal(t, "Standard", config.Name)
				assert.Equal(t, 5, config.FreeWaitMinutes)
				assert.Equal(t, 0.30, config.ChargePerMinute)
				assert.Equal(t, 20, config.MaxWaitMinutes)
				assert.Equal(t, 12.0, config.MaxWaitCharge)
				assert.Equal(t, "pickup", config.AppliesTo)
				assert.True(t, config.IsActive)
				assert.NotZero(t, config.ID)
				assert.NotZero(t, config.CreatedAt)
			},
		},
		{
			name: "error - repository error",
			req: &CreateConfigRequest{
				Name:            "Test",
				FreeWaitMinutes: 3,
				ChargePerMinute: 0.25,
				MaxWaitMinutes:  15,
				MaxWaitCharge:   10.0,
				AppliesTo:       "both",
			},
			setupMocks: func(m *mockRepo) {
				m.On("CreateConfig", mock.Anything, mock.AnythingOfType("*waittime.WaitTimeConfig")).Return(errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			config, err := svc.CreateConfig(context.Background(), tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, config)
			} else {
				require.NoError(t, err)
				require.NotNil(t, config)
				tt.validate(t, config)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// ADMIN: LIST CONFIGS TESTS
// ========================================

func TestListConfigs(t *testing.T) {
	configID1 := uuid.New()
	configID2 := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, configs []WaitTimeConfig)
	}{
		{
			name: "success - returns configs",
			setupMocks: func(m *mockRepo) {
				configs := []WaitTimeConfig{
					{ID: configID1, Name: "Standard", IsActive: true},
					{ID: configID2, Name: "Airport", IsActive: false},
				}
				m.On("GetAllConfigs", mock.Anything).Return(configs, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, configs []WaitTimeConfig) {
				assert.Len(t, configs, 2)
				assert.Equal(t, "Standard", configs[0].Name)
				assert.Equal(t, "Airport", configs[1].Name)
			},
		},
		{
			name: "success - returns empty list when none exist",
			setupMocks: func(m *mockRepo) {
				m.On("GetAllConfigs", mock.Anything).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, configs []WaitTimeConfig) {
				assert.Len(t, configs, 0)
			},
		},
		{
			name: "error - repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetAllConfigs", mock.Anything).Return(nil, errors.New("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			configs, err := svc.ListConfigs(context.Background())

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, configs)
			} else {
				require.NoError(t, err)
				require.NotNil(t, configs)
				tt.validate(t, configs)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// BOUNDARY VALUE TESTS
// ========================================

func TestBoundaryValues(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name           string
		waitSeconds    int
		freeMinutes    int
		chargePerMin   float64
		maxCharge      float64
		expectedCharge float64
		expectedCapped bool
	}{
		{
			name:           "just under free limit",
			waitSeconds:    179, // 2 min 59 sec
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      10.0,
			expectedCharge: 0.0,
			expectedCapped: false,
		},
		{
			name:           "exactly at free limit",
			waitSeconds:    180, // 3 min exactly
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      10.0,
			expectedCharge: 0.0,
			expectedCapped: false,
		},
		{
			name:           "just over free limit",
			waitSeconds:    181, // 3 min 1 sec
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      10.0,
			expectedCharge: 0.0, // ~0.004, rounds to 0.00
			expectedCapped: false,
		},
		{
			name:           "1 minute over free limit",
			waitSeconds:    240, // 4 minutes
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      10.0,
			expectedCharge: 0.25,
			expectedCapped: false,
		},
		{
			name:           "zero wait time",
			waitSeconds:    0,
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      10.0,
			expectedCharge: 0.0,
			expectedCapped: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)

			arrivedAt := time.Now().Add(-time.Duration(tt.waitSeconds) * time.Second)
			activeRecord := &WaitTimeRecord{
				ID:              recordID,
				RideID:          rideID,
				DriverID:        driverID,
				ConfigID:        configID,
				ArrivedAt:       arrivedAt,
				FreeMinutes:     tt.freeMinutes,
				ChargePerMinute: tt.chargePerMin,
				Status:          "waiting",
			}
			config := &WaitTimeConfig{
				MaxWaitCharge: tt.maxCharge,
			}

			m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), tt.expectedCapped).Return(nil)

			svc := newTestService(m)
			req := &StopWaitRequest{RideID: rideID}

			record, err := svc.StopWait(context.Background(), driverID, req)

			require.NoError(t, err)
			require.NotNil(t, record)

			// Allow delta for timing
			assert.InDelta(t, tt.expectedCharge, record.TotalCharge, 0.05)
			assert.Equal(t, tt.expectedCapped, record.WasCapped)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// RATE PER MINUTE CALCULATION TESTS
// ========================================

func TestRatePerMinuteCalculations(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	recordID := uuid.New()
	configID := uuid.New()

	tests := []struct {
		name           string
		waitMinutes    int
		freeMinutes    int
		chargePerMin   float64
		maxCharge      float64
		expectedCharge float64
	}{
		{
			name:           "low rate $0.10/min",
			waitMinutes:    13,
			freeMinutes:    3,
			chargePerMin:   0.10,
			maxCharge:      100.0,
			expectedCharge: 1.0, // 10 chargeable min * $0.10
		},
		{
			name:           "standard rate $0.25/min",
			waitMinutes:    13,
			freeMinutes:    3,
			chargePerMin:   0.25,
			maxCharge:      100.0,
			expectedCharge: 2.50, // 10 chargeable min * $0.25
		},
		{
			name:           "premium rate $0.50/min",
			waitMinutes:    13,
			freeMinutes:    3,
			chargePerMin:   0.50,
			maxCharge:      100.0,
			expectedCharge: 5.0, // 10 chargeable min * $0.50
		},
		{
			name:           "high rate $1.00/min",
			waitMinutes:    13,
			freeMinutes:    3,
			chargePerMin:   1.00,
			maxCharge:      100.0,
			expectedCharge: 10.0, // 10 chargeable min * $1.00
		},
		{
			name:           "very high rate $2.00/min",
			waitMinutes:    8,
			freeMinutes:    3,
			chargePerMin:   2.00,
			maxCharge:      100.0,
			expectedCharge: 10.0, // 5 chargeable min * $2.00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)

			arrivedAt := time.Now().Add(-time.Duration(tt.waitMinutes) * time.Minute)
			activeRecord := &WaitTimeRecord{
				ID:              recordID,
				RideID:          rideID,
				DriverID:        driverID,
				ConfigID:        configID,
				ArrivedAt:       arrivedAt,
				FreeMinutes:     tt.freeMinutes,
				ChargePerMinute: tt.chargePerMin,
				Status:          "waiting",
			}
			config := &WaitTimeConfig{
				MaxWaitCharge: tt.maxCharge,
			}

			m.On("GetActiveWaitByRide", mock.Anything, rideID).Return(activeRecord, nil)
			m.On("GetActiveConfig", mock.Anything).Return(config, nil)
			m.On("CompleteWait", mock.Anything, recordID, mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), mock.AnythingOfType("float64"), false).Return(nil)

			svc := newTestService(m)
			req := &StopWaitRequest{RideID: rideID}

			record, err := svc.StopWait(context.Background(), driverID, req)

			require.NoError(t, err)
			require.NotNil(t, record)

			// Allow small delta for timing
			assert.InDelta(t, tt.expectedCharge, record.TotalCharge, 0.1)

			m.AssertExpectations(t)
		})
	}
}
