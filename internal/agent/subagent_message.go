package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// messengerKey is the context key for the SubagentMessenger singleton.
type messengerKey struct{}

// SubagentMessenger lets the parent agent send mid-task guidance to running
// sub-agents via their steer queue. It is thread-safe and lives for the
// duration of the controller session.
type SubagentMessenger struct {
	mu     sync.RWMutex
	agents map[string]*Agent // subagentID → Agent
}

// NewSubagentMessenger creates an empty messenger.
func NewSubagentMessenger() *SubagentMessenger {
	return &SubagentMessenger{
		agents: make(map[string]*Agent),
	}
}

// Register adds a running sub-agent to the messenger. The id should be the
// subagent reference (sa_...) or job ID that the parent will use to address
// messages. Must be paired with Unregister when the sub-agent completes.
func (m *SubagentMessenger) Register(id string, a *Agent) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents[id] = a
}

// Unregister removes a sub-agent from the messenger. Safe to call even if
// the id was never registered.
func (m *SubagentMessenger) Unregister(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.agents, id)
}

// Send delivers a message to the named sub-agent via its steer queue.
// The sub-agent will see it on its next tool-call round with the standard
// MidTurnSteerPrefix. Returns an error if the sub-agent is not found.
func (m *SubagentMessenger) Send(id, message string) error {
	m.mu.RLock()
	a, ok := m.agents[id]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("subagent %q not found (may have completed or never existed)", id)
	}
	a.Steer(message)
	return nil
}

// WithMessenger stores a messenger in the context so tools can reach it.
func WithMessenger(ctx context.Context, m *SubagentMessenger) context.Context {
	return context.WithValue(ctx, messengerKey{}, m)
}

// MessengerFromContext retrieves the messenger from the context.
func MessengerFromContext(ctx context.Context) (*SubagentMessenger, bool) {
	m, ok := ctx.Value(messengerKey{}).(*SubagentMessenger)
	return m, ok
}

// SendToSubagentTool is the model-facing tool that lets the parent send
// mid-task guidance to a running background sub-agent.
type SendToSubagentTool struct{}

// NewSendToSubagentTool creates the send_to_subagent tool.
func NewSendToSubagentTool() *SendToSubagentTool {
	return &SendToSubagentTool{}
}

func (*SendToSubagentTool) Name() string { return "send_to_subagent" }

func (*SendToSubagentTool) Description() string {
	return "Send a mid-task guidance message to a running background sub-agent. The sub-agent will receive it as a steer on its next tool-call round. Use the subagent reference (sa_...) or job ID from the sub-agent spawn result."
}

func (*SendToSubagentTool) Schema() json.RawMessage {
	return json.RawMessage(`{
"type":"object",
"properties":{
  "subagent_id":{"type":"string","description":"The subagent reference (e.g. sa_20260630_...) or job ID (e.g. task-3) to send the message to."},
  "message":{"type":"string","description":"The guidance message to send. Will be delivered as a mid-turn steer with the standard prefix."}
},
"required":["subagent_id","message"]
}`)
}

func (*SendToSubagentTool) ReadOnly() bool { return true }

func (s *SendToSubagentTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var p struct {
		SubagentID string `json:"subagent_id"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return "", fmt.Errorf("invalid args: %w", err)
	}
	if p.SubagentID == "" {
		return "", fmt.Errorf("subagent_id is required")
	}
	if p.Message == "" {
		return "", fmt.Errorf("message is required")
	}

	m, _ := MessengerFromContext(ctx)
	if m == nil {
		return "", fmt.Errorf("subagent messenger is not available in this context")
	}

	if err := m.Send(p.SubagentID, p.Message); err != nil {
		return "", err
	}

	return fmt.Sprintf("Message sent to sub-agent %q.", p.SubagentID), nil
}
