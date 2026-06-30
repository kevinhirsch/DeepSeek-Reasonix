// Package cli implements the CLI command handling for reasonix.
//
// Ask.go implements the /ask conversational codebase Q&A panel
// (Issue #21 Comp 20). The /ask command opens a focused Q&A experience
// for natural-language questions about the codebase. It uses the
// knowledge base for instant answers to repeat questions and spawns
// explorer subagents when novel questions need investigation.
package cli

import (
	"context"
	"flag"
	"fmt"
	"path/filepath"
	"strings"

	"reasonix/internal/agent"
	"reasonix/internal/boot"
	"reasonix/internal/config"
	"reasonix/internal/event"
	"reasonix/internal/knowledge"
	"reasonix/internal/tool"
)

// askCommand handles the standalone "reasonix ask" CLI command.
func askCommand(args []string) int {
	fs := flag.NewFlagSet("ask", flag.ContinueOnError)
	question := fs.String("question", "", "question about the codebase")
	model := fs.String("model", "", "provider name override")
	maxResults := fs.Int("max", 3, "maximum KB results to show")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	q := strings.TrimSpace(*question)
	if q == "" && fs.NArg() > 0 {
		q = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}
	if q == "" {
		fmt.Println("Usage: reasonix ask --question \"<natural language question about the codebase>\"")
		fmt.Println("   or: reasonix ask <question>")
		return 1
	}

	fmt.Printf("Q: %s\n\n", q)

	// 1. Try KB lookup first.
	kbStore := openKnowledgeBase()
	if kbStore != nil {
		if answer := knowledge.InstantAnswer(kbStore, q, 0.7); answer != "" {
			fmt.Printf("(instant KB answer)\n%s\n", answer)
			return 0
		}

		results := knowledge.SemanticQuery(kbStore, q, *maxResults)
		if len(results) > 0 {
			fmt.Println("Related KB entries:")
			for i, r := range results {
				fmt.Printf("  %d. [%.0f%% match] %s\n", i+1, r.Score*100, r.Entry.Question)
			}
			fmt.Println()
		}
	}

	// 2. KB miss — spawn a live explorer if a provider is configured.
	cfg, err := config.Load()
	if err != nil {
		fmt.Println("(no provider configured for live exploration)")
		fmt.Println("Tip: configure a provider to enable live codebase exploration.")
		return 0
	}

	if *model == "" {
		*model = cfg.DefaultModel
	}
	entry, ok := cfg.ResolveModel(*model)
	if !ok {
		fmt.Printf("Unknown model %q — check your config.\n", *model)
		return 1
	}
	if err := cfg.Validate(*model); err != nil {
		fmt.Println("Config error:", err)
		return 1
	}

	prov, err := boot.NewProviderWithProxy(entry, cfg.NetworkProxySpec())
	if err != nil {
		fmt.Println("Failed to create provider:", err)
		return 1
	}

	// Build a lightweight subagent registry for exploration.
	reg := tool.NewRegistry()
	for _, name := range []string{"read", "glob", "grep", "bash", "ls", "git"} {
		if tl, ok := tool.LookupBuiltin(name); ok {
			reg.Add(tl)
		}
	}
	subReg := agent.SubagentToolRegistry(reg, nil)

	explorerPrompt := fmt.Sprintf(
		"You are a codebase explorer. Answer the following question about the codebase using only the tools available (read files, search code, run safe commands). Be concise and factual. If you cannot find the answer, say so clearly.\n\nQuestion: %s",
		q,
	)

	result, err := agent.RunSubAgentWithSession(
		context.Background(),
		prov,
		subReg,
		agent.NewSession(""),
		explorerPrompt,
		agent.Options{
			MaxSteps:    10,
			Temperature: cfg.Agent.Temperature,
			Pricing:     entry.Price,
		},
		event.Discard,
	)
	if err != nil {
		fmt.Println("Exploration error:", err)
		return 1
	}

	fmt.Print(result)

	// Store the result in the KB for future instant answers.
	if kbStore != nil {
		kbStore.Add(q, result, nil, entry.Kind, 0.8)
	}

	return 0
}

// runAskCommand handles the /ask slash command from within a chat TUI session.
// It checks the knowledge base for instant answers and reports results directly.
func (m *chatTUI) runAskCommand(input string) {
	args := tokenizeArgs(input)
	if len(args) < 2 {
		m.notice("Usage: /ask <question about the codebase>")
		return
	}

	question := strings.TrimSpace(strings.Join(args[1:], " "))
	if question == "" {
		m.notice("Usage: /ask <question about the codebase>")
		return
	}

	// Check the knowledge base for an instant answer.
	kbStore := resolveKnowledgeBase(m)
	if kbStore != nil {
		if answer := knowledge.InstantAnswer(kbStore, question, 0.7); answer != "" {
			m.notice(fmt.Sprintf("(KB) %s", answer))
			return
		}

		results := knowledge.SemanticQuery(kbStore, question, 3)
		if len(results) > 0 {
			var b strings.Builder
			b.WriteString("Related KB entries:\n")
			for i, r := range results {
				fmt.Fprintf(&b, "  %d. [%.0f%%] %s\n", i+1, r.Score*100, r.Entry.Question)
			}
			m.notice(strings.TrimRight(b.String(), "\n"))
			return
		}
	}

	m.notice("No cached answer found. Use the main chat to ask an explorer subagent to investigate this question about the codebase.")
}

// resolveKnowledgeBase returns the active knowledge base store, or nil.
func resolveKnowledgeBase(m *chatTUI) *knowledge.Store {
	if m.ctrl == nil {
		return nil
	}
	sessionDir := m.ctrl.SessionDir()
	if sessionDir == "" {
		return nil
	}
	kbDir := filepath.Join(sessionDir, "..", "knowledge")
	return knowledge.NewStore(kbDir)
}

// openKnowledgeBase opens the knowledge base for the current working directory.
func openKnowledgeBase() *knowledge.Store {
	return knowledge.NewStore(".reasonix/knowledge")
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
