package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// RemoteEntry defines a remote worker in reasonix.toml [[remotes]] section.
type RemoteEntry struct {
	Name         string   `toml:"name"`
	URL          string   `toml:"url"`
	AuthToken    string   `toml:"auth_token"`
	AuthTokenEnv string   `toml:"auth_token_env"`
	MaxConcurrent int     `toml:"max_concurrent"`
	PreferFor    []string `toml:"prefer_for"`
}

// Validate checks the remote entry for required fields and valid values.
func (r *RemoteEntry) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("remote name is required")
	}
	if r.URL == "" {
		return fmt.Errorf("remote url is required for %q", r.Name)
	}
	if _, err := url.Parse(r.URL); err != nil {
		return fmt.Errorf("invalid remote url for %q: %w", r.Name, err)
	}
	// Resolve auth token from env var
	if r.AuthToken == "" && r.AuthTokenEnv != "" {
		// Token resolution happens at runtime in boot.go via ResolveRemoteToken
	}
	if r.MaxConcurrent <= 0 {
		r.MaxConcurrent = 4
	}
	return nil
}

// ResolveRemoteToken resolves the auth token for a remote entry. If auth_token
// is set directly, it's used. Otherwise auth_token_env is resolved.
func ResolveRemoteToken(r *RemoteEntry) string {
	if strings.TrimSpace(r.AuthToken) != "" {
		return r.AuthToken
	}
	if r.AuthTokenEnv != "" {
		return strings.TrimSpace(resolveEnvVar(r.AuthTokenEnv))
	}
	return ""
}

// ResolveRemote returns the named remote entry, or nil if not found.
func (c *Config) ResolveRemote(name string) *RemoteEntry {
	for i := range c.Remotes {
		if c.Remotes[i].Name == name {
			return &c.Remotes[i]
		}
	}
	return nil
}
