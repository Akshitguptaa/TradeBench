package status

import "sync"

type State string

const (
	Queued    State = "queued"
	Running   State = "running"
	Completed State = "completed"
	Failed    State = "failed"
)

// current status of a single submission.
type Entry struct {
	SubmissionID string `json:"submission_id"`
	ContestantID string `json:"contestant_id"`
	Status       State  `json:"status"`
	RunID        string `json:"run_id,omitempty"`
}

// thread-safe in-memory submission status store
type Tracker struct {
	mu      sync.RWMutex
	entries map[string]*Entry
}

func NewTracker() *Tracker {
	return &Tracker{
		entries: make(map[string]*Entry),
	}
}

// Set stores or updates the status for a submission.
func (t *Tracker) Set(submissionID, contestantID string, status State) {
	t.mu.Lock()
	defer t.mu.Unlock()

	e, ok := t.entries[submissionID]
	if !ok {
		t.entries[submissionID] = &Entry{
			SubmissionID: submissionID,
			ContestantID: contestantID,
			Status:       status,
		}
		return
	}
	e.Status = status
	if contestantID != "" {
		e.ContestantID = contestantID
	}
}

// SetRunID records the run_id once the sandbox picks up the submission.
func (t *Tracker) SetRunID(submissionID, runID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if e, ok := t.entries[submissionID]; ok {
		e.RunID = runID
	}
}

// Get returns the status entry for a submission, or nil if unknown.
func (t *Tracker) Get(submissionID string) *Entry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	e, ok := t.entries[submissionID]
	if !ok {
		return nil
	}

	copy := *e
	return &copy
}
