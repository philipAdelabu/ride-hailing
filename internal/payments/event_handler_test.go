package payments

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func makeEvent(t *testing.T, data eventbus.RideCompletedData) *eventbus.Event {
	t.Helper()
	raw, err := json.Marshal(data)
	require.NoError(t, err)
	return &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      "ride.completed",
		Source:    "rides-service",
		Timestamp: time.Now(),
		Data:      raw,
	}
}

func newEventHandlerWithMock(mockRepo *mocks.MockPaymentsRepository) (*EventHandler, *mocks.MockStripeClient) {
	mockStripe := new(mocks.MockStripeClient)
	service := NewService(mockRepo, mockStripe, nil)
	return NewEventHandler(service), mockStripe
}

// ─── handleRideCompleted ──────────────────────────────────────────────────────

func TestHandleRideCompleted_Success(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	handler, _ := newEventHandlerWithMock(mockRepo)

	rideID := uuid.New()
	driverID := uuid.New()
	fare := 100.0
	commission := fare * defaultCommissionRate
	net := fare - commission

	mockRepo.On("RecordRideEarning",
		mock.Anything, driverID, rideID,
		fare, commission, net, mock.AnythingOfType("string"),
	).Return(nil)

	data := eventbus.RideCompletedData{
		RideID:      rideID,
		RiderID:     uuid.New(),
		DriverID:    driverID,
		FareAmount:  fare,
		DistanceKm:  10.5,
		DurationMin: 20,
		CompletedAt: time.Now(),
	}

	err := handler.handleRideCompleted(context.Background(), makeEvent(t, data))

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestHandleRideCompleted_ZeroFare_Skipped(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	handler, _ := newEventHandlerWithMock(mockRepo)

	data := eventbus.RideCompletedData{
		RideID:      uuid.New(),
		RiderID:     uuid.New(),
		DriverID:    uuid.New(),
		FareAmount:  0, // zero fare — should be skipped
		DistanceKm:  5.0,
		DurationMin: 10,
		CompletedAt: time.Now(),
	}

	err := handler.handleRideCompleted(context.Background(), makeEvent(t, data))

	assert.NoError(t, err)
	mockRepo.AssertNotCalled(t, "RecordRideEarning")
}

func TestHandleRideCompleted_RepoError(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	handler, _ := newEventHandlerWithMock(mockRepo)

	rideID := uuid.New()
	driverID := uuid.New()
	fare := 75.0
	commission := fare * defaultCommissionRate
	net := fare - commission

	mockRepo.On("RecordRideEarning",
		mock.Anything, driverID, rideID,
		fare, commission, net, mock.AnythingOfType("string"),
	).Return(errors.New("db error"))

	data := eventbus.RideCompletedData{
		RideID:      rideID,
		RiderID:     uuid.New(),
		DriverID:    driverID,
		FareAmount:  fare,
		DistanceKm:  7.5,
		DurationMin: 15,
		CompletedAt: time.Now(),
	}

	err := handler.handleRideCompleted(context.Background(), makeEvent(t, data))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "record driver earning")
	mockRepo.AssertExpectations(t)
}

func TestHandleRideCompleted_MalformedEvent(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	handler, _ := newEventHandlerWithMock(mockRepo)

	event := &eventbus.Event{
		ID:        uuid.New().String(),
		Type:      "ride.completed",
		Source:    "test",
		Timestamp: time.Now(),
		Data:      []byte("invalid json{"),
	}

	err := handler.handleRideCompleted(context.Background(), event)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal ride completed")
	mockRepo.AssertNotCalled(t, "RecordRideEarning")
}

func TestHandleRideCompleted_CommissionCalculation(t *testing.T) {
	tests := []struct {
		name           string
		fare           float64
		wantCommission float64
		wantNet        float64
	}{
		{"100 fare 20% commission", 100.0, 20.0, 80.0},
		{"50 fare 20% commission", 50.0, 10.0, 40.0},
		{"6.78 fare 20% commission", 6.78, 6.78 * 0.20, 6.78 * 0.80},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := new(mocks.MockPaymentsRepository)
			handler, _ := newEventHandlerWithMock(mockRepo)

			rideID := uuid.New()
			driverID := uuid.New()

			mockRepo.On("RecordRideEarning",
				mock.Anything, driverID, rideID,
				tt.fare, tt.wantCommission, tt.wantNet, mock.AnythingOfType("string"),
			).Return(nil)

			data := eventbus.RideCompletedData{
				RideID:      rideID,
				DriverID:    driverID,
				RiderID:     uuid.New(),
				FareAmount:  tt.fare,
				CompletedAt: time.Now(),
			}

			err := handler.handleRideCompleted(context.Background(), makeEvent(t, data))

			assert.NoError(t, err)
			mockRepo.AssertExpectations(t)
		})
	}
}

// ─── NewEventHandler ─────────────────────────────────────────────────────────

func TestNewEventHandler(t *testing.T) {
	mockRepo := new(mocks.MockPaymentsRepository)
	handler, _ := newEventHandlerWithMock(mockRepo)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.service)
}
