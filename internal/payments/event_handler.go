package payments

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// EventHandler processes ride events and triggers payment operations.
type EventHandler struct {
	service *Service
}

// NewEventHandler creates an event handler backed by the payment service.
func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{service: service}
}

// RegisterSubscriptions subscribes to ride completion events on the bus.
func (h *EventHandler) RegisterSubscriptions(ctx context.Context, bus *eventbus.Bus) error {
	if err := bus.Subscribe(ctx, "rides.completed", "payments-ride-completed", h.handleRideCompleted); err != nil {
		return fmt.Errorf("subscribe to rides.completed: %w", err)
	}
	logger.Info("payments: subscribed to ride completion events for driver payouts")
	return nil
}

func (h *EventHandler) handleRideCompleted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCompletedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride completed: %w", err)
	}

	logger.Info("payments: processing driver payout for completed ride",
		zap.String("ride_id", data.RideID.String()),
		zap.String("driver_id", data.DriverID.String()),
		zap.Float64("fare_amount", data.FareAmount),
	)

	// Find the completed payment for this ride
	payments, err := h.service.repo.GetPaymentsByRideID(ctx, data.RideID)
	if err != nil {
		logger.Error("payments: failed to get payments for ride",
			zap.String("ride_id", data.RideID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("get payments for ride: %w", err)
	}

	// Find a completed payment to trigger payout
	for _, payment := range payments {
		if payment.Status == "completed" {
			if err := h.service.PayoutToDriver(ctx, payment.ID); err != nil {
				logger.Error("payments: failed to process driver payout",
					zap.String("payment_id", payment.ID.String()),
					zap.String("ride_id", data.RideID.String()),
					zap.Error(err),
				)
				return fmt.Errorf("payout to driver: %w", err)
			}

			logger.Info("payments: driver payout processed successfully",
				zap.String("payment_id", payment.ID.String()),
				zap.String("driver_id", data.DriverID.String()),
			)
			return nil
		}
	}

	logger.Warn("payments: no completed payment found for ride, skipping payout",
		zap.String("ride_id", data.RideID.String()),
	)
	return nil
}
