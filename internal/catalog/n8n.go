package catalog

import "github.com/mikkel-kaj/iceberg/internal/compose"

func n8nEntry() Entry {
	return Entry{
		Name:        "n8n",
		Description: "Workflow automation",
		Port:        5678,
		HealthPath:  "/healthz",
		MinRAMMB:    200,
		MinDiskMB:   1000,
		BuildSpec: func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error) {
			enc := envOverrides["N8N_ENCRYPTION_KEY"]
			if enc == "" {
				generated, err := randomHex(16)
				if err != nil {
					return nil, err
				}
				enc = generated
			}
			env := mergedEnv(map[string]string{"N8N_ENCRYPTION_KEY": enc}, envOverrides)
			return &compose.DeploySpec{Services: []compose.ServiceSpec{{
				Name:       "n8n",
				Image:      "docker.n8n.io/n8nio/n8n:latest",
				Port:       5678,
				Env:        env,
				Volumes:    []string{"./n8n-data:/home/node/.n8n"},
				HealthPath: "/healthz",
				Domain:     domain,
			}}}, nil
		},
	}
}
