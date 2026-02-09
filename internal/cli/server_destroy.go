package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newServerDestroyCmd(cfgPath *string) *cobra.Command {
	return newServerDestroyCmdWithControl(cfgPath, nil)
}

func newServerDestroyCmdWithControl(cfgPath *string, controlURL *string) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy <name>",
		Short: "Destroy a server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if controlURL != nil && strings.TrimSpace(*controlURL) != "" {
				if err := newControlClient(*controlURL).DestroyServer(context.Background(), *cfgPath, name); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Destroyed server %s\n", name)
				return nil
			}
			if err := runServerDestroyLocal(context.Background(), *cfgPath, name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Destroyed server %s\n", name)
			return nil
		},
	}
}
