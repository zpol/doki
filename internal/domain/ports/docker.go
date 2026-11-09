package ports

import (
	"context"
	"io"

	"github.com/zpol/doki/internal/domain/models"
)

// ContainerFilter controls which containers are returned by ListContainers.
type ContainerFilter struct {
	All    bool
	Filter map[string][]string
}

// ContainerAction identifies a lifecycle operation.
type ContainerAction string

const (
	ActionStart   ContainerAction = "start"
	ActionStop    ContainerAction = "stop"
	ActionRestart ContainerAction = "restart"
	ActionPause   ContainerAction = "pause"
	ActionUnpause ContainerAction = "unpause"
	ActionRemove  ContainerAction = "remove"
)

// RemoveOptions controls container deletion behavior.
type RemoveOptions struct {
	Force         bool
	RemoveVolumes bool // never implied; caller must opt in explicitly
}

// LogOptions controls container log streaming.
type LogOptions struct {
	Follow     bool
	Tail       string
	Timestamps bool
	Stdout     bool
	Stderr     bool
}

// ExecOptions configures an interactive exec session.
type ExecOptions struct {
	Cmd          []string
	WorkingDir   string
	Env          []string
	TTY          bool
	AttachStdin  bool
	AttachStdout bool
	AttachStderr bool
}

// ExecSession represents an attached interactive exec stream.
type ExecSession interface {
	Reader() io.Reader
	Writer() io.Writer
	Resize(width, height uint) error
	CloseWrite() error
	Close() error
	ExecID() string
}

// DockerClient abstracts Docker Engine operations used by the application layer.
type DockerClient interface {
	Ping(ctx context.Context) error
	HostInfo(ctx context.Context) (models.HostInfo, error)
	ListContainers(ctx context.Context, filter ContainerFilter) ([]models.Container, error)
	InspectContainer(ctx context.Context, id string) (models.ContainerDetails, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, id string, timeoutSeconds *int) error
	RestartContainer(ctx context.Context, id string, timeoutSeconds *int) error
	PauseContainer(ctx context.Context, id string) error
	UnpauseContainer(ctx context.Context, id string) error
	RemoveContainer(ctx context.Context, id string, opts RemoveOptions) error

	ContainerLogs(ctx context.Context, id string, opts LogOptions) (io.ReadCloser, error)
	ContainerStats(ctx context.Context, id string) (models.ContainerStats, error)
	ListImages(ctx context.Context) ([]models.Image, error)
	InspectImage(ctx context.Context, id string) (models.ImageDetails, error)
	ListVolumes(ctx context.Context) ([]models.Volume, error)
	InspectVolume(ctx context.Context, name string) (models.VolumeDetails, error)
	ListNetworks(ctx context.Context) ([]models.Network, error)
	InspectNetwork(ctx context.Context, id string) (models.NetworkDetails, error)
	Events(ctx context.Context) (<-chan models.Event, <-chan error)
	CreateExec(ctx context.Context, containerID string, opts ExecOptions) (string, error)
	AttachExec(ctx context.Context, execID string) (ExecSession, error)
	Close() error
}
