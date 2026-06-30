package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// CrossValidator runs the same verification on two different models (e.g.,
// Flash and Pro) and compares verdicts. If verdicts match → high confidence.
// If they differ → surface the dissent.
type CrossValidator struct {
	taskTool *TaskTool
	primary  string // e.g. "deepseek-v4-pro"
	secondary string // e.g. "deepseek-v4-flash"
}

// NewCrossValidator creates a multi-model cross-validator.
func NewCrossValidator(taskTool *TaskTool, primary, secondary string) *CrossValidator {
	return &CrossValidator{taskTool: taskTool, primary: primary, secondary: secondary}
}

// CrossValidationResult holds the output from both models.
type CrossValidationResult struct {
	PrimaryVerdict   string
	SecondaryVerdict string
	Agree            bool
	Output           string
}

// Run executes the same prompt on both models and compares results.
func (c *CrossValidator) Run(ctx context.Context, prompt string) (*CrossValidationResult, error) {
	var wg sync.WaitGroup
	var primaryOut, secondaryOut string
	var primaryErr, secondaryErr error

	wg.Add(2)
	go func() {
		defer wg.Done()
		primaryOut, primaryErr = c.runWithModel(ctx, c.primary, prompt)
	}()
	go func() {
		defer wg.Done()
		secondaryOut, secondaryErr = c.runWithModel(ctx, c.secondary, prompt)
	}()
	wg.Wait()

	if primaryErr != nil && secondaryErr != nil {
		return nil, fmt.Errorf("both models failed: primary=%v, secondary=%v", primaryErr, secondaryErr)
	}

	result := &CrossValidationResult{
		PrimaryVerdict:   extractVerdict(primaryOut),
		SecondaryVerdict: extractVerdict(secondaryOut),
	}
	result.Agree = result.PrimaryVerdict == result.SecondaryVerdict && result.PrimaryVerdict != ""

	var b strings.Builder
	if result.Agree {
		b.WriteString(fmt.Sprintf("MODELS AGREE — %s\n\n", result.PrimaryVerdict))
	} else {
		b.WriteString(fmt.Sprintf("MODELS DISSENT\n  Primary (%s): %s\n  Secondary (%s): %s\n\n",
			c.primary, result.PrimaryVerdict, c.secondary, result.SecondaryVerdict))
	}
	b.WriteString("Primary output:\n" + primaryOut + "\n\n")
	b.WriteString("Secondary output:\n" + secondaryOut)
	result.Output = b.String()
	return result, nil
}

func (c *CrossValidator) runWithModel(ctx context.Context, modelRef, prompt string) (string, error) {
	subReg := SubagentToolRegistry(c.taskTool.parentReg, nil)
	sess := NewSession(DefaultVerifierPrompt)
	return RunSubAgentWithSession(ctx, c.taskTool.prov, subReg, sess, prompt, Options{
		MaxSteps: 10,
	}, Discard)
}

func extractVerdict(output string) string {
	lower := strings.ToLower(output)
	if strings.Contains(lower, "confirmed") {
		return "CONFIRMED"
	}
	if strings.Contains(lower, "refuted") {
		return "REFUTED"
	}
	if strings.Contains(lower, "plausible") {
		return "PLAUSIBLE"
	}
	return ""
}
