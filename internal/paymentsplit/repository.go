package paymentsplit

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles payment split data access
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository creates a new payment split repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// ========================================
// PAYMENT SPLIT OPERATIONS
// ========================================

// CreateSplit creates a payment split with participants in a transaction
func (r *Repository) CreateSplit(ctx context.Context, split *PaymentSplit, participants []*SplitParticipant) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert split
	splitQuery := `
		INSERT INTO payment_splits (
			id, ride_id, initiator_id, split_type, total_amount,
			currency, collected_amount, status, expires_at, note,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`
	_, err = tx.Exec(ctx, splitQuery,
		split.ID, split.RideID, split.InitiatorID, split.SplitType,
		split.TotalAmount, split.Currency, split.CollectedAmount,
		split.Status, split.ExpiresAt, split.Note,
		split.CreatedAt, split.UpdatedAt,
	)
	if err != nil {
		return err
	}

	// Insert participants
	participantQuery := `
		INSERT INTO split_participants (
			id, split_id, user_id, phone, email, display_name,
			amount, percentage, status, invite_sent_at, reminder_count,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
	`
	for _, p := range participants {
		_, err = tx.Exec(ctx, participantQuery,
			p.ID, p.SplitID, p.UserID, p.Phone, p.Email,
			p.DisplayName, p.Amount, p.Percentage, p.Status,
			p.InviteSentAt, p.ReminderCount, p.CreatedAt, p.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetSplit retrieves a payment split by ID
func (r *Repository) GetSplit(ctx context.Context, splitID uuid.UUID) (*PaymentSplit, error) {
	query := `
		SELECT id, ride_id, initiator_id, split_type, total_amount,
			   currency, collected_amount, status, expires_at, note,
			   created_at, updated_at
		FROM payment_splits
		WHERE id = $1
	`

	split := &PaymentSplit{}
	err := r.db.QueryRow(ctx, query, splitID).Scan(
		&split.ID, &split.RideID, &split.InitiatorID, &split.SplitType,
		&split.TotalAmount, &split.Currency, &split.CollectedAmount,
		&split.Status, &split.ExpiresAt, &split.Note,
		&split.CreatedAt, &split.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return split, nil
}

// GetSplitByRideID retrieves a payment split for a specific ride
func (r *Repository) GetSplitByRideID(ctx context.Context, rideID uuid.UUID) (*PaymentSplit, error) {
	query := `
		SELECT id, ride_id, initiator_id, split_type, total_amount,
			   currency, collected_amount, status, expires_at, note,
			   created_at, updated_at
		FROM payment_splits
		WHERE ride_id = $1 AND status != 'cancelled'
		ORDER BY created_at DESC
		LIMIT 1
	`

	split := &PaymentSplit{}
	err := r.db.QueryRow(ctx, query, rideID).Scan(
		&split.ID, &split.RideID, &split.InitiatorID, &split.SplitType,
		&split.TotalAmount, &split.Currency, &split.CollectedAmount,
		&split.Status, &split.ExpiresAt, &split.Note,
		&split.CreatedAt, &split.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return split, nil
}

// UpdateSplitStatus updates the status of a split
func (r *Repository) UpdateSplitStatus(ctx context.Context, splitID uuid.UUID, status SplitStatus) error {
	query := `
		UPDATE payment_splits
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, splitID, status)
	return err
}

// UpdateCollectedAmount updates the collected amount for a split
func (r *Repository) UpdateCollectedAmount(ctx context.Context, splitID uuid.UUID, amount float64) error {
	query := `
		UPDATE payment_splits
		SET collected_amount = $2, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, splitID, amount)
	return err
}

// ========================================
// PARTICIPANT OPERATIONS
// ========================================

// GetParticipants retrieves all participants for a split
func (r *Repository) GetParticipants(ctx context.Context, splitID uuid.UUID) ([]*SplitParticipant, error) {
	query := `
		SELECT id, split_id, user_id, phone, email, display_name,
			   amount, percentage, payment_method, status, payment_id,
			   invite_sent_at, responded_at, paid_at, reminder_count,
			   last_reminder_at, created_at, updated_at
		FROM split_participants
		WHERE split_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.Query(ctx, query, splitID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*SplitParticipant
	for rows.Next() {
		p := &SplitParticipant{}
		err := rows.Scan(
			&p.ID, &p.SplitID, &p.UserID, &p.Phone, &p.Email,
			&p.DisplayName, &p.Amount, &p.Percentage, &p.PaymentMethod,
			&p.Status, &p.PaymentID, &p.InviteSentAt, &p.RespondedAt,
			&p.PaidAt, &p.ReminderCount, &p.LastReminderAt,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// GetParticipantByUserID gets a specific participant by user ID
func (r *Repository) GetParticipantByUserID(ctx context.Context, splitID, userID uuid.UUID) (*SplitParticipant, error) {
	query := `
		SELECT id, split_id, user_id, phone, email, display_name,
			   amount, percentage, payment_method, status, payment_id,
			   invite_sent_at, responded_at, paid_at, reminder_count,
			   last_reminder_at, created_at, updated_at
		FROM split_participants
		WHERE split_id = $1 AND user_id = $2
	`

	p := &SplitParticipant{}
	err := r.db.QueryRow(ctx, query, splitID, userID).Scan(
		&p.ID, &p.SplitID, &p.UserID, &p.Phone, &p.Email,
		&p.DisplayName, &p.Amount, &p.Percentage, &p.PaymentMethod,
		&p.Status, &p.PaymentID, &p.InviteSentAt, &p.RespondedAt,
		&p.PaidAt, &p.ReminderCount, &p.LastReminderAt,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return p, nil
}

// UpdateParticipantStatus updates a participant's status
func (r *Repository) UpdateParticipantStatus(ctx context.Context, participantID uuid.UUID, status ParticipantStatus) error {
	query := `
		UPDATE split_participants
		SET status = $2, responded_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, participantID, status)
	return err
}

// UpdateParticipantPayment records payment details for a participant
func (r *Repository) UpdateParticipantPayment(ctx context.Context, participantID uuid.UUID, paymentID uuid.UUID, paymentMethod string) error {
	query := `
		UPDATE split_participants
		SET status = 'paid',
			payment_id = $2,
			payment_method = $3,
			paid_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, participantID, paymentID, paymentMethod)
	return err
}

// UpdateParticipantReminder updates reminder tracking
func (r *Repository) UpdateParticipantReminder(ctx context.Context, participantID uuid.UUID) error {
	query := `
		UPDATE split_participants
		SET reminder_count = reminder_count + 1,
			last_reminder_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, participantID)
	return err
}

// ========================================
// SPLIT GROUP OPERATIONS
// ========================================

// CreateGroup creates a saved split group
func (r *Repository) CreateGroup(ctx context.Context, group *SplitGroup) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	groupQuery := `
		INSERT INTO split_groups (id, owner_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err = tx.Exec(ctx, groupQuery, group.ID, group.OwnerID, group.Name, group.CreatedAt, group.UpdatedAt)
	if err != nil {
		return err
	}

	memberQuery := `
		INSERT INTO split_group_members (id, group_id, user_id, phone, display_name, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	for _, m := range group.Members {
		_, err = tx.Exec(ctx, memberQuery,
			m.ID, m.GroupID, m.UserID, m.Phone, m.DisplayName, m.CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetGroupsByOwner lists split groups for a user
func (r *Repository) GetGroupsByOwner(ctx context.Context, ownerID uuid.UUID) ([]*SplitGroup, error) {
	query := `
		SELECT id, owner_id, name, created_at, updated_at
		FROM split_groups
		WHERE owner_id = $1
		ORDER BY updated_at DESC
	`

	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*SplitGroup
	for rows.Next() {
		g := &SplitGroup{}
		if err := rows.Scan(&g.ID, &g.OwnerID, &g.Name, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}

		// Load members
		members, err := r.GetGroupMembers(ctx, g.ID)
		if err != nil {
			return nil, err
		}
		g.Members = members

		groups = append(groups, g)
	}

	return groups, nil
}

// GetGroupMembers retrieves members of a split group
func (r *Repository) GetGroupMembers(ctx context.Context, groupID uuid.UUID) ([]SplitGroupMember, error) {
	query := `
		SELECT id, group_id, user_id, phone, display_name, created_at
		FROM split_group_members
		WHERE group_id = $1
		ORDER BY display_name ASC
	`

	rows, err := r.db.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []SplitGroupMember
	for rows.Next() {
		m := SplitGroupMember{}
		if err := rows.Scan(&m.ID, &m.GroupID, &m.UserID, &m.Phone, &m.DisplayName, &m.CreatedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}

	return members, nil
}

// DeleteGroup deletes a split group
func (r *Repository) DeleteGroup(ctx context.Context, groupID, ownerID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Delete members first
	_, err = tx.Exec(ctx, "DELETE FROM split_group_members WHERE group_id = $1", groupID)
	if err != nil {
		return err
	}

	// Delete group
	_, err = tx.Exec(ctx, "DELETE FROM split_groups WHERE id = $1 AND owner_id = $2", groupID, ownerID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// ========================================
// HISTORY & STATS
// ========================================

// GetSplitsByInitiator gets splits initiated by a user
func (r *Repository) GetSplitsByInitiator(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PaymentSplit, error) {
	query := `
		SELECT id, ride_id, initiator_id, split_type, total_amount,
			   currency, collected_amount, status, expires_at, note,
			   created_at, updated_at
		FROM payment_splits
		WHERE initiator_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []*PaymentSplit
	for rows.Next() {
		s := &PaymentSplit{}
		err := rows.Scan(
			&s.ID, &s.RideID, &s.InitiatorID, &s.SplitType,
			&s.TotalAmount, &s.Currency, &s.CollectedAmount,
			&s.Status, &s.ExpiresAt, &s.Note,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		splits = append(splits, s)
	}

	return splits, nil
}

// GetSplitsForParticipant gets splits where user is a participant
func (r *Repository) GetSplitsForParticipant(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*PaymentSplit, error) {
	query := `
		SELECT DISTINCT ps.id, ps.ride_id, ps.initiator_id, ps.split_type,
			   ps.total_amount, ps.currency, ps.collected_amount, ps.status,
			   ps.expires_at, ps.note, ps.created_at, ps.updated_at
		FROM payment_splits ps
		INNER JOIN split_participants sp ON ps.id = sp.split_id
		WHERE sp.user_id = $1 AND ps.initiator_id != $1
		ORDER BY ps.created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []*PaymentSplit
	for rows.Next() {
		s := &PaymentSplit{}
		err := rows.Scan(
			&s.ID, &s.RideID, &s.InitiatorID, &s.SplitType,
			&s.TotalAmount, &s.Currency, &s.CollectedAmount,
			&s.Status, &s.ExpiresAt, &s.Note,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		splits = append(splits, s)
	}

	return splits, nil
}

// GetSplitStats gets payment split statistics for a user
func (r *Repository) GetSplitStats(ctx context.Context, userID uuid.UUID) (*SplitStats, error) {
	query := `
		SELECT
			(SELECT COUNT(*) FROM payment_splits WHERE initiator_id = $1) as initiated,
			(SELECT COUNT(DISTINCT ps.id) FROM payment_splits ps
			 INNER JOIN split_participants sp ON ps.id = sp.split_id
			 WHERE sp.user_id = $1 AND ps.initiator_id != $1) as joined,
			(SELECT COALESCE(SUM(sp.amount), 0) FROM split_participants sp
			 WHERE sp.user_id = $1 AND sp.status = 'paid') as amount_paid,
			(SELECT COALESCE(SUM(collected_amount), 0) FROM payment_splits
			 WHERE initiator_id = $1 AND status = 'completed') as amount_received
	`

	stats := &SplitStats{}
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&stats.TotalSplitsInitiated, &stats.TotalSplitsJoined,
		&stats.TotalAmountPaid, &stats.TotalAmountReceived,
	)
	if err != nil {
		return &SplitStats{}, nil
	}

	return stats, nil
}

// GetExpiredSplits gets splits that have expired but are still pending
func (r *Repository) GetExpiredSplits(ctx context.Context) ([]*PaymentSplit, error) {
	query := `
		SELECT id, ride_id, initiator_id, split_type, total_amount,
			   currency, collected_amount, status, expires_at, note,
			   created_at, updated_at
		FROM payment_splits
		WHERE status = 'pending' AND expires_at < NOW()
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var splits []*PaymentSplit
	for rows.Next() {
		s := &PaymentSplit{}
		err := rows.Scan(
			&s.ID, &s.RideID, &s.InitiatorID, &s.SplitType,
			&s.TotalAmount, &s.Currency, &s.CollectedAmount,
			&s.Status, &s.ExpiresAt, &s.Note,
			&s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		splits = append(splits, s)
	}

	return splits, nil
}

// GetPendingReminders gets participants who need reminders
func (r *Repository) GetPendingReminders(ctx context.Context) ([]*SplitParticipant, error) {
	query := `
		SELECT sp.id, sp.split_id, sp.user_id, sp.phone, sp.email,
			   sp.display_name, sp.amount, sp.percentage, sp.payment_method,
			   sp.status, sp.payment_id, sp.invite_sent_at, sp.responded_at,
			   sp.paid_at, sp.reminder_count, sp.last_reminder_at,
			   sp.created_at, sp.updated_at
		FROM split_participants sp
		INNER JOIN payment_splits ps ON sp.split_id = ps.id
		WHERE sp.status IN ('invited', 'accepted')
		  AND ps.status IN ('pending', 'active')
		  AND sp.reminder_count < 3
		  AND (sp.last_reminder_at IS NULL OR sp.last_reminder_at < NOW() - INTERVAL '1 hour')
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []*SplitParticipant
	for rows.Next() {
		p := &SplitParticipant{}
		err := rows.Scan(
			&p.ID, &p.SplitID, &p.UserID, &p.Phone, &p.Email,
			&p.DisplayName, &p.Amount, &p.Percentage, &p.PaymentMethod,
			&p.Status, &p.PaymentID, &p.InviteSentAt, &p.RespondedAt,
			&p.PaidAt, &p.ReminderCount, &p.LastReminderAt,
			&p.CreatedAt, &p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}
