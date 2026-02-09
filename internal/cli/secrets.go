package cli

import (
	"context"
	"fmt"
	"strings"
)

func resolveEnvSecrets(ctx context.Context, env map[string]string) (map[string]string, error) {
	if len(env) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(env))
	for key, value := range env {
		if isBitwardenRef(value) {
			resolved, err := resolveBitwardenSecret(ctx, value)
			if err != nil {
				return nil, fmt.Errorf("resolve %s from bitwarden: %w", key, err)
			}
			out[key] = resolved
			continue
		}
		out[key] = value
	}
	return out, nil
}

func isBitwardenRef(v string) bool {
	return strings.HasPrefix(v, "bw://") || strings.HasPrefix(v, "bitwarden://")
}
