package resilience

import (
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sony/gobreaker"
)

var (
	breakerStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_breaker_state",
		Help: "Current state of circuit breakers (0=closed, 0.5=half-open, 1=open)",
	}, []string{"breaker"})

	breakerRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_requests_total",
		Help: "Total number of operations executed through a circuit breaker",
	}, []string{"breaker"})

	breakerFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_failures_total",
		Help: "Total number of circuit breaker executions that resulted in an error",
	}, []string{"breaker"})

	breakerFallbacksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_fallbacks_total",
		Help: "Total number of times breaker fallbacks were triggered because the breaker was open",
	}, []string{"breaker"})

	breakerStateTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_state_changes_total",
		Help: "Total number of circuit breaker state transitions",
	}, []string{"breaker", "from", "to"})

	breakerIDCounter uint64
)

func nextBreakerName(base string) string {
	if base != "" {
		return base
	}
	id := atomic.AddUint64(&breakerIDCounter, 1)
	return "breaker-" + strconv.FormatUint(id, 10)
}

func breakerStateValue(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateClosed:
		return 0
	case gobreaker.StateHalfOpen:
		return 0.5
	case gobreaker.StateOpen:
		return 1
	default:
		return -1
	}
}

func recordBreakerState(name string, state gobreaker.State) {
	breakerStateGauge.WithLabelValues(name).Set(breakerStateValue(state))
}

func recordBreakerStateChange(name string, from, to gobreaker.State) {
	breakerStateTransitions.WithLabelValues(name, from.String(), to.String()).Inc()
	recordBreakerState(name, to)
}

func recordBreakerRequest(name string) {
	breakerRequestsTotal.WithLabelValues(name).Inc()
}

func recordBreakerFailure(name string) {
	breakerFailuresTotal.WithLabelValues(name).Inc()
}

func recordBreakerFallback(name string) {
	breakerFallbacksTotal.WithLabelValues(name).Inc()
}
