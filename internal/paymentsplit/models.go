package paymentsplit

import (
	"time"

	"github.com/google/uuid"
)

// SplitStatus represents the status of a payment split
type SplitStatus string

const (
	SplitStatusPending   SplitStatus = "pending"   // Waiting for participants to accept
	SplitStatusActive    SplitStatus = "active"     // All accepted, waiting for ride completion
	SplitStatusCompleted SplitStatus = "completed"  // All payments collected
	SplitStatusPartial   SplitStatus = "partial"    // Some payments collected
	SplitStatusCancelled SplitStatus = "cancelled"  // Split cancelled
	SplitStatusExpired   SplitStatus = "expired"    // Invitation expired
)

// ParticipantStatus represents a participant's status in the split
type ParticipantStatus string

const (
	ParticipantStatusInvited  ParticipantStatus = "invited"  // Invite sent
	ParticipantStatusAccepted ParticipantStatus = "accepted" // Accepted the split
	ParticipantStatusDeclined ParticipantStatus = "declined" // Declined the split
	ParticipantStatusPaid     ParticipantStatus = "paid"     // Payment collected
	ParticipantStatusFailed   ParticipantStatus = "failed"   // Payment failed
)

// SplitType represents how the fare is divided
type SplitType string

const (
	SplitTypeEqual      SplitType = "equal"      // Divide equally among participants
	SplitTypeCustom     SplitType = "custom"     // Custom amounts per participant
	SplitTypePercentage SplitType = "percentage" // Custom percentages
)

// PaymentSplit represents a fare split between multiple riders
type PaymentSplit struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	RideID           uuid.UUID   `json:"ride_id" db:"ride_id"`
	InitiatorID      uuid.UUID   `json:"initiator_id" db:"initiator_id"`
	SplitType        SplitType   `json:"split_type" db:"split_type"`
	TotalAmount      float64     `json:"total_amount" db:"total_amount"`
	Currency         string      `json:"currency" db:"currency"`
	CollectedAmount  float64     `json:"collected_amount" db:"collected_amount"`
	Status           SplitStatus `json:"status" db:"status"`
	ExpiresAt        time.Time   `json:"expires_at" db:"expires_at"`
	Note             *string     `json:"note,omitempty" db:"note"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

// SplitParticipant represents a participant in a payment split
type SplitParticipant struct {
	ID               uuid.UUID         `json:"id" db:"id"`
	SplitID          uuid.UUID         `json:"split_id" db:"split_id"`
	UserID           *uuid.UUID        `json:"user_id,omitempty" db:"user_id"`
	Phone            *string           `json:"phone,omitempty" db:"phone"`           // For non-app users
	Email            *string           `json:"email,omitempty" db:"email"`           // For non-app users
	DisplayName      string            `json:"display_name" db:"display_name"`
	Amount           float64           `json:"amount" db:"amount"`
	Percentage       *float64          `json:"percentage,omitempty" db:"percentage"` // For percentage splits
	PaymentMethod    *string           `json:"payment_method,omitempty" db:"payment_method"`
	Status           ParticipantStatus `json:"status" db:"status"`
	PaymentID        *uuid.UUID        `json:"payment_id,omitempty" db:"payment_id"`     // Link to actual payment
	InviteSentAt     *time.Time        `json:"invite_sent_at,omitempty" db:"invite_sent_at"`
	RespondedAt      *time.Time        `json:"responded_at,omitempty" db:"responded_at"`
	PaidAt           *time.Time        `json:"paid_at,omitempty" db:"paid_at"`
	ReminderCount    int               `json:"reminder_count" db:"reminder_count"`
	LastReminderAt   *time.Time        `json:"last_reminder_at,omitempty" db:"last_reminder_at"`
	CreatedAt        time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at" db:"updated_at"`
}

// SplitGroup represents a saved group of people for frequent splits
type SplitGroup struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	OwnerID   uuid.UUID       `json:"owner_id" db:"owner_id"`
	Name      string          `json:"name" db:"name"`
	Members   []SplitGroupMember `json:"members"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// SplitGroupMember represents a member in a split group
type SplitGroupMember struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	GroupID     uuid.UUID  `json:"group_id" db:"group_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty" db:"user_id"`
	Phone       *string    `json:"phone,omitempty" db:"phone"`
	DisplayName string     `json:"display_name" db:"display_name"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreateSplitRequest represents a request to create a payment split
type CreateSplitRequest struct {
	RideID       uuid.UUID          `json:"ride_id" binding:"required"`
	SplitType    SplitType          `json:"split_type" binding:"required"`
	Participants []ParticipantInput `json:"participants" binding:"required,min=1"`
	Note         *string            `json:"note,omitempty"`
}

// ParticipantInput defines a participant when creating a split
type ParticipantInput struct {
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	Email       *string    `json:"email,omitempty"`
	DisplayName string     `json:"display_name"`
	Amount      *float64   `json:"amount,omitempty"`     // For custom splits
	Percentage  *float64   `json:"percentage,omitempty"` // For percentage splits
}

// UpdateSplitRequest allows modifying a split before it's finalized
type UpdateSplitRequest struct {
	SplitType    *SplitType         `json:"split_type,omitempty"`
	Participants []ParticipantInput `json:"participants,omitempty"`
	Note         *string            `json:"note,omitempty"`
}

// RespondToSplitRequest represents a participant's response
type RespondToSplitRequest struct {
	Accept        bool    `json:"accept"`
	PaymentMethod *string `json:"payment_method,omitempty"` // card, wallet
}

// SplitResponse represents a full split with participant details
type SplitResponse struct {
	Split        *PaymentSplit      `json:"split"`
	Participants []SplitParticipant `json:"participants"`
	MySplit      *SplitParticipant  `json:"my_split,omitempty"` // Current user's portion
	Summary      *SplitSummary      `json:"summary"`
}

// SplitSummary provides a quick overview of the split status
type SplitSummary struct {
	TotalParticipants int     `json:"total_participants"`
	AcceptedCount     int     `json:"accepted_count"`
	PaidCount         int     `json:"paid_count"`
	DeclinedCount     int     `json:"declined_count"`
	PendingCount      int     `json:"pending_count"`
	CollectedAmount   float64 `json:"collected_amount"`
	RemainingAmount   float64 `json:"remaining_amount"`
	AllAccepted       bool    `json:"all_accepted"`
	AllPaid           bool    `json:"all_paid"`
}

// CreateGroupRequest creates a saved split group
type CreateGroupRequest struct {
	Name    string             `json:"name" binding:"required"`
	Members []GroupMemberInput `json:"members" binding:"required,min=1"`
}

// GroupMemberInput defines a member when creating a group
type GroupMemberInput struct {
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Phone       *string    `json:"phone,omitempty"`
	DisplayName string     `json:"display_name" binding:"required"`
}

// SplitHistory represents a user's split payment history
type SplitHistory struct {
	Initiated []*SplitResponse `json:"initiated"` // Splits I started
	Received  []*SplitResponse `json:"received"`  // Splits I'm part of
	Stats     SplitStats       `json:"stats"`
}

// SplitStats tracks split payment statistics
type SplitStats struct {
	TotalSplitsInitiated int     `json:"total_splits_initiated"`
	TotalSplitsJoined    int     `json:"total_splits_joined"`
	TotalAmountPaid      float64 `json:"total_amount_paid"`
	TotalAmountReceived  float64 `json:"total_amount_received"`
	AvgSplitParticipants float64 `json:"avg_split_participants"`
}
