package config

import (
	"os"
	"sync"
	"time"
)

// Watcher monitors reasonix.toml for external changes and reloads the config.
// Settings panels can use this to avoid clobbering changes made by another
// frontend (CLI, desktop, another tab).
type Watcher struct {
	mu       sync.Mutex
	path     string
	modTime  time.Time
	interval time.Duration
	onReload func()
	stop     chan struct{}
}

// NewWatcher creates a config file watcher.
func NewWatcher(path string, interval time.Duration, onReload func()) *Watcher {
	if interval <= 0 {
		interval = 5 * time.Second
	}
	return &Watcher{
		path:     path,
		interval: interval,
		onReload: onReload,
		stop:     make(chan struct{}),
	}
}

// Start begins periodic monitoring.
func (w *Watcher) Start() {
	w.refreshModTime()
	go func() {
		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()
		for {
			select {
			case <-w.stop:
				return
			case <-ticker.C:
				w.check()
			}
		}
	}()
}

// Stop halts monitoring.
func (w *Watcher) Stop() {
	close(w.stop)
}

func (w *Watcher) refreshModTime() {
	info, err := os.Stat(w.path)
	if err == nil {
		w.modTime = info.ModTime()
	}
}

func (w *Watcher) check() {
	info, err := os.Stat(w.path)
	if err != nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if info.ModTime().After(w.modTime) {
		w.modTime = info.ModTime()
		if w.onReload != nil {
			w.onReload()
		}
	}
}
