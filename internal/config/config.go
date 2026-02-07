package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HetznerToken  string        `yaml:"hetzner_token"`
	TailscaleKey  string        `yaml:"tailscale_auth_key"`
	DefaultDomain string        `yaml:"default_domain,omitempty"`
	DNS           *DNSConfig    `yaml:"dns,omitempty"`
	Servers       []ServerEntry `yaml:"servers,omitempty"`
}

type DNSConfig struct {
	Provider        string `yaml:"provider"`
	CloudflareToken string `yaml:"cloudflare_token"`
}

type ServerEntry struct {
	Name              string `yaml:"name"`
	TailscaleHostname string `yaml:"tailscale_hostname"`
	HetznerID         int64  `yaml:"hetzner_id"`
	IP                string `yaml:"ip"` // agent endpoint IP (prefer Tailscale)
	PublicIP          string `yaml:"public_ip,omitempty"`
	AgentToken        string `yaml:"agent_token"`
	SSHKeyID          int64  `yaml:"ssh_key_id,omitempty"`
	FirewallID        int64  `yaml:"firewall_id,omitempty"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".iceberg", "config.yaml")
	}
	return filepath.Join(home, ".iceberg", "config.yaml")
}

func Load(path string) (*Config, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &Config{}
	if err := yaml.Unmarshal(body, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Servers == nil {
		cfg.Servers = []ServerEntry{}
	}
	return cfg, nil
}

func (c *Config) Save(path string) error {
	if c == nil {
		return errors.New("config is nil")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	body, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(path, body, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *Config) AddServer(s ServerEntry) {
	c.Servers = append(c.Servers, s)
}

func (c *Config) RemoveServer(name string) {
	out := c.Servers[:0]
	for _, s := range c.Servers {
		if s.Name != name {
			out = append(out, s)
		}
	}
	c.Servers = out
}

func (c *Config) GetServer(name string) (*ServerEntry, error) {
	for i := range c.Servers {
		if c.Servers[i].Name == name {
			return &c.Servers[i], nil
		}
	}
	return nil, fmt.Errorf("server %q not found", name)
}
