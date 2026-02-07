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

const defaultCaddyCompose = `services:
  caddy:
    image: caddy:2
    container_name: iceberg-caddy
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
      - "443:443/udp"
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - ./caddy_data:/data
      - ./caddy_config:/config
    logging:
      driver: local
`

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
	if err := a.ensureCaddyCompose(); err != nil {
		return err
	}
	body, err := a.GenerateCaddyfile(entries)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(a.CaddyDir, "Caddyfile"), body, 0o644)
}

func (a *Agent) ReloadCaddy() error {
	if err := a.ensureCaddyCompose(); err != nil {
		return err
	}
	return a.runCompose(context.Background(), a.CaddyDir, "-f", "caddy-compose.yaml", "up", "-d", "caddy")
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
			out = append(out, CaddyEntry{Domain: svc.Domain, UpstreamHost: "host.docker.internal", UpstreamPort: svc.Port})
		}
	}
	return out, nil
}

func (a *Agent) ensureCaddyCompose() error {
	path := filepath.Join(a.CaddyDir, "caddy-compose.yaml")
	_, statErr := os.Stat(path)
	if statErr == nil {
		return nil
	}
	if !os.IsNotExist(statErr) {
		return statErr
	}
	if err := os.MkdirAll(filepath.Join(a.CaddyDir, "caddy_data"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(a.CaddyDir, "caddy_config"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(defaultCaddyCompose), 0o644)
}
