package scoring

// weights for the final score
const (
	weightLatency     = 0.35
	weightThroughput  = 0.35
	weightCorrectness = 0.30
)

// baseline values — anything worse than these scores zero
const (
	maxAcceptableP99Ms = 500.0
	minAcceptableTPS   = 100
)

// Calculate returns a 0–100 score from raw run metrics.
// Lower latency is better, higher TPS is better, higher correctness is better.
func Calculate(p99Ms float64, maxTPS int64, correctness float64) float64 {
	latencyScore := latencyComponent(p99Ms)
	tpsScore := throughputComponent(maxTPS)
	correctnessScore := correctness

	raw := weightLatency*latencyScore + weightThroughput*tpsScore + weightCorrectness*correctnessScore
	return clamp(raw*100, 0, 100)
}

func latencyComponent(p99Ms float64) float64 {
	if p99Ms <= 0 {
		return 1.0
	}
	if p99Ms >= maxAcceptableP99Ms {
		return 0.0
	}
	// linear scale: 0ms → 1.0, 500ms → 0.0
	return 1.0 - (p99Ms / maxAcceptableP99Ms)
}

func throughputComponent(maxTPS int64) float64 {
	if maxTPS <= 0 {
		return 0.0
	}
	if maxTPS >= int64(minAcceptableTPS*10) {
		return 1.0
	}
	// linear scale: 0 → 0.0, 1000 → 1.0
	return float64(maxTPS) / float64(minAcceptableTPS*10)
}

func clamp(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}
