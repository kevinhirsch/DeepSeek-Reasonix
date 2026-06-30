package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"reasonix/internal/provider"
)

// CompensationCost tracks cumulative token spend for a single compensation.
type CompensationCost struct {
	// Name is the compensation label (e.g. "speculative", "cross-validate",
	// "postprocess", "janitor", "premortem").
	Name string `json:"name"`

	// PromptTokens is the lifetime prompt tokens consumed by this compensation.
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the lifetime completion tokens consumed.
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the lifetime total tokens.
	TotalTokens int `json:"total_tokens"`

	// EstimatedCost is an approximate dollar cost (price per M tokens × total).
	// Updated when provider pricing is set or changed.
	EstimatedCost float64 `json:"estimated_cost"`

	// DaySpend tracks cumulative spend for the current day (UTC).
	DaySpend float64 `json:"day_spend"`

	// MonthSpend tracks cumulative spend for the current month (UTC).
	MonthSpend float64 `json:"month_spend"`

	// DayBudget is the user-configured daily token budget for this compensation.
	// Zero means unlimited.
	DayBudget int `json:"day_budget,omitempty"`

	// MonthBudget is the user-configured monthly token budget. Zero means unlimited.
	MonthBudget int `json:"month_budget,omitempty"`

	// DayTokens is the token count used today.
	DayTokens int `json:"day_tokens"`

	// MonthTokens is the token count used this month.
	MonthTokens int `json:"month_tokens"`

	// Paused indicates whether this compensation is currently paused (manually
	// or automatically by a spike detector).
	Paused bool `json:"paused"`

	// PausedReason explains why the compensation was paused, if applicable.
	PausedReason string `json:"paused_reason,omitempty"`

	// LastSpendDay and LastSpendMonth are used to detect day/month rollover.
	lastSpendDay   int
	lastSpendMonth time.Month

	// PricePerMPrompt is the current provider price per million prompt tokens.
	PricePerMPrompt float64 `json:"price_per_m_prompt"`
	// PricePerMCompletion is the current provider price per million completion tokens.
	PricePerMCompletion float64 `json:"price_per_m_completion"`
}

// Record records token usage against this compensation, updating lifetime, day,
// and month counters. It automatically resets day/month counters on rollover.
func (c *CompensationCost) Record(usage provider.Usage) {
	now := time.Now().UTC()

	// Reset day counters on day rollover.
	if now.Day() != c.lastSpendDay || now.Month() != c.lastSpendMonth {
		c.DayTokens = 0
		c.DaySpend = 0
	}
	// Reset month counters on month rollover.
	if now.Month() != c.lastSpendMonth {
		c.MonthTokens = 0
		c.MonthSpend = 0
	}

	c.lastSpendDay = now.Day()
	c.lastSpendMonth = now.Month()

	c.PromptTokens += usage.PromptTokens
	c.CompletionTokens += usage.CompletionTokens
	c.TotalTokens += usage.TotalTokens
	c.DayTokens += usage.TotalTokens
	c.MonthTokens += usage.TotalTokens

	// Convert tokens to dollars using the current price.
	promptCost := float64(usage.PromptTokens) / 1_000_000 * c.PricePerMPrompt
	completionCost := float64(usage.CompletionTokens) / 1_000_000 * c.PricePerMCompletion
	cost := promptCost + completionCost
	c.EstimatedCost += cost
	c.DaySpend += cost
	c.MonthSpend += cost
}

// OverBudget reports whether this compensation has exceeded either its daily
// or monthly budget (when one is set).
func (c *CompensationCost) OverBudget() (bool, string) {
	if c.DayBudget > 0 && c.DayTokens > c.DayBudget {
		return true, fmt.Sprintf("%s: daily budget exceeded (%d/%d tokens)",
			c.Name, c.DayTokens, c.DayBudget)
	}
	if c.MonthBudget > 0 && c.MonthTokens > c.MonthBudget {
		return true, fmt.Sprintf("%s: monthly budget exceeded (%d/%d tokens)",
			c.Name, c.MonthTokens, c.MonthBudget)
	}
	return false, ""
}

// SetPricing updates the provider pricing used to calculate EstimatedCost and
// recalculates all cost estimates from the recorded token counts.
func (c *CompensationCost) SetPricing(pricePerMPrompt, pricePerMCompletion float64) {
	c.PricePerMPrompt = pricePerMPrompt
	c.PricePerMCompletion = pricePerMCompletion

	promptCost := float64(c.PromptTokens) / 1_000_000 * pricePerMPrompt
	completionCost := float64(c.CompletionTokens) / 1_000_000 * pricePerMCompletion
	c.EstimatedCost = promptCost + completionCost
}

// CompensationCostStore tracks per-compensation spend against user-configured
// thresholds and responds to provider price changes.
type CompensationCostStore struct {
	mu     sync.RWMutex
	costs  map[string]*CompensationCost // name → cost tracker
	paused map[string]string            // name → reason for pause

	// Global budgets (shared across all non-budgeted compensations).
	globalDayBudget   int
	globalMonthBudget int
	globalDayTokens   int
	globalMonthTokens int

	// spikeDetectors track price change history per model.
	spikeDetectors map[string]*SpikeDetector
}

// NewCompensationCostStore creates a cost tracker.
func NewCompensationCostStore() *CompensationCostStore {
	return &CompensationCostStore{
		costs:          make(map[string]*CompensationCost),
		paused:         make(map[string]string),
		spikeDetectors: make(map[string]*SpikeDetector),
	}
}

// GetOrCreate returns the cost tracker for a compensation, creating one with
// the default pricing if it doesn't exist.
func (s *CompensationCostStore) GetOrCreate(name string, pricePerMPrompt, pricePerMCompletion float64) *CompensationCost {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.costs[name]; ok {
		return c
	}
	c := &CompensationCost{
		Name:                name,
		PricePerMPrompt:     pricePerMPrompt,
		PricePerMCompletion: pricePerMCompletion,
	}
	s.costs[name] = c
	return c
}

// Record records token usage for a named compensation.
func (s *CompensationCostStore) Record(name string, usage provider.Usage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.costs[name]
	if !ok {
		// Create with zero pricing — caller should call SetPricing before use
		// or this compensation hasn't been initialised with GetOrCreate.
		c = &CompensationCost{Name: name}
		s.costs[name] = c
	}
	c.Record(usage)
	s.globalDayTokens += usage.TotalTokens
	s.globalMonthTokens += usage.TotalTokens
}

// SetBudget configures a per-compensation budget.
func (s *CompensationCostStore) SetBudget(name string, dayBudget, monthBudget int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.costs[name]
	if !ok {
		c = &CompensationCost{Name: name}
		s.costs[name] = c
	}
	c.DayBudget = dayBudget
	c.MonthBudget = monthBudget
}

// SetGlobalBudget configures the global budget across all compensations.
func (s *CompensationCostStore) SetGlobalBudget(dayBudget, monthBudget int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.globalDayBudget = dayBudget
	s.globalMonthBudget = monthBudget
}

// BudgetExceeded returns a list of compensations exceeding their budgets.
func (s *CompensationCostStore) BudgetExceeded() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var exceeded []string
	for _, c := range s.costs {
		if over, reason := c.OverBudget(); over {
			exceeded = append(exceeded, reason)
		}
	}
	if s.globalDayBudget > 0 && s.globalDayTokens > s.globalDayBudget {
		exceeded = append(exceeded, fmt.Sprintf(
			"global daily budget exceeded (%d/%d tokens)",
			s.globalDayTokens, s.globalDayBudget))
	}
	if s.globalMonthBudget > 0 && s.globalMonthTokens > s.globalMonthBudget {
		exceeded = append(exceeded, fmt.Sprintf(
			"global monthly budget exceeded (%d/%d tokens)",
			s.globalMonthTokens, s.globalMonthBudget))
	}
	return exceeded
}

// Pause sets the paused flag on a compensation.
func (s *CompensationCostStore) Pause(name, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.costs[name]; ok {
		c.Paused = true
		c.PausedReason = reason
	}
	s.paused[name] = reason
}

// Unpause clears the paused flag on a compensation.
func (s *CompensationCostStore) Unpause(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.costs[name]; ok {
		c.Paused = false
		c.PausedReason = ""
	}
	delete(s.paused, name)
}

// IsPaused reports whether a compensation is paused.
func (s *CompensationCostStore) IsPaused(name string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.paused[name]
	return ok
}

// PausedList returns the names and reasons of all paused compensations.
func (s *CompensationCostStore) PausedList() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(s.paused))
	for k, v := range s.paused {
		out[k] = v
	}
	return out
}

// MassPause pauses all compensations that are not already paused.
func (s *CompensationCostStore) MassPause(reason string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var paused []string
	for name, c := range s.costs {
		if !c.Paused {
			c.Paused = true
			c.PausedReason = reason
			s.paused[name] = reason
			paused = append(paused, name)
		}
	}
	sort.Strings(paused)
	return paused
}

// OnPriceChange handles a provider price update. It recalculates cost estimates
// for all compensations, returns a list of compensations whose costs now exceed
// their budgets, and triggers mass-pause on extreme spikes.
//
// If the price change is a spike of 10× or more compared to the last known
// price, it auto-pauses ALL compensations and returns a prominent notice.
// For smaller changes, it returns the list of compensations affected so the
// caller can offer mass-pause via UI.
func (s *CompensationCostStore) OnPriceChange(model string, newPromptPrice, newCompletionPrice float64) *PriceChangeResult {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for extreme spike.
	sd := s.spikeDetectors[model]
	if sd == nil {
		sd = NewSpikeDetector(model)
		s.spikeDetectors[model] = sd
	}
	isSpike, spikeRatio := sd.Detect(newPromptPrice, newCompletionPrice)

	result := &PriceChangeResult{
		Model:             model,
		NewPromptPrice:    newPromptPrice,
		NewCompletionPrice: newCompletionPrice,
		IsSpike:           isSpike,
		SpikeRatio:        spikeRatio,
	}

	// Recalculate cost estimates for all compensations at the new price.
	var affected []string
	for name, c := range s.costs {
		oldCost := c.EstimatedCost
		c.SetPricing(newPromptPrice, newCompletionPrice)
		newCost := c.EstimatedCost

		if newCost > oldCost*1.05 { // >5% increase is notable
			result.Affected = append(result.Affected, name)
		}
		if over, reason := c.OverBudget(); over {
			affected = append(affected, reason)
		}
	}

	// On extreme spike (10×+): auto-pause all compensations.
	if isSpike {
		for name, c := range s.costs {
			if !c.Paused {
				c.Paused = true
				c.PausedReason = fmt.Sprintf("auto-paused: %s price spiked %.1f×", model, spikeRatio)
				s.paused[name] = c.PausedReason
				result.AutoPaused = append(result.AutoPaused, name)
			}
		}
		result.AutoPauseNote = fmt.Sprintf(
			"⚠ Provider price spike detected for %s: %.1f× increase. "+
				"All %d compensations have been auto-paused to prevent unexpected charges. "+
				"Review and re-enable in Settings → Compensations.",
			model, spikeRatio, len(result.AutoPaused))
	} else if len(affected) > 0 {
		result.ExceedingBudget = affected
	}

	return result
}

// PriceChangeResult summarises the outcome of a provider price change.
type PriceChangeResult struct {
	Model              string   `json:"model"`
	NewPromptPrice     float64  `json:"new_prompt_price"`
	NewCompletionPrice float64  `json:"new_completion_price"`
	IsSpike            bool     `json:"is_spike"`
	SpikeRatio         float64  `json:"spike_ratio"`
	Affected           []string `json:"affected"`
	ExceedingBudget    []string `json:"exceeding_budget,omitempty"`
	AutoPaused         []string `json:"auto_paused,omitempty"`
	AutoPauseNote      string   `json:"auto_pause_note,omitempty"`
}

// HasNotableChanges reports whether the price change produced actionable results.
func (r *PriceChangeResult) HasNotableChanges() bool {
	return r.IsSpike || len(r.ExceedingBudget) > 0 || len(r.Affected) > 0
}

// SpikeDetector detects extreme price spikes (10× or more) by comparing new
// provider pricing against the last recorded baseline.
type SpikeDetector struct {
	Model string `json:"model"`

	lastPromptPrice     float64
	lastCompletionPrice float64

	// LastChecked is the time of the most recent price check.
	LastChecked time.Time `json:"last_checked"`
}

// NewSpikeDetector creates a spike detector for a model.
func NewSpikeDetector(model string) *SpikeDetector {
	return &SpikeDetector{Model: model}
}

// Detect compares new prices against the last known baseline. Returns true and
// the spike ratio if the new price is at least 10× the baseline. A zero
// baseline (first check) is never a spike.
func (d *SpikeDetector) Detect(newPromptPrice, newCompletionPrice float64) (bool, float64) {
	d.LastChecked = time.Now().UTC()

	// First pricing — no baseline, not a spike.
	if d.lastPromptPrice == 0 && d.lastCompletionPrice == 0 {
		d.lastPromptPrice = newPromptPrice
		d.lastCompletionPrice = newCompletionPrice
		return false, 0
	}

	// Use the larger ratio (prompt or completion) to determine spike severity.
	promptRatio := 1.0
	if d.lastPromptPrice > 0 && newPromptPrice > 0 {
		promptRatio = newPromptPrice / d.lastPromptPrice
	}
	completionRatio := 1.0
	if d.lastCompletionPrice > 0 && newCompletionPrice > 0 {
		completionRatio = newCompletionPrice / d.lastCompletionPrice
	}

	maxRatio := promptRatio
	if completionRatio > maxRatio {
		maxRatio = completionRatio
	}

	// Update baseline to the new price.
	d.lastPromptPrice = newPromptPrice
	d.lastCompletionPrice = newCompletionPrice

	// 10× or more is an extreme spike.
	if maxRatio >= 10.0 {
		return true, maxRatio
	}
	// 2× or more is notable but not an auto-pause trigger.
	if maxRatio >= 2.0 {
		return false, maxRatio
	}
	return false, 0
}

// Baseline returns the last known prices for the model.
func (d *SpikeDetector) Baseline() (promptPrice, completionPrice float64) {
	return d.lastPromptPrice, d.lastCompletionPrice
}

// BuildMassPauseNotice constructs a user-visible notice describing the pause
// action taken. It includes the reason, the list of paused compensations, and
// instructions for re-enabling them.
func BuildMassPauseNotice(reason string, paused []string) string {
	var b strings.Builder
	b.WriteString("⚠ Compensation mass-pause\n\n")
	b.WriteString(reason)
	b.WriteString("\n\nPaused compensations:\n")
	for _, name := range paused {
		b.WriteString("- " + name + "\n")
	}
	b.WriteString("\nTo re-enable, open Settings → Compensations and toggle individual items, or click \"Resume All\".")
	return b.String()
}

// Recorder is an interface for recording compensation token usage, so cost
// tracking can be injected without a concrete dependency on CompensationCostStore.
type CompensationRecorder interface {
	RecordCompensation(ctx context.Context, name string, usage provider.Usage)
}

// compensationRecorderKey is the context key for the compensation recorder.
type compensationRecorderKey struct{}

// WithCompensationRecorder attaches a CompensationRecorder to the context.
func WithCompensationRecorder(ctx context.Context, r CompensationRecorder) context.Context {
	return context.WithValue(ctx, compensationRecorderKey{}, r)
}

// CompensationRecorderFromContext retrieves the CompensationRecorder from a context.
func CompensationRecorderFromContext(ctx context.Context) CompensationRecorder {
	r, _ := ctx.Value(compensationRecorderKey{}).(CompensationRecorder)
	return r
}

// RecordCompensation implements CompensationRecorder on CompensationCostStore.
func (s *CompensationCostStore) RecordCompensation(ctx context.Context, name string, usage provider.Usage) {
	s.Record(name, usage)
}
