package terraform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type CreateOptions struct {
	WorkDir      string
	Name         string
	SSHPublicKey string
	UserData     string
	ServerType   string
	Image        string
	Location     string
	Token        string
}

type CreateResult struct {
	ServerID   int64
	IPv4       string
	SSHKeyID   int64
	FirewallID int64
}

type DestroyOptions struct {
	WorkDir string
	Token   string
}

type commandRunner interface {
	Run(ctx context.Context, dir string, env map[string]string, name string, args ...string) ([]byte, error)
}

type execCommandRunner struct{}

func (r *execCommandRunner) Run(ctx context.Context, dir string, env map[string]string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	if len(env) > 0 {
		base := os.Environ()
		for key, value := range env {
			base = append(base, key+"="+value)
		}
		cmd.Env = base
	}
	return cmd.CombinedOutput()
}

type Provisioner struct {
	runner commandRunner
}

func New() *Provisioner {
	return &Provisioner{runner: &execCommandRunner{}}
}

func IsAvailable() bool {
	_, err := exec.LookPath("terraform")
	return err == nil
}

func (p *Provisioner) CreateServer(ctx context.Context, opts CreateOptions) (*CreateResult, error) {
	if p == nil || p.runner == nil {
		return nil, errors.New("terraform provisioner runner is nil")
	}
	opts = withCreateDefaults(opts)
	if err := validateCreateOptions(opts); err != nil {
		return nil, err
	}

	if err := os.MkdirAll(opts.WorkDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(opts.WorkDir, "main.tf"), []byte(mainTF), 0o600); err != nil {
		return nil, err
	}

	varsPath := filepath.Join(opts.WorkDir, "terraform.tfvars.json")
	rawVars, err := json.MarshalIndent(map[string]string{
		"name":           opts.Name,
		"server_type":    opts.ServerType,
		"image":          opts.Image,
		"location":       opts.Location,
		"ssh_public_key": strings.TrimSpace(opts.SSHPublicKey),
		"user_data":      opts.UserData,
	}, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(varsPath, rawVars, 0o600); err != nil {
		return nil, err
	}

	env := map[string]string{"HCLOUD_TOKEN": opts.Token}
	if _, err := p.runner.Run(ctx, opts.WorkDir, env, "terraform", "init", "-input=false", "-no-color"); err != nil {
		return nil, err
	}
	if _, err := p.runner.Run(ctx, opts.WorkDir, env, "terraform", "apply", "-auto-approve", "-input=false", "-no-color"); err != nil {
		return nil, err
	}
	out, err := p.runner.Run(ctx, opts.WorkDir, env, "terraform", "output", "-json", "-no-color")
	if err != nil {
		return nil, err
	}
	return parseOutput(out)
}

func (p *Provisioner) DestroyServer(ctx context.Context, opts DestroyOptions) error {
	if p == nil || p.runner == nil {
		return errors.New("terraform provisioner runner is nil")
	}
	if strings.TrimSpace(opts.WorkDir) == "" {
		return errors.New("work dir is required")
	}
	if strings.TrimSpace(opts.Token) == "" {
		return errors.New("hcloud token is required")
	}
	if _, err := os.Stat(opts.WorkDir); os.IsNotExist(err) {
		return nil
	}
	env := map[string]string{"HCLOUD_TOKEN": opts.Token}
	if _, err := p.runner.Run(ctx, opts.WorkDir, env, "terraform", "destroy", "-auto-approve", "-input=false", "-no-color"); err != nil {
		return err
	}
	return os.RemoveAll(opts.WorkDir)
}

func withCreateDefaults(opts CreateOptions) CreateOptions {
	if opts.ServerType == "" {
		opts.ServerType = "ccx13"
	}
	if opts.Image == "" {
		opts.Image = "ubuntu-24.04"
	}
	if opts.Location == "" {
		opts.Location = "nbg1"
	}
	return opts
}

func validateCreateOptions(opts CreateOptions) error {
	switch {
	case strings.TrimSpace(opts.WorkDir) == "":
		return errors.New("work dir is required")
	case strings.TrimSpace(opts.Name) == "":
		return errors.New("name is required")
	case strings.TrimSpace(opts.SSHPublicKey) == "":
		return errors.New("ssh public key is required")
	case strings.TrimSpace(opts.UserData) == "":
		return errors.New("user data is required")
	case strings.TrimSpace(opts.Token) == "":
		return errors.New("hcloud token is required")
	default:
		return nil
	}
}

func parseOutput(raw []byte) (*CreateResult, error) {
	type tfOutput struct {
		Value any `json:"value"`
	}
	var out map[string]tfOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse terraform output: %w", err)
	}
	serverID, err := toInt64(out["server_id"].Value)
	if err != nil {
		return nil, fmt.Errorf("parse server_id: %w", err)
	}
	sshKeyID, err := toInt64(out["ssh_key_id"].Value)
	if err != nil {
		return nil, fmt.Errorf("parse ssh_key_id: %w", err)
	}
	firewallID, err := toInt64(out["firewall_id"].Value)
	if err != nil {
		return nil, fmt.Errorf("parse firewall_id: %w", err)
	}
	ip, ok := out["server_ipv4"].Value.(string)
	if !ok || strings.TrimSpace(ip) == "" {
		return nil, errors.New("parse server_ipv4: missing value")
	}
	return &CreateResult{ServerID: serverID, IPv4: ip, SSHKeyID: sshKeyID, FirewallID: firewallID}, nil
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case float64:
		return int64(n), nil
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	case string:
		var j json.Number = json.Number(n)
		return j.Int64()
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", v)
	}
}

const mainTF = `
terraform {
  required_providers {
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.49"
    }
  }
}

provider "hcloud" {}

variable "name" { type = string }
variable "server_type" { type = string }
variable "image" { type = string }
variable "location" { type = string }
variable "ssh_public_key" { type = string }
variable "user_data" { type = string }

resource "hcloud_ssh_key" "iceberg" {
  name       = "${var.name}-ssh"
  public_key = var.ssh_public_key
}

resource "hcloud_firewall" "iceberg" {
  name = "${var.name}-fw"

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "22"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "udp"
    port       = "3478"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "udp"
    port       = "41641"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "80"
    source_ips = ["0.0.0.0/0", "::/0"]
  }

  rule {
    direction  = "in"
    protocol   = "tcp"
    port       = "443"
    source_ips = ["0.0.0.0/0", "::/0"]
  }
}

resource "hcloud_server" "iceberg" {
  name         = var.name
  server_type  = var.server_type
  image        = var.image
  location     = var.location
  user_data    = var.user_data
  ssh_keys     = [hcloud_ssh_key.iceberg.id]
  firewall_ids = [hcloud_firewall.iceberg.id]
}

output "server_id" {
  value = hcloud_server.iceberg.id
}

output "server_ipv4" {
  value = hcloud_server.iceberg.ipv4_address
}

output "ssh_key_id" {
  value = hcloud_ssh_key.iceberg.id
}

output "firewall_id" {
  value = hcloud_firewall.iceberg.id
}
`
