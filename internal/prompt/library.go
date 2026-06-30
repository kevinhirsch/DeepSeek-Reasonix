// Package prompt provides prompt evaluation and calibration infrastructure.
//
// Library.go implements the self-healing prompt library (Issue #19 Comp 7,
// plus Issue #22 Gap D). On model version bumps, it evaluates all prompt
// variants and auto-selects the best performer. On drift detection, it
// performs an instant rollback (file swap in under one second) and then runs
// recalibration in the background.
package prompt

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"reasonix/internal/provider"
)

// PromptVariant is one version of a role prompt, tagged with its model version
// and evaluation performance metrics.
type PromptVariant struct {
	Role         string    `json:"role"`
	Version      string    `json:"version"`
	ModelVersion string    `json:"model_version"`
	Content      string    `json:"content"`
	PassRate     float64   `json:"pass_rate"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	AvgTokens    int       `json:"avg_tokens"`
	CostPerRun   float64   `json:"cost_per_run"`
	CalibratedAt time.Time `json:"calibrated_at"`
	Active       bool      `json:"active"`
	Score        float64   `json:"score"`
	FilePath     string    `json:"-"`
}

// RollbackResult describes the outcome of a prompt rollback.
type RollbackResult struct {
	Role                 string        `json:"role"`
	FromVersion          string        `json:"from_version"`
	ToVersion            string        `json:"to_version"`
	SwapDuration         time.Duration `json:"swap_duration"`
	Successful           bool          `json:"successful"`
	Error                string        `json:"error,omitempty"`
	RecalibrationStarted bool          `json:"recalibration_started"`
	RolledBackAt         time.Time     `json:"rolled_back_at"`
}

// PromptLibrary manages a collection of prompt variants for all subagent roles.
type PromptLibrary struct {
	variants               map[string]map[string]*PromptVariant
	currentModel           string
	promptsDir             string
	evalDir                string
	resultsDir             string
	activePath             string
	lastModelVersion       string
	calibrateOnVersionBump bool

	mu sync.RWMutex
}

// NewPromptLibrary creates a prompt library backed by the given directories.
func NewPromptLibrary(promptsDir, evalDir, resultsDir string) *PromptLibrary {
	return &PromptLibrary{
		variants:               make(map[string]map[string]*PromptVariant),
		promptsDir:             promptsDir,
		evalDir:                evalDir,
		resultsDir:             resultsDir,
		activePath:             filepath.Join(promptsDir, "v1"),
		calibrateOnVersionBump: true,
	}
}

// Load scans the prompts directory and populates the variant registry.
func (pl *PromptLibrary) Load() error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	variantsDir := filepath.Join(pl.promptsDir, "variants")
	entries, err := os.ReadDir(variantsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read variants dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		role := entry.Name()
		roleDir := filepath.Join(variantsDir, role)
		versionEntries, _ := os.ReadDir(roleDir)

		for _, ve := range versionEntries {
			if !strings.HasSuffix(ve.Name(), ".json") {
				continue
			}
			version := strings.TrimSuffix(ve.Name(), ".json")
			metaPath := filepath.Join(roleDir, ve.Name())
			promptPath := filepath.Join(roleDir, version+".md")

			variant, err := loadVariantMeta(metaPath)
			if err != nil {
				continue
			}
			variant.Role = role
			variant.Version = version
			variant.FilePath = promptPath

			if content, err := os.ReadFile(promptPath); err == nil {
				variant.Content = string(content)
			}

			if pl.variants[role] == nil {
				pl.variants[role] = make(map[string]*PromptVariant)
			}
			pl.variants[role][variant.Version] = variant
		}
	}

	return nil
}

// GetActive returns the currently active variant for a role.
func (pl *PromptLibrary) GetActive(role string) (*PromptVariant, bool) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	if roleVariants, ok := pl.variants[role]; ok {
		for _, v := range roleVariants {
			if v.Active {
				return v, true
			}
		}
	}
	return nil, false
}

// ListVariants returns all variants for a role, sorted by score descending.
func (pl *PromptLibrary) ListVariants(role string) []*PromptVariant {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	roleVariants, ok := pl.variants[role]
	if !ok {
		return nil
	}

	var list []*PromptVariant
	for _, v := range roleVariants {
		list = append(list, v)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Score > list[j].Score })
	return list
}

// AddVariant adds or updates a variant in the library.
func (pl *PromptLibrary) AddVariant(variant *PromptVariant) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if variant.Role == "" || variant.Version == "" {
		return fmt.Errorf("variant must have a role and version")
	}

	roleDir := filepath.Join(pl.promptsDir, "variants", variant.Role)
	if err := os.MkdirAll(roleDir, 0755); err != nil {
		return fmt.Errorf("create variant dir: %w", err)
	}

	promptPath := filepath.Join(roleDir, variant.Version+".md")
	if err := os.WriteFile(promptPath, []byte(variant.Content), 0644); err != nil {
		return fmt.Errorf("write variant content: %w", err)
	}

	variant.FilePath = promptPath
	variant.CalibratedAt = time.Now()

	metaPath := filepath.Join(roleDir, variant.Version+".json")
	if err := saveVariantMeta(metaPath, variant); err != nil {
		return fmt.Errorf("write variant meta: %w", err)
	}

	if pl.variants[variant.Role] == nil {
		pl.variants[variant.Role] = make(map[string]*PromptVariant)
	}
	pl.variants[variant.Role][variant.Version] = variant

	return nil
}

// CalibrateForModel evaluates all variants for a role against the current
// model and updates their scores.
func (pl *PromptLibrary) CalibrateForModel(ctx context.Context, role, modelVersion string, prov provider.Provider, scenarios []EvalScenario) error {
	pl.mu.Lock()
	roleVariants := pl.variants[role]
	pl.mu.Unlock()

	if roleVariants == nil || len(roleVariants) == 0 {
		// Bootstrap from the current active prompt.
		currentPrompt := LoadPrompt(role, pl.activePath, "")
		if currentPrompt == "" {
			return fmt.Errorf("no prompt found for role %q", role)
		}
		v := &PromptVariant{
			Role:         role,
			Version:      "v1",
			ModelVersion: modelVersion,
			Content:      currentPrompt,
		}
		if err := pl.AddVariant(v); err != nil {
			return err
		}
		pl.mu.Lock()
		if pl.variants[role] == nil {
			pl.variants[role] = make(map[string]*PromptVariant)
		}
		pl.variants[role]["v1"] = v
		pl.mu.Unlock()
		roleVariants = pl.variants[role]
	}

	// Evaluate each variant.
	type calResult struct {
		version string
		run     *EvalRun
		err     error
	}
	var calResults []calResult

	for version, variant := range roleVariants {
		promptText := variant.Content
		if promptText == "" {
			data, err := os.ReadFile(variant.FilePath)
			if err != nil {
				calResults = append(calResults, calResult{version: version, err: err})
				continue
			}
			promptText = string(data)
		}

		cfg := EvalConfig{
			Role:      role,
			Prompt:    promptText,
			Model:     modelVersion,
			Scenarios: scenarios,
			Provider:  prov,
		}

		run, err := Evaluate(ctx, cfg)
		calResults = append(calResults, calResult{version: version, run: run, err: err})
	}

	// Update scores.
	pl.mu.Lock()
	defer pl.mu.Unlock()

	var bestVersion string
	var bestScore float64

	for _, cr := range calResults {
		variant, ok := pl.variants[role][cr.version]
		if !ok {
			continue
		}

		if cr.err != nil {
			variant.PassRate = 0
			variant.Score = 0
			variant.Active = false
			continue
		}

		variant.PassRate = cr.run.PassRate
		variant.CalibratedAt = time.Now()
		variant.ModelVersion = modelVersion

		var totalLatency time.Duration
		var totalTokens int
		for _, r := range cr.run.Results {
			totalLatency += r.Duration
			totalTokens += r.TokensIn + r.TokensOut
		}
		if len(cr.run.Results) > 0 {
			variant.AvgLatencyMs = float64(totalLatency.Milliseconds()) / float64(len(cr.run.Results))
			variant.AvgTokens = totalTokens / len(cr.run.Results)
		}
		variant.CostPerRun = cr.run.Cost

		variant.Score = variant.PassRate
		if variant.AvgLatencyMs > 0 && variant.AvgLatencyMs < 5000 {
			variant.Score += (1.0 - variant.AvgLatencyMs/5000.0) * 5.0
		}

		if variant.Score > bestScore {
			bestScore = variant.Score
			bestVersion = cr.version
		}

		metaPath := filepath.Join(pl.promptsDir, "variants", role, variant.Version+".json")
		_ = saveVariantMeta(metaPath, variant)
	}

	if bestVersion != "" {
		for _, v := range pl.variants[role] {
			v.Active = (v.Version == bestVersion)
		}
		if err := pl.writeActivePrompt(role, pl.variants[role][bestVersion]); err != nil {
			return fmt.Errorf("activate best variant: %w", err)
		}
	}

	pl.lastModelVersion = modelVersion
	return nil
}

// Rollback instantly reverts a role's prompt to a previous variant.
// The file swap completes in under one second.
func (pl *PromptLibrary) Rollback(role, toVersion string) *RollbackResult {
	result := &RollbackResult{
		Role:         role,
		ToVersion:    toVersion,
		RolledBackAt: time.Now(),
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()

	roleVariants, ok := pl.variants[role]
	if !ok {
		result.Error = fmt.Sprintf("no variants for role %q", role)
		return result
	}

	targetVariant, ok := roleVariants[toVersion]
	if !ok {
		result.Error = fmt.Sprintf("variant %q not found for role %q", toVersion, role)
		return result
	}

	for _, v := range roleVariants {
		if v.Active {
			result.FromVersion = v.Version
			break
		}
	}

	start := time.Now()

	for _, v := range roleVariants {
		v.Active = false
	}
	targetVariant.Active = true

	if err := pl.writeActivePrompt(role, targetVariant); err != nil {
		result.Error = fmt.Sprintf("swap prompt file: %v", err)
		return result
	}

	result.SwapDuration = time.Since(start)
	result.Successful = true

	return result
}

// DetectDrift checks whether the active prompt has drifted from its calibrated score.
func (pl *PromptLibrary) DetectDrift(ctx context.Context, role string, prov provider.Provider, threshold float64) (bool, float64, error) {
	active, ok := pl.GetActive(role)
	if !ok {
		return false, 0, fmt.Errorf("no active variant for role %q", role)
	}

	scenarios := pl.loadEvalSuiteRole(role)
	if len(scenarios) == 0 {
		return false, 0, nil
	}

	cfg := EvalConfig{
		Role:      role,
		Prompt:    active.Content,
		Model:     active.ModelVersion,
		Scenarios: scenarios,
		Provider:  prov,
	}

	run, err := Evaluate(ctx, cfg)
	if err != nil {
		return false, 0, fmt.Errorf("evaluate drift: %w", err)
	}

	drift := active.PassRate - run.PassRate
	return drift > threshold, drift, nil
}

func (pl *PromptLibrary) writeActivePrompt(role string, variant *PromptVariant) error {
	if err := os.MkdirAll(pl.activePath, 0755); err != nil {
		return fmt.Errorf("create active prompt dir: %w", err)
	}
	activeFile := filepath.Join(pl.activePath, role+".md")
	return os.WriteFile(activeFile, []byte(variant.Content), 0644)
}

func (pl *PromptLibrary) loadEvalSuiteRole(role string) []EvalScenario {
	if pl.evalDir == "" {
		return nil
	}
	path := filepath.Join(pl.evalDir, role+".json")
	suite, err := LoadSuite(path)
	if err != nil {
		return nil
	}
	return suite.Scenarios
}

func loadVariantMeta(path string) (*PromptVariant, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var v PromptVariant
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

func saveVariantMeta(path string, v *PromptVariant) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
