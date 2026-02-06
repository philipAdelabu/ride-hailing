package experiments

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

type mockExperimentsRepository struct {
	mock.Mock
}

// Feature Flags
func (m *mockExperimentsRepository) CreateFlag(ctx context.Context, flag *FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetFlagByKey(ctx context.Context, key string) (*FeatureFlag, error) {
	args := m.Called(ctx, key)
	flag, _ := args.Get(0).(*FeatureFlag)
	return flag, args.Error(1)
}

func (m *mockExperimentsRepository) GetFlagByID(ctx context.Context, id uuid.UUID) (*FeatureFlag, error) {
	args := m.Called(ctx, id)
	flag, _ := args.Get(0).(*FeatureFlag)
	return flag, args.Error(1)
}

func (m *mockExperimentsRepository) ListFlags(ctx context.Context, status *FlagStatus, limit, offset int) ([]*FeatureFlag, error) {
	args := m.Called(ctx, status, limit, offset)
	flags, _ := args.Get(0).([]*FeatureFlag)
	return flags, args.Error(1)
}

func (m *mockExperimentsRepository) UpdateFlag(ctx context.Context, flag *FeatureFlag) error {
	args := m.Called(ctx, flag)
	return args.Error(0)
}

func (m *mockExperimentsRepository) UpdateFlagStatus(ctx context.Context, id uuid.UUID, status FlagStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetAllActiveFlags(ctx context.Context) ([]*FeatureFlag, error) {
	args := m.Called(ctx)
	flags, _ := args.Get(0).([]*FeatureFlag)
	return flags, args.Error(1)
}

// Flag Overrides
func (m *mockExperimentsRepository) CreateOverride(ctx context.Context, override *FlagOverride) error {
	args := m.Called(ctx, override)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetOverride(ctx context.Context, flagID, userID uuid.UUID) (*FlagOverride, error) {
	args := m.Called(ctx, flagID, userID)
	override, _ := args.Get(0).(*FlagOverride)
	return override, args.Error(1)
}

func (m *mockExperimentsRepository) ListOverrides(ctx context.Context, flagID uuid.UUID) ([]*FlagOverride, error) {
	args := m.Called(ctx, flagID)
	overrides, _ := args.Get(0).([]*FlagOverride)
	return overrides, args.Error(1)
}

func (m *mockExperimentsRepository) DeleteOverride(ctx context.Context, flagID, userID uuid.UUID) error {
	args := m.Called(ctx, flagID, userID)
	return args.Error(0)
}

// Experiments
func (m *mockExperimentsRepository) CreateExperiment(ctx context.Context, experiment *Experiment, variants []*Variant) error {
	args := m.Called(ctx, experiment, variants)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetExperimentByKey(ctx context.Context, key string) (*Experiment, error) {
	args := m.Called(ctx, key)
	exp, _ := args.Get(0).(*Experiment)
	return exp, args.Error(1)
}

func (m *mockExperimentsRepository) GetExperimentByID(ctx context.Context, id uuid.UUID) (*Experiment, error) {
	args := m.Called(ctx, id)
	exp, _ := args.Get(0).(*Experiment)
	return exp, args.Error(1)
}

func (m *mockExperimentsRepository) ListExperiments(ctx context.Context, status *ExperimentStatus, limit, offset int) ([]*Experiment, error) {
	args := m.Called(ctx, status, limit, offset)
	exps, _ := args.Get(0).([]*Experiment)
	return exps, args.Error(1)
}

func (m *mockExperimentsRepository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status ExperimentStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetVariants(ctx context.Context, experimentID uuid.UUID) ([]*Variant, error) {
	args := m.Called(ctx, experimentID)
	variants, _ := args.Get(0).([]*Variant)
	return variants, args.Error(1)
}

// Assignments
func (m *mockExperimentsRepository) GetAssignment(ctx context.Context, experimentID, userID uuid.UUID) (*ExperimentAssignment, error) {
	args := m.Called(ctx, experimentID, userID)
	assignment, _ := args.Get(0).(*ExperimentAssignment)
	return assignment, args.Error(1)
}

func (m *mockExperimentsRepository) CreateAssignment(ctx context.Context, assignment *ExperimentAssignment) error {
	args := m.Called(ctx, assignment)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetAssignmentCount(ctx context.Context, experimentID uuid.UUID) (map[uuid.UUID]int, error) {
	args := m.Called(ctx, experimentID)
	counts, _ := args.Get(0).(map[uuid.UUID]int)
	return counts, args.Error(1)
}

// Events
func (m *mockExperimentsRepository) RecordEvent(ctx context.Context, event *ExperimentEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *mockExperimentsRepository) GetVariantMetrics(ctx context.Context, experimentID uuid.UUID) ([]VariantMetrics, error) {
	args := m.Called(ctx, experimentID)
	metrics, _ := args.Get(0).([]VariantMetrics)
	return metrics, args.Error(1)
}

func (m *mockExperimentsRepository) GetActiveExperimentsForUser(ctx context.Context) ([]*Experiment, error) {
	args := m.Called(ctx)
	exps, _ := args.Get(0).([]*Experiment)
	return exps, args.Error(1)
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func newTestService(repo *mockExperimentsRepository) *Service {
	return NewService(repo)
}

func createTestFlag(key string, flagType FlagType, status FlagStatus, enabled bool) *FeatureFlag {
	return &FeatureFlag{
		ID:        uuid.New(),
		Key:       key,
		Name:      "Test Flag " + key,
		FlagType:  flagType,
		Status:    status,
		Enabled:   enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func createTestUserContext() *UserContext {
	return &UserContext{
		UserID:      uuid.New(),
		Role:        "rider",
		Country:     "US",
		City:        "San Francisco",
		Platform:    "ios",
		AppVersion:  "2.5.0",
		TotalRides:  50,
		Rating:      4.8,
		AccountAge:  365,
		LoyaltyTier: "gold",
	}
}

func createTestExperiment(key string, status ExperimentStatus) *Experiment {
	return &Experiment{
		ID:                uuid.New(),
		Key:               key,
		Name:              "Test Experiment " + key,
		Status:            status,
		TrafficPercentage: 100,
		MinSampleSize:     100,
		ConfidenceLevel:   0.95,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
}

func createTestVariants(experimentID uuid.UUID) []*Variant {
	return []*Variant{
		{
			ID:           uuid.New(),
			ExperimentID: experimentID,
			Key:          "control",
			Name:         "Control",
			IsControl:    true,
			Weight:       50,
		},
		{
			ID:           uuid.New(),
			ExperimentID: experimentID,
			Key:          "variant_a",
			Name:         "Variant A",
			IsControl:    false,
			Weight:       50,
		},
	}
}

// ========================================
// FEATURE FLAG EVALUATION TESTS
// ========================================

func TestIsEnabled(t *testing.T) {
	tests := []struct {
		name           string
		flagKey        string
		userCtx        *UserContext
		setupMock      func(*mockExperimentsRepository, *UserContext)
		expectedResult bool
		expectError    bool
	}{
		{
			name:    "flag not found returns false",
			flagKey: "unknown_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				// Cache refresh returns empty, so flag not found returns nil
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{}, nil).Once()
			},
			expectedResult: false,
			expectError:    false,
		},
		{
			name:    "inactive flag returns false",
			flagKey: "inactive_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				// Inactive flags are not in active cache, so returns not found
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{}, nil).Once()
			},
			expectedResult: false,
			expectError:    false,
		},
		{
			name:    "boolean flag enabled returns true",
			flagKey: "enabled_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("enabled_flag", FlagTypeBoolean, FlagStatusActive, true)
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
			},
			expectedResult: true,
			expectError:    false,
		},
		{
			name:    "boolean flag disabled returns false",
			flagKey: "disabled_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("disabled_flag", FlagTypeBoolean, FlagStatusActive, false)
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
			},
			expectedResult: false,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			tt.setupMock(repo, tt.userCtx)

			result, err := service.IsEnabled(context.Background(), tt.flagKey, tt.userCtx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestEvaluateFlag(t *testing.T) {
	tests := []struct {
		name           string
		flagKey        string
		userCtx        *UserContext
		setupMock      func(*mockExperimentsRepository, *UserContext)
		expectedSource string
		expectedEnabled bool
	}{
		{
			name:    "flag not found returns not_found source",
			flagKey: "unknown_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				// Cache refresh returns empty, flag not found
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{}, nil).Once()
			},
			expectedSource:  "not_found",
			expectedEnabled: false,
		},
		{
			name:    "inactive flag returns not_found since not in cache",
			flagKey: "inactive_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				// Inactive flags are not in active cache
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{}, nil).Once()
			},
			expectedSource:  "not_found",
			expectedEnabled: false,
		},
		{
			name:    "override enabled returns override source",
			flagKey: "override_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("override_flag", FlagTypeBoolean, FlagStatusActive, false)
				override := &FlagOverride{
					ID:      uuid.New(),
					FlagID:  flag.ID,
					UserID:  userCtx.UserID,
					Enabled: true,
				}
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return(override, nil).Once()
			},
			expectedSource:  "override",
			expectedEnabled: true,
		},
		{
			name:    "blocked user returns blocked source",
			flagKey: "blocked_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("blocked_flag", FlagTypeBoolean, FlagStatusActive, true)
				flag.BlockedUserIDs = []uuid.UUID{userCtx.UserID}
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
			},
			expectedSource:  "blocked",
			expectedEnabled: false,
		},
		{
			name:    "allowed user returns user_list source",
			flagKey: "allowed_flag",
			userCtx: createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("allowed_flag", FlagTypeUserList, FlagStatusActive, false)
				flag.AllowedUserIDs = []uuid.UUID{userCtx.UserID}
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
			},
			expectedSource:  "user_list",
			expectedEnabled: true,
		},
		{
			name:    "percentage flag no context returns no_context source",
			flagKey: "percentage_flag",
			userCtx: nil,
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("percentage_flag", FlagTypePercentage, FlagStatusActive, true)
				flag.RolloutPercentage = 50
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
			},
			expectedSource:  "no_context",
			expectedEnabled: false,
		},
		{
			name:    "segment flag no context returns default",
			flagKey: "segment_flag",
			userCtx: nil,
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				flag := createTestFlag("segment_flag", FlagTypeSegment, FlagStatusActive, true)
				flag.SegmentRules = &SegmentRules{Countries: []string{"US"}}
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
			},
			expectedSource:  "default",
			expectedEnabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			tt.setupMock(repo, tt.userCtx)

			result, err := service.EvaluateFlag(context.Background(), tt.flagKey, tt.userCtx)

			require.NoError(t, err)
			assert.Equal(t, tt.flagKey, result.Key)
			assert.Equal(t, tt.expectedSource, result.Source)
			assert.Equal(t, tt.expectedEnabled, result.Enabled)
			repo.AssertExpectations(t)
		})
	}
}

func TestEvaluateFlags(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)
	userCtx := createTestUserContext()

	flag1 := createTestFlag("flag1", FlagTypeBoolean, FlagStatusActive, true)
	flag2 := createTestFlag("flag2", FlagTypeBoolean, FlagStatusActive, false)

	repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag1, flag2}, nil).Once()
	repo.On("GetOverride", mock.Anything, flag1.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
	repo.On("GetOverride", mock.Anything, flag2.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

	result, err := service.EvaluateFlags(ctx, []string{"flag1", "flag2"}, userCtx)

	require.NoError(t, err)
	assert.Len(t, result.Flags, 2)
	assert.True(t, result.Flags["flag1"].Enabled)
	assert.False(t, result.Flags["flag2"].Enabled)
	repo.AssertExpectations(t)
}

// ========================================
// PERCENTAGE ROLLOUT TESTS (CONSISTENT HASHING)
// ========================================

func TestPercentageRollout(t *testing.T) {
	tests := []struct {
		name              string
		rolloutPercentage int
		expectAllEnabled  bool
		expectAllDisabled bool
	}{
		{
			name:              "100% rollout enables all",
			rolloutPercentage: 100,
			expectAllEnabled:  true,
		},
		{
			name:              "0% rollout disables all",
			rolloutPercentage: 0,
			expectAllDisabled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			flag := createTestFlag("percentage_flag", FlagTypePercentage, FlagStatusActive, true)
			flag.RolloutPercentage = tt.rolloutPercentage

			// First call refreshes cache, subsequent calls use cache
			repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()

			// Test with multiple users (cache is set after first call)
			for i := 0; i < 10; i++ {
				userCtx := createTestUserContext()
				repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

				result, err := service.EvaluateFlag(context.Background(), "percentage_flag", userCtx)
				require.NoError(t, err)

				if tt.expectAllEnabled {
					assert.True(t, result.Enabled, "Expected enabled for 100% rollout")
				}
				if tt.expectAllDisabled {
					assert.False(t, result.Enabled, "Expected disabled for 0% rollout")
				}
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestPercentageRolloutConsistency(t *testing.T) {
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flag := createTestFlag("consistent_flag", FlagTypePercentage, FlagStatusActive, true)
	flag.RolloutPercentage = 50

	userCtx := createTestUserContext()

	// First call refreshes cache, subsequent calls use cache
	repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()

	// All calls check for override
	for i := 0; i < 5; i++ {
		repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()
	}

	var firstResult bool
	for i := 0; i < 5; i++ {
		result, err := service.EvaluateFlag(context.Background(), "consistent_flag", userCtx)
		require.NoError(t, err)

		if i == 0 {
			firstResult = result.Enabled
		} else {
			assert.Equal(t, firstResult, result.Enabled, "Percentage rollout should be consistent for same user")
		}
	}
	repo.AssertExpectations(t)
}

// ========================================
// SEGMENT MATCHING TESTS
// ========================================

func TestMatchesSegmentRules(t *testing.T) {
	tests := []struct {
		name          string
		userCtx       *UserContext
		segmentRules  *SegmentRules
		expectedMatch bool
	}{
		{
			name: "matches all rules",
			userCtx: &UserContext{
				UserID:      uuid.New(),
				Role:        "rider",
				Country:     "US",
				City:        "San Francisco",
				Platform:    "ios",
				TotalRides:  100,
				Rating:      4.5,
				AccountAge:  180,
				LoyaltyTier: "gold",
			},
			segmentRules: &SegmentRules{
				Roles:       []string{"rider", "driver"},
				Countries:   []string{"US", "CA"},
				Cities:      []string{"San Francisco", "Los Angeles"},
				Platform:    []string{"ios", "android"},
				MinRides:    intPtr(50),
				MinRating:   float64Ptr(4.0),
				AccountAge:  intPtr(90),
				LoyaltyTier: []string{"gold", "platinum"},
			},
			expectedMatch: true,
		},
		{
			name: "fails role check",
			userCtx: &UserContext{
				UserID: uuid.New(),
				Role:   "admin",
			},
			segmentRules: &SegmentRules{
				Roles: []string{"rider", "driver"},
			},
			expectedMatch: false,
		},
		{
			name: "fails country check",
			userCtx: &UserContext{
				UserID:  uuid.New(),
				Country: "MX",
			},
			segmentRules: &SegmentRules{
				Countries: []string{"US", "CA"},
			},
			expectedMatch: false,
		},
		{
			name: "fails city check",
			userCtx: &UserContext{
				UserID: uuid.New(),
				City:   "New York",
			},
			segmentRules: &SegmentRules{
				Cities: []string{"San Francisco", "Los Angeles"},
			},
			expectedMatch: false,
		},
		{
			name: "fails min rides check",
			userCtx: &UserContext{
				UserID:     uuid.New(),
				TotalRides: 10,
			},
			segmentRules: &SegmentRules{
				MinRides: intPtr(50),
			},
			expectedMatch: false,
		},
		{
			name: "fails max rides check",
			userCtx: &UserContext{
				UserID:     uuid.New(),
				TotalRides: 100,
			},
			segmentRules: &SegmentRules{
				MaxRides: intPtr(50),
			},
			expectedMatch: false,
		},
		{
			name: "fails min rating check",
			userCtx: &UserContext{
				UserID: uuid.New(),
				Rating: 3.5,
			},
			segmentRules: &SegmentRules{
				MinRating: float64Ptr(4.0),
			},
			expectedMatch: false,
		},
		{
			name: "fails account age check",
			userCtx: &UserContext{
				UserID:     uuid.New(),
				AccountAge: 30,
			},
			segmentRules: &SegmentRules{
				AccountAge: intPtr(90),
			},
			expectedMatch: false,
		},
		{
			name: "fails platform check",
			userCtx: &UserContext{
				UserID:   uuid.New(),
				Platform: "web",
			},
			segmentRules: &SegmentRules{
				Platform: []string{"ios", "android"},
			},
			expectedMatch: false,
		},
		{
			name: "fails loyalty tier check",
			userCtx: &UserContext{
				UserID:      uuid.New(),
				LoyaltyTier: "silver",
			},
			segmentRules: &SegmentRules{
				LoyaltyTier: []string{"gold", "platinum"},
			},
			expectedMatch: false,
		},
		{
			name:    "empty rules match all",
			userCtx: createTestUserContext(),
			segmentRules: &SegmentRules{
				// All fields empty
			},
			expectedMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			// Create a flag with a consistent ID for each test
			flagID := uuid.New()
			flag := &FeatureFlag{
				ID:           flagID,
				Key:          "segment_flag",
				Name:         "Segment Flag",
				FlagType:     FlagTypeSegment,
				Status:       FlagStatusActive,
				Enabled:      true,
				SegmentRules: tt.segmentRules,
			}

			repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
			repo.On("GetOverride", mock.Anything, flagID, tt.userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

			result, err := service.EvaluateFlag(context.Background(), "segment_flag", tt.userCtx)

			require.NoError(t, err)
			assert.Equal(t, tt.expectedMatch, result.Enabled)
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// CACHE TESTS
// ========================================

func TestCacheHitAndMiss(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)
	userCtx := createTestUserContext()

	flag := createTestFlag("cached_flag", FlagTypeBoolean, FlagStatusActive, true)

	// First call should refresh cache
	repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
	repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Twice()

	// First call - cache miss, fetches from DB
	result1, err := service.EvaluateFlag(ctx, "cached_flag", userCtx)
	require.NoError(t, err)
	assert.True(t, result1.Enabled)

	// Second call - cache hit (within TTL)
	result2, err := service.EvaluateFlag(ctx, "cached_flag", userCtx)
	require.NoError(t, err)
	assert.True(t, result2.Enabled)

	repo.AssertExpectations(t)
}

func TestCacheInvalidation(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)
	adminID := uuid.New()

	flag := createTestFlag("invalidate_flag", FlagTypeBoolean, FlagStatusActive, true)

	// First call populates cache
	repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
	repo.On("GetOverride", mock.Anything, flag.ID, mock.AnythingOfType("uuid.UUID")).Return((*FlagOverride)(nil), nil).Once()

	userCtx := createTestUserContext()
	_, err := service.EvaluateFlag(ctx, "invalidate_flag", userCtx)
	require.NoError(t, err)

	// Update flag - should invalidate cache
	repo.On("GetFlagByID", mock.Anything, flag.ID).Return(flag, nil).Once()
	repo.On("UpdateFlag", mock.Anything, mock.AnythingOfType("*experiments.FeatureFlag")).Return(nil).Once()

	newName := "Updated Flag"
	err = service.UpdateFlag(ctx, flag.ID, &UpdateFlagRequest{Name: &newName})
	require.NoError(t, err)

	// Next call should fetch fresh data from DB (cache was invalidated)
	repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
	repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

	_, err = service.EvaluateFlag(ctx, "invalidate_flag", userCtx)
	require.NoError(t, err)

	repo.AssertExpectations(t)

	// Verify that CreateFlag was called by admin
	_ = adminID // Used to demonstrate admin-level operations
}

// ========================================
// FEATURE FLAG MANAGEMENT TESTS
// ========================================

func TestCreateFlag(t *testing.T) {
	tests := []struct {
		name        string
		req         *CreateFlagRequest
		setupMock   func(*mockExperimentsRepository)
		expectError bool
		errorType   string
	}{
		{
			name: "successful creation",
			req: &CreateFlagRequest{
				Key:      "new_flag",
				Name:     "New Flag",
				FlagType: FlagTypeBoolean,
				Enabled:  true,
			},
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetFlagByKey", mock.Anything, "new_flag").Return((*FeatureFlag)(nil), nil).Once()
				repo.On("CreateFlag", mock.Anything, mock.MatchedBy(func(flag *FeatureFlag) bool {
					return flag.Key == "new_flag" && flag.Status == FlagStatusActive
				})).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "duplicate key fails",
			req: &CreateFlagRequest{
				Key:      "existing_flag",
				Name:     "Existing Flag",
				FlagType: FlagTypeBoolean,
			},
			setupMock: func(repo *mockExperimentsRepository) {
				existingFlag := createTestFlag("existing_flag", FlagTypeBoolean, FlagStatusActive, true)
				repo.On("GetFlagByKey", mock.Anything, "existing_flag").Return(existingFlag, nil).Once()
			},
			expectError: true,
			errorType:   "bad_request",
		},
		{
			name: "repository error fails",
			req: &CreateFlagRequest{
				Key:      "new_flag",
				Name:     "New Flag",
				FlagType: FlagTypeBoolean,
			},
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetFlagByKey", mock.Anything, "new_flag").Return((*FeatureFlag)(nil), nil).Once()
				repo.On("CreateFlag", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
			},
			expectError: true,
			errorType:   "internal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)
			adminID := uuid.New()

			tt.setupMock(repo)

			flag, err := service.CreateFlag(context.Background(), adminID, tt.req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, flag)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, flag)
				assert.Equal(t, tt.req.Key, flag.Key)
				assert.Equal(t, FlagStatusActive, flag.Status)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestUpdateFlag(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()
	flag := &FeatureFlag{
		ID:       flagID,
		Key:      "test_flag",
		Name:     "Original Name",
		Enabled:  false,
		FlagType: FlagTypeBoolean,
		Status:   FlagStatusActive,
	}

	repo.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil).Once()
	repo.On("UpdateFlag", mock.Anything, mock.MatchedBy(func(f *FeatureFlag) bool {
		return f.Name == "Updated Name" && f.Enabled == true
	})).Return(nil).Once()

	newName := "Updated Name"
	enabled := true
	err := service.UpdateFlag(ctx, flagID, &UpdateFlagRequest{
		Name:    &newName,
		Enabled: &enabled,
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestToggleFlag(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()
	flag := &FeatureFlag{
		ID:      flagID,
		Key:     "toggle_flag",
		Enabled: false,
		Status:  FlagStatusActive,
	}

	repo.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil).Once()
	repo.On("UpdateFlag", mock.Anything, mock.MatchedBy(func(f *FeatureFlag) bool {
		return f.Enabled == true
	})).Return(nil).Once()

	err := service.ToggleFlag(ctx, flagID, true)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestArchiveFlag(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()

	repo.On("UpdateFlagStatus", mock.Anything, flagID, FlagStatusArchived).Return(nil).Once()

	err := service.ArchiveFlag(ctx, flagID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListFlags(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flags := []*FeatureFlag{
		createTestFlag("flag1", FlagTypeBoolean, FlagStatusActive, true),
		createTestFlag("flag2", FlagTypeBoolean, FlagStatusActive, false),
	}

	activeStatus := FlagStatusActive
	repo.On("ListFlags", mock.Anything, &activeStatus, 50, 0).Return(flags, nil).Once()

	result, err := service.ListFlags(ctx, &activeStatus, 0, 0)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	repo.AssertExpectations(t)
}

// ========================================
// FLAG OVERRIDE TESTS
// ========================================

func TestCreateOverride(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()
	adminID := uuid.New()
	userID := uuid.New()

	flag := &FeatureFlag{ID: flagID, Key: "override_flag"}

	repo.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil).Once()
	repo.On("CreateOverride", mock.Anything, mock.MatchedBy(func(o *FlagOverride) bool {
		return o.FlagID == flagID && o.UserID == userID && o.Enabled == true
	})).Return(nil).Once()

	err := service.CreateOverride(ctx, adminID, flagID, &CreateOverrideRequest{
		UserID:  userID,
		Enabled: true,
		Reason:  "Test override",
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateOverrideWithExpiry(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()
	adminID := uuid.New()
	userID := uuid.New()

	flag := &FeatureFlag{ID: flagID, Key: "override_flag"}
	expiresAt := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	repo.On("GetFlagByID", mock.Anything, flagID).Return(flag, nil).Once()
	repo.On("CreateOverride", mock.Anything, mock.MatchedBy(func(o *FlagOverride) bool {
		return o.ExpiresAt != nil
	})).Return(nil).Once()

	err := service.CreateOverride(ctx, adminID, flagID, &CreateOverrideRequest{
		UserID:    userID,
		Enabled:   true,
		Reason:    "Temporary override",
		ExpiresAt: &expiresAt,
	})

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeleteOverride(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	flagID := uuid.New()
	userID := uuid.New()

	repo.On("DeleteOverride", mock.Anything, flagID, userID).Return(nil).Once()

	err := service.DeleteOverride(ctx, flagID, userID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// A/B EXPERIMENT TESTS
// ========================================

func TestCreateExperiment(t *testing.T) {
	tests := []struct {
		name        string
		req         *CreateExperimentRequest
		setupMock   func(*mockExperimentsRepository)
		expectError bool
		errorMsg    string
	}{
		{
			name: "successful creation",
			req: &CreateExperimentRequest{
				Key:               "new_experiment",
				Name:              "New Experiment",
				Hypothesis:        "Test hypothesis",
				TrafficPercentage: 100,
				PrimaryMetric:     "conversion_rate",
				Variants: []CreateVariantInput{
					{Key: "control", Name: "Control", IsControl: true, Weight: 50},
					{Key: "variant_a", Name: "Variant A", Weight: 50},
				},
			},
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetExperimentByKey", mock.Anything, "new_experiment").Return((*Experiment)(nil), nil).Once()
				repo.On("CreateExperiment", mock.Anything, mock.AnythingOfType("*experiments.Experiment"), mock.AnythingOfType("[]*experiments.Variant")).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "duplicate key fails",
			req: &CreateExperimentRequest{
				Key:  "existing_experiment",
				Name: "Existing Experiment",
				Variants: []CreateVariantInput{
					{Key: "control", Name: "Control", IsControl: true, Weight: 50},
					{Key: "variant_a", Name: "Variant A", Weight: 50},
				},
			},
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("existing_experiment", ExperimentStatusDraft)
				repo.On("GetExperimentByKey", mock.Anything, "existing_experiment").Return(exp, nil).Once()
			},
			expectError: true,
			errorMsg:    "experiment key already exists",
		},
		{
			name: "invalid weights fails",
			req: &CreateExperimentRequest{
				Key:  "bad_weights",
				Name: "Bad Weights",
				Variants: []CreateVariantInput{
					{Key: "control", Name: "Control", IsControl: true, Weight: 40},
					{Key: "variant_a", Name: "Variant A", Weight: 40},
				},
			},
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetExperimentByKey", mock.Anything, "bad_weights").Return((*Experiment)(nil), nil).Once()
			},
			expectError: true,
			errorMsg:    "variant weights must sum to 100",
		},
		{
			name: "no control fails",
			req: &CreateExperimentRequest{
				Key:  "no_control",
				Name: "No Control",
				Variants: []CreateVariantInput{
					{Key: "variant_a", Name: "Variant A", Weight: 50},
					{Key: "variant_b", Name: "Variant B", Weight: 50},
				},
			},
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetExperimentByKey", mock.Anything, "no_control").Return((*Experiment)(nil), nil).Once()
			},
			expectError: true,
			errorMsg:    "at least one variant must be marked as control",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)
			adminID := uuid.New()

			tt.setupMock(repo)

			exp, err := service.CreateExperiment(context.Background(), adminID, tt.req)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, exp)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, exp)
				assert.Equal(t, ExperimentStatusDraft, exp.Status)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestStartExperiment(t *testing.T) {
	tests := []struct {
		name        string
		status      ExperimentStatus
		setupMock   func(*mockExperimentsRepository, uuid.UUID)
		expectError bool
	}{
		{
			name:   "start from draft",
			status: ExperimentStatusDraft,
			setupMock: func(repo *mockExperimentsRepository, expID uuid.UUID) {
				exp := createTestExperiment("test", ExperimentStatusDraft)
				exp.ID = expID
				repo.On("GetExperimentByID", mock.Anything, expID).Return(exp, nil).Once()
				repo.On("UpdateExperimentStatus", mock.Anything, expID, ExperimentStatusRunning).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:   "start from paused",
			status: ExperimentStatusPaused,
			setupMock: func(repo *mockExperimentsRepository, expID uuid.UUID) {
				exp := createTestExperiment("test", ExperimentStatusPaused)
				exp.ID = expID
				repo.On("GetExperimentByID", mock.Anything, expID).Return(exp, nil).Once()
				repo.On("UpdateExperimentStatus", mock.Anything, expID, ExperimentStatusRunning).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name:   "cannot start completed",
			status: ExperimentStatusCompleted,
			setupMock: func(repo *mockExperimentsRepository, expID uuid.UUID) {
				exp := createTestExperiment("test", ExperimentStatusCompleted)
				exp.ID = expID
				repo.On("GetExperimentByID", mock.Anything, expID).Return(exp, nil).Once()
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)
			expID := uuid.New()

			tt.setupMock(repo, expID)

			err := service.StartExperiment(context.Background(), expID)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestPauseExperiment(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	expID := uuid.New()
	repo.On("UpdateExperimentStatus", mock.Anything, expID, ExperimentStatusPaused).Return(nil).Once()

	err := service.PauseExperiment(ctx, expID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestConcludeExperiment(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	expID := uuid.New()
	repo.On("UpdateExperimentStatus", mock.Anything, expID, ExperimentStatusCompleted).Return(nil).Once()

	err := service.ConcludeExperiment(ctx, expID)

	require.NoError(t, err)
	repo.AssertExpectations(t)
}

// ========================================
// VARIANT ASSIGNMENT TESTS (CONSISTENT HASHING)
// ========================================

func TestGetVariantForUser(t *testing.T) {
	tests := []struct {
		name           string
		experimentKey  string
		userCtx        *UserContext
		setupMock      func(*mockExperimentsRepository, *UserContext)
		expectVariant  bool
		expectNil      bool
	}{
		{
			name:          "experiment not found returns nil",
			experimentKey: "unknown_exp",
			userCtx:       createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				repo.On("GetExperimentByKey", mock.Anything, "unknown_exp").Return((*Experiment)(nil), nil).Once()
			},
			expectNil: true,
		},
		{
			name:          "experiment not running returns nil",
			experimentKey: "draft_exp",
			userCtx:       createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				exp := createTestExperiment("draft_exp", ExperimentStatusDraft)
				repo.On("GetExperimentByKey", mock.Anything, "draft_exp").Return(exp, nil).Once()
			},
			expectNil: true,
		},
		{
			name:          "user not in traffic percentage returns nil",
			experimentKey: "low_traffic_exp",
			userCtx:       createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				exp := createTestExperiment("low_traffic_exp", ExperimentStatusRunning)
				exp.TrafficPercentage = 0 // 0% traffic
				repo.On("GetExperimentByKey", mock.Anything, "low_traffic_exp").Return(exp, nil).Once()
			},
			expectNil: true,
		},
		{
			name:          "user fails segment rules returns nil",
			experimentKey: "segment_exp",
			userCtx: &UserContext{
				UserID:  uuid.New(),
				Country: "MX",
			},
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				exp := createTestExperiment("segment_exp", ExperimentStatusRunning)
				exp.SegmentRules = &SegmentRules{Countries: []string{"US"}}
				repo.On("GetExperimentByKey", mock.Anything, "segment_exp").Return(exp, nil).Once()
			},
			expectNil: true,
		},
		{
			name:          "returns existing assignment",
			experimentKey: "assigned_exp",
			userCtx:       createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				exp := createTestExperiment("assigned_exp", ExperimentStatusRunning)
				variants := createTestVariants(exp.ID)
				assignment := &ExperimentAssignment{
					ID:           uuid.New(),
					ExperimentID: exp.ID,
					UserID:       userCtx.UserID,
					VariantID:    variants[0].ID,
				}
				repo.On("GetExperimentByKey", mock.Anything, "assigned_exp").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, userCtx.UserID).Return(assignment, nil).Once()
				repo.On("GetVariants", mock.Anything, exp.ID).Return(variants, nil).Once()
			},
			expectVariant: true,
		},
		{
			name:          "assigns new user to variant",
			experimentKey: "new_assignment_exp",
			userCtx:       createTestUserContext(),
			setupMock: func(repo *mockExperimentsRepository, userCtx *UserContext) {
				exp := createTestExperiment("new_assignment_exp", ExperimentStatusRunning)
				variants := createTestVariants(exp.ID)
				repo.On("GetExperimentByKey", mock.Anything, "new_assignment_exp").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, userCtx.UserID).Return((*ExperimentAssignment)(nil), nil).Once()
				repo.On("GetVariants", mock.Anything, exp.ID).Return(variants, nil).Once()
				repo.On("CreateAssignment", mock.Anything, mock.AnythingOfType("*experiments.ExperimentAssignment")).Return(nil).Once()
			},
			expectVariant: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			tt.setupMock(repo, tt.userCtx)

			variant, err := service.GetVariantForUser(context.Background(), tt.experimentKey, tt.userCtx)

			require.NoError(t, err)
			if tt.expectNil {
				assert.Nil(t, variant)
			}
			if tt.expectVariant {
				assert.NotNil(t, variant)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestVariantAssignmentConsistency(t *testing.T) {
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)
	userCtx := createTestUserContext()

	exp := createTestExperiment("consistent_exp", ExperimentStatusRunning)
	variants := createTestVariants(exp.ID)

	// Setup mock for multiple calls
	repo.On("GetExperimentByKey", mock.Anything, "consistent_exp").Return(exp, nil).Times(5)
	repo.On("GetAssignment", mock.Anything, exp.ID, userCtx.UserID).Return((*ExperimentAssignment)(nil), nil).Times(5)
	repo.On("GetVariants", mock.Anything, exp.ID).Return(variants, nil).Times(5)
	repo.On("CreateAssignment", mock.Anything, mock.AnythingOfType("*experiments.ExperimentAssignment")).Return(nil).Times(5)

	var firstVariantID uuid.UUID
	for i := 0; i < 5; i++ {
		variant, err := service.GetVariantForUser(context.Background(), "consistent_exp", userCtx)
		require.NoError(t, err)
		require.NotNil(t, variant)

		if i == 0 {
			firstVariantID = variant.ID
		} else {
			assert.Equal(t, firstVariantID, variant.ID, "Variant assignment should be consistent for same user")
		}
	}
	repo.AssertExpectations(t)
}

// ========================================
// EVENT TRACKING TESTS
// ========================================

func TestTrackEvent(t *testing.T) {
	tests := []struct {
		name        string
		req         *TrackEventRequest
		setupMock   func(*mockExperimentsRepository, uuid.UUID)
		expectError bool
	}{
		{
			name: "successful event tracking",
			req: &TrackEventRequest{
				ExperimentKey: "track_exp",
				EventType:     "conversion",
				EventValue:    float64Ptr(100.0),
			},
			setupMock: func(repo *mockExperimentsRepository, userID uuid.UUID) {
				exp := createTestExperiment("track_exp", ExperimentStatusRunning)
				assignment := &ExperimentAssignment{
					ID:           uuid.New(),
					ExperimentID: exp.ID,
					UserID:       userID,
					VariantID:    uuid.New(),
				}
				repo.On("GetExperimentByKey", mock.Anything, "track_exp").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, userID).Return(assignment, nil).Once()
				repo.On("RecordEvent", mock.Anything, mock.MatchedBy(func(e *ExperimentEvent) bool {
					return e.EventType == "conversion" && e.ExperimentID == exp.ID
				})).Return(nil).Once()
			},
			expectError: false,
		},
		{
			name: "unknown experiment returns nil gracefully",
			req: &TrackEventRequest{
				ExperimentKey: "unknown_exp",
				EventType:     "conversion",
			},
			setupMock: func(repo *mockExperimentsRepository, userID uuid.UUID) {
				repo.On("GetExperimentByKey", mock.Anything, "unknown_exp").Return((*Experiment)(nil), nil).Once()
			},
			expectError: false, // Graceful degradation
		},
		{
			name: "user not assigned returns nil gracefully",
			req: &TrackEventRequest{
				ExperimentKey: "unassigned_exp",
				EventType:     "conversion",
			},
			setupMock: func(repo *mockExperimentsRepository, userID uuid.UUID) {
				exp := createTestExperiment("unassigned_exp", ExperimentStatusRunning)
				repo.On("GetExperimentByKey", mock.Anything, "unassigned_exp").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, userID).Return((*ExperimentAssignment)(nil), nil).Once()
			},
			expectError: false, // Graceful degradation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)
			userID := uuid.New()

			tt.setupMock(repo, userID)

			err := service.TrackEvent(context.Background(), userID, tt.req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// STATISTICAL ANALYSIS TESTS
// ========================================

func TestGetResults(t *testing.T) {
	ctx := context.Background()
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	expID := uuid.New()
	exp := &Experiment{
		ID:              expID,
		Key:             "results_exp",
		Status:          ExperimentStatusRunning,
		MinSampleSize:   100,
		ConfidenceLevel: 0.95,
	}

	metrics := []VariantMetrics{
		{
			VariantID:      uuid.New(),
			VariantKey:     "control",
			SampleSize:     500,
			Conversions:    50,
			ConversionRate: 0.10,
		},
		{
			VariantID:      uuid.New(),
			VariantKey:     "variant_a",
			SampleSize:     500,
			Conversions:    75,
			ConversionRate: 0.15,
		},
	}

	repo.On("GetExperimentByID", mock.Anything, expID).Return(exp, nil).Once()
	repo.On("GetVariantMetrics", mock.Anything, expID).Return(metrics, nil).Once()

	results, err := service.GetResults(ctx, expID)

	require.NoError(t, err)
	assert.NotNil(t, results)
	assert.Len(t, results.Variants, 2)
	assert.True(t, results.CanConclude) // Both have >= 100 samples
	repo.AssertExpectations(t)
}

func TestAnalyzeResultsStatisticalSignificance(t *testing.T) {
	tests := []struct {
		name              string
		controlRate       float64
		controlSize       int
		variantRate       float64
		variantSize       int
		minSampleSize     int
		confidenceLevel   float64
		expectSignificant bool
		expectCanConclude bool
	}{
		{
			name:              "significant result 95% confidence",
			controlRate:       0.10,
			controlSize:       1000,
			variantRate:       0.15,
			variantSize:       1000,
			minSampleSize:     100,
			confidenceLevel:   0.95,
			expectSignificant: true,
			expectCanConclude: true,
		},
		{
			name:              "not significant small difference",
			controlRate:       0.10,
			controlSize:       1000,
			variantRate:       0.101,
			variantSize:       1000,
			minSampleSize:     100,
			confidenceLevel:   0.95,
			expectSignificant: false,
			expectCanConclude: true,
		},
		{
			name:              "insufficient sample size",
			controlRate:       0.10,
			controlSize:       50,
			variantRate:       0.20,
			variantSize:       50,
			minSampleSize:     100,
			confidenceLevel:   0.95,
			expectSignificant: false,
			expectCanConclude: false,
		},
		{
			name:              "99% confidence level stricter",
			controlRate:       0.10,
			controlSize:       500,
			variantRate:       0.12,
			variantSize:       500,
			minSampleSize:     100,
			confidenceLevel:   0.99,
			expectSignificant: false, // 2% diff might not be significant at 99%
			expectCanConclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			expID := uuid.New()
			exp := &Experiment{
				ID:              expID,
				MinSampleSize:   tt.minSampleSize,
				ConfidenceLevel: tt.confidenceLevel,
			}

			results := &ExperimentResults{
				Experiment: exp,
				Variants: []VariantMetrics{
					{
						VariantKey:     "control",
						SampleSize:     tt.controlSize,
						ConversionRate: tt.controlRate,
					},
					{
						VariantKey:     "variant_a",
						SampleSize:     tt.variantSize,
						ConversionRate: tt.variantRate,
					},
				},
			}

			service.analyzeResults(results, tt.minSampleSize, tt.confidenceLevel)

			assert.Equal(t, tt.expectCanConclude, results.CanConclude)
			assert.Equal(t, tt.expectSignificant, results.IsSignificant)
		})
	}
}

func TestAnalyzeResultsUpliftCalculation(t *testing.T) {
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	expID := uuid.New()
	exp := &Experiment{
		ID:              expID,
		MinSampleSize:   100,
		ConfidenceLevel: 0.95,
	}

	results := &ExperimentResults{
		Experiment: exp,
		Variants: []VariantMetrics{
			{
				VariantKey:     "control",
				SampleSize:     1000,
				ConversionRate: 0.10, // 10%
			},
			{
				VariantKey:     "variant_a",
				SampleSize:     1000,
				ConversionRate: 0.15, // 15%
			},
		},
	}

	service.analyzeResults(results, 100, 0.95)

	require.NotNil(t, results.Uplift)
	assert.InDelta(t, 50.0, *results.Uplift, 0.1) // 50% uplift (0.15 - 0.10) / 0.10
}

func TestAnalyzeResultsInsufficientVariants(t *testing.T) {
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	results := &ExperimentResults{
		Variants: []VariantMetrics{
			{VariantKey: "control", SampleSize: 100, ConversionRate: 0.10},
		},
	}

	service.analyzeResults(results, 100, 0.95)

	assert.Equal(t, "insufficient_variants", results.RecommendedAction)
}

func TestAnalyzeResultsPValueCalculation(t *testing.T) {
	repo := new(mockExperimentsRepository)
	service := newTestService(repo)

	expID := uuid.New()
	exp := &Experiment{
		ID:              expID,
		MinSampleSize:   100,
		ConfidenceLevel: 0.95,
	}

	results := &ExperimentResults{
		Experiment: exp,
		Variants: []VariantMetrics{
			{
				VariantKey:     "control",
				SampleSize:     1000,
				ConversionRate: 0.10,
			},
			{
				VariantKey:     "variant_a",
				SampleSize:     1000,
				ConversionRate: 0.15,
			},
		},
	}

	service.analyzeResults(results, 100, 0.95)

	require.NotNil(t, results.PValue)
	assert.Less(t, *results.PValue, 0.05) // Should be significant
}

// ========================================
// GRACEFUL DEGRADATION TESTS
// ========================================

func TestGracefulDegradationPatterns(t *testing.T) {
	tests := []struct {
		name        string
		description string
		setupMock   func(*mockExperimentsRepository)
		testFunc    func(*Service) error
		expectError bool
	}{
		{
			name:        "GetVariantForUser returns nil for unknown experiment",
			description: "Pattern 1: return nil, nil for not found",
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetExperimentByKey", mock.Anything, "unknown").Return((*Experiment)(nil), nil).Once()
			},
			testFunc: func(s *Service) error {
				variant, err := s.GetVariantForUser(context.Background(), "unknown", createTestUserContext())
				if err != nil {
					return err
				}
				if variant != nil {
					return errors.New("expected nil variant")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "TrackEvent silently ignores unknown experiments",
			description: "Pattern 2: return nil for tracking unknown",
			setupMock: func(repo *mockExperimentsRepository) {
				repo.On("GetExperimentByKey", mock.Anything, "unknown").Return((*Experiment)(nil), nil).Once()
			},
			testFunc: func(s *Service) error {
				return s.TrackEvent(context.Background(), uuid.New(), &TrackEventRequest{
					ExperimentKey: "unknown",
					EventType:     "conversion",
				})
			},
			expectError: false,
		},
		{
			name:        "TrackEvent silently ignores unassigned users",
			description: "Pattern 3: return nil for unassigned user",
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("exp", ExperimentStatusRunning)
				repo.On("GetExperimentByKey", mock.Anything, "exp").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, mock.AnythingOfType("uuid.UUID")).Return((*ExperimentAssignment)(nil), nil).Once()
			},
			testFunc: func(s *Service) error {
				return s.TrackEvent(context.Background(), uuid.New(), &TrackEventRequest{
					ExperimentKey: "exp",
					EventType:     "conversion",
				})
			},
			expectError: false,
		},
		{
			name:        "GetVariantForUser returns nil for non-running experiment",
			description: "Pattern 4: return nil for inactive",
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("draft", ExperimentStatusDraft)
				repo.On("GetExperimentByKey", mock.Anything, "draft").Return(exp, nil).Once()
			},
			testFunc: func(s *Service) error {
				variant, err := s.GetVariantForUser(context.Background(), "draft", createTestUserContext())
				if err != nil {
					return err
				}
				if variant != nil {
					return errors.New("expected nil variant for draft experiment")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "GetVariantForUser returns nil for user outside traffic",
			description: "Pattern 5: return nil for out of traffic",
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("zero_traffic", ExperimentStatusRunning)
				exp.TrafficPercentage = 0
				repo.On("GetExperimentByKey", mock.Anything, "zero_traffic").Return(exp, nil).Once()
			},
			testFunc: func(s *Service) error {
				variant, err := s.GetVariantForUser(context.Background(), "zero_traffic", createTestUserContext())
				if err != nil {
					return err
				}
				if variant != nil {
					return errors.New("expected nil variant for 0% traffic")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "GetVariantForUser returns nil for segment mismatch",
			description: "Pattern 6: return nil for segment mismatch",
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("segment_exp", ExperimentStatusRunning)
				exp.SegmentRules = &SegmentRules{Countries: []string{"CA"}}
				repo.On("GetExperimentByKey", mock.Anything, "segment_exp").Return(exp, nil).Once()
			},
			testFunc: func(s *Service) error {
				userCtx := &UserContext{UserID: uuid.New(), Country: "US"}
				variant, err := s.GetVariantForUser(context.Background(), "segment_exp", userCtx)
				if err != nil {
					return err
				}
				if variant != nil {
					return errors.New("expected nil variant for segment mismatch")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "GetVariantForUser returns nil for no variants",
			description: "Pattern 7: return nil for empty variants",
			setupMock: func(repo *mockExperimentsRepository) {
				exp := createTestExperiment("no_variants", ExperimentStatusRunning)
				repo.On("GetExperimentByKey", mock.Anything, "no_variants").Return(exp, nil).Once()
				repo.On("GetAssignment", mock.Anything, exp.ID, mock.AnythingOfType("uuid.UUID")).Return((*ExperimentAssignment)(nil), nil).Once()
				repo.On("GetVariants", mock.Anything, exp.ID).Return([]*Variant{}, nil).Once()
			},
			testFunc: func(s *Service) error {
				variant, err := s.GetVariantForUser(context.Background(), "no_variants", createTestUserContext())
				if err != nil {
					return err
				}
				if variant != nil {
					return errors.New("expected nil variant for no variants")
				}
				return nil
			},
			expectError: false,
		},
		{
			name:        "EvaluateFlag returns false for not found flag",
			description: "Pattern 8: return false for not found flag",
			setupMock: func(repo *mockExperimentsRepository) {
				// Cache refresh returns empty, flag not in cache = not found
				repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{}, nil).Once()
			},
			testFunc: func(s *Service) error {
				result, err := s.EvaluateFlag(context.Background(), "unknown_flag", createTestUserContext())
				if err != nil {
					return err
				}
				if result.Enabled {
					return errors.New("expected disabled for unknown flag")
				}
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := new(mockExperimentsRepository)
			service := newTestService(repo)

			tt.setupMock(repo)

			err := tt.testFunc(service)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			repo.AssertExpectations(t)
		})
	}
}

// ========================================
// EDGE CASE TESTS
// ========================================

func TestEdgeCases(t *testing.T) {
	t.Run("nil user context for boolean flag", func(t *testing.T) {
		repo := new(mockExperimentsRepository)
		service := newTestService(repo)

		flag := createTestFlag("nil_ctx_flag", FlagTypeBoolean, FlagStatusActive, true)
		repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()

		result, err := service.EvaluateFlag(context.Background(), "nil_ctx_flag", nil)

		require.NoError(t, err)
		assert.True(t, result.Enabled)
		assert.Equal(t, "default", result.Source)
		repo.AssertExpectations(t)
	})

	t.Run("empty segment rules match all", func(t *testing.T) {
		repo := new(mockExperimentsRepository)
		service := newTestService(repo)
		userCtx := createTestUserContext()

		flag := createTestFlag("empty_segment", FlagTypeSegment, FlagStatusActive, false)
		flag.SegmentRules = &SegmentRules{}

		repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
		repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

		result, err := service.EvaluateFlag(context.Background(), "empty_segment", userCtx)

		require.NoError(t, err)
		assert.True(t, result.Enabled) // Empty rules match all
		repo.AssertExpectations(t)
	})

	t.Run("nil segment rules uses default", func(t *testing.T) {
		repo := new(mockExperimentsRepository)
		service := newTestService(repo)
		userCtx := createTestUserContext()

		flag := createTestFlag("nil_segment", FlagTypeSegment, FlagStatusActive, true)
		flag.SegmentRules = nil

		repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
		repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

		result, err := service.EvaluateFlag(context.Background(), "nil_segment", userCtx)

		require.NoError(t, err)
		assert.True(t, result.Enabled)
		assert.Equal(t, "default", result.Source)
		repo.AssertExpectations(t)
	})

	t.Run("zero min sample size defaults", func(t *testing.T) {
		repo := new(mockExperimentsRepository)
		service := newTestService(repo)
		adminID := uuid.New()

		repo.On("GetExperimentByKey", mock.Anything, "default_sample").Return((*Experiment)(nil), nil).Once()
		repo.On("CreateExperiment", mock.Anything, mock.MatchedBy(func(exp *Experiment) bool {
			return exp.MinSampleSize == 100 && exp.ConfidenceLevel == 0.95
		}), mock.Anything).Return(nil).Once()

		_, err := service.CreateExperiment(context.Background(), adminID, &CreateExperimentRequest{
			Key:             "default_sample",
			Name:            "Default Sample",
			Hypothesis:      "Test",
			MinSampleSize:   0, // Should default to 100
			ConfidenceLevel: 0, // Should default to 0.95
			Variants: []CreateVariantInput{
				{Key: "control", Name: "Control", IsControl: true, Weight: 50},
				{Key: "variant", Name: "Variant", Weight: 50},
			},
		})

		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("user list flag not in list returns false", func(t *testing.T) {
		repo := new(mockExperimentsRepository)
		service := newTestService(repo)
		userCtx := createTestUserContext()

		flag := createTestFlag("user_list_flag", FlagTypeUserList, FlagStatusActive, true)
		flag.AllowedUserIDs = []uuid.UUID{uuid.New()} // Different user

		repo.On("GetAllActiveFlags", mock.Anything).Return([]*FeatureFlag{flag}, nil).Once()
		repo.On("GetOverride", mock.Anything, flag.ID, userCtx.UserID).Return((*FlagOverride)(nil), nil).Once()

		result, err := service.EvaluateFlag(context.Background(), "user_list_flag", userCtx)

		require.NoError(t, err)
		assert.False(t, result.Enabled)
		assert.Equal(t, "user_list", result.Source)
		repo.AssertExpectations(t)
	})
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}
