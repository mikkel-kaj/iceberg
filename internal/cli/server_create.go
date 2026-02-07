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

			pub, _, err := hetzner.GenerateSSHKeyPair()
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

			cfg.AddServer(config.ServerEntry{Name: name, TailscaleHostname: name, HetznerID: srv.ID, IP: srv.IPv4, AgentToken: agentToken, SSHKeyID: sshKey.ID, FirewallID: fw.ID})
			if err := cfg.Save(*cfgPath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created server %s (%s)\n", name, srv.IPv4)
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
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\n", s.Name, s.IP, s.HetznerID)
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
