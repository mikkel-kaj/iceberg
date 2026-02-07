package cli

import (
	"context"
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newServerDestroyCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <name>",
		Short: "Destroy a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			srv, err := cfg.GetServer(name)
			if err != nil {
				return err
			}
			h := newHetznerClient(cfg.HetznerToken)
			_ = h.DeleteServer(context.Background(), srv.HetznerID)
			if srv.SSHKeyID > 0 {
				_ = h.DeleteSSHKey(context.Background(), srv.SSHKeyID)
			}
			if srv.FirewallID > 0 {
				_ = h.DeleteFirewall(context.Background(), srv.FirewallID)
			}
			cfg.RemoveServer(name)
			if err := cfg.Save(*cfgPath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Destroyed server %s\n", name)
			return nil
		},
	}
}
