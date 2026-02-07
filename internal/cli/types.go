package cli

import (
	"context"
	"io"

	"github.com/mikkel-kaj/iceberg/internal/agentclient"
	"github.com/mikkel-kaj/iceberg/internal/cloudflare"
	"github.com/mikkel-kaj/iceberg/internal/compose"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
)

type HetznerAPI interface {
	ValidateToken(ctx context.Context) error
	CreateSSHKey(ctx context.Context, name string, publicKey string) (*hetzner.SSHKey, error)
	DeleteSSHKey(ctx context.Context, id int64) error
	CreateFirewall(ctx context.Context, name string, rules []hetzner.FirewallRule) (*hetzner.Firewall, error)
	DeleteFirewall(ctx context.Context, id int64) error
	CreateServer(ctx context.Context, opts hetzner.CreateServerOpts) (*hetzner.Server, error)
	DeleteServer(ctx context.Context, id int64) error
}

type CloudflareAPI interface {
	ValidateToken(ctx context.Context) error
	GetZoneID(ctx context.Context, domain string) (string, error)
	CreateARecord(ctx context.Context, zoneID, subdomain, ip string) (*cloudflare.DNSRecord, error)
	DeleteRecord(ctx context.Context, zoneID, recordID string) error
	FindRecord(ctx context.Context, zoneID, subdomain string) (*cloudflare.DNSRecord, error)
}

type AgentAPI interface {
	Deploy(ctx context.Context, name string, spec compose.DeploySpec, domain string) error
	Destroy(ctx context.Context, name string) error
	Status(ctx context.Context) (*agentclient.StatusResponse, error)
	Logs(ctx context.Context, name string) (io.ReadCloser, error)
}
