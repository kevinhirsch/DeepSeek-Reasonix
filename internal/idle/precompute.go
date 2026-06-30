// Package idle manages idle-time pre-computation: when reasonix is idle for
// a configurable duration, it spawns explorer subagents to build understanding
// of the most-referenced unexplored parts of the codebase.
package idle

import (
	"context"
	"sync"
	"time"
)

// Detector watches for idle periods and triggers callbacks.
type Detector struct {
	idleAfter     time.Duration
	debounceAfter time.Duration

	mu            sync.Mutex
	lastActivity  time.Time
	active        bool
	running       bool
	onIdle        func()
	stop          chan struct{}
}

// NewDetector creates an idle detector. idleAfter is how long to wait before
// considering the system idle. debounceAfter is the grace period after activity
// before the idle timer restarts (prevents spawn/cancel churn).
func NewDetector(idleAfter, debounceAfter time.Duration) *Detector {
	if idleAfter <= 0 {
		idleAfter = 2 * time.Minute
	}
	if debounceAfter <= 0 {
		debounceAfter = 2 * time.Second
	}
	return &Detector{
		idleAfter:     idleAfter,
		debounceAfter: debounceAfter,
		lastActivity:  time.Now(),
		stop:          make(chan struct{}),
	}
}

// Touch records user activity, resetting the idle timer.
func (d *Detector) Touch() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastActivity = time.Now()
}

// Start begins monitoring for idle periods. Calls onIdle when idle is detected.
// Only one idle callback runs at a time.
func (d *Detector) Start(onIdle func()) {
	d.mu.Lock()
	d.onIdle = onIdle
	d.mu.Unlock()

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-d.stop:
				return
			case <-ticker.C:
				d.tick()
			}
		}
	}()
}

func (d *Detector) tick() {
	d.mu.Lock()
	elapsed := time.Since(d.lastActivity)
	d.mu.Unlock()

	if elapsed < d.debounceAfter {
		d.mu.Lock()
		d.active = false
		d.mu.Unlock()
		return
	}

	d.mu.Lock()
	if !d.active && elapsed >= d.idleAfter && !d.running && d.onIdle != nil {
		d.active = true
		d.running = true
		d.mu.Unlock()
		d.onIdle()
		d.mu.Lock()
		d.running = false
	}
	d.mu.Unlock()
}

// Stop cancels idle monitoring.
func (d *Detector) Stop() {
	close(d.stop)
}
