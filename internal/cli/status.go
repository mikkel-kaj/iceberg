package cli

import (
	"context"
	"fmt"
	"text/tabwriter"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newStatusCmd(cfgPath *string) *cobra.Command {
	var serverName string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show server and service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			srv, err := pickServer(cfg, serverName)
			if err != nil {
				return err
			}
			ag := newAgentClient("http://"+srv.IP+":8443", srv.AgentToken)
			st, err := ag.Status(context.Background())
			if err != nil {
				return err
			}
			if st.Metrics != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "CPU %.1f%% | RAM %d/%d MB | Disk %d/%d GB\n", st.Metrics.CPUPercent, st.Metrics.MemoryUsedMB, st.Metrics.MemoryTotalMB, st.Metrics.DiskUsedGB, st.Metrics.DiskTotalGB)
			}
			if len(st.Services) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No services deployed.")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "NAME\tSTATUS\tMEM(MB)\tDOMAIN")
			for _, svc := range st.Services {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", svc.Name, svc.Status, svc.MemoryUsedMB, svc.Domain)
			}
			_ = tw.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&serverName, "server", "", "Server name")
	return cmd
}
