# 11 — cmd/reasonix-worker Binary

**Phase:** Remote | **Effort:** 0.5 day | **Priority:** P2

## Problem

Remote worker needs dedicated binary entry point. Headless daemon, no TUI.

## Design

`cmd/reasonix-worker/main.go` — minimal entry:
```go
func main() {
    w, _ := worker.Load(os.Args[1:])
    w.Run(ctx)
}
```

Built alongside `reasonix` in Makefile. Included in GitHub releases for bootstrap download.

## Files
- `cmd/reasonix-worker/main.go` (NEW)
- `Makefile` — add worker target
- `.goreleaser.yaml` — add binary to releases

## Acceptance
- [ ] `reasonix-worker --config worker.toml` starts daemon
- [ ] Cross-compiles for all 6 targets
- [ ] CGO_ENABLED=0 static binary
- [ ] In release artifacts
