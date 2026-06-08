package window

import (
	"sync"

	"github.com/tradebench/ingester/internal/correctness"
	"github.com/tradebench/ingester/internal/histogram"
)

// TelemetryEvent mirrors the JSON published by the bots service to telemetry.raw.
type TelemetryEvent struct {
	RunID       string  `json:"run_id"`
	BotID       string  `json:"bot_id"`
	OrderID     string  `json:"order_id"`
	SentAtNs    int64   `json:"sent_at_ns"`
	AckAtNs     int64   `json:"ack_at_ns"`
	CorrectFill bool    `json:"correct_fill"`
	OrderType   string  `json:"order_type"`
	Rejected    bool    `json:"rejected"`
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	Price       float64 `json:"price,omitempty"`
	Quantity    int     `json:"quantity"`
}

// RunSnapshot holds the final computed metrics for a completed run.
type RunSnapshot struct {
	RunID        string
	ContestantID string
	P50Ms        float64
	P90Ms        float64
	P99Ms        float64
	MaxTPS       int64
	Correctness  float64
}

// runWindow accumulates telemetry for a single run.
type runWindow struct {
	contestantID  string
	hist          *histogram.Histogram
	orderCount    int64
	startTimeNs   int64
	lastEventNs   int64
	durationSecs  int
	events        []correctness.FillEvent
	mu            sync.Mutex
}

// Manager tracks active run windows and detects when they complete.
type Manager struct {
	windows            map[string]*runWindow
	mu                 sync.Mutex
	defaultDurationSec int
}

// NewManager creates a window manager. defaultDurationSec is used when we
// don't know the exact run duration (it comes from the RunStartedEvent which
// the ingester doesn't consume — the bots do).
func NewManager(defaultDurationSec int) *Manager {
	return &Manager{
		windows:            make(map[string]*runWindow),
		defaultDurationSec: defaultDurationSec,
	}
}

// AddEvent records a telemetry event into the appropriate run window.
// Creates a new window on first event for a given run_id.
func (m *Manager) AddEvent(event TelemetryEvent) {
	m.mu.Lock()
	w, ok := m.windows[event.RunID]
	if !ok {
		w = &runWindow{
			hist:         histogram.New(),
			startTimeNs:  event.SentAtNs,
			durationSecs: m.defaultDurationSec,
		}
		m.windows[event.RunID] = w
	}
	m.mu.Unlock()

	w.mu.Lock()
	defer w.mu.Unlock()

	w.orderCount++

	if !event.Rejected {
		w.events = append(w.events, correctness.FillEvent{
			OrderID:  event.OrderID,
			Symbol:   event.Symbol,
			Side:     event.Side,
			Type:     event.OrderType,
			Quantity: event.Quantity,
			Price:    event.Price,
			SentAtNs: event.SentAtNs,
			Filled:   event.CorrectFill,
		})
	}

	latencyNs := event.AckAtNs - event.SentAtNs
	if latencyNs > 0 {
		w.hist.Record(latencyNs)
	}

	if event.SentAtNs < w.startTimeNs {
		w.startTimeNs = event.SentAtNs
	}
	if event.AckAtNs > w.lastEventNs {
		w.lastEventNs = event.AckAtNs
	}
}

// SetContestantID associates a contestant with a run. Called when we can
// infer the contestant from the first event or from external context.
func (m *Manager) SetContestantID(runID, contestantID string) {
	m.mu.Lock()
	w, ok := m.windows[runID]
	m.mu.Unlock()
	if !ok {
		return
	}
	w.mu.Lock()
	w.contestantID = contestantID
	w.mu.Unlock()
}

// CheckCompleted scans all active windows and returns snapshots for any
// runs that have exceeded their duration. Completed windows are removed.
func (m *Manager) CheckCompleted() []RunSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	var completed []RunSnapshot

	for runID, w := range m.windows {
		w.mu.Lock()
		if w.isComplete() {
			snap := w.snapshot(runID)
			completed = append(completed, snap)
			w.mu.Unlock()
			delete(m.windows, runID)
		} else {
			w.mu.Unlock()
		}
	}

	return completed
}

// isComplete returns true when enough time has elapsed between first and last event.
// Must be called with w.mu held.
func (w *runWindow) isComplete() bool {
	if w.orderCount == 0 {
		return false
	}
	elapsedNs := w.lastEventNs - w.startTimeNs
	elapsedSec := elapsedNs / 1e9
	return elapsedSec >= int64(w.durationSecs)
}

// snapshot computes final metrics. Must be called with w.mu held.
func (w *runWindow) snapshot(runID string) RunSnapshot {
	elapsedSec := float64(w.lastEventNs-w.startTimeNs) / 1e9
	var maxTPS int64
	if elapsedSec > 0 {
		maxTPS = int64(float64(w.orderCount) / elapsedSec)
	}

	v := correctness.NewValidator()
	correctnessRatio := v.Validate(w.events)

	return RunSnapshot{
		RunID:        runID,
		ContestantID: w.contestantID,
		P50Ms:        w.hist.Percentile(50.0),
		P90Ms:        w.hist.Percentile(90.0),
		P99Ms:        w.hist.Percentile(99.0),
		MaxTPS:       maxTPS,
		Correctness:  correctnessRatio,
	}
}
