package models

import "time"

// Image is a Docker image summary.
type Image struct {
	ID         string
	Repository string
	Tag        string
	Size       int64
	CreatedAt  time.Time
	Containers int64
}

// ImageDetails is inspect-level image information.
type ImageDetails struct {
	Image
	RepoTags    []string
	RepoDigests []string
	Architecture string
	OS          string
	Author      string
	Entrypoint  []string
	Cmd         []string
	Env         []string
	WorkingDir  string
	ExposedPorts []string
	Labels      map[string]string
	Comment     string
}

// ShortID returns a truncated image id.
func (i Image) ShortID() string {
	id := i.ID
	if len(id) > 7 && (id[:7] == "sha256:") {
		id = id[7:]
	}
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// Volume is a Docker volume summary.
type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	Labels     map[string]string
	CreatedAt  time.Time
}

// VolumeDetails is inspect-level volume information.
type VolumeDetails struct {
	Volume
	Options  map[string]string
	RefCount int64
	Size     int64
}

// Network is a Docker network summary.
type Network struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	Attachable bool
	Containers int
}

// NetworkEndpointInfo is a container attached to a network.
type NetworkEndpointInfo struct {
	ContainerID   string
	ContainerName string
	IPv4Address   string
	IPv6Address   string
	MacAddress    string
}

// NetworkDetails is inspect-level network information.
type NetworkDetails struct {
	Network
	Created    time.Time
	EnableIPv4 bool
	EnableIPv6 bool
	Ingress    bool
	Options    map[string]string
	Labels     map[string]string
	Subnet     string
	Gateway    string
	Endpoints  []NetworkEndpointInfo
}

// ShortID returns a truncated network id.
func (n Network) ShortID() string {
	if len(n.ID) <= 12 {
		return n.ID
	}
	return n.ID[:12]
}

// Event is a Docker daemon event.
type Event struct {
	Time      time.Time
	Type      string
	Action    string
	ActorID   string
	ActorName string
	Attributes map[string]string
	Message   string
}

// ContainerStats holds a single stats sample for a container.
type ContainerStats struct {
	ContainerID   string
	Name          string
	CPUPercent    float64
	MemoryUsage   uint64
	MemoryLimit   uint64
	MemoryPercent float64
	NetRx         uint64
	NetTx         uint64
	BlockRead     uint64
	BlockWrite    uint64
	PIDs          uint64
	Timestamp     time.Time
	Available     bool
}

// MemoryPercentOrZero returns memory percent when a limit is known.
func (s ContainerStats) MemoryPercentOrZero() float64 {
	if s.MemoryLimit == 0 {
		return 0
	}
	return s.MemoryPercent
}
