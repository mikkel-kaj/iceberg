package cli

import (
	"context"
	"io"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newLogsCmd(cfgPath *string) *cobra.Command {
	var serverName string
	cmd := &cobra.Command{
		Use:   "logs <service>",
		Short: "Stream service logs",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			srv, err := pickServer(cfg, serverName)
			if err != nil {
				return err
			}
			baseURL, err := agentBaseURL(srv)
			if err != nil {
				return err
			}
			ag := newAgentClient(baseURL, srv.AgentToken)
			rc, err := ag.Logs(context.Background(), name)
			if err != nil {
				return err
			}
			defer rc.Close()
			_, err = io.Copy(cmd.OutOrStdout(), rc)
			return err
		},
	}
	cmd.Flags().StringVar(&serverName, "server", "", "Server name")
	return cmd
}
