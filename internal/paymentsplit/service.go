package paymentsplit

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/richxcame/ride-hailing/pkg/common"
	"github.com/richxcame/ride-hailing/pkg/logger"
	"go.uber.org/zap"
)

// PaymentService interface for processing individual payments
type PaymentService interface {
	GetRideFare(ctx context.Context, rideID uuid.UUID) (float64, string, error) // Returns amount, currency
	ProcessSplitPayment(ctx context.Context, userID uuid.UUID, rideID uuid.UUID, amount float64, method string) (uuid.UUID, error)
}

// NotificationService interface for sending split notifications
type NotificationService interface {
	SendSplitInvitation(ctx context.Context, participantID uuid.UUID, splitID uuid.UUID, initiatorName string, amount float64) error
	SendSplitReminder(ctx context.Context, participantID uuid.UUID, splitID uuid.UUID, amount float64) error
	SendSplitAccepted(ctx context.Context, initiatorID uuid.UUID, participantName string) error
	SendSplitCompleted(ctx context.Context, splitID uuid.UUID) error
}

// Service handles payment split business logic
type Service struct {
	repo            *Repository
	paymentSvc      PaymentService
	notificationSvc NotificationService
}

// NewService creates a new payment split service
func NewService(repo *Repository, paymentSvc PaymentService, notificationSvc NotificationService) *Service {
	return &Service{
		repo:            repo,
		paymentSvc:      paymentSvc,
		notificationSvc: notificationSvc,
	}
}

// ========================================
// SPLIT MANAGEMENT
// ========================================

// CreateSplit creates a new payment split for a ride
func (s *Service) CreateSplit(ctx context.Context, initiatorID uuid.UUID, req *CreateSplitRequest) (*SplitResponse, error) {
	// Get ride fare
	totalAmount, currency, err := s.paymentSvc.GetRideFare(ctx, req.RideID)
	if err != nil {
		return nil, common.NewBadRequestError("failed to get ride fare", err)
	}

	// Check for existing split
	existing, _ := s.repo.GetSplitByRideID(ctx, req.RideID)
	if existing != nil {
		return nil, common.NewBadRequestError("split already exists for this ride", nil)
	}

	// Validate participants
	if len(req.Participants) < 1 {
		return nil, common.NewBadRequestError("at least one participant required", nil)
	}

	// Calculate amounts based on split type
	totalParticipants := len(req.Participants) + 1 // +1 for initiator
	participants, err := s.calculateParticipantAmounts(initiatorID, req, totalAmount, totalParticipants)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	split := &PaymentSplit{
		ID:              uuid.New(),
		RideID:          req.RideID,
		InitiatorID:     initiatorID,
		SplitType:       req.SplitType,
		TotalAmount:     totalAmount,
		Currency:        currency,
		CollectedAmount: 0,
		Status:          SplitStatusPending,
		ExpiresAt:       now.Add(24 * time.Hour), // 24-hour expiry
		Note:            req.Note,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Set split ID on all participants
	for _, p := range participants {
		p.SplitID = split.ID
	}

	if err := s.repo.CreateSplit(ctx, split, participants); err != nil {
		return nil, common.NewInternalServerError("failed to create split")
	}

	// Send invitations
	for _, p := range participants {
		if p.UserID != nil && *p.UserID != initiatorID {
			if s.notificationSvc != nil {
				_ = s.notificationSvc.SendSplitInvitation(ctx, p.ID, split.ID, "User", p.Amount)
			}
		}
	}

	logger.Info("Payment split created",
		zap.String("split_id", split.ID.String()),
		zap.String("ride_id", req.RideID.String()),
		zap.Int("participants", len(participants)),
		zap.Float64("total", totalAmount),
	)

	return s.buildSplitResponse(ctx, split, participants, &initiatorID)
}

// GetSplit gets a payment split with details
func (s *Service) GetSplit(ctx context.Context, userID uuid.UUID, splitID uuid.UUID) (*SplitResponse, error) {
	split, err := s.repo.GetSplit(ctx, splitID)
	if err != nil || split == nil {
		return nil, common.NewNotFoundError("split not found", err)
	}

	participants, err := s.repo.GetParticipants(ctx, splitID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get participants")
	}

	// Verify user is a participant or initiator
	isParticipant := split.InitiatorID == userID
	if !isParticipant {
		for _, p := range participants {
			if p.UserID != nil && *p.UserID == userID {
				isParticipant = true
				break
			}
		}
	}
	if !isParticipant {
		return nil, common.NewForbiddenError("not authorized to view this split")
	}

	return s.buildSplitResponse(ctx, split, participants, &userID)
}

// GetSplitByRide gets the split for a specific ride
func (s *Service) GetSplitByRide(ctx context.Context, userID uuid.UUID, rideID uuid.UUID) (*SplitResponse, error) {
	split, err := s.repo.GetSplitByRideID(ctx, rideID)
	if err != nil || split == nil {
		return nil, common.NewNotFoundError("no split found for this ride", err)
	}

	participants, err := s.repo.GetParticipants(ctx, split.ID)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get participants")
	}

	return s.buildSplitResponse(ctx, split, participants, &userID)
}

// RespondToSplit handles a participant's response to a split invitation
func (s *Service) RespondToSplit(ctx context.Context, userID uuid.UUID, splitID uuid.UUID, req *RespondToSplitRequest) error {
	split, err := s.repo.GetSplit(ctx, splitID)
	if err != nil || split == nil {
		return common.NewNotFoundError("split not found", err)
	}

	if split.Status != SplitStatusPending && split.Status != SplitStatusActive {
		return common.NewBadRequestError("split is no longer accepting responses", nil)
	}

	// Find participant
	participant, err := s.repo.GetParticipantByUserID(ctx, splitID, userID)
	if err != nil || participant == nil {
		return common.NewNotFoundError("you are not a participant in this split", err)
	}

	if participant.Status != ParticipantStatusInvited {
		return common.NewBadRequestError("you have already responded to this split", nil)
	}

	if req.Accept {
		if err := s.repo.UpdateParticipantStatus(ctx, participant.ID, ParticipantStatusAccepted); err != nil {
			return common.NewInternalServerError("failed to accept split")
		}

		// Notify initiator
		if s.notificationSvc != nil {
			_ = s.notificationSvc.SendSplitAccepted(ctx, split.InitiatorID, participant.DisplayName)
		}

		// Check if all have accepted
		s.checkAndActivateSplit(ctx, splitID)

		logger.Info("Split invitation accepted",
			zap.String("split_id", splitID.String()),
			zap.String("user_id", userID.String()),
		)
	} else {
		if err := s.repo.UpdateParticipantStatus(ctx, participant.ID, ParticipantStatusDeclined); err != nil {
			return common.NewInternalServerError("failed to decline split")
		}

		// Redistribute the declined amount to remaining participants
		s.redistributeDeclinedAmount(ctx, splitID, participant.Amount)

		logger.Info("Split invitation declined",
			zap.String("split_id", splitID.String()),
			zap.String("user_id", userID.String()),
		)
	}

	return nil
}

// PaySplit processes payment for a participant's share
func (s *Service) PaySplit(ctx context.Context, userID uuid.UUID, splitID uuid.UUID, paymentMethod string) error {
	split, err := s.repo.GetSplit(ctx, splitID)
	if err != nil || split == nil {
		return common.NewNotFoundError("split not found", err)
	}

	participant, err := s.repo.GetParticipantByUserID(ctx, splitID, userID)
	if err != nil || participant == nil {
		return common.NewNotFoundError("you are not a participant in this split", err)
	}

	if participant.Status == ParticipantStatusPaid {
		return common.NewBadRequestError("you have already paid", nil)
	}

	if participant.Status != ParticipantStatusAccepted {
		return common.NewBadRequestError("you must accept the split first", nil)
	}

	// Process payment
	paymentID, err := s.paymentSvc.ProcessSplitPayment(ctx, userID, split.RideID, participant.Amount, paymentMethod)
	if err != nil {
		if updateErr := s.repo.UpdateParticipantStatus(ctx, participant.ID, ParticipantStatusFailed); updateErr != nil {
			logger.Error("failed to update participant status", zap.Error(updateErr))
		}
		return common.NewInternalServerError("payment processing failed")
	}

	// Update participant
	if err := s.repo.UpdateParticipantPayment(ctx, participant.ID, paymentID, paymentMethod); err != nil {
		return common.NewInternalServerError("failed to update payment record")
	}

	// Update collected amount
	newCollected := split.CollectedAmount + participant.Amount
	if err := s.repo.UpdateCollectedAmount(ctx, split.ID, newCollected); err != nil {
		logger.Error("failed to update collected amount", zap.Error(err))
	}

	// Check if all payments collected
	s.checkAndCompleteSplit(ctx, splitID)

	logger.Info("Split payment processed",
		zap.String("split_id", splitID.String()),
		zap.String("user_id", userID.String()),
		zap.Float64("amount", participant.Amount),
	)

	return nil
}

// CancelSplit cancels a payment split
func (s *Service) CancelSplit(ctx context.Context, userID uuid.UUID, splitID uuid.UUID) error {
	split, err := s.repo.GetSplit(ctx, splitID)
	if err != nil || split == nil {
		return common.NewNotFoundError("split not found", err)
	}

	if split.InitiatorID != userID {
		return common.NewForbiddenError("only the initiator can cancel the split")
	}

	if split.Status == SplitStatusCompleted {
		return common.NewBadRequestError("cannot cancel a completed split", nil)
	}

	if err := s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusCancelled); err != nil {
		return common.NewInternalServerError("failed to cancel split")
	}

	logger.Info("Payment split cancelled",
		zap.String("split_id", splitID.String()),
		zap.String("initiator_id", userID.String()),
	)

	return nil
}

// ========================================
// SPLIT GROUPS
// ========================================

// CreateGroup creates a saved split group
func (s *Service) CreateGroup(ctx context.Context, ownerID uuid.UUID, req *CreateGroupRequest) (*SplitGroup, error) {
	now := time.Now()
	group := &SplitGroup{
		ID:        uuid.New(),
		OwnerID:   ownerID,
		Name:      req.Name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	for _, m := range req.Members {
		member := SplitGroupMember{
			ID:          uuid.New(),
			GroupID:     group.ID,
			UserID:      m.UserID,
			Phone:       m.Phone,
			DisplayName: m.DisplayName,
			CreatedAt:   now,
		}
		group.Members = append(group.Members, member)
	}

	if err := s.repo.CreateGroup(ctx, group); err != nil {
		return nil, common.NewInternalServerError("failed to create group")
	}

	return group, nil
}

// ListGroups lists saved split groups for a user
func (s *Service) ListGroups(ctx context.Context, ownerID uuid.UUID) ([]*SplitGroup, error) {
	return s.repo.GetGroupsByOwner(ctx, ownerID)
}

// DeleteGroup deletes a saved split group
func (s *Service) DeleteGroup(ctx context.Context, ownerID uuid.UUID, groupID uuid.UUID) error {
	return s.repo.DeleteGroup(ctx, groupID, ownerID)
}

// ========================================
// HISTORY & STATS
// ========================================

// GetHistory gets split payment history for a user
func (s *Service) GetHistory(ctx context.Context, userID uuid.UUID, limit, offset int) (*SplitHistory, error) {
	if limit == 0 {
		limit = 20
	}

	initiated, err := s.repo.GetSplitsByInitiator(ctx, userID, limit, offset)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get initiated splits")
	}

	received, err := s.repo.GetSplitsForParticipant(ctx, userID, limit, offset)
	if err != nil {
		return nil, common.NewInternalServerError("failed to get received splits")
	}

	stats, _ := s.repo.GetSplitStats(ctx, userID)

	// Build responses
	var initiatedResponses []*SplitResponse
	for _, split := range initiated {
		participants, _ := s.repo.GetParticipants(ctx, split.ID)
		resp, _ := s.buildSplitResponse(ctx, split, participants, &userID)
		if resp != nil {
			initiatedResponses = append(initiatedResponses, resp)
		}
	}

	var receivedResponses []*SplitResponse
	for _, split := range received {
		participants, _ := s.repo.GetParticipants(ctx, split.ID)
		resp, _ := s.buildSplitResponse(ctx, split, participants, &userID)
		if resp != nil {
			receivedResponses = append(receivedResponses, resp)
		}
	}

	return &SplitHistory{
		Initiated: initiatedResponses,
		Received:  receivedResponses,
		Stats:     *stats,
	}, nil
}

// ========================================
// BACKGROUND WORKERS
// ========================================

// ProcessExpiredSplits marks expired splits
func (s *Service) ProcessExpiredSplits(ctx context.Context) error {
	expired, err := s.repo.GetExpiredSplits(ctx)
	if err != nil {
		return err
	}

	for _, split := range expired {
		// If partially collected, mark as partial
		if split.CollectedAmount > 0 {
			_ = s.repo.UpdateSplitStatus(ctx, split.ID, SplitStatusPartial)
		} else {
			_ = s.repo.UpdateSplitStatus(ctx, split.ID, SplitStatusExpired)
		}

		logger.Info("Split expired",
			zap.String("split_id", split.ID.String()),
			zap.Float64("collected", split.CollectedAmount),
			zap.Float64("total", split.TotalAmount),
		)
	}

	return nil
}

// SendPendingReminders sends reminders to participants who haven't paid
func (s *Service) SendPendingReminders(ctx context.Context) error {
	pendingParticipants, err := s.repo.GetPendingReminders(ctx)
	if err != nil {
		return err
	}

	for _, p := range pendingParticipants {
		if s.notificationSvc != nil {
			_ = s.notificationSvc.SendSplitReminder(ctx, p.ID, p.SplitID, p.Amount)
		}
		_ = s.repo.UpdateParticipantReminder(ctx, p.ID)
	}

	return nil
}

// ========================================
// HELPER FUNCTIONS
// ========================================

func (s *Service) calculateParticipantAmounts(initiatorID uuid.UUID, req *CreateSplitRequest, totalAmount float64, totalParticipants int) ([]*SplitParticipant, error) {
	now := time.Now()
	var participants []*SplitParticipant

	switch req.SplitType {
	case SplitTypeEqual:
		// Divide equally among all (including initiator)
		perPerson := math.Floor(totalAmount/float64(totalParticipants)*100) / 100
		remainder := totalAmount - perPerson*float64(totalParticipants)

		// Initiator's share (gets remainder for rounding)
		initiatorAmount := perPerson + remainder
		participants = append(participants, &SplitParticipant{
			ID:          uuid.New(),
			UserID:      &initiatorID,
			DisplayName: "You",
			Amount:      initiatorAmount,
			Status:      ParticipantStatusAccepted, // Initiator auto-accepts
			InviteSentAt: &now,
			RespondedAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		})

		// Other participants
		for _, p := range req.Participants {
			participants = append(participants, &SplitParticipant{
				ID:           uuid.New(),
				UserID:       p.UserID,
				Phone:        p.Phone,
				Email:        p.Email,
				DisplayName:  p.DisplayName,
				Amount:       perPerson,
				Status:       ParticipantStatusInvited,
				InviteSentAt: &now,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}

	case SplitTypeCustom:
		// Validate custom amounts sum to total
		var customTotal float64
		for _, p := range req.Participants {
			if p.Amount == nil {
				return nil, common.NewBadRequestError("amount required for custom split", nil)
			}
			customTotal += *p.Amount
		}

		// Initiator pays the remainder
		initiatorAmount := totalAmount - customTotal
		if initiatorAmount < 0 {
			return nil, common.NewBadRequestError("participant amounts exceed total fare", nil)
		}

		participants = append(participants, &SplitParticipant{
			ID:          uuid.New(),
			UserID:      &initiatorID,
			DisplayName: "You",
			Amount:      initiatorAmount,
			Status:      ParticipantStatusAccepted,
			InviteSentAt: &now,
			RespondedAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		})

		for _, p := range req.Participants {
			participants = append(participants, &SplitParticipant{
				ID:           uuid.New(),
				UserID:       p.UserID,
				Phone:        p.Phone,
				Email:        p.Email,
				DisplayName:  p.DisplayName,
				Amount:       *p.Amount,
				Status:       ParticipantStatusInvited,
				InviteSentAt: &now,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}

	case SplitTypePercentage:
		// Validate percentages sum correctly
		var totalPct float64
		for _, p := range req.Participants {
			if p.Percentage == nil {
				return nil, common.NewBadRequestError("percentage required for percentage split", nil)
			}
			totalPct += *p.Percentage
		}

		if totalPct > 100 {
			return nil, common.NewBadRequestError("percentages exceed 100%", nil)
		}

		// Initiator gets remaining percentage
		initiatorPct := 100 - totalPct
		initiatorAmount := math.Floor(totalAmount*initiatorPct) / 100

		participants = append(participants, &SplitParticipant{
			ID:          uuid.New(),
			UserID:      &initiatorID,
			DisplayName: "You",
			Amount:      initiatorAmount,
			Percentage:  &initiatorPct,
			Status:      ParticipantStatusAccepted,
			InviteSentAt: &now,
			RespondedAt: &now,
			CreatedAt:   now,
			UpdatedAt:   now,
		})

		for _, p := range req.Participants {
			amount := math.Floor(totalAmount*(*p.Percentage)) / 100
			pct := *p.Percentage
			participants = append(participants, &SplitParticipant{
				ID:           uuid.New(),
				UserID:       p.UserID,
				Phone:        p.Phone,
				Email:        p.Email,
				DisplayName:  p.DisplayName,
				Amount:       amount,
				Percentage:   &pct,
				Status:       ParticipantStatusInvited,
				InviteSentAt: &now,
				CreatedAt:    now,
				UpdatedAt:    now,
			})
		}

	default:
		return nil, common.NewBadRequestError("invalid split type", nil)
	}

	return participants, nil
}

func (s *Service) checkAndActivateSplit(ctx context.Context, splitID uuid.UUID) {
	participants, err := s.repo.GetParticipants(ctx, splitID)
	if err != nil {
		return
	}

	allAccepted := true
	for _, p := range participants {
		if p.Status == ParticipantStatusInvited {
			allAccepted = false
			break
		}
	}

	if allAccepted {
		_ = s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusActive)
	}
}

func (s *Service) checkAndCompleteSplit(ctx context.Context, splitID uuid.UUID) {
	participants, err := s.repo.GetParticipants(ctx, splitID)
	if err != nil {
		return
	}

	allPaid := true
	for _, p := range participants {
		if p.Status != ParticipantStatusPaid && p.Status != ParticipantStatusDeclined {
			allPaid = false
			break
		}
	}

	if allPaid {
		_ = s.repo.UpdateSplitStatus(ctx, splitID, SplitStatusCompleted)
		if s.notificationSvc != nil {
			_ = s.notificationSvc.SendSplitCompleted(ctx, splitID)
		}
	}
}

func (s *Service) redistributeDeclinedAmount(ctx context.Context, splitID uuid.UUID, declinedAmount float64) {
	participants, err := s.repo.GetParticipants(ctx, splitID)
	if err != nil {
		return
	}

	// Count active participants (not declined/failed)
	var activeCount int
	for _, p := range participants {
		if p.Status != ParticipantStatusDeclined && p.Status != ParticipantStatusFailed {
			activeCount++
		}
	}

	if activeCount == 0 {
		return
	}

	// Add declined amount equally to remaining participants
	extra := declinedAmount / float64(activeCount)
	_ = extra // In production, update each participant's amount
	// This is simplified - a full implementation would update the DB
}

func (s *Service) buildSplitResponse(ctx context.Context, split *PaymentSplit, participants []*SplitParticipant, currentUserID *uuid.UUID) (*SplitResponse, error) {
	summary := &SplitSummary{
		TotalParticipants: len(participants),
		CollectedAmount:   split.CollectedAmount,
		RemainingAmount:   split.TotalAmount - split.CollectedAmount,
	}

	var mySplit *SplitParticipant

	for _, p := range participants {
		switch p.Status {
		case ParticipantStatusAccepted:
			summary.AcceptedCount++
		case ParticipantStatusPaid:
			summary.PaidCount++
		case ParticipantStatusDeclined:
			summary.DeclinedCount++
		case ParticipantStatusInvited:
			summary.PendingCount++
		}

		if currentUserID != nil && p.UserID != nil && *p.UserID == *currentUserID {
			mySplit = p
		}
	}

	summary.AllAccepted = summary.PendingCount == 0
	summary.AllPaid = summary.PaidCount == summary.TotalParticipants

	// Convert to non-pointer slice
	participantSlice := make([]SplitParticipant, len(participants))
	for i, p := range participants {
		participantSlice[i] = *p
	}

	return &SplitResponse{
		Split:        split,
		Participants: participantSlice,
		MySplit:      mySplit,
		Summary:      summary,
	}, nil
}
