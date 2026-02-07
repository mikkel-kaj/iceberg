package catalog

import "github.com/mikkel-kaj/iceberg/internal/compose"

func supabaseEntry() Entry {
	return Entry{
		Name:        "supabase",
		Description: "Self-hosted backend platform",
		Port:        8000,
		HealthPath:  "/",
		MinRAMMB:    700,
		MinDiskMB:   3000,
		BuildSpec: func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error) {
			jwtSecret := envOverrides["JWT_SECRET"]
			anonKey := envOverrides["ANON_KEY"]
			serviceKey := envOverrides["SERVICE_KEY"]
			var err error
			if jwtSecret == "" {
				jwtSecret, err = randomHex(32)
				if err != nil {
					return nil, err
				}
			}
			if anonKey == "" {
				anonKey, err = randomHex(24)
				if err != nil {
					return nil, err
				}
			}
			if serviceKey == "" {
				serviceKey, err = randomHex(24)
				if err != nil {
					return nil, err
				}
			}
			env := mergedEnv(map[string]string{"JWT_SECRET": jwtSecret, "ANON_KEY": anonKey, "SERVICE_KEY": serviceKey}, envOverrides)
			return &compose.DeploySpec{Services: []compose.ServiceSpec{
				{Name: "supabase-db", Image: "supabase/postgres:15.8.1.039", Env: map[string]string{"POSTGRES_PASSWORD": "postgres"}, Volumes: []string{"./supabase-db:/var/lib/postgresql/data"}},
				{Name: "supabase-auth", Image: "supabase/gotrue:v2.158.1", Env: env, DependsOn: []string{"supabase-db"}},
				{Name: "supabase-rest", Image: "supabase/postgrest:v12.2.8", Env: env, DependsOn: []string{"supabase-db"}},
				{Name: "supabase-studio", Image: "supabase/studio:20250113-ce42139", Port: 8000, Env: env, DependsOn: []string{"supabase-auth", "supabase-rest"}, Domain: domain, HealthPath: "/"},
			}}, nil
		},
	}
}
