package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Streamer sends sub-agent events back to the server as an event stream.
type Streamer struct {
	serverURL string
	authToken string
	client    *http.Client
}

// NewStreamer creates an event streamer.
func NewStreamer(serverURL, authToken string) *Streamer {
	return &Streamer{
		serverURL: serverURL,
		authToken: authToken,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// StreamEvent is one event to send to the server.
type StreamEvent struct {
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
	Tool      string `json:"tool,omitempty"`
	ToolCalls int    `json:"tool_calls,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Send posts a single event to the server's work item event endpoint.
func (s *Streamer) Send(ctx context.Context, workID string, ev StreamEvent) error {
	jsonBody, _ := json.Marshal(ev)
	req, err := http.NewRequestWithContext(ctx, "POST",
		s.serverURL+"/v1/work/"+workID+"/events", strings.NewReader(string(jsonBody)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("stream event: %w", err)
	}
	defer resp.Body.Close()
	return nil
}
