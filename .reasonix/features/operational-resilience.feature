Feature: Operational Resilience
  As a developer who depends on reasonix daily
  I want the system to handle corrupted state, disk exhaustion, force quits, and environmental edge cases
  So that reasonix never loses my work and always has a recovery path

  Background:
    Given reasonix has operational resilience features active
    And all persistence uses atomic writes (temp file + rename)

  # ── O7: Corrupted State Recovery ─────────────────────────────────

  @op-corruption-detect
  Scenario: Corrupted knowledge base detected on boot
    Given the knowledge base SQLite file is corrupted (invalid header)
    When reasonix starts
    Then the corruption is detected during boot validation
    And a notice shows: "⚠ Knowledge base corrupted. Rebuilding... (previous analyses preserved in transcripts)"
    And the KB is re-initialized as an empty database
    And reasonix starts normally with an empty KB

  @op-corruption-detect
  Scenario: Corrupted prompt library falls back to built-in defaults
    Given a prompt file is manually edited to invalid JSON
    When the prompt library loads
    Then the corrupted file is detected
    And a notice shows: "⚠ explorer.md is corrupted. Using built-in default prompt."
    And the built-in default is loaded
    And the corrupted file is renamed to explorer.md.corrupted for user inspection

  @op-corruption-detect
  Scenario: Corrupted confidence profiles reset to zero
    Given the confidence profiles JSON is truncated mid-write
    When the profiles load
    Then the corruption is detected
    And a notice shows: "⚠ Confidence profiles corrupted. Reset to empty. Profiles will rebuild over time."
    And all profiles are re-initialized as empty
    And routing falls back to default (no profile-based routing)

  @op-corruption-detect
  Scenario: Corrupted scheduled tasks file handled gracefully
    Given `.reasonix/scheduled_tasks.json` is corrupted
    When the schedule loads
    Then a notice shows: "⚠ Scheduled tasks file corrupted. No tasks loaded. Previous tasks backed up to scheduled_tasks.json.corrupted."
    And no cron jobs are loaded
    And durable jobs are re-created from the backup if recoverable

  @op-corruption-detect
  Scenario: Checksum validation catches silent corruption
    Given every persisted JSON file has an accompanying `.sha256` checksum file
    And a file's content has been silently corrupted (bit flip on disk)
    When the file is loaded
    Then the checksum mismatch is detected
    And the file is treated as corrupted
    And the recovery path for that file type is triggered

  # ── O14: Factory Reset ───────────────────────────────────────────

  @op-factory-reset
  Scenario: Factory reset with keep-config preserves config
    When the user runs "reasonix reset --keep-config"
    Then a confirmation prompt shows: "This will reset: knowledge base, prompt library, confidence profiles, telemetry, subagent transcripts. Your reasonix.toml and sessions will be preserved. Continue? [y/N]"
    And confirming deletes: .reasonix/knowledge/, .reasonix/prompts/v1/ (user-edited), .reasonix/confidence/, .reasonix/telemetry/
    And preserving: reasonix.toml, sessions/, scheduled_tasks.json (durable)
    And a backup is created at .reasonix/backups/reset-YYYY-MM-DD-HHMM.tar.gz
    And on completion: "✓ Reset complete. Config preserved. Backup: .reasonix/backups/reset-2026-07-01-1422.tar.gz"

  @op-factory-reset
  Scenario: Full factory reset wipes everything
    When the user runs "reasonix reset --everything"
    Then a confirmation prompt shows: "This will reset EVERYTHING. Your config, knowledge base, prompts, profiles, sessions, telemetry — all of it. A backup will be created. Continue? [y/N]"
    And confirming requires typing "RESET" (not just y/N)
    And on confirmation, everything under .reasonix/ is backed up then deleted
    And reasonix restarts into first-run setup flow

  @op-factory-reset
  Scenario: Factory reset is idempotent
    Given a factory reset was just completed
    When the user runs "reasonix reset --everything" again
    Then the reset still succeeds (nothing to delete, but backup still created)
    And no error occurs from trying to delete already-deleted paths

  # ── O9: Disk Space Exhaustion ────────────────────────────────────

  @op-disk-monitor
  Scenario: Disk space warning at 90%
    Given the disk containing .reasonix/ is 90% full
    When the disk monitor checks (every 5 minutes)
    Then a warning notice shows: "⚠ Disk 90% full (45GB free). Consider cleaning up old sessions or running 'reasonix reset --keep-config'."
    And the status line shows a disk indicator: "💾 90%"

  @op-disk-monitor
  Scenario: Degraded mode at 95%
    Given the disk is 95% full
    When the degraded threshold is reached
    Then transcript writes are paused (in-memory only, warning shown)
    And KB eviction is aggressive (evict 50% of entries)
    And telemetry writes are paused
    And new subagent transcripts are ephemeral only (not persisted)
    And a prominent notice shows: "🔴 Disk 95% full — running in degraded mode. Free space to restore normal operation."
    And the notice persists until space is freed

  @op-disk-monitor
  Scenario: Critical mode at 98%
    Given the disk is 98% full
    When the critical threshold is reached
    Then reasonix enters read-only mode
    And a notice shows: "⛔ Disk 98% full — read-only mode. No writes possible. Free space immediately."
    And the user can still read files and view transcripts
    But no tool calls that write files are permitted
    And the agent is informed: "Disk is full. Read-only until space is freed."

  @op-disk-monitor
  Scenario: Recovery when space freed
    Given reasonix was in degraded mode at 95%
    When the user frees space to 85%
    Then the disk monitor detects the recovery
    And a notice shows: "✓ Disk 85% — normal operation restored."
    And transcript writes resume
    And KB eviction returns to normal policy
    And deferred writes are flushed

  # ── O8: Force Quit / Sleep Recovery ──────────────────────────────

  @op-force-quit
  Scenario: SIGKILL mid-write does not corrupt state
    Given a subagent transcript is being written via atomic write (temp + rename)
    When the process receives SIGKILL during the temp file write
    Then the original file is intact (rename never happened)
    And the partial temp file is cleaned up on next boot
    And no data is lost beyond the in-progress write

  @op-force-quit
  Scenario: SIGKILL mid-KB-write recovers cleanly
    Given the knowledge base is writing an entry via atomic write
    When SIGKILL occurs mid-write
    Then the KB is intact (atomic write guarantees: either old file or new file, never partial)
    And on next boot, the KB loads normally

  @op-force-quit
  Scenario: Sleep/wake during remote worker communication recovers
    Given a remote worker is executing a subagent
    And the laptop sleeps for 5 minutes
    When the laptop wakes
    Then the SSH/HTTP connection to the remote worker is detected as broken
    And the work item is NOT re-queued (the remote worker may still be executing)
    And a notice shows: "⚠ Connection to build-server lost during sleep. Check worker status."
    And the user can manually check or wait for the worker's next heartbeat

  @op-force-quit
  Scenario: Battery death mid-operation recovers
    Given reasonix is writing a workflow manifest
    And the battery dies mid-write (equivalent to SIGKILL)
    When the laptop powers on and reasonix starts
    Then the workflow manifest is intact (atomic write)
    And the workflow is detected as interrupted (manifest status = "running")
    And the crash recovery flow (Feature: workflow-crash-recovery) is triggered

  # ── O11: Symlink/TOCTOU Sandbox ──────────────────────────────────

  @op-sandbox-symlink
  Scenario: Symlink escape from workspace is blocked
    Given the sandbox WriteRoots is ["/workspace"]
    When a subagent creates a symlink `/workspace/escape` → `/etc/passwd`
    Then the symlink creation is allowed (it's within WriteRoots)
    But when the subagent tries to read `/workspace/escape`
    Then the sandbox resolves the symlink target to `/etc/passwd`
    And blocks the read: "Path escapes workspace root via symlink: /etc/passwd"

  @op-sandbox-toctou
  Scenario: TOCTOU attack caught — file replaced with symlink
    Given a subagent creates a file `/workspace/data.txt`
    And the path is validated as within WriteRoots
    When the subagent replaces data.txt with a symlink to `/etc/passwd` before writing
    Then the sandbox resolves the symlink before the write
    And blocks the write: "Path escapes workspace root via symlink: /etc/passwd"

  @op-sandbox-git-hooks
  Scenario: Git hook injection blocked
    Given `.git/hooks/` is in ForbidReadRoots by default
    When a subagent attempts to write to `.git/hooks/pre-commit`
    Then the sandbox blocks the write
    And returns: "Path is in a forbidden directory: .git/hooks/"

  # ── O10: Concurrent Config Modification ──────────────────────────

  @op-config-watcher
  Scenario: External config edit detected and auto-reloaded
    Given reasonix is running with reasonix.toml loaded at boot
    When the user edits reasonix.toml in an external editor
    Then the file watcher detects the change within 5 seconds
    And a notice shows: "Config changed externally. Reloading..."
    And the new config values take effect immediately
    And the active model, effort, and provider settings update

  @op-config-watcher
  Scenario: Settings panel reads before write to avoid clobbering
    Given the user has the settings panel open
    And the user edited reasonix.toml externally
    When the user saves a setting in the settings panel
    Then the panel re-reads the current config first
    And if the config has changed externally, a warning shows: "Config changed externally since panel opened. Your change will be merged."
    And the user's change is applied on top of the external changes

  @op-config-watcher
  Scenario: Invalid external config edit is rejected with clear error
    Given the user edits reasonix.toml with invalid TOML syntax
    When the file watcher detects the change
    Then the config load fails
    And a notice shows: "⚠ reasonix.toml has a syntax error at line 42. Using previous valid config. Fix the file and it will auto-reload."
    And reasonix continues with the last known good config

  # ── O12: Multi-Monitor & DPI ─────────────────────────────────────

  @op-desktop-dpi
  Scenario: Window renders correctly at 200% DPI
    Given the monitor is at 200% DPI scaling
    When the desktop app renders
    Then all text is sharp (not blurry from bitmap scaling)
    And UI elements are correctly sized (not tiny)
    And the composer, transcript, and panels are fully visible

  @op-desktop-multi-monitor
  Scenario: Window moves between different DPI monitors
    Given monitor A is at 125% and monitor B is at 200%
    When the user drags the reasonix window from A to B
    Then the window re-renders at the new DPI
    And no elements are clipped or offscreen
    And the window size adjusts proportionally

  @op-desktop-unplug
  Scenario: External monitor unplugged recovers window
    Given reasonix is displayed on an external monitor
    When the monitor is unplugged
    Then the window moves to the primary display
    And the window is fully visible (not offscreen)
    And the window state is preserved

  # ── O13: Terminal Edge Cases ─────────────────────────────────────

  @op-tui-minimum
  Scenario: TUI degrades gracefully at 40 columns
    Given the terminal is 40 columns wide
    When the TUI renders
    Then the chat area, composer, and status line are all visible
    And no text is clipped
    And the layout switches to single-column compact mode
    And long labels are truncated with ellipsis

  @op-tui-dumb
  Scenario: TERM=dumb falls back to plain text
    Given TERM=dumb (no ANSI support)
    When the TUI renders
    Then no ANSI escape sequences are emitted
    And all text is plain
    And spinners are replaced with static text: "[running]" not "⠋"
    And colors are absent

  @op-tui-cjk
  Scenario: CJK double-width characters maintain alignment
    Given the terminal supports CJK
    When the TUI renders a panel containing Chinese text
    Then column alignment is correct
    And double-width characters consume 2 columns
    And borders and dividers align correctly

  @op-tui-tmux
  Scenario: tmux nested session keybinding conflict detected
    Given reasonix is running inside tmux
    And the reasonix keybinding Ctrl+B conflicts with tmux prefix
    When reasonix detects tmux
    Then a notice shows: "⚠ Ctrl+B conflicts with tmux prefix. Background panel toggle rebound to Ctrl+Shift+B."
    And the rebound key works

  # ── O15: Network Partition Mid-Operation ─────────────────────────

  @op-network-partition
  Scenario: Network loss during subagent execution surfaces clear error
    Given a subagent is streaming output from DeepSeek
    When the network drops mid-stream
    Then the stream error is detected within the idle timeout (120s)
    And the subagent is marked "failed — network error"
    And the partial output is preserved and displayed
    And a notice shows: "⚠ Subagent 'auth-review' failed: network error. Retry?"
    And a "Retry" button is available

  @op-network-partition
  Scenario: Network loss during prompt calibration surfaces clear error
    Given prompt calibration is running live API evaluations
    When the network drops during scenario 3 of 4
    Then the calibration run fails with: "Network error during scenario 3/4. 2 scenarios completed. Results saved."
    And the partial results are available for review
    And the calibration can be resumed

  @op-network-partition
  Scenario: Network loss does not affect local operations
    Given the network is down
    When the user asks a question answerable from the knowledge base
    Then the answer is served from the local KB (no network needed)
    And when the user opens a local file
    Then the file opens normally (no network needed)
    And local-only features continue to work
