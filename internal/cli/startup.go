package cli

import (
	"fmt"
	"time"
)

// StartupPhase represents a named startup stage for the cold-start spinner.
type StartupPhase struct {
	Name     string
	started  time.Time
	done     bool
	err      error
}

// StartupProgress tracks multi-phase startup with spinner text.
type StartupProgress struct {
	phases []StartupPhase
}

// NewStartupProgress creates a startup progress tracker with standard phases.
func NewStartupProgress() *StartupProgress {
	return &StartupProgress{
		phases: []StartupPhase{
			{Name: "Loading config"},
			{Name: "Checking providers"},
			{Name: "Loading skills"},
			{Name: "Ready"},
		}
	}
}

// Complete marks a phase as done. Returns the display text for the next phase.
func (sp *StartupProgress) Complete(index int) string {
	if index < 0 || index >= len(sp.phases) {
		return ""
	}
	sp.phases[index].done = true
	sp.phases[index].started = time.Now()
	if index+1 < len(sp.phases) {
		sp.phases[index+1].started = time.Now()
		return sp.render()
	}
	return sp.render()
}

// Fail marks a phase as failed.
func (sp *StartupProgress) Fail(index int, err error) string {
	if index < 0 || index >= len(sp.phases) {
		return ""
	}
	sp.phases[index].err = err
	return sp.render()
}

func (sp *StartupProgress) render() string {
	var result string
	for _, p := range sp.phases {
		switch {
		case p.err != nil:
			result += fmt.Sprintf("✗ %s: %v\n", p.Name, p.err)
		case p.done:
			result += fmt.Sprintf("✓ %s\n", p.Name)
		case !p.started.IsZero():
			result += fmt.Sprintf("… %s\n", p.Name)
		default:
			result += fmt.Sprintf("  %s\n", p.Name)
		}
	}
	return result
}

// CrashRecoveryMessage returns the recovery notice for session restore.
func CrashRecoveryMessage(sessionCount int) string {
	if sessionCount == 0 {
		return ""
	}
	return fmt.Sprintf("Recovering %d session(s) from previous run. This may take a moment.", sessionCount)
}

// UpdateAvailableNotice returns the update notification.
func UpdateAvailableNotice(currentVersion, newVersion string) string {
	return fmt.Sprintf("reasonix %s is available (current: %s). Run 'reasonix upgrade' to update.",
		newVersion, currentVersion)
}
