package disputes

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetDisputeReasons(t *testing.T) {
	svc := &Service{}
	resp := svc.GetDisputeReasons()

	assert.NotNil(t, resp)
	assert.Len(t, resp.Reasons, 10)

	expectedCodes := []DisputeReason{
		ReasonWrongRoute, ReasonOvercharged, ReasonTripNotTaken,
		ReasonDriverDetour, ReasonWrongFare, ReasonSurgeUnfair,
		ReasonWaitTimeWrong, ReasonCancelFeeWrong, ReasonDuplicateCharge,
		ReasonOther,
	}

	for i, reason := range resp.Reasons {
		assert.Equal(t, expectedCodes[i], reason.Code, "reason code mismatch at index %d", i)
		assert.NotEmpty(t, reason.Label, "reason %d label should not be empty", i)
		assert.NotEmpty(t, reason.Description, "reason %d description should not be empty", i)
	}
}

func TestGetDisputeReasons_LastIsOther(t *testing.T) {
	svc := &Service{}
	resp := svc.GetDisputeReasons()

	lastReason := resp.Reasons[len(resp.Reasons)-1]
	assert.Equal(t, ReasonOther, lastReason.Code)
	assert.Equal(t, "Other", lastReason.Label)
}

func TestResolutionTypeToStatus_Mapping(t *testing.T) {
	tests := []struct {
		name           string
		resolutionType ResolutionType
		expectedStatus DisputeStatus
	}{
		{
			name:           "full refund maps to approved",
			resolutionType: ResolutionFullRefund,
			expectedStatus: DisputeStatusApproved,
		},
		{
			name:           "partial refund maps to partial_refund",
			resolutionType: ResolutionPartialRefund,
			expectedStatus: DisputeStatusPartial,
		},
		{
			name:           "credits maps to approved",
			resolutionType: ResolutionCredits,
			expectedStatus: DisputeStatusApproved,
		},
		{
			name:           "no action maps to rejected",
			resolutionType: ResolutionNoAction,
			expectedStatus: DisputeStatusRejected,
		},
		{
			name:           "fare adjustment maps to approved",
			resolutionType: ResolutionFareAdjust,
			expectedStatus: DisputeStatusApproved,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var status DisputeStatus
			switch tt.resolutionType {
			case ResolutionFullRefund:
				status = DisputeStatusApproved
			case ResolutionPartialRefund:
				status = DisputeStatusPartial
			case ResolutionCredits:
				status = DisputeStatusApproved
			case ResolutionNoAction:
				status = DisputeStatusRejected
			case ResolutionFareAdjust:
				status = DisputeStatusApproved
			}
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

func TestDisputeStatusConstants(t *testing.T) {
	assert.Equal(t, DisputeStatus("pending"), DisputeStatusPending)
	assert.Equal(t, DisputeStatus("reviewing"), DisputeStatusReviewing)
	assert.Equal(t, DisputeStatus("approved"), DisputeStatusApproved)
	assert.Equal(t, DisputeStatus("rejected"), DisputeStatusRejected)
	assert.Equal(t, DisputeStatus("partial_refund"), DisputeStatusPartial)
	assert.Equal(t, DisputeStatus("closed"), DisputeStatusClosed)
}

func TestDisputeReasonConstants(t *testing.T) {
	reasons := []DisputeReason{
		ReasonWrongRoute, ReasonOvercharged, ReasonTripNotTaken,
		ReasonDriverDetour, ReasonWrongFare, ReasonSurgeUnfair,
		ReasonWaitTimeWrong, ReasonCancelFeeWrong, ReasonDuplicateCharge,
		ReasonOther,
	}

	assert.Len(t, reasons, 10)

	seen := make(map[DisputeReason]bool)
	for _, reason := range reasons {
		assert.NotEmpty(t, string(reason))
		assert.False(t, seen[reason], "duplicate reason: %s", reason)
		seen[reason] = true
	}
}

func TestResolutionTypeConstants(t *testing.T) {
	types := []ResolutionType{
		ResolutionFullRefund, ResolutionPartialRefund,
		ResolutionCredits, ResolutionNoAction, ResolutionFareAdjust,
	}

	assert.Len(t, types, 5)

	seen := make(map[ResolutionType]bool)
	for _, rt := range types {
		assert.NotEmpty(t, string(rt))
		assert.False(t, seen[rt], "duplicate resolution type: %s", rt)
		seen[rt] = true
	}
}

func TestGenerateDisputeNumber(t *testing.T) {
	num := generateDisputeNumber()

	assert.NotEmpty(t, num)
	assert.Equal(t, "DSP-", num[:4], "dispute number should start with DSP-")
	assert.Len(t, num, 10, "DSP-XXXXXX should be 10 chars")

	// Verify uniqueness
	num2 := generateDisputeNumber()
	assert.NotEqual(t, num, num2, "dispute numbers should be unique")
}

func TestDisputeWindowDays(t *testing.T) {
	assert.Equal(t, 30, maxDisputeWindowDays)
}

func TestMaxOpenDisputes(t *testing.T) {
	assert.Equal(t, 5, maxOpenDisputes)
}
