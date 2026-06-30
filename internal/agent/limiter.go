// Package agent — global subagent concurrency limiter.
//
// GlobalSubagentLimiter enforces a single process-wide cap on concurrent
// subagents across ALL sources (user-initiated tasks, workflow stages,
// compensations, ambient probes, and idle pre-computation). It respects
// provider rate limits and prioritises admission so that user-initiated
// work never starves behind background activity.
//
// Priority order (descending):
//  1. UserInitiated   — foreground `task` / `parallel_tasks` the user asked for
//  2. Workflow        — workflow stage subagents
//  3. Compensations   — speculative execution, cross-validation, guardian
//  4. Ambient         — idle pre-computation, dependency watchdog, KB maintenance
//
// Wait() blocks until a slot is available for the caller's source. Callers
// MUST call Release() when the subagent finishes so the slot returns to the
// pool.
package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// SubagentSource classifies where a subagent was spawned from so the limiter
// can prioritise admission.
type SubagentSource int

const (
	// UserInitiated subagents are foreground tasks the user explicitly asked
	// for — they get the highest priority and preempt lower sources.
	UserInitiated SubagentSource = iota

	// Workflow subagents are spawned by the workflow tool to execute stages.
	Workflow

	// Compensations subagents run speculative, cross-validation, or guardian
	// passes that improve quality but are not user-blocking.
	Compensations

	// Ambient subagents run idle pre-computation, dependency watching, or KB
	// maintenance — lowest priority, can be preempted by anything above.
	Ambient
)

// String returns a human-readable label for the source.
func (s SubagentSource) String() string {
	switch s {
	case UserInitiated:
		return "user-initiated"
	case Workflow:
		return "workflow"
	case Compensations:
		return "compensations"
	case Ambient:
		return "ambient"
	default:
		return fmt.Sprintf("unknown(%d)", s)
	}
}

// ParseSubagentSource maps a string label back to a SubagentSource. Unknown
// labels default to Ambient (safest: lowest priority).
func ParseSubagentSource(label string) SubagentSource {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "user-initiated", "user", "foreground", "task":
		return UserInitiated
	case "workflow", "stage":
		return Workflow
	case "compensations", "compensation", "guardian", "speculative":
		return Compensations
	case "ambient", "idle", "background", "watchdog":
		return Ambient
	default:
		return Ambient
	}
}

// limiterSlot represents one leased concurrency slot.
type limiterSlot struct {
	source    SubagentSource
	subagentID string
	acquiredAt time.Time
}

// GlobalSubagentLimiter enforces a process-wide cap on concurrent subagents
// across all sources, with priority-ordered admission. It is safe for
// concurrent use. The zero value is a valid limiter with MaxConcurrent=0
// (unlimited); call SetMax to configure it.
type GlobalSubagentLimiter struct {
	mu sync.Mutex

	// maxConcurrent is the total cap on concurrently running subagents.
	// 0 means unlimited (no limiting). Set via SetMax.
	maxConcurrent int

	// active is the set of currently leased slots, keyed by subagent ID.
	active map[string]limiterSlot

	// waiters is a priority-ordered queue of callers blocked in Wait().
	// Each waiter holds a channel that is signalled when a slot opens.
	waiters []*waiter

	// stats track admission/rejection counts for observability.
	totalAdmitted   atomic.Int64
	totalReleased   atomic.Int64
	totalWaited     atomic.Int64
	totalRejected   atomic.Int64
	sourceAdmitted  [4]atomic.Int64 // indexed by SubagentSource
	sourceRejected  [4]atomic.Int64
}

// waiter is one caller blocked in Wait().
type waiter struct {
	source SubagentSource
	ch     chan struct{} // closed when the waiter should retry
}

// NewGlobalSubagentLimiter creates a limiter with the given max concurrent
// subagents. maxConcurrent <= 0 means unlimited (Wait() never blocks).
func NewGlobalSubagentLimiter(maxConcurrent int) *GlobalSubagentLimiter {
	if maxConcurrent < 0 {
		maxConcurrent = 0
	}
	return &GlobalSubagentLimiter{
		maxConcurrent: maxConcurrent,
		active:        make(map[string]limiterSlot),
	}
}

// SetMax updates the concurrency cap. Reducing it does not evict running
// subagents; the new cap takes effect on future Wait() calls.
func (l *GlobalSubagentLimiter) SetMax(n int) {
	if n < 0 {
		n = 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.maxConcurrent = n
}

// Max returns the current concurrency cap. 0 means unlimited.
func (l *GlobalSubagentLimiter) Max() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.maxConcurrent
}

// Active returns the number of currently leased slots.
func (l *GlobalSubagentLimiter) Active() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.active)
}

// ActiveBySource returns the count of active leases per source.
func (l *GlobalSubagentLimiter) ActiveBySource() map[SubagentSource]int {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[SubagentSource]int, 4)
	for _, s := range l.active {
		out[s.source]++
	}
	return out
}

// Wait blocks until a concurrency slot is available for the given subagent
// and source. It returns a release function that the caller MUST invoke
// exactly once when the subagent finishes. If ctx is cancelled before a
// slot opens, Wait returns ctx.Err() and the caller must not call release.
//
// Priority ordering: when multiple callers are waiting, the limiter wakes
// the highest-priority waiter first (UserInitiated > Workflow >
// Compensations > Ambient). Within the same priority, waiters are FIFO.
func (l *GlobalSubagentLimiter) Wait(ctx context.Context, subagentID string, source SubagentSource) (release func(), err error) {
	subagentID = strings.TrimSpace(subagentID)
	if subagentID == "" {
		return nil, fmt.Errorf("subagent id is required")
	}

	l.mu.Lock()
	max := l.maxConcurrent

	// Unlimited mode: admit immediately.
	if max <= 0 {
		l.admitLocked(subagentID, source)
		l.mu.Unlock()
		l.totalAdmitted.Add(1)
		l.sourceAdmitted[source].Add(1)
		return func() { l.Release(subagentID) }, nil
	}

	// Fast path: slot available now.
	if len(l.active) < max {
		l.admitLocked(subagentID, source)
		l.mu.Unlock()
		l.totalAdmitted.Add(1)
		l.sourceAdmitted[source].Add(1)
		return func() { l.Release(subagentID) }, nil
	}

	// Slow path: must wait. Register as a waiter.
	w := &waiter{source: source, ch: make(chan struct{})}
	l.waiters = append(l.waiters, w)
	l.mu.Unlock()

	l.totalWaited.Add(1)

	// Wait for a signal or context cancellation.
	select {
	case <-ctx.Done():
		l.mu.Lock()
		l.removeWaiter(w)
		l.mu.Unlock()
		l.totalRejected.Add(1)
		l.sourceRejected[source].Add(1)
		return nil, ctx.Err()
	case <-w.ch:
		// We were signalled — retry admission.
	}

	// Retry loop: we might have been woken but need to wait again if
	// higher-priority waiters jumped ahead. This continues until we
	// either get a slot or ctx is done.
	for {
		l.mu.Lock()
		if len(l.active) < l.maxConcurrent {
			l.admitLocked(subagentID, source)
			l.mu.Unlock()
			l.totalAdmitted.Add(1)
			l.sourceAdmitted[source].Add(1)
			return func() { l.Release(subagentID) }, nil
		}
		// Still full — re-register and wait again.
		w2 := &waiter{source: source, ch: make(chan struct{})}
		l.waiters = append(l.waiters, w2)
		l.mu.Unlock()

		select {
		case <-ctx.Done():
			l.mu.Lock()
			l.removeWaiter(w2)
			l.mu.Unlock()
			l.totalRejected.Add(1)
			l.sourceRejected[source].Add(1)
			return nil, ctx.Err()
		case <-w2.ch:
		}
	}
}

// admitLocked records a new active lease. Caller holds l.mu.
func (l *GlobalSubagentLimiter) admitLocked(subagentID string, source SubagentSource) {
	l.active[subagentID] = limiterSlot{
		source:     source,
		subagentID: subagentID,
		acquiredAt: time.Now(),
	}
}

// Release returns a slot to the pool. It is idempotent — calling it for an
// unknown or already-released subagent ID is a no-op. When a slot frees up,
// the highest-priority waiting caller (if any) is signalled.
func (l *GlobalSubagentLimiter) Release(subagentID string) {
	l.mu.Lock()
	if _, ok := l.active[subagentID]; !ok {
		l.mu.Unlock()
		return
	}
	delete(l.active, subagentID)

	// Signal the highest-priority waiter.
	l.signalHighestPriorityWaiterLocked()
	l.mu.Unlock()
	l.totalReleased.Add(1)
}

// signalHighestPriorityWaiterLocked picks the waiter with the best (lowest)
// SubagentSource value and signals it. Caller holds l.mu.
func (l *GlobalSubagentLimiter) signalHighestPriorityWaiterLocked() {
	if len(l.waiters) == 0 {
		return
	}

	// Find the highest-priority waiter (lowest source value).
	bestIdx := 0
	for i := 1; i < len(l.waiters); i++ {
		if l.waiters[i].source < l.waiters[bestIdx].source {
			bestIdx = i
		}
	}

	w := l.waiters[bestIdx]
	// Remove from the slice (order among same-priority doesn't matter
	// for signalling — only one wakes up per Release).
	l.waiters = append(l.waiters[:bestIdx], l.waiters[bestIdx+1:]...)
	close(w.ch)
}

// removeWaiter deletes a specific waiter from the queue (used on
// cancellation). Caller holds l.mu.
func (l *GlobalSubagentLimiter) removeWaiter(w *waiter) {
	for i, c := range l.waiters {
		if c == w {
			l.waiters = append(l.waiters[:i], l.waiters[i+1:]...)
			return
		}
	}
}

// Stats returns a snapshot of limiter telemetry.
type LimiterStats struct {
	MaxConcurrent   int
	Active          int
	TotalAdmitted   int64
	TotalReleased   int64
	TotalWaited     int64
	TotalRejected   int64
	BySource        map[SubagentSource]int
	SourceAdmitted  map[SubagentSource]int64
	SourceRejected  map[SubagentSource]int64
	Waiters         int
}

// Stats returns a snapshot of current limiter state and counters.
func (l *GlobalSubagentLimiter) Stats() LimiterStats {
	l.mu.Lock()
	defer l.mu.Unlock()

	bySource := make(map[SubagentSource]int, 4)
	for _, s := range l.active {
		bySource[s.source]++
	}

	admitted := make(map[SubagentSource]int64, 4)
	rejected := make(map[SubagentSource]int64, 4)
	for i := SubagentSource(0); i <= Ambient; i++ {
		admitted[i] = l.sourceAdmitted[i].Load()
		rejected[i] = l.sourceRejected[i].Load()
	}

	return LimiterStats{
		MaxConcurrent:  l.maxConcurrent,
		Active:         len(l.active),
		TotalAdmitted:  l.totalAdmitted.Load(),
		TotalReleased:  l.totalReleased.Load(),
		TotalWaited:    l.totalWaited.Load(),
		TotalRejected:  l.totalRejected.Load(),
		BySource:       bySource,
		SourceAdmitted: admitted,
		SourceRejected: rejected,
		Waiters:        len(l.waiters),
	}
}

// WaitQueue returns the ordered list of waiting sources, highest priority
// first. Useful for diagnostics.
func (l *GlobalSubagentLimiter) WaitQueue() []SubagentSource {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Sort waiters by priority, then copy sources.
	sorted := append([]*waiter(nil), l.waiters...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].source < sorted[j].source
	})

	out := make([]SubagentSource, len(sorted))
	for i, w := range sorted {
		out[i] = w.source
	}
	return out
}
