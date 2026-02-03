package subscriptions

import (
	"time"

	"github.com/google/uuid"
)

// PlanStatus represents the status of a subscription plan
type PlanStatus string

const (
	PlanStatusActive   PlanStatus = "active"
	PlanStatusInactive PlanStatus = "inactive"
	PlanStatusArchived PlanStatus = "archived"
)

// BillingPeriod represents the billing cycle
type BillingPeriod string

const (
	BillingWeekly  BillingPeriod = "weekly"
	BillingMonthly BillingPeriod = "monthly"
	BillingYearly  BillingPeriod = "yearly"
)

// PlanType represents the type of ride pass
type PlanType string

const (
	PlanTypeUnlimited PlanType = "unlimited" // Unlimited rides within constraints
	PlanTypePackage   PlanType = "package"   // Fixed number of rides
	PlanTypeDiscount  PlanType = "discount"  // Discount on all rides
	PlanTypePriority  PlanType = "priority"  // Priority matching + perks
)

// SubscriptionStatus represents a user's subscription status
type SubscriptionStatus string

const (
	SubStatusActive    SubscriptionStatus = "active"
	SubStatusPaused    SubscriptionStatus = "paused"
	SubStatusCancelled SubscriptionStatus = "cancelled"
	SubStatusExpired   SubscriptionStatus = "expired"
	SubStatusPastDue   SubscriptionStatus = "past_due"
)

// SubscriptionPlan represents a ride pass plan
type SubscriptionPlan struct {
	ID            uuid.UUID     `json:"id" db:"id"`
	Name          string        `json:"name" db:"name"`
	Slug          string        `json:"slug" db:"slug"` // URL-friendly identifier
	Description   string        `json:"description" db:"description"`
	PlanType      PlanType      `json:"plan_type" db:"plan_type"`
	BillingPeriod BillingPeriod `json:"billing_period" db:"billing_period"`
	Price         float64       `json:"price" db:"price"`
	Currency      string        `json:"currency" db:"currency"`
	Status        PlanStatus    `json:"status" db:"status"`

	// Ride allowance
	RidesIncluded *int     `json:"rides_included,omitempty" db:"rides_included"` // nil = unlimited
	MaxRideValue  *float64 `json:"max_ride_value,omitempty" db:"max_ride_value"` // Per-ride cap
	DiscountPct   float64  `json:"discount_pct" db:"discount_pct"`              // e.g., 15 = 15% off

	// Constraints
	AllowedRideTypes []string `json:"allowed_ride_types,omitempty"` // Empty = all types
	AllowedCities    []string `json:"allowed_cities,omitempty"`     // Empty = all cities
	MaxDistanceKm    *float64 `json:"max_distance_km,omitempty" db:"max_distance_km"`

	// Perks
	PriorityMatching bool    `json:"priority_matching" db:"priority_matching"` // Faster driver matching
	FreeUpgrades     int     `json:"free_upgrades" db:"free_upgrades"`         // Free upgrade rides per period
	FreeCancellations int    `json:"free_cancellations" db:"free_cancellations"` // Free cancellation count
	WaitTimeGuarantee *int   `json:"wait_time_guarantee,omitempty" db:"wait_time_guarantee"` // Max wait in minutes
	SurgeProtection  bool    `json:"surge_protection" db:"surge_protection"`   // No surge pricing
	SurgeMaxCap      *float64 `json:"surge_max_cap,omitempty" db:"surge_max_cap"` // Cap surge at this multiplier

	// Display
	PopularBadge  bool   `json:"popular_badge" db:"popular_badge"`   // Show "Most Popular" badge
	SavingsLabel  string `json:"savings_label" db:"savings_label"`   // e.g., "Save up to 20%"
	DisplayOrder  int    `json:"display_order" db:"display_order"`

	// Trial
	TrialDays   int  `json:"trial_days" db:"trial_days"` // Free trial period
	TrialRides  int  `json:"trial_rides" db:"trial_rides"` // Free rides during trial

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Subscription represents a user's active subscription
type Subscription struct {
	ID                 uuid.UUID          `json:"id" db:"id"`
	UserID             uuid.UUID          `json:"user_id" db:"user_id"`
	PlanID             uuid.UUID          `json:"plan_id" db:"plan_id"`
	Status             SubscriptionStatus `json:"status" db:"status"`
	CurrentPeriodStart time.Time          `json:"current_period_start" db:"current_period_start"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end" db:"current_period_end"`

	// Usage tracking
	RidesUsed          int     `json:"rides_used" db:"rides_used"`
	RidesRemaining     *int    `json:"rides_remaining,omitempty"` // nil = unlimited
	UpgradesUsed       int     `json:"upgrades_used" db:"upgrades_used"`
	CancellationsUsed  int     `json:"cancellations_used" db:"cancellations_used"`
	TotalSaved         float64 `json:"total_saved" db:"total_saved"` // Total savings from subscription

	// Payment
	PaymentMethod    string     `json:"payment_method" db:"payment_method"`
	StripeSubID      *string    `json:"stripe_sub_id,omitempty" db:"stripe_sub_id"`
	NextBillingDate  *time.Time `json:"next_billing_date,omitempty" db:"next_billing_date"`
	LastPaymentDate  *time.Time `json:"last_payment_date,omitempty" db:"last_payment_date"`
	FailedPayments   int        `json:"failed_payments" db:"failed_payments"`

	// Trial
	IsTrialActive  bool       `json:"is_trial_active" db:"is_trial_active"`
	TrialEndsAt    *time.Time `json:"trial_ends_at,omitempty" db:"trial_ends_at"`

	// Lifecycle
	ActivatedAt  *time.Time `json:"activated_at,omitempty" db:"activated_at"`
	PausedAt     *time.Time `json:"paused_at,omitempty" db:"paused_at"`
	CancelledAt  *time.Time `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancelReason *string    `json:"cancel_reason,omitempty" db:"cancel_reason"`
	AutoRenew    bool       `json:"auto_renew" db:"auto_renew"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// SubscriptionUsageLog records each time the subscription benefit is used
type SubscriptionUsageLog struct {
	ID             uuid.UUID `json:"id" db:"id"`
	SubscriptionID uuid.UUID `json:"subscription_id" db:"subscription_id"`
	RideID         uuid.UUID `json:"ride_id" db:"ride_id"`
	UsageType      string    `json:"usage_type" db:"usage_type"` // "ride", "upgrade", "cancellation"
	OriginalFare   float64   `json:"original_fare" db:"original_fare"`
	DiscountedFare float64   `json:"discounted_fare" db:"discounted_fare"`
	SavingsAmount  float64   `json:"savings_amount" db:"savings_amount"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// ========================================
// REQUEST/RESPONSE TYPES
// ========================================

// CreatePlanRequest creates a new subscription plan
type CreatePlanRequest struct {
	Name              string        `json:"name" binding:"required"`
	Slug              string        `json:"slug" binding:"required"`
	Description       string        `json:"description" binding:"required"`
	PlanType          PlanType      `json:"plan_type" binding:"required"`
	BillingPeriod     BillingPeriod `json:"billing_period" binding:"required"`
	Price             float64       `json:"price" binding:"required"`
	Currency          string        `json:"currency" binding:"required"`
	RidesIncluded     *int          `json:"rides_included,omitempty"`
	MaxRideValue      *float64      `json:"max_ride_value,omitempty"`
	DiscountPct       float64       `json:"discount_pct"`
	AllowedRideTypes  []string      `json:"allowed_ride_types,omitempty"`
	AllowedCities     []string      `json:"allowed_cities,omitempty"`
	MaxDistanceKm     *float64      `json:"max_distance_km,omitempty"`
	PriorityMatching  bool          `json:"priority_matching"`
	FreeUpgrades      int           `json:"free_upgrades"`
	FreeCancellations int           `json:"free_cancellations"`
	SurgeProtection   bool          `json:"surge_protection"`
	SurgeMaxCap       *float64      `json:"surge_max_cap,omitempty"`
	TrialDays         int           `json:"trial_days"`
	TrialRides        int           `json:"trial_rides"`
	PopularBadge      bool          `json:"popular_badge"`
	SavingsLabel      string        `json:"savings_label"`
	DisplayOrder      int           `json:"display_order"`
}

// SubscribeRequest represents a request to subscribe to a plan
type SubscribeRequest struct {
	PlanID        uuid.UUID `json:"plan_id" binding:"required"`
	PaymentMethod string    `json:"payment_method" binding:"required"`
	AutoRenew     bool      `json:"auto_renew"`
}

// SubscriptionResponse returns subscription details with plan info
type SubscriptionResponse struct {
	Subscription *Subscription     `json:"subscription"`
	Plan         *SubscriptionPlan `json:"plan"`
	Usage        *UsageSummary     `json:"usage"`
}

// UsageSummary provides current period usage overview
type UsageSummary struct {
	RidesUsed       int      `json:"rides_used"`
	RidesRemaining  *int     `json:"rides_remaining,omitempty"` // nil = unlimited
	UpgradesUsed    int      `json:"upgrades_used"`
	UpgradesLeft    int      `json:"upgrades_left"`
	CancellationsUsed int   `json:"cancellations_used"`
	CancellationsLeft int   `json:"cancellations_left"`
	TotalSaved      float64  `json:"total_saved"`
	DaysRemaining   int      `json:"days_remaining"`
	UtilizationPct  float64  `json:"utilization_pct"` // How much of the plan they've used
}

// PlanComparisonResponse compares available plans
type PlanComparisonResponse struct {
	Plans           []*SubscriptionPlan `json:"plans"`
	UserAvgSpend    float64             `json:"user_avg_monthly_spend"` // User's average spending
	BestValuePlan   *uuid.UUID          `json:"best_value_plan_id,omitempty"` // Recommended plan
	EstimatedSavings map[uuid.UUID]float64 `json:"estimated_savings"` // Per-plan savings estimate
}
