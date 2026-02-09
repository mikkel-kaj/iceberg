package cli

import (
	"context"
	"fmt"

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
			if srv.Provisioner != provisionerTerraform {
				return fmt.Errorf("server %s is not terraform-managed (provisioner=%q); refusing API deletion", name, srv.Provisioner)
			}
			if !terraformBinaryPresent() {
				return fmt.Errorf("terraform is required to destroy %s (install terraform and retry)", name)
			}
			tfDir := srv.TerraformDir
			if tfDir == "" {
				tfDir = terraformStateDir(*cfgPath, srv.Name)
			}
			if err := newTerraformClient().DestroyServer(context.Background(), terraformprov.DestroyOptions{
				WorkDir: tfDir,
				Token:   cfg.HetznerToken,
			}); err != nil {
				return err
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
