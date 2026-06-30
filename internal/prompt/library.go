package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// PromptVariant is one versioned prompt file.
type PromptVariant struct {
	Version   string
	Role      string
	Content   string
	PassRate  float64
	CreatedAt time.Time
	Active    bool
}

// PromptLibrary manages a collection of prompt variants, auto-selecting the
// best on model version bump. Supports instant rollback on drift detection.
// On model version bump: evaluate all variants, auto-select best.
// On drift: rollback to last known good in <1s via file swap.
type PromptLibrary struct {
	mu        sync.RWMutex
	promptsDir string
	variants   map[string][]PromptVariant // role → variants
}

// NewPromptLibrary opens the prompt directory.
func NewPromptLibrary(promptsDir string) *PromptLibrary {
	lib := &PromptLibrary{
		promptsDir: promptsDir,
		variants:   make(map[string][]PromptVariant),
	}
	lib.load()
	return lib
}

// SelectBest returns the highest-scoring variant for a role.
func (l *PromptLibrary) SelectBest(role string) *PromptVariant {
	l.mu.RLock()
	defer l.mu.RUnlock()
	variants := l.variants[role]
	if len(variants) == 0 {
		return nil
	}
	sort.Slice(variants, func(i, j int) bool {
		return variants[i].PassRate > variants[j].PassRate
	})
	return &variants[0]
}

// RecordVariant stores an evaluated variant.
func (l *PromptLibrary) RecordVariant(v PromptVariant) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.variants[v.Role] = append(l.variants[v.Role], v)
}

// Rollback reverts to the last known good version for a role.
// Returns the rolled-back variant or nil if none found.
func (l *PromptLibrary) Rollback(role string) (*PromptVariant, error) {
	l.mu.RLock()
	variants := l.variants[role]
	l.mu.RUnlock()

	if len(variants) == 0 {
		return nil, fmt.Errorf("no variants for role %q", role)
	}

	sort.Slice(variants, func(i, j int) bool {
		return variants[i].PassRate > variants[j].PassRate
	})

	best := variants[0]
	// File swap: write the best variant's content to the active file
	activePath := filepath.Join(l.promptsDir, role+".md")
	backupPath := activePath + ".backup." + time.Now().Format("20060102-150405")

	if err := os.Rename(activePath, backupPath); err == nil {
		os.WriteFile(activePath, []byte(best.Content), 0644)
		return &best, nil
	}
	return nil, fmt.Errorf("rollback file swap failed for %q", role)
}

// GetAllRoles returns all roles with stored variants.
func (l *PromptLibrary) GetAllRoles() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var roles []string
	for r := range l.variants {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	return roles
}

func (l *PromptLibrary) load() {
	entries, _ := os.ReadDir(l.promptsDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, _ := os.ReadFile(filepath.Join(l.promptsDir, e.Name()))
		role := strings.TrimSuffix(e.Name(), ".md")
		l.variants[role] = append(l.variants[role], PromptVariant{
			Version:   "v1",
			Role:      role,
			Content:   string(data),
			PassRate:  100.0,
			CreatedAt: time.Now(),
			Active:    true,
		})
	}
}
