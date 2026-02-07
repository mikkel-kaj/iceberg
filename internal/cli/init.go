package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd(cfgPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize Iceberg CLI config",
		RunE: func(cmd *cobra.Command, args []string) error {
			in := bufio.NewReader(cmd.InOrStdin())
			prompt := func(label string) (string, error) {
				fmt.Fprint(cmd.OutOrStdout(), label)
				line, err := in.ReadString('\n')
				if err != nil {
					return "", err
				}
				return strings.TrimSpace(line), nil
			}

			hToken, err := prompt("Hetzner token: ")
			if err != nil {
				return err
			}
			tKey, err := prompt("Tailscale auth key: ")
			if err != nil {
				return err
			}
			domain, err := prompt("Default domain (optional): ")
			if err != nil {
				return err
			}
			var cfToken string
			if domain != "" {
				cfToken, err = prompt("Cloudflare token: ")
				if err != nil {
					return err
				}
			}

			h := newHetznerClient(hToken)
			if err := h.ValidateToken(context.Background()); err != nil {
				return fmt.Errorf("invalid Hetzner token: %w", err)
			}

			cfg := &config.Config{HetznerToken: hToken, TailscaleKey: tKey, DefaultDomain: domain}
			if domain != "" && cfToken != "" {
				cf := newCloudflareClient(cfToken)
				if err := cf.ValidateToken(context.Background()); err != nil {
					return fmt.Errorf("invalid Cloudflare token: %w", err)
				}
				cfg.DNS = &config.DNSConfig{Provider: "cloudflare", CloudflareToken: cfToken}
			}
			if err := cfg.Save(*cfgPath); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Config written to %s\n", *cfgPath)
			return nil
		},
	}
}
