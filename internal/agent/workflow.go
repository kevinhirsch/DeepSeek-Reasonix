package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Workflow strategy constants.
type WorkflowStrategy string

const (
	StrategyPipeline     WorkflowStrategy = "pipeline"
	StrategyBarrier      WorkflowStrategy = "barrier"
	StrategyLoopUntilDry WorkflowStrategy = "loop_until_dry"
)

// Verification modes.
type WorkflowMode string

const (
	WorkflowPrecision WorkflowMode = "precision"
	WorkflowRecall    WorkflowMode = "recall"
)

// WorkflowStage defines one stage in a multi-stage workflow.
type WorkflowStage struct {
	Name           string `json:"name"`            // human-readable label
	Role           string `json:"role"`            // explorer | reviewer | verifier | planner | executor
	PromptTemplate string `json:"prompt_template"` // can contain {{item}} placeholder
	ItemsFrom      string `json:"items_from"`      // "items" | "stage_N" — where inputs come from
	FanOut         int    `json:"fan_out"`         // N parallel sub-agents per item
	Voting         string `json:"voting"`          // "majority" | "unanimous" | "" (none)
	Model          string `json:"model"`           // optional model override
	Effort         string `json:"effort"`          // optional effort override
}

// WorkflowSpec is the model-supplied workflow definition.
type WorkflowSpec struct {
	Strategy  WorkflowStrategy `json:"strategy"`
	Stages    []WorkflowStage  `json:"stages"`
	Items     []string         `json:"items"`      // input items for pipeline/barrier
	DryRounds int              `json:"dry_rounds"` // for loop_until_dry
	Mode      WorkflowMode     `json:"mode"`       // "precision" | "recall"
	MaxTasks  int              `json:"max_tasks"`  // safety cap (default 50)
}

// WorkflowTool lets the model compose multi-stage subagent strategies
// (pipeline, barrier, loop-until-dry) in a single tool call. It reuses
// TaskTool.runSubSession() for subagent dispatch — orchestration layer,
// not a new engine.
type WorkflowTool struct {
	taskTool *TaskTool
	reg      *tool.Registry
}

// NewWorkflowTool creates a workflow tool that reuses the given TaskTool's
// subagent infrastructure.
func NewWorkflowTool(taskTool *TaskTool, reg *tool.Registry) *WorkflowTool {
	return &WorkflowTool{taskTool: taskTool, reg: reg}
}

func (w *WorkflowTool) Name() string { return "workflow" }

func (w *WorkflowTool) Description() string {
	return "Compose multi-stage subagent strategies in one call. Strategies: pipeline (items flow independently through stages), barrier (all items finish stage N before N+1 starts), loop_until_dry (keep spawning finders until K consecutive rounds produce nothing new). Verification modes: precision (3 finders, 1-vote verify) or recall (5 finders, 3-vote verify, PLAUSIBLE included)."
}

func (w *WorkflowTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "strategy":{"type":"string","enum":["pipeline","barrier","loop_until_dry"],"description":"How items flow through stages. pipeline: items progress independently. barrier: all finish stage N before N+1. loop_until_dry: keep spawning finders until no new results."},
  "stages":{"type":"array","minItems":1,"maxItems":5,"items":{"type":"object","properties":{
    "name":{"type":"string","description":"Human-readable stage label."},
    "role":{"type":"string","description":"Subagent role: explorer, reviewer, verifier, or executor."},
    "prompt_template":{"type":"string","description":"Subagent prompt. Use {{item}} to inject the current item and {{stage_N_output}} for prior stage results."},
    "items_from":{"type":"string","description":"Where this stage gets its inputs: \"items\" (top-level items) or \"stage_N\" (output from a previous stage)."},
    "fan_out":{"type":"integer","minimum":1,"maximum":10,"description":"Number of parallel subagents per item (default 1)."},
    "voting":{"type":"string","enum":["majority","unanimous",""],"description":"How to combine fan_out results. majority: >50% must agree. unanimous: all must agree. Empty: no voting, raw output passed through."},
    "model":{"type":"string","description":"Optional model override."},
    "effort":{"type":"string","description":"Optional reasoning effort override."}
  },"required":["name","role","prompt_template"]}},
  "items":{"type":"array","items":{"type":"string"},"description":"Input items for pipeline/barrier strategies. Ignored for loop_until_dry."},
  "dry_rounds":{"type":"integer","minimum":1,"maximum":5,"description":"For loop_until_dry: stop after this many consecutive rounds with nothing new. Default 2."},
  "mode":{"type":"string","enum":["precision","recall"],"description":"Verification mode. precision: 3 finders, 1-vote verify (default). recall: 5 finders, 3-vote verify, PLAUSIBLE included."},
  "max_tasks":{"type":"integer","minimum":1,"maximum":100,"description":"Safety cap on total subagent spawns. Default 50."}
},
"required":["strategy","stages"]
}`)
}

func (w *WorkflowTool) ReadOnly() bool { return false }

func (w *WorkflowTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var spec WorkflowSpec
	if err := json.Unmarshal(args, &spec); err != nil {
		return "", fmt.Errorf("invalid workflow args: %w", err)
	}
	if len(spec.Stages) == 0 {
		return "", fmt.Errorf("at least one stage is required")
	}
	if spec.DryRounds <= 0 {
		spec.DryRounds = 2
	}
	if spec.MaxTasks <= 0 {
		spec.MaxTasks = 50
	}
	if spec.Mode == "" {
		spec.Mode = WorkflowPrecision
	}

	// Validate strategy-specific requirements
	switch spec.Strategy {
	case StrategyPipeline, StrategyBarrier:
		if len(spec.Items) == 0 {
			return "", fmt.Errorf("%s strategy requires at least one item", spec.Strategy)
		}
	case StrategyLoopUntilDry:
		// loop_until_dry doesn't need pre-defined items — generates its own
		if len(spec.Stages) != 1 {
			return "", fmt.Errorf("loop_until_dry supports exactly one stage (finder)")
		}
	}

	// Enforce max tasks cap BEFORE any subagent spawns
	estimated := w.estimateTaskCount(spec)
	if estimated > spec.MaxTasks {
		return "", fmt.Errorf("estimated %d subagent spawns exceeds max_tasks cap of %d; reduce stages, items, or fan_out", estimated, spec.MaxTasks)
	}

	// Dispatch to strategy
	switch spec.Strategy {
	case StrategyPipeline:
		return w.runPipeline(ctx, spec)
	case StrategyBarrier:
		return w.runBarrier(ctx, spec)
	case StrategyLoopUntilDry:
		return w.runLoopUntilDry(ctx, spec)
	default:
		return "", fmt.Errorf("unknown strategy: %s", spec.Strategy)
	}
}

// estimateTaskCount computes the maximum number of subagent spawns for cap enforcement.
func (w *WorkflowTool) estimateTaskCount(spec WorkflowSpec) int {
	count := 0
	for _, stage := range spec.Stages {
		items := len(spec.Items)
		if stage.ItemsFrom != "" && stage.ItemsFrom != "items" {
			items = len(spec.Items) // assume same cardinality from prior stage
		}
		fanOut := stage.FanOut
		if fanOut <= 0 {
			fanOut = 1
		}
		// Voting stages spawn fanOut × 3 verifiers per item
		if stage.Voting != "" {
			count += items * fanOut * 3
		} else {
			count += items * fanOut
		}
	}
	return count
}

// ---------------------------------------------------------------------------
// Pipeline strategy — items flow through stages independently.
// Item 1 can be at stage 3 while Item 2 is still at stage 1.
// ---------------------------------------------------------------------------

func (w *WorkflowTool) runPipeline(ctx context.Context, spec WorkflowSpec) (string, error) {
	n := len(spec.Items)
	m := len(spec.Stages)

	type itemResult struct {
		itemIndex int
		stageName string
		output    string
	}

	results := make([][]string, n)
	for i := range results {
		results[i] = make([]string, m)
	}

	var wg sync.WaitGroup
	// One goroutine per item — each processes stages sequentially (no barrier)
	for i, item := range spec.Items {
		wg.Add(1)
		go func(itemIdx int, prompt string) {
			defer wg.Done()
			for stageIdx, stage := range spec.Stages {
				select {
				case <-ctx.Done():
					return
				default:
				}
				// Build prompt from template + prior stage outputs
				stagePrompt := w.expandPrompt(stage.PromptTemplate, prompt, results[itemIdx], stageIdx)
				output, err := w.dispatchStage(ctx, stage, stagePrompt, itemIdx, stageIdx)
				if err != nil && ctx.Err() == nil {
					results[itemIdx][stageIdx] = fmt.Sprintf("[ERROR: %v]", err)
				} else {
					results[itemIdx][stageIdx] = output
				}
			}
		}(i, item)
	}
	wg.Wait()

	return w.formatPipelineResults(spec, results)
}

// ---------------------------------------------------------------------------
// Barrier strategy — all items finish stage N before any starts N+1.
// ---------------------------------------------------------------------------

func (w *WorkflowTool) runBarrier(ctx context.Context, spec WorkflowSpec) (string, error) {
	n := len(spec.Items)
	m := len(spec.Stages)
	results := make([][]string, n)
	for i := range results {
		results[i] = make([]string, m)
	}

	for stageIdx, stage := range spec.Stages {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		// Parallel across items within this stage
		var wg sync.WaitGroup
		for itemIdx, item := range spec.Items {
			wg.Add(1)
			go func(iidx int, prompt string, sidx int, s WorkflowStage) {
				defer wg.Done()
				stagePrompt := w.expandPrompt(s.PromptTemplate, prompt, results[iidx], sidx)
				output, err := w.dispatchStage(ctx, s, stagePrompt, iidx, sidx)
				if err != nil && ctx.Err() == nil {
					results[iidx][sidx] = fmt.Sprintf("[ERROR: %v]", err)
				} else {
					results[iidx][sidx] = output
				}
			}(itemIdx, item, stageIdx, stage)
		}
		wg.Wait()
	}

	return w.formatPipelineResults(spec, results)
}

// ---------------------------------------------------------------------------
// Loop-until-dry — keep spawning finders until K rounds with nothing new.
// Deduplicates findings by content hash.
// ---------------------------------------------------------------------------

func (w *WorkflowTool) runLoopUntilDry(ctx context.Context, spec WorkflowSpec) (string, error) {
	finder := spec.Stages[0]
	dryRounds := spec.DryRounds

	seen := make(map[string]bool)
	var allFindings []string
	dryCount := 0
	round := 0

	for round < 50 { // hard safety limit
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		round++

		// Build prompt with prior findings context
		prompt := finder.PromptTemplate
		if len(allFindings) > 0 {
			prompt += "\n\nPreviously found (do NOT repeat these):\n" + strings.Join(allFindings, "\n")
		}

		output, err := w.dispatchStage(ctx, finder, prompt, 0, 0)
		if err != nil {
			return "", err
		}

		// Extract new findings
		newFindings := w.extractFindings(output)
		var fresh []string
		for _, f := range newFindings {
			key := dedupKey(f)
			if !seen[key] {
				seen[key] = true
				fresh = append(fresh, f)
			}
		}

		if len(fresh) == 0 {
			dryCount++
			if dryCount >= dryRounds {
				break
			}
		} else {
			dryCount = 0
			allFindings = append(allFindings, fresh...)
		}
	}

	return w.formatLoopResults(allFindings, round, dryCount), nil
}

// ---------------------------------------------------------------------------
// Stage dispatch — runs a single stage against one item.
// Handles fan_out and voting.
// ---------------------------------------------------------------------------

func (w *WorkflowTool) dispatchStage(ctx context.Context, stage WorkflowStage, prompt string, itemIdx, stageIdx int) (string, error) {
	fanOut := stage.FanOut
	if fanOut <= 0 {
		fanOut = 1
	}

	if fanOut == 1 {
		// Simple single-subagent dispatch
		return w.runStageSubagent(ctx, stage, prompt, 0)
	}

	// Multiple parallel sub-agents per item
	var wg sync.WaitGroup
	outputs := make([]string, fanOut)
	errs := make([]error, fanOut)

	for f := 0; f < fanOut; f++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			outputs[idx], errs[idx] = w.runStageSubagent(ctx, stage, prompt, idx)
		}(f)
	}
	wg.Wait()

	// Collect non-error outputs
	var valid []string
	for i, o := range outputs {
		if errs[i] == nil && strings.TrimSpace(o) != "" {
			valid = append(valid, o)
		}
	}

	if len(valid) == 0 {
		return "", fmt.Errorf("all %d subagent(s) failed for stage %q", fanOut, stage.Name)
	}

	// If no voting, concatenate results
	if stage.Voting == "" {
		return strings.Join(valid, "\n\n"), nil
	}

	// Voting: extract verdicts from each output and apply threshold
	return w.applyVoting(valid, stage.Voting), nil
}

// runStageSubagent dispatches a single subagent for a workflow stage.
func (w *WorkflowTool) runStageSubagent(ctx context.Context, stage WorkflowStage, prompt string, fanIdx int) (string, error) {
	// Build a sub-registry appropriate for the role
	subReg := SubagentToolRegistry(w.taskTool.parentReg, nil)

	// Read-only stage roles get a read-only registry
	isReadOnly := stage.Role == "explorer" || stage.Role == "reviewer" ||
		stage.Role == "verifier" || stage.Role == "planner"
	if isReadOnly {
		subReg = ReadOnlySubagentToolRegistry(w.taskTool.parentReg, nil)
	}

	// Resolve model and effort
	modelRef, effortRef := w.taskTool.effectiveProfile(stage.Model, stage.Effort)
	prov, pricing, ctxWin, err := resolveSubagentProvider(w.taskTool, modelRef, effortRef)
	if err != nil {
		return "", err
	}

	// Create session with role-specific system prompt
	sysPrompt := w.taskTool.sysPrompt
	rolePrompt := RolePrompt(stage.Role, false) // non-DeepSeek default
	if rolePrompt != "" {
		sysPrompt = rolePrompt
	}
	sess := NewSession(sysPrompt)

	steps := w.taskTool.maxSteps
	if steps > 0 && steps /= 2; steps < 5 {
		steps = 5
	}
	if isReadOnly && steps > 0 && steps /= 2; steps < 5 {
		steps = 5
	}

	return RunSubAgentWithSession(ctx, prov, subReg, sess, prompt, Options{
		MaxSteps:            steps,
		Temperature:         w.taskTool.temperature,
		Pricing:             pricing,
		UsageSource:         "subagent",
		Gate:                w.taskTool.gate,
		ContextWindow:       ctxWin,
		RecentKeep:          w.taskTool.recentKeep,
		SoftCompactRatio:    w.taskTool.softCompactRatio,
		ToolResultSnipRatio: w.taskTool.toolResultSnipRatio,
		CompactRatio:        w.taskTool.compactRatio,
		CompactForceRatio:   w.taskTool.compactForceRatio,
		ArchiveDir:          w.taskTool.archiveDir,
		KeepPolicy:          w.taskTool.keepPolicy,
		IsDeepSeek:          w.taskTool.isDeepSeek,
	}, Discard)
}

// ---------------------------------------------------------------------------
// Voting
// ---------------------------------------------------------------------------

// applyVoting aggregates fan_out results using the specified voting strategy.
func applyVoting(outputs []string, strategy string) string {
	// Parse each output for verdicts
	type verdictCount struct {
		confirmed int
		plausible int
		refuted   int
	}
	var verdicts []verdictCount

	for _, output := range outputs {
		vc := verdictCount{}
		lower := strings.ToLower(output)
		// Count verdict occurrences in output
		vc.confirmed = strings.Count(lower, "confirmed")
		vc.plausible = strings.Count(lower, "plausible")
		vc.refuted = strings.Count(lower, "refuted")
		verdicts = append(verdicts, vc)
	}

	total := len(verdicts)
	if total == 0 {
		return strings.Join(outputs, "\n\n")
	}

	required := total/2 + 1 // majority default
	if strategy == "unanimous" {
		required = total
	}

	// Count how many outputs claim CONFIRMED (at least one CONFIRMED mention, no REFUTED)
	confirmed := 0
	plausible := 0
	for _, vc := range verdicts {
		if vc.confirmed > 0 && vc.refuted == 0 {
			confirmed++
		} else if vc.plausible > 0 || vc.confirmed > 0 {
			plausible++
		}
	}

	var b strings.Builder
	if confirmed >= required {
		fmt.Fprintf(&b, "CONFIRMED (%d/%d verifiers agree)\n\n", confirmed, total)
	} else if strategy != "unanimous" && confirmed+plausible >= required {
		fmt.Fprintf(&b, "PLAUSIBLE (%d/%d verifiers — confirmed %d, plausible %d)\n\n", confirmed+plausible, total, confirmed, plausible)
	} else {
		fmt.Fprintf(&b, "REFUTED (%d/%d verifiers did not confirm)\n\n", total-confirmed, total)
	}
	b.WriteString(strings.Join(outputs, "\n\n———\n\n"))
	return b.String()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// expandPrompt replaces {{item}} and {{stage_N_output}} placeholders.
func (w *WorkflowTool) expandPrompt(template, item string, prior []string, stageIdx int) string {
	result := template
	if strings.Contains(result, "{{item}}") {
		result = strings.ReplaceAll(result, "{{item}}", item)
	}
	// Replace {{stage_N_output}} with prior stage results
	for i, output := range prior {
		if i >= stageIdx {
			break
		}
		placeholder := fmt.Sprintf("{{stage_%d_output}}", i+1)
		if strings.Contains(result, placeholder) {
			result = strings.ReplaceAll(result, placeholder, output)
		}
	}
	return result
}

// extractFindings splits a finder output into individual finding lines.
func (w *WorkflowTool) extractFindings(output string) []string {
	lines := strings.Split(output, "\n")
	var findings []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Lines starting with "- file:" or "- " are findings
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			findings = append(findings, line)
		}
	}
	return findings
}

// dedupKey creates a stable key for deduplication by normalizing the finding text.
func dedupKey(finding string) string {
	key := strings.TrimSpace(finding)
	key = strings.TrimPrefix(key, "- ")
	key = strings.TrimPrefix(key, "* ")
	return strings.TrimSpace(key)
}

// formatPipelineResults aggregates results from pipeline/barrier strategies.
func (w *WorkflowTool) formatPipelineResults(spec WorkflowSpec, results [][]string) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "Workflow complete (%s, %d items × %d stages)\n\n", spec.Strategy, len(spec.Items), len(spec.Stages))
	for si, stage := range spec.Stages {
		fmt.Fprintf(&b, "## Stage: %s\n", stage.Name)
		for ii, item := range spec.Items {
			output := results[ii][si]
			if output == "" {
				output = "(no output)"
			}
			fmt.Fprintf(&b, "### Item: %s\n%s\n\n", truncateItem(item, 80), output)
		}
	}
	return b.String(), nil
}

// formatLoopResults formats the loop-until-dry output.
func (w *WorkflowTool) formatLoopResults(findings []string, rounds, dryCount int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Loop-until-dry complete after %d rounds (%d findings, stopped after %d dry rounds)\n\n", rounds, len(findings), dryCount)
	for _, f := range findings {
		b.WriteString(f)
		b.WriteString("\n")
	}
	return b.String()
}

func truncateItem(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
