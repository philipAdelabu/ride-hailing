package analytics

import (
	"context"
	"time"
)

// AnalyticsRepository defines the persistence operations required by the service.
type AnalyticsRepository interface {
	GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error)
	GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error)
	GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error)
	GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error)
	GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error)
	GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error)
	GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error)
	GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error)
	GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error)
	// New analytics endpoints
	GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error)
	GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error)
	GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error)
	GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error)
	GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error)
	GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error)
	GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error)
}

// Service handles analytics business logic
type Service struct {
	repo AnalyticsRepository
}

// NewService creates a new analytics service
func NewService(repo AnalyticsRepository) *Service {
	return &Service{repo: repo}
}

// GetRevenueMetrics retrieves revenue statistics
func (s *Service) GetRevenueMetrics(ctx context.Context, startDate, endDate time.Time) (*RevenueMetrics, error) {
	return s.repo.GetRevenueMetrics(ctx, startDate, endDate)
}

// GetPromoCodePerformance retrieves promo code statistics
func (s *Service) GetPromoCodePerformance(ctx context.Context, startDate, endDate time.Time) ([]*PromoCodePerformance, error) {
	return s.repo.GetPromoCodePerformance(ctx, startDate, endDate)
}

// GetRideTypeStats retrieves ride type statistics
func (s *Service) GetRideTypeStats(ctx context.Context, startDate, endDate time.Time) ([]*RideTypeStats, error) {
	return s.repo.GetRideTypeStats(ctx, startDate, endDate)
}

// GetReferralMetrics retrieves referral program statistics
func (s *Service) GetReferralMetrics(ctx context.Context, startDate, endDate time.Time) (*ReferralMetrics, error) {
	return s.repo.GetReferralMetrics(ctx, startDate, endDate)
}

// GetTopDrivers retrieves top performing drivers
func (s *Service) GetTopDrivers(ctx context.Context, startDate, endDate time.Time, limit int) ([]*DriverPerformance, error) {
	return s.repo.GetTopDrivers(ctx, startDate, endDate, limit)
}

// GetDashboardMetrics retrieves overall platform metrics
func (s *Service) GetDashboardMetrics(ctx context.Context) (*DashboardMetrics, error) {
	return s.repo.GetDashboardMetrics(ctx)
}

// GetDemandHeatMap retrieves geographic demand data for heat map visualization
func (s *Service) GetDemandHeatMap(ctx context.Context, startDate, endDate time.Time, gridSize float64) ([]*DemandHeatMap, error) {
	return s.repo.GetDemandHeatMap(ctx, startDate, endDate, gridSize)
}

// GetFinancialReport generates a comprehensive financial report for a period
func (s *Service) GetFinancialReport(ctx context.Context, startDate, endDate time.Time) (*FinancialReport, error) {
	return s.repo.GetFinancialReport(ctx, startDate, endDate)
}

// GetDemandZones identifies high-demand geographic zones
func (s *Service) GetDemandZones(ctx context.Context, startDate, endDate time.Time, minRides int) ([]*DemandZone, error) {
	return s.repo.GetDemandZones(ctx, startDate, endDate, minRides)
}

// GetRevenueTimeSeries retrieves time-series revenue data for charts
func (s *Service) GetRevenueTimeSeries(ctx context.Context, startDate, endDate time.Time, granularity string) ([]*RevenueTimeSeries, error) {
	return s.repo.GetRevenueTimeSeries(ctx, startDate, endDate, granularity)
}

// GetHourlyDistribution retrieves ride distribution by hour
func (s *Service) GetHourlyDistribution(ctx context.Context, startDate, endDate time.Time) ([]*HourlyDistribution, error) {
	return s.repo.GetHourlyDistribution(ctx, startDate, endDate)
}

// GetDriverAnalytics retrieves overall driver performance analytics
func (s *Service) GetDriverAnalytics(ctx context.Context, startDate, endDate time.Time) (*DriverAnalytics, error) {
	return s.repo.GetDriverAnalytics(ctx, startDate, endDate)
}

// GetRiderGrowth retrieves rider growth and retention metrics
func (s *Service) GetRiderGrowth(ctx context.Context, startDate, endDate time.Time) (*RiderGrowth, error) {
	return s.repo.GetRiderGrowth(ctx, startDate, endDate)
}

// GetRideMetrics retrieves quality of service metrics
func (s *Service) GetRideMetrics(ctx context.Context, startDate, endDate time.Time) (*RideMetrics, error) {
	return s.repo.GetRideMetrics(ctx, startDate, endDate)
}

// GetTopDriversDetailed retrieves top performing drivers with detailed metrics
func (s *Service) GetTopDriversDetailed(ctx context.Context, startDate, endDate time.Time, limit int) ([]*TopDriver, error) {
	return s.repo.GetTopDriversDetailed(ctx, startDate, endDate, limit)
}

// GetPeriodComparison compares metrics between two periods
func (s *Service) GetPeriodComparison(ctx context.Context, currentStart, currentEnd, previousStart, previousEnd time.Time) (*PeriodComparison, error) {
	return s.repo.GetPeriodComparison(ctx, currentStart, currentEnd, previousStart, previousEnd)
}
