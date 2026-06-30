package disk

import (
	"context"
	"sync"
	"time"
)

// Threshold is a disk usage level at which the monitor fires.
type Threshold struct {
	Name  string  // "warn", "degraded", "critical"
	Pct   float64 // 0-100
	Pause bool    // whether to engage degraded-mode actions
}

var defaultThresholds = []Threshold{
	{Name: "warn", Pct: 90, Pause: false},
	{Name: "degraded", Pct: 95, Pause: true},
	{Name: "critical", Pct: 98, Pause: true},
}

// Notifier is called when a threshold is crossed (entering a level).
type Notifier func(threshold Threshold)

// DegradedAction is called when degraded mode is entered or exited. When entering
// is true, the action should engage protective measures (pause transcript writes,
// evict KB entries, stop telemetry). When false, the action should restore
// normal operation.
type DegradedAction func(entering bool)

// Monitor checks disk space at a configurable interval and notifies
// when thresholds are crossed. When a threshold with Pause=true is reached, the
// monitor enters degraded mode and fires onDegraded(true). When disk usage drops
// back below the lowest pause threshold, onRecovered fires to signal normal
// operation can resume.
type Monitor struct {
	mu         sync.Mutex
	interval   time.Duration
	thresholds []Threshold
	notify     Notifier
	current    string
	stop       chan struct{}

	// Degraded-mode lifecycle.
	onDegraded  DegradedAction
	onRecovered DegradedAction
	degraded    bool
}

// NewMonitor creates a disk space monitor. interval is the check frequency (<=0
// defaults to 5 minutes). notify is called each time a different threshold level
// is crossed.
func NewMonitor(interval time.Duration, notify Notifier) *Monitor {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &Monitor{
		interval:   interval,
		thresholds: defaultThresholds,
		notify:     notify,
		stop:       make(chan struct{}),
	}
}

// OnDegraded registers a callback invoked when degraded mode is entered (entering
// = true) or exited (entering = false). Typical actions:
//   - entering: pause transcript writes, evict KB entries, stop telemetry
//   - exiting: resume writes, re-enable telemetry
func (m *Monitor) OnDegraded(action DegradedAction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onDegraded = action
}

// OnRecovered registers a callback invoked when the monitor exits all threshold
// levels and returns to normal operation.
func (m *Monitor) OnRecovered(action DegradedAction) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onRecovered = action
}

// IsDegraded reports whether the monitor is currently in degraded mode.
func (m *Monitor) IsDegraded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.degraded
}

// Start begins periodic checking. The caller must cancel ctx to shut down.
func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stop:
			return
		case <-ticker.C:
			m.check()
		}
	}
}

// Stop halts monitoring. It is safe to call multiple times.
func (m *Monitor) Stop() {
	m.mu.Lock()
	select {
	case <-m.stop:
	default:
		close(m.stop)
	}
	m.mu.Unlock()
}

// CheckNow performs an immediate disk check. Useful for testing or manual trigger.
func (m *Monitor) CheckNow() {
	m.check()
}

func (m *Monitor) check() {
	pct := getDiskUsage()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Find the deepest threshold that matches, if any.
	var matched *Threshold
	for i := len(m.thresholds) - 1; i >= 0; i-- {
		if pct >= m.thresholds[i].Pct {
			t := m.thresholds[i]
			matched = &t
			break
		}
	}

	if matched != nil {
		// We are at or above a threshold.
		if m.current != matched.Name {
			m.current = matched.Name
			// Fire notify outside lock? We're already holding it; notify is
			// expected to be non-blocking.
			if m.notify != nil {
				m.notify(*matched)
			}
			// Engage degraded mode if this threshold requires it and we are not
			// already degraded.
			if matched.Pause && !m.degraded {
				m.degraded = true
				if m.onDegraded != nil {
					action := m.onDegraded
					m.mu.Unlock()
					action(true)
					m.mu.Lock()
				}
			}
		}
		return
	}

	// We are below all thresholds.
	if m.current != "" {
		m.current = ""
		if m.notify != nil {
			m.notify(Threshold{Name: "recovered", Pct: pct})
		}
		// Exit degraded mode and signal recovery.
		if m.degraded {
			m.degraded = false
			if m.onRecovered != nil {
				action := m.onRecovered
				m.mu.Unlock()
				action(false)
				m.mu.Lock()
			}
		}
	}
}

// PauseTranscriptWrites is a DegradedAction that pauses transcript writes by
// setting an external flag. The caller wires this to their transcript writer.
func PauseTranscriptWrites(pause *bool) DegradedAction {
	return func(entering bool) {
		if pause != nil {
			*pause = entering
		}
	}
}

// EvictKBCallback is a function that the knowledge store can register for
// eviction requests from the disk monitor when degraded mode engages.
type EvictKBCallback func()

// StopTelemetryCallback is a function that the telemetry system can register to
// pause/resume telemetry emission.
type StopTelemetryCallback func(stop bool)

// DegradedActions bundles the three degraded-mode behaviors into a single callback.
// Callers wire each action independently; nil actions are silently skipped.
func DegradedActions(pauseTranscript *bool, evictKB EvictKBCallback, stopTelemetry StopTelemetryCallback) DegradedAction {
	return func(entering bool) {
		// Transcript writes.
		if pauseTranscript != nil {
			*pauseTranscript = entering
		}
		// KB eviction: only on entering degraded mode, not on recovery.
		if entering && evictKB != nil {
			evictKB()
		}
		// Telemetry.
		if stopTelemetry != nil {
			stopTelemetry(entering)
		}
	}
}
