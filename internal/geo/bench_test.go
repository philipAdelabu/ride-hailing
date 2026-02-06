package geo

import "testing"

func BenchmarkHaversineDistance(b *testing.B) {
	for i := 0; i < b.N; i++ {
		haversineDistance(37.7749, -122.4194, 34.0522, -118.2437)
	}
}

func BenchmarkEstimateETAMinutes_Moving(b *testing.B) {
	for i := 0; i < b.N; i++ {
		estimateETAMinutes(10.0, 45.0)
	}
}

func BenchmarkEstimateETAMinutes_Fallback(b *testing.B) {
	for i := 0; i < b.N; i++ {
		estimateETAMinutes(10.0, 2.0)
	}
}
