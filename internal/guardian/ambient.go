package guardian

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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

// AmbientSeverity classifies the outcome of an ambient file review.
type AmbientSeverity int

const (
	SeveritySafe AmbientSeverity = iota
	SeverityWarning
	SeverityIssue
)

func (s AmbientSeverity) String() string {
	switch s {
	case SeveritySafe:
		return "safe"
	case SeverityWarning:
		return "warning"
	case SeverityIssue:
		return "issue"
	default:
		return "unknown"
	}
}

// AmbientResult is the structured outcome of an ambient file-save review.
type AmbientResult struct {
	File       string    `json:"file"`
	ModTime    time.Time `json:"mod_time"`
	Severity   AmbientSeverity `json:"severity"`
	Summary    string    `json:"summary"`
	Detail     string    `json:"detail"`
	Suggestion string    `json:"suggestion,omitempty"`
	Hash       string    `json:"hash"`
}

// DNDMode controls whether the ambient guardian is active.
type DNDMode int32

const (
	DNDOff DNDMode = 0
	DNDOn  DNDMode = 1
)

func (m *DNDMode) IsActive() bool {
	return atomic.LoadInt32((*int32)(m)) == int32(DNDOff)
}

func (m *DNDMode) Set(on bool) bool {
	var val int32
	if on {
		val = int32(DNDOn)
	}
	prev := atomic.SwapInt32((*int32)(m), val)
	return prev == int32(DNDOff)
}

// FileWatcher polls directories for file modifications using os.Stat.
type FileWatcher struct {
	dirs    []string
	exts    []string
	exclude []string
	state   map[string]fileState
	mu      sync.Mutex
}

type fileState struct {
	ModTime time.Time
	Size    int64
	Hash    string
}

func NewFileWatcher(dirs, exts, exclude []string) *FileWatcher {
	return &FileWatcher{
		dirs:    dirs,
		exts:    exts,
		exclude: exclude,
		state:   make(map[string]fileState),
	}
}

func (w *FileWatcher) Scan() (changed []string, removed []string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	seen := make(map[string]bool)
	for _, dir := range w.dirs {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			for _, prefix := range w.exclude {
				rel, _ := filepath.Rel(dir, path)
				if strings.HasPrefix(rel, prefix) || strings.HasPrefix(path, prefix) {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if d.IsDir() {
				return nil
			}
			if !w.hasTrackedExt(path) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil
			}
			seen[path] = true
			prev, known := w.state[path]
			if !known || info.ModTime().After(prev.ModTime) || info.Size() != prev.Size {
				changed = append(changed, path)
				w.state[path] = fileState{
					ModTime: info.ModTime(),
					Size:    info.Size(),
					Hash:    prev.Hash,
				}
			}
			return nil
		})
	}
	for path := range w.state {
		if !seen[path] {
			removed = append(removed, path)
			delete(w.state, path)
		}
	}
	return changed, removed
}

func (w *FileWatcher) MarkReviewed(path, hash string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if s, ok := w.state[path]; ok {
		s.Hash = hash
		w.state[path] = s
	}
}

func (w *FileWatcher) DirsToWatch() []string {
	return append([]string(nil), w.dirs...)
}

func (w *FileWatcher) hasTrackedExt(path string) bool {
	if len(w.exts) == 0 {
		return true
	}
	ext := filepath.Ext(path)
	for _, e := range w.exts {
		if strings.EqualFold(ext, e) {
			return true
		}
	}
	return false
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}

// AmbientGuardian watches file saves and spawns read-only Flash sub-agents.
type AmbientGuardian struct {
	watcher        *FileWatcher
	prov           provider.Provider
	roReg          *tool.Registry
	sink           event.Sink
	dnd            DNDMode
	pollInterval   time.Duration
	maxConcurrent  int
	sem            chan struct{}
	reviewed       map[string]string
	mu             sync.Mutex
	flashModel     string
	flashMaxTokens int
	flashTemp      float64
	cancelBackground context.CancelFunc
}

// AmbientConfig configures the ambient guardian.
type AmbientConfig struct {
	Dirs             []string
	Exts             []string
	Exclude          []string
	PollInterval     time.Duration
	MaxConcurrent    int
	FlashModel       string
	FlashMaxTokens   int
	FlashTemperature float64
	Sink             event.Sink
}

func (c *AmbientConfig) defaults() {
	if c.PollInterval <= 0 {
		c.PollInterval = 5 * time.Second
	}
	if c.MaxConcurrent <= 0 {
		c.MaxConcurrent = 2
	}
	if c.FlashModel == "" {
		c.FlashModel = "deepseek-chat"
	}
	if c.FlashMaxTokens <= 0 {
		c.FlashMaxTokens = 2000
	}
	if len(c.Exts) == 0 {
		c.Exts = []string{".go", ".py", ".ts", ".tsx", ".js", ".rs", ".java", ".rb"}
	}
	if len(c.Exclude) == 0 {
		c.Exclude = []string{".git", "vendor", "node_modules", "__pycache__", ".venv", "dist", "build"}
	}
}

func NewAmbientGuardian(prov provider.Provider, roReg *tool.Registry, cfg AmbientConfig) *AmbientGuardian {
	cfg.defaults()
	return &AmbientGuardian{
		watcher:        NewFileWatcher(cfg.Dirs, cfg.Exts, cfg.Exclude),
		prov:           prov,
		roReg:          roReg,
		sink:           cfg.Sink,
		pollInterval:   cfg.PollInterval,
		maxConcurrent:  cfg.MaxConcurrent,
		sem:            make(chan struct{}, cfg.MaxConcurrent),
		reviewed:       make(map[string]string),
		flashModel:     cfg.FlashModel,
		flashMaxTokens: cfg.FlashMaxTokens,
		flashTemp:      cfg.FlashTemperature,
	}
}

func (g *AmbientGuardian) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancelBackground = cancel
	go g.poll(ctx)
}

func (g *AmbientGuardian) Close() {
	if g.cancelBackground != nil {
		g.cancelBackground()
	}
	for i := 0; i < g.maxConcurrent; i++ {
		g.sem <- struct{}{}
	}
}

func (g *AmbientGuardian) DND() *DNDMode { return &g.dnd }

func (g *AmbientGuardian) ReviewNow(ctx context.Context, path string) (AmbientResult, error) {
	if !g.dnd.IsActive() {
		return AmbientResult{
			File:     path,
			Severity: SeveritySafe,
			Summary:  "ambient guardian is in do-not-disturb mode",
		}, nil
	}
	return g.reviewFile(ctx, path)
}

func (g *AmbientGuardian) poll(ctx context.Context) {
	ticker := time.NewTicker(g.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !g.dnd.IsActive() {
				continue
			}
			changed, _ := g.watcher.Scan()
			for _, path := range changed {
				if h, err := hashFile(path); err == nil {
					g.mu.Lock()
					if prev, ok := g.reviewed[path]; ok && prev == h {
						g.mu.Unlock()
						continue
					}
					g.reviewed[path] = h
					g.mu.Unlock()
				}
				select {
				case <-ctx.Done():
					return
				case g.sem <- struct{}{}:
				}
				go func(filePath string) {
					defer func() { <-g.sem }()
					g.reviewAndEmit(context.Background(), filePath)
				}(path)
			}
		}
	}
}

func (g *AmbientGuardian) reviewAndEmit(ctx context.Context, path string) {
	result, err := g.reviewFile(ctx, path)
	if err != nil || result.Severity == SeveritySafe {
		return
	}
	id := fmt.Sprintf("ambient-%d", time.Now().UnixNano())
	sink := g.sink
	if sink == nil {
		return
	}
	sink.Emit(event.Event{
		Kind: event.GuardianAssessment,
		Guardian: event.GuardianResult{
			ID:         id,
			Tool:       "ambient_watch",
			Subject:    filepath.Base(path),
			Outcome:    "allow",
			RiskLevel:  ambientToRiskLevel(result.Severity),
			Rationale:  result.Summary,
			DurationMs: 0,
		},
	})
}

func ambientToRiskLevel(s AmbientSeverity) string {
	switch s {
	case SeverityWarning:
		return "medium"
	case SeverityIssue:
		return "high"
	default:
		return "low"
	}
}

func (g *AmbientGuardian) reviewFile(ctx context.Context, path string) (AmbientResult, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return AmbientResult{}, err
	}
	fileHash, _ := hashFile(path)

	sess := agent.NewSession(ambientSystemPrompt)
	ag := agent.New(g.prov, g.roReg, sess, agent.Options{
		MaxSteps:       3,
		Temperature:    g.flashTemp,
		ContextWindow:  16_000,
		CompactRatio:   0.8,
	}, event.Discard)

	reviewCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	reviewPrompt := fmt.Sprintf(ambientReviewTemplate,
		path, filepath.Base(path), filepath.Ext(path),
		truncateAmbient(string(content), 8000))

	sess.Add(provider.Message{Role: provider.RoleUser, Content: reviewPrompt})

	agentErr := ag.Run(reviewCtx, "")

	result := AmbientResult{
		File:    path,
		ModTime: time.Now(),
		Hash:    fileHash,
	}

	if agentErr != nil {
		result.Severity = SeveritySafe
		result.Summary = fmt.Sprintf("ambient review failed for %s: %v", filepath.Base(path), agentErr)
		return result, nil
	}

	lastText := lastAssistantFromMessages(sess.Snapshot())
	parsed := parseAmbientVerdict(lastText)
	result.Severity = parsed.severity
	result.Summary = parsed.summary
	result.Detail = lastText
	result.Suggestion = parsed.suggestion

	return result, nil
}

const ambientSystemPrompt = "You are an ambient code reviewer embedded in the reasonix coding agent. Your task: inspect a single file that was just saved and look for bugs, security issues, or code-quality problems. You have read-only access.\n\nRules:\n1. Be SILENT if the file looks safe. Only surface real issues.\n2. Classify findings as \"safe\", \"warning\", or \"issue\".\n3. Never suggest edits — only flag what looks wrong.\n4. Keep your analysis brief (1-3 sentences per finding).\n5. Focus on: nil dereferences, race conditions, security issues, logic bugs, incorrect error handling, resource leaks.\n\nReply with JSON only:\n{\"severity\":\"safe|warning|issue\",\"summary\":\"one-line description\",\"suggestion\":\"optional fix hint\"}"

const ambientReviewTemplate = "File: %s\nName: %s\nExtension: %s\n\nContent:\n%s\n\nReview this file now. Output ONLY the JSON verdict."

type ambientVerdict struct {
	severity   AmbientSeverity
	summary    string
	suggestion string
}

func parseAmbientVerdict(text string) ambientVerdict {
	text = strings.TrimSpace(text)
	if text == "" {
		return ambientVerdict{severity: SeveritySafe, summary: "empty sub-agent output"}
	}
	if start := strings.IndexByte(text, '{'); start >= 0 {
		if end := strings.LastIndexByte(text, '}'); end > start {
			text = text[start : end+1]
		}
	}

	v := ambientVerdict{}
	v.summary = extractJSONString(text, "summary")
	v.suggestion = extractJSONString(text, "suggestion")
	sev := extractJSONString(text, "severity")

	switch strings.ToLower(strings.TrimSpace(sev)) {
	case "issue":
		v.severity = SeverityIssue
	case "warning":
		v.severity = SeverityWarning
	default:
		v.severity = SeveritySafe
	}
	if v.summary == "" {
		v.summary = "no issues detected"
		v.severity = SeveritySafe
	}
	return v
}

func extractJSONString(jsonText, key string) string {
	search := `"` + key + `"`
	idx := strings.Index(jsonText, search)
	if idx < 0 {
		return ""
	}
	idx += len(search)
	for idx < len(jsonText) && (jsonText[idx] == ':' || jsonText[idx] == ' ' || jsonText[idx] == '\t' || jsonText[idx] == '\n' || jsonText[idx] == '\r') {
		idx++
	}
	if idx >= len(jsonText) || jsonText[idx] != '"' {
		return ""
	}
	idx++
	start := idx
	escaped := false
	for idx < len(jsonText) {
		c := jsonText[idx]
		if escaped {
			escaped = false
			idx++
			continue
		}
		if c == '\\' {
			escaped = true
			idx++
			continue
		}
		if c == '"' {
			return jsonText[start:idx]
		}
		idx++
	}
	return ""
}

func lastAssistantFromMessages(msgs []provider.Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == provider.RoleAssistant && strings.TrimSpace(msgs[i].Content) != "" {
			return msgs[i].Content
		}
	}
	return ""
}

func truncateAmbient(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "\n... [truncated]"
}
