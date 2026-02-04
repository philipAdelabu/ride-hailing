package paymentsplit

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCalculateParticipantAmounts_EqualSplit(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	tests := []struct {
		name              string
		totalAmount       float64
		participantCount  int
		totalParticipants int // includes initiator
	}{
		{
			name:              "2-way split of $20",
			totalAmount:       20.00,
			participantCount:  1,
			totalParticipants: 2,
		},
		{
			name:              "3-way split of $30",
			totalAmount:       30.00,
			participantCount:  2,
			totalParticipants: 3,
		},
		{
			name:              "4-way split of $100",
			totalAmount:       100.00,
			participantCount:  3,
			totalParticipants: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			participants := make([]ParticipantInput, tt.participantCount)
			for i := range participants {
				participants[i] = ParticipantInput{
					DisplayName: "Participant",
				}
			}

			req := &CreateSplitRequest{
				SplitType:    SplitTypeEqual,
				Participants: participants,
			}

			result, err := svc.calculateParticipantAmounts(initiatorID, req, tt.totalAmount, tt.totalParticipants)
			assert.NoError(t, err)
			assert.Len(t, result, tt.totalParticipants)

			// Sum of all amounts should equal total
			var sum float64
			for _, p := range result {
				sum += p.Amount
			}
			assert.InDelta(t, tt.totalAmount, sum, 0.01,
				"sum of amounts (%.2f) should equal total (%.2f)", sum, tt.totalAmount)

			// Initiator should be first with accepted status
			assert.Equal(t, &initiatorID, result[0].UserID)
			assert.Equal(t, ParticipantStatusAccepted, result[0].Status)

			// Other participants should be invited
			for i := 1; i < len(result); i++ {
				assert.Equal(t, ParticipantStatusInvited, result[i].Status)
			}
		})
	}
}

func TestCalculateParticipantAmounts_EqualSplit_Remainder(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	// $10 split 3 ways: $3.33, $3.33, $3.33 = $9.99, remainder $0.01 goes to initiator
	req := &CreateSplitRequest{
		SplitType: SplitTypeEqual,
		Participants: []ParticipantInput{
			{DisplayName: "Alice"},
			{DisplayName: "Bob"},
		},
	}

	result, err := svc.calculateParticipantAmounts(initiatorID, req, 10.00, 3)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Per person is floor(10.00 / 3 * 100) / 100 = floor(333.33) / 100 = 3.33
	// Remainder = 10.00 - 3.33 * 3 = 10.00 - 9.99 = 0.01
	// Initiator gets 3.33 + 0.01 = 3.34

	assert.Equal(t, 3.34, result[0].Amount, "initiator should get per-person + remainder")
	assert.Equal(t, 3.33, result[1].Amount, "participant should get per-person amount")
	assert.Equal(t, 3.33, result[2].Amount, "participant should get per-person amount")

	// Total should be exact
	total := result[0].Amount + result[1].Amount + result[2].Amount
	assert.InDelta(t, 10.00, total, 0.001)
}

func TestCalculateParticipantAmounts_CustomSplit(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	amount1 := 15.00
	amount2 := 10.00

	req := &CreateSplitRequest{
		SplitType: SplitTypeCustom,
		Participants: []ParticipantInput{
			{DisplayName: "Alice", Amount: &amount1},
			{DisplayName: "Bob", Amount: &amount2},
		},
	}

	result, err := svc.calculateParticipantAmounts(initiatorID, req, 50.00, 3)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Initiator pays remainder: 50 - 15 - 10 = 25
	assert.Equal(t, 25.00, result[0].Amount, "initiator pays the remainder")
	assert.Equal(t, 15.00, result[1].Amount)
	assert.Equal(t, 10.00, result[2].Amount)
}

func TestCalculateParticipantAmounts_CustomSplit_AmountsExceedTotal(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	amount1 := 30.00
	amount2 := 25.00

	req := &CreateSplitRequest{
		SplitType: SplitTypeCustom,
		Participants: []ParticipantInput{
			{DisplayName: "Alice", Amount: &amount1},
			{DisplayName: "Bob", Amount: &amount2},
		},
	}

	// Total is 40, but participants want 30 + 25 = 55
	_, err := svc.calculateParticipantAmounts(initiatorID, req, 40.00, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed total fare")
}

func TestCalculateParticipantAmounts_CustomSplit_MissingAmount(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	req := &CreateSplitRequest{
		SplitType: SplitTypeCustom,
		Participants: []ParticipantInput{
			{DisplayName: "Alice"}, // No amount set
		},
	}

	_, err := svc.calculateParticipantAmounts(initiatorID, req, 50.00, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "amount required")
}

func TestCalculateParticipantAmounts_PercentageSplit(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	pct1 := 30.0
	pct2 := 20.0

	req := &CreateSplitRequest{
		SplitType: SplitTypePercentage,
		Participants: []ParticipantInput{
			{DisplayName: "Alice", Percentage: &pct1},
			{DisplayName: "Bob", Percentage: &pct2},
		},
	}

	result, err := svc.calculateParticipantAmounts(initiatorID, req, 100.00, 3)
	assert.NoError(t, err)
	assert.Len(t, result, 3)

	// Initiator gets 100 - 30 - 20 = 50%
	assert.NotNil(t, result[0].Percentage)
	assert.Equal(t, 50.0, *result[0].Percentage)

	// Check that percentages are assigned
	assert.NotNil(t, result[1].Percentage)
	assert.Equal(t, 30.0, *result[1].Percentage)
	assert.NotNil(t, result[2].Percentage)
	assert.Equal(t, 20.0, *result[2].Percentage)
}

func TestCalculateParticipantAmounts_PercentageSplit_ExceedsHundred(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	pct1 := 60.0
	pct2 := 50.0

	req := &CreateSplitRequest{
		SplitType: SplitTypePercentage,
		Participants: []ParticipantInput{
			{DisplayName: "Alice", Percentage: &pct1},
			{DisplayName: "Bob", Percentage: &pct2},
		},
	}

	_, err := svc.calculateParticipantAmounts(initiatorID, req, 100.00, 3)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceed 100%")
}

func TestCalculateParticipantAmounts_PercentageSplit_MissingPercentage(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	req := &CreateSplitRequest{
		SplitType: SplitTypePercentage,
		Participants: []ParticipantInput{
			{DisplayName: "Alice"}, // No percentage set
		},
	}

	_, err := svc.calculateParticipantAmounts(initiatorID, req, 100.00, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "percentage required")
}

func TestCalculateParticipantAmounts_InvalidSplitType(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	req := &CreateSplitRequest{
		SplitType: SplitType("invalid"),
		Participants: []ParticipantInput{
			{DisplayName: "Alice"},
		},
	}

	_, err := svc.calculateParticipantAmounts(initiatorID, req, 100.00, 2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid split type")
}

func TestCalculateParticipantAmounts_EqualSplit_TwoWay(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	req := &CreateSplitRequest{
		SplitType: SplitTypeEqual,
		Participants: []ParticipantInput{
			{DisplayName: "Friend"},
		},
	}

	result, err := svc.calculateParticipantAmounts(initiatorID, req, 25.00, 2)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// 25.00 / 2 = 12.50 each, no remainder
	assert.Equal(t, 12.50, result[0].Amount)
	assert.Equal(t, 12.50, result[1].Amount)
}

func TestCalculateParticipantAmounts_EqualSplit_LargeAmount(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	participants := make([]ParticipantInput, 6)
	for i := range participants {
		participants[i] = ParticipantInput{DisplayName: "P"}
	}

	req := &CreateSplitRequest{
		SplitType:    SplitTypeEqual,
		Participants: participants,
	}

	// 100.00 / 7 = 14.285714... => floor to 14.28 per person
	// Remainder = 100.00 - 14.28 * 7 = 100.00 - 99.96 = 0.04
	result, err := svc.calculateParticipantAmounts(initiatorID, req, 100.00, 7)
	assert.NoError(t, err)

	var sum float64
	for _, p := range result {
		sum += p.Amount
	}
	assert.InDelta(t, 100.00, sum, 0.01, "amounts must sum to total")
}

func TestCalculateParticipantAmounts_CustomSplit_InitiatorPaysNothing(t *testing.T) {
	svc := &Service{}
	initiatorID := uuid.New()

	amount := 50.00

	req := &CreateSplitRequest{
		SplitType: SplitTypeCustom,
		Participants: []ParticipantInput{
			{DisplayName: "Rich Friend", Amount: &amount},
		},
	}

	result, err := svc.calculateParticipantAmounts(initiatorID, req, 50.00, 2)
	assert.NoError(t, err)

	// Initiator pays 50 - 50 = 0
	assert.Equal(t, 0.0, result[0].Amount)
	assert.Equal(t, 50.0, result[1].Amount)
}

func TestBuildSplitResponse(t *testing.T) {
	svc := &Service{}

	userID := uuid.New()
	otherID := uuid.New()

	split := &PaymentSplit{
		ID:              uuid.New(),
		TotalAmount:     100.00,
		CollectedAmount: 25.00,
	}

	participants := []*SplitParticipant{
		{ID: uuid.New(), UserID: &userID, Amount: 50.0, Status: ParticipantStatusAccepted},
		{ID: uuid.New(), UserID: &otherID, Amount: 25.0, Status: ParticipantStatusPaid},
		{ID: uuid.New(), Amount: 25.0, Status: ParticipantStatusInvited},
	}

	resp, err := svc.buildSplitResponse(nil, split, participants, &userID)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Check summary
	assert.Equal(t, 3, resp.Summary.TotalParticipants)
	assert.Equal(t, 1, resp.Summary.AcceptedCount)
	assert.Equal(t, 1, resp.Summary.PaidCount)
	assert.Equal(t, 1, resp.Summary.PendingCount)
	assert.Equal(t, 0, resp.Summary.DeclinedCount)
	assert.Equal(t, 25.00, resp.Summary.CollectedAmount)
	assert.Equal(t, 75.00, resp.Summary.RemainingAmount)
	assert.False(t, resp.Summary.AllAccepted)
	assert.False(t, resp.Summary.AllPaid)

	// Check my split
	assert.NotNil(t, resp.MySplit)
	assert.Equal(t, 50.0, resp.MySplit.Amount)
}

func TestBuildSplitResponse_AllPaid(t *testing.T) {
	svc := &Service{}

	userID := uuid.New()

	split := &PaymentSplit{
		ID:              uuid.New(),
		TotalAmount:     100.00,
		CollectedAmount: 100.00,
	}

	participants := []*SplitParticipant{
		{ID: uuid.New(), UserID: &userID, Amount: 50.0, Status: ParticipantStatusPaid},
		{ID: uuid.New(), Amount: 50.0, Status: ParticipantStatusPaid},
	}

	resp, err := svc.buildSplitResponse(nil, split, participants, &userID)
	assert.NoError(t, err)

	assert.True(t, resp.Summary.AllAccepted)
	assert.True(t, resp.Summary.AllPaid)
	assert.Equal(t, 0.0, resp.Summary.RemainingAmount)
}

func TestBuildSplitResponse_NoCurrentUser(t *testing.T) {
	svc := &Service{}

	split := &PaymentSplit{
		ID:              uuid.New(),
		TotalAmount:     100.00,
		CollectedAmount: 0.00,
	}

	participants := []*SplitParticipant{
		{ID: uuid.New(), Amount: 50.0, Status: ParticipantStatusInvited},
		{ID: uuid.New(), Amount: 50.0, Status: ParticipantStatusInvited},
	}

	resp, err := svc.buildSplitResponse(nil, split, participants, nil)
	assert.NoError(t, err)
	assert.Nil(t, resp.MySplit)
}
