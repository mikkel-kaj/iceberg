package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/auth"
	"github.com/mikkel-kaj/iceberg/internal/catalog"
	"github.com/mikkel-kaj/iceberg/internal/compose"
	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
	terraformprov "github.com/mikkel-kaj/iceberg/internal/infra/terraform"
)

const (
	provisionerTerraform = "terraform"
)

type ServerCreateResult struct {
	Name        string `json:"name"`
	PublicIP    string `json:"public_ip"`
	AgentIP     string `json:"agent_ip"`
	Provisioner string `json:"provisioner"`
}

type DeployInput struct {
	CatalogName string            `json:"catalog_name,omitempty"`
	Image       string            `json:"image,omitempty"`
	Name        string            `json:"name,omitempty"`
	Port        int               `json:"port,omitempty"`
	Domain      string            `json:"domain,omitempty"`
	ServerName  string            `json:"server_name,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
}

type DeployResult struct {
	ServiceName string `json:"service_name"`
	ServerName  string `json:"server_name"`
	DNSMessage  string `json:"dns_message,omitempty"`
}

type createdServer struct {
	Provisioner  string
	TerraformDir string
	ServerID     int64
	PublicIP     string
	SSHKeyID     int64
	FirewallID   int64
}

func runServerCreateLocal(ctx context.Context, cfgPath string) (*ServerCreateResult, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	if !terraformBinaryPresent() {
		return nil, fmt.Errorf("terraform is required for server provisioning; install terraform and retry")
	}
	name := nextServerName(cfg.Servers)

	pub, prv, err := hetzner.GenerateSSHKeyPair()
	if err != nil {
		return nil, err
	}
	agentToken, err := auth.GenerateToken()
	if err != nil {
		return nil, err
	}
	cloudInit, err := hetzner.RenderCloudInit(hetzner.CloudInitParams{
		Hostname:     name,
		SSHPublicKey: strings.TrimSpace(pub),
		TailscaleKey: cfg.TailscaleKey,
		AgentToken:   agentToken,
	})
	if err != nil {
		return nil, err
	}

	tf := newTerraformClient()
	tfDir := terraformStateDir(cfgPath, name)
	out, err := tf.CreateServer(ctx, terraformprov.CreateOptions{
		WorkDir:      tfDir,
		Name:         name,
		SSHPublicKey: pub,
		UserData:     cloudInit,
		Token:        cfg.HetznerToken,
	})
	if err != nil {
		return nil, err
	}
	created := createdServer{
		Provisioner:  provisionerTerraform,
		TerraformDir: tfDir,
		ServerID:     out.ServerID,
		PublicIP:     out.IPv4,
		SSHKeyID:     out.SSHKeyID,
		FirewallID:   out.FirewallID,
	}

	agentIP, err := bootstrapServer(ctx, name, created.PublicIP, agentToken, prv)
	if err != nil {
		_ = rollbackCreatedServer(ctx, cfg.HetznerToken, created)
		return nil, fmt.Errorf("bootstrap %s: %w", name, err)
	}

	cfg.AddServer(config.ServerEntry{
		Name:              name,
		TailscaleHostname: name,
		HetznerID:         created.ServerID,
		IP:                agentIP,
		PublicIP:          created.PublicIP,
		Provisioner:       created.Provisioner,
		TerraformDir:      created.TerraformDir,
		AgentToken:        agentToken,
		SSHKeyID:          created.SSHKeyID,
		FirewallID:        created.FirewallID,
	})
	if err := cfg.Save(cfgPath); err != nil {
		_ = rollbackCreatedServer(ctx, cfg.HetznerToken, created)
		return nil, err
	}

	return &ServerCreateResult{
		Name:        name,
		PublicIP:    created.PublicIP,
		AgentIP:     agentIP,
		Provisioner: created.Provisioner,
	}, nil
}

func runServerDestroyLocal(ctx context.Context, cfgPath, name string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	srv, err := cfg.GetServer(name)
	if err != nil {
		return err
	}
	if srv.Provisioner != provisionerTerraform {
		return fmt.Errorf("server %s is not terraform-managed (provisioner=%q); refusing API deletion", name, srv.Provisioner)
	}
	if !terraformBinaryPresent() {
		return fmt.Errorf("terraform is required to destroy %s (install terraform and retry)", name)
	}
	tfDir := srv.TerraformDir
	if tfDir == "" {
		tfDir = terraformStateDir(cfgPath, srv.Name)
	}
	if err := newTerraformClient().DestroyServer(ctx, terraformprov.DestroyOptions{
		WorkDir: tfDir,
		Token:   cfg.HetznerToken,
	}); err != nil {
		return err
	}
	cfg.RemoveServer(name)
	return cfg.Save(cfgPath)
}

func runDeployLocal(ctx context.Context, cfgPath string, in DeployInput) (*DeployResult, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	resolvedEnv, err := resolveEnvSecrets(ctx, in.Env)
	if err != nil {
		return nil, err
	}
	server, err := pickServer(cfg, in.ServerName)
	if err != nil {
		return nil, err
	}

	var spec compose.DeploySpec
	var serviceName string
	if in.Image != "" {
		if in.Name == "" {
			return nil, fmt.Errorf("--image requires --name")
		}
		if in.Port <= 0 {
			return nil, fmt.Errorf("--image requires --port")
		}
		spec = compose.DeploySpec{Services: []compose.ServiceSpec{{Name: in.Name, Image: in.Image, Port: in.Port, Env: resolvedEnv, Domain: in.Domain}}}
		serviceName = in.Name
	} else {
		if in.CatalogName == "" {
			return nil, fmt.Errorf("catalog service name required")
		}
		entry, err := catalog.Get(in.CatalogName)
		if err != nil {
			return nil, err
		}
		specPtr, err := entry.BuildSpec(domainOrDefault(in.Domain, cfg.DefaultDomain), resolvedEnv)
		if err != nil {
			return nil, err
		}
		spec = *specPtr
		serviceName = entry.Name
	}

	baseURL, err := agentBaseURL(server)
	if err != nil {
		return nil, err
	}
	ag := newAgentClient(baseURL, server.AgentToken)
	if err := ag.Deploy(ctx, serviceName, spec, domainOrDefault(in.Domain, cfg.DefaultDomain)); err != nil {
		return nil, err
	}

	deployedDomain := domainOrDefault(in.Domain, cfg.DefaultDomain)
	dnsTarget := publicDNSIP(server)
	var dnsMessage string
	if deployedDomain != "" {
		if cfg.DNS != nil && strings.EqualFold(cfg.DNS.Provider, "cloudflare") {
			if dnsTarget == "" {
				dnsMessage = fmt.Sprintf("Manual DNS required: create A record %s -> <server-public-ip>", deployedDomain)
			} else {
				cf := newCloudflareClient(cfg.DNS.CloudflareToken)
				zoneID, err := cf.GetZoneID(ctx, deployedDomain)
				if err != nil {
					dnsMessage = fmt.Sprintf("Cloudflare zone lookup failed (%v). Manual DNS required: create A record %s -> %s", err, deployedDomain, dnsTarget)
				} else if _, err := cf.CreateARecord(ctx, zoneID, deployedDomain, dnsTarget); err != nil {
					dnsMessage = fmt.Sprintf("Cloudflare DNS update failed (%v). Manual DNS required: create A record %s -> %s", err, deployedDomain, dnsTarget)
				}
			}
		} else if dnsTarget != "" {
			dnsMessage = fmt.Sprintf("Manual DNS required: create A record %s -> %s", deployedDomain, dnsTarget)
		}
	}
	return &DeployResult{ServiceName: serviceName, ServerName: server.Name, DNSMessage: dnsMessage}, nil
}

func rollbackCreatedServer(ctx context.Context, hetznerToken string, created createdServer) error {
	if created.TerraformDir == "" {
		return nil
	}
	return newTerraformClient().DestroyServer(ctx, terraformprov.DestroyOptions{
		WorkDir: created.TerraformDir,
		Token:   hetznerToken,
	})
}

func terraformStateDir(cfgPath, name string) string {
	return filepath.Join(filepath.Dir(cfgPath), "terraform", name)
}
