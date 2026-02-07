package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type CaddyEntry struct {
	Domain       string
	UpstreamHost string
	UpstreamPort int
}

func (a *Agent) GenerateCaddyfile(entries []CaddyEntry) ([]byte, error) {
	if len(entries) == 0 {
		return []byte{}, nil
	}
	var b bytes.Buffer
	for _, e := range entries {
		if e.Domain == "" || e.UpstreamHost == "" || e.UpstreamPort <= 0 {
			continue
		}
		_, _ = fmt.Fprintf(&b, "%s {\n    reverse_proxy %s:%d\n}\n\n", e.Domain, e.UpstreamHost, e.UpstreamPort)
	}
	return b.Bytes(), nil
}

func (a *Agent) WriteCaddyfile(entries []CaddyEntry) error {
	if err := os.MkdirAll(a.CaddyDir, 0o755); err != nil {
		return err
	}
	body, err := a.GenerateCaddyfile(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.CaddyDir, "Caddyfile"), body, 0o644)
}

func (a *Agent) ReloadCaddy() error {
	return a.runInDir(context.Background(), a.CaddyDir, "docker", "compose", "-f", "caddy-compose.yaml", "restart", "caddy")
}

func (a *Agent) GetCaddyEntries() ([]CaddyEntry, error) {
	names, err := a.listServiceNames()
	if err != nil {
		return nil, err
	}
	out := make([]CaddyEntry, 0, len(names))
	for _, name := range names {
		meta, err := readServiceMeta(filepath.Join(a.ServicesDir, name, serviceMetaFile))
		if err != nil {
			continue
		}
		for _, svc := range meta.Spec.Services {
			if svc.Domain == "" || svc.Port <= 0 {
				continue
			}
			out = append(out, CaddyEntry{Domain: svc.Domain, UpstreamHost: svc.Name, UpstreamPort: svc.Port})
		}
	}
	return out, nil
}
