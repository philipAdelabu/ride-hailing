package verification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/config"
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

func (m *mockRepo) CreateBackgroundCheck(ctx context.Context, check *BackgroundCheck) error {
	args := m.Called(ctx, check)
	return args.Error(0)
}

func (m *mockRepo) GetBackgroundCheck(ctx context.Context, checkID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, checkID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) GetBackgroundCheckByExternalID(ctx context.Context, provider BackgroundCheckProvider, externalID string) (*BackgroundCheck, error) {
	args := m.Called(ctx, provider, externalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) GetLatestBackgroundCheck(ctx context.Context, driverID uuid.UUID) (*BackgroundCheck, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) UpdateBackgroundCheckStatus(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, notes *string, failureReasons []string) error {
	args := m.Called(ctx, checkID, status, notes, failureReasons)
	return args.Error(0)
}

func (m *mockRepo) UpdateBackgroundCheckStarted(ctx context.Context, checkID uuid.UUID, externalID string) error {
	args := m.Called(ctx, checkID, externalID)
	return args.Error(0)
}

func (m *mockRepo) UpdateBackgroundCheckCompleted(ctx context.Context, checkID uuid.UUID, status BackgroundCheckStatus, reportURL *string, expiresAt *time.Time) error {
	args := m.Called(ctx, checkID, status, reportURL, expiresAt)
	return args.Error(0)
}

func (m *mockRepo) GetPendingBackgroundChecks(ctx context.Context, limit int) ([]*BackgroundCheck, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*BackgroundCheck), args.Error(1)
}

func (m *mockRepo) CreateSelfieVerification(ctx context.Context, verification *SelfieVerification) error {
	args := m.Called(ctx, verification)
	return args.Error(0)
}

func (m *mockRepo) GetSelfieVerification(ctx context.Context, verificationID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, verificationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *mockRepo) GetLatestSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *mockRepo) GetTodaysSelfieVerification(ctx context.Context, driverID uuid.UUID) (*SelfieVerification, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*SelfieVerification), args.Error(1)
}

func (m *mockRepo) UpdateSelfieVerificationResult(ctx context.Context, verificationID uuid.UUID, status SelfieVerificationStatus, confidenceScore *float64, matchResult *bool, failureReason *string) error {
	args := m.Called(ctx, verificationID, status, confidenceScore, matchResult, failureReason)
	return args.Error(0)
}

func (m *mockRepo) GetDriverReferencePhoto(ctx context.Context, driverID uuid.UUID) (*string, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*string), args.Error(1)
}

func (m *mockRepo) GetDriverVerificationStatus(ctx context.Context, driverID uuid.UUID) (*DriverVerificationStatus, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverVerificationStatus), args.Error(1)
}

func (m *mockRepo) UpdateDriverApproval(ctx context.Context, driverID uuid.UUID, approved bool, approvedBy *uuid.UUID, reason *string) error {
	args := m.Called(ctx, driverID, approved, approvedBy, reason)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	cfg := &config.Config{}
	return NewService(repo, cfg)
}

func newTestServiceWithConfig(repo RepositoryInterface, cfg *config.Config) *Service {
	return NewService(repo, cfg)
}

// ========================================
// TESTS: InitiateBackgroundCheck
// ========================================

func TestInitiateBackgroundCheck(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	baseRequest := &InitiateBackgroundCheckRequest{
		DriverID:      driverID,
		Provider:      ProviderMock,
		FirstName:     "John",
		LastName:      "Doe",
		DateOfBirth:   "1990-01-15",
		Email:         "john.doe@example.com",
		Phone:         "+1234567890",
		StreetAddress: "123 Main St",
		City:          "San Francisco",
		State:         "CA",
		ZipCode:       "94102",
		LicenseNumber: "D1234567",
		LicenseState:  "CA",
	}

	tests := []struct {
		name       string
		req        *InitiateBackgroundCheckRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *BackgroundCheckResponse)
	}{
		{
			name: "success - initiates background check with mock provider",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				// No existing check
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
				m.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *BackgroundCheckResponse) {
				assert.NotEqual(t, uuid.Nil, resp.CheckID)
				assert.Equal(t, BGCheckStatusInProgress, resp.Status)
				assert.Equal(t, ProviderMock, resp.Provider)
				assert.NotNil(t, resp.ExternalID)
				assert.Contains(t, *resp.ExternalID, "mock-")
				assert.Equal(t, "1-3 business days", resp.EstimatedTime)
			},
		},
		{
			name: "error - uses default provider (checkr) when not specified and fails without API key",
			req: &InitiateBackgroundCheckRequest{
				DriverID:      driverID,
				Provider:      "", // Empty - should default to checkr
				FirstName:     "John",
				LastName:      "Doe",
				DateOfBirth:   "1990-01-15",
				Email:         "john.doe@example.com",
				Phone:         "+1234567890",
				StreetAddress: "123 Main St",
				City:          "San Francisco",
				State:         "CA",
				ZipCode:       "94102",
				LicenseNumber: "D1234567",
				LicenseState:  "CA",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
				// Since checkr API key is not configured, expect the check to fail at provider initiation
				m.On("UpdateBackgroundCheckStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), BGCheckStatusFailed, mock.AnythingOfType("*string"), ([]string)(nil)).Return(nil)
			},
			wantErr:    true,
			errContain: "internal server error", // NewInternalServerError returns this
		},
		{
			name: "error - pending check already exists",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				existingCheck := &BackgroundCheck{
					ID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					DriverID: driverID,
					Status:   BGCheckStatusPending,
				}
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)
			},
			wantErr:    true,
			errContain: "a background check is already in progress",
		},
		{
			name: "error - in-progress check already exists",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				existingCheck := &BackgroundCheck{
					ID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					DriverID: driverID,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)
			},
			wantErr:    true,
			errContain: "a background check is already in progress",
		},
		{
			name: "success - allows new check when previous check is completed/passed",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				existingCheck := &BackgroundCheck{
					ID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					DriverID: driverID,
					Status:   BGCheckStatusPassed, // Previous check passed
				}
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
				m.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *BackgroundCheckResponse) {
				assert.NotEqual(t, uuid.Nil, resp.CheckID)
				assert.Equal(t, BGCheckStatusInProgress, resp.Status)
			},
		},
		{
			name: "success - allows new check when previous check failed",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				existingCheck := &BackgroundCheck{
					ID:       uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					DriverID: driverID,
					Status:   BGCheckStatusFailed, // Previous check failed
				}
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(existingCheck, nil)
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
				m.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *BackgroundCheckResponse) {
				assert.NotEqual(t, uuid.Nil, resp.CheckID)
			},
		},
		{
			name: "error - unsupported provider",
			req: &InitiateBackgroundCheckRequest{
				DriverID:      driverID,
				Provider:      BackgroundCheckProvider("unsupported_provider"),
				FirstName:     "John",
				LastName:      "Doe",
				DateOfBirth:   "1990-01-15",
				Email:         "john.doe@example.com",
				Phone:         "+1234567890",
				StreetAddress: "123 Main St",
				City:          "San Francisco",
				State:         "CA",
				ZipCode:       "94102",
				LicenseNumber: "D1234567",
				LicenseState:  "CA",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)
			},
			wantErr:    true,
			errContain: "unsupported background check provider",
		},
		{
			name: "error - repository error on create",
			req:  baseRequest,
			setupMocks: func(m *mockRepo) {
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(errors.New("database error"))
			},
			wantErr:    true,
			errContain: "internal server error",
		},
		{
			name: "success - uses default check type when not specified",
			req: &InitiateBackgroundCheckRequest{
				DriverID:      driverID,
				Provider:      ProviderMock,
				CheckType:     "", // Empty - should default to driver_standard
				FirstName:     "John",
				LastName:      "Doe",
				DateOfBirth:   "1990-01-15",
				Email:         "john.doe@example.com",
				Phone:         "+1234567890",
				StreetAddress: "123 Main St",
				City:          "San Francisco",
				State:         "CA",
				ZipCode:       "94102",
				LicenseNumber: "D1234567",
				LicenseState:  "CA",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("CreateBackgroundCheck", mock.Anything, mock.MatchedBy(func(check *BackgroundCheck) bool {
					return check.CheckType == "driver_standard"
				})).Return(nil)
				m.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *BackgroundCheckResponse) {
				assert.NotEqual(t, uuid.Nil, resp.CheckID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.InitiateBackgroundCheck(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetBackgroundCheckStatus
// ========================================

func TestGetBackgroundCheckStatus(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		checkID    uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, check *BackgroundCheck)
	}{
		{
			name:    "success - returns background check",
			checkID: checkID,
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:        checkID,
					DriverID:  driverID,
					Provider:  ProviderCheckr,
					Status:    BGCheckStatusInProgress,
					CheckType: "driver_standard",
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetBackgroundCheck", mock.Anything, checkID).Return(check, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, check *BackgroundCheck) {
				assert.Equal(t, checkID, check.ID)
				assert.Equal(t, driverID, check.DriverID)
				assert.Equal(t, BGCheckStatusInProgress, check.Status)
				assert.Equal(t, ProviderCheckr, check.Provider)
			},
		},
		{
			name:    "success - returns completed check with report URL",
			checkID: checkID,
			setupMocks: func(m *mockRepo) {
				reportURL := "https://provider.com/reports/123"
				expiresAt := time.Now().AddDate(1, 0, 0)
				check := &BackgroundCheck{
					ID:        checkID,
					DriverID:  driverID,
					Provider:  ProviderCheckr,
					Status:    BGCheckStatusPassed,
					CheckType: "driver_standard",
					ReportURL: &reportURL,
					ExpiresAt: &expiresAt,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}
				m.On("GetBackgroundCheck", mock.Anything, checkID).Return(check, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, check *BackgroundCheck) {
				assert.Equal(t, BGCheckStatusPassed, check.Status)
				assert.NotNil(t, check.ReportURL)
				assert.NotNil(t, check.ExpiresAt)
			},
		},
		{
			name:    "error - check not found",
			checkID: checkID,
			setupMocks: func(m *mockRepo) {
				m.On("GetBackgroundCheck", mock.Anything, checkID).Return(nil, errors.New("not found"))
			},
			wantErr:    true,
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			check, err := svc.GetBackgroundCheckStatus(context.Background(), tt.checkID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, check)
			} else {
				require.NoError(t, err)
				require.NotNil(t, check)
				if tt.validate != nil {
					tt.validate(t, check)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDriverBackgroundStatus
// ========================================

func TestGetDriverBackgroundStatus(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	checkID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		driverID   uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, check *BackgroundCheck)
	}{
		{
			name:     "success - returns latest check for driver",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:        checkID,
					DriverID:  driverID,
					Provider:  ProviderCheckr,
					Status:    BGCheckStatusPassed,
					CheckType: "driver_standard",
					CreatedAt: time.Now(),
				}
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(check, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, check *BackgroundCheck) {
				assert.Equal(t, checkID, check.ID)
				assert.Equal(t, driverID, check.DriverID)
				assert.Equal(t, BGCheckStatusPassed, check.Status)
			},
		},
		{
			name:     "error - no check found for driver",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
			},
			wantErr:    true,
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			check, err := svc.GetDriverBackgroundStatus(context.Background(), tt.driverID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, check)
			} else {
				require.NoError(t, err)
				require.NotNil(t, check)
				if tt.validate != nil {
					tt.validate(t, check)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: ProcessWebhook
// ========================================

func TestProcessWebhook(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-123456"

	tests := []struct {
		name       string
		payload    *WebhookPayload
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
	}{
		{
			name: "success - checkr clear status maps to passed",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "clear",
				Data: map[string]interface{}{
					"report_url": "https://checkr.com/reports/123",
				},
				Timestamp: time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderCheckr,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, mock.AnythingOfType("*string"), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - checkr consider status maps to failed",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "consider",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderCheckr,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), (*time.Time)(nil)).Return(nil)
				m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), []string{"Background check flagged for review"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - checkr pending status maps to in_progress",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.updated",
				ExternalID: externalID,
				Status:     "pending",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderCheckr,
					Status:   BGCheckStatusPending,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusInProgress, (*string)(nil), (*time.Time)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - sterling pass status maps to passed",
			payload: &WebhookPayload{
				Provider:   ProviderSterling,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "Pass",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderSterling,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderSterling, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - sterling fail status maps to failed",
			payload: &WebhookPayload{
				Provider:   ProviderSterling,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "Fail",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderSterling,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderSterling, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), (*time.Time)(nil)).Return(nil)
				m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), []string{"Background check failed"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - onfido complete with clear result maps to passed",
			payload: &WebhookPayload{
				Provider:   ProviderOnfido,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "complete",
				Data: map[string]interface{}{
					"result": "clear",
				},
				Timestamp: time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderOnfido,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - onfido complete without clear result maps to failed",
			payload: &WebhookPayload{
				Provider:   ProviderOnfido,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "complete",
				Data: map[string]interface{}{
					"result": "consider",
				},
				Timestamp: time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderOnfido,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), (*time.Time)(nil)).Return(nil)
				m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), []string{"Document verification failed"}).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - onfido in_progress status",
			payload: &WebhookPayload{
				Provider:   ProviderOnfido,
				EventType:  "check.updated",
				ExternalID: externalID,
				Status:     "in_progress",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderOnfido,
					Status:   BGCheckStatusPending,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusInProgress, (*string)(nil), (*time.Time)(nil)).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error - check not found",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "clear",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(nil, errors.New("not found"))
			},
			wantErr:    true,
			errContain: "not found",
		},
		{
			name: "error - unsupported provider",
			payload: &WebhookPayload{
				Provider:   BackgroundCheckProvider("unknown"),
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "clear",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: BackgroundCheckProvider("unknown"),
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, BackgroundCheckProvider("unknown"), externalID).Return(check, nil)
			},
			wantErr:    true,
			errContain: "unsupported provider",
		},
		{
			name: "error - update completed fails",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "clear",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderCheckr,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(errors.New("database error"))
			},
			wantErr:    true,
			errContain: "internal server error",
		},
		{
			name: "success - checkr unknown status maps to failed with reason",
			payload: &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     "unknown_status",
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			},
			setupMocks: func(m *mockRepo) {
				check := &BackgroundCheck{
					ID:       checkID,
					DriverID: driverID,
					Provider: ProviderCheckr,
					Status:   BGCheckStatusInProgress,
				}
				m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), (*time.Time)(nil)).Return(nil)
				m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), []string{"Unknown status: unknown_status"}).Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			err := svc.ProcessWebhook(context.Background(), tt.payload)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
			} else {
				require.NoError(t, err)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: VerifySelfie
// ========================================

func TestVerifySelfie(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	rideID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name       string
		req        *SubmitSelfieRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		errContain string
		validate   func(t *testing.T, resp *SelfieVerificationResponse)
	}{
		{
			name: "success - selfie verified with high confidence",
			req: &SubmitSelfieRequest{
				DriverID:  driverID,
				RideID:    &rideID,
				SelfieURL: "https://storage.example.com/selfies/driver-123.jpg",
			},
			setupMocks: func(m *mockRepo) {
				referenceURL := "https://storage.example.com/documents/driver-123-license.jpg"
				m.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&referenceURL, nil)
				m.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(nil)
				// The mock face comparison in the service returns 95.5% confidence and match=true
				m.On("UpdateSelfieVerificationResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), SelfieStatusVerified, mock.AnythingOfType("*float64"), mock.AnythingOfType("*bool"), (*string)(nil)).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *SelfieVerificationResponse) {
				assert.NotEqual(t, uuid.Nil, resp.VerificationID)
				assert.Equal(t, SelfieStatusVerified, resp.Status)
				assert.NotNil(t, resp.ConfidenceScore)
				assert.True(t, *resp.ConfidenceScore >= 90.0)
				assert.NotNil(t, resp.MatchResult)
				assert.True(t, *resp.MatchResult)
				assert.Equal(t, "Identity verified successfully", resp.Message)
			},
		},
		{
			name: "success - selfie verification without ride ID (pre-shift check)",
			req: &SubmitSelfieRequest{
				DriverID:  driverID,
				RideID:    nil,
				SelfieURL: "https://storage.example.com/selfies/driver-123-preshift.jpg",
			},
			setupMocks: func(m *mockRepo) {
				referenceURL := "https://storage.example.com/documents/driver-123-license.jpg"
				m.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&referenceURL, nil)
				m.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(nil)
				m.On("UpdateSelfieVerificationResult", mock.Anything, mock.AnythingOfType("uuid.UUID"), SelfieStatusVerified, mock.AnythingOfType("*float64"), mock.AnythingOfType("*bool"), (*string)(nil)).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, resp *SelfieVerificationResponse) {
				assert.Equal(t, SelfieStatusVerified, resp.Status)
			},
		},
		{
			name: "error - no reference photo found",
			req: &SubmitSelfieRequest{
				DriverID:  driverID,
				SelfieURL: "https://storage.example.com/selfies/driver-123.jpg",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(nil, errors.New("not found"))
			},
			wantErr:    true,
			errContain: "driver reference photo not found",
		},
		{
			name: "error - reference photo URL is nil",
			req: &SubmitSelfieRequest{
				DriverID:  driverID,
				SelfieURL: "https://storage.example.com/selfies/driver-123.jpg",
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverReferencePhoto", mock.Anything, driverID).Return((*string)(nil), nil)
			},
			wantErr:    true,
			errContain: "driver reference photo not found",
		},
		{
			name: "error - create verification record fails",
			req: &SubmitSelfieRequest{
				DriverID:  driverID,
				SelfieURL: "https://storage.example.com/selfies/driver-123.jpg",
			},
			setupMocks: func(m *mockRepo) {
				referenceURL := "https://storage.example.com/documents/driver-123-license.jpg"
				m.On("GetDriverReferencePhoto", mock.Anything, driverID).Return(&referenceURL, nil)
				m.On("CreateSelfieVerification", mock.Anything, mock.AnythingOfType("*verification.SelfieVerification")).Return(errors.New("database error"))
			},
			wantErr:    true,
			errContain: "internal server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			resp, err := svc.VerifySelfie(context.Background(), tt.req)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetSelfieVerificationStatus
// ========================================

func TestGetSelfieVerificationStatus(t *testing.T) {
	verificationID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	tests := []struct {
		name           string
		verificationID uuid.UUID
		setupMocks     func(m *mockRepo)
		wantErr        bool
		errContain     string
		validate       func(t *testing.T, v *SelfieVerification)
	}{
		{
			name:           "success - returns verification",
			verificationID: verificationID,
			setupMocks: func(m *mockRepo) {
				confidence := 95.5
				match := true
				v := &SelfieVerification{
					ID:              verificationID,
					DriverID:        driverID,
					SelfieURL:       "https://storage.example.com/selfies/123.jpg",
					Status:          SelfieStatusVerified,
					ConfidenceScore: &confidence,
					MatchResult:     &match,
					Provider:        "rekognition",
					CreatedAt:       time.Now(),
				}
				m.On("GetSelfieVerification", mock.Anything, verificationID).Return(v, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *SelfieVerification) {
				assert.Equal(t, verificationID, v.ID)
				assert.Equal(t, driverID, v.DriverID)
				assert.Equal(t, SelfieStatusVerified, v.Status)
				assert.NotNil(t, v.ConfidenceScore)
				assert.Equal(t, 95.5, *v.ConfidenceScore)
			},
		},
		{
			name:           "success - returns failed verification",
			verificationID: verificationID,
			setupMocks: func(m *mockRepo) {
				confidence := 75.0
				match := false
				failureReason := "Face does not match reference photo"
				v := &SelfieVerification{
					ID:              verificationID,
					DriverID:        driverID,
					SelfieURL:       "https://storage.example.com/selfies/123.jpg",
					Status:          SelfieStatusFailed,
					ConfidenceScore: &confidence,
					MatchResult:     &match,
					FailureReason:   &failureReason,
					Provider:        "rekognition",
					CreatedAt:       time.Now(),
				}
				m.On("GetSelfieVerification", mock.Anything, verificationID).Return(v, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, v *SelfieVerification) {
				assert.Equal(t, SelfieStatusFailed, v.Status)
				assert.NotNil(t, v.FailureReason)
			},
		},
		{
			name:           "error - verification not found",
			verificationID: verificationID,
			setupMocks: func(m *mockRepo) {
				m.On("GetSelfieVerification", mock.Anything, verificationID).Return(nil, errors.New("not found"))
			},
			wantErr:    true,
			errContain: "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			v, err := svc.GetSelfieVerificationStatus(context.Background(), tt.verificationID)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errContain != "" {
					assert.Contains(t, err.Error(), tt.errContain)
				}
				assert.Nil(t, v)
			} else {
				require.NoError(t, err)
				require.NotNil(t, v)
				if tt.validate != nil {
					tt.validate(t, v)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: RequiresSelfieVerification
// ========================================

func TestRequiresSelfieVerification(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name         string
		driverID     uuid.UUID
		setupMocks   func(m *mockRepo)
		wantRequired bool
		wantErr      bool
	}{
		{
			name:     "requires verification - no verification today",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				m.On("GetTodaysSelfieVerification", mock.Anything, driverID).Return(nil, errors.New("not found"))
			},
			wantRequired: true,
			wantErr:      false,
		},
		{
			name:     "no verification required - already verified today",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				v := &SelfieVerification{
					ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
					DriverID:  driverID,
					Status:    SelfieStatusVerified,
					CreatedAt: time.Now(),
				}
				m.On("GetTodaysSelfieVerification", mock.Anything, driverID).Return(v, nil)
			},
			wantRequired: false,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			required, err := svc.RequiresSelfieVerification(context.Background(), tt.driverID)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRequired, required)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDriverVerificationStatus
// ========================================

func TestGetDriverVerificationStatus(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name       string
		driverID   uuid.UUID
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, status *DriverVerificationStatus)
	}{
		{
			name:     "success - fully verified driver",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				expiresAt := time.Now().AddDate(1, 0, 0)
				status := &DriverVerificationStatus{
					DriverID:                   driverID,
					BackgroundCheckStatus:      BGCheckStatusPassed,
					BackgroundCheckExpiresAt:   &expiresAt,
					SelfieVerificationRequired: false,
					IsFullyVerified:            true,
					VerificationMessage:        "Fully verified and ready to accept rides",
				}
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, status *DriverVerificationStatus) {
				assert.Equal(t, driverID, status.DriverID)
				assert.Equal(t, BGCheckStatusPassed, status.BackgroundCheckStatus)
				assert.True(t, status.IsFullyVerified)
				assert.False(t, status.SelfieVerificationRequired)
				assert.Contains(t, status.VerificationMessage, "Fully verified")
			},
		},
		{
			name:     "success - driver needs selfie verification",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				status := &DriverVerificationStatus{
					DriverID:                   driverID,
					BackgroundCheckStatus:      BGCheckStatusPassed,
					SelfieVerificationRequired: true,
					IsFullyVerified:            false,
					VerificationMessage:        "Daily selfie verification required",
				}
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, status *DriverVerificationStatus) {
				assert.False(t, status.IsFullyVerified)
				assert.True(t, status.SelfieVerificationRequired)
				assert.Contains(t, status.VerificationMessage, "selfie verification")
			},
		},
		{
			name:     "success - driver background check pending",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				status := &DriverVerificationStatus{
					DriverID:                   driverID,
					BackgroundCheckStatus:      BGCheckStatusPending,
					SelfieVerificationRequired: true,
					IsFullyVerified:            false,
					VerificationMessage:        "Background check not passed",
				}
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, status *DriverVerificationStatus) {
				assert.False(t, status.IsFullyVerified)
				assert.Equal(t, BGCheckStatusPending, status.BackgroundCheckStatus)
			},
		},
		{
			name:     "success - driver background check in progress",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				status := &DriverVerificationStatus{
					DriverID:                   driverID,
					BackgroundCheckStatus:      BGCheckStatusInProgress,
					SelfieVerificationRequired: true,
					IsFullyVerified:            false,
					VerificationMessage:        "Background check not passed",
				}
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, status *DriverVerificationStatus) {
				assert.False(t, status.IsFullyVerified)
				assert.Equal(t, BGCheckStatusInProgress, status.BackgroundCheckStatus)
			},
		},
		{
			name:     "success - driver background check failed",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				status := &DriverVerificationStatus{
					DriverID:                   driverID,
					BackgroundCheckStatus:      BGCheckStatusFailed,
					SelfieVerificationRequired: true,
					IsFullyVerified:            false,
					VerificationMessage:        "Background check not passed",
				}
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(status, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, status *DriverVerificationStatus) {
				assert.False(t, status.IsFullyVerified)
				assert.Equal(t, BGCheckStatusFailed, status.BackgroundCheckStatus)
			},
		},
		{
			name:     "error - repository error",
			driverID: driverID,
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverVerificationStatus", mock.Anything, driverID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			status, err := svc.GetDriverVerificationStatus(context.Background(), tt.driverID)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, status)
			} else {
				require.NoError(t, err)
				require.NotNil(t, status)
				if tt.validate != nil {
					tt.validate(t, status)
				}
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: Status Mapping Functions (via ProcessWebhook)
// ========================================

func TestMapCheckrStatus(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "checkr-ext-123"

	tests := []struct {
		name           string
		status         string
		expectedStatus BackgroundCheckStatus
		expectReasons  bool
	}{
		{
			name:           "clear maps to passed",
			status:         "clear",
			expectedStatus: BGCheckStatusPassed,
			expectReasons:  false,
		},
		{
			name:           "consider maps to failed with reason",
			status:         "consider",
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
		{
			name:           "pending maps to in_progress",
			status:         "pending",
			expectedStatus: BGCheckStatusInProgress,
			expectReasons:  false,
		},
		{
			name:           "unknown maps to failed with reason",
			status:         "some_unknown_status",
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			check := &BackgroundCheck{
				ID:       checkID,
				DriverID: driverID,
				Provider: ProviderCheckr,
				Status:   BGCheckStatusInProgress,
			}
			m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)

			if tt.expectedStatus == BGCheckStatusPassed {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			} else if tt.expectedStatus == BGCheckStatusInProgress {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
			} else {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
				if tt.expectReasons {
					m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("[]string")).Return(nil)
				}
			}

			svc := newTestService(m)
			payload := &WebhookPayload{
				Provider:   ProviderCheckr,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     tt.status,
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			}

			err := svc.ProcessWebhook(context.Background(), payload)
			require.NoError(t, err)

			m.AssertExpectations(t)
		})
	}
}

func TestMapSterlingStatus(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "sterling-ext-123"

	tests := []struct {
		name           string
		status         string
		expectedStatus BackgroundCheckStatus
		expectReasons  bool
	}{
		{
			name:           "Completed maps to passed",
			status:         "Completed",
			expectedStatus: BGCheckStatusPassed,
			expectReasons:  false,
		},
		{
			name:           "Pass maps to passed",
			status:         "Pass",
			expectedStatus: BGCheckStatusPassed,
			expectReasons:  false,
		},
		{
			name:           "Fail maps to failed",
			status:         "Fail",
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
		{
			name:           "Alert maps to failed",
			status:         "Alert",
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
		{
			name:           "InProgress maps to in_progress",
			status:         "InProgress",
			expectedStatus: BGCheckStatusInProgress,
			expectReasons:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			check := &BackgroundCheck{
				ID:       checkID,
				DriverID: driverID,
				Provider: ProviderSterling,
				Status:   BGCheckStatusInProgress,
			}
			m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderSterling, externalID).Return(check, nil)

			if tt.expectedStatus == BGCheckStatusPassed {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			} else if tt.expectedStatus == BGCheckStatusInProgress {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
			} else {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
				if tt.expectReasons {
					m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("[]string")).Return(nil)
				}
			}

			svc := newTestService(m)
			payload := &WebhookPayload{
				Provider:   ProviderSterling,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     tt.status,
				Data:       map[string]interface{}{},
				Timestamp:  time.Now(),
			}

			err := svc.ProcessWebhook(context.Background(), payload)
			require.NoError(t, err)

			m.AssertExpectations(t)
		})
	}
}

func TestMapOnfidoStatus(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "onfido-ext-123"

	tests := []struct {
		name           string
		status         string
		data           map[string]interface{}
		expectedStatus BackgroundCheckStatus
		expectReasons  bool
	}{
		{
			name:           "complete with clear result maps to passed",
			status:         "complete",
			data:           map[string]interface{}{"result": "clear"},
			expectedStatus: BGCheckStatusPassed,
			expectReasons:  false,
		},
		{
			name:           "complete with consider result maps to failed",
			status:         "complete",
			data:           map[string]interface{}{"result": "consider"},
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
		{
			name:           "complete without result maps to failed",
			status:         "complete",
			data:           map[string]interface{}{},
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
		{
			name:           "in_progress maps to in_progress",
			status:         "in_progress",
			data:           map[string]interface{}{},
			expectedStatus: BGCheckStatusInProgress,
			expectReasons:  false,
		},
		{
			name:           "awaiting_data maps to in_progress",
			status:         "awaiting_data",
			data:           map[string]interface{}{},
			expectedStatus: BGCheckStatusInProgress,
			expectReasons:  false,
		},
		{
			name:           "unknown status maps to failed",
			status:         "cancelled",
			data:           map[string]interface{}{},
			expectedStatus: BGCheckStatusFailed,
			expectReasons:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			check := &BackgroundCheck{
				ID:       checkID,
				DriverID: driverID,
				Provider: ProviderOnfido,
				Status:   BGCheckStatusInProgress,
			}
			m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderOnfido, externalID).Return(check, nil)

			if tt.expectedStatus == BGCheckStatusPassed {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
				m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)
			} else if tt.expectedStatus == BGCheckStatusInProgress {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
			} else {
				m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), (*time.Time)(nil)).Return(nil)
				if tt.expectReasons {
					m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, tt.expectedStatus, (*string)(nil), mock.AnythingOfType("[]string")).Return(nil)
				}
			}

			svc := newTestService(m)
			payload := &WebhookPayload{
				Provider:   ProviderOnfido,
				EventType:  "check.completed",
				ExternalID: externalID,
				Status:     tt.status,
				Data:       tt.data,
				Timestamp:  time.Now(),
			}

			err := svc.ProcessWebhook(context.Background(), payload)
			require.NoError(t, err)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: Edge Cases and Error Handling
// ========================================

func TestInitiateBackgroundCheck_ProviderIntegration(t *testing.T) {
	driverID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	tests := []struct {
		name     string
		provider BackgroundCheckProvider
		cfg      *config.Config
		wantErr  bool
	}{
		{
			name:     "checkr without API key fails",
			provider: ProviderCheckr,
			cfg:      &config.Config{},
			wantErr:  true,
		},
		{
			name:     "onfido without API key fails",
			provider: ProviderOnfido,
			cfg:      &config.Config{},
			wantErr:  true,
		},
		{
			name:     "sterling returns placeholder ID",
			provider: ProviderSterling,
			cfg:      &config.Config{},
			wantErr:  false, // Sterling returns placeholder in current implementation
		},
		{
			name:     "mock provider always succeeds",
			provider: ProviderMock,
			cfg:      &config.Config{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			m.On("GetLatestBackgroundCheck", mock.Anything, driverID).Return(nil, errors.New("not found"))
			m.On("CreateBackgroundCheck", mock.Anything, mock.AnythingOfType("*verification.BackgroundCheck")).Return(nil)

			if tt.wantErr {
				m.On("UpdateBackgroundCheckStatus", mock.Anything, mock.AnythingOfType("uuid.UUID"), BGCheckStatusFailed, mock.AnythingOfType("*string"), ([]string)(nil)).Return(nil)
			} else {
				m.On("UpdateBackgroundCheckStarted", mock.Anything, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("string")).Return(nil)
			}

			svc := newTestServiceWithConfig(m, tt.cfg)
			req := &InitiateBackgroundCheckRequest{
				DriverID:      driverID,
				Provider:      tt.provider,
				FirstName:     "John",
				LastName:      "Doe",
				DateOfBirth:   "1990-01-15",
				Email:         "john.doe@example.com",
				Phone:         "+1234567890",
				StreetAddress: "123 Main St",
				City:          "San Francisco",
				State:         "CA",
				ZipCode:       "94102",
				LicenseNumber: "D1234567",
				LicenseState:  "CA",
			}

			resp, err := svc.InitiateBackgroundCheck(context.Background(), req)

			if tt.wantErr {
				require.Error(t, err)
				assert.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				assert.Equal(t, BGCheckStatusInProgress, resp.Status)
			}

			m.AssertExpectations(t)
		})
	}
}

func TestProcessWebhook_WithReportURL(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-123"
	reportURL := "https://provider.com/reports/detailed-report-123"

	m := new(mockRepo)
	check := &BackgroundCheck{
		ID:       checkID,
		DriverID: driverID,
		Provider: ProviderCheckr,
		Status:   BGCheckStatusInProgress,
	}
	m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
	m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, mock.MatchedBy(func(url *string) bool {
		return url != nil && *url == reportURL
	}), mock.AnythingOfType("*time.Time")).Return(nil)
	m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)

	svc := newTestService(m)
	payload := &WebhookPayload{
		Provider:   ProviderCheckr,
		EventType:  "check.completed",
		ExternalID: externalID,
		Status:     "clear",
		Data: map[string]interface{}{
			"report_url": reportURL,
		},
		Timestamp: time.Now(),
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestProcessWebhook_AutoApprovesDriverOnPass(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-123"

	m := new(mockRepo)
	check := &BackgroundCheck{
		ID:       checkID,
		DriverID: driverID,
		Provider: ProviderCheckr,
		Status:   BGCheckStatusInProgress,
	}
	m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
	m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusPassed, (*string)(nil), mock.AnythingOfType("*time.Time")).Return(nil)
	// Verify UpdateDriverApproval is called with approved=true
	m.On("UpdateDriverApproval", mock.Anything, driverID, true, (*uuid.UUID)(nil), (*string)(nil)).Return(nil)

	svc := newTestService(m)
	payload := &WebhookPayload{
		Provider:   ProviderCheckr,
		EventType:  "check.completed",
		ExternalID: externalID,
		Status:     "clear",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	m.AssertExpectations(t)
}

func TestProcessWebhook_DoesNotApproveOnFailure(t *testing.T) {
	checkID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	driverID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	externalID := "ext-123"

	m := new(mockRepo)
	check := &BackgroundCheck{
		ID:       checkID,
		DriverID: driverID,
		Provider: ProviderCheckr,
		Status:   BGCheckStatusInProgress,
	}
	m.On("GetBackgroundCheckByExternalID", mock.Anything, ProviderCheckr, externalID).Return(check, nil)
	m.On("UpdateBackgroundCheckCompleted", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), (*time.Time)(nil)).Return(nil)
	m.On("UpdateBackgroundCheckStatus", mock.Anything, checkID, BGCheckStatusFailed, (*string)(nil), mock.AnythingOfType("[]string")).Return(nil)
	// NOTE: UpdateDriverApproval should NOT be called

	svc := newTestService(m)
	payload := &WebhookPayload{
		Provider:   ProviderCheckr,
		EventType:  "check.completed",
		ExternalID: externalID,
		Status:     "consider",
		Data:       map[string]interface{}{},
		Timestamp:  time.Now(),
	}

	err := svc.ProcessWebhook(context.Background(), payload)
	require.NoError(t, err)

	m.AssertExpectations(t)
	// Verify UpdateDriverApproval was never called
	m.AssertNotCalled(t, "UpdateDriverApproval", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
