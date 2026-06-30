// Package agent — compensation_safety.go: safety wrapper for compensations.
//
// Every compensation call is wrapped in a recover() so that a panic in any
// compensation (nil dereference, out-of-bounds, unexpected provider error) is
// caught and the original output — the result the parent task produced before
// compensation ran — is returned unchanged. A notice is emitted so the user is
// aware that a compensation failed silently. No compensation failure ever blocks
// the parent task.
package agent

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
)

// CompensationSafety wraps every compensation call in a recover() block. If the
// compensation panics, the original output is returned unchanged, an error notice
// is emitted, and the failure is logged. The parent task proceeds as if the
// compensation had never run.
//
// Usage:
//
//	safety := NewCompensationSafety()
//	result := safety.Wrap(ctx, originalOutput, func() (string, error) {
//	    return runCompensation(ctx, input)
//	}, "speculative")
//
// OnFailure can be set to receive structured notices for UI surfacing.
type CompensationSafety struct {
	// OnFailure is called when a compensation panics or returns an error. If nil,
	// failures are only logged to slog.
	OnFailure func(ctx context.Context, notice CompensationFailureNotice)

	mu        sync.Mutex
	failures  []CompensationFailureNotice
	logFailed bool
}

// CompensationFailureNotice describes a compensation failure that was caught and
// suppressed so the parent task could continue.
type CompensationFailureNotice struct {
	// CompensationName identifies which compensation failed (e.g., "speculative",
	// "cross-validate").
	CompensationName string `json:"compensation_name"`

	// Error is the error message or panic value, formatted as a string.
	Error string `json:"error"`

	// StackTrace is the goroutine stack at the point of the panic. Empty if the
	// compensation returned a regular error instead of panicking.
	StackTrace string `json:"stack_trace,omitempty"`

	// OriginalOutputSize is the length (in bytes) of the original output that
	// was returned instead, so the caller can gauge impact.
	OriginalOutputSize int `json:"original_output_size"`

	// SubagentID is the subagent whose compensation failed, if applicable.
	SubagentID string `json:"subagent_id,omitempty"`
}

// NewCompensationSafety creates a safety wrapper.
func NewCompensationSafety() *CompensationSafety {
	return &CompensationSafety{logFailed: true}
}

// Wrap runs fn() inside a recover(). If fn panics, the original output is
// returned unchanged, a notice is recorded and emitted via OnFailure, and the
// stack trace is logged at WARN level.
//
// If fn returns normally without error, the compensation's result is returned
// instead of the original output (the compensation succeeded).
//
// If fn returns an error (without panicking), the original output is returned
// unchanged, a notice is recorded, and the error is logged — same as a panic,
// minus the stack trace.
//
// Wrap is safe for concurrent use.
func (s *CompensationSafety) Wrap(
	ctx context.Context,
	originalOutput string,
	fn func() (string, error),
	compensationName string,
) string {
	var result string
	var fnErr error
	var panicVal interface{}

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicVal = r
			}
		}()
		result, fnErr = fn()
	}()

	// Normal success — return the compensation result.
	if panicVal == nil && fnErr == nil {
		return result
	}

	// Compensation failed. Build a notice.
	notice := CompensationFailureNotice{
		CompensationName:   compensationName,
		OriginalOutputSize: len(originalOutput),
	}

	if panicVal != nil {
		notice.Error = fmt.Sprintf("panic: %v", panicVal)
		notice.StackTrace = string(debug.Stack())
	} else {
		notice.Error = fnErr.Error()
	}

	s.mu.Lock()
	s.failures = append(s.failures, notice)
	s.mu.Unlock()

	if s.logFailed {
		if panicVal != nil {
			slog.Warn("compensation panicked, returning original output unchanged",
				"compensation", compensationName,
				"panic", panicVal,
				"original_output_size", formatOutputSize(len(originalOutput)),
			)
		} else {
			slog.Warn("compensation failed, returning original output unchanged",
				"compensation", compensationName,
				"error", fnErr.Error(),
				"original_output_size", formatOutputSize(len(originalOutput)),
			)
		}
	}

	if s.OnFailure != nil {
		// Call OnFailure in a separate recover so a buggy callback doesn't
		// defeat the purpose of the safety wrapper.
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("CompensationSafety.OnFailure callback panicked",
						"compensation", compensationName,
						"panic", r,
					)
				}
			}()
			s.OnFailure(ctx, notice)
		}()
	}

	// Always return the original output. The parent task must never see a
	// compensation failure.
	return originalOutput
}

// WrapNoError is like Wrap but for compensations that don't return an error.
// The fn is expected to return only the processed string; a panic is still
// caught and the original output returned.
func (s *CompensationSafety) WrapNoError(
	ctx context.Context,
	originalOutput string,
	fn func() string,
	compensationName string,
) string {
	return s.Wrap(ctx, originalOutput, func() (string, error) {
		return fn(), nil
	}, compensationName)
}

// WrapWithSubagent is like Wrap but records the subagent ID in the failure
// notice so the caller can associate failures with specific subagents.
func (s *CompensationSafety) WrapWithSubagent(
	ctx context.Context,
	originalOutput string,
	fn func() (string, error),
	compensationName string,
	subagentID string,
) string {
	var result string
	succeeded := false

	result = s.Wrap(ctx, originalOutput, func() (string, error) {
		r, err := fn()
		if err == nil {
			succeeded = true
		}
		return r, err
	}, compensationName)

	// If the compensation failed, annotate the most recent notice with the
	// subagent ID.
	if !succeeded {
		s.mu.Lock()
		if len(s.failures) > 0 {
			s.failures[len(s.failures)-1].SubagentID = subagentID
		}
		s.mu.Unlock()
	}

	return result
}

// Failures returns a copy of all failure notices recorded since creation or the
// last call to ResetFailures. Returns notices in order of occurrence.
func (s *CompensationSafety) Failures() []CompensationFailureNotice {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]CompensationFailureNotice, len(s.failures))
	copy(out, s.failures)
	return out
}

// ResetFailures clears the recorded failure notices.
func (s *CompensationSafety) ResetFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failures = nil
}

// FailureCount returns the total number of suppressed compensation failures.
func (s *CompensationSafety) FailureCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.failures)
}

// formatOutputSize formats a byte length in a human-readable way for logging.
func formatOutputSize(n int) string {
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}
