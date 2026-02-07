package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultBootstrapTimeout = 12 * time.Minute
	sshReadyTimeout         = 8 * time.Minute
	agentReadyTimeout       = 5 * time.Minute
	sshPollInterval         = 5 * time.Second
)

const agentServiceUnit = `[Unit]
Description=Iceberg Agent
After=network-online.target docker.service
Wants=network-online.target

[Service]
EnvironmentFile=/etc/iceberg-agent.env
Environment=ICEBERG_AGENT_ADDR=:8443
Environment=ICEBERG_SERVICES_DIR=/opt/iceberg/services
Environment=ICEBERG_CADDY_DIR=/opt/iceberg/caddy
ExecStart=/usr/local/bin/iceberg-agent
Restart=always
RestartSec=3
User=root

[Install]
WantedBy=multi-user.target
`

const agentBootstrapScript = `set -euo pipefail
sudo install -d -m 0755 /opt/iceberg/services /opt/iceberg/caddy
sudo install -m 0755 /tmp/iceberg-agent /usr/local/bin/iceberg-agent
sudo install -m 0600 /tmp/iceberg-agent.env /etc/iceberg-agent.env
sudo install -m 0644 /tmp/iceberg-agent.service /etc/systemd/system/iceberg-agent.service

if ! docker compose version >/dev/null 2>&1; then
  if ! command -v docker-compose >/dev/null 2>&1; then
    arch="$(uname -m)"
    case "$arch" in
      x86_64|amd64) compose_arch="x86_64" ;;
      aarch64|arm64) compose_arch="aarch64" ;;
      *) echo "unsupported architecture for docker compose: $arch" >&2; exit 1 ;;
    esac
    sudo mkdir -p /usr/local/lib/docker/cli-plugins
    sudo curl -fsSL "https://github.com/docker/compose/releases/download/v2.29.7/docker-compose-linux-${compose_arch}" -o /usr/local/lib/docker/cli-plugins/docker-compose
    sudo chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
  fi
fi

sudo systemctl daemon-reload
sudo systemctl enable --now iceberg-agent
sudo rm -f /tmp/iceberg-agent /tmp/iceberg-agent.env /tmp/iceberg-agent.service
`

func bootstrapServerDefault(ctx context.Context, serverName, publicIP, agentToken string, privateKey []byte) (string, error) {
	if strings.TrimSpace(publicIP) == "" {
		return "", errors.New("server public IP is required")
	}
	if strings.TrimSpace(agentToken) == "" {
		return "", errors.New("agent token is required")
	}
	if len(privateKey) == 0 {
		return "", errors.New("ssh private key is required")
	}

	ctx, cancel := context.WithTimeout(ctx, defaultBootstrapTimeout)
	defer cancel()

	agentBinaryPath, cleanupBinary, err := resolveLinuxAgentBinary(ctx)
	if err != nil {
		return "", err
	}
	defer cleanupBinary()

	tempDir, err := os.MkdirTemp("", "iceberg-bootstrap-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	privateKeyPath := filepath.Join(tempDir, "id_ed25519")
	if err := os.WriteFile(privateKeyPath, privateKey, 0o600); err != nil {
		return "", err
	}
	envPath := filepath.Join(tempDir, "iceberg-agent.env")
	if err := os.WriteFile(envPath, []byte("ICEBERG_AGENT_TOKEN="+agentToken+"\n"), 0o600); err != nil {
		return "", err
	}
	servicePath := filepath.Join(tempDir, "iceberg-agent.service")
	if err := os.WriteFile(servicePath, []byte(agentServiceUnit), 0o644); err != nil {
		return "", err
	}

	target := "iceberg@" + strings.TrimSpace(publicIP)
	if err := waitForSSH(ctx, privateKeyPath, target, sshReadyTimeout); err != nil {
		return "", fmt.Errorf("wait for ssh on %s: %w", serverName, err)
	}
	if err := copyFileOverSCP(ctx, privateKeyPath, agentBinaryPath, target, "/tmp/iceberg-agent"); err != nil {
		return "", err
	}
	if err := copyFileOverSCP(ctx, privateKeyPath, envPath, target, "/tmp/iceberg-agent.env"); err != nil {
		return "", err
	}
	if err := copyFileOverSCP(ctx, privateKeyPath, servicePath, target, "/tmp/iceberg-agent.service"); err != nil {
		return "", err
	}
	if _, err := runSSHScript(ctx, privateKeyPath, target, agentBootstrapScript); err != nil {
		return "", err
	}

	tailscaleRaw, err := runSSHScript(ctx, privateKeyPath, target, "tailscale ip -4 | head -n1")
	if err != nil {
		return "", err
	}
	agentIP, err := firstIPv4(string(tailscaleRaw))
	if err != nil {
		return "", fmt.Errorf("tailscale IP not available for %s: %w", serverName, err)
	}
	if err := waitForAgent(ctx, agentIP, agentReadyTimeout); err != nil {
		return "", err
	}
	return agentIP, nil
}

func resolveLinuxAgentBinary(ctx context.Context) (string, func(), error) {
	if explicit := strings.TrimSpace(os.Getenv("ICEBERG_AGENT_BINARY")); explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", nil, fmt.Errorf("ICEBERG_AGENT_BINARY %q: %w", explicit, err)
		}
		return explicit, func() {}, nil
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		return "", nil, fmt.Errorf("cannot locate go.mod from current directory; set ICEBERG_AGENT_BINARY to a Linux iceberg-agent binary")
	}

	goarch := strings.TrimSpace(os.Getenv("ICEBERG_AGENT_GOARCH"))
	if goarch == "" {
		goarch = "amd64"
	}

	tempDir, err := os.MkdirTemp("", "iceberg-agent-build-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { _ = os.RemoveAll(tempDir) }
	outputPath := filepath.Join(tempDir, "iceberg-agent")

	cmd := exec.CommandContext(ctx, "go", "build", "-o", outputPath, "./cmd/iceberg-agent")
	cmd.Dir = moduleRoot
	cmd.Env = append(os.Environ(), "GOWORK=off", "CGO_ENABLED=0", "GOOS=linux", "GOARCH="+goarch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("build Linux iceberg-agent: %w (%s)", err, compactOutput(out))
	}

	return outputPath, cleanup, nil
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", errors.New("go.mod not found")
}

func waitForSSH(ctx context.Context, keyPath, target string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		attemptCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		_, err := runSSHScript(attemptCtx, keyPath, target, "echo ready")
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if time.Now().After(deadline) {
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(sshPollInterval):
		}
	}
}

func waitForAgent(ctx context.Context, ip string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	url := "http://" + ip + ":8443/health"
	client := &http.Client{Timeout: 6 * time.Second}
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return nil
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			if lastErr == nil {
				lastErr = errors.New("agent health did not become ready")
			}
			return lastErr
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}
}

func copyFileOverSCP(ctx context.Context, keyPath, srcPath, target, dstPath string) error {
	args := append(scpOptions(keyPath), srcPath, target+":"+dstPath)
	cmd := exec.CommandContext(ctx, "scp", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("scp %s -> %s failed: %w (%s)", srcPath, target+":"+dstPath, err, compactOutput(out))
	}
	return nil
}

func runSSHScript(ctx context.Context, keyPath, target, script string) ([]byte, error) {
	args := append(sshOptions(keyPath), target, "bash", "-s")
	cmd := exec.CommandContext(ctx, "ssh", args...)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("ssh %s failed: %w (%s)", target, err, compactOutput(out))
	}
	return out, nil
}

func sshOptions(keyPath string) []string {
	return []string{
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=8",
	}
}

func scpOptions(keyPath string) []string {
	return []string{
		"-i", keyPath,
		"-o", "IdentitiesOnly=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=8",
	}
}

func firstIPv4(raw string) (string, error) {
	for _, token := range strings.Fields(raw) {
		ip := net.ParseIP(token)
		if ip != nil && ip.To4() != nil {
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("no ipv4 found in output %q", compactOutput([]byte(raw)))
}

func compactOutput(out []byte) string {
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "no output"
	}
	const max = 220
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
