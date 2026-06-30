package cli

import (
	"fmt"
	"strings"
	"time"

	"reasonix/internal/jobs"
)

// backgroundPanel renders an inline overview of running sub-agent background
// tasks. It preserves terminal scrollback by rendering as part of the normal
// output buffer (not an overlay), so the panel scrolls away like any other
// content as new messages arrive.
type backgroundPanel struct {
	// autoCollapseTimeout is how long a completed task stays visible before
	// collapsing. Default 3s.
	autoCollapseTimeout time.Duration

	// completed tracks when each completed task finished.
	completed map[string]time.Time

	// selectedIdx is the 0-based index of the highlighted row, or -1 for none.
	selectedIdx int

	// visible controls whether the panel is currently shown.
	visible bool
}

// newBackgroundPanel creates a panel with default settings.
func newBackgroundPanel() *backgroundPanel {
	return &backgroundPanel{
		autoCollapseTimeout: 3 * time.Second,
		completed:           make(map[string]time.Time),
		selectedIdx:         0,
	}
}

// Render returns the panel content as a string suitable for inline display.
// Returns empty string when there are no jobs to show.
func (bp *backgroundPanel) Render(views []jobs.View) string {
	if len(views) == 0 {
		bp.visible = false
		return ""
	}
	bp.visible = true

	now := time.Now()
	var running, done int
	var totalTokens int
	for _, v := range views {
		if v.Status == "running" {
			running++
		} else {
			done++
		}
		if v.Usage != nil {
			totalTokens += v.Usage.InputTokens + v.Usage.OutputTokens
		}
	}

	var b strings.Builder
	bp.writeHeader(&b, running, done, totalTokens)

	for i, v := range views {
		// Filter out completed jobs past auto-collapse timeout
		if v.Status != "running" {
			if finished, ok := bp.completed[v.ID]; ok {
				if now.Sub(finished) > bp.autoCollapseTimeout {
					if _, stillPresent := bp.completed[v.ID]; stillPresent {
						continue // skip collapsed jobs
					}
				}
			} else {
				bp.completed[v.ID] = now
				continue // newly completed, show brief flash then collapse next render
			}
		}
		bp.writeRow(&b, v, i == bp.selectedIdx)
	}

	bp.writeFooter(&b)
	return b.String()
}

func (bp *backgroundPanel) writeHeader(b *strings.Builder, running, done, tokens int) {
	icon := "●"
	if running == 0 {
		icon = "✓"
	}
	tok := ""
	if tokens > 1000 {
		tok = fmt.Sprintf(" · %.1fK tokens", float64(tokens)/1000)
	} else if tokens > 0 {
		tok = fmt.Sprintf(" · %d tokens", tokens)
	}
	b.WriteString(fmt.Sprintf("┌─ %s Background Tasks (%d running · %d done%s) ────┐\n",
		icon, running, done, tok))
}

func (bp *backgroundPanel) writeRow(b *strings.Builder, v jobs.View, selected bool) {
	// Icon by status
	icon, color := statusIcon(v.Status)
	_ = color

	// Label
	label := v.Label
	if v.Kind != "" && label == "" {
		label = v.Kind
	}
	if label == "" {
		label = v.ID
	}

	// Status tag
	statusTag := fmt.Sprintf("[%s]", v.Status)
	if v.Status == "running" {
		statusTag = "[running]"
	}

	// Tool info
	toolInfo := ""
	if v.ToolCalls > 0 {
		toolInfo = fmt.Sprintf("%d calls", v.ToolCalls)
	}
	if v.LastTool != "" {
		if toolInfo != "" {
			toolInfo += fmt.Sprintf("  last: %s", v.LastTool)
		} else {
			toolInfo = fmt.Sprintf("last: %s", v.LastTool)
		}
	}

	// Model/effort
	identity := ""
	if v.Model != "" {
		identity = v.Model
		if v.Effort != "" {
			identity += "/" + v.Effort
		}
	}

	// Remote indicator
	remoteTag := ""
	// (future: set by remote workers)

	// Line 1: icon + label + status + tool info
	selMarker := " "
	if selected {
		selMarker = "▶"
	}
	row := fmt.Sprintf("│ %s %s %-25s %-10s %-20s%s",
		selMarker, icon, truncateForPanel(label, 25), statusTag, toolInfo, remoteTag)

	if identity != "" {
		row += fmt.Sprintf(" %s", identity)
	}
	b.WriteString(row + "\n")

	// Line 2: reasoning tail or result preview
	reasoning := ""
	if v.LastReasoning != "" {
		reasoning = fmt.Sprintf("  \"%s\"", truncateForPanel(v.LastReasoning, 100))
	} else if v.ResultPreview != "" {
		reasoning = fmt.Sprintf("  \"%s\"", truncateForPanel(v.ResultPreview, 100))
	}
	if reasoning != "" {
		b.WriteString(fmt.Sprintf("│%s\n", reasoning))
	}

	// Dependencies
	if len(v.DependsOn) > 0 {
		b.WriteString(fmt.Sprintf("│   depends on: %s\n", strings.Join(v.DependsOn, ", ")))
	}
}

func (bp *backgroundPanel) writeFooter(b *strings.Builder) {
	b.WriteString("└─────────────────────────────────────────────────────────────┘\n")
	b.WriteString("  [Enter:peek] [s:send] [k:kill] [j/k:navigate] [Esc:close]\n")
}

// SelectNext moves the selection cursor down. Wraps at bottom.
func (bp *backgroundPanel) SelectNext(max int) {
	if bp.selectedIdx < max-1 {
		bp.selectedIdx++
	}
}

// SelectPrev moves the selection cursor up. Wraps at top.
func (bp *backgroundPanel) SelectPrev() {
	if bp.selectedIdx > 0 {
		bp.selectedIdx--
	}
}

// SelectFirst sets the cursor to the first row.
func (bp *backgroundPanel) SelectFirst() {
	bp.selectedIdx = 0
}

// ToggleVisibility shows/hides the panel.
func (bp *backgroundPanel) ToggleVisibility() {
	bp.visible = !bp.visible
}

// IsVisible reports whether the panel is currently shown.
func (bp *backgroundPanel) IsVisible() bool {
	return bp.visible
}

// SelectedID returns the ID of the currently selected job, or empty string.
func (bp *backgroundPanel) SelectedID(views []jobs.View) string {
	if bp.selectedIdx < 0 || bp.selectedIdx >= len(views) {
		return ""
	}
	return views[bp.selectedIdx].ID
}

// NotifyComplete records a job completion time for auto-collapse tracking.
func (bp *backgroundPanel) NotifyComplete(jobID string) {
	bp.completed[jobID] = time.Now()
}

// Reset clears all state (e.g. on session switch).
func (bp *backgroundPanel) Reset() {
	bp.completed = make(map[string]time.Time)
	bp.selectedIdx = 0
	bp.visible = false
}

// statusIcon returns the display icon for a job status.
func statusIcon(status string) (icon string, colorName string) {
	switch status {
	case "running":
		return "●", "green"
	case "done":
		return "✓", "green"
	case "failed":
		return "✗", "red"
	case "killed":
		return "⊘", "gray"
	default:
		return "○", "gray"
	}
}

// truncateForPanel limits a string to n characters with "..." suffix.
func truncateForPanel(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}
