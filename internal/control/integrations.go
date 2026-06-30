package control

import (
	"context"
	"log"
	"time"

	"reasonix/internal/idle"
	"reasonix/internal/knowledge"
	"reasonix/internal/watchdog"
)

// Controller fields for integrations (added to struct definition in controller.go).
// idleDetector    *idle.Detector
// cleanupFns      []func()
// watchdogCancel  context.CancelFunc

// WireIntegrations initializes idle detection and dependency watching.
// Call from Controller constructor after jobs.Manager is set.
func WireIntegrations(c *Controller, kb *knowledge.Store, workspaceRoot string) {
	// --- Idle detection ---
	d := idle.NewDetector(2*time.Minute, 2*time.Second)
	d.Start(func() {
		log.Println("[idle] starting KB precomputation...")
		if kb != nil && kb.Size() == 0 {
			log.Println("[idle] KB empty — nothing to precompute")
		}
	})
	c.idleDetector = d

	// --- Dependency watchdog ---
	ctx, cancel := context.WithCancel(context.Background())
	c.watchdogCancel = cancel
	ch := watchdog.NewDependencyChecker(workspaceRoot)

	go func() {
		ticker := time.NewTicker(7 * 24 * time.Hour) // weekly
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				summary, err := ch.CheckAll(ctx)
				if err != nil {
					log.Printf("[watchdog] check error: %v", err)
					continue
				}
				for _, r := range summary.Reports {
					if r.Status == "safe" {
						log.Printf("[watchdog] %s %s→%s: safe", r.Dependency, r.CurrentVersion, r.LatestVersion)
					} else if r.Error != "" {
						log.Printf("[watchdog] %s %s→%s: %s", r.Dependency, r.CurrentVersion, r.LatestVersion, r.Error)
					}
				}
				log.Printf("[watchdog] %d/%d safe, %d break", summary.Safe, summary.Total, summary.Breaks)
			}
		}
	}()

	// --- Cleanup chain ---
	c.cleanupFns = append(c.cleanupFns,
		func() { d.Stop() },
		func() { cancel() },
	)
}

// TouchIdle signals user activity. Call from Controller.Run() before turn.
func TouchIdle(c *Controller) {
	if c.idleDetector != nil {
		c.idleDetector.Touch()
	}
}

// StopWatchdog cancels the watchdog goroutine.
func StopWatchdog(c *Controller) {
	if c.watchdogCancel != nil {
		c.watchdogCancel()
	}
}

// RunCleanup runs all registered cleanup functions.
func RunCleanup(c *Controller) {
	for _, fn := range c.cleanupFns {
		fn()
	}
}
