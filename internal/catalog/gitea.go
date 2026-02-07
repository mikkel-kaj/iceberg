package catalog

import "github.com/mikkel-kaj/iceberg/internal/compose"

func giteaEntry() Entry {
	return Entry{
		Name:        "gitea",
		Description: "Self-hosted git platform",
		Port:        3000,
		HealthPath:  "/api/v1/version",
		MinRAMMB:    200,
		MinDiskMB:   2000,
		BuildSpec: func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error) {
			return &compose.DeploySpec{Services: []compose.ServiceSpec{
				{Name: "gitea", Image: "gitea/gitea:latest", Port: 3000, DependsOn: []string{"gitea-db"}, Volumes: []string{"./gitea:/data"}, Domain: domain, HealthPath: "/api/v1/version"},
				{Name: "gitea-db", Image: "postgres:16", Env: map[string]string{"POSTGRES_DB": "gitea", "POSTGRES_USER": "gitea", "POSTGRES_PASSWORD": "gitea"}, Volumes: []string{"./gitea-db:/var/lib/postgresql/data"}},
			}}, nil
		},
	}
}
