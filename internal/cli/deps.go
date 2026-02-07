package cli

import (
	"github.com/mikkel-kaj/iceberg/internal/agentclient"
	"github.com/mikkel-kaj/iceberg/internal/cloudflare"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
)

var (
	newHetznerClient    = func(token string) HetznerAPI { return hetzner.NewClient(token) }
	newCloudflareClient = func(token string) CloudflareAPI { return cloudflare.NewClient(token) }
	newAgentClient      = func(baseURL, token string) AgentAPI { return agentclient.NewClient(baseURL, token) }
)
