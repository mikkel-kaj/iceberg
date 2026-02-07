package catalog

import (
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

func plausibleEntry() Entry {
	return Entry{
		Name:        "plausible",
		Description: "Open-source web analytics",
		Port:        8000,
		HealthPath:  "/api/health",
		MinRAMMB:    400,
		MinDiskMB:   1500,
		BuildSpec: func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error) {
			secretKey := envOverrides["SECRET_KEY_BASE"]
			if secretKey == "" {
				generated, err := randomHex(32)
				if err != nil {
					return nil, err
				}
				secretKey = generated
			}
			baseURL := ""
			if domain != "" {
				baseURL = fmt.Sprintf("https://%s", domain)
			}
			env := mergedEnv(map[string]string{
				"BASE_URL":            baseURL,
				"SECRET_KEY_BASE":     secretKey,
				"DATABASE_URL":        "postgres://postgres:postgres@plausible-db:5432/plausible_db",
				"CLICKHOUSE_DATABASE": "plausible",
			}, envOverrides)
			return &compose.DeploySpec{Services: []compose.ServiceSpec{
				{Name: "plausible", Image: "plausible/analytics:latest", Port: 8000, Env: env, DependsOn: []string{"plausible-db", "plausible-events-db"}, HealthPath: "/api/health", Domain: domain},
				{Name: "plausible-db", Image: "postgres:16", Env: map[string]string{"POSTGRES_PASSWORD": "postgres", "POSTGRES_DB": "plausible_db"}, Volumes: []string{"./plausible-db:/var/lib/postgresql/data"}},
				{Name: "plausible-events-db", Image: "clickhouse/clickhouse-server:24", Volumes: []string{"./plausible-events-db:/var/lib/clickhouse"}},
			}}, nil
		},
	}
}
