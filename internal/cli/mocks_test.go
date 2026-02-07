package cli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/agent"
	"github.com/mikkel-kaj/iceberg/internal/agentclient"
	"github.com/mikkel-kaj/iceberg/internal/cloudflare"
	"github.com/mikkel-kaj/iceberg/internal/compose"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
)

type mockHetzner struct {
	validateErr error
	createSSH   *hetzner.SSHKey
	createFW    *hetzner.Firewall
	createSrv   *hetzner.Server
	calls       []string
}

func (m *mockHetzner) ValidateToken(ctx context.Context) error {
	m.calls = append(m.calls, "validate")
	return m.validateErr
}
func (m *mockHetzner) CreateSSHKey(ctx context.Context, name, public string) (*hetzner.SSHKey, error) {
	m.calls = append(m.calls, "create_ssh")
	if m.createSSH == nil {
		m.createSSH = &hetzner.SSHKey{ID: 1, Name: name}
	}
	return m.createSSH, nil
}
func (m *mockHetzner) DeleteSSHKey(ctx context.Context, id int64) error {
	m.calls = append(m.calls, "delete_ssh")
	return nil
}
func (m *mockHetzner) CreateFirewall(ctx context.Context, name string, rules []hetzner.FirewallRule) (*hetzner.Firewall, error) {
	m.calls = append(m.calls, "create_fw")
	if m.createFW == nil {
		m.createFW = &hetzner.Firewall{ID: 2, Name: name}
	}
	return m.createFW, nil
}
func (m *mockHetzner) DeleteFirewall(ctx context.Context, id int64) error {
	m.calls = append(m.calls, "delete_fw")
	return nil
}
func (m *mockHetzner) CreateServer(ctx context.Context, opts hetzner.CreateServerOpts) (*hetzner.Server, error) {
	m.calls = append(m.calls, "create_server")
	if m.createSrv == nil {
		m.createSrv = &hetzner.Server{ID: 3, Name: opts.Name, IPv4: "100.64.0.1"}
	}
	return m.createSrv, nil
}
func (m *mockHetzner) DeleteServer(ctx context.Context, id int64) error {
	m.calls = append(m.calls, "delete_server")
	return nil
}

type mockCloudflare struct {
	validateErr error
	zoneID      string
	record      *cloudflare.DNSRecord
	calls       []string
}

func (m *mockCloudflare) ValidateToken(ctx context.Context) error {
	m.calls = append(m.calls, "validate")
	return m.validateErr
}
func (m *mockCloudflare) GetZoneID(ctx context.Context, domain string) (string, error) {
	m.calls = append(m.calls, "zone")
	if m.zoneID == "" {
		return "", errors.New("no zone")
	}
	return m.zoneID, nil
}
func (m *mockCloudflare) CreateARecord(ctx context.Context, zoneID, subdomain, ip string) (*cloudflare.DNSRecord, error) {
	m.calls = append(m.calls, "create")
	if m.record == nil {
		m.record = &cloudflare.DNSRecord{ID: "r1", Name: subdomain, Type: "A", Content: ip}
	}
	return m.record, nil
}
func (m *mockCloudflare) DeleteRecord(ctx context.Context, zoneID, recordID string) error {
	m.calls = append(m.calls, "delete")
	return nil
}
func (m *mockCloudflare) FindRecord(ctx context.Context, zoneID, subdomain string) (*cloudflare.DNSRecord, error) {
	m.calls = append(m.calls, "find")
	if m.record == nil {
		return nil, nil
	}
	return m.record, nil
}

type mockAgent struct {
	deployErr    error
	destroyErr   error
	statusResp   *agentclient.StatusResponse
	logsBody     string
	deployName   string
	deploySpec   compose.DeploySpec
	deployDomain string
	calls        []string
}

func (m *mockAgent) Deploy(ctx context.Context, name string, spec compose.DeploySpec, domain string) error {
	m.calls = append(m.calls, "deploy")
	m.deployName = name
	m.deploySpec = spec
	m.deployDomain = domain
	return m.deployErr
}
func (m *mockAgent) Destroy(ctx context.Context, name string) error {
	m.calls = append(m.calls, "destroy")
	return m.destroyErr
}
func (m *mockAgent) Status(ctx context.Context) (*agentclient.StatusResponse, error) {
	m.calls = append(m.calls, "status")
	if m.statusResp == nil {
		m.statusResp = &agentclient.StatusResponse{Metrics: &agent.ServerMetrics{CPUPercent: 1}, Services: []agent.ServiceHealth{}}
	}
	return m.statusResp, nil
}
func (m *mockAgent) Logs(ctx context.Context, name string) (io.ReadCloser, error) {
	m.calls = append(m.calls, "logs")
	if m.logsBody == "" {
		m.logsBody = "line\n"
	}
	return io.NopCloser(strings.NewReader(m.logsBody)), nil
}
