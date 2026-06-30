package agent

import (
	"sync"
	"time"
)

// ConfidenceProfile tracks per-(role, model, language) accuracy over time.
// Profiles are used for weighted voting and routing subagents to their
// strongest configurations.
type ConfidenceProfile struct {
	Role           string    `json:"role"`
	Model          string    `json:"model"`
	TotalTasks     int       `json:"total_tasks"`
	ConfirmedCount int       `json:"confirmed"`
	PlausibleCount int       `json:"plausible"`
	RefutedCount   int       `json:"refuted"`
	Accuracy       float64   `json:"accuracy"`
	Weight         float64   `json:"weight"` // 0.0-1.0, decayed on model bump
	LastSample     time.Time `json:"last_sample"`
	ModelVersion   string    `json:"model_version"`
}

// ConfidenceStore tracks accuracy profiles for subagent roles.
type ConfidenceStore struct {
	mu       sync.RWMutex
	profiles map[string]*ConfidenceProfile // key: "role:model"
}

// NewConfidenceStore creates a confidence tracker.
func NewConfidenceStore() *ConfidenceStore {
	return &ConfidenceStore{
		profiles: make(map[string]*ConfidenceProfile),
	}
}

// GetOrCreate returns the profile for a role+model, creating one if needed.
func (c *ConfidenceStore) GetOrCreate(role, model string) *ConfidenceProfile {
	key := role + ":" + model
	c.mu.RLock()
	p, ok := c.profiles[key]
	c.mu.RUnlock()
	if ok {
		return p
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	p = &ConfidenceProfile{
		Role:  role,
		Model: model,
		Weight: 1.0,
	}
	c.profiles[key] = p
	return p
}

// RecordOutcome updates a profile with a verification outcome.
func (c *ConfidenceStore) RecordOutcome(role, model, verdict string) {
	p := c.GetOrCreate(role, model)
	c.mu.Lock()
	defer c.mu.Unlock()
	p.TotalTasks++
	p.LastSample = time.Now()
	switch verdict {
	case "CONFIRMED":
		p.ConfirmedCount++
	case "PLAUSIBLE":
		p.PlausibleCount++
	case "REFUTED":
		p.RefutedCount++
	}
	if p.TotalTasks > 0 {
		p.Accuracy = float64(p.ConfirmedCount) / float64(p.TotalTasks)
	}
}

// DecayOnModelBump halves profile weight when the model version changes.
// Weight recovers as new samples accumulate (up to 10 samples to full weight).
func (c *ConfidenceStore) DecayOnModelBump(newVersion string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range c.profiles {
		if p.ModelVersion != "" && p.ModelVersion != newVersion {
			p.Weight *= 0.5
			p.TotalTasks = 0 // reset sample count for recovery
		}
		p.ModelVersion = newVersion
	}
}

// BestProfile returns the highest-accuracy profile for a given role.
func (c *ConfidenceStore) BestProfile(role string) *ConfidenceProfile {
	c.mu.RLock()
	defer c.mu.RUnlock()
	var best *ConfidenceProfile
	for _, p := range c.profiles {
		if p.Role != role {
			continue
		}
		if best == nil || p.Accuracy*p.Weight > best.Accuracy*best.Weight {
			best = p
		}
	}
	return best
}
