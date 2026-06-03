package histogram

import (
	hdr "github.com/HdrHistogram/hdrhistogram-go"
)

// Histogram wraps an HDR histogram for recording nanosecond latencies
// and extracting millisecond percentiles.
type Histogram struct {
	h *hdr.Histogram
}

// New creates a histogram that can record latencies from 1 nanosecond
// up to 10 seconds (10,000,000,000 ns) with 3 significant digits of precision.
func New() *Histogram {
	return &Histogram{
		h: hdr.New(1, 10_000_000_000, 3),
	}
}

// Record adds a single latency measurement in nanoseconds.
func (h *Histogram) Record(latencyNs int64) {
	if latencyNs < 1 {
		latencyNs = 1
	}
	_ = h.h.RecordValue(latencyNs)
}

// Percentile returns the latency at the given percentile converted to milliseconds.
// For example, Percentile(99.0) returns the p99 latency in ms.
func (h *Histogram) Percentile(p float64) float64 {
	return float64(h.h.ValueAtQuantile(p)) / 1e6
}

// TotalCount returns the number of recorded values.
func (h *Histogram) TotalCount() int64 {
	return h.h.TotalCount()
}

// Reset clears all recorded values.
func (h *Histogram) Reset() {
	h.h.Reset()
}
