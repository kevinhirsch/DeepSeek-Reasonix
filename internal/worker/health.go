package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// HealthReporter periodically reports worker health and capabilities to the server.
type HealthReporter struct {
	serverURL string
	authToken string
	interval  time.Duration
	client    *http.Client
}

// NewHealthReporter creates a health reporter.
func NewHealthReporter(serverURL, authToken string, interval time.Duration) *HealthReporter {
	return &HealthReporter{
		serverURL: serverURL,
		authToken: authToken,
		interval:  interval,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// HealthReport describes the worker's current state.
type HealthReport struct {
	Name          string `json:"name"`
	Version       string `json:"version"`
	OS            string `json:"os"`
	Arch          string `json:"arch"`
	CPUCores      int    `json:"cpu_cores"`
	MaxConcurrent int    `json:"max_concurrent"`
	Uptime        string `json:"uptime"`
	ActiveJobs    int    `json:"active_jobs"`
}

// Run sends health reports at the configured interval until ctx is cancelled.
func (h *HealthReporter) Run(ctx context.Context, cfg *Config, startTime time.Time, activeJobs func() int) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	report := HealthReport{
		Name:          cfg.Name,
		Version:       "0.1.0",
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		CPUCores:      runtime.NumCPU(),
		MaxConcurrent: cfg.MaxConcurrent,
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			report.Uptime = time.Since(startTime).Round(time.Second).String()
			report.ActiveJobs = activeJobs()
			h.send(ctx, report)
		}
	}
}

func (h *HealthReporter) send(ctx context.Context, report HealthReport) {
	body, _ := json.Marshal(report)
	req, err := http.NewRequestWithContext(ctx, "POST",
		h.serverURL+"/v1/work/health", nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+h.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
