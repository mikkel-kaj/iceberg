package agent

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

func (a *Agent) handleRestart(w http.ResponseWriter, r *http.Request) {
	name, err := serviceNameFromPath(r.URL.Path, "restart")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	dir := filepath.Join(a.ServicesDir, name)
	if _, err := os.Stat(dir); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("service %s not found", name))
		return
	}
	if err := a.runInDir(r.Context(), dir, "docker", "compose", "restart"); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}
