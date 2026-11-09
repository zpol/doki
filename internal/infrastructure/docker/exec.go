package docker

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

	"github.com/zpol/doki/internal/domain/ports"
)

type execSession struct {
	execID string
	resp   types.HijackedResponse
	cli    *Client
	once   sync.Once
}

func (s *execSession) Reader() io.Reader { return s.resp.Reader }
func (s *execSession) Writer() io.Writer { return s.resp.Conn }
func (s *execSession) ExecID() string    { return s.execID }

func (s *execSession) Resize(width, height uint) error {
	return s.cli.cli.ContainerExecResize(context.Background(), s.execID, container.ResizeOptions{
		Width:  width,
		Height: height,
	})
}

func (s *execSession) CloseWrite() error {
	return s.resp.CloseWrite()
}

func (s *execSession) Close() error {
	s.once.Do(func() {
		s.resp.Close()
	})
	return nil
}

// CreateExec creates an exec instance inside a container.
func (c *Client) CreateExec(ctx context.Context, containerID string, opts ports.ExecOptions) (string, error) {
	cmd := opts.Cmd
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}
	resp, err := c.cli.ContainerExecCreate(ctx, containerID, container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   opts.WorkingDir,
		Env:          opts.Env,
		Tty:          opts.TTY,
		AttachStdin:  opts.AttachStdin,
		AttachStdout: opts.AttachStdout,
		AttachStderr: opts.AttachStderr,
	})
	if err != nil {
		return "", fmt.Errorf("exec create %s: %w", containerID, err)
	}
	return resp.ID, nil
}

// AttachExec attaches to a previously created exec instance.
func (c *Client) AttachExec(ctx context.Context, execID string) (ports.ExecSession, error) {
	hijacked, err := c.cli.ContainerExecAttach(ctx, execID, container.ExecAttachOptions{
		Tty: true,
	})
	if err != nil {
		return nil, fmt.Errorf("exec attach %s: %w", execID, err)
	}
	return &execSession{
		execID: execID,
		resp:   hijacked,
		cli:    c,
	}, nil
}
