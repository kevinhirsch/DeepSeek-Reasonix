package prompt

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckExpectation_ToolUsed(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindToolUsed, Value: "grep"},
		[]string{"grep", "read_file"},
		"",
	)
	if len(failures) > 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCheckExpectation_ToolUsedMissing(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindToolUsed, Value: "grep"},
		[]string{"read_file", "bash"},
		"",
	)
	if len(failures) == 0 {
		t.Fatal("expected failure for missing tool")
	}
}

func TestCheckExpectation_ToolNotUsed(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindToolNotUsed, Value: "task"},
		[]string{"grep", "read_file"},
		"",
	)
	if len(failures) > 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCheckExpectation_ToolNotUsedViolation(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindToolNotUsed, Value: "task"},
		[]string{"grep", "task"},
		"",
	)
	if len(failures) == 0 {
		t.Fatal("expected failure for disallowed tool")
	}
}

func TestCheckExpectation_OutputContains(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindOutputContains, Value: "file:line"},
		[]string{},
		"Found something at file:line reference.",
	)
	if len(failures) > 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCheckExpectation_OutputNotContains(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindOutputNotContain, Value: "I recommend"},
		[]string{},
		"I recommend refactoring this module.",
	)
	if len(failures) == 0 {
		t.Fatal("expected failure for forbidden content")
	}
}

func TestCheckExpectation_StopsWithinNCalls(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindStopsWithinNCalls, Count: 10},
		[]string{"grep", "read_file", "grep", "read_file", "bash"},
		"",
	)
	if len(failures) > 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCheckExpectation_StopsWithinNCallsExceeded(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindStopsWithinNCalls, Count: 3},
		[]string{"grep", "read_file", "grep", "read_file", "bash"},
		"",
	)
	if len(failures) == 0 {
		t.Fatal("expected failure for exceeding call limit")
	}
}

func TestCheckExpectation_NoNarration(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindNoNarration},
		[]string{"grep", "read_file"},
		"Results: found 3 instances",
	)
	if len(failures) > 0 {
		t.Fatalf("expected no failures, got %v", failures)
	}
}

func TestCheckExpectation_NoNarrationViolation(t *testing.T) {
	failures := CheckExpectation(
		Expectation{Kind: KindNoNarration},
		[]string{},
		"Now I'll start by reading the file.",
	)
	if len(failures) == 0 {
		t.Fatal("expected failure for narration")
	}
}

func TestLoadSuite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	data := `{"scenarios": [{"name": "s1", "input": "test", "expectations": [{"kind": "tool_used", "value": "grep"}]}]}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	suite, err := LoadSuite(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(suite.Scenarios) != 1 {
		t.Fatalf("expected 1 scenario, got %d", len(suite.Scenarios))
	}
	if suite.Scenarios[0].Name != "s1" {
		t.Fatalf("expected name 's1', got %q", suite.Scenarios[0].Name)
	}
}

func TestLoadSuite_MissingFile(t *testing.T) {
	_, err := LoadSuite("/nonexistent/path.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadPrompt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "explorer.md")
	if err := os.WriteFile(path, []byte("Explorer prompt content"), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadPrompt("explorer", dir, "default prompt")
	if got != "Explorer prompt content" {
		t.Fatalf("got %q, want 'Explorer prompt content'", got)
	}
}

func TestLoadPrompt_FallbackToDefault(t *testing.T) {
	got := LoadPrompt("explorer", "/nonexistent", "default prompt")
	if got != "default prompt" {
		t.Fatalf("got %q, want 'default prompt'", got)
	}
}

func TestLoadPrompt_EmptyFileFallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "explorer.md")
	if err := os.WriteFile(path, []byte("   \n  \n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadPrompt("explorer", dir, "default prompt")
	if got != "default prompt" {
		t.Fatalf("got %q, want 'default prompt'", got)
	}
}

func TestLoadDeepSeekNotes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "deepseek_notes.md")
	if err := os.WriteFile(path, []byte("DeepSeek notes content"), 0644); err != nil {
		t.Fatal(err)
	}

	got := LoadDeepSeekNotes(dir)
	if got != "DeepSeek notes content" {
		t.Fatalf("got %q, want 'DeepSeek notes content'", got)
	}
}

func TestLoadDeepSeekNotes_Missing(t *testing.T) {
	got := LoadDeepSeekNotes("/nonexistent")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestEvaluate(t *testing.T) {
	ctx := context.Background()
	cfg := EvalConfig{
		Role:   "explorer",
		Model:  "mock",
		Scenarios: []EvalScenario{
			{
				Name: "test-scenario",
				Input: "test input",
				Expectations: []Expectation{
					{Kind: KindToolUsed, Value: "grep"},
				},
			},
		},
	}

	run, err := Evaluate(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if run.Role != "explorer" {
		t.Fatalf("role = %q, want 'explorer'", run.Role)
	}
}

func TestPassRateString(t *testing.T) {
	s := PassRateString(8, 2)
	expected := "8/10 (80%)"
	if s != expected {
		t.Fatalf("got %q, want %q", s, expected)
	}
}

func TestPassRateString_Zero(t *testing.T) {
	s := PassRateString(0, 0)
	if s != "0/0 (0%)" {
		t.Fatalf("got %q, want '0/0 (0%%)'", s)
	}
}

func TestArchiveRun(t *testing.T) {
	dir := t.TempDir()
	run := &EvalRun{
		Role:     "explorer",
		Model:    "mock",
		Passed:   3,
		Failed:   1,
		PassRate: 75.0,
		Results: []EvalResult{
			{Scenario: "s1", Passed: true},
		},
	}

	if err := ArchiveRun(run, dir); err != nil {
		t.Fatal(err)
	}

	// Verify a file was created
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
	if !filepath.HasPrefix(entries[0].Name(), "explorer-") {
		t.Fatalf("expected filename prefix 'explorer-', got %q", entries[0].Name())
	}
}
