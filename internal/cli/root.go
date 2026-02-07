package cli

import (
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	var cfgPath string
	root := &cobra.Command{
		Use:   "iceberg",
		Short: "Iceberg CLI",
	}
	root.PersistentFlags().StringVar(&cfgPath, "config", config.DefaultPath(), "Path to Iceberg config")

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newInitCmd(&cfgPath))
	root.AddCommand(newServerCmd(&cfgPath))
	root.AddCommand(newDeployCmd(&cfgPath))
	root.AddCommand(newStatusCmd(&cfgPath))
	root.AddCommand(newDestroyCmd(&cfgPath))
	root.AddCommand(newLogsCmd(&cfgPath))
	root.AddCommand(newCatalogCmd())

	return root
}

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		},
	}
}
