package scheduling

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseTime(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name           string
		timeStr        string
		expectedHour   int
		expectedMinute int
	}{
		{"morning", "08:30", 8, 30},
		{"midnight", "00:00", 0, 0},
		{"noon", "12:00", 12, 0},
		{"evening", "18:45", 18, 45},
		{"end of day", "23:59", 23, 59},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hour, minute := svc.parseTime(tt.timeStr)
			assert.Equal(t, tt.expectedHour, hour)
			assert.Equal(t, tt.expectedMinute, minute)
		})
	}
}

func TestParsePickupTime(t *testing.T) {
	svc := &Service{}

	date := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		timeStr  string
		timezone string
		expected time.Time
	}{
		{
			name:     "morning UTC",
			timeStr:  "08:30",
			timezone: "UTC",
			expected: time.Date(2024, 6, 15, 8, 30, 0, 0, time.UTC),
		},
		{
			name:     "evening UTC",
			timeStr:  "18:00",
			timezone: "UTC",
			expected: time.Date(2024, 6, 15, 18, 0, 0, 0, time.UTC),
		},
		{
			name:     "invalid timezone falls back to UTC",
			timeStr:  "09:00",
			timezone: "Invalid/Zone",
			expected: time.Date(2024, 6, 15, 9, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.parsePickupTime(date, tt.timeStr, tt.timezone)
			assert.Equal(t, tt.expected.Hour(), result.Hour())
			assert.Equal(t, tt.expected.Minute(), result.Minute())
			assert.Equal(t, tt.expected.Year(), result.Year())
			assert.Equal(t, tt.expected.Month(), result.Month())
			assert.Equal(t, tt.expected.Day(), result.Day())
		})
	}
}

func TestMatchesPattern_Daily(t *testing.T) {
	svc := &Service{}

	// Daily should match every day
	for day := 0; day < 7; day++ {
		// June 9, 2024 is Sunday (day 0)
		date := time.Date(2024, 6, 9+day, 8, 0, 0, 0, time.UTC)
		assert.True(t, svc.matchesPattern(date, RecurrenceDaily, nil),
			"daily pattern should match %s", date.Weekday())
	}
}

func TestMatchesPattern_Weekdays(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name    string
		date    time.Time
		matches bool
	}{
		{"Monday", time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC), true},
		{"Tuesday", time.Date(2024, 6, 11, 8, 0, 0, 0, time.UTC), true},
		{"Wednesday", time.Date(2024, 6, 12, 8, 0, 0, 0, time.UTC), true},
		{"Thursday", time.Date(2024, 6, 13, 8, 0, 0, 0, time.UTC), true},
		{"Friday", time.Date(2024, 6, 14, 8, 0, 0, 0, time.UTC), true},
		{"Saturday", time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC), false},
		{"Sunday", time.Date(2024, 6, 9, 8, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.matchesPattern(tt.date, RecurrenceWeekdays, nil)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestMatchesPattern_Weekends(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name    string
		date    time.Time
		matches bool
	}{
		{"Monday", time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC), false},
		{"Friday", time.Date(2024, 6, 14, 8, 0, 0, 0, time.UTC), false},
		{"Saturday", time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC), true},
		{"Sunday", time.Date(2024, 6, 9, 8, 0, 0, 0, time.UTC), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.matchesPattern(tt.date, RecurrenceWeekends, nil)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestMatchesPattern_Weekly(t *testing.T) {
	svc := &Service{}

	// Match on Monday and Wednesday (1, 3)
	daysOfWeek := []int{1, 3}

	tests := []struct {
		name    string
		date    time.Time
		matches bool
	}{
		{"Monday", time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC), true},
		{"Tuesday", time.Date(2024, 6, 11, 8, 0, 0, 0, time.UTC), false},
		{"Wednesday", time.Date(2024, 6, 12, 8, 0, 0, 0, time.UTC), true},
		{"Thursday", time.Date(2024, 6, 13, 8, 0, 0, 0, time.UTC), false},
		{"Friday", time.Date(2024, 6, 14, 8, 0, 0, 0, time.UTC), false},
		{"Saturday", time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC), false},
		{"Sunday", time.Date(2024, 6, 9, 8, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.matchesPattern(tt.date, RecurrenceWeekly, daysOfWeek)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestMatchesPattern_Custom(t *testing.T) {
	svc := &Service{}

	// Custom: Tuesday, Thursday, Saturday (2, 4, 6)
	daysOfWeek := []int{2, 4, 6}

	tests := []struct {
		name    string
		date    time.Time
		matches bool
	}{
		{"Monday", time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC), false},
		{"Tuesday", time.Date(2024, 6, 11, 8, 0, 0, 0, time.UTC), true},
		{"Wednesday", time.Date(2024, 6, 12, 8, 0, 0, 0, time.UTC), false},
		{"Thursday", time.Date(2024, 6, 13, 8, 0, 0, 0, time.UTC), true},
		{"Friday", time.Date(2024, 6, 14, 8, 0, 0, 0, time.UTC), false},
		{"Saturday", time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC), true},
		{"Sunday", time.Date(2024, 6, 9, 8, 0, 0, 0, time.UTC), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.matchesPattern(tt.date, RecurrenceCustom, daysOfWeek)
			assert.Equal(t, tt.matches, result)
		})
	}
}

func TestMatchesPattern_EmptyDaysOfWeek(t *testing.T) {
	svc := &Service{}

	date := time.Date(2024, 6, 10, 8, 0, 0, 0, time.UTC) // Monday
	// Weekly with no days specified should not match
	assert.False(t, svc.matchesPattern(date, RecurrenceWeekly, nil))
	assert.False(t, svc.matchesPattern(date, RecurrenceCustom, []int{}))
}

func TestGenerateScheduleDates_Daily(t *testing.T) {
	svc := &Service{}

	start := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC) // Monday
	dates := svc.generateScheduleDates(start, nil, RecurrenceDaily, nil, 5)

	assert.Len(t, dates, 5)
	// Should be consecutive days
	for i := 0; i < len(dates); i++ {
		expected := start.AddDate(0, 0, i)
		assert.Equal(t, expected.Year(), dates[i].Year())
		assert.Equal(t, expected.Month(), dates[i].Month())
		assert.Equal(t, expected.Day(), dates[i].Day())
	}
}

func TestGenerateScheduleDates_Weekdays(t *testing.T) {
	svc := &Service{}

	start := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC) // Monday
	dates := svc.generateScheduleDates(start, nil, RecurrenceWeekdays, nil, 5)

	assert.Len(t, dates, 5)
	// All should be weekdays
	for _, d := range dates {
		weekday := d.Weekday()
		assert.True(t, weekday >= time.Monday && weekday <= time.Friday,
			"%s should be a weekday", d.Weekday())
	}
}

func TestGenerateScheduleDates_WithEndDate(t *testing.T) {
	svc := &Service{}

	start := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC) // Monday
	end := time.Date(2024, 6, 14, 0, 0, 0, 0, time.UTC)    // Friday

	dates := svc.generateScheduleDates(start, &end, RecurrenceDaily, nil, 100)

	// Should have at most 4 days (Mon-Thu, since Fri is end boundary)
	assert.LessOrEqual(t, len(dates), 4)

	// All dates should be before end date
	for _, d := range dates {
		assert.True(t, d.Before(end),
			"%s should be before end date %s", d.Format("2006-01-02"), end.Format("2006-01-02"))
	}
}

func TestGenerateScheduleDates_MaxOccurrences(t *testing.T) {
	svc := &Service{}

	start := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC)
	dates := svc.generateScheduleDates(start, nil, RecurrenceDaily, nil, 3)

	assert.Len(t, dates, 3)
}

func TestGenerateScheduleDates_WeeklySpecificDays(t *testing.T) {
	svc := &Service{}

	start := time.Date(2024, 6, 10, 0, 0, 0, 0, time.UTC) // Monday
	daysOfWeek := []int{1, 3, 5}                            // Mon, Wed, Fri

	dates := svc.generateScheduleDates(start, nil, RecurrenceWeekly, daysOfWeek, 6)

	assert.Len(t, dates, 6)
	// Verify they fall on the correct days
	for _, d := range dates {
		weekday := int(d.Weekday())
		assert.Contains(t, daysOfWeek, weekday,
			"%s (weekday %d) should be in days %v", d.Format("2006-01-02"), weekday, daysOfWeek)
	}
}

func TestGetEstimatedFare(t *testing.T) {
	svc := &Service{}

	tests := []struct {
		name     string
		ride     *RecurringRide
		expected float64
	}{
		{
			name: "locked price returns locked value",
			ride: &RecurringRide{
				LockedPrice: floatPtr(25.50),
			},
			expected: 25.50,
		},
		{
			name: "no locked price returns default",
			ride: &RecurringRide{
				LockedPrice: nil,
			},
			expected: 15.00, // default estimate
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.getEstimatedFare(tt.ride)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestToInstanceSlice(t *testing.T) {
	svc := &Service{}

	instances := []*ScheduledRideInstance{
		{ScheduledTime: "08:00", EstimatedFare: 10.0},
		{ScheduledTime: "09:00", EstimatedFare: 12.0},
		{ScheduledTime: "10:00", EstimatedFare: 14.0},
	}

	result := svc.toInstanceSlice(instances)

	assert.Len(t, result, 3)
	assert.Equal(t, "08:00", result[0].ScheduledTime)
	assert.Equal(t, "09:00", result[1].ScheduledTime)
	assert.Equal(t, "10:00", result[2].ScheduledTime)
	assert.Equal(t, 10.0, result[0].EstimatedFare)
}

func TestToInstanceSlice_Empty(t *testing.T) {
	svc := &Service{}

	result := svc.toInstanceSlice([]*ScheduledRideInstance{})

	assert.Len(t, result, 0)
}

func TestCalculateNextScheduledDate_Daily(t *testing.T) {
	svc := &Service{}

	// Use a date far in the future so it's always "tomorrow or later"
	fromDate := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	result := svc.calculateNextScheduledDate(fromDate, RecurrenceDaily, nil, "08:30", "UTC")

	assert.NotNil(t, result)
	assert.Equal(t, 8, result.Hour())
	assert.Equal(t, 30, result.Minute())
}

func TestCalculateNextScheduledDate_Weekdays(t *testing.T) {
	svc := &Service{}

	// Start from a Saturday in the far future
	fromDate := time.Date(2030, 1, 5, 0, 0, 0, 0, time.UTC) // Saturday
	result := svc.calculateNextScheduledDate(fromDate, RecurrenceWeekdays, nil, "09:00", "UTC")

	assert.NotNil(t, result)
	weekday := result.Weekday()
	assert.True(t, weekday >= time.Monday && weekday <= time.Friday,
		"next weekday should be a weekday, got %s", weekday)
}

func TestCalculateNextScheduledDate_Weekly(t *testing.T) {
	svc := &Service{}

	// Start from Monday in the far future, looking for Wednesday
	fromDate := time.Date(2030, 1, 7, 0, 0, 0, 0, time.UTC) // Monday
	result := svc.calculateNextScheduledDate(fromDate, RecurrenceWeekly, []int{3}, "08:00", "UTC")

	assert.NotNil(t, result)
	assert.Equal(t, time.Wednesday, result.Weekday())
}

func TestCalculateNextScheduledDate_InvalidTimezone(t *testing.T) {
	svc := &Service{}

	fromDate := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	result := svc.calculateNextScheduledDate(fromDate, RecurrenceDaily, nil, "08:00", "Invalid/TZ")

	// Should fall back to UTC and still work
	assert.NotNil(t, result)
}

func TestRecurrencePattern_Constants(t *testing.T) {
	assert.Equal(t, RecurrencePattern("daily"), RecurrenceDaily)
	assert.Equal(t, RecurrencePattern("weekdays"), RecurrenceWeekdays)
	assert.Equal(t, RecurrencePattern("weekends"), RecurrenceWeekends)
	assert.Equal(t, RecurrencePattern("weekly"), RecurrenceWeekly)
	assert.Equal(t, RecurrencePattern("biweekly"), RecurrenceBiweekly)
	assert.Equal(t, RecurrencePattern("monthly"), RecurrenceMonthly)
	assert.Equal(t, RecurrencePattern("custom"), RecurrenceCustom)
}

func TestMatchesPattern_Monthly(t *testing.T) {
	svc := &Service{}

	// Monthly matches any date (simplified implementation)
	date := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	assert.True(t, svc.matchesPattern(date, RecurrenceMonthly, nil))
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
