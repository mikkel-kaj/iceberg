package catalog

import "github.com/mikkel-kaj/iceberg/internal/compose"

func uptimeKumaEntry() Entry {
	return Entry{
		Name:        "uptime-kuma",
		Description: "Simple uptime monitoring dashboard",
		Port:        3001,
		HealthPath:  "/",
		MinRAMMB:    128,
		MinDiskMB:   500,
		BuildSpec: func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error) {
			return &compose.DeploySpec{Services: []compose.ServiceSpec{{
				Name:       "uptime-kuma",
				Image:      "louislam/uptime-kuma:1",
				Port:       3001,
				Volumes:    []string{"./data:/app/data"},
				HealthPath: "/",
				Domain:     domain,
			}}}, nil
		},
	}
}
