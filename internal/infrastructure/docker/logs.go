package docker

import (
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"

	"github.com/zpol/doki/internal/domain/ports"
)

// ContainerLogs opens a Docker log stream. Caller must Close the reader.
func (c *Client) ContainerLogs(ctx context.Context, id string, opts ports.LogOptions) (io.ReadCloser, error) {
	stdout, stderr := opts.Stdout, opts.Stderr
	if !stdout && !stderr {
		stdout, stderr = true, true
	}
	tail := opts.Tail
	if tail == "" {
		tail = "300"
	}
	rc, err := c.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: stdout,
		ShowStderr: stderr,
		Follow:     opts.Follow,
		Tail:       tail,
		Timestamps: opts.Timestamps,
	})
	if err != nil {
		return nil, fmt.Errorf("container logs %s: %w", id, err)
	}
	return rc, nil
}
