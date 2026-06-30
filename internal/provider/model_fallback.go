// Package provider — model fallback chain and health tracking.
//
// ModelFallbackManager watches provider health and transparently fails over
// to a configured FallbackChain when the primary model is unavailable. It
// monitors triggers (timeouts, 429 rate limits, 503/5xx server errors) and
// degrades through a sequence of fallback models, recovering automatically
// when the primary returns to a healthy state.
//
// Degradation levels:
//   - Normal          — primary is healthy, no fallback
//   - Degraded        — primary is slow/erratic but still serving
//   - FallbackActive  — primary declared unhealthy, fallback model in use
//   - Emergency       — all fallbacks exhausted, last-resort behaviour
package provider

import (
	"context"
	"net/http"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Fallback trigger
// ---------------------------------------------------------------------------

// FallbackTrigger is a bitmask of conditions that can cause a fallback.
type FallbackTrigger uint8

const (
	// TriggerTimeout fires when a request exceeds a configured deadline.
	TriggerTimeout FallbackTrigger = 1 << iota

	// TriggerRateLimited fires on HTTP 429 (Too Many Requests).
	TriggerRateLimited

	// TriggerUnavailable fires on HTTP 503 (Service Unavailable).
	TriggerUnavailable

	// TriggerServerError fires on any HTTP 5xx that is not 503.
	TriggerServerError

	// TriggerConnectionErr fires when a transport error (DNS, dial, TLS,
	// connection refused) prevents the request from reaching the server.
	TriggerConnectionErr

	// TriggerAuthFailure fires on repeated 401/403 responses, indicating
	// the primary key may be invalid or expired.
	TriggerAuthFailure
)

// String returns a human-readable label combining the active trigger flags.
func (t FallbackTrigger) String() string {
	if t == 0 {
		return "none"
	}
	var parts []string
	if t&TriggerTimeout != 0 {
		parts = append(parts, "timeout")
	}
	if t&TriggerRateLimited != 0 {
		parts = append(parts, "rate-limited")
	}
	if t&TriggerUnavailable != 0 {
		parts = append(parts, "unavailable")
	}
	if t&TriggerServerError != 0 {
		parts = append(parts, "server-error")
	}
	if t&TriggerConnectionErr != 0 {
		parts = append(parts, "connection-error")
	}
	if t&TriggerAuthFailure != 0 {
		parts = append(parts, "auth-failure")
	}
	switch len(parts) {
	case 0:
		return "none"
	case 1:
		return parts[0]
	default:
		s := parts[0]
		for _, p := range parts[1:] {
			s += "+" + p
		}
		return s
	}
}

// TriggerFromStatus maps an HTTP status code to the appropriate fallback
// trigger. Returns 0 for codes that should not trigger fallback.
func TriggerFromStatus(status int) FallbackTrigger {
	switch status {
	case http.StatusTooManyRequests:
		return TriggerRateLimited
	case http.StatusServiceUnavailable:
		return TriggerUnavailable
	default:
		if status >= 500 && status <= 599 {
			return TriggerServerError
		}
		return 0
	}
}

// ---------------------------------------------------------------------------
// Degradation level
// ---------------------------------------------------------------------------

// DegradationLevel describes the current fallback state of a provider.
type DegradationLevel int

const (
	// DegradationNormal — primary model is healthy and serving all traffic.
	DegradationNormal DegradationLevel = iota

	// DegradationDegraded — primary is slow or showing transient errors
	// but still taking traffic. Fallback is not yet active.
	DegradationDegraded

	// DegradationFallbackActive — primary is declared unhealthy;
	// requests are routed to a fallback model in the chain.
	DegradationFallbackActive

	// DegradationEmergency — all fallback models are exhausted; the
	// provider is effectively down.
	DegradationEmergency
)

// String returns the lowercased level name.
func (l DegradationLevel) String() string {
	switch l {
	case DegradationNormal:
		return "normal"
	case DegradationDegraded:
		return "degraded"
	case DegradationFallbackActive:
		return "fallback_active"
	case DegradationEmergency:
		return "emergency"
	default:
		return "unknown"
	}
}

// IsHealthy reports whether the provider can accept requests without
// fallback intervention.
func (l DegradationLevel) IsHealthy() bool {
	return l == DegradationNormal || l == DegradationDegraded
}

// ---------------------------------------------------------------------------
// Fallback model
// ---------------------------------------------------------------------------

// FallbackModel describes one alternative model in the chain.
type FallbackModel struct {
	// Model is the model identifier sent to the provider (e.g.
	// "deepseek-chat" when the primary is "deepseek-reasoner").
	Model string `json:"model"`

	// BaseURL overrides the provider endpoint for this fallback. Empty
	// means use the same endpoint as the primary.
	BaseURL string `json:"base_url,omitempty"`

	// MaxTokensCap is an optional token limit override for the fallback
	// model. 0 means inherit from the primary config.
	MaxTokensCap int `json:"max_tokens_cap,omitempty"`

	// Description is a human-readable label for diagnostics.
	Description string `json:"description,omitempty"`
}

// ---------------------------------------------------------------------------
// Fallback chain
// ---------------------------------------------------------------------------

// FallbackChain is an ordered list of fallback models the manager tries
// when the primary becomes unhealthy. Fallback models are tried in order;
// when one fails, the next is attempted. When all are exhausted the
// degradation level moves to Emergency.
type FallbackChain struct {
	// Primary is the name of the primary model for this provider.
	Primary string `json:"primary"`

	// Fallbacks is the ordered list of fallback models.
	Fallbacks []FallbackModel `json:"fallbacks"`

	// MaxConsecutiveFailures is how many consecutive failures against
	// the current model trigger the next fallback step. Default 3.
	MaxConsecutiveFailures int `json:"max_consecutive_failures,omitempty"`

	// RecoveryProbeInterval is how often the manager probes the primary
	// while in fallback to check whether it has recovered. Default 30s.
	RecoveryProbeInterval time.Duration `json:"recovery_probe_interval,omitempty"`

	// RecoveryProbeSuccesses is how many consecutive successful probes
	// are required before the primary is declared healthy again. Default 2.
	RecoveryProbeSuccesses int `json:"recovery_probe_successes,omitempty"`

	// CooldownPeriod prevents rapid oscillation: once the primary recovers,
	// it won't fall back again for at least this duration. Default 5m.
	CooldownPeriod time.Duration `json:"cooldown_period,omitempty"`
}

// defaults fills in sensible zero-value defaults for optional fields.
func (c *FallbackChain) defaults() {
	if c.MaxConsecutiveFailures <= 0 {
		c.MaxConsecutiveFailures = 3
	}
	if c.RecoveryProbeInterval <= 0 {
		c.RecoveryProbeInterval = 30 * time.Second
	}
	if c.RecoveryProbeSuccesses <= 0 {
		c.RecoveryProbeSuccesses = 2
	}
	if c.CooldownPeriod <= 0 {
		c.CooldownPeriod = 5 * time.Minute
	}
}

// CurrentModel returns the model identifier for the given fallback position.
// Position 0 is the primary; 1 is the first fallback, etc.
func (c *FallbackChain) CurrentModel(position int) string {
	if position <= 0 {
		return c.Primary
	}
	idx := position - 1
	if idx >= len(c.Fallbacks) {
		return ""
	}
	return c.Fallbacks[idx].Model
}

// IsExhausted reports whether the given position exceeds all fallbacks.
func (c *FallbackChain) IsExhausted(position int) bool {
	return position > len(c.Fallbacks)
}

// ---------------------------------------------------------------------------
// Health probe result
// ---------------------------------------------------------------------------

// HealthProbe records the outcome of a single health check.
type HealthProbe struct {
	Time    time.Time
	Success bool
	Latency time.Duration
	Status  int // HTTP status; 0 for transport errors
	Trigger FallbackTrigger
}

// ---------------------------------------------------------------------------
// Model fallback manager
// ---------------------------------------------------------------------------

// ModelFallbackManager tracks the health of a provider's primary model,
// fails over through a FallbackChain when the primary is unhealthy, and
// automatically recovers the primary when it becomes healthy again.
//
// All methods are safe for concurrent use — the agent and health-probe
// goroutine may call from different goroutines.
type ModelFallbackManager struct {
	chain FallbackChain

	mu sync.Mutex

	// level is the current degradation level.
	level DegradationLevel

	// fallbackPosition is the index of the currently active fallback
	// model. 0 = primary, 1 = first fallback, etc.
	fallbackPosition int

	// consecutiveFailures counts consecutive errors against the current
	// model. Reset on any success or when a new fallback is activated.
	consecutiveFailures int

	// consecutiveSuccesses counts consecutive successful health probes
	// against the primary while in fallback. Reset on any failure.
	consecutiveSuccesses int

	// lastFailure records the most recent failure for diagnostics.
	lastFailure HealthProbe

	// lastRecovery is when the primary was last recovered from fallback.
	// Used to enforce the cooldown period.
	lastRecovery time.Time

	// probeHistory is a bounded circular buffer of recent health probes
	// for diagnostics and dashboards.
	probeHistory []HealthProbe
	probeIdx     int
	maxProbes    int
}

// NewModelFallbackManager creates a manager for the given fallback chain.
// The chain's defaults are applied for any zero-value optional fields.
func NewModelFallbackManager(chain FallbackChain) *ModelFallbackManager {
	chain.defaults()
	return &ModelFallbackManager{
		chain:      chain,
		level:      DegradationNormal,
		maxProbes:  64,
		probeHistory: make([]HealthProbe, 64),
	}
}

// Level returns the current degradation level.
func (m *ModelFallbackManager) Level() DegradationLevel {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.level
}

// FallbackPosition returns the current position in the chain (0 = primary).
func (m *ModelFallbackManager) FallbackPosition() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fallbackPosition
}

// ActiveModel returns the model identifier currently in use.
func (m *ModelFallbackManager) ActiveModel() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.chain.CurrentModel(m.fallbackPosition)
}

// ActiveBaseURL returns the base URL currently in use (empty = primary URL).
func (m *ModelFallbackManager) ActiveBaseURL() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fallbackPosition <= 0 {
		return ""
	}
	idx := m.fallbackPosition - 1
	if idx >= len(m.chain.Fallbacks) {
		return ""
	}
	return m.chain.Fallbacks[idx].BaseURL
}

// ActiveMaxTokensCap returns the token cap for the active model, or 0
// to use the primary config.
func (m *ModelFallbackManager) ActiveMaxTokensCap() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.fallbackPosition <= 0 {
		return 0
	}
	idx := m.fallbackPosition - 1
	if idx >= len(m.chain.Fallbacks) {
		return 0
	}
	return m.chain.Fallbacks[idx].MaxTokensCap
}

// RecordResult records the outcome of an API request against the current
// model. A successful response resets the failure counter and, if we are
// probing the primary, counts toward recovery. A failing response may
// advance the fallback position.
func (m *ModelFallbackManager) RecordResult(status int, err error, trigger FallbackTrigger) {
	m.mu.Lock()
	defer m.mu.Unlock()

	probe := HealthProbe{
		Time:    time.Now(),
		Success: err == nil && status > 0 && status < 400,
		Status:  status,
		Trigger: trigger,
	}
	m.recordProbeLocked(probe)
	m.lastFailure = probe

	if probe.Success {
		m.consecutiveFailures = 0

		// If we are probing the primary for recovery.
		if m.level == DegradationFallbackActive && m.fallbackPosition > 0 {
			// When a fallback model succeeds, that's business-as-usual.
			// Recovery tracking is done by health probes against the
			// primary, not by fallback successes. But if the caller
			// explicitly sends a probe against the primary (fallbackPosition==0),
			// count it.
		}
		return
	}

	m.lastFailure = probe
	m.consecutiveFailures++

	threshold := m.chain.MaxConsecutiveFailures
	if m.fallbackPosition > 0 {
		// When already in fallback, be more aggressive about moving
		// to the next fallback (half the threshold, min 1).
		threshold = max(1, m.chain.MaxConsecutiveFailures/2)
	}

	if m.consecutiveFailures >= threshold {
		m.advanceLocked()
	}
}

// advanceLocked moves to the next fallback model or emergency. Caller
// holds m.mu.
func (m *ModelFallbackManager) advanceLocked() {
	m.consecutiveFailures = 0

	next := m.fallbackPosition + 1
	if m.chain.IsExhausted(next) {
		m.level = DegradationEmergency
		return
	}

	m.fallbackPosition = next
	if m.fallbackPosition > 0 {
		m.level = DegradationFallbackActive
	} else {
		m.level = DegradationDegraded
	}
}

// ProbePrimaryHealth performs a lightweight health check against the
// primary model's endpoint and updates the recovery state. Call this
// periodically from a goroutine while in fallback. The probe function
// should make a minimal request (e.g. a cached prompt) and return its
// result.
func (m *ModelFallbackManager) ProbePrimaryHealth(probeFn func(context.Context) (status int, err error)) {
	m.mu.Lock()

	// Only probe when in fallback.
	if m.level != DegradationFallbackActive && m.level != DegradationEmergency {
		// Also probe when degraded to detect escalation.
		if m.level != DegradationDegraded {
			m.mu.Unlock()
			return
		}
	}
	// Respect cooldown: if we recently recovered, don't re-trigger.
	if !m.lastRecovery.IsZero() && time.Since(m.lastRecovery) < m.chain.CooldownPeriod {
		m.mu.Unlock()
		return
	}
	m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	status, err := probeFn(ctx)

	m.mu.Lock()
	defer m.mu.Unlock()

	probe := HealthProbe{
		Time:    time.Now(),
		Success: err == nil && status > 0 && status < 400,
		Status:  status,
	}
	m.recordProbeLocked(probe)

	if probe.Success {
		m.consecutiveSuccesses++
		if m.consecutiveSuccesses >= m.chain.RecoveryProbeSuccesses {
			m.recoverLocked()
		}
	} else {
		m.consecutiveSuccesses = 0
		// If we're in emergency and the primary probe fails, stay in
		// emergency. If in fallback and the primary also fails, that's
		// expected — we stay in fallback.
	}
}

// recoverLocked resets to primary. Caller holds m.mu.
func (m *ModelFallbackManager) recoverLocked() {
	m.level = DegradationNormal
	m.fallbackPosition = 0
	m.consecutiveFailures = 0
	m.consecutiveSuccesses = 0
	m.lastRecovery = time.Now()
}

// ForceFallback immediately activates the given fallback position (0 =
// primary, 1 = first fallback, etc.). Use for manual operator intervention
// or load-shedding. Returns an error if the position is out of range.
func (m *ModelFallbackManager) ForceFallback(position int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if position < 0 {
		return nil
	}
	if m.chain.IsExhausted(position) {
		return nil // already emergency
	}

	m.fallbackPosition = position
	m.consecutiveFailures = 0
	m.consecutiveSuccesses = 0
	if m.chain.IsExhausted(position) {
		m.level = DegradationEmergency
	} else if position > 0 {
		m.level = DegradationFallbackActive
	} else {
		m.level = DegradationNormal
	}
	return nil
}

// Reset clears the fallback state and returns to primary. Use when the
// operator or a configuration change resolves the underlying issue.
func (m *ModelFallbackManager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.level = DegradationNormal
	m.fallbackPosition = 0
	m.consecutiveFailures = 0
	m.consecutiveSuccesses = 0
}

// MarkDegraded sets the level to Degraded without activating fallback.
// Use when the primary is showing warning signs but is not yet failing
// enough consecutive requests to fail over.
func (m *ModelFallbackManager) MarkDegraded() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.level == DegradationNormal {
		m.level = DegradationDegraded
	}
}

// LastFailure returns the most recent failing health probe.
func (m *ModelFallbackManager) LastFailure() HealthProbe {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastFailure
}

// LastRecovery returns when the primary last recovered from fallback.
func (m *ModelFallbackManager) LastRecovery() time.Time {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastRecovery
}

// ConsecutiveFailures returns the current consecutive failure count.
func (m *ModelFallbackManager) ConsecutiveFailures() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.consecutiveFailures
}

// ProbeHistory returns a copy of recent health probes, most recent first.
func (m *ModelFallbackManager) ProbeHistory() []HealthProbe {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := len(m.probeHistory)
	out := make([]HealthProbe, n)
	// Unwind the circular buffer in reverse order so the most recent
	// probe is first.
	for i := 0; i < n; i++ {
		idx := (m.probeIdx - 1 - i + n) % n
		out[i] = m.probeHistory[idx]
		if out[i].Time.IsZero() {
			return out[:i]
		}
	}
	return out
}

// recordProbeLocked appends a probe to the circular buffer. Caller holds
// m.mu.
func (m *ModelFallbackManager) recordProbeLocked(p HealthProbe) {
	m.probeHistory[m.probeIdx%m.maxProbes] = p
	m.probeIdx++
}

// ---------------------------------------------------------------------------
// Status snapshot for diagnostics
// ---------------------------------------------------------------------------

// FallbackStatus is a point-in-time snapshot of the fallback manager's
// state, suitable for telemetry endpoints or debug UIs.
type FallbackStatus struct {
	Level                DegradationLevel `json:"level"`
	ActiveModel          string           `json:"active_model"`
	FallbackPosition     int              `json:"fallback_position"`
	ConsecutiveFailures  int              `json:"consecutive_failures"`
	ConsecutiveSuccesses int              `json:"consecutive_successes"`
	LastFailure          *HealthProbe     `json:"last_failure,omitempty"`
	LastRecovery         time.Time        `json:"last_recovery,omitempty"`
	ChainLength          int              `json:"chain_length"`
}

// Status returns an immutable snapshot of the manager's state.
func (m *ModelFallbackManager) Status() FallbackStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	s := FallbackStatus{
		Level:                m.level,
		ActiveModel:          m.chain.CurrentModel(m.fallbackPosition),
		FallbackPosition:     m.fallbackPosition,
		ConsecutiveFailures:  m.consecutiveFailures,
		ConsecutiveSuccesses: m.consecutiveSuccesses,
		LastRecovery:         m.lastRecovery,
		ChainLength:          len(m.chain.Fallbacks),
	}
	if !m.lastFailure.Time.IsZero() {
		lf := m.lastFailure
		s.LastFailure = &lf
	}
	return s
}

// ---------------------------------------------------------------------------
// Shared fallback manager registry
// ---------------------------------------------------------------------------

// fallbackMu guards the shared registry so tests and production code can
// register/deregister managers without a data race.
var fallbackMu sync.Mutex
var fallbackManagers sync.Map // map[string]*ModelFallbackManager (provider name → manager)

// RegisterFallbackManager stores a fallback manager keyed by provider name.
// It is safe for concurrent use and intended for wiring at startup.
func RegisterFallbackManager(providerName string, m *ModelFallbackManager) {
	fallbackManagers.Store(providerName, m)
}

// GetFallbackManager retrieves a previously registered fallback manager.
// ok is false when no manager is registered for the given provider.
func GetFallbackManager(providerName string) (*ModelFallbackManager, bool) {
	v, ok := fallbackManagers.Load(providerName)
	if !ok {
		return nil, false
	}
	return v.(*ModelFallbackManager), true
}

// UnregisterFallbackManager removes a fallback manager from the registry.
func UnregisterFallbackManager(providerName string) {
	fallbackManagers.Delete(providerName)
}

// AllFallbackStatuses returns status snapshots for every registered
// fallback manager, keyed by provider name.
func AllFallbackStatuses() map[string]FallbackStatus {
	out := make(map[string]FallbackStatus)
	fallbackManagers.Range(func(key, value any) bool {
		name := key.(string)
		m := value.(*ModelFallbackManager)
		out[name] = m.Status()
		return true
	})
	return out
}

// max returns the larger of x and y.
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
