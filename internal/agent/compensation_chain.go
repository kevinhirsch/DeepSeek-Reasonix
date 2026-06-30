package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// MaxChainDepth is the maximum number of cascading compensations allowed before
// the chain is capped. When compensation A triggers B, and B triggers C, and so
// on, depth 5 is the hard limit.
const MaxChainDepth = 5

// ChainLink records one link in a compensation trigger chain: compensation A
// (the trigger) produced output that caused compensation B (the triggered) to
// be invoked.
type ChainLink struct {
	// Trigger is the compensation that caused the cascade (e.g. "speculative").
	Trigger string `json:"trigger"`
	// Triggered is the compensation triggered by the output (e.g. "postprocess").
	Triggered string `json:"triggered"`
	// TriggerReason is a short human-readable description of why the trigger
	// caused the downstream compensation to fire.
	TriggerReason string `json:"trigger_reason"`
	// Depth is the position of this link in the chain (1-based).
	Depth int `json:"depth"`
	// Timestamp records when the chain link was created.
	Timestamp time.Time `json:"timestamp"`
	// ChainID links together all ChainLinks in the same cascade.
	ChainID string `json:"chain_id"`
	// TotalTokensAtLink tracks cumulative tokens across the chain up to this link.
	TotalTokensAtLink int `json:"total_tokens_at_link"`
}

// CompensationChain tracks chains of escalating compensation calls so costs
// can be attributed and runaway cascades can be stopped.
type CompensationChain struct {
	// ID uniquely identifies this chain for cost attribution.
	ID string `json:"id"`
	// Depth is the current number of compensations in this chain.
	Depth int `json:"depth"`
	// Links is the ordered list of trigger→triggered transitions.
	Links []ChainLink `json:"links"`
	// RootCause is the name of the compensation that started the chain.
	RootCause string `json:"root_cause"`
	// TotalTokens tracks cumulative token spend across all links.
	TotalTokens int `json:"total_tokens"`
	// CappedAt indicates whether this chain was capped (MaxChainDepth reached).
	CappedAt int `json:"capped_at,omitempty"`
	// CreatedAt is when the chain started.
	CreatedAt time.Time `json:"created_at"`
	// LastActivity is when the most recent link was added.
	LastActivity time.Time `json:"last_activity"`
}

// Summary returns a one-line description of the chain for logging.
func (c *CompensationChain) Summary() string {
	var b strings.Builder
	fmt.Fprintf(&b, "chain %s: depth=%d/%d root=%s tokens=%d links=[",
		c.ID, c.Depth, MaxChainDepth, c.RootCause, c.TotalTokens)
	for i, link := range c.Links {
		if i > 0 {
			b.WriteString(" → ")
		}
		fmt.Fprintf(&b, "%s→%s", link.Trigger, link.Triggered)
	}
	b.WriteString("]")
	if c.CappedAt > 0 {
		fmt.Fprintf(&b, " CAPPED")
	}
	return b.String()
}

// CostReport returns a detailed breakdown of the chain's token cost per link.
func (c *CompensationChain) CostReport() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Chain %s Cost Report\n\n", c.ID)
	fmt.Fprintf(&b, "Root cause: %s\n", c.RootCause)
	fmt.Fprintf(&b, "Depth: %d (max %d)\n", c.Depth, MaxChainDepth)
	fmt.Fprintf(&b, "Total tokens: %d\n", c.TotalTokens)
	if c.CappedAt > 0 {
		fmt.Fprintf(&b, "⚠ Capped at depth %d\n\n", c.CappedAt)
	} else {
		b.WriteString("\n")
	}
	for _, link := range c.Links {
		fmt.Fprintf(&b, "  L%d: %s → %s  %q  (%d tokens)\n",
			link.Depth, link.Trigger, link.Triggered, link.TriggerReason, link.TotalTokensAtLink)
	}
	return b.String()
}

// CompensationChainTracker maintains a registry of active and historical
// compensation chains. It caps chain depth at MaxChainDepth and provides
// cost analysis across chains.
type CompensationChainTracker struct {
	mu sync.RWMutex

	// active is the chain currently being built for each chain ID.
	active map[string]*CompensationChain

	// history stores completed chains for cost analysis.
	history []*CompensationChain

	// chainIdx is a monotonic counter for generating chain IDs.
	chainIdx int

	// totalTokensAcrossAllChains tracks lifetime tokens across all chains.
	totalTokensAcrossAllChains int

	// cappedCount tracks how many chains were capped at MaxChainDepth.
	cappedCount int
}

// NewCompensationChainTracker creates a chain tracker.
func NewCompensationChainTracker() *CompensationChainTracker {
	return &CompensationChainTracker{
		active: make(map[string]*CompensationChain),
	}
}

// StartChain begins a new compensation chain with the given root compensation
// as the initiator. Returns the chain ID. If the root is already deeper than
// allowed, it returns an empty string and the chain is not started.
func (t *CompensationChainTracker) StartChain(rootCompensation string) string {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.chainIdx++
	id := fmt.Sprintf("cc-%d", t.chainIdx)

	chain := &CompensationChain{
		ID:         id,
		Depth:      0,
		RootCause:  rootCompensation,
		CreatedAt:  time.Now().UTC(),
		LastActivity: time.Now().UTC(),
	}
	t.active[id] = chain
	return id
}

// AddLink adds a trigger→triggered link to an active chain. Returns false and
// an error if the chain would exceed MaxChainDepth. When capped, the chain is
// automatically closed.
func (t *CompensationChainTracker) AddLink(chainID, trigger, triggered, reason string, tokens int) (bool, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	chain, ok := t.active[chainID]
	if !ok {
		return false, fmt.Errorf("chain %s not found", chainID)
	}

	if chain.Depth >= MaxChainDepth {
		// Cap the chain — this link is refused.
		chain.CappedAt = chain.Depth
		t.closeChainLocked(chain)
		return false, fmt.Errorf(
			"compensation chain %s capped at depth %d: %s → %s refused (reason: %s)",
			chainID, MaxChainDepth, trigger, triggered, reason)
	}

	chain.Depth++
	link := ChainLink{
		Trigger:           trigger,
		Triggered:         triggered,
		TriggerReason:     reason,
		Depth:             chain.Depth,
		Timestamp:         time.Now().UTC(),
		ChainID:           chainID,
		TotalTokensAtLink: chain.TotalTokens + tokens,
	}
	chain.Links = append(chain.Links, link)
	chain.TotalTokens += tokens
	chain.LastActivity = time.Now().UTC()
	t.totalTokensAcrossAllChains += tokens

	return true, nil
}

// CloseChain finalises an active chain and moves it to history.
func (t *CompensationChainTracker) CloseChain(chainID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	chain, ok := t.active[chainID]
	if !ok {
		return
	}
	t.closeChainLocked(chain)
}

func (t *CompensationChainTracker) closeChainLocked(chain *CompensationChain) {
	t.history = append(t.history, chain)
	if chain.CappedAt > 0 {
		t.cappedCount++
	}
	delete(t.active, chainID)
}

// ActiveChains returns the chain IDs currently being built.
func (t *CompensationChainTracker) ActiveChains() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	var ids []string
	for id := range t.active {
		ids = append(ids, id)
	}
	return ids
}

// GetChain returns a chain by ID, searching both active and history.
func (t *CompensationChainTracker) GetChain(chainID string) *CompensationChain {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if c, ok := t.active[chainID]; ok {
		return c
	}
	for _, c := range t.history {
		if c.ID == chainID {
			return c
		}
	}
	return nil
}

// History returns all completed chains, most recent first.
func (t *CompensationChainTracker) History() []*CompensationChain {
	t.mu.RLock()
	defer t.mu.RUnlock()
	out := make([]*CompensationChain, len(t.history))
	copy(out, t.history)
	// Reverse so most recent is first.
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// CostAnalysisReport returns a report summarising cost across all chains,
// suitable for display in the Settings → Compensations dashboard.
func (t *CompensationChainTracker) CostAnalysisReport() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var b strings.Builder
	b.WriteString("# Compensation Chain Cost Analysis\n\n")
	fmt.Fprintf(&b, "Total chains tracked: %d\n", len(t.history)+len(t.active))
	fmt.Fprintf(&b, "Completed chains: %d\n", len(t.history))
	fmt.Fprintf(&b, "Active chains: %d\n", len(t.active))
	fmt.Fprintf(&b, "Capped chains (depth %d): %d\n", MaxChainDepth, t.cappedCount)
	fmt.Fprintf(&b, "Total tokens across all chains: %d\n\n", t.totalTokensAcrossAllChains)

	// Per-root-cause breakdown.
	rootCost := make(map[string]int)
	rootCount := make(map[string]int)
	for _, c := range t.history {
		rootCost[c.RootCause] += c.TotalTokens
		rootCount[c.RootCause]++
	}
	for _, c := range t.active {
		rootCost[c.RootCause] += c.TotalTokens
		rootCount[c.RootCause]++
	}

	if len(rootCost) > 0 {
		b.WriteString("## By root cause\n\n")
		for root, tokens := range rootCost {
			fmt.Fprintf(&b, "- %s: %d chains, %d tokens\n", root, rootCount[root], tokens)
		}
	}

	// Average chain depth.
	var totalDepth int
	totalCompleted := len(t.history)
	for _, c := range t.history {
		totalDepth += c.Depth
	}
	if totalCompleted > 0 {
		avgDepth := float64(totalDepth) / float64(totalCompleted)
		fmt.Fprintf(&b, "\nAverage chain depth: %.1f (max %d)\n", avgDepth, MaxChainDepth)
	}

	return b.String()
}

// MostExpensiveChains returns the top N chains by token cost.
func (t *CompensationChainTracker) MostExpensiveChains(n int) []*CompensationChain {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Combine active and history for ranking.
	all := make([]*CompensationChain, 0, len(t.history)+len(t.active))
	all = append(all, t.history...)
	for _, c := range t.active {
		all = append(all, c)
	}

	// Sort by TotalTokens descending (bubble sort since n is small).
	for i := 0; i < len(all)-1; i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].TotalTokens > all[i].TotalTokens {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// CappedCount returns how many chains have been capped at MaxChainDepth.
func (t *CompensationChainTracker) CappedCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.cappedCount
}

// CheckDepthBeforeLink is a helper that checks whether adding another link
// to the given chain would exceed MaxChainDepth before calling AddLink.
// Returns (allowed, currentDepth).
func (t *CompensationChainTracker) CheckDepthBeforeLink(chainID string) (bool, int) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	chain, ok := t.active[chainID]
	if !ok {
		return false, 0
	}
	return chain.Depth < MaxChainDepth, chain.Depth
}
