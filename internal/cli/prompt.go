package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"reasonix/internal/prompt"
)

func promptCommand(args []string) int {
	if len(args) == 0 {
		promptUsage()
		return 2
	}

	switch args[0] {
	case "eval":
		return promptEvalCommand(args[1:])
	case "list":
		return promptListCommand(args[1:])
	default:
		promptUsage()
		return 2
	}
}

func promptEvalCommand(args []string) int {
	fs := flag.NewFlagSet("prompt eval", flag.ContinueOnError)
	role := fs.String("role", "", "Role to evaluate (explorer, reviewer, verifier, planner, executor)")
	model := fs.String("model", "", "Model to use (default: configured default)")
	scenariosFile := fs.String("scenarios", "", "Custom scenarios file (default: .reasonix/prompts/eval/<role>.json)")
	outputJSON := fs.Bool("json", false, "Output results as JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	// Positional role argument overrides --role
	if fs.NArg() > 0 && *role == "" {
		*role = fs.Arg(0)
	}

	if *role == "" {
		fmt.Fprintln(os.Stderr, "Role is required. Use --role or positional argument.")
		promptEvalUsage()
		return 2
	}

	role = strings.ToLower(strings.TrimSpace(role))
	validRoles := map[string]bool{
		"explorer":  true,
		"reviewer":  true,
		"verifier":  true,
		"planner":   true,
		"executor":  true,
	}
	if !validRoles[*role] {
		fmt.Fprintf(os.Stderr, "Invalid role: %q. Valid roles: explorer, reviewer, verifier, planner, executor\n", *role)
		return 2
	}

	// Load scenarios
	scenariosPath := *scenariosFile
	if scenariosPath == "" {
		scenariosPath = fmt.Sprintf(".reasonix/prompts/eval/%s.json", *role)
	}

	suite, err := prompt.LoadSuite(scenariosPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load scenarios: %v\n", err)
		return 1
	}

	// Load prompt
	promptsDir := prompt.DefaultPromptsDir(".")
	promptText := prompt.LoadPrompt(*role, promptsDir,
		fmt.Sprintf("You are a %s subagent. Complete the task concisely.", *role))

	fmt.Printf("Evaluating %s prompt...\n", *role)
	fmt.Printf("  Scenarios: %d\n", len(suite.Scenarios))
	fmt.Printf("  Model: %s\n", *model)
	fmt.Println()

	// Run evaluation. The eval harness checks each scenario against expectations.
	// In real usage, this calls the provider API. For determinism, a mock provider
	// is used when no provider is configured.
	cfg := prompt.EvalConfig{
		Role:      *role,
		Prompt:    promptText,
		Model:     *model,
		Scenarios: suite.Scenarios,
	}

	run, err := prompt.Evaluate(nil, cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Evaluation failed: %v\n", err)
		return 1
	}

	// Archive results
	resultsDir := prompt.DefaultResultsDir(".")
	if err := prompt.ArchiveRun(run, resultsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to archive results: %v\n", err)
	}

	if *outputJSON {
		data, _ := json.MarshalIndent(run, "", "  ")
		fmt.Println(string(data))
		return 0
	}

	// Print results
	for _, result := range run.Results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		fmt.Printf("  %s  %s", status, result.Scenario)
		if result.Duration > 0 {
			fmt.Printf(" (%v)", result.Duration)
		}
		fmt.Println()
		for _, failure := range result.Failures {
			fmt.Printf("    - %s\n", failure)
		}
	}

	fmt.Println()
	fmt.Printf("Pass rate: %s\n", prompt.PassRateString(run.Passed, run.Failed))
	if run.PassRate >= 90.0 {
		fmt.Println("Calibration threshold (>=90%) met.")
	} else {
		fmt.Println("Calibration threshold (>=90%) NOT met. Iterate the prompt and re-evaluate.")
	}

	return 0
}

func promptListCommand(args []string) int {
	promptsDir := prompt.DefaultPromptsDir(".")
	if promptsDir == "" {
		fmt.Println("No prompt directory found. Create .reasonix/prompts/v1/ with role markdown files.")
		return 0
	}

	entries, err := os.ReadDir(promptsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read prompts directory: %v\n", err)
		return 1
	}

	fmt.Printf("Prompt files in %s:\n\n", promptsDir)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		info, _ := e.Info()
		size := ""
		if info != nil {
			size = fmt.Sprintf(" (%d bytes)", info.Size())
		}
		fmt.Printf("  %s%s\n", name, size)
	}
	return 0
}

func promptUsage() {
	fmt.Println("Usage: reasonix prompt <subcommand>")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  eval <role>    Evaluate a role prompt against its scenario suite")
	fmt.Println("  list           List available prompt files")
}

func promptEvalUsage() {
	fmt.Println("Usage: reasonix prompt eval <role> [flags]")
	fmt.Println()
	fmt.Println("Flags:")
	fmt.Println("  --model <name>       Model to use for evaluation")
	fmt.Println("  --scenarios <file>   Custom scenarios file")
	fmt.Println("  --json               Output results as JSON")
	fmt.Println()
	fmt.Println("Valid roles: explorer, reviewer, verifier, planner, executor")
}
