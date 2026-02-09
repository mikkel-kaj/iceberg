package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

var bootstrapServer = bootstrapServerDefault

func newServerCmd(cfgPath *string) *cobra.Command {
	return newServerCmdWithControl(cfgPath, nil)
}

func newServerCmdWithControl(cfgPath *string, controlURL *string) *cobra.Command {
	cmd := &cobra.Command{Use: "server", Short: "Manage servers"}
	cmd.AddCommand(newServerCreateCmdWithControl(cfgPath, controlURL), newServerDestroyCmdWithControl(cfgPath, controlURL), newServerListCmd(cfgPath))
	return cmd
}

func newServerCreateCmd(cfgPath *string) *cobra.Command {
	return newServerCreateCmdWithControl(cfgPath, nil)
}

func newServerCreateCmdWithControl(cfgPath *string, controlURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "create",
		Short: "Create a new server",
		RunE: func(cmd *cobra.Command, args []string) error {
			if controlURL != nil && strings.TrimSpace(*controlURL) != "" {
				cc := newControlClient(*controlURL)
				res, err := cc.CreateServer(context.Background(), *cfgPath)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Created server %s (public %s, agent %s) via %s\n", res.Name, res.PublicIP, res.AgentIP, res.Provisioner)
				return nil
			}
			res, err := runServerCreateLocal(context.Background(), *cfgPath)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created server %s (public %s, agent %s) via %s\n", res.Name, res.PublicIP, res.AgentIP, res.Provisioner)
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
