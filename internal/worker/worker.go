package worker

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Worker is the main daemon process that polls the server work queue, claims
// items, executes them in Docker containers, and streams events back.
type Worker struct {
	config   *Config
	poller   *Poller
	streamer *Streamer
	executor *Executor
	health   *HealthReporter

	mu        sync.Mutex
	activeJobs map[string]context.CancelFunc
}

// New creates a Worker from the given configuration.
func New(cfg *Config) *Worker {
	pollTimeout := time.Duration(cfg.PollTimeout) * time.Second
	healthInterval := time.Duration(cfg.HealthInterval) * time.Second
	return &Worker{
		config:     cfg,
		poller:     NewPoller(cfg.ServerURL, cfg.AuthToken, pollTimeout),
		streamer:   NewStreamer(cfg.ServerURL, cfg.AuthToken),
		executor:   NewExecutor(cfg),
		health:     NewHealthReporter(cfg.ServerURL, cfg.AuthToken, healthInterval),
		activeJobs: make(map[string]context.CancelFunc),
	}
}

// Run starts the main worker loop. Blocks until ctx is cancelled or a
// fatal error occurs.
func (w *Worker) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		log.Println("received shutdown signal, draining in-flight jobs...")
		cancel()
	}()

	startTime := time.Now()

	// Start health reporting
	go w.health.Run(ctx, w.config, startTime, w.activeJobCount)

	log.Printf("worker %q starting (server: %s, max_concurrent: %d)",
		w.config.Name, w.config.ServerURL, w.config.MaxConcurrent)

	// Main poll→claim→execute→stream loop
	for {
		select {
		case <-ctx.Done():
			log.Println("worker shutting down...")
			w.drainInflight()
			return ctx.Err()
		default:
		}

		// Wait for capacity
		for w.activeJobCount() >= w.config.MaxConcurrent {
			select {
			case <-ctx.Done():
				w.drainInflight()
				return ctx.Err()
			case <-time.After(1 * time.Second):
			}
		}

		// Poll for work
		item, err := w.poller.Poll(ctx)
		if err != nil {
			log.Printf("poll error: %v", err)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(5 * time.Second):
			}
			continue
		}
		if item == nil {
			continue // timeout, re-poll
		}

		// Claim it
		if err := w.poller.Claim(ctx, item.ID); err != nil {
			log.Printf("claim error for %q: %v", item.ID, err)
			continue
		}

		// Execute
		log.Printf("claimed work item %q: %s", item.ID, truncateStr(item.Prompt, 120))
		w.startJob(item)

		go func(it *WorkItem) {
			defer w.finishJob(it.ID)
			defer w.executor.Cleanup(it.ID)

			// Set timeout
			jobCtx, jobCancel := context.WithTimeout(context.Background(),
				time.Duration(it.TimeoutSec)*time.Second)
			defer jobCancel()
			w.setJobCancel(it.ID, jobCancel)

			// Stream start event
			w.streamer.Send(jobCtx, it.ID, StreamEvent{
				Kind: "started",
				Text: fmt.Sprintf("running: %s", truncateStr(it.Prompt, 80)),
			})

			result, err := w.executor.RunTask(jobCtx, it)

			if err != nil {
				log.Printf("task %q failed: %v", it.ID, err)
				w.streamer.Send(context.Background(), it.ID, StreamEvent{
					Kind:  "failed",
					Error: err.Error(),
				})
				w.poller.Complete(context.Background(), it.ID, fmt.Sprintf("FAILED: %v", err))
				return
			}

			// Stream completion
			w.streamer.Send(context.Background(), it.ID, StreamEvent{
				Kind: "completed",
				Text: truncateStr(result, 200),
			})
			w.poller.Complete(context.Background(), it.ID, result)
		}(item)
	}
}

func (w *Worker) startJob(item *WorkItem) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.activeJobs[item.ID] = nil
}

func (w *Worker) finishJob(id string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.activeJobs, id)
}

func (w *Worker) setJobCancel(id string, cancel context.CancelFunc) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.activeJobs[id] = cancel
}

func (w *Worker) activeJobCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.activeJobs)
}

func (w *Worker) drainInflight() {
	w.mu.Lock()
	defer w.mu.Unlock()
	for id, cancel := range w.activeJobs {
		if cancel != nil {
			log.Printf("cancelling job %q", id)
			cancel()
		}
	}
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
