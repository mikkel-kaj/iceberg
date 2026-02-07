package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

const (
	composeFileName = "docker-compose.yaml"
	serviceMetaFile = "service.json"
)

type serviceMeta struct {
	Name       string             `json:"name"`
	Domain     string             `json:"domain,omitempty"`
	HealthPath string             `json:"health_path,omitempty"`
	Spec       compose.DeploySpec `json:"spec"`
}

func (a *Agent) handleDeploy(w http.ResponseWriter, r *http.Request) {
	name, err := serviceNameFromPath(r.URL.Path, "deploy")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var spec compose.DeploySpec
	if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if len(spec.Services) == 0 {
		writeError(w, http.StatusBadRequest, errors.New("services is required"))
		return
	}
	for _, svc := range spec.Services {
		if svc.Image == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("service %s missing image", svc.Name))
			return
		}
	}
	if err := a.writeService(name, spec); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := a.runner.Run(r.Context(), filepath.Join(a.ServicesDir, name), "docker", "compose", "up", "-d"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.syncCaddy(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "deployed"})
}

func (a *Agent) writeService(name string, spec compose.DeploySpec) error {
	dir := filepath.Join(a.ServicesDir, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	composeBody, err := compose.GenerateComposeFile(spec)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, composeFileName), composeBody, 0o644); err != nil {
		return err
	}
	meta := serviceMeta{Name: name, Spec: spec}
	for _, svc := range spec.Services {
		if svc.Domain != "" && meta.Domain == "" {
			meta.Domain = svc.Domain
		}
		if svc.HealthPath != "" && meta.HealthPath == "" {
			meta.HealthPath = svc.HealthPath
		}
	}
	raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, serviceMetaFile), raw, 0o644)
}

func (a *Agent) handleDestroy(w http.ResponseWriter, r *http.Request) {
	name, err := serviceNameFromPath(r.URL.Path, "destroy")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	dir := filepath.Join(a.ServicesDir, name)
	if _, err := os.Stat(dir); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("service %s not found", name))
		return
	}
	_ = a.runner.Run(r.Context(), dir, "docker", "compose", "down", "--remove-orphans")
	if err := os.RemoveAll(dir); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = a.syncCaddy(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{"status": "destroyed"})
}

func readServiceMeta(path string) (*serviceMeta, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta serviceMeta
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func serviceNameFromPath(path, action string) (string, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 3 || parts[0] != "services" || parts[2] != action {
		return "", fmt.Errorf("invalid path %s", path)
	}
	name := parts[1]
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, string(filepath.Separator)) {
		return "", fmt.Errorf("invalid service name %q", name)
	}
	return name, nil
}

func (a *Agent) syncCaddy(ctx context.Context) error {
	entries, err := a.GetCaddyEntries()
	if err != nil {
		return err
	}
	if err := a.WriteCaddyfile(entries); err != nil {
		return err
	}
	if err := a.ReloadCaddy(); err != nil {
		return err
	}
	_ = ctx
	return nil
}

func (a *Agent) runInDir(ctx context.Context, dir string, args ...string) error {
	if len(args) == 0 {
		return errors.New("no command")
	}
	return a.runner.Run(ctx, dir, args[0], args[1:]...)
}
