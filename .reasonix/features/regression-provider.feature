Feature: Regression — Provider Surface
  As a developer protecting reasonix's existing functionality
  I want no additive feature to break any provider's core behavior
  So that users on Anthropic, DeepSeek, MiniMax, and generic OpenAI providers are unaffected

  Background:
    Given reasonix has provider implementations for anthropic, openai (DeepSeek/MiniMax/generic)

  # ── Anthropic Messages API ────────────────────────────────────────

  @regression-provider-anthropic-basic
  Scenario: Basic Anthropic message returns text
    Given the anthropic provider is configured with model "claude-opus-4-8"
    When a simple message "What is the capital of France?" is sent
    Then the response contains text
    And the stop_reason is "end_turn" or "stop"
    And usage.input_tokens > 0

  @regression-provider-anthropic-streaming
  Scenario: Anthropic streaming produces incremental deltas
    Given the anthropic provider is streaming
    When a message "Write a haiku" is sent
    Then multiple text deltas are received
    And the accumulated text forms a coherent response
    And usage is emitted in message_delta

  @regression-provider-anthropic-tool-use
  Scenario: Anthropic tool use round-trips correctly
    Given a tool "get_weather" is defined with a location parameter
    When a message "What's the weather in Paris?" is sent
    Then a tool_use block is received with name "get_weather"
    And the tool_use input contains "Paris"
    And when a tool_result is sent back, the model produces a final text answer

  @regression-provider-anthropic-thinking
  Scenario: Anthropic adaptive thinking works
    Given the anthropic provider has thinking: "adaptive"
    When a message "Solve: 27 × 453" is sent
    Then the response contains thinking blocks with signature
    And the thinking blocks precede the text blocks
    And the signature is non-empty

  @regression-provider-anthropic-cache
  Scenario: Anthropic prompt caching produces cache hits
    Given the anthropic provider has a large system prompt
    When two identical requests are sent
    Then the second request shows cache_read_input_tokens > 0
    And the second request has lower input_tokens than the first

  @regression-provider-anthropic-images
  Scenario: Anthropic vision processes images
    Given the anthropic provider has vision enabled
    When a message with a base64 PNG image "What's in this image?" is sent
    Then the request body contains an image content block with type "image"
    And the response contains text describing the image

  @regression-provider-anthropic-refusal
  Scenario: Anthropic refusal stop_reason is handled
    Given the anthropic provider
    When a request triggers a refusal (e.g., harmful content)
    Then stop_reason is "refusal"
    And stop_details is non-null
    And the agent does not crash trying to read content[0].text

  @regression-provider-anthropic-baseurl
  Scenario: Anthropic custom base_url with /v1 suffix works
    Given the anthropic provider has base_url "https://proxy.example.com/v1"
    When the provider is constructed
    Then the /v1 suffix is stripped from the stored baseURL
    And requests go to "https://proxy.example.com/v1/messages", not "/v1/v1/messages"

  # ── OpenAI / DeepSeek ──────────────────────────────────────────────

  @regression-provider-deepseek-reasoning
  Scenario: DeepSeek reasoning_content preserved on tool-call turns
    Given the provider is DeepSeek with thinking enabled
    When a tool call is made and the assistant turn has reasoning_content
    Then the next request includes reasoning_content for that assistant message
    And the API does not return 400

  @regression-provider-deepseek-reasoning
  Scenario: DeepSeek reasoning_content stripped on no-tool turns
    Given the provider is DeepSeek
    When a final answer is produced with no tool calls and reasoning_content present
    Then the next request (if conversation continues) does NOT include reasoning_content
    And no token waste occurs from re-uploading reasoning

  @regression-provider-deepseek-effort
  Scenario: DeepSeek effort aliasing works
    Given the provider is DeepSeek
    When effort "low" is configured
    Then a notice fires: "DeepSeek maps 'low' to 'high'"
    And the actual effort sent to the API is "high"

  @regression-provider-deepseek-effort
  Scenario: DeepSeek max effort preserved
    Given the provider is DeepSeek
    When effort "max" is configured
    Then the actual effort sent to the API is "max"
    And no aliasing notice fires

  @regression-provider-deepseek-non-thinking
  Scenario: DeepSeek non-thinking mode works
    Given the provider is DeepSeek with thinking disabled
    When a message "What is 2+2?" is sent
    Then no reasoning_content is returned
    And the response contains only content

  # ── OpenAI / MiniMax ──────────────────────────────────────────────

  @regression-provider-minimax
  Scenario: MiniMax thinking mode uses adaptive toggle
    Given the provider is MiniMax
    When thinking is enabled
    Then the request includes thinking.type = "adaptive"
    And no reasoning_effort parameter is present
    And the response is received without error

  # ── OpenAI / Generic Compatible ────────────────────────────────────

  @regression-provider-generic
  Scenario: Generic OpenAI provider uses reasoning_effort scale
    Given the provider is a generic OpenAI-compatible endpoint (not DeepSeek, not MiniMax)
    When effort "medium" is configured
    Then the request includes reasoning_effort: "medium"
    And no thinking.type parameter is present

  @regression-provider-generic
  Scenario: Generic provider chat_url override works
    Given the provider has chat_url "https://api.example.com/custom/chat"
    When a request is built
    Then the request goes to "https://api.example.com/custom/chat"
    And not to base_url + "/chat/completions"

  # ── All Providers ──────────────────────────────────────────────────

  @regression-provider-retry
  Scenario: Rate limit triggers retry with backoff
    Given any provider returns 429 with retry-after: 1
    When the SDK retry logic engages
    Then the request is retried after at least 1 second
    And the retry count is incremented
    And after max_retries, the error is surfaced

  @regression-provider-retry
  Scenario: Server error triggers retry
    Given any provider returns 500
    When the SDK retry logic engages
    Then the request is retried
    And retries continue until success or max_retries

  @regression-provider-proxy
  Scenario: HTTP_PROXY is honored
    Given HTTP_PROXY is set to "http://proxy:8080"
    When any provider makes a request
    Then the request goes through the proxy
    And the proxy environment variable is respected

  @regression-provider-model-resolution
  Scenario: "provider/model" syntax resolves
    Given config has provider "deepseek" with models ["v4-flash", "v4-pro"] and default "v4-flash"
    When model "deepseek/v4-pro" is requested
    Then the resolved model is "v4-pro" on the "deepseek" provider

  @regression-provider-model-resolution
  Scenario: Bare model name resolves to correct provider
    Given config has providers with overlapping model names
    When model "v4-flash" is requested
    Then the first provider listing "v4-flash" in its models is used

  @regression-provider-model-resolution
  Scenario: Provider name resolves to its default model
    Given config has provider "deepseek" with default "v4-flash"
    When model "deepseek" is requested
    Then the resolved model is "v4-flash" on the "deepseek" provider

  @regression-provider-token-count
  Scenario: Token counting returns provider-specific count
    Given providers DeepSeek and Anthropic are both configured
    When count_tokens is called with the same text on both providers
    Then the two counts may differ (different tokenizers)
    And each count corresponds to its provider's tokenizer
