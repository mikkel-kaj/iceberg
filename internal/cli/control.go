package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	controlCreateServer  = runServerCreateLocal
	controlDestroyServer = runServerDestroyLocal
	controlDeployService = runDeployLocal
)

func newControlCmd(cfgPath *string, controlURL *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "control",
		Short: "Run or manage the local Iceberg control agent",
	}
	cmd.AddCommand(newControlServeCmd(cfgPath, controlURL))
	return cmd
}

func newControlServeCmd(cfgPath *string, controlURL *string) *cobra.Command {
	var listenAddr string
	defaultAddr := "127.0.0.1:19090"
	if controlURL != nil && strings.TrimSpace(*controlURL) != "" {
		if addr := hostPortFromURL(*controlURL); addr != "" {
			defaultAddr = addr
		}
	}
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run control agent HTTP API",
		RunE: func(cmd *cobra.Command, args []string) error {
			mux := http.NewServeMux()
			mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
				writeControlJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			})
			mux.HandleFunc("/v1/servers", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.NotFound(w, r)
					return
				}
				var req struct {
					ConfigPath string `json:"config_path"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeControlErr(w, http.StatusBadRequest, err)
					return
				}
				path := req.ConfigPath
				if strings.TrimSpace(path) == "" {
					path = *cfgPath
				}
				res, err := controlCreateServer(r.Context(), path)
				if err != nil {
					writeControlErr(w, http.StatusInternalServerError, err)
					return
				}
				writeControlJSON(w, http.StatusOK, res)
			})
			mux.HandleFunc("/v1/servers/", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					http.NotFound(w, r)
					return
				}
				name := strings.TrimPrefix(r.URL.Path, "/v1/servers/")
				if name == "" {
					writeControlErr(w, http.StatusBadRequest, fmt.Errorf("missing server name"))
					return
				}
				path := r.URL.Query().Get("config_path")
				if strings.TrimSpace(path) == "" {
					path = *cfgPath
				}
				if err := controlDestroyServer(r.Context(), path, name); err != nil {
					writeControlErr(w, http.StatusInternalServerError, err)
					return
				}
				writeControlJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			})
			mux.HandleFunc("/v1/deploy", func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					http.NotFound(w, r)
					return
				}
				var req struct {
					ConfigPath string      `json:"config_path"`
					Input      DeployInput `json:"input"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					writeControlErr(w, http.StatusBadRequest, err)
					return
				}
				path := req.ConfigPath
				if strings.TrimSpace(path) == "" {
					path = *cfgPath
				}
				res, err := controlDeployService(r.Context(), path, req.Input)
				if err != nil {
					writeControlErr(w, http.StatusInternalServerError, err)
					return
				}
				writeControlJSON(w, http.StatusOK, res)
			})

			srv := &http.Server{Addr: listenAddr, Handler: mux}
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			go func() {
				<-ctx.Done()
				_ = srv.Shutdown(context.Background())
			}()
			fmt.Fprintf(cmd.OutOrStdout(), "Control agent listening on %s\n", listenAddr)
			err := srv.ListenAndServe()
			if err == nil || err == http.ErrServerClosed {
				return nil
			}
			return err
		},
	}
	cmd.Flags().StringVar(&listenAddr, "listen", defaultAddr, "Control agent listen address")
	return cmd
}

func writeControlJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeControlErr(w http.ResponseWriter, status int, err error) {
	writeControlJSON(w, status, map[string]string{"error": err.Error()})
}

func hostPortFromURL(raw string) string {
	u, err := neturlParse(raw)
	if err != nil {
		return ""
	}
	host := u.Host
	if host == "" {
		return ""
	}
	if _, _, err := net.SplitHostPort(host); err == nil {
		return host
	}
	switch u.Scheme {
	case "https":
		return net.JoinHostPort(host, "443")
	default:
		return net.JoinHostPort(host, "80")
	}
}

var neturlParse = func(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
