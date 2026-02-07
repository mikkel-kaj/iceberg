package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newDestroyCmd(cfgPath *string) *cobra.Command {
	var domain, serverName string
	cmd := &cobra.Command{
		Use:   "destroy <service>",
		Short: "Destroy a deployed service",
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
			ag := newAgentClient("http://"+srv.IP+":8443", srv.AgentToken)
			if err := ag.Destroy(context.Background(), name); err != nil {
				return err
			}

			d := domainOrDefault(domain, cfg.DefaultDomain)
			if d != "" && cfg.DNS != nil && strings.EqualFold(cfg.DNS.Provider, "cloudflare") {
				cf := newCloudflareClient(cfg.DNS.CloudflareToken)
				if zoneID, err := cf.GetZoneID(context.Background(), d); err == nil {
					if rec, err := cf.FindRecord(context.Background(), zoneID, d); err == nil && rec != nil {
						_ = cf.DeleteRecord(context.Background(), zoneID, rec.ID)
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Destroyed service %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Domain to cleanup")
	cmd.Flags().StringVar(&serverName, "server", "", "Server name")
	return cmd
}
