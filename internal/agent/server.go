package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mikkel-kaj/iceberg/internal/dashboard"
)

type Agent struct {
	Token       string
	ListenAddr  string
	ServicesDir string
	CaddyDir    string
	Router      *http.ServeMux

	srv     *http.Server
	mu      sync.Mutex
	runner  CommandRunner
	loger   LogStreamer
	monitor *HealthMonitor
}

func New(token, listenAddr, servicesDir, caddyDir string) *Agent {
	a := &Agent{
		Token:       token,
		ListenAddr:  listenAddr,
		ServicesDir: servicesDir,
		CaddyDir:    caddyDir,
		Router:      http.NewServeMux(),
		runner:      &execRunner{},
		loger:       &dockerLogStreamer{},
	}
	a.registerRoutes()
	return a
}

func (a *Agent) registerRoutes() {
	a.Router.Handle("/", http.HandlerFunc(a.handleRoot))
	a.Router.Handle("/health", http.HandlerFunc(a.handleHealth))
	a.Router.Handle("/status", http.HandlerFunc(a.handleStatus))
	a.Router.Handle("/metrics", http.HandlerFunc(a.handleMetrics))
	a.Router.Handle("/services", http.HandlerFunc(a.handleListServices))
	a.Router.Handle("/services/", http.HandlerFunc(a.handleServiceRoutes))

	static, _ := fsSub(dashboard.Static, "static")
	a.Router.Handle("/dashboard/", http.StripPrefix("/dashboard/", http.FileServer(http.FS(static))))
}

func (a *Agent) Start(ctx context.Context) error {
	if err := os.MkdirAll(a.ServicesDir, 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(a.CaddyDir, 0o755); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", a.ListenAddr)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.srv = &http.Server{Handler: a.Router}
	a.mu.Unlock()
	go func() {
		<-ctx.Done()
		_ = a.Stop(context.Background())
	}()
	err = a.srv.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *Agent) Stop(ctx context.Context) error {
	a.mu.Lock()
	if a.srv == nil {
		a.mu.Unlock()
		return nil
	}
	srv := a.srv
	a.srv = nil
	a.mu.Unlock()
	return srv.Shutdown(ctx)
}

func (a *Agent) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/dashboard/", http.StatusFound)
}

func (a *Agent) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *Agent) handleListServices(w http.ResponseWriter, _ *http.Request) {
	names, err := a.listServiceNames()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, names)
}

func (a *Agent) handleServiceRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/services/")
	path = strings.Trim(path, "/")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	parts := strings.Split(path, "/")
	name := parts[0]
	if len(parts) == 1 {
		if r.Method == http.MethodGet {
			a.handleServiceDetail(w, r, name)
			return
		}
		http.NotFound(w, r)
		return
	}
	action := parts[1]
	switch {
	case action == "deploy" && r.Method == http.MethodPost:
		if !a.requireAuth(w, r) {
			return
		}
		a.handleDeploy(w, r)
	case action == "destroy" && r.Method == http.MethodDelete:
		if !a.requireAuth(w, r) {
			return
		}
		a.handleDestroy(w, r)
	case action == "logs" && r.Method == http.MethodGet:
		a.handleLogs(w, r)
	case action == "restart" && r.Method == http.MethodPost:
		if !a.requireAuth(w, r) {
			return
		}
		a.handleRestart(w, r)
	default:
		http.NotFound(w, r)
	}
	_ = name
}

func (a *Agent) handleStatus(w http.ResponseWriter, r *http.Request) {
	metrics, _ := CollectMetrics()
	services, err := a.readAllServiceHealth()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	resp := StatusResponse{Metrics: metrics, Services: services}
	writeJSON(w, http.StatusOK, resp)
}

func (a *Agent) handleServiceDetail(w http.ResponseWriter, r *http.Request, name string) {
	if a.monitor != nil {
		if h, err := a.monitor.Get(name); err == nil {
			writeJSON(w, http.StatusOK, h)
			return
		}
	}
	meta, err := readServiceMeta(filepath.Join(a.ServicesDir, name, serviceMetaFile))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	status := ServiceHealth{Name: name, Status: HealthStatusStopped, ContainerUp: false, Domain: meta.Domain}
	writeJSON(w, http.StatusOK, status)
}

func (a *Agent) listServiceNames() ([]string, error) {
	entries, err := os.ReadDir(a.ServicesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

func (a *Agent) readAllServiceHealth() ([]ServiceHealth, error) {
	if a.monitor != nil {
		return a.monitor.GetAll(), nil
	}
	names, err := a.listServiceNames()
	if err != nil {
		return nil, err
	}
	out := make([]ServiceHealth, 0, len(names))
	for _, name := range names {
		meta, err := readServiceMeta(filepath.Join(a.ServicesDir, name, serviceMetaFile))
		if err != nil {
			continue
		}
		out = append(out, ServiceHealth{Name: name, Status: HealthStatusStopped, ContainerUp: false, Domain: meta.Domain})
	}
	return out, nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
