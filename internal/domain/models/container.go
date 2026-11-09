package models

import "time"

// ContainerStatus represents the high-level lifecycle state of a container.
type ContainerStatus string

const (
	StatusCreated    ContainerStatus = "created"
	StatusRunning    ContainerStatus = "running"
	StatusPaused     ContainerStatus = "paused"
	StatusRestarting ContainerStatus = "restarting"
	StatusRemoving   ContainerStatus = "removing"
	StatusExited     ContainerStatus = "exited"
	StatusDead       ContainerStatus = "dead"
	StatusUnknown    ContainerStatus = "unknown"
)

// HealthStatus represents Docker health-check state when configured.
type HealthStatus string

const (
	HealthNone        HealthStatus = "none"
	HealthStarting    HealthStatus = "starting"
	HealthHealthy     HealthStatus = "healthy"
	HealthUnhealthy   HealthStatus = "unhealthy"
	HealthUnknown     HealthStatus = "unknown"
)

// PortBinding describes a published container port.
type PortBinding struct {
	IP          string
	PrivatePort uint16
	PublicPort  uint16
	Type        string
}

// Container is the domain representation of a Docker container summary.
type Container struct {
	ID           string
	Name         string
	Image        string
	Status       ContainerStatus
	StatusText   string
	Health       HealthStatus
	CreatedAt    time.Time
	Ports        []PortBinding
	Labels       map[string]string
	ComposeProject string
	ComposeService string
	RestartCount int
}

// ShortID returns the truncated container identifier.
func (c Container) ShortID() string {
	if len(c.ID) <= 12 {
		return c.ID
	}
	return c.ID[:12]
}

// IsRunning reports whether the container is actively running.
func (c Container) IsRunning() bool {
	return c.Status == StatusRunning
}

// ContainerDetails holds inspect-level container information.
type ContainerDetails struct {
	Container
	Path         string
	Args         []string
	Env          []string
	Cmd          []string
	Entrypoint   []string
	WorkingDir   string
	User         string
	Platform     string
	Driver       string
	NetworkMode     string
	Privileged      bool
	RestartPolicy   string
	ReadonlyRootfs  bool
	CapAdd          []string
	CapDrop         []string
	SecurityOpt     []string
	Mounts          []Mount
	Networks        []NetworkEndpoint
	StartedAt       time.Time
	FinishedAt      time.Time
	ExitCode        int
	OOMKilled       bool
	Pid             int
	RawJSON         string
}

// Mount describes a container mount point.
type Mount struct {
	Type        string
	Name        string
	Source      string
	Destination string
	Mode        string
	RW          bool
}

// NetworkEndpoint describes a container network attachment.
type NetworkEndpoint struct {
	NetworkID   string
	NetworkName string
	IPAddress   string
	Gateway     string
	MacAddress  string
}
