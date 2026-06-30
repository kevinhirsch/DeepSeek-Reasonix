// Package agent — compensation_chain.go: trigger-chain tracking for compensations.
//
// When one compensation's output triggers another compensation (e.g., speculative
// execution flags an ambiguity -> cross-validation fires), the chain is recorded
// here. Chains are capped at depth 5 to prevent runaway cascades, and each step
// is logged for cost analysis so teams can identify expensive trigger paths and
// tune or disable them.
package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// ChainStep is one link in a compensation trigger chain.
type ChainStep struct {
	// Depth is the 1-based position in the chain (1 = origin).
	Depth int `json:"depth"`

	// TriggerCompensation is the name of the compensation that was running
	// when the trigger fired.
	TriggerCompensation string `json:"trigger_compensation"`

	// TriggeredCompensation is the name of the compensation that was triggered
	// in response.
	TriggeredCompensation string `json:"triggered_compensation"`

	// Reason is a human-readable explanation of why the trigger fired (e.g.,
	// "speculative output ambiguity threshold exceeded").
	Reason string `json:"reason"`

	// Timestamp is when this chain step was recorded.
	Timestamp time.Time `json:"timestamp"`

	// SubagentID is the subagent whose turn produced this trigger (empty if
	// triggered from the parent agent).
	SubagentID string `json:"subagent_id,omitempty"`

	// TokenCost is the estimated tokens consumed by the triggered compensation
	// (populated retroactively when the triggered compensation completes).
	TokenCost int `json:"token_cost,omitempty"`
}

// Chain is a complete compensation trigger chain. A chain starts when the first
// compensation is triggered (its output becomes the first step's trigger) and
// grows as each step fires further compensations. Depth is capped at 5.
type Chain struct {
	// ID uniquely identifies the chain for log correlation.
	ID string `json:"id"`

	// RootCause is the original event that started the chain (e.g., "user turn",
	// "subagent result processing").
	RootCause string `json:"root_cause"`

	// StartedAt is when the first step was recorded.
	StartedAt time.Time `json:"started_at"`

	// EndedAt is when the chain was closed (depth cap reached or no further
	// triggers). Zero if still open.
	EndedAt time.Time `json:"ended_at,omitempty"`

	// Steps is the ordered list of trigger steps, oldest first.
	Steps []ChainStep `json:"steps"`

	// TotalTokenCost is the sum of all TokenCost values in steps.
	TotalTokenCost int `json:"total_token_cost"`

	// Capped is true when the chain reached the maximum depth and was forcibly
	// terminated.
	Capped bool `json:"capped"`
}

// CompensationChainTracker records compensation trigger chains and enforces the
// maximum depth cap of 5. It is the single source of truth for understanding
// which compensations cascade into others and what the total cost of each chain
// is.
type CompensationChainTracker struct {
	mu     sync.Mutex
	chains []*Chain
	active map[string]*Chain // chain ID -> chain (currently open)
	closed []*Chain          // completed chains

	maxDepth int
}

// NewCompensationChainTracker creates a chain tracker with the default depth cap
// of 5.
func NewCompensationChainTracker() *CompensationChainTracker {
	return &CompensationChainTracker{
		maxDepth: 5,
		active:   make(map[string]*Chain),
	}
}

// WithMaxDepth overrides the default depth cap. The caller must ensure the value
// is positive.
func (t *CompensationChainTracker) WithMaxDepth(depth int) *CompensationChainTracker {
	if depth > 0 {
		t.maxDepth = depth
	}
	return t
}

// BeginChain starts a new compensation trigger chain and returns its ID. The
// rootCause describes what initiated the chain (e.g., "user turn 3",
// "subagent ref=abc123 result processing").
func (t *CompensationChainTracker) BeginChain(rootCause string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	chain := &Chain{
		ID:        fmt.Sprintf("chain-%d-%s", time.Now().UnixNano(), randomSuffixChain(6)),
		RootCause: rootCause,
		StartedAt: time.Now().UTC(),
	}
	t.chains = append(t.chains, chain)
	t.active[chain.ID] = chain
	return chain.ID
}

// RecordStep appends a step to an existing chain. Returns false with an error
// if the chain is unknown or has already reached the depth cap (in which case
// the step is not recorded).
//
// When the depth cap is reached, the chain is automatically closed with
// Capped = true, and the caller should NOT fire the triggered compensation.
func (t *CompensationChainTracker) RecordStep(chainID string, step ChainStep) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	chain, ok := t.active[chainID]
	if !ok {
		return false, fmt.Errorf("unknown chain %q", chainID)
	}

	step.Depth = len(chain.Steps) + 1
	if step.Timestamp.IsZero() {
		step.Timestamp = time.Now().UTC()
	}

	if step.Depth > t.maxDepth {
		chain.Capped = true
		chain.EndedAt = time.Now().UTC()
		delete(t.active, chainID)
		t.closed = append(t.closed, chain)
		return false, fmt.Errorf(
			"chain %q depth cap (%d) reached; step from %s -> %s not recorded",
			chainID, t.maxDepth, step.TriggerCompensation, step.TriggeredCompensation)
	}

	chain.Steps = append(chain.Steps, step)

	// If we just hit depth cap on this step, still record it but mark capped.
	if step.Depth == t.maxDepth {
		chain.Capped = true
	}

	return true, nil
}

// CloseChain finalises a chain that completed normally (no cap reached). Sets
// the end time and moves it from active to closed.
func (t *CompensationChainTracker) CloseChain(chainID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	chain, ok := t.active[chainID]
	if !ok {
		return
	}
	chain.EndedAt = time.Now().UTC()
	delete(t.active, chainID)
	t.closed = append(t.closed, chain)
}

// UpdateStepCost retroactively sets the token cost on a chain step. The caller
// provides the chain ID and step index (0-based).
func (t *CompensationChainTracker) UpdateStepCost(chainID string, stepIndex int, tokenCost int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Check active and closed chains.
	for _, chains := range [][]*Chain{t.activeValues(), t.closed} {
		for _, chain := range chains {
			if chain.ID == chainID && stepIndex >= 0 && stepIndex < len(chain.Steps) {
				chain.Steps[stepIndex].TokenCost = tokenCost
				// Recompute total.
				total := 0
				for _, s := range chain.Steps {
					total += s.TokenCost
				}
				chain.TotalTokenCost = total
				return
			}
		}
	}
}

// activeValues returns a slice of active chains while the caller holds the lock.
func (t *CompensationChainTracker) activeValues() []*Chain {
	vals := make([]*Chain, 0, len(t.active))
	for _, c := range t.active {
		vals = append(vals, c)
	}
	return vals
}

// ActiveChains returns the currently open chains.
func (t *CompensationChainTracker) ActiveChains() []*Chain {
	t.mu.Lock()
	defer t.mu.Unlock()
	vals := make([]*Chain, 0, len(t.active))
	for _, c := range t.active {
		cp := *c
		vals = append(vals, &cp)
	}
	return vals
}

// ClosedChains returns completed chains, most recent first.
func (t *CompensationChainTracker) ClosedChains() []*Chain {
	t.mu.Lock()
	defer t.mu.Unlock()
	vals := make([]*Chain, len(t.closed))
	for i, c := range t.closed {
		cp := *c
		vals[i] = &cp
	}
	// Most recent first.
	sort.Slice(vals, func(i, j int) bool {
		return vals[i].EndedAt.After(vals[j].EndedAt)
	})
	return vals
}

// AllChains returns every chain (active + closed), most recently started first.
func (t *CompensationChainTracker) AllChains() []*Chain {
	t.mu.Lock()
	defer t.mu.Unlock()

	all := make([]*Chain, 0, len(t.active)+len(t.closed))
	for _, c := range t.active {
		cp := *c
		all = append(all, &cp)
	}
	for _, c := range t.closed {
		cp := *c
		all = append(all, &cp)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].StartedAt.After(all[j].StartedAt)
	})
	return all
}

// CostSummary returns a breakdown of total tokens and number of chains per
// trigger compensation. Useful for identifying which compensations are the most
// expensive cascade starters.
func (t *CompensationChainTracker) CostSummary() map[string]ChainCostSummary {
	t.mu.Lock()
	defer t.mu.Unlock()

	summary := make(map[string]ChainCostSummary)
	all := append([]*Chain(nil), t.activeValues()...)
	all = append(all, t.closed...)

	for _, chain := range all {
		for _, step := range chain.Steps {
			s := summary[step.TriggerCompensation]
			if s.TriggersInto == nil {
				s.TriggersInto = make(map[string]int)
			}
			s.TriggerCompensation = step.TriggerCompensation
			s.TriggerCount++
			s.TotalTokenCost += step.TokenCost
			if step.TriggeredCompensation != "" {
				s.TriggersInto[step.TriggeredCompensation]++
			}
			summary[step.TriggerCompensation] = s
		}
	}
	return summary
}

// ChainCostSummary is a per-compensation aggregation of chain costs.
type ChainCostSummary struct {
	TriggerCompensation string         `json:"trigger_compensation"`
	TriggerCount        int            `json:"trigger_count"`
	TotalTokenCost      int            `json:"total_token_cost"`
	TriggersInto        map[string]int `json:"triggers_into"`
}

// CappedChains reports which chains were terminated due to depth cap, with the
// original root cause and the last step that was recorded.
func (t *CompensationChainTracker) CappedChains() []CapReport {
	t.mu.Lock()
	defer t.mu.Unlock()

	var reports []CapReport
	for _, chain := range t.closed {
		if !chain.Capped {
			continue
		}
		r := CapReport{
			ChainID:   chain.ID,
			RootCause: chain.RootCause,
			Depth:     len(chain.Steps),
		}
		if len(chain.Steps) > 0 {
			last := chain.Steps[len(chain.Steps)-1]
			r.LastStep = fmt.Sprintf("%s -> %s", last.TriggerCompensation, last.TriggeredCompensation)
		}
		reports = append(reports, r)
	}
	return reports
}

// CapReport describes a chain that was forcibly terminated.
type CapReport struct {
	ChainID   string `json:"chain_id"`
	RootCause string `json:"root_cause"`
	Depth     int    `json:"depth"`
	LastStep  string `json:"last_step"`
}

// LogChainForCostAnalysis formats a chain as a single-line log entry suitable
// for structured logging and cost analysis dashboards. Each line contains the
// chain ID, depth, total tokens, and a compact step trace.
func LogChainForCostAnalysis(chain *Chain) string {
	var stepStrs []string
	for _, s := range chain.Steps {
		stepStrs = append(stepStrs, fmt.Sprintf("%s->%s(%d)",
			s.TriggerCompensation, s.TriggeredCompensation, s.TokenCost))
	}
	capped := ""
	if chain.Capped {
		capped = " [CAPPED]"
	}
	return fmt.Sprintf("compensation-chain id=%s root=%s depth=%d tokens=%d trace=%s%s",
		chain.ID, chain.RootCause, len(chain.Steps), chain.TotalTokenCost,
		strings.Join(stepStrs, ","), capped)
}

// randomSuffixChain produces a short random-looking string for chain ID
// uniqueness without importing crypto.
func randomSuffixChain(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	ns := time.Now().UnixNano()
	for i := range b {
		b[i] = letters[int(ns>>uint(i*5))%len(letters)]
	}
	return string(b)
}

// compensationChainCtxKey is the context key for a CompensationChainTracker.
type compensationChainCtxKey struct{}

// WithChainTracker attaches a CompensationChainTracker to the context so
// compensations can record trigger steps without a global singleton.
func WithChainTracker(ctx context.Context, t *CompensationChainTracker) context.Context {
	return context.WithValue(ctx, compensationChainCtxKey{}, t)
}

// ChainTrackerFromContext retrieves the CompensationChainTracker from a context,
// or nil if none was attached.
func ChainTrackerFromContext(ctx context.Context) *CompensationChainTracker {
	t, _ := ctx.Value(compensationChainCtxKey{}).(*CompensationChainTracker)
	return t
}
