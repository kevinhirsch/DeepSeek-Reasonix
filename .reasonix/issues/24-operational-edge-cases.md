# 24 — Operational Edge Cases: CI, Release, Security, Dependencies, Backward Compat, Performance

**Phase:** Resilience | **Effort:** 4 days total | **Priority:** P2

## O1: CI/CD Pipeline — 1 day
New Makefile targets, GHA workflows, coverage gates. See `issues/23-operational-gaps.md` for full spec.

## O2: Release Versioning — 0.5 day
Semantic versioning, changelog format, migration notice on first run. Dogfood Comp 24 (automated release notes).

## O3: Backward Compatibility Testing — 0.5 day
Cross-version session loading. v1.0 → v2.0 normalization. v2.0 → v1.0 graceful rejection. Config migration with real v1.0 samples.

## O4: Performance Regression CI — 0.5 day
Benchmark suite. Baseline storage. CI gate at >10% regression. Historical dashboard.

## O5: Security Audit of New Tools — 1 day
Per-tool checklist: web_search (SSRF), monitor (injection), send_notification (XSS), KB (data at rest), webhook (URL validation). Update `SECURITY.md`.

## O6: License/Dependency Audit — 0.5 day
CI gate: dependency count must remain 1 (BurntSushi/toml). Exception: `DEP:` commit trailer with justification.

## Files
- `Makefile` — new targets: test-short, test-full, test-live, test-stress, build-worker, cross-worker
- `.github/workflows/ci.yml` (NEW) — CI pipeline
- `.github/workflows/nightly.yml` (NEW) — live API tests
- `.github/workflows/weekly.yml` (NEW) — stress tests
- `docs/RELEASING.md` — update for v2.0
- `SECURITY.md` — per-tool security review
- `internal/agent/testutil/compat_test.go` (NEW) — cross-version tests
- `internal/perf/bench_test.go` (NEW) — benchmark suite

## Acceptance
- [ ] `make test-short` completes in <2 min, no network
- [ ] `make test-full` completes in <10 min, mock providers only
- [ ] `make test-live` runs nightly against real APIs
- [ ] `make test-stress` runs weekly, 1000-subagent spawn
- [ ] CI gate blocks merge on: test failure, coverage drop, dependency addition, >10% perf regression
- [ ] Cross-version test: v1.0 session loads in v2.0
- [ ] Cross-version test: v2.0 session rejected gracefully in v1.0
- [ ] All new tools pass security review checklist
- [ ] Dependency count audit gate active
