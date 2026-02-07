package agent

import (
	"bufio"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type ServerMetrics struct {
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskTotalGB   int64   `json:"disk_total_gb"`
	DiskUsedGB    int64   `json:"disk_used_gb"`
	DiskPercent   float64 `json:"disk_percent"`
	UptimeSeconds int64   `json:"uptime_seconds"`
}

func (a *Agent) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	m, err := CollectMetrics()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

func CollectMetrics() (*ServerMetrics, error) {
	if runtime.GOOS != "linux" {
		return &ServerMetrics{CPUPercent: 0, MemoryTotalMB: 1024, MemoryUsedMB: 256, MemoryPercent: 25, DiskTotalGB: 50, DiskUsedGB: 10, DiskPercent: 20, UptimeSeconds: 1}, nil
	}
	cpu, err := readCPUPercent()
	if err != nil {
		cpu = 0
	}
	memTotal, memAvail, err := readMeminfo()
	if err != nil {
		return nil, err
	}
	memUsed := memTotal - memAvail
	memTotalMB := memTotal / (1024 * 1024)
	memUsedMB := memUsed / (1024 * 1024)
	memPct := 0.0
	if memTotal > 0 {
		memPct = float64(memUsed) / float64(memTotal) * 100
	}
	diskTotalGB, diskUsedGB, diskPct := readDisk("/")
	uptime, _ := readUptime()
	return &ServerMetrics{CPUPercent: cpu, MemoryTotalMB: memTotalMB, MemoryUsedMB: memUsedMB, MemoryPercent: memPct, DiskTotalGB: diskTotalGB, DiskUsedGB: diskUsedGB, DiskPercent: diskPct, UptimeSeconds: uptime}, nil
}

func readCPUPercent() (float64, error) {
	idle1, total1, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	time.Sleep(100 * time.Millisecond)
	idle2, total2, err := readCPUStat()
	if err != nil {
		return 0, err
	}
	dIdle := float64(idle2 - idle1)
	dTotal := float64(total2 - total1)
	if dTotal <= 0 {
		return 0, nil
	}
	pct := (1 - dIdle/dTotal) * 100
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct, nil
}

func readCPUStat() (idle, total uint64, err error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	if !s.Scan() {
		return 0, 0, errors.New("/proc/stat empty")
	}
	fields := strings.Fields(s.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0, 0, fmt.Errorf("unexpected cpu line: %q", s.Text())
	}
	for i := 1; i < len(fields); i++ {
		v, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		total += v
		if i == 4 { // idle
			idle = v
		}
	}
	return idle, total, nil
}

func readMeminfo() (totalBytes, availableBytes int64, err error) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var totalKB, availKB int64
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			_, _ = fmt.Sscanf(line, "MemTotal: %d kB", &totalKB)
		}
		if strings.HasPrefix(line, "MemAvailable:") {
			_, _ = fmt.Sscanf(line, "MemAvailable: %d kB", &availKB)
		}
	}
	if totalKB == 0 {
		return 0, 0, errors.New("mem total unavailable")
	}
	if availKB == 0 {
		availKB = totalKB
	}
	return totalKB * 1024, availKB * 1024, nil
}

func readDisk(path string) (totalGB, usedGB int64, percent float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total := int64(stat.Blocks) * int64(stat.Bsize)
	free := int64(stat.Bfree) * int64(stat.Bsize)
	used := total - free
	if total > 0 {
		percent = float64(used) / float64(total) * 100
	}
	return total / (1024 * 1024 * 1024), used / (1024 * 1024 * 1024), percent
}

func readUptime() (int64, error) {
	body, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(body))
	if len(fields) == 0 {
		return 0, errors.New("/proc/uptime empty")
	}
	seconds, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}
	return int64(seconds), nil
}
