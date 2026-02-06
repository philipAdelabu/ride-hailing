package preferences

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
// INTERNAL MOCK (implements RepositoryInterface within this package)
// ========================================

type mockRepo struct {
	mock.Mock
}

func (m *mockRepo) GetRiderPreferences(ctx context.Context, userID uuid.UUID) (*RiderPreferences, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RiderPreferences), args.Error(1)
}

func (m *mockRepo) UpsertRiderPreferences(ctx context.Context, p *RiderPreferences) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *mockRepo) SetRideOverride(ctx context.Context, o *RidePreferenceOverride) error {
	args := m.Called(ctx, o)
	return args.Error(0)
}

func (m *mockRepo) GetRideOverride(ctx context.Context, rideID uuid.UUID) (*RidePreferenceOverride, error) {
	args := m.Called(ctx, rideID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*RidePreferenceOverride), args.Error(1)
}

func (m *mockRepo) GetDriverCapabilities(ctx context.Context, driverID uuid.UUID) (*DriverCapabilities, error) {
	args := m.Called(ctx, driverID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*DriverCapabilities), args.Error(1)
}

func (m *mockRepo) UpsertDriverCapabilities(ctx context.Context, dc *DriverCapabilities) error {
	args := m.Called(ctx, dc)
	return args.Error(0)
}

// ========================================
// TEST HELPERS
// ========================================

func newTestService(repo RepositoryInterface) *Service {
	return NewService(repo)
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}

func tempPtr(t TemperaturePreference) *TemperaturePreference {
	return &t
}

func musicPtr(m MusicPreference) *MusicPreference {
	return &m
}

func convPtr(c ConversationPreference) *ConversationPreference {
	return &c
}

func routePtr(r RoutePreference) *RoutePreference {
	return &r
}

// ========================================
// TESTS: GetPreferences
// ========================================

func TestGetPreferences(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, prefs *RiderPreferences)
	}{
		{
			name: "existing preferences found",
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:              uuid.New(),
					UserID:          userID,
					Temperature:     TempCool,
					Music:           MusicQuiet,
					Conversation:    ConversationQuiet,
					Route:           RouteFastest,
					PetFriendly:     true,
					WheelchairAccess: true,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.Equal(t, userID, prefs.UserID)
				assert.Equal(t, TempCool, prefs.Temperature)
				assert.Equal(t, MusicQuiet, prefs.Music)
				assert.Equal(t, ConversationQuiet, prefs.Conversation)
				assert.Equal(t, RouteFastest, prefs.Route)
				assert.True(t, prefs.PetFriendly)
				assert.True(t, prefs.WheelchairAccess)
			},
		},
		{
			name: "new user - creates default preferences",
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.Equal(t, userID, prefs.UserID)
				assert.Equal(t, TempNoPreference, prefs.Temperature)
				assert.Equal(t, MusicNoPreference, prefs.Music)
				assert.Equal(t, ConversationNoPreference, prefs.Conversation)
				assert.Equal(t, RouteNoPreference, prefs.Route)
				assert.False(t, prefs.ChildSeat)
				assert.False(t, prefs.PetFriendly)
				assert.False(t, prefs.WheelchairAccess)
			},
		},
		{
			name: "repository error - returns error",
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				// Not called when error
			},
		},
		{
			name: "upsert error on default creation - returns error",
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(errors.New("upsert failed"))
			},
			wantErr: true,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				// Not called when error
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			prefs, err := svc.GetPreferences(context.Background(), userID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, prefs)
			} else {
				require.NoError(t, err)
				require.NotNil(t, prefs)
				tt.validate(t, prefs)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdatePreferences
// ========================================

func TestUpdatePreferences(t *testing.T) {
	userID := uuid.New()

	tests := []struct {
		name       string
		req        *UpdatePreferencesRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, prefs *RiderPreferences)
	}{
		{
			name: "update single field - temperature",
			req: &UpdatePreferencesRequest{
				Temperature: tempPtr(TempWarm),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
					return p.Temperature == TempWarm
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.Equal(t, TempWarm, prefs.Temperature)
			},
		},
		{
			name: "update multiple fields",
			req: &UpdatePreferencesRequest{
				Temperature:     tempPtr(TempCool),
				Music:          musicPtr(MusicLow),
				Conversation:   convPtr(ConversationFriendly),
				Route:          routePtr(RouteScenic),
				PetFriendly:    boolPtr(true),
				WheelchairAccess: boolPtr(true),
				ChildSeat:      boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.Equal(t, TempCool, prefs.Temperature)
				assert.Equal(t, MusicLow, prefs.Music)
				assert.Equal(t, ConversationFriendly, prefs.Conversation)
				assert.Equal(t, RouteScenic, prefs.Route)
				assert.True(t, prefs.PetFriendly)
				assert.True(t, prefs.WheelchairAccess)
				assert.True(t, prefs.ChildSeat)
			},
		},
		{
			name: "update optional fields - max passengers and language",
			req: &UpdatePreferencesRequest{
				MaxPassengers:     intPtr(4),
				PreferredLanguage: strPtr("en"),
				SpecialNeeds:      strPtr("hearing impaired"),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				require.NotNil(t, prefs.MaxPassengers)
				assert.Equal(t, 4, *prefs.MaxPassengers)
				require.NotNil(t, prefs.PreferredLanguage)
				assert.Equal(t, "en", *prefs.PreferredLanguage)
				require.NotNil(t, prefs.SpecialNeeds)
				assert.Equal(t, "hearing impaired", *prefs.SpecialNeeds)
			},
		},
		{
			name: "update for new user - creates defaults first",
			req: &UpdatePreferencesRequest{
				PetFriendly: boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				// First call: no existing preferences
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, pgx.ErrNoRows).Once()
				// Create defaults
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(nil).Once()
				// Update with new value
				m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
					return p.PetFriendly == true
				})).Return(nil).Once()
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.True(t, prefs.PetFriendly)
			},
		},
		{
			name: "get preferences error",
			req: &UpdatePreferencesRequest{
				Temperature: tempPtr(TempWarm),
			},
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				// Not called
			},
		},
		{
			name: "upsert error",
			req: &UpdatePreferencesRequest{
				Temperature: tempPtr(TempWarm),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.AnythingOfType("*preferences.RiderPreferences")).Return(errors.New("upsert failed"))
			},
			wantErr: true,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				// Not called
			},
		},
		{
			name: "update female driver preference",
			req: &UpdatePreferencesRequest{
				PreferFemaleDriver: boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
					return p.PreferFemaleDriver == true
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.True(t, prefs.PreferFemaleDriver)
			},
		},
		{
			name: "update luggage assistance",
			req: &UpdatePreferencesRequest{
				LuggageAssistance: boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				existingPrefs := &RiderPreferences{
					ID:           uuid.New(),
					UserID:       userID,
					Temperature:  TempNoPreference,
					Music:        MusicNoPreference,
					Conversation: ConversationNoPreference,
					Route:        RouteNoPreference,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
				m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
					return p.LuggageAssistance == true
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, prefs *RiderPreferences) {
				assert.True(t, prefs.LuggageAssistance)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			prefs, err := svc.UpdatePreferences(context.Background(), userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, prefs)
			} else {
				require.NoError(t, err)
				require.NotNil(t, prefs)
				tt.validate(t, prefs)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetDriverCapabilities
// ========================================

func TestGetDriverCapabilities(t *testing.T) {
	driverID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, caps *DriverCapabilities)
	}{
		{
			name: "existing capabilities found",
			setupMocks: func(m *mockRepo) {
				langJSON := `["en","es"]`
				existingCaps := &DriverCapabilities{
					ID:               uuid.New(),
					DriverID:         driverID,
					HasChildSeat:     true,
					PetFriendly:      true,
					WheelchairAccess: false,
					LuggageCapacity:  3,
					MaxPassengers:    4,
					LanguagesJSON:    &langJSON,
				}
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(existingCaps, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				assert.Equal(t, driverID, caps.DriverID)
				assert.True(t, caps.HasChildSeat)
				assert.True(t, caps.PetFriendly)
				assert.False(t, caps.WheelchairAccess)
				assert.Equal(t, 3, caps.LuggageCapacity)
				assert.Equal(t, 4, caps.MaxPassengers)
				assert.Equal(t, []string{"en", "es"}, caps.Languages)
			},
		},
		{
			name: "no capabilities - returns empty with driver ID",
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, nil)
			},
			wantErr: false,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				assert.Equal(t, driverID, caps.DriverID)
				assert.False(t, caps.HasChildSeat)
				assert.False(t, caps.PetFriendly)
				assert.False(t, caps.WheelchairAccess)
			},
		},
		{
			name: "repository error",
			setupMocks: func(m *mockRepo) {
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("database error"))
			},
			wantErr: true,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				// Not called
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			caps, err := svc.GetDriverCapabilities(context.Background(), driverID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, caps)
			} else {
				require.NoError(t, err)
				require.NotNil(t, caps)
				tt.validate(t, caps)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: UpdateDriverCapabilities
// ========================================

func TestUpdateDriverCapabilities(t *testing.T) {
	driverID := uuid.New()

	tests := []struct {
		name       string
		req        *UpdateDriverCapabilitiesRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, caps *DriverCapabilities)
	}{
		{
			name: "update existing capabilities",
			req: &UpdateDriverCapabilitiesRequest{
				HasChildSeat:     boolPtr(true),
				PetFriendly:      boolPtr(true),
				WheelchairAccess: boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				existingCaps := &DriverCapabilities{
					ID:        uuid.New(),
					DriverID:  driverID,
					CreatedAt: time.Now().Add(-time.Hour),
				}
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(existingCaps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.MatchedBy(func(c *DriverCapabilities) bool {
					return c.HasChildSeat && c.PetFriendly && c.WheelchairAccess
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				assert.True(t, caps.HasChildSeat)
				assert.True(t, caps.PetFriendly)
				assert.True(t, caps.WheelchairAccess)
			},
		},
		{
			name: "create new capabilities",
			req: &UpdateDriverCapabilitiesRequest{
				HasChildSeat:    boolPtr(true),
				LuggageCapacity: intPtr(5),
				MaxPassengers:   intPtr(6),
			},
			setupMocks: func(m *mockRepo) {
				// Service checks: err != nil && caps == nil to create new
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("not found"))
				m.On("UpsertDriverCapabilities", mock.Anything, mock.MatchedBy(func(c *DriverCapabilities) bool {
					return c.HasChildSeat && c.LuggageCapacity == 5 && c.MaxPassengers == 6
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				assert.True(t, caps.HasChildSeat)
				assert.Equal(t, 5, caps.LuggageCapacity)
				assert.Equal(t, 6, caps.MaxPassengers)
			},
		},
		{
			name: "update languages",
			req: &UpdateDriverCapabilitiesRequest{
				Languages: []string{"en", "es", "fr"},
			},
			setupMocks: func(m *mockRepo) {
				existingCaps := &DriverCapabilities{
					ID:        uuid.New(),
					DriverID:  driverID,
					CreatedAt: time.Now().Add(-time.Hour),
				}
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(existingCaps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.MatchedBy(func(c *DriverCapabilities) bool {
					return len(c.Languages) == 3 && c.Languages[0] == "en"
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				assert.Equal(t, []string{"en", "es", "fr"}, caps.Languages)
				require.NotNil(t, caps.LanguagesJSON)
			},
		},
		{
			name: "upsert error",
			req: &UpdateDriverCapabilitiesRequest{
				HasChildSeat: boolPtr(true),
			},
			setupMocks: func(m *mockRepo) {
				existingCaps := &DriverCapabilities{
					ID:       uuid.New(),
					DriverID: driverID,
				}
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(existingCaps, nil)
				m.On("UpsertDriverCapabilities", mock.Anything, mock.AnythingOfType("*preferences.DriverCapabilities")).Return(errors.New("upsert failed"))
			},
			wantErr: true,
			validate: func(t *testing.T, caps *DriverCapabilities) {
				// Not called
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			caps, err := svc.UpdateDriverCapabilities(context.Background(), driverID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, caps)
			} else {
				require.NoError(t, err)
				require.NotNil(t, caps)
				tt.validate(t, caps)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: SetRidePreferences (per-ride overrides)
// ========================================

func TestSetRidePreferences(t *testing.T) {
	userID := uuid.New()
	rideID := uuid.New()

	tests := []struct {
		name       string
		req        *SetRidePreferencesRequest
		setupMocks func(m *mockRepo)
		wantErr    bool
		validate   func(t *testing.T, override *RidePreferenceOverride)
	}{
		{
			name: "set temperature override",
			req: &SetRidePreferencesRequest{
				RideID:      rideID,
				Temperature: tempPtr(TempCool),
			},
			setupMocks: func(m *mockRepo) {
				m.On("SetRideOverride", mock.Anything, mock.MatchedBy(func(o *RidePreferenceOverride) bool {
					return o.RideID == rideID && o.UserID == userID && *o.Temperature == TempCool
				})).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, override *RidePreferenceOverride) {
				assert.Equal(t, rideID, override.RideID)
				assert.Equal(t, userID, override.UserID)
				require.NotNil(t, override.Temperature)
				assert.Equal(t, TempCool, *override.Temperature)
			},
		},
		{
			name: "set multiple overrides",
			req: &SetRidePreferencesRequest{
				RideID:       rideID,
				Temperature:  tempPtr(TempWarm),
				Music:        musicPtr(MusicQuiet),
				Conversation: convPtr(ConversationQuiet),
				Route:        routePtr(RouteFastest),
				ChildSeat:    boolPtr(true),
				PetFriendly:  boolPtr(true),
				Notes:        strPtr("I have a small dog"),
			},
			setupMocks: func(m *mockRepo) {
				m.On("SetRideOverride", mock.Anything, mock.AnythingOfType("*preferences.RidePreferenceOverride")).Return(nil)
			},
			wantErr: false,
			validate: func(t *testing.T, override *RidePreferenceOverride) {
				assert.Equal(t, rideID, override.RideID)
				require.NotNil(t, override.Temperature)
				assert.Equal(t, TempWarm, *override.Temperature)
				require.NotNil(t, override.Music)
				assert.Equal(t, MusicQuiet, *override.Music)
				require.NotNil(t, override.Conversation)
				assert.Equal(t, ConversationQuiet, *override.Conversation)
				require.NotNil(t, override.Route)
				assert.Equal(t, RouteFastest, *override.Route)
				require.NotNil(t, override.ChildSeat)
				assert.True(t, *override.ChildSeat)
				require.NotNil(t, override.PetFriendly)
				assert.True(t, *override.PetFriendly)
				require.NotNil(t, override.Notes)
				assert.Equal(t, "I have a small dog", *override.Notes)
			},
		},
		{
			name: "repository error",
			req: &SetRidePreferencesRequest{
				RideID:      rideID,
				Temperature: tempPtr(TempCool),
			},
			setupMocks: func(m *mockRepo) {
				m.On("SetRideOverride", mock.Anything, mock.AnythingOfType("*preferences.RidePreferenceOverride")).Return(errors.New("database error"))
			},
			wantErr: true,
			validate: func(t *testing.T, override *RidePreferenceOverride) {
				// Not called
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			override, err := svc.SetRidePreferences(context.Background(), userID, tt.req)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, override)
			} else {
				require.NoError(t, err)
				require.NotNil(t, override)
				tt.validate(t, override)
			}

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetRidePreferences
// ========================================

func TestGetRidePreferences(t *testing.T) {
	userID := uuid.New()
	rideID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		validate   func(t *testing.T, summary *PreferenceSummary)
	}{
		{
			name: "returns preferences and override",
			setupMocks: func(m *mockRepo) {
				prefs := &RiderPreferences{
					ID:          uuid.New(),
					UserID:      userID,
					Temperature: TempNoPreference,
					Music:       MusicLow,
					PetFriendly: true,
				}
				override := &RidePreferenceOverride{
					ID:          uuid.New(),
					RideID:      rideID,
					UserID:      userID,
					Temperature: tempPtr(TempCool),
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("GetRideOverride", mock.Anything, rideID).Return(override, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				require.NotNil(t, summary.Preferences)
				assert.Equal(t, MusicLow, summary.Preferences.Music)
				assert.True(t, summary.Preferences.PetFriendly)
				require.NotNil(t, summary.Override)
				assert.Equal(t, TempCool, *summary.Override.Temperature)
			},
		},
		{
			name: "no override - returns preferences only",
			setupMocks: func(m *mockRepo) {
				prefs := &RiderPreferences{
					ID:          uuid.New(),
					UserID:      userID,
					Temperature: TempWarm,
				}
				m.On("GetRiderPreferences", mock.Anything, userID).Return(prefs, nil)
				m.On("GetRideOverride", mock.Anything, rideID).Return(nil, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				require.NotNil(t, summary.Preferences)
				assert.Equal(t, TempWarm, summary.Preferences.Temperature)
				assert.Nil(t, summary.Override)
			},
		},
		{
			name: "handles errors gracefully",
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, errors.New("db error"))
				m.On("GetRideOverride", mock.Anything, rideID).Return(nil, errors.New("db error"))
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				assert.Nil(t, summary.Preferences)
				assert.Nil(t, summary.Override)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.GetRidePreferences(context.Background(), rideID, userID)

			// GetRidePreferences always returns non-nil summary
			require.NoError(t, err)
			require.NotNil(t, summary)
			tt.validate(t, summary)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: GetPreferencesForDriver
// ========================================

func TestGetPreferencesForDriver(t *testing.T) {
	rideID := uuid.New()
	driverID := uuid.New()
	riderID := uuid.New()

	tests := []struct {
		name       string
		setupMocks func(m *mockRepo)
		validate   func(t *testing.T, summary *PreferenceSummary)
	}{
		{
			name: "full match report with all capabilities",
			setupMocks: func(m *mockRepo) {
				prefs := &RiderPreferences{
					ID:               uuid.New(),
					UserID:           riderID,
					ChildSeat:        true,
					PetFriendly:      true,
					WheelchairAccess: true,
				}
				caps := &DriverCapabilities{
					ID:               uuid.New(),
					DriverID:         driverID,
					HasChildSeat:     true,
					PetFriendly:      true,
					WheelchairAccess: true,
				}
				m.On("GetRiderPreferences", mock.Anything, riderID).Return(prefs, nil)
				m.On("GetRideOverride", mock.Anything, rideID).Return(nil, nil)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				require.NotNil(t, summary.Preferences)
				require.NotNil(t, summary.Capabilities)
				require.NotNil(t, summary.Matches)
				assert.Len(t, summary.Matches, 3) // child_seat, pet_friendly, wheelchair_access
				for _, match := range summary.Matches {
					assert.True(t, match.Match, "expected match for %s", match.Preference)
				}
			},
		},
		{
			name: "partial match - driver lacks some capabilities",
			setupMocks: func(m *mockRepo) {
				prefs := &RiderPreferences{
					ID:               uuid.New(),
					UserID:           riderID,
					ChildSeat:        true,
					PetFriendly:      true,
					WheelchairAccess: false,
				}
				caps := &DriverCapabilities{
					ID:               uuid.New(),
					DriverID:         driverID,
					HasChildSeat:     false, // Doesn't have child seat
					PetFriendly:      true,
					WheelchairAccess: false,
				}
				m.On("GetRiderPreferences", mock.Anything, riderID).Return(prefs, nil)
				m.On("GetRideOverride", mock.Anything, rideID).Return(nil, nil)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				require.NotNil(t, summary.Matches)
				assert.Len(t, summary.Matches, 2) // child_seat and pet_friendly requested
				var childSeatMatch, petMatch *PreferenceMatch
				for i := range summary.Matches {
					if summary.Matches[i].Preference == "child_seat" {
						childSeatMatch = &summary.Matches[i]
					}
					if summary.Matches[i].Preference == "pet_friendly" {
						petMatch = &summary.Matches[i]
					}
				}
				require.NotNil(t, childSeatMatch)
				assert.False(t, childSeatMatch.Match)
				assert.False(t, childSeatMatch.Available)
				require.NotNil(t, petMatch)
				assert.True(t, petMatch.Match)
			},
		},
		{
			name: "override takes precedence over default",
			setupMocks: func(m *mockRepo) {
				prefs := &RiderPreferences{
					ID:          uuid.New(),
					UserID:      riderID,
					ChildSeat:   false, // Default: no child seat
					PetFriendly: false,
				}
				override := &RidePreferenceOverride{
					ID:        uuid.New(),
					RideID:    rideID,
					UserID:    riderID,
					ChildSeat: boolPtr(true), // Override: needs child seat
				}
				caps := &DriverCapabilities{
					ID:           uuid.New(),
					DriverID:     driverID,
					HasChildSeat: true,
					PetFriendly:  true,
				}
				m.On("GetRiderPreferences", mock.Anything, riderID).Return(prefs, nil)
				m.On("GetRideOverride", mock.Anything, rideID).Return(override, nil)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(caps, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				require.NotNil(t, summary.Override)
				require.NotNil(t, summary.Matches)
				// Should have child_seat match from override
				var childSeatMatch *PreferenceMatch
				for i := range summary.Matches {
					if summary.Matches[i].Preference == "child_seat" {
						childSeatMatch = &summary.Matches[i]
						break
					}
				}
				require.NotNil(t, childSeatMatch)
				assert.True(t, childSeatMatch.Requested)
				assert.True(t, childSeatMatch.Available)
				assert.True(t, childSeatMatch.Match)
			},
		},
		{
			name: "no preferences or capabilities",
			setupMocks: func(m *mockRepo) {
				m.On("GetRiderPreferences", mock.Anything, riderID).Return(nil, errors.New("not found"))
				m.On("GetRideOverride", mock.Anything, rideID).Return(nil, nil)
				m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, nil)
			},
			validate: func(t *testing.T, summary *PreferenceSummary) {
				assert.Nil(t, summary.Preferences)
				assert.Nil(t, summary.Capabilities)
				assert.Nil(t, summary.Matches)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := new(mockRepo)
			tt.setupMocks(m)
			svc := newTestService(m)

			summary, err := svc.GetPreferencesForDriver(context.Background(), rideID, driverID, riderID)

			require.NoError(t, err)
			require.NotNil(t, summary)
			tt.validate(t, summary)

			m.AssertExpectations(t)
		})
	}
}

// ========================================
// TESTS: buildMatchReport (matching algorithm)
// ========================================

func TestBuildMatchReport(t *testing.T) {
	tests := []struct {
		name     string
		prefs    *RiderPreferences
		override *RidePreferenceOverride
		caps     *DriverCapabilities
		validate func(t *testing.T, matches []PreferenceMatch)
	}{
		{
			name: "all preferences match",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			override: nil,
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 3)
				for _, m := range matches {
					assert.True(t, m.Match, "expected match for %s", m.Preference)
					assert.True(t, m.Requested)
					assert.True(t, m.Available)
				}
			},
		},
		{
			name: "no matches - all capabilities missing",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			override: nil,
			caps: &DriverCapabilities{
				HasChildSeat:     false,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 3)
				for _, m := range matches {
					assert.False(t, m.Match, "expected no match for %s", m.Preference)
					assert.True(t, m.Requested)
					assert.False(t, m.Available)
				}
			},
		},
		{
			name: "no preferences requested - empty matches",
			prefs: &RiderPreferences{
				ChildSeat:        false,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			override: nil,
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 0)
			},
		},
		{
			name: "override enables child seat",
			prefs: &RiderPreferences{
				ChildSeat:   false,
				PetFriendly: false,
			},
			override: &RidePreferenceOverride{
				ChildSeat: boolPtr(true),
			},
			caps: &DriverCapabilities{
				HasChildSeat: true,
				PetFriendly:  false,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 1)
				assert.Equal(t, "child_seat", matches[0].Preference)
				assert.True(t, matches[0].Match)
			},
		},
		{
			name: "override disables preference from default",
			prefs: &RiderPreferences{
				ChildSeat:   true, // Default wants child seat
				PetFriendly: true,
			},
			override: &RidePreferenceOverride{
				ChildSeat: boolPtr(false), // Override: don't need child seat
			},
			caps: &DriverCapabilities{
				HasChildSeat: false, // Driver has no child seat
				PetFriendly:  true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				// Should only have pet_friendly since child_seat is overridden to false
				assert.Len(t, matches, 1)
				assert.Equal(t, "pet_friendly", matches[0].Preference)
				assert.True(t, matches[0].Match)
			},
		},
		{
			name: "pet friendly override",
			prefs: &RiderPreferences{
				PetFriendly: false,
			},
			override: &RidePreferenceOverride{
				PetFriendly: boolPtr(true),
			},
			caps: &DriverCapabilities{
				PetFriendly: true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 1)
				assert.Equal(t, "pet_friendly", matches[0].Preference)
				assert.True(t, matches[0].Match)
			},
		},
		{
			name: "wheelchair access - not in override (uses default)",
			prefs: &RiderPreferences{
				WheelchairAccess: true,
			},
			override: &RidePreferenceOverride{
				// Override doesn't specify wheelchair
			},
			caps: &DriverCapabilities{
				WheelchairAccess: true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 1)
				assert.Equal(t, "wheelchair_access", matches[0].Preference)
				assert.True(t, matches[0].Match)
			},
		},
		{
			name: "mixed results",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			override: nil,
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      false, // Mismatch
				WheelchairAccess: true,
			},
			validate: func(t *testing.T, matches []PreferenceMatch) {
				assert.Len(t, matches, 3)
				matchMap := make(map[string]PreferenceMatch)
				for _, m := range matches {
					matchMap[m.Preference] = m
				}
				assert.True(t, matchMap["child_seat"].Match)
				assert.False(t, matchMap["pet_friendly"].Match)
				assert.True(t, matchMap["wheelchair_access"].Match)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := buildMatchReport(tt.prefs, tt.override, tt.caps)
			tt.validate(t, matches)
		})
	}
}

// ========================================
// TESTS: Matching Score Calculation
// ========================================

func TestMatchingScoreCalculation(t *testing.T) {
	tests := []struct {
		name          string
		prefs         *RiderPreferences
		caps          *DriverCapabilities
		expectedScore float64
	}{
		{
			name: "100% match - all requested preferences met",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			expectedScore: 100.0,
		},
		{
			name: "66% match - 2 of 3 preferences met",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      true,
				WheelchairAccess: false,
			},
			expectedScore: 66.67,
		},
		{
			name: "33% match - 1 of 3 preferences met",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			expectedScore: 33.33,
		},
		{
			name: "0% match - no preferences met",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: true,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     false,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			expectedScore: 0.0,
		},
		{
			name: "100% - no preferences requested",
			prefs: &RiderPreferences{
				ChildSeat:        false,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     false,
				PetFriendly:      false,
				WheelchairAccess: false,
			},
			expectedScore: 100.0, // No preferences = perfect match
		},
		{
			name: "50% match - 1 of 2 preferences met",
			prefs: &RiderPreferences{
				ChildSeat:        true,
				PetFriendly:      true,
				WheelchairAccess: false,
			},
			caps: &DriverCapabilities{
				HasChildSeat:     true,
				PetFriendly:      false,
				WheelchairAccess: true,
			},
			expectedScore: 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := buildMatchReport(tt.prefs, nil, tt.caps)

			// Calculate score
			if len(matches) == 0 {
				assert.Equal(t, tt.expectedScore, 100.0)
				return
			}

			matchCount := 0
			for _, m := range matches {
				if m.Match {
					matchCount++
				}
			}

			score := (float64(matchCount) / float64(len(matches))) * 100
			assert.InDelta(t, tt.expectedScore, score, 0.01)
		})
	}
}

// ========================================
// TESTS: Default Preferences for New Users
// ========================================

func TestCreateDefaults(t *testing.T) {
	userID := uuid.New()

	m := new(mockRepo)
	m.On("GetRiderPreferences", mock.Anything, userID).Return(nil, pgx.ErrNoRows)
	m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
		return p.UserID == userID &&
			p.Temperature == TempNoPreference &&
			p.Music == MusicNoPreference &&
			p.Conversation == ConversationNoPreference &&
			p.Route == RouteNoPreference &&
			!p.ChildSeat &&
			!p.PetFriendly &&
			!p.WheelchairAccess &&
			!p.PreferFemaleDriver &&
			!p.LuggageAssistance
	})).Return(nil)

	svc := newTestService(m)
	prefs, err := svc.GetPreferences(context.Background(), userID)

	require.NoError(t, err)
	require.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.Equal(t, TempNoPreference, prefs.Temperature)
	assert.Equal(t, MusicNoPreference, prefs.Music)
	assert.Equal(t, ConversationNoPreference, prefs.Conversation)
	assert.Equal(t, RouteNoPreference, prefs.Route)
	assert.False(t, prefs.ChildSeat)
	assert.False(t, prefs.PetFriendly)
	assert.False(t, prefs.WheelchairAccess)
	assert.NotZero(t, prefs.CreatedAt)
	assert.NotZero(t, prefs.UpdatedAt)

	m.AssertExpectations(t)
}

// ========================================
// TESTS: Preference Dimensions
// ========================================

func TestPreferenceDimensions(t *testing.T) {
	userID := uuid.New()

	t.Run("temperature preferences", func(t *testing.T) {
		temps := []TemperaturePreference{TempCool, TempNormal, TempWarm, TempNoPreference}
		for _, temp := range temps {
			m := new(mockRepo)
			existingPrefs := &RiderPreferences{ID: uuid.New(), UserID: userID}
			m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
			m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
				return p.Temperature == temp
			})).Return(nil)

			svc := newTestService(m)
			prefs, err := svc.UpdatePreferences(context.Background(), userID, &UpdatePreferencesRequest{
				Temperature: &temp,
			})

			require.NoError(t, err)
			assert.Equal(t, temp, prefs.Temperature)
			m.AssertExpectations(t)
		}
	})

	t.Run("music preferences", func(t *testing.T) {
		musics := []MusicPreference{MusicQuiet, MusicLow, MusicDriverChoice, MusicNoPreference}
		for _, music := range musics {
			m := new(mockRepo)
			existingPrefs := &RiderPreferences{ID: uuid.New(), UserID: userID}
			m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
			m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
				return p.Music == music
			})).Return(nil)

			svc := newTestService(m)
			prefs, err := svc.UpdatePreferences(context.Background(), userID, &UpdatePreferencesRequest{
				Music: &music,
			})

			require.NoError(t, err)
			assert.Equal(t, music, prefs.Music)
			m.AssertExpectations(t)
		}
	})

	t.Run("conversation preferences", func(t *testing.T) {
		convs := []ConversationPreference{ConversationQuiet, ConversationFriendly, ConversationNoPreference}
		for _, conv := range convs {
			m := new(mockRepo)
			existingPrefs := &RiderPreferences{ID: uuid.New(), UserID: userID}
			m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
			m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
				return p.Conversation == conv
			})).Return(nil)

			svc := newTestService(m)
			prefs, err := svc.UpdatePreferences(context.Background(), userID, &UpdatePreferencesRequest{
				Conversation: &conv,
			})

			require.NoError(t, err)
			assert.Equal(t, conv, prefs.Conversation)
			m.AssertExpectations(t)
		}
	})

	t.Run("route preferences", func(t *testing.T) {
		routes := []RoutePreference{RouteFastest, RouteCheapest, RouteScenic, RouteNoPreference}
		for _, route := range routes {
			m := new(mockRepo)
			existingPrefs := &RiderPreferences{ID: uuid.New(), UserID: userID}
			m.On("GetRiderPreferences", mock.Anything, userID).Return(existingPrefs, nil)
			m.On("UpsertRiderPreferences", mock.Anything, mock.MatchedBy(func(p *RiderPreferences) bool {
				return p.Route == route
			})).Return(nil)

			svc := newTestService(m)
			prefs, err := svc.UpdatePreferences(context.Background(), userID, &UpdatePreferencesRequest{
				Route: &route,
			})

			require.NoError(t, err)
			assert.Equal(t, route, prefs.Route)
			m.AssertExpectations(t)
		}
	})
}

// ========================================
// TESTS: Edge Cases
// ========================================

func TestEdgeCases(t *testing.T) {
	t.Run("nil override in buildMatchReport", func(t *testing.T) {
		prefs := &RiderPreferences{
			ChildSeat:   true,
			PetFriendly: true,
		}
		caps := &DriverCapabilities{
			HasChildSeat: true,
			PetFriendly:  true,
		}

		matches := buildMatchReport(prefs, nil, caps)
		assert.Len(t, matches, 2)
		for _, m := range matches {
			assert.True(t, m.Match)
		}
	})

	t.Run("empty override in buildMatchReport", func(t *testing.T) {
		prefs := &RiderPreferences{
			ChildSeat:   true,
			PetFriendly: true,
		}
		override := &RidePreferenceOverride{} // Empty override
		caps := &DriverCapabilities{
			HasChildSeat: true,
			PetFriendly:  true,
		}

		matches := buildMatchReport(prefs, override, caps)
		assert.Len(t, matches, 2)
	})

	t.Run("override with false values", func(t *testing.T) {
		prefs := &RiderPreferences{
			ChildSeat:   true,
			PetFriendly: true,
		}
		override := &RidePreferenceOverride{
			ChildSeat:   boolPtr(false),
			PetFriendly: boolPtr(false),
		}
		caps := &DriverCapabilities{
			HasChildSeat: false,
			PetFriendly:  false,
		}

		matches := buildMatchReport(prefs, override, caps)
		assert.Len(t, matches, 0) // No matches because override set to false
	})

	t.Run("capabilities error returns nil capabilities", func(t *testing.T) {
		driverID := uuid.New()

		m := new(mockRepo)
		m.On("GetDriverCapabilities", mock.Anything, driverID).Return(nil, errors.New("db error"))

		svc := newTestService(m)
		caps, err := svc.GetDriverCapabilities(context.Background(), driverID)

		assert.Error(t, err)
		assert.Nil(t, caps)
		m.AssertExpectations(t)
	})
}
