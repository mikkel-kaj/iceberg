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
	provisionerTerraform = "terraform"
)

func newServerCmd(cfgPath *string) *cobra.Command {
	cmd := &cobra.Command{Use: "server", Short: "Manage servers"}
	cmd.AddCommand(newServerCreateCmd(cfgPath), newServerDestroyCmd(cfgPath), newServerListCmd(cfgPath))
	return cmd
}

func newServerCreateCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			if !terraformBinaryPresent() {
				return fmt.Errorf("terraform is required for server provisioning; install terraform and retry")
			}
			name := nextServerName(cfg.Servers)

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
			created := createdServer{
				Provisioner:  provisionerTerraform,
				TerraformDir: tfDir,
				ServerID:     out.ServerID,
				PublicIP:     out.IPv4,
				SSHKeyID:     out.SSHKeyID,
				FirewallID:   out.FirewallID,
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
}

type createdServer struct {
	Provisioner  string
	TerraformDir string
	ServerID     int64
	PublicIP     string
	SSHKeyID     int64
	FirewallID   int64
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
