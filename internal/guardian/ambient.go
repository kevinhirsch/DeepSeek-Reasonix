package guardian

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"reasonix/internal/agent"
	"reasonix/internal/event"
	"reasonix/internal/provider"
	"reasonix/internal/tool"
)

// AmbientResult is the structured output of an ambient file-save check by a
// read-only Flash subagent. Safe changes are silent; warnings and issues are
// delivered to the user proactively (or queued while do-not-disturb is active).
type AmbientResult struct {
	Status   string `json:"status"`   // "safe" | "warning" | "issue_detected"
	File     string `json:"file"`     // path of the changed file
	Line     int    `json:"line"`     // approximate line of the issue (0 if N/A)
	Summary  string `json:"summary"`  // one-line description
	Detail   string `json:"detail"`   // longer explanation
	Severity string `json:"severity"` // "low" | "medium" | "high"
}

// FileWatcher watches a set of filesystem paths for modifications using periodic
// os.Stat polling. It is an fsnotify-like mechanism that works without external
// dependencies and is suitable for ambient-guardian file-save detection.
type FileWatcher struct {
	mu       sync.Mutex
	paths    map[string]watchEntry // tracked path -> last-seen stat info
	interval time.Duration
	onChange func(path string)
	stop     chan struct{}
}

type watchEntry struct {
	modTime time.Time
	size    int64
}

// NewFileWatcher creates a polling file watcher that checks tracked paths every
// interval. If interval <= 0 a sensible default is chosen.
func NewFileWatcher(interval time.Duration) *FileWatcher {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &FileWatcher{
		paths:    make(map[string]watchEntry),
		interval: interval,
		stop:     make(chan struct{}),
	}
}

// Watch adds path to the watch set. If the path is already tracked it is a no-op.
// Directories are ignored during polling — only regular files can trigger the
// on-change callback.
func (fw *FileWatcher) Watch(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if _, ok := fw.paths[path]; ok {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		fw.paths[path] = watchEntry{}
		return
	}
	fw.paths[path] = watchEntry{modTime: info.ModTime(), size: info.Size()}
}

// Unwatch removes path from the watch set.
func (fw *FileWatcher) Unwatch(path string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	delete(fw.paths, path)
}

// OnChange registers the callback that fires each time a tracked file is detected
// as modified. The callback is invoked in its own goroutine so it does not stall
// the poll loop.
func (fw *FileWatcher) OnChange(fn func(path string)) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.onChange = fn
}

// Start begins the background poll loop. The caller must cancel ctx to shut down.
func (fw *FileWatcher) Start(ctx context.Context) {
	ticker := time.NewTicker(fw.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-fw.stop:
			return
		case <-ticker.C:
			fw.poll()
		}
	}
}

// Stop signals the poll loop to exit. It is safe to call multiple times.
func (fw *FileWatcher) Stop() {
	select {
	case <-fw.stop:
	default:
		close(fw.stop)
	}
}

func (fw *FileWatcher) poll() {
	fw.mu.Lock()
	snapshot := make(map[string]watchEntry, len(fw.paths))
	for p, e := range fw.paths {
		snapshot[p] = e
	}
	cb := fw.onChange
	fw.mu.Unlock()

	for path, prev := range snapshot {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			continue
		}
		if !info.ModTime().Equal(prev.modTime) || info.Size() != prev.size {
			fw.mu.Lock()
			fw.paths[path] = watchEntry{modTime: info.ModTime(), size: info.Size()}
			fw.mu.Unlock()
			if cb != nil {
				go cb(path)
			}
		}
	}
}

// dndResult is a queued ambient result that was generated while do-not-disturb
// was active and should be delivered when DND is disabled.
type dndResult struct {
	result AmbientResult
	ts     time.Time
}

// AmbientGuardian watches file saves and spawns a read-only Flash subagent to
// check for bugs. It is silent on safe changes and proactive when issues are
// detected. When do-not-disturb mode is active, results queue and are delivered
// when DND is disabled.
//
// An AmbientGuardian creates short-lived, single-purpose agent sessions for each
// check rather than reusing a long-lived one — ambient reviews are infrequent and
// independent, so prefix-cache reuse is less important than isolation.
type AmbientGuardian struct {
	mu      sync.Mutex
	watcher *FileWatcher
	prov    provider.Provider
	reg     *tool.Registry
	sink    event.Sink
	pricing *provider.Pricing
	model   string // model id for ambient checks, e.g. "deepseek-flash"

	// Concurrency gate: at most maxAmbientChecks run simultaneously.
	activeChecks atomic.Int32

	// Checked file hashes: avoid re-checking files whose content has not changed
	// since the last check (even if mtime differs due to toolchain touches).
	checked map[string]string // path -> sha256 hex

	// Do-not-disturb.
	dnd      atomic.Bool
	dndQueue []dndResult

	// Callback for proactive issue notification (outside the event sink).
	onIssue func(result AmbientResult)

	stop chan struct{}
}

const maxAmbientChecks = 2

// NewAmbientGuardian creates an ambient guardian backed by the given provider and
// read-only tool registry. model is the model id used for ambient checks (e.g.
// "deepseek-flash"); pricing feeds per-check cost display.
func NewAmbientGuardian(prov provider.Provider, reg *tool.Registry, model string, pricing *provider.Pricing, sink event.Sink) *AmbientGuardian {
	if sink == nil {
		sink = event.Discard
	}
	ag := &AmbientGuardian{
		watcher: NewFileWatcher(3 * time.Second),
		prov:    prov,
		reg:     reg,
		sink:    sink,
		pricing: pricing,
		model:   model,
		checked: make(map[string]string),
		stop:    make(chan struct{}),
	}
	ag.watcher.OnChange(ag.onFileChange)
	return ag
}

// Start begins watching tracked files. The caller must cancel ctx to shut down
// the background poll loop.
func (ag *AmbientGuardian) Start(ctx context.Context) {
	go ag.watcher.Start(ctx)
}

// Stop halts the file watcher. Pending checks that have already begun their agent
// call will complete; new checks are discarded.
func (ag *AmbientGuardian) Stop() {
	select {
	case <-ag.stop:
	default:
		close(ag.stop)
	}
	ag.watcher.Stop()
}

// Watch adds path to the tracked set.
func (ag *AmbientGuardian) Watch(path string) {
	ag.watcher.Watch(path)
}

// Unwatch removes path from the tracked set.
func (ag *AmbientGuardian) Unwatch(path string) {
	ag.watcher.Unwatch(path)
}

// SetDoNotDisturb toggles do-not-disturb mode. When transitioning from on to off
// any queued results are flushed.
func (ag *AmbientGuardian) SetDoNotDisturb(on bool) {
	prev := ag.dnd.Swap(on)
	if prev && !on {
		ag.flushDNDQueue()
	}
}

// IsDoNotDisturb reports the current DND state.
func (ag *AmbientGuardian) IsDoNotDisturb() bool {
	return ag.dnd.Load()
}

// OnIssue registers a callback for proactive issue notification. It fires for
// every non-safe result, including those delivered after a DND period.
func (ag *AmbientGuardian) OnIssue(fn func(result AmbientResult)) {
	ag.mu.Lock()
	defer ag.mu.Unlock()
	ag.onIssue = fn
}

// onFileChange is the FileWatcher callback. It is invoked in its own goroutine by
// the watcher, so it must not block.
func (ag *AmbientGuardian) onFileChange(path string) {
	select {
	case <-ag.stop:
		return
	default:
	}

	info, err := os.Stat(path)
	if err != nil {
		return
	}
	if info.IsDir() {
		return
	}

	ext := strings.ToLower(filepath.Ext(path))
	if !isCodeFile(ext) {
		return
	}

	if ag.activeChecks.Load() >= maxAmbientChecks {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	h := sha256.Sum256(data)
	hash := fmt.Sprintf("%x", h)

	ag.mu.Lock()
	prev, exists := ag.checked[path]
	if exists && prev == hash {
		ag.mu.Unlock()
		return
	}
	ag.checked[path] = hash
	ag.mu.Unlock()

	ag.activeChecks.Add(1)
	go func() {
		defer ag.activeChecks.Add(-1)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		result := ag.check(ctx, path)
		if result.Status != "safe" {
			ag.deliverResult(result)
		}
	}()
}

// check runs a single read-only agent call to inspect the file at path.
func (ag *AmbientGuardian) check(ctx context.Context, path string) AmbientResult {
	content, err := readFileCapped(path, 200_000)
	if err != nil {
		return AmbientResult{Status: "safe", File: path}
	}

	prompt := buildCheckPrompt(path, filepath.Ext(path), content)
	sess := agent.NewSession(prompt)
	agt := agent.New(ag.prov, ag.reg, sess, agent.Options{
		MaxSteps:      3,
		Temperature:   0,
		ContextWindow: 50_000,
	}, event.Discard)

	if err := agt.Run(ctx, ""); err != nil {
		return AmbientResult{Status: "safe", File: path}
	}

	msgs := sess.Snapshot()
	var lastText string
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == provider.RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			lastText = msgs[i].Content
			break
		}
	}

	return parseAmbientResult(lastText, path)
}

// deliverResult either queues the result (DND active) or emits it immediately.
func (ag *AmbientGuardian) deliverResult(result AmbientResult) {
	if ag.dnd.Load() {
		ag.mu.Lock()
		ag.dndQueue = append(ag.dndQueue, dndResult{result: result, ts: time.Now()})
		ag.mu.Unlock()
		return
	}

	ag.sink.Emit(event.Event{
		Kind:  event.Notice,
		Text:  fmt.Sprintf("ambient guardian: %s in %s — %s", result.Status, result.File, result.Summary),
		Level: event.LevelWarn,
	})

	ag.mu.Lock()
	cb := ag.onIssue
	ag.mu.Unlock()
	if cb != nil {
		cb(result)
	}
}

// flushDNDQueue delivers all queued DND results.
func (ag *AmbientGuardian) flushDNDQueue() {
	ag.mu.Lock()
	queue := ag.dndQueue
	ag.dndQueue = nil
	cb := ag.onIssue
	ag.mu.Unlock()

	for _, d := range queue {
		age := time.Since(d.ts).Round(time.Second)
		ag.sink.Emit(event.Event{
			Kind:  event.Notice,
			Text:  fmt.Sprintf("ambient guardian (queued %s ago): %s in %s — %s", age, d.result.Status, d.result.File, d.result.Summary),
			Level: event.LevelWarn,
		})
		if cb != nil {
			cb(d.result)
		}
	}
}

// --- helpers ---------------------------------------------------------------

// isCodeFile reports whether ext is a source-code extension the ambient guardian
// should inspect. Configuration files and docs are excluded to reduce noise.
func isCodeFile(ext string) bool {
	switch ext {
	case ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rs", ".c", ".cpp", ".cc",
		".h", ".hpp", ".java", ".kt", ".kts", ".swift", ".rb", ".php",
		".sh", ".bash", ".yaml", ".yml", ".toml", ".sql", ".proto",
		".vue", ".svelte", ".cs", ".fs", ".fsx", ".scala", ".clj",
		".ex", ".exs", ".ml", ".mli", ".erl", ".hrl", ".dart",
		".lua", ".zig", ".nim", ".r", ".jl":
		return true
	}
	return false
}

// readFileCapped reads up to maxBytes of the file at path. If the file is larger,
// the returned content is truncated with a trailing annotation.
func readFileCapped(path string, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() <= maxBytes {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	buf := make([]byte, maxBytes)
	n, _ := f.Read(buf)
	return string(buf[:n]) + "\n\n[file truncated — exceeds review limit]", nil
}

// buildCheckPrompt constructs the system prompt for a single ambient file review.
func buildCheckPrompt(path, ext, content string) string {
	return fmt.Sprintf(`You are a code-safety reviewer. Examine the following file for potential bugs, security issues, or correctness problems. Be concise and specific — flag only real, actionable issues.

File: %s
Language: %s

%s

Respond with EXACTLY one JSON object and nothing else:
{"status": "safe"|"warning"|"issue_detected", "line": <int>, "summary": "<one line>", "detail": "<specific explanation>", "severity": "low"|"medium"|"high"}

Use "safe" when the code looks correct. Use "warning" for patterns that may cause problems. Use "issue_detected" for concrete bugs. Do not hallucinate — only flag problems you can see.`, path, ext, content)
}

// parseAmbientResult extracts a structured result from the model's raw output.
// Unparseable output yields a safe result (fail-silent).
func parseAmbientResult(text, path string) AmbientResult {
	var result AmbientResult
	text = strings.TrimSpace(text)
	if text == "" {
		return AmbientResult{Status: "safe", File: path}
	}
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		result.File = path
		return validateAmbientResult(result)
	}
	if start := strings.IndexByte(text, '{'); start >= 0 {
		if end := strings.LastIndexByte(text, '}'); end > start {
			slice := text[start : end+1]
			if err := json.Unmarshal([]byte(slice), &result); err == nil {
				result.File = path
				return validateAmbientResult(result)
			}
		}
	}
	return AmbientResult{Status: "safe", File: path}
}

func validateAmbientResult(r AmbientResult) AmbientResult {
	switch r.Status {
	case "safe", "warning", "issue_detected":
	default:
		r.Status = "safe"
	}
	switch r.Severity {
	case "low", "medium", "high":
	default:
		if r.Status == "issue_detected" {
			r.Severity = "medium"
		} else {
			r.Severity = "low"
		}
	}
	if r.Status == "safe" {
		r.Summary = ""
		r.Detail = ""
	}
	return r
}
