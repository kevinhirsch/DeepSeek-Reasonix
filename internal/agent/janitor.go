package agent

import (
	"context"
	"fmt"
	"strings"

	"reasonix/internal/provider"
	"reasonix/internal/event"
)

// ContextJanitor semantically compacts long-running sessions every N turns.
// A cheap Flash subagent reads the session history and drops dead ends,
// keeping only what matters for the current task context.
type ContextJanitor struct {
	enabled      bool
	interval     int // compact every N turns
	turnsSince   int
	compactModel string
	taskTool     *TaskTool
}

// NewContextJanitor creates a context janitor.
func NewContextJanitor(taskTool *TaskTool) *ContextJanitor {
	return &ContextJanitor{
		enabled:      true,
		interval:     5,
		compactModel: "deepseek-v4-flash",
		taskTool:     taskTool,
	}
}

// MaybeCompact checks whether a compaction is due this turn.
// Returns (compacted messages, error).
func (j *ContextJanitor) MaybeCompact(ctx context.Context, messages []provider.Message) (string, error) {
	if !j.enabled {
		return "", nil
	}
	j.turnsSince++
	if j.turnsSince < j.interval {
		return "", nil
	}
	j.turnsSince = 0
	return j.compact(ctx, messages)
}


func truncateMessageContent(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n-3] + "..."
}
func (j *ContextJanitor) compact(ctx context.Context, messages []provider.Message) (string, error) {
	if len(messages) < 10 {
		return "", nil
	}
	// Build a summary prompt
	var b strings.Builder
	b.WriteString("Summarize this conversation history. Keep: task goals, decisions made, ")
	b.WriteString("files changed, errors encountered, and key findings. Drop: dead ends, ")
	b.WriteString("unsuccessful attempts, repeated content, and routine tool output.\n\n")
	for i, m := range messages {
		if i > 50 { // only feed last 50 messages for context
			break
		}
		if m.Role == provider.RoleUser {
			b.WriteString("User: " + truncateMessageContent(m.Content, 200) + "\n")
		} else if m.Role == provider.RoleAssistant && strings.TrimSpace(m.Content) != "" {
			b.WriteString("Assistant: " + truncateMessageContent(m.Content, 200) + "\n")
		}
	}
	prompt := b.String()
	subReg := SubagentToolRegistry(j.taskTool.parentReg, nil)
	sess := NewSession("You are a summarization subagent. Return a bullet-point summary.")

	result, err := RunSubAgentWithSession(ctx, j.taskTool.prov, subReg, sess, prompt, Options{
		MaxSteps:  5,
	}, event.Discard)
	if err != nil {
		return "", fmt.Errorf("context janitor compaction: %w", err)
	}
	return result, nil
}

// Enable or disable the janitor.
func (j *ContextJanitor) Enable(v bool) { j.enabled = v }
