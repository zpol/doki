package models

import "time"

// ConnectionState describes Docker Engine reachability.
type ConnectionState string

const (
	ConnectionConnected    ConnectionState = "connected"
	ConnectionConnecting   ConnectionState = "connecting"
	ConnectionDisconnected ConnectionState = "disconnected"
	ConnectionError        ConnectionState = "error"
)

// HostInfo contains Docker Engine identity and connection details.
type HostInfo struct {
	Name            string
	ContextName     string
	ServerVersion   string
	APIVersion      string
	OperatingSystem string
	Architecture    string
	NCPU            int
	MemTotal        int64
	Containers      int
	ContainersRunning int
	ContainersPaused  int
	ContainersStopped int
	Images          int
	Driver          string
	State           ConnectionState
	LastError       string
	CheckedAt       time.Time
}
