package worker

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds the worker's runtime configuration, loaded from worker.toml.
type Config struct {
	Name            string   `toml:"name"`
	ServerURL       string   `toml:"server_url"`
	AuthToken       string   `toml:"auth_token"`
	AuthTokenEnv    string   `toml:"auth_token_env"`
	MaxConcurrent   int      `toml:"max_concurrent"`
	MemoryLimitMB   int      `toml:"memory_limit_mb"`
	TimeoutMinutes  int      `toml:"timeout_minutes"`
	Sandbox         string   `toml:"sandbox"`
	DockerImage     string   `toml:"docker_image"`
	AllowedHosts    []string `toml:"allowed_hosts"`
	HealthInterval  int      `toml:"health_interval_sec"`
	PollTimeout     int      `toml:"poll_timeout_sec"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		MaxConcurrent:  4,
		MemoryLimitMB:  4096,
		TimeoutMinutes: 30,
		Sandbox:        "docker",
		DockerImage:    "reasonix/sandbox:latest",
		HealthInterval: 30,
		PollTimeout:    30,
	}
}

// Load reads and validates a worker.toml file. Returns an error if required
// fields are missing.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()
	if _, err := toml.DecodeFile(path, &cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load worker config: %w", err)
	}
	if cfg.Name == "" {
		host, _ := os.Hostname()
		cfg.Name = host
	}
	if cfg.ServerURL == "" {
		cfg.ServerURL = "http://localhost:9090"
	}
	if cfg.AuthToken == "" && cfg.AuthTokenEnv != "" {
		cfg.AuthToken = os.Getenv(cfg.AuthTokenEnv)
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 4
	}
	if cfg.MemoryLimitMB <= 0 {
		cfg.MemoryLimitMB = 4096
	}
	if cfg.TimeoutMinutes <= 0 {
		cfg.TimeoutMinutes = 30
	}
	if cfg.HealthInterval <= 0 {
		cfg.HealthInterval = 30
	}
	if cfg.PollTimeout <= 0 {
		cfg.PollTimeout = 30
	}
	return &cfg, cfg.Validate()
}

// Validate checks that required fields are set.
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("worker name is required")
	}
	if c.ServerURL == "" {
		return fmt.Errorf("server_url is required")
	}
	if c.AuthToken == "" {
		return fmt.Errorf("auth_token or auth_token_env is required")
	}
	return nil
}
