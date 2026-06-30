# Mock Provider Design

> Needed before any implementation. The existing `internal/agent/testutil/mock_provider.go` must be extended to simulate DeepSeek and Anthropic behavior deterministically.

## Architecture

```
internal/provider/testutil/
├── mock.go              — MockProvider: configurable behavior via MockMode
├── mock_deepseek.go     — DeepSeek-specific: reasoning_content, cognitive loops, 429s
├── mock_anthropic.go    — Anthropic-specific: thinking blocks, cache hits, refusals
└── mock_replay.go       — Record/Replay: capture real responses, replay deterministically
```

## MockMode Enum

```go
type MockMode int

const (
    ModeNormal       MockMode = iota // standard successful responses
    ModeEmpty                        // returns empty content (simulates model with nothing to say)
    ModeLooping                      // returns identical reasoning each turn (cognitive loop)
    ModeRateLimited                  // returns 429 on first attempt, succeeds on retry
    ModeOutage                       // returns 500 for N consecutive requests
    ModeSlow                         // adds configurable latency to each chunk
    ModeThinking                     // returns thinking blocks before text (Anthropic)
    ModeCacheHit                     // second identical request shows cache read tokens
    ModeRefusal                      // returns stop_reason: "refusal"
    ModeToolUse                      // returns a tool call matching the prompt
)
```

## MockProvider Interface Extension

```go
type MockProvider struct {
    provider.Provider
    mode       MockMode
    modeCount  int            // tracks turns for multi-turn behavior
    responses  []MockResponse // pre-configured response sequence
    recordMode bool           // record real responses for replay
    recordings []RawResponse  // captured real API responses
}

type MockResponse struct {
    Chunks []provider.Chunk
    Error  error
    Usage  *provider.Usage
}

func (m *MockProvider) SetMode(mode MockMode)
func (m *MockProvider) SetResponseSequence(responses []MockResponse)
func (m *MockProvider) EnableRecording(dir string)
func (m *MockProvider) LoadReplay(dir string) error
```

## Usage in Tests

```go
func TestCognitiveLoopDetector(t *testing.T) {
    mp := testutil.NewMock("deepseek-v4-flash")
    mp.SetMode(testutil.ModeLooping)
    
    agent := agent.New(mp, echoRegistry(), agent.NewSession(""), agent.Options{}, event.Discard)
    err := agent.Run(context.Background(), "find the bug")
    
    // Should be interrupted by loop detector, not hang forever
    if err == nil {
        t.Fatal("expected loop detector to interrupt")
    }
    if !strings.Contains(err.Error(), "cognitive loop") {
        t.Fatalf("expected cognitive loop error, got: %v", err)
    }
}
```

## Record/Replay

```go
func TestWorkflowReplay(t *testing.T) {
    // Load recorded responses from a real workflow run
    mp := testutil.NewMock("deepseek-v4-pro")
    mp.LoadReplay("testdata/workflows/security-audit")
    
    // Run workflow deterministically
    result := runWorkflow(t, mp, "audit auth.go")
    
    // Compare to golden file
    golden := loadGolden("testdata/workflows/security-audit.golden")
    if diff := cmp.Diff(golden, result); diff != "" {
        t.Fatalf("workflow output diverged from golden:\n%s", diff)
    }
}
```

## Recording Format

```
testdata/workflows/<name>/
├── manifest.json          — workflow spec + metadata
├── turn-001.json          — raw API request + response
├── turn-002.json
└── ...
```

Each turn file:
```json
{
  "request": { "model": "deepseek-v4-pro", "messages": [...], "tools": [...] },
  "response": { "chunks": [...], "usage": {...} },
  "duration_ms": 1234
}
```

## Test Tiers

| Tier | Command | What Runs | When |
|---|---|---|---|
| Short | `go test -short ./...` | Unit tests, mock provider tests | Every commit |
| Full | `go test ./...` | All mock tests, integration tests | Every PR |
| Live | `go test -tags=live ./...` | Real API calls | Nightly |
| Stress | `go test -tags=stress ./...` | 1000 subagent spawns | Weekly |
