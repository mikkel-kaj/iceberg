package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mikkel-kaj/iceberg/internal/agent"
)

const version = "v0.1.0-dev"

func main() {
	token := os.Getenv("ICEBERG_AGENT_TOKEN")
	if token == "" {
		fmt.Printf("iceberg-agent %s\n", version)
		return
	}

	addr := envOr("ICEBERG_AGENT_ADDR", ":8443")
	servicesDir := envOr("ICEBERG_SERVICES_DIR", "/opt/iceberg/services")
	caddyDir := envOr("ICEBERG_CADDY_DIR", "/opt/iceberg/caddy")

	a := agent.New(token, addr, servicesDir, caddyDir)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = a.Stop(context.Background())
	}()

	log.Printf("iceberg-agent %s listening on %s", version, addr)
	if err := a.Start(ctx); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
