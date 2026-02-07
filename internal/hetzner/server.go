package hetzner

import (
	"context"
	"net/http"
)

type Server struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	IPv4   string
	IPv6   string
	Status string `json:"status"`
}

type CreateServerOpts struct {
	Name        string
	ServerType  string
	Image       string
	Location    string
	SSHKeyIDs   []int64
	FirewallIDs []int64
	UserData    string
}

func (c *Client) CreateServer(ctx context.Context, opts CreateServerOpts) (*Server, error) {
	if opts.ServerType == "" {
		opts.ServerType = "ccx13"
	}
	if opts.Image == "" {
		opts.Image = "ubuntu-24.04"
	}
	if opts.Location == "" {
		opts.Location = "nbg1"
	}
	payload := map[string]any{
		"name":        opts.Name,
		"server_type": opts.ServerType,
		"image":       opts.Image,
		"location":    opts.Location,
		"ssh_keys":    opts.SSHKeyIDs,
		"firewalls":   idsToFirewallRefs(opts.FirewallIDs),
		"user_data":   opts.UserData,
	}
	var resp struct {
		Server struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			PublicNet struct {
				IPv4 struct {
					IP string `json:"ip"`
				} `json:"ipv4"`
				IPv6 struct {
					IP string `json:"ip"`
				} `json:"ipv6"`
			} `json:"public_net"`
		} `json:"server"`
	}
	if err := c.do(ctx, http.MethodPost, "/servers", payload, &resp); err != nil {
		return nil, err
	}
	return &Server{
		ID:     resp.Server.ID,
		Name:   resp.Server.Name,
		IPv4:   resp.Server.PublicNet.IPv4.IP,
		IPv6:   resp.Server.PublicNet.IPv6.IP,
		Status: resp.Server.Status,
	}, nil
}

func (c *Client) GetServer(ctx context.Context, id int64) (*Server, error) {
	var resp struct {
		Server struct {
			ID        int64  `json:"id"`
			Name      string `json:"name"`
			Status    string `json:"status"`
			PublicNet struct {
				IPv4 struct {
					IP string `json:"ip"`
				} `json:"ipv4"`
				IPv6 struct {
					IP string `json:"ip"`
				} `json:"ipv6"`
			} `json:"public_net"`
		} `json:"server"`
	}
	if err := c.do(ctx, http.MethodGet, "/servers/"+itoa64(id), nil, &resp); err != nil {
		return nil, err
	}
	return &Server{ID: resp.Server.ID, Name: resp.Server.Name, IPv4: resp.Server.PublicNet.IPv4.IP, IPv6: resp.Server.PublicNet.IPv6.IP, Status: resp.Server.Status}, nil
}

func (c *Client) DeleteServer(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, "/servers/"+itoa64(id), nil, nil)
}

func idsToFirewallRefs(ids []int64) []map[string]int64 {
	out := make([]map[string]int64, 0, len(ids))
	for _, id := range ids {
		out = append(out, map[string]int64{"firewall": id})
	}
	return out
}
