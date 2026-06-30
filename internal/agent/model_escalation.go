package agent

import (
	"fmt"
	"strings"
	"sync"
)

// EscalationLevel describes the compute tier for a subagent.
type EscalationLevel int

const (
	EscalateFlash EscalationLevel = iota // cheapest, default
	EscalatePro                          // complex multi-step
	EscalateMax                          // adversarial verification, security
)

func (l EscalationLevel) String() string {
	switch l {
	case EscalateFlash:
		return "flash"
	case EscalatePro:
		return "pro"
	case EscalateMax:
		return "max"
	default:
		return "unknown"
	}
}

// EscalationRule determines when to escalate from Flash → Pro → Max.
type EscalationRule struct {
	Role            string   // explorer, reviewer, verifier, etc.
	DefaultLevel    EscalationLevel
	EscalateOnRoles []string // tasks with these roles auto-escalate
	MinSteps        int      // escalate if task needs >N tool calls
	ComplexKeywords []string // escalate if prompt contains these
}

var defaultEscalationRules = []EscalationRule{
	{
		Role:         "explore",
		DefaultLevel: EscalateFlash,
		MinSteps:     5, // if explorer needs >5 calls, escalate to pro next time
	},
	{
		Role:         "review",
		DefaultLevel: EscalateFlash,
		MinSteps:     8,
	},
	{
		Role:            "verify",
		DefaultLevel:    EscalatePro,
		EscalateOnRoles: []string{"security_review"},
		ComplexKeywords:  []string{"adversarial", "refute", "break", "exploit"},
	},
	{
		Role:            "security_review",
		DefaultLevel:    EscalateMax,
		ComplexKeywords:  []string{"injection", "overflow", "bypass", "escape"},
	},
	{
		Role:            "planner",
		DefaultLevel:    EscalateFlash,
		MinSteps:         3,
		ComplexKeywords:  []string{"architecture", "multi-file", "cross-cutting"},
	},
	{
		Role:         "executor",
		DefaultLevel: EscalateFlash,
		MinSteps:     10,
		ComplexKeywords: []string{"refactor", "migration", "multi-file", "cross-cutting"},
	},
}

// ModelEscalator optimizes for cost by defaulting to Flash and
// auto-upgrading to Pro or Max when task complexity warrants it.
type ModelEscalator struct {
	mu    sync.RWMutex
	rules map[string]EscalationRule
	// Track task history per role to learn which need escalation
	history map[string][]escalationRecord
}

type escalationRecord struct {
	Role     string
	Steps    int
	Escalated bool
}

// NewModelEscalator creates an escalator with default rules.
func NewModelEscalator() *ModelEscalator {
	e := &ModelEscalator{
		rules:   make(map[string]EscalationRule),
		history: make(map[string][]escalationRecord),
	}
	for _, r := range defaultEscalationRules {
		e.rules[r.Role] = r
	}
	return e
}

// ResolveLevel returns the appropriate escalation level for a task.
func (e *ModelEscalator) ResolveLevel(role, prompt string, priority int) EscalationLevel {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rule, ok := e.rules[role]
	if !ok {
		return EscalateFlash
	}

	// Priority 0 (user-initiated foreground) gets the default level.
	// Priority >0 (workflow, compensations) gets at least Pro.
	if priority > 0 && rule.DefaultLevel < EscalatePro {
		return EscalatePro
	}

	// Check complex keywords
	promptLower := strings.ToLower(prompt)
	for _, kw := range rule.ComplexKeywords {
		if strings.Contains(promptLower, kw) {
			return minLevel(EscalatePro, rule.DefaultLevel)
		}
	}

	// Check role-based escalation
	for _, r := range rule.EscalateOnRoles {
		if strings.EqualFold(role, r) {
			return EscalateMax
		}
	}

	// Check history: if this role has consistently needed >MinSteps,
	// escalate future tasks
	if rule.MinSteps > 0 {
		avgSteps := e.avgStepsForRole(role)
		if avgSteps > float64(rule.MinSteps) {
			return minLevel(EscalatePro, rule.DefaultLevel)
		}
	}

	return rule.DefaultLevel
}

// RecordResult logs the outcome of a task to influence future escalation.
func (e *ModelEscalator) RecordResult(role string, steps int, wasEscalated bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.history[role] = append(e.history[role], escalationRecord{
		Role:     role,
		Steps:    steps,
		Escalated: wasEscalated,
	})
	// Keep last 20 records
	if len(e.history[role]) > 20 {
		e.history[role] = e.history[role][len(e.history[role])-20:]
	}
}

// ModelRef returns the provider model string for a level.
func (e *ModelEscalator) ModelRef(level EscalationLevel) string {
	switch level {
	case EscalateFlash:
		return "deepseek-v4-flash"
	case EscalatePro:
		return "deepseek-v4-pro"
	case EscalateMax:
		return "deepseek-v4-pro"
	default:
		return "deepseek-v4-flash"
	}
}

// EffortFor translates a level to the corresponding reasoning effort.
func (e *ModelEscalator) EffortFor(level EscalationLevel) string {
	switch level {
	case EscalateFlash:
		return "high"
	case EscalatePro:
		return "high"
	case EscalateMax:
		return "max"
	default:
		return "high"
	}
}

// Summary returns a human-readable explanation of current escalation state.
func (e *ModelEscalator) Summary() string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	var b strings.Builder
	b.WriteString("Model Escalation: Flash (default) → Pro (complex) → Max (adversarial)\n")
	for role, rule := range e.rules {
		avg := e.avgStepsForRole(role)
		fmt.Fprintf(&b, "  %s: default=%s avg_steps=%.1f\n", role, rule.DefaultLevel, avg)
	}
	return b.String()
}

func (e *ModelEscalator) avgStepsForRole(role string) float64 {
	records := e.history[role]
	if len(records) == 0 {
		return 0
	}
	sum := 0
	for _, r := range records {
		sum += r.Steps
	}
	return float64(sum) / float64(len(records))
}

func minLevel(a, b EscalationLevel) EscalationLevel {
	if a > b {
		return a
	}
	return b
}
