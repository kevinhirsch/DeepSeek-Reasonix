Feature: Remote Worker Security
  As a developer running subagents on remote infrastructure
  I want the remote execution channel to be secure against impersonation, interception, and exfiltration
  So that a compromised build server cannot leak my code or API keys

  Background:
    Given reasonix has remote worker infrastructure (Issues #9-15)
    And TLS is enabled on all work queue endpoints

  # ── Worker Authentication ─────────────────────────────────────────

  @remote-sec-auth
  Scenario: Each worker has a unique authentication token
    Given 3 workers are configured: build-server, gpu-runner, test-runner
    When the worker tokens are generated
    Then each worker receives a unique token
    And no token is shared between workers
    And the token format is sk-ant-oat01-<random>

  @remote-sec-auth
  Scenario: Worker authenticates with its token on every request
    Given worker "build-server" has token "sk-ant-oat01-abc123"
    When the worker polls GET /v1/work/pending
    Then the request includes Authorization: Bearer sk-ant-oat01-abc123
    And the server validates the token before returning work items

  @remote-sec-auth
  Scenario: Invalid token rejected
    Given a request carries an invalid or expired token
    When the server receives it
    Then 401 is returned
    And the worker name is NOT revealed in the error
    And the failed attempt is logged with timestamp and source IP

  @remote-sec-auth
  Scenario: Token rotation invalidates old token
    Given worker "build-server" has token "sk-ant-oat01-old"
    When the token is rotated to "sk-ant-oat01-new"
    Then requests with the old token return 401
    And requests with the new token succeed
    And the transition is atomic (no window where both tokens work)

  # ── TLS & Transport Security ───────────────────────────────────────

  @remote-sec-tls
  Scenario: Work queue endpoint requires TLS
    Given the work queue is served at https://build-server:9090
    When a worker attempts to connect via plain HTTP
    Then the connection is rejected
    And the worker logs a TLS requirement error

  @remote-sec-tls
  Scenario: Certificate pinning prevents MITM
    Given the worker has the server's certificate fingerprint configured
    When a MITM presents a different certificate
    Then the worker refuses the connection
    And the worker logs a certificate mismatch error

  @remote-sec-tls
  Scenario: Self-signed certificates accepted with explicit config
    Given the server uses a self-signed certificate
    And the worker config has tls_skip_verify = false (default) but tls_ca_cert points to the CA
    When the worker connects
    Then the connection succeeds after CA validation
    And the certificate chain is verified

  # ── Work Item Confidentiality ──────────────────────────────────────

  @remote-sec-confidentiality
  Scenario: Work item payloads encrypted for the target worker
    Given worker "build-server" has a public key registered
    When a work item is created targeting "build-server"
    Then the work item prompt and parameters are encrypted with the worker's public key
    And only the worker (with the private key) can decrypt the payload
    And the encrypted payload is unreadable in transit and at rest

  @remote-sec-confidentiality
  Scenario: Work item decrypted only inside the sandbox
    Given the worker received an encrypted work item
    When the worker decrypts it
    Then decryption happens inside the Docker container
    And the decrypted prompt is written to a tmpfs (memory-only) mount
    And the decrypted prompt is never written to the host filesystem

  # ── API Key Scoping ────────────────────────────────────────────────

  @remote-sec-api-keys
  Scenario: Worker API keys scoped to specific models
    Given the worker needs DeepSeek Flash for finders and DeepSeek Pro for verifiers
    When the API key is provisioned
    Then the key is scoped to only those models
    And requests to other models (e.g., the most expensive Pro-max variant) are rejected

  @remote-sec-api-keys
  Scenario: Worker API key has spending limit
    Given the worker API key has a monthly spending limit of $50
    When the worker's cumulative spend approaches $50
    Then a warning is emitted at 80%: "Worker API key at 80% of monthly limit ($40/$50)"
    And at 100%, requests are rejected: "Monthly spending limit reached"

  @remote-sec-api-keys
  Scenario: API key never logged or surfaced
    Given the worker uses an API key for DeepSeek calls
    When the worker logs request information
    Then the API key is never included in logs
    And the key is masked in any error messages: "Authorization: Bearer sk-...abc"
    And the key value cannot be retrieved by any tool or endpoint

  # ── Replay Attack Prevention ───────────────────────────────────────

  @remote-sec-replay
  Scenario: Work item claim requests are nonce-protected
    Given a work item claim request is captured by an attacker
    When the attacker replays the captured request
    Then the server detects the replayed nonce
    And 409 Conflict is returned: "Nonce already used"
    And the original claim (if it succeeded) is not invalidated

  @remote-sec-replay
  Scenario: Event stream replay is protected
    Given the worker streams events to POST /v1/work/{id}/events
    And each event carries a sequence number
    When an attacker replays event #3
    Then the server detects the duplicate sequence number
    And the duplicate is silently dropped
    And the original event ordering is preserved

  # ── Worker Identity Verification ───────────────────────────────────

  @remote-sec-identity
  Scenario: Worker identity verified on first connection
    Given a new worker attempts to register as "build-server"
    When the worker first connects
    Then the server issues a challenge
    And the worker must prove it controls the private key associated with "build-server"
    And if verification fails, the worker is rejected

  @remote-sec-identity
  Scenario: Worker impersonation detected
    Given worker "build-server" is already connected
    When another process attempts to connect as "build-server"
    Then the server detects the duplicate identity
    And the existing connection is NOT terminated (surviving connection wins)
    And the new connection is rejected: "Worker 'build-server' is already connected"
    And an alert is emitted: "Possible impersonation attempt for 'build-server'"

  # ── UI ─────────────────────────────────────────────────────────────

  @remote-sec-ui
  Scenario: Security status shown in remotes panel
    Given remote "build-server" is connected with TLS and worker-auth
    When the remotes panel renders
    Then the connection shows 🔒 for TLS
    And "Worker authenticated" is shown for identity verification
    And the token age is shown: "Token rotated 3 days ago"

  @remote-sec-ui
  Scenario: Security incident alert
    Given a failed authentication attempt was detected
    When the user opens reasonix
    Then an alert is shown: "⚠ Security: 3 failed auth attempts for worker 'build-server' from IP 192.168.1.100"
    And a "Review" button opens the security log
