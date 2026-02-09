package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newDeployCmd(cfgPath *string) *cobra.Command {
	return newDeployCmdWithControl(cfgPath, nil)
}

func newDeployCmdWithControl(cfgPath *string, controlURL *string) *cobra.Command {
	var image, name, domain, serverName string
	var port int
	var envPairs []string

	cmd := &cobra.Command{
		Use:   "deploy [catalog-name]",
		Short: "Deploy a catalog service or custom image",
		RunE: func(cmd *cobra.Command, args []string) error {
			in := DeployInput{
				Image:      image,
				Name:       name,
				Port:       port,
				Domain:     domain,
				ServerName: serverName,
				Env:        parseEnvPairs(envPairs),
			}
			if image == "" {
				if len(args) != 1 {
					return fmt.Errorf("catalog service name required")
				}
				in.CatalogName = args[0]
			} else if len(args) == 1 {
				in.CatalogName = args[0]
			}

			if controlURL != nil && strings.TrimSpace(*controlURL) != "" {
				res, err := newControlClient(*controlURL).Deploy(context.Background(), *cfgPath, in)
				if err != nil {
					return err
				}
				if res.DNSMessage != "" {
					fmt.Fprintln(cmd.OutOrStdout(), res.DNSMessage)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Deployed %s to %s\n", res.ServiceName, res.ServerName)
				return nil
			}

			res, err := runDeployLocal(context.Background(), *cfgPath, in)
			if err != nil {
				return err
			}
			if res.DNSMessage != "" {
				fmt.Fprintln(cmd.OutOrStdout(), res.DNSMessage)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deployed %s to %s\n", res.ServiceName, res.ServerName)
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
