package ratings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSuggestedTags_Rider(t *testing.T) {
	svc := &Service{}
	tags := svc.GetSuggestedTags(RaterTypeRider)

	assert.Len(t, tags, 8)

	expectedTags := []RatingTag{
		TagGreatConversation, TagSmoothDriving, TagCleanCar,
		TagKnowsRoute, TagFriendly, TagProfessional,
		TagGoodMusic, TagSafeDriver,
	}

	for i, tag := range tags {
		assert.Equal(t, expectedTags[i], tag)
	}
}

func TestGetSuggestedTags_Driver(t *testing.T) {
	svc := &Service{}
	tags := svc.GetSuggestedTags(RaterTypeDriver)

	assert.Len(t, tags, 4)

	expectedTags := []RatingTag{
		TagPoliteRider, TagOnTime, TagRespectful, TagGoodDirections,
	}

	for i, tag := range tags {
		assert.Equal(t, expectedTags[i], tag)
	}
}

func TestGetSuggestedTags_RiderHasMoreTags(t *testing.T) {
	svc := &Service{}
	riderTags := svc.GetSuggestedTags(RaterTypeRider)
	driverTags := svc.GetSuggestedTags(RaterTypeDriver)

	assert.Greater(t, len(riderTags), len(driverTags),
		"riders should have more tags to choose from than drivers")
}

func TestGetSuggestedTags_NoDuplicates(t *testing.T) {
	svc := &Service{}

	for _, raterType := range []RaterType{RaterTypeRider, RaterTypeDriver} {
		t.Run(string(raterType), func(t *testing.T) {
			tags := svc.GetSuggestedTags(raterType)
			seen := make(map[RatingTag]bool)
			for _, tag := range tags {
				assert.False(t, seen[tag], "duplicate tag found: %s", tag)
				seen[tag] = true
			}
		})
	}
}

func TestRatingConstants(t *testing.T) {
	// Verify rater types
	assert.Equal(t, RaterType("rider"), RaterTypeRider)
	assert.Equal(t, RaterType("driver"), RaterTypeDriver)

	// Verify all tag constants are non-empty strings
	allTags := []RatingTag{
		TagGreatConversation, TagSmoothDriving, TagCleanCar,
		TagKnowsRoute, TagFriendly, TagProfessional,
		TagGoodMusic, TagSafeDriver,
		TagRoughDriving, TagDirtyCar, TagRude, TagUnsafe,
		TagLostRoute, TagPhoneUse, TagLateArrival,
		TagPoliteRider, TagOnTime, TagRespectful, TagGoodDirections,
		TagRudeRider, TagMessyRider, TagLatePickup, TagSlammedDoor,
	}

	for _, tag := range allTags {
		assert.NotEmpty(t, string(tag))
	}
}
