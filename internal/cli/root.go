package cli

import (
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	var cfgPath, controlURL string
	root := &cobra.Command{
		Use:   "iceberg",
		Short: "Iceberg CLI",
	}
	root.PersistentFlags().StringVar(&cfgPath, "config", config.DefaultPath(), "Path to Iceberg config")
	root.PersistentFlags().StringVar(&controlURL, "control-url", "http://127.0.0.1:19090", "Control agent URL (empty to run command logic locally)")

	root.AddCommand(newVersionCmd(version))
	root.AddCommand(newInitCmd(&cfgPath))
	root.AddCommand(newServerCmdWithControl(&cfgPath, &controlURL))
	root.AddCommand(newDeployCmdWithControl(&cfgPath, &controlURL))
	root.AddCommand(newStatusCmd(&cfgPath))
	root.AddCommand(newDestroyCmd(&cfgPath))
	root.AddCommand(newLogsCmd(&cfgPath))
	root.AddCommand(newCatalogCmd())
	root.AddCommand(newControlCmd(&cfgPath, &controlURL))

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
