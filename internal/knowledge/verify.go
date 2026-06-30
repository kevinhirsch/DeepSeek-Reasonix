package knowledge

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// VerifyResult captures the outcome of a cross-validation pass.
type VerifyResult struct {
	ID          string    `json:"id"`
	Passed      bool      `json:"passed"`
	Score       float64   `json:"score"`
	Model       string    `json:"model"`
	ReviewedAt  time.Time `json:"reviewed_at"`
	Conflicts   []string  `json:"conflicts,omitempty"`
	Notes       string    `json:"notes,omitempty"`
	IsStale     bool      `json:"is_stale"`
}

// VerifyPass records the outcome of a single verification pass against a
// knowledge entry. Multiple passes from different model versions build
// confidence; a pass from a newer model marks older passes as potentially
// stale.
type VerifyPass struct {
	EntryID   string    `json:"entry_id"`
	Model     string    `json:"model"`
	Score     float64   `json:"score"`
	Passed    bool      `json:"passed"`
	Conflicts []string  `json:"conflicts,omitempty"`
	Notes     string    `json:"notes,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// VerifyOptions controls verification behavior.
type VerifyOptions struct {
	// MinConfidence is the minimum confidence threshold before an entry is
	// stored. Entries below this threshold are discarded. Default 0.6.
	MinConfidence float64

	// RequireModelTag forces entries to carry a Model version tag.
	// Empty-model entries are rejected when true.
	RequireModelTag bool

	// CrossVerifyModel is an alternate model version to re-verify against.
	// When set, entries are reviewed by a second model and must agree within
	// the AgreementThreshold.
	CrossVerifyModel string

	// AgreementThreshold is the minimum score delta allowed between the
	// primary model and the cross-verify model. Default 0.2.
	AgreementThreshold float64

	// StaleAfter marks entries not accessed within this duration as stale.
	// A zero duration disables staleness checking.
	StaleAfter time.Duration

	// MaxConflictAge removes conflict flags older than this duration.
	MaxConflictAge time.Duration
}

func (o *VerifyOptions) defaults() {
	if o.MinConfidence <= 0 {
		o.MinConfidence = 0.6
	}
	if o.AgreementThreshold <= 0 {
		o.AgreementThreshold = 0.2
	}
}

// VerifyStore extends Store with cross-validation and staleness tracking.
type VerifyStore struct {
	*Store
	mu    sync.RWMutex
	passes map[string][]VerifyPass // entry ID -> verification passes
	opts   VerifyOptions
}

// NewVerifyStore wraps an existing Store with verification capabilities.
func NewVerifyStore(s *Store, opts VerifyOptions) *VerifyStore {
	opts.defaults()
	return &VerifyStore{
		Store:  s,
		passes: make(map[string][]VerifyPass),
		opts:   opts,
	}
}

// VerifyEntry cross-validates a candidate entry before it is stored.
// It runs these checks in order:
//   1. Model tag required check
//   2. Confidence floor check
//   3. Cross-verify against a second model pass (if configured)
// Returns a VerifyResult indicating pass/fail and identified conflicts.
func (v *VerifyStore) VerifyEntry(e *Entry, crossVerifyFn func(question, answer string) (float64, []string, error)) (*VerifyResult, error) {
	vr := &VerifyResult{
		ID:         e.ID,
		Model:      e.Model,
		ReviewedAt: time.Now(),
		Passed:     true,
	}

	// Gate 1: model tag.
	if v.opts.RequireModelTag && strings.TrimSpace(e.Model) == "" {
		vr.Passed = false
		vr.Conflicts = append(vr.Conflicts, "missing model version tag")
		vr.Notes = "entry rejected: model tag required"
		return vr, nil
	}

	// Gate 2: confidence floor.
	if e.Confidence < v.opts.MinConfidence {
		vr.Passed = false
		vr.Conflicts = append(vr.Conflicts,
			fmt.Sprintf("confidence %.2f below floor %.2f", e.Confidence, v.opts.MinConfidence))
		vr.Notes = "entry rejected: confidence below floor"
		return vr, nil
	}

	// Gate 3: cross-verify against a second model.
	if v.opts.CrossVerifyModel != "" && crossVerifyFn != nil {
		crossScore, conflicts, err := crossVerifyFn(e.Question, e.Answer)
		if err != nil {
			vr.Passed = false
			vr.Notes = fmt.Sprintf("cross-verify error: %v", err)
			return vr, nil
		}
		delta := e.Confidence - crossScore
		if delta < 0 {
			delta = -delta
		}
		if delta > v.opts.AgreementThreshold {
			vr.Passed = false
			vr.Conflicts = append(vr.Conflicts, fmt.Sprintf(
				"model divergence: primary=%.2f cross=%.2f delta=%.2f > threshold=%.2f",
				e.Confidence, crossScore, delta, v.opts.AgreementThreshold))
		}
		vr.Score = (e.Confidence + crossScore) / 2

		// Record the cross-verify pass.
		v.mu.Lock()
		v.passes[e.ID] = append(v.passes[e.ID], VerifyPass{
			EntryID:   e.ID,
			Model:     v.opts.CrossVerifyModel,
			Score:     crossScore,
			Passed:    delta <= v.opts.AgreementThreshold,
			Conflicts: conflicts,
			CreatedAt: time.Now(),
		})
		v.mu.Unlock()
	}

	return vr, nil
}

// AddWithVerification combines cross-validation with storage.
// If verification fails, the entry is not stored and the VerifyResult
// is returned. On pass, the entry is stored and a primary verification
// pass is recorded.
func (v *VerifyStore) AddWithVerification(q, answer string, files []string, model string, confidence float64, crossVerifyFn func(question, answer string) (float64, []string, error)) (*Entry, *VerifyResult, error) {
	// Build a temporary entry for verification.
	e := &Entry{
		Question:   q,
		Answer:     answer,
		Files:      files,
		Confidence: confidence,
		Model:      model,
	}

	vr, err := v.VerifyEntry(e, crossVerifyFn)
	if err != nil {
		return nil, nil, err
	}

	if !vr.Passed {
		return nil, vr, nil
	}

	// Store the entry.
	stored := v.Store.Add(q, answer, files, model, confidence)

	// Record the primary pass.
	v.mu.Lock()
	v.passes[stored.ID] = append(v.passes[stored.ID], VerifyPass{
		EntryID:   stored.ID,
		Model:     model,
		Score:     confidence,
		Passed:    true,
		CreatedAt: time.Now(),
	})
	v.mu.Unlock()

	return stored, vr, nil
}

// CheckStale examines an entry on access and returns whether it is stale
// and the reason. An entry is stale when:
//   - It has not been accessed within StaleAfter.
//   - Its primary verification model is out of date (newer model available).
//   - Conflict flags are older than MaxConflictAge.
func (v *VerifyStore) CheckStale(e *Entry) (bool, string) {
	if v.opts.StaleAfter > 0 {
		if time.Since(e.AccessedAt) > v.opts.StaleAfter {
			return true, fmt.Sprintf("last accessed %s ago, threshold %s",
				time.Since(e.AccessedAt).Round(time.Second), v.opts.StaleAfter)
		}
	}

	// Check if the entry's model is out of date relative to any passes.
	v.mu.RLock()
	passes := v.passes[e.ID]
	v.mu.RUnlock()

	if len(passes) > 1 {
		var newestPass *VerifyPass
		for i := range passes {
			if newestPass == nil || passes[i].CreatedAt.After(newestPass.CreatedAt) {
				newestPass = &passes[i]
			}
		}
		if newestPass != nil && newestPass.Model != "" && newestPass.Model != e.Model {
			return true, fmt.Sprintf("newer verification available from model %s (%s ago)",
				newestPass.Model, time.Since(newestPass.CreatedAt).Round(time.Second))
		}
	}

	// Check conflict age.
	if v.opts.MaxConflictAge > 0 {
		for _, p := range passes {
			if !p.Passed && time.Since(p.CreatedAt) > v.opts.MaxConflictAge {
				return false, "" // conflicts aged out; entry is no longer considered stale from them
			}
		}
	}

	return false, ""
}

// ReVerify re-runs verification from a different model version and updates
// the entry's confidence to the average of all passing verification scores.
func (v *VerifyStore) ReVerify(e *Entry, model string, crossVerifyFn func(question, answer string) (float64, []string, error)) (*VerifyPass, error) {
	if crossVerifyFn == nil {
		return nil, fmt.Errorf("crossVerifyFn is nil")
	}

	score, conflicts, err := crossVerifyFn(e.Question, e.Answer)
	if err != nil {
		return nil, fmt.Errorf("re-verify error for %s: %w", e.ID, err)
	}

	pass := VerifyPass{
		EntryID:   e.ID,
		Model:     model,
		Score:     score,
		Passed:    score >= v.opts.MinConfidence,
		Conflicts: conflicts,
		CreatedAt: time.Now(),
	}

	v.mu.Lock()
	v.passes[e.ID] = append(v.passes[e.ID], pass)
	// Update entry confidence to the mean of all passing scores.
	var sum float64
	count := 0
	for _, p := range v.passes[e.ID] {
		if p.Passed {
			sum += p.Score
			count++
		}
	}
	if count > 0 {
		e.Confidence = sum / float64(count)
	}
	v.mu.Unlock()

	// If this pass flagged conflicts with a passing current entry, lower confidence.
	if len(conflicts) > 0 {
		e.Confidence *= 0.8 // 20% penalty for identified conflicts
	}

	return &pass, nil
}

// VerificationPasses returns all verification passes for an entry.
func (v *VerifyStore) VerificationPasses(entryID string) []VerifyPass {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]VerifyPass, len(v.passes[entryID]))
	copy(out, v.passes[entryID])
	return out
}

// StaleEntries returns all entries marked as stale.
func (v *VerifyStore) StaleEntries() []*Entry {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var stale []*Entry
	for _, e := range v.Store.entries {
		if isStale, _ := v.CheckStale(e); isStale {
			stale = append(stale, e)
		}
	}
	return stale
}

// CleanStale removes entries that are stale and returns the count removed.
func (v *VerifyStore) CleanStale() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	var toRemove []string
	for id, e := range v.Store.entries {
		if isStale, _ := v.CheckStale(e); isStale {
			toRemove = append(toRemove, id)
		}
	}
	for _, id := range toRemove {
		delete(v.Store.entries, id)
		delete(v.passes, id)
	}
	return len(toRemove)
}

// VerifyStats provides aggregate verification statistics for observability.
type VerifyStats struct {
	TotalEntries      int     `json:"total_entries"`
	VerifiedEntries   int     `json:"verified_entries"`
	StaleEntries      int     `json:"stale_entries"`
	MeanConfidence    float64 `json:"mean_confidence"`
	ConflictCount     int     `json:"conflict_count"`
	LastReVerifyModel string  `json:"last_reverify_model,omitempty"`
}

// Stats returns aggregate verification statistics.
func (v *VerifyStore) Stats() VerifyStats {
	v.mu.RLock()
	defer v.mu.RUnlock()

	s := VerifyStats{}
	for id, e := range v.Store.entries {
		s.TotalEntries++
		s.MeanConfidence += e.Confidence
		if passes, ok := v.passes[id]; ok && len(passes) > 0 {
			s.VerifiedEntries++
			for _, p := range passes {
				if !p.Passed {
					s.ConflictCount++
				}
			}
		}
	}
	if s.TotalEntries > 0 {
		s.MeanConfidence /= float64(s.TotalEntries)
	}
	s.StaleEntries = len(v.StaleEntries())
	return s
}
