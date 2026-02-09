package cli

import (
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func pickServer(cfg *config.Config, name string) (*config.ServerEntry, error) {
	if name != "" {
		return cfg.GetServer(name)
	}
	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("no servers configured")
	}
	return &cfg.Servers[0], nil
}
