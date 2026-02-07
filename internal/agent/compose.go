package agent

import (
	"context"
	"fmt"
)

func (a *Agent) runCompose(ctx context.Context, dir string, args ...string) error {
	composeArgs := append([]string{"compose"}, args...)
	if err := a.runner.Run(ctx, dir, "docker", composeArgs...); err == nil {
		return nil
	} else if err2 := a.runner.Run(ctx, dir, "docker-compose", args...); err2 == nil {
		return nil
	} else {
		return fmt.Errorf("docker compose failed: %v; docker-compose failed: %w", err, err2)
	}
}
