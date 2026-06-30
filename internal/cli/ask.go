package cli

import (
	"fmt"
	"strings"

	"reasonix/internal/knowledge"
)

// AskPanel is a focused Q&A mode for natural-language codebase questions.
// Uses the knowledge base for instant answers and spawns explorers when needed.
// Activated via /ask command.

// AskConfig configures the ask panel behavior.
type AskConfig struct {
	KnowledgeStore *knowledge.Store
	MaxContextLen  int
}

// AskResult contains the answer and context information.
type AskResult struct {
	Answer      string
	FromKB      bool
	Confidence  float64
	ContextUsed []string
}

// Ask runs a question against the codebase. KB hits return instantly.
// KB misses spawn exploration subagents.
func Ask(question string, cfg AskConfig) (*AskResult, error) {
	if cfg.KnowledgeStore == nil {
		return &AskResult{
			Answer: "Knowledge base not configured. Start a reasonix session first to build the KB.",
		}, nil
	}

	// Check KB first
	if answer := knowledge.InstantAnswer(cfg.KnowledgeStore, question, 0.7); answer != "" {
		return &AskResult{
			Answer:     answer,
			FromKB:     true,
			Confidence: 0.9,
		}, nil
	}

	// KB miss — would spawn explorer in live session
	return &AskResult{
		Answer:    fmt.Sprintf("No cached answer for: %s\nSpawn explorer subagent to investigate? (live session required)", question),
		FromKB:    false,
	}, nil
}

// AskCommand handles the /ask slash command.
func AskCommand(args []string, store *knowledge.Store) string {
	question := strings.Join(args, " ")
	if strings.TrimSpace(question) == "" {
		return "Usage: /ask <question about the codebase>"
	}

	result, _ := Ask(question, AskConfig{KnowledgeStore: store})

	var b strings.Builder
	if result.FromKB {
		b.WriteString("(instant KB answer)\n\n")
	} else {
		b.WriteString("(no cached answer — explorer would be needed)\n\n")
	}
	b.WriteString(result.Answer)
	return b.String()
}
