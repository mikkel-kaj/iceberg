package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/auth"
	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
	"github.com/spf13/cobra"
)

var bootstrapServer = bootstrapServerDefault

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
			h := newHetznerClient(cfg.HetznerToken)
			name := nextServerName(cfg.Servers)

			pub, prv, err := hetzner.GenerateSSHKeyPair()
			if err != nil {
				return err
			}
			agentToken, err := auth.GenerateToken()
			if err != nil {
				return err
			}

			sshKey, err := h.CreateSSHKey(context.Background(), name+"-ssh", pub)
			if err != nil {
				return err
			}
			fw, err := h.CreateFirewall(context.Background(), name+"-fw", hetzner.DefaultFirewallRules())
			if err != nil {
				_ = h.DeleteSSHKey(context.Background(), sshKey.ID)
				return err
			}

			cloudInit, err := hetzner.RenderCloudInit(hetzner.CloudInitParams{Hostname: name, SSHPublicKey: strings.TrimSpace(pub), TailscaleKey: cfg.TailscaleKey, AgentToken: agentToken})
			if err != nil {
				_ = h.DeleteSSHKey(context.Background(), sshKey.ID)
				_ = h.DeleteFirewall(context.Background(), fw.ID)
				return err
			}
			srv, err := h.CreateServer(context.Background(), hetzner.CreateServerOpts{Name: name, SSHKeyIDs: []int64{sshKey.ID}, FirewallIDs: []int64{fw.ID}, UserData: cloudInit})
			if err != nil {
				_ = h.DeleteSSHKey(context.Background(), sshKey.ID)
				_ = h.DeleteFirewall(context.Background(), fw.ID)
				return err
			}

			agentIP, err := bootstrapServer(context.Background(), name, srv.IPv4, agentToken, prv)
			if err != nil {
				_ = h.DeleteServer(context.Background(), srv.ID)
				_ = h.DeleteSSHKey(context.Background(), sshKey.ID)
				_ = h.DeleteFirewall(context.Background(), fw.ID)
				return fmt.Errorf("bootstrap %s: %w", name, err)
			}

			cfg.AddServer(config.ServerEntry{
				Name:              name,
				TailscaleHostname: name,
				HetznerID:         srv.ID,
				IP:                agentIP,
				PublicIP:          srv.IPv4,
				AgentToken:        agentToken,
				SSHKeyID:          sshKey.ID,
				FirewallID:        fw.ID,
			})
			if err := cfg.Save(*cfgPath); err != nil {
				_ = h.DeleteServer(context.Background(), srv.ID)
				_ = h.DeleteSSHKey(context.Background(), sshKey.ID)
				_ = h.DeleteFirewall(context.Background(), fw.ID)
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created server %s (public %s, agent %s)\n", name, srv.IPv4, agentIP)
			return nil
		},
	}
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
