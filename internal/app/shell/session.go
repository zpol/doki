package shell

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/term"

	"github.com/zpol/doki/internal/domain/ports"
)

// Service opens interactive container shells via Docker exec.
type Service struct {
	docker ports.DockerClient
}

// New creates a shell service.
func New(docker ports.DockerClient) *Service {
	return &Service{docker: docker}
}

// Command is a tea.ExecCommand implementation that owns the terminal.
type Command struct {
	docker      ports.DockerClient
	containerID string
	cmd         []string
	stdin       io.Reader
	stdout      io.Writer
	stderr      io.Writer
}

// NewCommand prepares an interactive shell command for tea.Exec.
func (s *Service) NewCommand(containerID string, cmd []string) *Command {
	if len(cmd) == 0 {
		cmd = []string{"/bin/sh"}
	}
	return &Command{
		docker:      s.docker,
		containerID: containerID,
		cmd:         cmd,
	}
}

func (c *Command) SetStdin(r io.Reader)  { c.stdin = r }
func (c *Command) SetStdout(w io.Writer) { c.stdout = w }
func (c *Command) SetStderr(w io.Writer) { c.stderr = w }

// Run attaches an interactive exec session and copies stdio until exit.
func (c *Command) Run() error {
	ctx := context.Background()
	execID, err := c.docker.CreateExec(ctx, c.containerID, ports.ExecOptions{
		Cmd:          c.cmd,
		TTY:          true,
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: false, // TTY multiplexes into a single stream
	})
	if err != nil {
		return err
	}
	session, err := c.docker.AttachExec(ctx, execID)
	if err != nil {
		return err
	}

	var closeOnce sync.Once
	closeSession := func() {
		closeOnce.Do(func() {
			_ = session.Close()
		})
	}
	defer closeSession()

	fd := int(os.Stdin.Fd())
	var oldState *term.State
	if term.IsTerminal(fd) {
		oldState, err = term.MakeRaw(fd)
		if err != nil {
			return fmt.Errorf("make raw terminal: %w", err)
		}
		defer func() {
			_ = term.Restore(fd, oldState)
		}()
		if w, h, err := term.GetSize(fd); err == nil {
			_ = session.Resize(uint(w), uint(h))
		}
	}

	stdin := c.stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := c.stdout
	if stdout == nil {
		stdout = os.Stdout
	}

	errCh := make(chan error, 2)

	// stdin -> container
	go func() {
		_, copyErr := io.Copy(session.Writer(), stdin)
		// User typed EOF / stdin closed: half-close so the remote shell can exit.
		_ = session.CloseWrite()
		errCh <- copyErr
	}()

	// container -> stdout
	go func() {
		_, copyErr := io.Copy(stdout, session.Reader())
		// Remote exited: close the hijack so the stdin Copy unblocks.
		// Without this, Run() hangs forever waiting on stdin.
		closeSession()
		errCh <- copyErr
	}()

	// Wait for both directions to finish. The first exit unblocks the second
	// via Close / CloseWrite above.
	err1 := <-errCh
	err2 := <-errCh
	if err1 != nil && err1 != io.EOF {
		return err1
	}
	if err2 != nil && err2 != io.EOF {
		return err2
	}
	return nil
}
