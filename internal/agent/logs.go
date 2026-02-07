package agent

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func (a *Agent) handleLogs(w http.ResponseWriter, r *http.Request) {
	name, err := serviceNameFromPath(r.URL.Path, "logs")
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	dir := filepath.Join(a.ServicesDir, name)
	if _, err := os.Stat(dir); err != nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("service %s not found", name))
		return
	}
	tail := 100
	if s := r.URL.Query().Get("tail"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			tail = n
		}
	}
	follow := r.URL.Query().Get("follow") == "true"
	reader, err := a.loger.Read(r.Context(), dir, name, tail, follow)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	defer reader.Close()

	if !follow {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = io.Copy(w, reader)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		_, _ = fmt.Fprintf(w, "data: %s\n\n", scanner.Text())
		flusher.Flush()
	}
}
