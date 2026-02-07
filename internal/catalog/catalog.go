package catalog

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

type Entry struct {
	Name        string
	Description string
	Port        int
	HealthPath  string
	MinRAMMB    int
	MinDiskMB   int
	EnvDefaults map[string]string
	EnvRequired []string
	BuildSpec   func(domain string, envOverrides map[string]string) (*compose.DeploySpec, error)
}

var entries = []Entry{
	uptimeKumaEntry(),
	plausibleEntry(),
	n8nEntry(),
	giteaEntry(),
	supabaseEntry(),
}

func All() []Entry {
	out := make([]Entry, len(entries))
	copy(out, entries)
	return out
}

func Get(name string) (*Entry, error) {
	for _, e := range entries {
		if e.Name == name {
			cpy := e
			return &cpy, nil
		}
	}
	return nil, fmt.Errorf("catalog entry %q not found", name)
}

func mergedEnv(defaults map[string]string, overrides map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range defaults {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
