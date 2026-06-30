package knowledge

import (
	"fmt"
	"strings"
	"time"
)

// KBVerifier cross-validates knowledge base entries before storage,
// tags them with model version, and flags stale entries on access.
type KBVerifier struct {
	currentModelVersion string
}

// NewKBVerifier creates a KB verifier.
func NewKBVerifier(modelVersion string) *KBVerifier {
	return &KBVerifier{currentModelVersion: modelVersion}
}

// VerificationResult describes the outcome of cross-validation.
type VerificationResult struct {
	Passed    bool
	Reason    string
	Verified  bool
	ModelTag  string
}

// TagEntryWithVersion stamps an entry with the current model version.
func (v *KBVerifier) TagEntryWithVersion(e *Entry) {
	e.Model = v.currentModelVersion
}

// IsStale reports whether an entry was produced by a significantly
// different model version and should be re-verified.
func (v *KBVerifier) IsStale(e *Entry) (bool, string) {
	if e.Model == "" || v.currentModelVersion == "" {
		return false, ""
	}
	if e.Model != v.currentModelVersion {
		return true, fmt.Sprintf("entry produced by %s, current model is %s", e.Model, v.currentModelVersion)
	}
	if time.Since(e.CreatedAt) > 30*24*time.Hour {
		return true, "entry is over 30 days old"
	}
	return false, ""
}

// CrossValidate checks whether two independent analyses agree.
func (v *KBVerifier) CrossValidate(primary, secondary string) (bool, string) {
	primaryLower := strings.ToLower(primary)
	secondaryLower := strings.ToLower(secondary)

	primaryWords := strings.Fields(primaryLower)
	secondaryWords := strings.Fields(secondaryLower)

	if len(primaryWords) < 3 || len(secondaryWords) < 3 {
		return false, "insufficient content for cross-validation"
	}

	// Count keyword overlap
	matches := 0
	for _, pw := range primaryWords {
		for _, sw := range secondaryWords {
			if pw == sw {
				matches++
				break
			}
		}
	}
	overlap := float64(matches) / float64(len(primaryWords))
	if overlap >= 0.5 {
		return true, ""
	}
	return false, fmt.Sprintf("keyword overlap %.0f%% below threshold", overlap*100)
}
