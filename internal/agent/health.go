package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/mikkel-kaj/iceberg/internal/compose"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusStopped   HealthStatus = "stopped"
	HealthStatusCrashLoop HealthStatus = "crash-looping"
)

type ServiceHealth struct {
	Name         string       `json:"name"`
	Status       HealthStatus `json:"status"`
	ContainerUp  bool         `json:"container_up"`
	HTTPHealthy  *bool        `json:"http_healthy,omitempty"`
	RestartCount int          `json:"restart_count"`
	MemoryUsedMB int64        `json:"memory_used_mb"`
	Domain       string       `json:"domain,omitempty"`
	CheckedAt    time.Time    `json:"checked_at"`
}

type ContainerChecker interface {
	IsRunning(containerName string) (bool, error)
	RestartCount(containerName string) (int, error)
	MemoryUsageMB(containerName string) (int64, error)
}

type HTTPChecker interface {
	Check(url string) (int, error)
}

type HealthMonitor struct {
	Interval         time.Duration
	ServicesDir      string
	ContainerChecker ContainerChecker
	HTTPChecker      HTTPChecker

	mu           sync.RWMutex
	statuses     map[string]*ServiceHealth
	lastRestarts map[string]int
}

func NewHealthMonitor(interval time.Duration, servicesDir string, cc ContainerChecker, hc HTTPChecker) *HealthMonitor {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &HealthMonitor{
		Interval:         interval,
		ServicesDir:      servicesDir,
		ContainerChecker: cc,
		HTTPChecker:      hc,
		statuses:         map[string]*ServiceHealth{},
		lastRestarts:     map[string]int{},
	}
}

func (h *HealthMonitor) Start(ctx context.Context) {
	h.tick()
	t := time.NewTicker(h.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.tick()
		}
	}
}

func (h *HealthMonitor) tick() {
	names, err := listServiceDirs(h.ServicesDir)
	if err != nil {
		return
	}
	for _, name := range names {
		meta, err := readServiceMeta(filepath.Join(h.ServicesDir, name, serviceMetaFile))
		if err != nil {
			continue
		}
		running, _ := h.ContainerChecker.IsRunning(name)
		restartCount, _ := h.ContainerChecker.RestartCount(name)
		memory, _ := h.ContainerChecker.MemoryUsageMB(name)
		status := HealthStatusStopped
		var httpHealthy *bool
		if running {
			status = HealthStatusHealthy
			if meta.HealthPath != "" && h.HTTPChecker != nil {
				code, err := h.HTTPChecker.Check(fmt.Sprintf("http://127.0.0.1:%d%s", firstPort(meta.Spec), meta.HealthPath))
				ok := err == nil && code >= 200 && code < 300
				httpHealthy = &ok
				if !ok {
					status = HealthStatusUnhealthy
				}
			}
		}
		if prev, ok := h.lastRestarts[name]; ok && restartCount-prev >= 3 {
			status = HealthStatusCrashLoop
		}
		h.lastRestarts[name] = restartCount
		h.mu.Lock()
		h.statuses[name] = &ServiceHealth{Name: name, Status: status, ContainerUp: running, HTTPHealthy: httpHealthy, RestartCount: restartCount, MemoryUsedMB: memory, Domain: meta.Domain, CheckedAt: time.Now().UTC()}
		h.mu.Unlock()
	}
}

func firstPort(spec compose.DeploySpec) int {
	if len(spec.Services) > 0 && spec.Services[0].Port > 0 {
		return spec.Services[0].Port
	}
	return 80
}

func (h *HealthMonitor) GetAll() []ServiceHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]ServiceHealth, 0, len(h.statuses))
	for _, s := range h.statuses {
		out = append(out, *s)
	}
	return out
}

func (h *HealthMonitor) Get(name string) (*ServiceHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	s, ok := h.statuses[name]
	if !ok {
		return nil, fmt.Errorf("service %q not found", name)
	}
	cpy := *s
	return &cpy, nil
}

func listServiceDirs(root string) ([]string, error) {
	dirs, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, d := range dirs {
		if d.IsDir() {
			out = append(out, d.Name())
		}
	}
	return out, nil
}
