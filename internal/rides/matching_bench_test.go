package rides

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type benchProvider struct {
	candidates []*DriverCandidate
}

func (p *benchProvider) GetNearbyDriverCandidates(_ context.Context, _, _ float64, _ float64, _ int) ([]*DriverCandidate, error) {
	return p.candidates, nil
}

func makeBenchCandidates(n int) []*DriverCandidate {
	candidates := make([]*DriverCandidate, n)
	for i := range candidates {
		candidates[i] = &DriverCandidate{
			DriverID:       uuid.New(),
			DistanceKm:     float64(i%20) + 0.5,
			Rating:         3.5 + float64(i%30)/20.0,
			AcceptanceRate: 0.5 + float64(i%50)/100.0,
			IdleMinutes:    float64(i%60) + 1,
		}
	}
	return candidates
}

func BenchmarkMatcher_FindBestDrivers_10(b *testing.B) {
	benchmarkMatcherN(b, 10)
}

func BenchmarkMatcher_FindBestDrivers_50(b *testing.B) {
	benchmarkMatcherN(b, 50)
}

func BenchmarkMatcher_FindBestDrivers_200(b *testing.B) {
	benchmarkMatcherN(b, 200)
}

func BenchmarkMatcher_FindBestDrivers_1000(b *testing.B) {
	benchmarkMatcherN(b, 1000)
}

func benchmarkMatcherN(b *testing.B, n int) {
	provider := &benchProvider{candidates: makeBenchCandidates(n)}
	matcher := NewMatcher(DefaultMatchingConfig(), provider)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = matcher.FindBestDrivers(context.Background(), 37.7749, -122.4194)
	}
}

func BenchmarkScoreCandidate(b *testing.B) {
	matcher := NewMatcher(DefaultMatchingConfig(), nil)
	c := &DriverCandidate{
		DistanceKm:     5.0,
		Rating:         4.5,
		AcceptanceRate: 0.85,
		IdleMinutes:    15.0,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.scoreCandidate(c, 20.0, 60.0)
	}
}
