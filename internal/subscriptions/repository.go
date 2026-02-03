package subscriptions

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles subscription data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new subscription repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// PLANS
// ========================================

// CreatePlan creates a new subscription plan
func (r *Repository) CreatePlan(ctx context.Context, plan *SubscriptionPlan) error {
	rideTypesJSON, _ := json.Marshal(plan.AllowedRideTypes)
	citiesJSON, _ := json.Marshal(plan.AllowedCities)

	query := `
		INSERT INTO subscription_plans (
			id, name, slug, description, plan_type, billing_period,
			price, currency, status, rides_included, max_ride_value,
			discount_pct, allowed_ride_types, allowed_cities, max_distance_km,
			priority_matching, free_upgrades, free_cancellations,
			wait_time_guarantee, surge_protection, surge_max_cap,
			popular_badge, savings_label, display_order,
			trial_days, trial_rides, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
			$16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28
		)
	`

	_, err := r.db.Exec(ctx, query,
		plan.ID, plan.Name, plan.Slug, plan.Description, plan.PlanType,
		plan.BillingPeriod, plan.Price, plan.Currency, plan.Status,
		plan.RidesIncluded, plan.MaxRideValue, plan.DiscountPct,
		rideTypesJSON, citiesJSON, plan.MaxDistanceKm,
		plan.PriorityMatching, plan.FreeUpgrades, plan.FreeCancellations,
		plan.WaitTimeGuarantee, plan.SurgeProtection, plan.SurgeMaxCap,
		plan.PopularBadge, plan.SavingsLabel, plan.DisplayOrder,
		plan.TrialDays, plan.TrialRides, plan.CreatedAt, plan.UpdatedAt,
	)
	return err
}

// GetPlanByID retrieves a plan by ID
func (r *Repository) GetPlanByID(ctx context.Context, id uuid.UUID) (*SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, description, plan_type, billing_period,
			   price, currency, status, rides_included, max_ride_value,
			   discount_pct, allowed_ride_types, allowed_cities, max_distance_km,
			   priority_matching, free_upgrades, free_cancellations,
			   wait_time_guarantee, surge_protection, surge_max_cap,
			   popular_badge, savings_label, display_order,
			   trial_days, trial_rides, created_at, updated_at
		FROM subscription_plans
		WHERE id = $1
	`
	return r.scanPlan(ctx, query, id)
}

// GetPlanBySlug retrieves a plan by slug
func (r *Repository) GetPlanBySlug(ctx context.Context, slug string) (*SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, description, plan_type, billing_period,
			   price, currency, status, rides_included, max_ride_value,
			   discount_pct, allowed_ride_types, allowed_cities, max_distance_km,
			   priority_matching, free_upgrades, free_cancellations,
			   wait_time_guarantee, surge_protection, surge_max_cap,
			   popular_badge, savings_label, display_order,
			   trial_days, trial_rides, created_at, updated_at
		FROM subscription_plans
		WHERE slug = $1
	`
	return r.scanPlan(ctx, query, slug)
}

// ListActivePlans lists all active plans for display
func (r *Repository) ListActivePlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, description, plan_type, billing_period,
			   price, currency, status, rides_included, max_ride_value,
			   discount_pct, allowed_ride_types, allowed_cities, max_distance_km,
			   priority_matching, free_upgrades, free_cancellations,
			   wait_time_guarantee, surge_protection, surge_max_cap,
			   popular_badge, savings_label, display_order,
			   trial_days, trial_rides, created_at, updated_at
		FROM subscription_plans
		WHERE status = 'active'
		ORDER BY display_order ASC, price ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPlans(rows)
}

// ListAllPlans lists all plans including inactive (admin)
func (r *Repository) ListAllPlans(ctx context.Context) ([]*SubscriptionPlan, error) {
	query := `
		SELECT id, name, slug, description, plan_type, billing_period,
			   price, currency, status, rides_included, max_ride_value,
			   discount_pct, allowed_ride_types, allowed_cities, max_distance_km,
			   priority_matching, free_upgrades, free_cancellations,
			   wait_time_guarantee, surge_protection, surge_max_cap,
			   popular_badge, savings_label, display_order,
			   trial_days, trial_rides, created_at, updated_at
		FROM subscription_plans
		WHERE status != 'archived'
		ORDER BY display_order ASC, price ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return r.scanPlans(rows)
}

// UpdatePlanStatus updates plan status
func (r *Repository) UpdatePlanStatus(ctx context.Context, id uuid.UUID, status PlanStatus) error {
	_, err := r.db.Exec(ctx, "UPDATE subscription_plans SET status = $2, updated_at = NOW() WHERE id = $1", id, status)
	return err
}

func (r *Repository) scanPlan(ctx context.Context, query string, arg interface{}) (*SubscriptionPlan, error) {
	plan := &SubscriptionPlan{}
	var rideTypesJSON, citiesJSON []byte
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.PlanType,
		&plan.BillingPeriod, &plan.Price, &plan.Currency, &plan.Status,
		&plan.RidesIncluded, &plan.MaxRideValue, &plan.DiscountPct,
		&rideTypesJSON, &citiesJSON, &plan.MaxDistanceKm,
		&plan.PriorityMatching, &plan.FreeUpgrades, &plan.FreeCancellations,
		&plan.WaitTimeGuarantee, &plan.SurgeProtection, &plan.SurgeMaxCap,
		&plan.PopularBadge, &plan.SavingsLabel, &plan.DisplayOrder,
		&plan.TrialDays, &plan.TrialRides, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(rideTypesJSON, &plan.AllowedRideTypes)
	json.Unmarshal(citiesJSON, &plan.AllowedCities)
	return plan, nil
}

func (r *Repository) scanPlans(rows pgx.Rows) ([]*SubscriptionPlan, error) {
	var plans []*SubscriptionPlan
	for rows.Next() {
		plan := &SubscriptionPlan{}
		var rideTypesJSON, citiesJSON []byte
		err := rows.Scan(
			&plan.ID, &plan.Name, &plan.Slug, &plan.Description, &plan.PlanType,
			&plan.BillingPeriod, &plan.Price, &plan.Currency, &plan.Status,
			&plan.RidesIncluded, &plan.MaxRideValue, &plan.DiscountPct,
			&rideTypesJSON, &citiesJSON, &plan.MaxDistanceKm,
			&plan.PriorityMatching, &plan.FreeUpgrades, &plan.FreeCancellations,
			&plan.WaitTimeGuarantee, &plan.SurgeProtection, &plan.SurgeMaxCap,
			&plan.PopularBadge, &plan.SavingsLabel, &plan.DisplayOrder,
			&plan.TrialDays, &plan.TrialRides, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		json.Unmarshal(rideTypesJSON, &plan.AllowedRideTypes)
		json.Unmarshal(citiesJSON, &plan.AllowedCities)
		plans = append(plans, plan)
	}
	return plans, nil
}

// ========================================
// SUBSCRIPTIONS
// ========================================

// CreateSubscription creates a new user subscription
func (r *Repository) CreateSubscription(ctx context.Context, sub *Subscription) error {
	query := `
		INSERT INTO subscriptions (
			id, user_id, plan_id, status, current_period_start, current_period_end,
			rides_used, upgrades_used, cancellations_used, total_saved,
			payment_method, stripe_sub_id, next_billing_date, last_payment_date,
			failed_payments, is_trial_active, trial_ends_at, activated_at,
			auto_renew, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21
		)
	`

	_, err := r.db.Exec(ctx, query,
		sub.ID, sub.UserID, sub.PlanID, sub.Status,
		sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.RidesUsed, sub.UpgradesUsed, sub.CancellationsUsed,
		sub.TotalSaved, sub.PaymentMethod, sub.StripeSubID,
		sub.NextBillingDate, sub.LastPaymentDate, sub.FailedPayments,
		sub.IsTrialActive, sub.TrialEndsAt, sub.ActivatedAt,
		sub.AutoRenew, sub.CreatedAt, sub.UpdatedAt,
	)
	return err
}

// GetActiveSubscription gets the active subscription for a user
func (r *Repository) GetActiveSubscription(ctx context.Context, userID uuid.UUID) (*Subscription, error) {
	query := `
		SELECT id, user_id, plan_id, status, current_period_start, current_period_end,
			   rides_used, upgrades_used, cancellations_used, total_saved,
			   payment_method, stripe_sub_id, next_billing_date, last_payment_date,
			   failed_payments, is_trial_active, trial_ends_at, activated_at,
			   paused_at, cancelled_at, cancel_reason, auto_renew,
			   created_at, updated_at
		FROM subscriptions
		WHERE user_id = $1 AND status IN ('active', 'paused')
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanSubscription(ctx, query, userID)
}

// GetSubscriptionByID gets a subscription by ID
func (r *Repository) GetSubscriptionByID(ctx context.Context, id uuid.UUID) (*Subscription, error) {
	query := `
		SELECT id, user_id, plan_id, status, current_period_start, current_period_end,
			   rides_used, upgrades_used, cancellations_used, total_saved,
			   payment_method, stripe_sub_id, next_billing_date, last_payment_date,
			   failed_payments, is_trial_active, trial_ends_at, activated_at,
			   paused_at, cancelled_at, cancel_reason, auto_renew,
			   created_at, updated_at
		FROM subscriptions
		WHERE id = $1
	`

	return r.scanSubscription(ctx, query, id)
}

// UpdateSubscription updates a subscription
func (r *Repository) UpdateSubscription(ctx context.Context, sub *Subscription) error {
	query := `
		UPDATE subscriptions SET
			status = $2, current_period_start = $3, current_period_end = $4,
			rides_used = $5, upgrades_used = $6, cancellations_used = $7,
			total_saved = $8, next_billing_date = $9, last_payment_date = $10,
			failed_payments = $11, is_trial_active = $12, trial_ends_at = $13,
			paused_at = $14, cancelled_at = $15, cancel_reason = $16,
			auto_renew = $17, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query,
		sub.ID, sub.Status, sub.CurrentPeriodStart, sub.CurrentPeriodEnd,
		sub.RidesUsed, sub.UpgradesUsed, sub.CancellationsUsed,
		sub.TotalSaved, sub.NextBillingDate, sub.LastPaymentDate,
		sub.FailedPayments, sub.IsTrialActive, sub.TrialEndsAt,
		sub.PausedAt, sub.CancelledAt, sub.CancelReason,
		sub.AutoRenew,
	)
	return err
}

// IncrementRideUsage increments the ride usage count and adds savings
func (r *Repository) IncrementRideUsage(ctx context.Context, subID uuid.UUID, savings float64) error {
	query := `
		UPDATE subscriptions
		SET rides_used = rides_used + 1,
			total_saved = total_saved + $2,
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, subID, savings)
	return err
}

// IncrementUpgradeUsage increments upgrade usage count
func (r *Repository) IncrementUpgradeUsage(ctx context.Context, subID uuid.UUID) error {
	query := `UPDATE subscriptions SET upgrades_used = upgrades_used + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, subID)
	return err
}

// IncrementCancellationUsage increments cancellation usage count
func (r *Repository) IncrementCancellationUsage(ctx context.Context, subID uuid.UUID) error {
	query := `UPDATE subscriptions SET cancellations_used = cancellations_used + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, subID)
	return err
}

func (r *Repository) scanSubscription(ctx context.Context, query string, arg interface{}) (*Subscription, error) {
	sub := &Subscription{}
	err := r.db.QueryRow(ctx, query, arg).Scan(
		&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status,
		&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
		&sub.RidesUsed, &sub.UpgradesUsed, &sub.CancellationsUsed,
		&sub.TotalSaved, &sub.PaymentMethod, &sub.StripeSubID,
		&sub.NextBillingDate, &sub.LastPaymentDate, &sub.FailedPayments,
		&sub.IsTrialActive, &sub.TrialEndsAt, &sub.ActivatedAt,
		&sub.PausedAt, &sub.CancelledAt, &sub.CancelReason,
		&sub.AutoRenew, &sub.CreatedAt, &sub.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return sub, nil
}

// ========================================
// USAGE LOGS
// ========================================

// CreateUsageLog records a subscription usage event
func (r *Repository) CreateUsageLog(ctx context.Context, log *SubscriptionUsageLog) error {
	query := `
		INSERT INTO subscription_usage_logs (id, subscription_id, ride_id, usage_type, original_fare, discounted_fare, savings_amount, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.Exec(ctx, query,
		log.ID, log.SubscriptionID, log.RideID, log.UsageType,
		log.OriginalFare, log.DiscountedFare, log.SavingsAmount, log.CreatedAt,
	)
	return err
}

// GetUsageLogs gets usage logs for a subscription
func (r *Repository) GetUsageLogs(ctx context.Context, subID uuid.UUID, limit, offset int) ([]*SubscriptionUsageLog, error) {
	query := `
		SELECT id, subscription_id, ride_id, usage_type, original_fare, discounted_fare, savings_amount, created_at
		FROM subscription_usage_logs
		WHERE subscription_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, subID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*SubscriptionUsageLog
	for rows.Next() {
		log := &SubscriptionUsageLog{}
		if err := rows.Scan(
			&log.ID, &log.SubscriptionID, &log.RideID, &log.UsageType,
			&log.OriginalFare, &log.DiscountedFare, &log.SavingsAmount, &log.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, nil
}

// GetUserAverageMonthlySpend calculates a user's average monthly ride spend
func (r *Repository) GetUserAverageMonthlySpend(ctx context.Context, userID uuid.UUID) (float64, error) {
	query := `
		SELECT COALESCE(AVG(monthly_spend), 0)
		FROM (
			SELECT DATE_TRUNC('month', created_at) as month, SUM(amount) as monthly_spend
			FROM payments
			WHERE rider_id = $1 AND status = 'completed'
			AND created_at >= NOW() - INTERVAL '6 months'
			GROUP BY DATE_TRUNC('month', created_at)
		) monthly
	`

	var avgSpend float64
	err := r.db.QueryRow(ctx, query, userID).Scan(&avgSpend)
	return avgSpend, err
}

// GetExpiredSubscriptions gets subscriptions that need renewal
func (r *Repository) GetExpiredSubscriptions(ctx context.Context) ([]*Subscription, error) {
	query := `
		SELECT id, user_id, plan_id, status, current_period_start, current_period_end,
			   rides_used, upgrades_used, cancellations_used, total_saved,
			   payment_method, stripe_sub_id, next_billing_date, last_payment_date,
			   failed_payments, is_trial_active, trial_ends_at, activated_at,
			   paused_at, cancelled_at, cancel_reason, auto_renew,
			   created_at, updated_at
		FROM subscriptions
		WHERE status = 'active'
		  AND current_period_end < NOW()
		  AND auto_renew = true
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []*Subscription
	for rows.Next() {
		sub := &Subscription{}
		err := rows.Scan(
			&sub.ID, &sub.UserID, &sub.PlanID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd,
			&sub.RidesUsed, &sub.UpgradesUsed, &sub.CancellationsUsed,
			&sub.TotalSaved, &sub.PaymentMethod, &sub.StripeSubID,
			&sub.NextBillingDate, &sub.LastPaymentDate, &sub.FailedPayments,
			&sub.IsTrialActive, &sub.TrialEndsAt, &sub.ActivatedAt,
			&sub.PausedAt, &sub.CancelledAt, &sub.CancelReason,
			&sub.AutoRenew, &sub.CreatedAt, &sub.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}

// GetSubscriberCount gets the number of active subscribers for a plan
func (r *Repository) GetSubscriberCount(ctx context.Context, planID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM subscriptions WHERE plan_id = $1 AND status IN ('active', 'paused')",
		planID,
	).Scan(&count)
	return count, err
}
