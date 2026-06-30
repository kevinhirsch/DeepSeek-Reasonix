package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"reasonix/internal/event"
)

// CompensationSafety wraps every compensation execution in recover() and a
// timeout. If a compensation panics, times out, or fails, the original output
// is returned unmodified with a notice attached. No compensation failure ever
// blocks the parent task — compensations are additive quality layers, not
// load-bearing infrastructure.
type CompensationSafety struct {
	// MaxDuration is the per-compensation execution timeout. Zero means no
	// timeout. Defaults to 30s for blocking compensations; background
	// compensations may have longer timeouts.
	MaxDuration time.Duration

	// Sink receives safety notices (panics, timeouts, errors) so the frontend
	// can surface them.
	Sink event.Sink
}

// NewCompensationSafety creates a safety wrapper with the default 30-second
// timeout.
func NewCompensationSafety() *CompensationSafety {
	return &CompensationSafety{
		MaxDuration: 30 * time.Second,
	}
}

// WithSink sets the event sink for safety notices.
func (s *CompensationSafety) WithSink(sink event.Sink) *CompensationSafety {
	s.Sink = sink
	return s
}

// SafeExecute runs a compensation function with panic recovery and an optional
// timeout. If the function panics, times out, or returns an error, the original
// input is returned unmodified and a notice is emitted via the configured Sink.
//
// name is a human-readable compensation name for logging/notices.
// originalInput is the value returned on failure (compensation input passes
// through unchanged).
// fn is the compensation work to execute safely.
//
// Returns (output, ok) where ok=false means the compensation failed and
// originalInput was returned.
func (s *CompensationSafety) SafeExecute(
	ctx context.Context,
	name string,
	originalInput string,
	fn func(context.Context) (string, error),
) (output string, ok bool) {
	// Create a channel to receive the result.
	type result struct {
		out string
		err error
	}
	done := make(chan result, 1)

	// Track if we recovered from a panic.
	var panicVal interface{}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
				// Send a placeholder result so the select below unblocks.
				// The actual output is discarded in favour of originalInput.
				done <- result{out: "", err: fmt.Errorf("panic: %v", r)}
			}
		}()
		out, err := fn(ctx)
		done <- result{out: out, err: err}
	}()

	// Wait with timeout support if configured.
	var r result
	if s.MaxDuration > 0 {
		timer := time.NewTimer(s.MaxDuration)
		defer timer.Stop()
		select {
		case r = <-done:
			// Normal completion (possibly with error or panic).
		case <-timer.C:
			// Timeout — don't block the parent.
			s.emitNotice(name, "timed out after "+s.MaxDuration.String(), originalInput)
			return originalInput, false
		case <-ctx.Done():
			// Context cancelled — don't block.
			s.emitNotice(name, "cancelled (context done)", originalInput)
			return originalInput, false
		}
	} else {
		select {
		case r = <-done:
		case <-ctx.Done():
			s.emitNotice(name, "cancelled (context done)", originalInput)
			return originalInput, false
		}
	}

	// Handle panic.
	if panicVal != nil {
		notice := fmt.Sprintf("compensation %q panicked: %v", name, panicVal)
		s.emitNotice(name, notice, originalInput)
		return originalInput, false
	}

	// Handle error.
	if r.err != nil {
		// Context cancellation is not a compensation failure — it's an
		// intentional parent action. Don't emit a notice.
		if ctx.Err() != nil {
			return originalInput, false
		}
		notice := fmt.Sprintf("compensation %q failed: %v", name, r.err)
		s.emitNotice(name, notice, originalInput)
		return originalInput, false
	}

	return r.out, true
}

// SafeExecuteNoInput is like SafeExecute but for compensations that don't
// have a string input/output (e.g. background analysis). Returns ok=false
// on failure.
func (s *CompensationSafety) SafeExecuteNoInput(
	ctx context.Context,
	name string,
	fn func(context.Context) error,
) (ok bool) {
	_, ok = s.SafeExecute(ctx, name, "", func(ctx context.Context) (string, error) {
		return "", fn(ctx)
	})
	return ok
}

// emitNotice sends a safety notice to the configured Sink. If no Sink is
// configured, the notice is silently dropped.
func (s *CompensationSafety) emitNotice(name, details, originalInput string) {
	if s.Sink == nil {
		return
	}
	// Truncate the original input for the notice — full input may be very large.
	preview := originalInput
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}
	text := fmt.Sprintf("⚠ Compensation safety: %s — %s. "+
		"Original output returned unmodified. Re-enable via Settings → Compensations.",
		name, details)
	s.Sink.Emit(event.Event{
		Kind:  event.Notice,
		Level: event.LevelWarn,
		Text:  text,
	})
}

// CompensationSafetyStats tracks safety statistics across all compensations.
type CompensationSafetyStats struct {
	mu          sync.Mutex
	TotalRuns   int            `json:"total_runs"`
	Successes   int            `json:"successes"`
	Failures    int            `json:"failures"`
	Panics      int            `json:"panics"`
	Timeouts    int            `json:"timeouts"`
	ByCompensation map[string]*CompensationSafetySnapshot `json:"by_compensation"`
}

// CompensationSafetySnapshot tracks outcomes for a single compensation.
type CompensationSafetySnapshot struct {
	Name      string `json:"name"`
	Runs      int    `json:"runs"`
	Successes int    `json:"successes"`
	Failures  int    `json:"failures"`
	Panics    int    `json:"panics"`
	Timeouts  int    `json:"timeouts"`
}

// NewCompensationSafetyStats creates a safety statistics tracker.
func NewCompensationSafetyStats() *CompensationSafetyStats {
	return &CompensationSafetyStats{
		ByCompensation: make(map[string]*CompensationSafetySnapshot),
	}
}

// RecordSuccess logs a successful compensation run.
func (st *CompensationSafetyStats) RecordSuccess(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.TotalRuns++
	st.Successes++
	st.snapshot(name).Successes++
	st.snapshot(name).Runs++
}

// RecordFailure logs a compensation failure.
func (st *CompensationSafetyStats) RecordFailure(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.TotalRuns++
	st.Failures++
	st.snapshot(name).Failures++
	st.snapshot(name).Runs++
}

// RecordPanic logs a compensation panic.
func (st *CompensationSafetyStats) RecordPanic(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.TotalRuns++
	st.Panics++
	st.snapshot(name).Panics++
	st.snapshot(name).Runs++
}

// RecordTimeout logs a compensation timeout.
func (st *CompensationSafetyStats) RecordTimeout(name string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.TotalRuns++
	st.Timeouts++
	st.snapshot(name).Timeouts++
	st.snapshot(name).Runs++
}

// snapshot returns the per-compensation stats entry, creating it if needed.
// Caller must hold st.mu.
func (st *CompensationSafetyStats) snapshot(name string) *CompensationSafetySnapshot {
	s, ok := st.ByCompensation[name]
	if !ok {
		s = &CompensationSafetySnapshot{Name: name}
		st.ByCompensation[name] = s
	}
	return s
}

// SuccessRate returns the overall compensation success rate as a fraction
// (0.0–1.0). Returns 1.0 if no compensations have run.
func (st *CompensationSafetyStats) SuccessRate() float64 {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.TotalRuns == 0 {
		return 1.0
	}
	return float64(st.Successes) / float64(st.TotalRuns)
}

// Report returns a human-readable summary of safety statistics.
func (st *CompensationSafetyStats) Report() string {
	st.mu.Lock()
	defer st.mu.Unlock()
	var b strings.Builder
	b.WriteString("# Compensation Safety Report\n\n")
	fmt.Fprintf(&b, "Total runs: %d\n", st.TotalRuns)
	fmt.Fprintf(&b, "Successes: %d (%.1f%%)\n",
		st.Successes, st.pct(st.Successes))
	fmt.Fprintf(&b, "Failures: %d (%.1f%%)\n",
		st.Failures, st.pct(st.Failures))
	fmt.Fprintf(&b, "Panics: %d (%.1f%%)\n",
		st.Panics, st.pct(st.Panics))
	fmt.Fprintf(&b, "Timeouts: %d (%.1f%%)\n\n",
		st.Timeouts, st.pct(st.Timeouts))

	if len(st.ByCompensation) > 0 {
		b.WriteString("## Per compensation\n\n")
		for _, snap := range st.ByCompensation {
			sr := 100.0
			if snap.Runs > 0 {
				sr = float64(snap.Successes) / float64(snap.Runs) * 100
			}
			fmt.Fprintf(&b, "- %s: %d runs, %.0f%% success, %d failures, %d panics, %d timeouts\n",
				snap.Name, snap.Runs, sr, snap.Failures, snap.Panics, snap.Timeouts)
		}
	}
	return b.String()
}

func (st *CompensationSafetyStats) pct(count int) float64 {
	if st.TotalRuns == 0 {
		return 0
	}
	return float64(count) / float64(st.TotalRuns) * 100
}
