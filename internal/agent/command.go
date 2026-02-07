package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strconv"
)

type CommandRunner interface {
	Run(ctx context.Context, dir string, name string, args ...string) error
}

type execRunner struct{}

func (e *execRunner) Run(ctx context.Context, dir string, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	return cmd.Run()
}

type LogStreamer interface {
	Read(ctx context.Context, serviceDir, service string, tail int, follow bool) (io.ReadCloser, error)
}

type dockerLogStreamer struct{}

func (d *dockerLogStreamer) Read(ctx context.Context, serviceDir, service string, tail int, follow bool) (io.ReadCloser, error) {
	if !follow {
		out, err := d.readOnce(ctx, serviceDir, service, tail, true)
		if err == nil {
			return io.NopCloser(bytesReader(out)), nil
		}
		outFallback, errFallback := d.readOnce(ctx, serviceDir, service, tail, false)
		if errFallback != nil {
			return nil, fmt.Errorf("docker compose logs failed: %v; docker-compose logs failed: %w", err, errFallback)
		}
		return io.NopCloser(bytesReader(outFallback)), nil
	}

	stream, err := d.readFollow(ctx, serviceDir, service, tail, true)
	if err == nil {
		return stream, nil
	}
	streamFallback, errFallback := d.readFollow(ctx, serviceDir, service, tail, false)
	if errFallback != nil {
		return nil, fmt.Errorf("docker compose logs follow failed: %v; docker-compose logs follow failed: %w", err, errFallback)
	}
	return streamFallback, nil
}

func (d *dockerLogStreamer) readOnce(ctx context.Context, dir, service string, tail int, usePlugin bool) ([]byte, error) {
	cmd := composeLogsCommand(ctx, dir, service, tail, false, usePlugin)
	return cmd.CombinedOutput()
}

func (d *dockerLogStreamer) readFollow(ctx context.Context, dir, service string, tail int, usePlugin bool) (io.ReadCloser, error) {
	cmd := composeLogsCommand(ctx, dir, service, tail, true, usePlugin)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &processReadCloser{ReadCloser: stdout, done: func() error {
		_ = cmd.Process.Kill()
		return cmd.Wait()
	}}, nil
}

func composeLogsCommand(ctx context.Context, dir, service string, tail int, follow, usePlugin bool) *exec.Cmd {
	if usePlugin {
		args := []string{"compose", "logs", "--tail", itoa(tail)}
		if follow {
			args = append(args, "--follow")
		}
		args = append(args, service)
		cmd := exec.CommandContext(ctx, "docker", args...)
		cmd.Dir = dir
		return cmd
	}

	args := []string{"logs", "--tail", itoa(tail)}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, service)
	cmd := exec.CommandContext(ctx, "docker-compose", args...)
	cmd.Dir = dir
	return cmd
}

type processReadCloser struct {
	io.ReadCloser
	done func() error
}

func (p *processReadCloser) Close() error {
	_ = p.ReadCloser.Close()
	if p.done != nil {
		return p.done()
	}
	return nil
}

func itoa(v int) string {
	return strconv.Itoa(v)
}

func bytesReader(b []byte) *bytes.Reader {
	return bytes.NewReader(b)
}
