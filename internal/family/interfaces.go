package family

import (
	"context"

	"github.com/google/uuid"
)

// RepositoryInterface defines the contract for family repository operations
type RepositoryInterface interface {
	// Family account operations
	CreateFamily(ctx context.Context, family *FamilyAccount) error
	GetFamilyByID(ctx context.Context, id uuid.UUID) (*FamilyAccount, error)
	GetFamilyByOwner(ctx context.Context, ownerID uuid.UUID) (*FamilyAccount, error)
	UpdateFamily(ctx context.Context, family *FamilyAccount) error
	IncrementSpend(ctx context.Context, familyID uuid.UUID, amount float64) error
	ResetMonthlySpend(ctx context.Context, dayOfMonth int) error

	// Member operations
	AddMember(ctx context.Context, member *FamilyMember) error
	GetMembersByFamily(ctx context.Context, familyID uuid.UUID) ([]FamilyMember, error)
	GetMemberByUserID(ctx context.Context, familyID, userID uuid.UUID) (*FamilyMember, error)
	GetFamilyForUser(ctx context.Context, userID uuid.UUID) (*FamilyAccount, *FamilyMember, error)
	UpdateMember(ctx context.Context, member *FamilyMember) error
	RemoveMember(ctx context.Context, memberID uuid.UUID) error
	IncrementMemberSpend(ctx context.Context, memberID uuid.UUID, amount float64) error

	// Invite operations
	CreateInvite(ctx context.Context, invite *FamilyInvite) error
	GetInviteByID(ctx context.Context, id uuid.UUID) (*FamilyInvite, error)
	GetPendingInvitesForUser(ctx context.Context, userID uuid.UUID, email *string) ([]FamilyInvite, error)
	UpdateInviteStatus(ctx context.Context, inviteID uuid.UUID, status InviteStatus) error

	// Ride log operations
	LogRide(ctx context.Context, log *FamilyRideLog) error
	GetSpendingReport(ctx context.Context, familyID uuid.UUID) ([]MemberSpend, int, float64, error)
}
