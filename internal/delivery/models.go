package delivery

import (
	"time"

	"github.com/google/uuid"
)

// DeliveryStatus represents the lifecycle of a delivery
type DeliveryStatus string

const (
	DeliveryStatusDraft       DeliveryStatus = "draft"        // Sender filling out details
	DeliveryStatusRequested   DeliveryStatus = "requested"    // Awaiting driver assignment
	DeliveryStatusAccepted    DeliveryStatus = "accepted"     // Driver assigned
	DeliveryStatusPickingUp   DeliveryStatus = "picking_up"   // Driver en route to pickup
	DeliveryStatusPickedUp    DeliveryStatus = "picked_up"    // Package collected from sender
	DeliveryStatusInTransit   DeliveryStatus = "in_transit"   // Package being delivered
	DeliveryStatusArrived     DeliveryStatus = "arrived"      // Driver at drop-off location
	DeliveryStatusDelivered   DeliveryStatus = "delivered"    // Package handed to recipient
	DeliveryStatusReturned    DeliveryStatus = "returned"     // Package returned to sender
	DeliveryStatusCancelled   DeliveryStatus = "cancelled"    // Delivery cancelled
	DeliveryStatusFailed      DeliveryStatus = "failed"       // Delivery failed (no recipient, etc.)
)

// PackageSize classifies packages for pricing and vehicle matching
type PackageSize string

const (
	PackageSizeEnvelope PackageSize = "envelope" // Documents, letters
	PackageSizeSmall    PackageSize = "small"    // Fits in hand (up to 5kg)
	PackageSizeMedium   PackageSize = "medium"   // Fits in car trunk (up to 20kg)
	PackageSizeLarge    PackageSize = "large"    // Requires SUV/van (up to 50kg)
	PackageSizeXLarge   PackageSize = "xlarge"   // Furniture, bulky (50kg+)
)

// DeliveryPriority controls dispatch urgency and pricing
type DeliveryPriority string

const (
	DeliveryPriorityStandard DeliveryPriority = "standard" // 1-3 hours
	DeliveryPriorityExpress  DeliveryPriority = "express"  // Under 1 hour
	DeliveryPriorityScheduled DeliveryPriority = "scheduled" // Specific time slot
)

// ProofType represents how delivery was confirmed
type ProofType string

const (
	ProofTypeSignature  ProofType = "signature"   // Recipient signature
	ProofTypePhoto      ProofType = "photo"        // Photo of delivered package
	ProofTypePIN        ProofType = "pin"          // Recipient enters PIN
	ProofTypeContactless ProofType = "contactless" // Left at door
)

// Delivery represents a package delivery order
type Delivery struct {
	ID               uuid.UUID        `json:"id" db:"id"`
	SenderID         uuid.UUID        `json:"sender_id" db:"sender_id"`
	DriverID         *uuid.UUID       `json:"driver_id,omitempty" db:"driver_id"`
	Status           DeliveryStatus   `json:"status" db:"status"`
	Priority         DeliveryPriority `json:"priority" db:"priority"`
	TrackingCode     string           `json:"tracking_code" db:"tracking_code"`

	// Pickup details
	PickupLatitude   float64    `json:"pickup_latitude" db:"pickup_latitude"`
	PickupLongitude  float64    `json:"pickup_longitude" db:"pickup_longitude"`
	PickupAddress    string     `json:"pickup_address" db:"pickup_address"`
	PickupContact    string     `json:"pickup_contact" db:"pickup_contact"`       // Contact name at pickup
	PickupPhone      string     `json:"pickup_phone" db:"pickup_phone"`           // Phone at pickup
	PickupNotes      *string    `json:"pickup_notes,omitempty" db:"pickup_notes"` // Building, floor, etc.

	// Dropoff details
	DropoffLatitude  float64    `json:"dropoff_latitude" db:"dropoff_latitude"`
	DropoffLongitude float64    `json:"dropoff_longitude" db:"dropoff_longitude"`
	DropoffAddress   string     `json:"dropoff_address" db:"dropoff_address"`
	RecipientName    string     `json:"recipient_name" db:"recipient_name"`
	RecipientPhone   string     `json:"recipient_phone" db:"recipient_phone"`
	DropoffNotes     *string    `json:"dropoff_notes,omitempty" db:"dropoff_notes"`

	// Package details
	PackageSize        PackageSize `json:"package_size" db:"package_size"`
	PackageDescription string      `json:"package_description" db:"package_description"`
	WeightKg           *float64    `json:"weight_kg,omitempty" db:"weight_kg"`
	IsFragile          bool        `json:"is_fragile" db:"is_fragile"`
	RequiresSignature  bool        `json:"requires_signature" db:"requires_signature"`
	DeclaredValue      *float64    `json:"declared_value,omitempty" db:"declared_value"` // For insurance

	// Route & pricing
	EstimatedDistance float64  `json:"estimated_distance" db:"estimated_distance"` // km
	EstimatedDuration int      `json:"estimated_duration" db:"estimated_duration"` // minutes
	EstimatedFare     float64  `json:"estimated_fare" db:"estimated_fare"`
	FinalFare         *float64 `json:"final_fare,omitempty" db:"final_fare"`
	SurgeMultiplier   float64  `json:"surge_multiplier" db:"surge_multiplier"`

	// Delivery proof
	ProofType     *ProofType `json:"proof_type,omitempty" db:"proof_type"`
	ProofPhotoURL *string    `json:"proof_photo_url,omitempty" db:"proof_photo_url"`
	ProofPIN      *string    `json:"-" db:"proof_pin"`     // Never exposed in API
	SignatureURL  *string    `json:"signature_url,omitempty" db:"signature_url"`

	// Scheduling
	ScheduledPickupAt  *time.Time `json:"scheduled_pickup_at,omitempty" db:"scheduled_pickup_at"`
	ScheduledDropoffAt *time.Time `json:"scheduled_dropoff_at,omitempty" db:"scheduled_dropoff_at"`

	// Timestamps
	RequestedAt  time.Time  `json:"requested_at" db:"requested_at"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty" db:"accepted_at"`
	PickedUpAt   *time.Time `json:"picked_up_at,omitempty" db:"picked_up_at"`
	DeliveredAt  *time.Time `json:"delivered_at,omitempty" db:"delivered_at"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancelReason *string    `json:"cancel_reason,omitempty" db:"cancel_reason"`

	// Ratings
	SenderRating    *int    `json:"sender_rating,omitempty" db:"sender_rating"`
	SenderFeedback  *string `json:"sender_feedback,omitempty" db:"sender_feedback"`
	DriverRating    *int    `json:"driver_rating,omitempty" db:"driver_rating"`
	DriverFeedback  *string `json:"driver_feedback,omitempty" db:"driver_feedback"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// DeliveryStop represents an intermediate stop for multi-stop deliveries
type DeliveryStop struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	DeliveryID      uuid.UUID  `json:"delivery_id" db:"delivery_id"`
	StopOrder       int        `json:"stop_order" db:"stop_order"`
	Latitude        float64    `json:"latitude" db:"latitude"`
	Longitude       float64    `json:"longitude" db:"longitude"`
	Address         string     `json:"address" db:"address"`
	ContactName     string     `json:"contact_name" db:"contact_name"`
	ContactPhone    string     `json:"contact_phone" db:"contact_phone"`
	Notes           *string    `json:"notes,omitempty" db:"notes"`
	Status          string     `json:"status" db:"status"` // pending, arrived, completed, skipped
	ArrivedAt       *time.Time `json:"arrived_at,omitempty" db:"arrived_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
	ProofPhotoURL   *string    `json:"proof_photo_url,omitempty" db:"proof_photo_url"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

// DeliveryTracking records location history during transit
type DeliveryTracking struct {
	ID         uuid.UUID `json:"id" db:"id"`
	DeliveryID uuid.UUID `json:"delivery_id" db:"delivery_id"`
	DriverID   uuid.UUID `json:"driver_id" db:"driver_id"`
	Latitude   float64   `json:"latitude" db:"latitude"`
	Longitude  float64   `json:"longitude" db:"longitude"`
	Status     string    `json:"status" db:"status"` // Event description
	Timestamp  time.Time `json:"timestamp" db:"timestamp"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateDeliveryRequest creates a new delivery order
type CreateDeliveryRequest struct {
	// Pickup
	PickupLatitude  float64 `json:"pickup_latitude" binding:"required"`
	PickupLongitude float64 `json:"pickup_longitude" binding:"required"`
	PickupAddress   string  `json:"pickup_address" binding:"required"`
	PickupContact   string  `json:"pickup_contact" binding:"required"`
	PickupPhone     string  `json:"pickup_phone" binding:"required"`
	PickupNotes     *string `json:"pickup_notes,omitempty"`

	// Dropoff
	DropoffLatitude  float64 `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude float64 `json:"dropoff_longitude" binding:"required"`
	DropoffAddress   string  `json:"dropoff_address" binding:"required"`
	RecipientName    string  `json:"recipient_name" binding:"required"`
	RecipientPhone   string  `json:"recipient_phone" binding:"required"`
	DropoffNotes     *string `json:"dropoff_notes,omitempty"`

	// Package
	PackageSize        PackageSize `json:"package_size" binding:"required"`
	PackageDescription string      `json:"package_description" binding:"required"`
	WeightKg           *float64    `json:"weight_kg,omitempty"`
	IsFragile          bool        `json:"is_fragile"`
	RequiresSignature  bool        `json:"requires_signature"`
	DeclaredValue      *float64    `json:"declared_value,omitempty"`

	// Priority & scheduling
	Priority          DeliveryPriority `json:"priority" binding:"required"`
	ScheduledPickupAt *time.Time       `json:"scheduled_pickup_at,omitempty"`

	// Multi-stop
	Stops []StopInput `json:"stops,omitempty"`
}

// StopInput defines an intermediate stop
type StopInput struct {
	Latitude     float64 `json:"latitude" binding:"required"`
	Longitude    float64 `json:"longitude" binding:"required"`
	Address      string  `json:"address" binding:"required"`
	ContactName  string  `json:"contact_name" binding:"required"`
	ContactPhone string  `json:"contact_phone" binding:"required"`
	Notes        *string `json:"notes,omitempty"`
}

// ConfirmPickupRequest is sent when driver collects the package
type ConfirmPickupRequest struct {
	PhotoURL *string `json:"photo_url,omitempty"` // Photo of collected package
	Notes    *string `json:"notes,omitempty"`
}

// ConfirmDeliveryRequest is sent when driver delivers the package
type ConfirmDeliveryRequest struct {
	ProofType    ProofType `json:"proof_type" binding:"required"`
	PhotoURL     *string   `json:"photo_url,omitempty"`
	SignatureURL *string   `json:"signature_url,omitempty"`
	PIN          *string   `json:"pin,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
}

// RateDeliveryRequest rates the delivery experience
type RateDeliveryRequest struct {
	Rating   int     `json:"rating" binding:"required,min=1,max=5"`
	Feedback *string `json:"feedback,omitempty"`
}

// DeliveryEstimateRequest for getting a fare estimate
type DeliveryEstimateRequest struct {
	PickupLatitude   float64      `json:"pickup_latitude" binding:"required"`
	PickupLongitude  float64      `json:"pickup_longitude" binding:"required"`
	DropoffLatitude  float64      `json:"dropoff_latitude" binding:"required"`
	DropoffLongitude float64      `json:"dropoff_longitude" binding:"required"`
	PackageSize      PackageSize  `json:"package_size" binding:"required"`
	Priority         DeliveryPriority `json:"priority" binding:"required"`
	Stops            []StopInput  `json:"stops,omitempty"`
}

// DeliveryEstimateResponse contains fare estimate details
type DeliveryEstimateResponse struct {
	EstimatedDistance float64          `json:"estimated_distance"` // km
	EstimatedDuration int             `json:"estimated_duration"` // minutes
	BaseFare          float64          `json:"base_fare"`
	SizeSurcharge     float64          `json:"size_surcharge"`
	PrioritySurcharge float64          `json:"priority_surcharge"`
	SurgeMultiplier   float64          `json:"surge_multiplier"`
	TotalEstimate     float64          `json:"total_estimate"`
	Currency          string           `json:"currency"`
	Priority          DeliveryPriority `json:"priority"`
	PackageSize       PackageSize      `json:"package_size"`
}

// DeliveryResponse enriches a delivery with contextual data
type DeliveryResponse struct {
	*Delivery
	Stops    []DeliveryStop     `json:"stops,omitempty"`
	Tracking []DeliveryTracking `json:"tracking,omitempty"`
}

// DeliveryListFilters filters for listing deliveries
type DeliveryListFilters struct {
	Status   *DeliveryStatus `json:"status,omitempty"`
	Priority *DeliveryPriority `json:"priority,omitempty"`
	FromDate *time.Time      `json:"from_date,omitempty"`
	ToDate   *time.Time      `json:"to_date,omitempty"`
}

// DeliveryStats provides delivery statistics for a user
type DeliveryStats struct {
	TotalDeliveries   int     `json:"total_deliveries"`
	CompletedCount    int     `json:"completed_count"`
	CancelledCount    int     `json:"cancelled_count"`
	InProgressCount   int     `json:"in_progress_count"`
	AverageRating     float64 `json:"average_rating"`
	TotalSpent        float64 `json:"total_spent"`
	AverageDeliveryMin float64 `json:"average_delivery_minutes"`
}
