package cli

import (
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/catalog"
	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "catalog",
		Short: "List catalog services",
		Run: func(cmd *cobra.Command, args []string) {
			for _, entry := range catalog.All() {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\n", entry.Name, entry.Description)
			}
		},
	}
}
