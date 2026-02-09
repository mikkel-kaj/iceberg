package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/auth"
	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
	terraformprov "github.com/mikkel-kaj/iceberg/internal/infra/terraform"
	"github.com/spf13/cobra"
)

var bootstrapServer = bootstrapServerDefault

const (
	provisionerAuto      = "auto"
	provisionerTerraform = "terraform"
	provisionerAPI       = "api"
)

func newServerCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "server", Short: "Manage servers"}
	cmd.AddCommand(newServerCreateCmd(cfgPath), newServerDestroyCmd(cfgPath), newServerListCmd(cfgPath))
	return cmd
}

func newServerCreateCmd(cfgPath *string) *cobra.Command {
	var provisionerMode string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			name := nextServerName(cfg.Servers)
			selectedProvisioner, err := chooseProvisioner(provisionerMode)
			if err != nil {
				return err
			}

			pub, prv, err := hetzner.GenerateSSHKeyPair()
			if err != nil {
				return err
			}
			agentToken, err := auth.GenerateToken()
			if err != nil {
				return err
			}
			cloudInit, err := hetzner.RenderCloudInit(hetzner.CloudInitParams{
				Hostname:     name,
				SSHPublicKey: strings.TrimSpace(pub),
				TailscaleKey: cfg.TailscaleKey,
				AgentToken:   agentToken,
			})
			if err != nil {
				return err
			}

			var created createdServer
			switch selectedProvisioner {
			case provisionerTerraform:
				tf := newTerraformClient()
				tfDir := terraformStateDir(*cfgPath, name)
				out, err := tf.CreateServer(context.Background(), terraformprov.CreateOptions{
					WorkDir:      tfDir,
					Name:         name,
					SSHPublicKey: pub,
					UserData:     cloudInit,
					Token:        cfg.HetznerToken,
				})
				if err != nil {
					return err
				}
				created = createdServer{
					Provisioner:  provisionerTerraform,
					TerraformDir: tfDir,
					ServerID:     out.ServerID,
					PublicIP:     out.IPv4,
					SSHKeyID:     out.SSHKeyID,
					FirewallID:   out.FirewallID,
				}
			default:
				created, err = createServerViaAPI(context.Background(), cfg.HetznerToken, name, pub, cloudInit)
				if err != nil {
					return err
				}
			}

			agentIP, err := bootstrapServer(context.Background(), name, created.PublicIP, agentToken, prv)
			if err != nil {
				_ = rollbackCreatedServer(context.Background(), cfg.HetznerToken, created)
				return fmt.Errorf("bootstrap %s: %w", name, err)
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
			if err := cfg.Save(*cfgPath); err != nil {
				_ = rollbackCreatedServer(context.Background(), cfg.HetznerToken, created)
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created server %s (public %s, agent %s) via %s\n", name, created.PublicIP, agentIP, created.Provisioner)
			return nil
		},
	}
	cmd.Flags().StringVar(&provisionerMode, "provisioner", provisionerAuto, "Provisioner to use (auto|terraform|api)")
	return cmd
}

type createdServer struct {
	Provisioner  string
	TerraformDir string
	ServerID     int64
	PublicIP     string
	SSHKeyID     int64
	FirewallID   int64
}

func createServerViaAPI(ctx context.Context, hetznerToken, name, pub, cloudInit string) (createdServer, error) {
	h := newHetznerClient(hetznerToken)
	sshKey, err := h.CreateSSHKey(ctx, name+"-ssh", pub)
	if err != nil {
		return createdServer{}, err
	}
	fw, err := h.CreateFirewall(ctx, name+"-fw", hetzner.DefaultFirewallRules())
	if err != nil {
		_ = h.DeleteSSHKey(ctx, sshKey.ID)
		return createdServer{}, err
	}
	srv, err := h.CreateServer(ctx, hetzner.CreateServerOpts{
		Name:        name,
		SSHKeyIDs:   []int64{sshKey.ID},
		FirewallIDs: []int64{fw.ID},
		UserData:    cloudInit,
	})
	if err != nil {
		_ = h.DeleteSSHKey(ctx, sshKey.ID)
		_ = h.DeleteFirewall(ctx, fw.ID)
		return createdServer{}, err
	}
	return createdServer{
		Provisioner: provisionerAPI,
		ServerID:    srv.ID,
		PublicIP:    srv.IPv4,
		SSHKeyID:    sshKey.ID,
		FirewallID:  fw.ID,
	}, nil
}

func rollbackCreatedServer(ctx context.Context, hetznerToken string, created createdServer) error {
	switch created.Provisioner {
	case provisionerTerraform:
		if created.TerraformDir == "" {
			return nil
		}
		return newTerraformClient().DestroyServer(ctx, terraformprov.DestroyOptions{
			WorkDir: created.TerraformDir,
			Token:   hetznerToken,
		})
	default:
		h := newHetznerClient(hetznerToken)
		if created.ServerID > 0 {
			_ = h.DeleteServer(ctx, created.ServerID)
		}
		if created.SSHKeyID > 0 {
			_ = h.DeleteSSHKey(ctx, created.SSHKeyID)
		}
		if created.FirewallID > 0 {
			_ = h.DeleteFirewall(ctx, created.FirewallID)
		}
		return nil
	}
}

func chooseProvisioner(mode string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", provisionerAuto:
		if terraformBinaryPresent() {
			return provisionerTerraform, nil
		}
		return provisionerAPI, nil
	case provisionerTerraform:
		if !terraformBinaryPresent() {
			return "", fmt.Errorf("terraform provisioner selected but terraform binary is not installed")
		}
		return provisionerTerraform, nil
	case provisionerAPI:
		return provisionerAPI, nil
	default:
		return "", fmt.Errorf("invalid provisioner %q (allowed: auto, terraform, api)", mode)
	}
}

func terraformStateDir(cfgPath, name string) string {
	return filepath.Join(filepath.Dir(cfgPath), "terraform", name)
}

func newServerListCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			if len(cfg.Servers) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No servers configured.")
				return nil
			}
			for _, s := range cfg.Servers {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%d\n", s.Name, agentEndpointIP(&s), publicDNSIP(&s), s.HetznerID)
			}
			return nil
		},
	}
}

func nextServerName(existing []config.ServerEntry) string {
	max := 0
	for _, s := range existing {
		if strings.HasPrefix(s.Name, "iceberg-") {
			n, err := strconv.Atoi(strings.TrimPrefix(s.Name, "iceberg-"))
			if err == nil && n > max {
				max = n
			}
		}
	}
	return fmt.Sprintf("iceberg-%02d", max+1)
}
