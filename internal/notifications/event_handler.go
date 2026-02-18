package notifications

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/eventbus"
	"github.com/richxcame/ride-hailing/pkg/i18n"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// EventHandler processes events from the NATS event bus and triggers notifications.
type EventHandler struct {
	service *Service
}

// NewEventHandler creates an event handler backed by the notification service.
func NewEventHandler(service *Service) *EventHandler {
	return &EventHandler{service: service}
}

// RegisterSubscriptions subscribes to ride lifecycle events on the bus.
func (h *EventHandler) RegisterSubscriptions(ctx context.Context, bus *eventbus.Bus) error {
	if err := bus.Subscribe(ctx, "rides.>", "notifications-rides", h.handleRideEvent); err != nil {
		return fmt.Errorf("subscribe to rides events: %w", err)
	}
	logger.Info("notifications: subscribed to ride lifecycle events")
	return nil
}

func (h *EventHandler) handleRideEvent(ctx context.Context, event *eventbus.Event) error {
	switch event.Type {
	case "ride.requested":
		return h.onRideRequested(ctx, event)
	case "ride.accepted":
		return h.onRideAccepted(ctx, event)
	case "ride.started":
		return h.onRideStarted(ctx, event)
	case "ride.completed":
		return h.onRideCompleted(ctx, event)
	case "ride.cancelled":
		return h.onRideCancelled(ctx, event)
	default:
		logger.Debug("notifications: ignoring unknown event type", zap.String("type", event.Type))
		return nil
	}
}

func (h *EventHandler) onRideRequested(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideRequestedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride requested: %w", err)
	}

	lang := h.service.userLang(ctx, data.RiderID)
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_requested", "push",
		i18n.Translate("notification.ride.requested.title", lang),
		i18n.Translate("notification.ride.requested.body", lang, data.PickupAddress),
		map[string]interface{}{"ride_id": data.RideID.String()},
	)
	if err != nil {
		logger.Warn("failed to send ride_requested notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideAccepted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideAcceptedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride accepted: %w", err)
	}

	lang := h.service.userLang(ctx, data.RiderID)
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_accepted", "push",
		i18n.Translate("notification.ride.accepted.title", lang),
		// ETA not in the event; use generic accepted body
		i18n.Translate("notification.ride.accepted.title", lang),
		map[string]interface{}{
			"ride_id":   data.RideID.String(),
			"driver_id": data.DriverID.String(),
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_accepted notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideStarted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideStartedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride started: %w", err)
	}

	lang := h.service.userLang(ctx, data.RiderID)
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_started", "push",
		i18n.Translate("notification.ride.started.title", lang),
		i18n.Translate("notification.ride.started.body", lang),
		map[string]interface{}{"ride_id": data.RideID.String()},
	)
	if err != nil {
		logger.Warn("failed to send ride_started notification", zap.Error(err))
	}
	return nil
}

func (h *EventHandler) onRideCompleted(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCompletedData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride completed: %w", err)
	}

	currency := data.Currency
	if currency == "" {
		currency = "USD"
	}
	driverEarnings := data.DriverEarnings
	if driverEarnings == 0 && data.FareAmount > 0 {
		driverEarnings = data.FareAmount * 0.80 // safe fallback
	}

	// Rider: push notification
	riderLang := h.service.userLang(ctx, data.RiderID)
	formattedFare := i18n.FormatAmount(data.FareAmount, currency)
	_, err := h.service.SendNotification(ctx, data.RiderID,
		"ride_completed", "push",
		i18n.Translate("notification.ride.completed.title.rider", riderLang),
		i18n.Translate("notification.ride.completed.body.rider", riderLang, formattedFare),
		map[string]interface{}{
			"ride_id":     data.RideID.String(),
			"fare_amount": data.FareAmount,
			"currency":    currency,
			"distance_km": data.DistanceKm,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_completed push notification", zap.Error(err))
	}

	// Rider: receipt email
	_, err = h.service.SendNotification(ctx, data.RiderID,
		"ride_receipt", "email",
		"Your Ride Receipt",
		fmt.Sprintf("Ride completed. Distance: %.1f km, Duration: %.0f min, Fare: %s",
			data.DistanceKm, data.DurationMin, formattedFare),
		map[string]interface{}{
			"ride_id":      data.RideID.String(),
			"fare_amount":  data.FareAmount,
			"currency":     currency,
			"distance_km":  data.DistanceKm,
			"duration_min": data.DurationMin,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride receipt email", zap.Error(err))
	}

	// Driver: earnings notification
	driverLang := h.service.userLang(ctx, data.DriverID)
	formattedEarnings := i18n.FormatAmount(driverEarnings, currency)
	_, err = h.service.SendNotification(ctx, data.DriverID,
		"payment_received", "push",
		i18n.Translate("notification.ride.completed.title.driver", driverLang),
		i18n.Translate("notification.ride.completed.body.driver", driverLang, formattedEarnings),
		map[string]interface{}{
			"ride_id":  data.RideID.String(),
			"amount":   driverEarnings,
			"currency": currency,
		},
	)
	if err != nil {
		logger.Warn("failed to send payment notification to driver", zap.Error(err))
	}

	return nil
}

func (h *EventHandler) onRideCancelled(ctx context.Context, event *eventbus.Event) error {
	var data eventbus.RideCancelledData
	if err := json.Unmarshal(event.Data, &data); err != nil {
		return fmt.Errorf("unmarshal ride cancelled: %w", err)
	}

	var recipientID uuid.UUID
	var cancelledByKey string

	if data.CancelledBy == "rider" && data.DriverID != uuid.Nil {
		recipientID = data.DriverID
		cancelledByKey = "rider"
	} else if data.CancelledBy == "driver" {
		recipientID = data.RiderID
		cancelledByKey = "driver"
	} else {
		return nil // no one to notify
	}

	lang := h.service.userLang(ctx, recipientID)
	translatedBy := i18n.Translate("notification.ride.cancelled.by."+cancelledByKey, lang)

	_, err := h.service.SendNotification(ctx, recipientID,
		"ride_cancelled", "push",
		i18n.Translate("notification.ride.cancelled.title", lang),
		i18n.Translate("notification.ride.cancelled.body", lang, translatedBy),
		map[string]interface{}{
			"ride_id":      data.RideID.String(),
			"cancelled_by": data.CancelledBy,
			"reason":       data.Reason,
		},
	)
	if err != nil {
		logger.Warn("failed to send ride_cancelled notification", zap.Error(err))
	}
	return nil
}
