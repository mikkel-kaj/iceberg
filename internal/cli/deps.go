package cli

import (
	"context"
	"os/exec"

	"github.com/mikkel-kaj/iceberg/internal/agentclient"
	"github.com/mikkel-kaj/iceberg/internal/cloudflare"
	"github.com/mikkel-kaj/iceberg/internal/hetzner"
	terraformprov "github.com/mikkel-kaj/iceberg/internal/infra/terraform"
	"github.com/mikkel-kaj/iceberg/internal/secrets"
)

var (
	newHetznerClient       = func(token string) HetznerAPI { return hetzner.NewClient(token) }
	newCloudflareClient    = func(token string) CloudflareAPI { return cloudflare.NewClient(token) }
	newAgentClient         = func(baseURL, token string) AgentAPI { return agentclient.NewClient(baseURL, token) }
	newControlClient       = func(baseURL string) ControlAPI { return newDefaultControlClient(baseURL) }
	newTerraformClient     = func() TerraformAPI { return terraformprov.New() }
	terraformBinaryPresent = func() bool {
		_, err := exec.LookPath("terraform")
		return err == nil
	}
	resolveBitwardenSecret = func(ctx context.Context, ref string) (string, error) {
		return secrets.ResolveBitwardenReference(ctx, ref)
	}
)
