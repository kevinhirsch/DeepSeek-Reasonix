// Package impact provides impact analysis pre-computation for code changes.
//
// Analyze.go implements impact analysis (Issue #20 Comp 18). When a function
// signature changes, the ImpactAnalyzer automatically analyzes every call site
// and reports what breaks, where, and how to fix it — all before the user asks.
package impact

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CallSite is one location where a changed symbol is referenced.
type CallSite struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column,omitempty"`
	SymbolName string `json:"symbol_name"`
	Context    string `json:"context"`
	Breaking   bool   `json:"breaking"`
	Reason     string `json:"reason,omitempty"`
}

// FixHint suggests how to update a call site for a breaking change.
type FixHint struct {
	CallSite     CallSite `json:"call_site"`
	Description  string   `json:"description"`
	BeforeCode   string   `json:"before_code,omitempty"`
	AfterCode    string   `json:"after_code,omitempty"`
	AutoFixable  bool     `json:"auto_fixable"`
}

// SignatureChange describes a change to a function or method signature.
type SignatureChange struct {
	Symbol       string   `json:"symbol"`
	File         string   `json:"file"`
	Line         int      `json:"line"`
	OldSignature string   `json:"old_signature"`
	NewSignature string   `json:"new_signature"`
	AddedParams  []string `json:"added_params,omitempty"`
	RemovedParams []string `json:"removed_params,omitempty"`
	RenamedParams []ParamRename `json:"renamed_params,omitempty"`
	TypeChanged  []ParamTypeChange `json:"type_changed,omitempty"`
	ReturnChanged bool     `json:"return_changed"`
}

// ParamRename records a renamed parameter.
type ParamRename struct {
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

// ParamTypeChange records a parameter type change.
type ParamTypeChange struct {
	ParamName string `json:"param_name"`
	OldType   string `json:"old_type"`
	NewType   string `json:"new_type"`
}

// ImpactReport is the full analysis of a signature change's impact.
type ImpactReport struct {
	Change        SignatureChange `json:"change"`
	CallSites     []CallSite      `json:"call_sites"`
	BreakingSites []CallSite      `json:"breaking_sites"`
	FixHints      []FixHint       `json:"fix_hints"`
	TotalAffected int             `json:"total_affected"`
	TotalBreaking int             `json:"total_breaking"`
	EstimatedFixTime string       `json:"estimated_fix_time"`
}

// ImpactAnalyzer analyzes the downstream effects of function signature changes.
// On every change, it scans the codebase to find every call site and reports
// what breaks, where, and how to fix it.
type ImpactAnalyzer struct {
	repoRoot string
	index    map[string][]fileSymbol // symbol name -> files/positions
}

type fileSymbol struct {
	File   string
	Line   int
	Column int
}

// NewImpactAnalyzer creates an impact analyzer for the given repository.
func NewImpactAnalyzer(repoRoot string) *ImpactAnalyzer {
	return &ImpactAnalyzer{
		repoRoot: repoRoot,
		index:    make(map[string][]fileSymbol),
	}
}

// BuildIndex scans the repository and builds a symbol-to-file index for fast
// call-site lookups. Should be called periodically or after significant changes.
func (a *ImpactAnalyzer) BuildIndex() error {
	a.index = make(map[string][]fileSymbol)

	// Walk Go source files in the repository.
	err := filepath.WalkDir(a.repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable files
		}
		if d.IsDir() {
			dirname := filepath.Base(path)
			if dirname == ".git" || dirname == "vendor" || dirname == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(a.repoRoot, path)
		lines := strings.Split(string(data), "\n")

		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Extract symbol definitions: function/method declarations.
			if sym, ok := extractSymbolDef(trimmed); ok {
				a.index[sym] = append(a.index[sym], fileSymbol{
					File:   relPath,
					Line:   i + 1,
					Column: strings.Index(line, sym) + 1,
				})
			}

			// Extract symbol references: function/method calls.
			for _, ref := range extractSymbolRefs(trimmed) {
				a.index[ref] = append(a.index[ref], fileSymbol{
					File:   relPath,
					Line:   i + 1,
					Column: strings.Index(line, ref) + 1,
				})
			}
		}

		return nil
	})

	return err
}

// AnalyzeChange analyzes the impact of a signature change on all call sites.
// Returns a comprehensive impact report including fix hints for each
// breaking call site.
func (a *ImpactAnalyzer) AnalyzeChange(change SignatureChange) (*ImpactReport, error) {
	if len(a.index) == 0 {
		_ = a.BuildIndex()
	}

	report := &ImpactReport{
		Change: change,
	}

	// Find all references to the changed symbol.
	refs, ok := a.index[change.Symbol]
	if !ok {
		return report, nil
	}

	for _, ref := range refs {
		if ref.File == change.File && ref.Line == change.Line {
			continue // skip the definition itself
		}

		context, _ := a.readLine(ref.File, ref.Line)
		site := CallSite{
			File:       ref.File,
			Line:       ref.Line,
			Column:     ref.Column,
			SymbolName: change.Symbol,
			Context:    strings.TrimSpace(context),
		}

		breaking, reason := a.isCallSiteBreaking(site, change)
		site.Breaking = breaking
		site.Reason = reason

		report.CallSites = append(report.CallSites, site)
		report.TotalAffected++

		if breaking {
			report.BreakingSites = append(report.BreakingSites, site)
			report.TotalBreaking++

			report.FixHints = append(report.FixHints, a.generateFixHint(site, change))
		}
	}

	sort.Slice(report.CallSites, func(i, j int) bool {
		if report.CallSites[i].File == report.CallSites[j].File {
			return report.CallSites[i].Line < report.CallSites[j].Line
		}
		return report.CallSites[i].File < report.CallSites[j].File
	})

	report.EstimatedFixTime = estimateFixTime(report.TotalBreaking)

	return report, nil
}

// AnalyzeFile analyzes the impact of all exported symbol changes in a file.
// This is the primary entry point: pass the changed file and the previous
// content to detect signature changes automatically.
func (a *ImpactAnalyzer) AnalyzeFile(file string, oldContent, newContent string) ([]*ImpactReport, error) {
	oldSigs := extractSignatures(oldContent)
	newSigs := extractSignatures(newContent)

	var changes []SignatureChange

	for sym, newSig := range newSigs {
		oldSig, existed := oldSigs[sym]
		if !existed {
			// New function — all call sites are unaffected (new code).
			continue
		}
		if oldSig == newSig {
			continue
		}
		changes = append(changes, diffSignatures(sym, file, oldSig, newSig))
	}

	// Also check for removed functions.
	for sym := range oldSigs {
		if _, exists := newSigs[sym]; !exists {
			changes = append(changes, SignatureChange{
				Symbol:       sym,
				File:         file,
				OldSignature: oldSigs[sym],
				NewSignature: "(removed)",
				ReturnChanged: true,
			})
		}
	}

	var reports []*ImpactReport
	for _, change := range changes {
		report, err := a.AnalyzeChange(change)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// isCallSiteBreaking determines whether a call site is broken by a signature change.
func (a *ImpactAnalyzer) isCallSiteBreaking(site CallSite, change SignatureChange) (bool, string) {
	// Function removed entirely.
	if change.NewSignature == "(removed)" {
		return true, fmt.Sprintf("%s was removed entirely", change.Symbol)
	}

	// Added required parameters.
	if len(change.AddedParams) > 0 {
		return true, fmt.Sprintf("new required parameters added: %s", strings.Join(change.AddedParams, ", "))
	}

	// Removed parameters that were used.
	if len(change.RemovedParams) > 0 {
		return true, fmt.Sprintf("parameters removed: %s", strings.Join(change.RemovedParams, ", "))
	}

	// Type changes.
	if len(change.TypeChanged) > 0 {
		var msgs []string
		for _, tc := range change.TypeChanged {
			msgs = append(msgs, fmt.Sprintf("%s: %s -> %s", tc.ParamName, tc.OldType, tc.NewType))
		}
		return true, fmt.Sprintf("parameter type changes: %s", strings.Join(msgs, "; "))
	}

	// Return type changed.
	if change.ReturnChanged {
		return true, "return type changed — callers may need adjustment"
	}

	return false, ""
}

// generateFixHint produces a suggested fix for a breaking call site.
func (a *ImpactAnalyzer) generateFixHint(site CallSite, change SignatureChange) FixHint {
	hint := FixHint{
		CallSite: site,
	}

	switch {
	case change.NewSignature == "(removed)":
		hint.Description = fmt.Sprintf("%s was removed. Find an alternative or reimplement the logic.", change.Symbol)
		hint.AutoFixable = false

	case len(change.AddedParams) > 0:
		hint.Description = fmt.Sprintf("Add the new parameters %s to the call at %s:%d.",
			strings.Join(change.AddedParams, ", "), site.File, site.Line)
		hint.AutoFixable = true
		for _, p := range change.AddedParams {
			hint.AfterCode += p + " (TODO: provide value), "
		}
		hint.AfterCode = strings.TrimSuffix(hint.AfterCode, ", ")

	case len(change.TypeChanged) > 0:
		var fixes []string
		for _, tc := range change.TypeChanged {
			fixes = append(fixes, fmt.Sprintf("convert %s from %s to %s", tc.ParamName, tc.OldType, tc.NewType))
		}
		hint.Description = strings.Join(fixes, "; ")
		hint.AutoFixable = false

	case change.ReturnChanged:
		hint.Description = "Adjust return value handling for the new signature."
		hint.AutoFixable = false

	default:
		hint.Description = "Review and update this call site."
		hint.AutoFixable = false
	}

	return hint
}

// readLine reads a single line from a file relative to the repo root.
func (a *ImpactAnalyzer) readLine(file string, line int) (string, error) {
	path := filepath.Join(a.repoRoot, file)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	lines := strings.Split(string(data), "\n")
	if line < 1 || line > len(lines) {
		return "", fmt.Errorf("line %d out of range for %s", line, file)
	}
	return lines[line-1], nil
}

// --- signature extraction helpers ---

// extractSignatures returns a map of symbol name -> signature text from Go source.
func extractSignatures(content string) map[string]string {
	sigs := make(map[string]string)
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if sym, ok := extractSymbolDef(trimmed); ok {
			// Extract the full signature (everything after "func SymbolName").
			prefix := "func " + sym
			if idx := strings.Index(trimmed, prefix); idx >= 0 {
				sig := strings.TrimSpace(trimmed[idx+len(prefix):])
				sigs[sym] = sig
			}
		}
	}

	return sigs
}

// extractSymbolDef extracts a function/method name from a declaration line.
func extractSymbolDef(line string) (string, bool) {
	// Match "func FuncName(" or "func (r *Receiver) MethodName("
	rest, ok := strings.CutPrefix(line, "func ")
	if !ok {
		return "", false
	}

	// Check for receiver: "func (r *Type) MethodName("
	if strings.HasPrefix(rest, "(") {
		closeParen := strings.Index(rest, ")")
		if closeParen < 0 {
			return "", false
		}
		rest = strings.TrimSpace(rest[closeParen+1:])
	}

	parenIdx := strings.Index(rest, "(")
	if parenIdx <= 0 {
		return "", false
	}

	name := strings.TrimSpace(rest[:parenIdx])
	// Filter out non-exported functions and invalid names.
	if name == "" || strings.ContainsAny(name, " \t") {
		return "", false
	}

	return name, true
}

// extractSymbolRefs extracts function call names from a line.
func extractSymbolRefs(line string) []string {
	var refs []string
	seen := make(map[string]bool)

	// Find patterns like "pkg.FuncName(" or "FuncName("
	for i := 0; i < len(line); i++ {
		if line[i] == '(' {
			// Walk backwards to find the function name.
			j := i - 1
			for j >= 0 && (isIdentByte(line[j]) || line[j] == '.') {
				j--
			}
			name := line[j+1 : i]
			if name != "" && !isKeyword(name) && !seen[name] {
				seen[name] = true
				// Only include exported names (starts with uppercase) for Go.
				if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
					refs = append(refs, name)
				}
			}
		}
	}

	return refs
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9') || b == '_'
}

func isKeyword(s string) bool {
	switch s {
	case "if", "for", "switch", "return", "func", "type", "var", "const",
		"import", "package", "go", "defer", "select", "case", "range",
		"chan", "map", "struct", "interface", "string", "int", "bool",
		"error", "nil", "true", "false", "make", "new", "len", "cap",
		"append", "copy", "close", "delete", "panic", "recover", "print",
		"println", "complex", "real", "imag":
		return true
	}
	return false
}

// diffSignatures compares two function signatures and produces a SignatureChange.
func diffSignatures(symbol, file, oldSig, newSig string) SignatureChange {
	change := SignatureChange{
		Symbol:       symbol,
		File:         file,
		OldSignature: oldSig,
		NewSignature: newSig,
	}

	oldParams := extractParams(oldSig)
	newParams := extractParams(newSig)

	// Detect added parameters.
	newSet := make(map[string]bool)
	for _, p := range newParams {
		newSet[p] = true
	}
	for _, p := range newParams {
		if !containsParam(oldParams, p) {
			change.AddedParams = append(change.AddedParams, p)
		}
	}

	// Detect removed parameters.
	oldSet := make(map[string]bool)
	for _, p := range oldParams {
		oldSet[p] = true
	}
	for _, p := range oldParams {
		if !newSet[p] {
			change.RemovedParams = append(change.RemovedParams, p)
		}
	}

	// Check return type change.
	oldRet := extractReturnType(oldSig)
	newRet := extractReturnType(newSig)
	change.ReturnChanged = oldRet != newRet

	return change
}

func extractParams(sig string) []string {
	// Extract content between the outermost parentheses for parameters.
	depth := 0
	start := -1
	for i, ch := range sig {
		if ch == '(' {
			if depth == 0 {
				start = i + 1
			}
			depth++
		} else if ch == ')' {
			depth--
			if depth == 0 && start >= 0 {
				content := sig[start:i]
				if content == "" {
					return nil
				}
				var params []string
				for _, p := range strings.Split(content, ",") {
					p = strings.TrimSpace(p)
					if p != "" {
						params = append(params, p)
					}
				}
				return params
			}
		}
	}
	return nil
}

func extractReturnType(sig string) string {
	// Find the last closing paren and get everything after it.
	lastClose := strings.LastIndex(sig, ")")
	if lastClose < 0 || lastClose >= len(sig)-1 {
		return ""
	}
	return strings.TrimSpace(sig[lastClose+1:])
}

func containsParam(params []string, target string) bool {
	for _, p := range params {
		if p == target {
			return true
		}
	}
	return false
}

func estimateFixTime(breakingCount int) string {
	switch {
	case breakingCount == 0:
		return "no fixes needed"
	case breakingCount <= 3:
		return "~5 minutes"
	case breakingCount <= 10:
		return "~15 minutes"
	case breakingCount <= 50:
		return "~1 hour"
	default:
		return fmt.Sprintf("~%d hours", breakingCount/30+1)
	}
}
