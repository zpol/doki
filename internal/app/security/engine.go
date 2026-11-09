package security

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Checker evaluates a container for a specific security condition.
type Checker interface {
	ID() string
	Check(details models.ContainerDetails) []models.Finding
}

// Engine runs read-only security configuration checks.
type Engine struct {
	docker   ports.DockerClient
	checkers []Checker
}

// New creates a security engine with the default rule set.
func New(docker ports.DockerClient) *Engine {
	return &Engine{
		docker: docker,
		checkers: []Checker{
			privilegedChecker{},
			hostNetworkChecker{},
			dockerSocketChecker{},
			sensitiveMountChecker{},
			capabilitiesChecker{},
			rootUserChecker{},
			writableRootFSChecker{},
			noNewPrivilegesChecker{},
			portExposureChecker{},
		},
	}
}

// Disclaimer clarifies scope limitations.
func Disclaimer() string {
	return "Security Inspector performs configuration analysis only. It is not a vulnerability scanner or complete runtime threat detector. Absence of findings does not mean a container is safe."
}

// InspectContainer evaluates one container.
func (e *Engine) InspectContainer(ctx context.Context, id string) ([]models.Finding, error) {
	details, err := e.docker.InspectContainer(ctx, id)
	if err != nil {
		return nil, err
	}
	var out []models.Finding
	for _, checker := range e.checkers {
		out = append(out, checker.Check(details)...)
	}
	return out, nil
}

// InspectAll evaluates all containers.
func (e *Engine) InspectAll(ctx context.Context) ([]models.Finding, error) {
	items, err := e.docker.ListContainers(ctx, ports.ContainerFilter{All: true})
	if err != nil {
		return nil, err
	}
	var out []models.Finding
	for _, item := range items {
		findings, err := e.InspectContainer(ctx, item.ID)
		if err != nil {
			continue
		}
		out = append(out, findings...)
	}
	return out, nil
}

type privilegedChecker struct{}

func (privilegedChecker) ID() string { return "privileged" }
func (privilegedChecker) Check(d models.ContainerDetails) []models.Finding {
	if !d.Privileged {
		return nil
	}
	return []models.Finding{{
		ID:           "SEC-PRIVILEGED",
		Title:        "Privileged execution",
		Severity:     models.SeverityHigh,
		ResourceID:   d.ID,
		ResourceName: d.Name,
		Description:  "Container is running with HostConfig.Privileged=true, which grants extensive host capabilities.",
		Evidence:     []models.Evidence{{Field: "HostConfig.Privileged", Value: "true"}},
		Recommendation: "Avoid privileged mode. Grant only the specific capabilities required.",
	}}
}

type hostNetworkChecker struct{}

func (hostNetworkChecker) ID() string { return "host-network" }
func (hostNetworkChecker) Check(d models.ContainerDetails) []models.Finding {
	if !strings.EqualFold(d.NetworkMode, "host") {
		return nil
	}
	return []models.Finding{{
		ID:           "SEC-HOST-NET",
		Title:        "Host networking",
		Severity:     models.SeverityHigh,
		ResourceID:   d.ID,
		ResourceName: d.Name,
		Description:  "Container uses the host network namespace.",
		Evidence:     []models.Evidence{{Field: "HostConfig.NetworkMode", Value: d.NetworkMode}},
		Recommendation: "Prefer bridge/custom networks unless host networking is explicitly required.",
	}}
}

type dockerSocketChecker struct{}

func (dockerSocketChecker) ID() string { return "docker-socket" }
func (dockerSocketChecker) Check(d models.ContainerDetails) []models.Finding {
	var findings []models.Finding
	for _, m := range d.Mounts {
		src := path.Clean(m.Source)
		if src == "/var/run/docker.sock" || strings.HasSuffix(src, "/docker.sock") {
			findings = append(findings, models.Finding{
				ID:           "SEC-DOCKER-SOCK",
				Title:        "Docker socket mounted",
				Severity:     models.SeverityHigh,
				ResourceID:   d.ID,
				ResourceName: d.Name,
				Description:  "Mounting the Docker socket often equates to host root access.",
				Evidence: []models.Evidence{
					{Field: "Mount.Source", Value: m.Source},
					{Field: "Mount.Destination", Value: m.Destination},
				},
				Recommendation: "Avoid docker.sock mounts. Use a tightly scoped API proxy if Docker access is required.",
			})
		}
	}
	return findings
}

type sensitiveMountChecker struct{}

func (sensitiveMountChecker) ID() string { return "sensitive-mounts" }
func (sensitiveMountChecker) Check(d models.ContainerDetails) []models.Finding {
	sensitive := []string{"/", "/etc", "/proc", "/sys"}
	var findings []models.Finding
	for _, m := range d.Mounts {
		src := path.Clean(m.Source)
		for _, s := range sensitive {
			if src == s {
				findings = append(findings, models.Finding{
					ID:           "SEC-SENSITIVE-MOUNT",
					Title:        "Sensitive host path mounted",
					Severity:     models.SeverityHigh,
					ResourceID:   d.ID,
					ResourceName: d.Name,
					Description:  fmt.Sprintf("Host path %s is mounted into the container.", src),
					Evidence: []models.Evidence{
						{Field: "Mount.Source", Value: m.Source},
						{Field: "Mount.Destination", Value: m.Destination},
						{Field: "Mount.RW", Value: fmt.Sprintf("%v", m.RW)},
					},
					Recommendation: "Mount only required subpaths with the least privileges needed.",
				})
			}
		}
	}
	return findings
}

type capabilitiesChecker struct{}

func (capabilitiesChecker) ID() string { return "capabilities" }
func (capabilitiesChecker) Check(d models.ContainerDetails) []models.Finding {
	risky := map[string]models.Severity{
		"SYS_ADMIN":  models.SeverityHigh,
		"SYS_PTRACE": models.SeverityHigh,
		"NET_ADMIN":  models.SeverityMedium,
	}
	var findings []models.Finding
	for _, capName := range d.CapAdd {
		name := strings.TrimPrefix(strings.ToUpper(capName), "CAP_")
		if sev, ok := risky[name]; ok {
			findings = append(findings, models.Finding{
				ID:           "SEC-CAP-" + name,
				Title:        "Risky capability added",
				Severity:     sev,
				ResourceID:   d.ID,
				ResourceName: d.Name,
				Description:  "Container adds capability " + name + ", which expands attack impact if compromised.",
				Evidence:     []models.Evidence{{Field: "HostConfig.CapAdd", Value: capName}},
				Recommendation: "Drop unused capabilities and avoid " + name + " unless strictly required.",
			})
		}
	}
	return findings
}

type rootUserChecker struct{}

func (rootUserChecker) ID() string { return "root-user" }
func (rootUserChecker) Check(d models.ContainerDetails) []models.Finding {
	user := strings.TrimSpace(d.User)
	if user != "" && user != "0" && user != "root" && !strings.HasPrefix(user, "0:") && !strings.HasPrefix(user, "root:") {
		return nil
	}
	sev := models.SeverityMedium
	desc := "Container is configured to run as root (or user is unset, which commonly defaults to root)."
	if user == "" {
		desc = "Container user is unset. Many images default to root; effective UID was not verified at runtime."
	}
	return []models.Finding{{
		ID:           "SEC-ROOT-USER",
		Title:        "Running as root",
		Severity:     sev,
		ResourceID:   d.ID,
		ResourceName: d.Name,
		Description:  desc,
		Evidence:     []models.Evidence{{Field: "Config.User", Value: empty(user, "(unset)")}},
		Recommendation: "Set an explicit non-root user in the image or container config.",
	}}
}

type writableRootFSChecker struct{}

func (writableRootFSChecker) ID() string { return "writable-rootfs" }
func (writableRootFSChecker) Check(d models.ContainerDetails) []models.Finding {
	if d.ReadonlyRootfs {
		return nil
	}
	return []models.Finding{{
		ID:           "SEC-WRITABLE-ROOTFS",
		Title:        "Root filesystem writable",
		Severity:     models.SeverityMedium,
		ResourceID:   d.ID,
		ResourceName: d.Name,
		Description:  "Container root filesystem is writable.",
		Evidence:     []models.Evidence{{Field: "HostConfig.ReadonlyRootfs", Value: "false"}},
		Recommendation: "Consider ReadonlyRootfs=true and mount specific writable paths as needed.",
	}}
}

type noNewPrivilegesChecker struct{}

func (noNewPrivilegesChecker) ID() string { return "no-new-privileges" }
func (noNewPrivilegesChecker) Check(d models.ContainerDetails) []models.Finding {
	for _, opt := range d.SecurityOpt {
		if strings.Contains(strings.ToLower(opt), "no-new-privileges") {
			return nil
		}
	}
	return []models.Finding{{
		ID:           "SEC-NO-NEW-PRIV",
		Title:        "no-new-privileges not set",
		Severity:     models.SeverityLow,
		ResourceID:   d.ID,
		ResourceName: d.Name,
		Description:  "Security option no-new-privileges was not detected in HostConfig.SecurityOpt.",
		Evidence:     []models.Evidence{{Field: "HostConfig.SecurityOpt", Value: strings.Join(d.SecurityOpt, ",")}},
		Recommendation: "Add security-opt no-new-privileges:true unless incompatible with the workload.",
	}}
}

type portExposureChecker struct{}

func (portExposureChecker) ID() string { return "port-exposure" }
func (portExposureChecker) Check(d models.ContainerDetails) []models.Finding {
	var findings []models.Finding
	for _, p := range d.Ports {
		if p.PublicPort == 0 {
			continue
		}
		ip := p.IP
		if ip == "" || ip == "0.0.0.0" || ip == "::" {
			findings = append(findings, models.Finding{
				ID:           "SEC-PORT-ALL-INTERFACES",
				Title:        "Port bound to all interfaces",
				Severity:     models.SeverityInfo,
				ResourceID:   d.ID,
				ResourceName: d.Name,
				Description:  "Published port is bound on all interfaces. Public reachability still depends on host routing/firewall.",
				Evidence: []models.Evidence{
					{Field: "Port.IP", Value: empty(ip, "0.0.0.0")},
					{Field: "Port.Public", Value: fmt.Sprintf("%d", p.PublicPort)},
					{Field: "Port.Private", Value: fmt.Sprintf("%d", p.PrivatePort)},
				},
				Recommendation: "Bind to loopback or a specific interface when external exposure is not required.",
			})
		}
	}
	return findings
}

func empty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
