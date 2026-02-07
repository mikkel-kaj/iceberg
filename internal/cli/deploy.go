package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/catalog"
	"github.com/mikkel-kaj/iceberg/internal/compose"
	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newDeployCmd(cfgPath *string) *cobra.Command {
	var image, name, domain, serverName string
	var port int
	var envPairs []string

	cmd := &cobra.Command{
		Use:   "deploy [catalog-name]",
		Short: "Deploy a catalog service or custom image",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*cfgPath)
			if err != nil {
				return err
			}
			server, err := pickServer(cfg, serverName)
			if err != nil {
				return err
			}

			var spec compose.DeploySpec
			var serviceName string
			if image != "" {
				if name == "" {
					return fmt.Errorf("--image requires --name")
				}
				if port <= 0 {
					return fmt.Errorf("--image requires --port")
				}
				spec = compose.DeploySpec{Services: []compose.ServiceSpec{{Name: name, Image: image, Port: port, Env: parseEnvPairs(envPairs), Domain: domain}}}
				serviceName = name
			} else {
				if len(args) != 1 {
					return fmt.Errorf("catalog service name required")
				}
				entry, err := catalog.Get(args[0])
				if err != nil {
					return err
				}
				specPtr, err := entry.BuildSpec(domainOrDefault(domain, cfg.DefaultDomain), parseEnvPairs(envPairs))
				if err != nil {
					return err
				}
				spec = *specPtr
				serviceName = entry.Name
			}

			baseURL, err := agentBaseURL(server)
			if err != nil {
				return err
			}
			ag := newAgentClient(baseURL, server.AgentToken)
			if err := ag.Deploy(context.Background(), serviceName, spec, domainOrDefault(domain, cfg.DefaultDomain)); err != nil {
				return err
			}

			deployedDomain := domainOrDefault(domain, cfg.DefaultDomain)
			dnsTarget := publicDNSIP(server)
			if deployedDomain != "" {
				if cfg.DNS != nil && strings.EqualFold(cfg.DNS.Provider, "cloudflare") {
					if dnsTarget == "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Manual DNS required: create A record %s -> <server-public-ip>\n", deployedDomain)
					} else {
						cf := newCloudflareClient(cfg.DNS.CloudflareToken)
						zoneID, err := cf.GetZoneID(context.Background(), deployedDomain)
						if err != nil {
							fmt.Fprintf(cmd.OutOrStdout(), "Cloudflare zone lookup failed (%v). Manual DNS required: create A record %s -> %s\n", err, deployedDomain, dnsTarget)
						} else if _, err := cf.CreateARecord(context.Background(), zoneID, deployedDomain, dnsTarget); err != nil {
							fmt.Fprintf(cmd.OutOrStdout(), "Cloudflare DNS update failed (%v). Manual DNS required: create A record %s -> %s\n", err, deployedDomain, dnsTarget)
						}
					}
				} else {
					if dnsTarget != "" {
						fmt.Fprintf(cmd.OutOrStdout(), "Manual DNS required: create A record %s -> %s\n", deployedDomain, dnsTarget)
					}
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deployed %s to %s\n", serviceName, server.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&image, "image", "", "Custom Docker image")
	cmd.Flags().StringVar(&name, "name", "", "Custom service name")
	cmd.Flags().IntVar(&port, "port", 0, "Public/internal port")
	cmd.Flags().StringVar(&domain, "domain", "", "Domain")
	cmd.Flags().StringSliceVar(&envPairs, "env", nil, "Environment variables KEY=VALUE")
	cmd.Flags().StringVar(&serverName, "server", "", "Server name")
	return cmd
}

func parseEnvPairs(pairs []string) map[string]string {
	out := map[string]string{}
	for _, p := range pairs {
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func domainOrDefault(domain, fallback string) string {
	if domain != "" {
		return domain
	}
	return fallback
}

func pickServer(cfg *config.Config, name string) (*config.ServerEntry, error) {
	if name != "" {
		return cfg.GetServer(name)
	}
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers configured")
	}
	return &cfg.Servers[0], nil
}
