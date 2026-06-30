package disk

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Threshold is a disk usage level at which the monitor fires.
type Threshold struct {
	Name  string  // "warn", "degraded", "critical"
	Pct   float64 // 0-100
	Pause bool    // whether to pause writes / evict KB / stop telemetry
}

var defaultThresholds = []Threshold{
	{Name: "warn", Pct: 90, Pause: false},
	{Name: "degraded", Pct: 95, Pause: true},
	{Name: "critical", Pct: 98, Pause: true},
}

// Notifier is called when a threshold is crossed.
type Notifier func(threshold Threshold)

// Monitor checks disk space at a configurable interval and notifies
// when thresholds are crossed.
type Monitor struct {
	mu         sync.Mutex
	interval   time.Duration
	thresholds []Threshold
	notify     Notifier
	current    string
	stop       chan struct{}
}

// NewMonitor creates a disk space monitor. interval is the check frequency.
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

// Start begins periodic checking.
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

// Stop halts monitoring.
func (m *Monitor) Stop() {
	close(m.stop)
}

func (m *Monitor) check() {
	// Use statfs-equivalent: check free space on the working directory
	pct := getDiskUsage()
	for i := len(m.thresholds) - 1; i >= 0; i-- {
		if pct >= m.thresholds[i].Pct {
			m.mu.Lock()
			if m.current != m.thresholds[i].Name {
				m.current = m.thresholds[i].Name
				m.mu.Unlock()
				if m.notify != nil {
					m.notify(m.thresholds[i])
				}
				return
			}
			m.mu.Unlock()
			return
		}
	}
	m.mu.Lock()
	if m.current != "" {
		m.current = ""
		m.mu.Unlock()
		if m.notify != nil {
			m.notify(Threshold{Name: "recovered", Pct: pct})
		}
		return
	}
	m.mu.Unlock()
}

// getDiskUsage is a placeholder for OS-specific disk usage queries.
func getDiskUsage() float64 {
	// Placeholder: in production this uses syscall.Statfs or equivalent
	// Returns 0.0 if unavailable (no monitor firing)
	return 0.0
}
