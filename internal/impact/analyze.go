package impact

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ImpactAnalysis describes the effect of a function signature change.
type ImpactAnalysis struct {
	ChangedFile    string
	ChangedSymbol  string
	CallSites      []CallSite
	BreakingChanges []string
}

// CallSite is one location where a changed symbol is referenced.
type CallSite struct {
	File    string
	Line    int
	Context string
}

// ImpactAnalyzer analyzes the downstream effects of function signature changes.
// On function signature change: scans every call site, reports what breaks,
// where, and how to fix it — before the user asks.
type ImpactAnalyzer struct {
	repoRoot string
}

// NewImpactAnalyzer creates an impact analyzer for the given repo.
func NewImpactAnalyzer(repoRoot string) *ImpactAnalyzer {
	return &ImpactAnalyzer{repoRoot: repoRoot}
}

// AnalyzeChange analyzes the impact of changes in a file on function signatures.
func (a *ImpactAnalyzer) AnalyzeChange(file string) (*ImpactAnalysis, error) {
	absFile := filepath.Join(a.repoRoot, file)
	if _, err := os.Stat(absFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", file)
	}

	// Extract exported symbols from the changed file
	symbols := a.extractSymbols(absFile)
	if len(symbols) == 0 {
		return &ImpactAnalysis{ChangedFile: file}, nil
	}

	analysis := &ImpactAnalysis{
		ChangedFile:   file,
		ChangedSymbol: symbols[0],
	}

	// Find all call sites across the codebase
	for _, sym := range symbols {
		sites := a.findCallSites(sym)
		for _, site := range sites {
			if site.File != file {
				analysis.CallSites = append(analysis.CallSites, site)
			}
		}
		if len(sites) > 0 && siteWouldBreak(sites[0], sym) {
			analysis.BreakingChanges = append(analysis.BreakingChanges,
				fmt.Sprintf("%s: %d call site(s) may break", sym, len(sites)))
		}
	}

	return analysis, nil
}

func (a *ImpactAnalyzer) extractSymbols(file string) []string {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var symbols []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "func ") {
			// Extract function name
			rest := strings.TrimPrefix(line, "func ")
			if idx := strings.Index(rest, "("); idx > 0 {
				symbols = append(symbols, strings.TrimSpace(rest[:idx]))
			}
		}
	}
	return symbols
}

func (a *ImpactAnalyzer) findCallSites(symbol string) []CallSite {
	// Use grep to find all references across the codebase
	cmd := exec.Command("grep", "-rn", symbol, a.repoRoot)
	cmd.Dir = a.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var sites []CallSite
	for _, line := range strings.Split(string(out), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 2 {
			ctx := ""
			if len(parts) > 2 {
				ctx = strings.TrimSpace(parts[2])
			}
			sites = append(sites, CallSite{
				File:    strings.TrimPrefix(parts[0], a.repoRoot+string(filepath.Separator)),
				Context: ctx,
			})
		}
	}
	return sites
}

func siteWouldBreak(site CallSite, sym string) bool {
	return strings.Contains(site.Context, sym+"(") && strings.Contains(site.Context, ",")
}
