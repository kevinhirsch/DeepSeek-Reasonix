package agent

import (
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// TranscriptRetention manages auto-rotation of subagent transcripts.
// Keeps 30 days or 1,000 latest, whichever is larger.
type TranscriptRetention struct {
	mu         sync.Mutex
	dir        string
	maxAge     time.Duration
	maxCount   int
}

// NewTranscriptRetention creates a retention manager.
func NewTranscriptRetention(dir string) *TranscriptRetention {
	return &TranscriptRetention{
		dir:      dir,
		maxAge:   30 * 24 * time.Hour,
		maxCount: 1000,
	}
}

// Cleanup removes expired transcripts. Preserves those referenced by
// active workflows or recent sessions.
func (r *TranscriptRetention) Cleanup(referenced []string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}

	refSet := make(map[string]bool)
	for _, ref := range referenced {
		refSet[ref] = true
	}

	type entryInfo struct {
		name    string
		modTime time.Time
	}
	var files []entryInfo
	cutoff := time.Now().Add(-r.maxAge)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, entryInfo{e.Name(), info.ModTime()})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	removed := 0
	for _, f := range files {
		if refSet[f.name] {
			continue
		}
		shouldRemove := f.modTime.Before(cutoff)
		if !shouldRemove && len(files)-removed > r.maxCount {
			shouldRemove = true
		}
		if shouldRemove {
			os.Remove(filepath.Join(r.dir, f.name))
			removed++
		}
	}
	return nil
}
