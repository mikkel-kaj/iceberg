package agent

import (
	"bytes"
	"context"
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
	Read(ctx context.Context, container string, tail int, follow bool) (io.ReadCloser, error)
}

type dockerLogStreamer struct{}

func (d *dockerLogStreamer) Read(ctx context.Context, container string, tail int, follow bool) (io.ReadCloser, error) {
	args := []string{"logs", "--tail", itoa(tail)}
	if follow {
		args = append(args, "--follow")
	}
	args = append(args, container)
	cmd := exec.CommandContext(ctx, "docker", args...)
	if !follow {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}
		return io.NopCloser(bytesReader(out)), nil
	}
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
