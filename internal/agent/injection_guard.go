package agent

import "strings"

// InjectionGuard checks tool outputs and file content for prompt injection
// patterns before they enter the agent's context. Non-blocking — flagged
// output gets a warning annotation but still enters context.
type InjectionGuard struct {
	enabled bool
}

// NewInjectionGuard creates an injection guard.
func NewInjectionGuard() *InjectionGuard {
	return &InjectionGuard{enabled: true}
}

// Enable or disable the guard.
func (g *InjectionGuard) Enable(v bool) { g.enabled = v }

// Check scans output for known injection markers. Returns the output
// (possibly annotated) and any warnings.
func (g *InjectionGuard) Check(output string) (string, []string) {
	if !g.enabled || output == "" {
		return output, nil
	}
	var warnings []string
	lower := strings.ToLower(output)
	markers := []string{
		"ignore previous instructions",
		"you are now",
		"<system-reminder>",
		"system:",
		"as an ai language model",
		"your new instructions are",
		"forget everything you know",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			warnings = append(warnings, "injection pattern detected: \""+m+"\"")
		}
	}
	if len(warnings) > 0 {
		output = "[⚠ Content flagged for injection patterns] " + output
	}
	return output, warnings
}
