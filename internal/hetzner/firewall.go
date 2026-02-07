package hetzner

import (
	"context"
	"net/http"
)

type Firewall struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type FirewallRule struct {
	Direction string   `json:"direction"`
	Protocol  string   `json:"protocol"`
	Port      string   `json:"port"`
	SourceIPs []string `json:"source_ips"`
}

func DefaultFirewallRules() []FirewallRule {
	sources := []string{"0.0.0.0/0", "::/0"}
	return []FirewallRule{
		{Direction: "in", Protocol: "tcp", Port: "22", SourceIPs: sources},
		{Direction: "in", Protocol: "udp", Port: "3478", SourceIPs: sources},
		{Direction: "in", Protocol: "udp", Port: "41641", SourceIPs: sources},
		{Direction: "in", Protocol: "tcp", Port: "80", SourceIPs: sources},
		{Direction: "in", Protocol: "tcp", Port: "443", SourceIPs: sources},
	}
}

func (c *Client) CreateFirewall(ctx context.Context, name string, rules []FirewallRule) (*Firewall, error) {
	var resp struct {
		Firewall Firewall `json:"firewall"`
	}
	if err := c.do(ctx, http.MethodPost, "/firewalls", map[string]any{"name": name, "rules": rules}, &resp); err != nil {
		return nil, err
	}
	return &resp.Firewall, nil
}

func (c *Client) DeleteFirewall(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/firewalls/"+itoa64(id), nil, nil)
}
