// Package export provides session export with secret redaction.
package export

import (
	"regexp"
	"strings"
)

// Redactor strips secrets from export output.
type Redactor struct {
	rules []RedactRule
}

// RedactRule defines one pattern to redact.
type RedactRule struct {
	Pattern     string
	Replacement string
}

// DefaultRedactor returns a redactor with standard secret patterns.
func DefaultRedactor() *Redactor {
	return &Redactor{
		rules: []RedactRule{
			{Pattern: `Authorization:\s*\S+`, Replacement: "Authorization: [REDACTED]"},
			{Pattern: `x-api-key:\s*\S+`, Replacement: "x-api-key: [REDACTED]"},
			{Pattern: `api_key=\S+`, Replacement: "api_key=[REDACTED]"},
			{Pattern: `token=\S+`, Replacement: "token=[REDACTED]"},
			{Pattern: `https://[^@]+@`, Replacement: "https://[REDACTED]@"},
		},
	}
}

// Redact applies all redaction rules to the input.
func (r *Redactor) Redact(input string) string {
	result := input
	for _, rule := range r.rules {
		re := regexp.MustCompile(rule.Pattern)
		result = re.ReplaceAllString(result, rule.Replacement)
	}
	return result
}

// RedactPaths replaces home directory paths with ~/ equivalents.
func RedactPaths(input, homeDir string) string {
	if homeDir == "" {
		return input
	}
	return strings.ReplaceAll(input, homeDir, "~")
}
