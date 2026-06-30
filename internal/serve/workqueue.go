package serve

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// WorkQueue manages the server-side work queue for remote workers.
// Work items are created by the task tool when target is set to a remote name.
type WorkQueue struct {
	mu     sync.Mutex
	items  map[string]*workItem
	order  []string
	seq    int
	claims map[string]string // itemID → workerID that claimed it
}

type workItem struct {
	ID         string    `json:"id"`
	RemoteName string    `json:"remote_name"`
	RepoURL    string    `json:"repo_url"`
	Branch     string    `json:"branch"`
	Prompt     string    `json:"prompt"`
	Model      string    `json:"model"`
	Effort     string    `json:"effort"`
	MaxSteps   int       `json:"max_steps"`
	TimeoutSec int       `json:"timeout_sec"`
	Status     string    `json:"status"` // pending | claimed | running | completed | failed
	Result     string    `json:"result,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	ClaimedAt  time.Time `json:"claimed_at,omitempty"`
	ClaimedBy  string    `json:"claimed_by,omitempty"`
	expiresAt  time.Time // unclaimed items expire
}

// NewWorkQueue creates an empty work queue.
func NewWorkQueue() *WorkQueue {
	return &WorkQueue{
		items:  make(map[string]*workItem),
		claims: make(map[string]string),
	}
}

// Enqueue adds a work item to the queue and returns its ID.
func (q *WorkQueue) Enqueue(remoteName, repoURL, branch, prompt, model, effort string, maxSteps, timeoutSec int) string {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.seq++
	id := fmt.Sprintf("wrk-%s-%d", remoteName, q.seq)
	item := &workItem{
		ID:         id,
		RemoteName: remoteName,
		RepoURL:    repoURL,
		Branch:     branch,
		Prompt:     prompt,
		Model:      model,
		Effort:     effort,
		MaxSteps:   maxSteps,
		TimeoutSec: timeoutSec,
		Status:     "pending",
		CreatedAt:  time.Now(),
	}
	q.items[id] = item
	q.order = append(q.order, id)
	return id
}

// Pending returns all pending work items for the given remote name.
// This implements long-poll: if no items are available, it blocks for up
// to the given timeout.
func (q *WorkQueue) Pending(remoteName string) []workItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	var pending []workItem
	for _, id := range q.order {
		item := q.items[id]
		if item.RemoteName == remoteName && item.Status == "pending" {
			pending = append(pending, *item)
		}
	}
	return pending
}

// Claim atomically claims a work item. Returns an error if already claimed.
func (q *WorkQueue) Claim(id, workerID string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	item, ok := q.items[id]
	if !ok {
		return fmt.Errorf("work item %q not found", id)
	}
	if item.Status != "pending" {
		return fmt.Errorf("work item %q already claimed", id)
	}
	item.Status = "claimed"
	item.ClaimedAt = time.Now()
	item.ClaimedBy = workerID
	q.claims[id] = workerID
	return nil
}

// Complete marks a work item as completed with its result.
func (q *WorkQueue) Complete(id, result, status string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	item, ok := q.items[id]
	if !ok {
		return fmt.Errorf("work item %q not found", id)
	}
	item.Status = status
	item.Result = result
	delete(q.claims, id)
	return nil
}

// Stats returns queue statistics.
type WorkQueueStats struct {
	Pending  int `json:"pending"`
	Claimed  int `json:"claimed"`
	Running  int `json:"running"`
	Complete int `json:"complete"`
	Failed   int `json:"failed"`
}

func (q *WorkQueue) Stats() WorkQueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()
	var s WorkQueueStats
	for _, item := range q.items {
		switch item.Status {
		case "pending":
			s.Pending++
		case "claimed":
			s.Claimed++
		case "running":
			s.Running++
		case "completed":
			s.Complete++
		case "failed":
			s.Failed++
		}
	}
	return s
}

// workQueueHandler is the HTTP handler for work queue endpoints.
type workQueueHandler struct {
	queue      *WorkQueue
	authTokens map[string]bool
}

func (h *workQueueHandler) checkAuth(r *http.Request) bool {
	if len(h.authTokens) == 0 {
		return true // no auth configured
	}
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return h.authTokens[token]
}

func (h *workQueueHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !h.checkAuth(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/v1/work/")
	path = strings.TrimSuffix(path, "/")

	switch {
	case path == "pending" && r.Method == "GET":
		h.handlePending(w, r)
	case path == "stats" && r.Method == "GET":
		h.handleStats(w, r)
	case path == "health" && r.Method == "POST":
		w.WriteHeader(http.StatusOK)
	case strings.HasSuffix(path, "/claim") && r.Method == "POST":
		h.handleClaim(w, r, strings.TrimSuffix(path, "/claim"))
	case strings.HasSuffix(path, "/events") && r.Method == "POST":
		w.WriteHeader(http.StatusOK)
	case strings.HasSuffix(path, "/complete") && r.Method == "POST":
		h.handleComplete(w, r, strings.TrimSuffix(path, "/complete"))
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *workQueueHandler) handlePending(w http.ResponseWriter, r *http.Request) {
	remoteName := r.URL.Query().Get("remote")
	if remoteName == "" {
		remoteName = "*"
	}
	items := h.queue.Pending(remoteName)
	if len(items) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}

func (h *workQueueHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.queue.Stats())
}

func (h *workQueueHandler) handleClaim(w http.ResponseWriter, r *http.Request, id string) {
	workerID := r.URL.Query().Get("worker")
	if workerID == "" {
		workerID = r.Header.Get("X-Worker-Name")
	}
	if err := h.queue.Claim(id, workerID); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "claimed"})
}

func (h *workQueueHandler) handleComplete(w http.ResponseWriter, r *http.Request, id string) {
	var body struct {
		Result string `json:"result"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	if body.Status == "" {
		body.Status = "completed"
	}
	if err := h.queue.Complete(id, body.Result, body.Status); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}
