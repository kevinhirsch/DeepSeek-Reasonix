package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// WorkItem is a pending work item from the server queue.
type WorkItem struct {
	ID          string `json:"id"`
	RepoURL     string `json:"repo_url"`
	Branch      string `json:"branch"`
	Prompt      string `json:"prompt"`
	Model       string `json:"model"`
	Effort      string `json:"effort"`
	MaxSteps    int    `json:"max_steps"`
	TimeoutSec  int    `json:"timeout_sec"`
	CreatedAt   string `json:"created_at"`
}

// Poller handles long-polling the server's work queue.
type Poller struct {
	serverURL  string
	authToken  string
	client     *http.Client
	pollTimeout time.Duration
}

// NewPoller creates a work queue poller.
func NewPoller(serverURL, authToken string, pollTimeout time.Duration) *Poller {
	return &Poller{
		serverURL:  serverURL,
		authToken:  authToken,
		client:     &http.Client{Timeout: pollTimeout + 10*time.Second},
		pollTimeout: pollTimeout,
	}
}

// Poll blocks until a work item is available or the context is cancelled.
// Returns nil, nil when the poll times out (caller should re-poll).
func (p *Poller) Poll(ctx context.Context) (*WorkItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET",
		p.serverURL+"/v1/work/pending?timeout="+fmt.Sprint(p.pollTimeout.Seconds())+"s", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("poll work queue: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var items []WorkItem
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			return nil, fmt.Errorf("decode work items: %w", err)
		}
		if len(items) == 0 {
			return nil, nil
		}
		return &items[0], nil
	case http.StatusNoContent:
		return nil, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized: check auth_token in worker config")
	default:
		return nil, fmt.Errorf("server returned %d", resp.StatusCode)
	}
}

// Claim atomically claims a work item. Returns an error if another worker
// already claimed it.
func (p *Poller) Claim(ctx context.Context, id string) error {
	req, err := http.NewRequestWithContext(ctx, "POST",
		p.serverURL+"/v1/work/"+id+"/claim", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("claim work item: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("claim failed: %d", resp.StatusCode)
	}
	return nil
}

// Complete reports a work item as completed with its result.
func (p *Poller) Complete(ctx context.Context, id, result string) error {
	jsonBody, _ := json.Marshal(map[string]string{"result": result})
	req, err := http.NewRequestWithContext(ctx, "POST",
		p.serverURL+"/v1/work/"+id+"/complete",
		strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+p.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("complete work item: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
