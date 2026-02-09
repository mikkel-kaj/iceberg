package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/config"
	terraformprov "github.com/mikkel-kaj/iceberg/internal/infra/terraform"
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
			if strings.EqualFold(srv.Provisioner, provisionerTerraform) && srv.TerraformDir != "" {
				if !terraformBinaryPresent() {
					return fmt.Errorf("terraform is required to destroy %s (install terraform or switch provisioner manually)", name)
				}
				if err := newTerraformClient().DestroyServer(context.Background(), terraformprov.DestroyOptions{
					WorkDir: srv.TerraformDir,
					Token:   cfg.HetznerToken,
				}); err != nil {
					return err
				}
			} else {
				h := newHetznerClient(cfg.HetznerToken)
				_ = h.DeleteServer(context.Background(), srv.HetznerID)
				if srv.SSHKeyID > 0 {
					_ = h.DeleteSSHKey(context.Background(), srv.SSHKeyID)
				}
				if srv.FirewallID > 0 {
					_ = h.DeleteFirewall(context.Background(), srv.FirewallID)
				}
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
