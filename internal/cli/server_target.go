package cli

import (
	"fmt"
	"strings"

	"github.com/mikkel-kaj/iceberg/internal/config"
)

func agentEndpointIP(server *config.ServerEntry) string {
	if server == nil {
		return ""
	}
	if ip := strings.TrimSpace(server.IP); ip != "" {
		return ip
	}
	return strings.TrimSpace(server.PublicIP)
}

func publicDNSIP(server *config.ServerEntry) string {
	if server == nil {
		return ""
	}
	if ip := strings.TrimSpace(server.PublicIP); ip != "" {
		return ip
	}
	return strings.TrimSpace(server.IP)
}

func agentBaseURL(server *config.ServerEntry) (string, error) {
	if server == nil {
		return "", fmt.Errorf("server is nil")
	}
	ip := agentEndpointIP(server)
	if ip == "" {
		return "", fmt.Errorf("server %q has no reachable endpoint IP configured", server.Name)
	}
	return "http://" + ip + ":8443", nil
}
