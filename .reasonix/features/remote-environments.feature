Feature: Remote Environments
  As a developer with a build server
  I want to offload subagent execution to remote machines
  So that resource-intensive tasks run on dedicated hardware without blocking my local machine

  Background:
    Given reasonix is configured with a remote "build-server" at https://build-server:9090
    And the remote machine is running reasonix-worker with Docker

  @09-bootstrap
  Scenario: Generate bootstrap script for Ubuntu remote
    When the user runs "reasonix remote bootstrap --name build-server --os ubuntu"
    Then a self-contained shell script is generated
    And the script detects OS automatically
    And the script installs Docker if missing
    And the script probes CPU, RAM, and disk
    And the script auto-calculates container limits at 80% of system resources
    And the script downloads reasonix-worker from GitHub releases
    And the script writes worker.toml with auto-detected limits
    And the script prompts for API keys interactively
    And the script installs a systemd service with NoNewPrivileges=true
    And the script starts the worker

  @09-bootstrap
  Scenario: Bootstrap script is idempotent
    Given the bootstrap script has already been run once on the remote machine
    When the script is run again
    Then Docker is not re-installed
    And the worker.toml is overwritten with current settings
    And the systemd service is restarted

  @10-worker
  Scenario: Worker claims and executes a subagent
    Given a work item exists in the queue for subagent "Run test suite on auth module"
    When the remote worker long-polls and claims the item
    Then no other worker can claim the same item
    And a Docker container is created with the workspace mounted at /workspace
    And the container has DEEPSEEK_API_KEY injected as an environment variable
    And the container's network is restricted to api.deepseek.com
    And "reasonix run --prompt 'Run test suite on auth module'" executes inside the container
    And stdout and stderr are streamed back as work events
    And the container is destroyed on completion

  @10-worker
  Scenario: Container timeout enforced
    Given the worker is configured with container_timeout_minutes: 30
    And a subagent has been running for 31 minutes
    Then the container is killed
    And the work item is marked as failed with reason "timeout"
    And no Docker resources are orphaned

  @10-worker
  Scenario: Worker drains in-flight work on shutdown
    Given the worker has 2 containers running subagents
    When the worker receives SIGTERM
    Then new work items are not claimed
    And the worker waits for both containers to complete
    And the worker exits after both containers finish

  @12-work-queue
  Scenario: Worker polls for pending work
    When a remote worker sends GET /v1/work/pending
    And no work items are pending
    Then the server holds the connection for 30 seconds
    And if work arrives within 30 seconds, the work item is returned
    And if no work arrives, an empty response is returned after 30 seconds

  @12-work-queue
  Scenario: Work claim is atomic
    Given a work item "work_abc" is pending
    And two workers claim it simultaneously
    Then exactly one worker receives a success response
    And the other worker receives a 409 Conflict

  @12-work-queue
  Scenario: Unclaimed work expires
    Given a work item was created 6 minutes ago
    And the configured work expiration is 5 minutes
    When a worker polls for pending work
    Then the expired work item is not returned

  @13-docker
  Scenario: Subagent writes files in Docker container
    Given a Docker container is running a subagent with workspace mounted read-write
    When the subagent writes a file to /workspace/output.txt
    Then the file is persisted on the host at the workspace mount path
    And the file survives container destruction

  @13-docker
  Scenario: API key not visible in container filesystem
    Given a Docker container was created with DEEPSEEK_API_KEY as an env var
    When a subagent runs "env" inside the container
    Then DEEPSEEK_API_KEY is visible in the environment
    But the key value does not appear in any file under /workspace
    And the key value does not appear in any file under /etc

  @14-config
  Scenario: Remote declared in reasonix.toml
    Given reasonix.toml contains:
      """
      [[remotes]]
      name = "build-server"
      url = "https://build-server:9090"
      auth_token = "${REASONIX_REMOTE_TOKEN}"
      max_concurrent = 4
      """
    When reasonix loads configuration
    Then the remote "build-server" is available with max 4 concurrent containers
    And the auth token is resolved from the REASONIX_REMOTE_TOKEN environment variable
    And the token value is never stored in the config file

  @14-config
  Scenario: Remote status shown in settings
    Given a remote "build-server" is configured and online
    When the user opens the remotes settings panel
    Then "build-server" shows 🟢 online
    And the panel shows current load (2/4 containers) and capabilities (OS, CPU, RAM)

  @15-target
  Scenario: Subagent routed to remote by target field
    When the model calls task with target: "build-server" and prompt: "Run full test suite"
    Then the subagent spec is serialized and posted to the build-server work queue
    And the subagent does NOT run locally
    And the background tasks panel shows 🌐 for this subagent

  @15-target
  Scenario: Subagent runs locally when target omitted
    When the model calls task with no target field
    Then the subagent runs in the local sandbox
    And no remote worker is contacted

  @15-target
  Scenario: Remote offline produces clear error
    Given the remote "build-server" is offline
    When the model calls task with target: "build-server"
    Then the tool returns an error: "remote 'build-server' is not reachable"
    And no work item is created

  # ── FE: Remote Panel Status ──────────────────────────────────────

  @fe-remote-panel-status
  Scenario: Remotes panel shows real-time load per worker
    Given remote "build-server" has 2 of 4 containers active
    When the remotes panel renders
    Then "build-server" shows a load bar: [████░░░░░░] 2/4
    And the bar color is green (<50%), yellow (50-80%), red (>80%)
    And hovering shows: "2 subagents running · 2 slots available · avg CPU 45% · avg RAM 3.2/8GB"

  @fe-remote-worker-latency
  Scenario: Remotes panel shows latency per worker
    Given remote "build-server" has 12ms latency and "gpu-runner" has 85ms latency
    When the remotes panel renders
    Then each worker shows its latency with a color: green (<30ms), yellow (30-100ms), red (>100ms)
    And the latency is updated every health check interval

  @fe-remote-bootstrap-ui
  Scenario: Bootstrap progress renders in terminal
    When the user runs "reasonix remote bootstrap --name build-server"
    Then the output shows steps with spinners:
      And "⠋ Detecting OS... → ✓ Ubuntu 24.04 detected"
      And "⠙ Installing Docker... → ✓ Docker 26.1.3 installed"
      And "⠹ Probing resources... → ✓ 8 CPUs, 32GB RAM, 200GB disk"
      And "⠸ Generating config... → ✓ worker.toml written"
      And "⠼ Testing API key... → ✓ DeepSeek API key valid"
      And finally: "✓ Worker 'build-server' ready. Run the bootstrap script on the remote machine."
    And each checkmark is green

  @fe-remote-test-command
  Scenario: Remote test command shows ping result
    When the user runs "reasonix remote test build-server"
    Then the output shows: "Pinging build-server... ✓ 12ms"
    And "Spawning test subagent... ✓ completed in 2.3s"
    And "Round-trip: 2.3s total · 12ms network · 2.3s execution"
    And the test result is green if all steps pass
