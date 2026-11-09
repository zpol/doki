package containers

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

type mockDocker struct {
	containers []models.Container
	listErr    error
	started    []string
	stopped    []string
	restarted  []string
	removed    []string
}

func (m *mockDocker) Ping(ctx context.Context) error { return nil }
func (m *mockDocker) HostInfo(ctx context.Context) (models.HostInfo, error) {
	return models.HostInfo{State: models.ConnectionConnected}, nil
}
func (m *mockDocker) ListContainers(ctx context.Context, filter ports.ContainerFilter) ([]models.Container, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return append([]models.Container(nil), m.containers...), nil
}
func (m *mockDocker) InspectContainer(ctx context.Context, id string) (models.ContainerDetails, error) {
	return models.ContainerDetails{Container: models.Container{ID: id, Name: "x"}}, nil
}
func (m *mockDocker) StartContainer(ctx context.Context, id string) error {
	m.started = append(m.started, id)
	return nil
}
func (m *mockDocker) StopContainer(ctx context.Context, id string, timeoutSeconds *int) error {
	m.stopped = append(m.stopped, id)
	return nil
}
func (m *mockDocker) RestartContainer(ctx context.Context, id string, timeoutSeconds *int) error {
	m.restarted = append(m.restarted, id)
	return nil
}
func (m *mockDocker) PauseContainer(ctx context.Context, id string) error   { return nil }
func (m *mockDocker) UnpauseContainer(ctx context.Context, id string) error { return nil }
func (m *mockDocker) RemoveContainer(ctx context.Context, id string, opts ports.RemoveOptions) error {
	if opts.RemoveVolumes {
		return errors.New("volumes must not be removed implicitly")
	}
	m.removed = append(m.removed, id)
	return nil
}
func (m *mockDocker) ContainerLogs(ctx context.Context, id string, opts ports.LogOptions) (io.ReadCloser, error) {
	return io.NopCloser(stringsNewReader("")), nil
}
func (m *mockDocker) ContainerStats(ctx context.Context, id string) (models.ContainerStats, error) {
	return models.ContainerStats{ContainerID: id, Available: true}, nil
}
func (m *mockDocker) ListImages(ctx context.Context) ([]models.Image, error) { return nil, nil }
func (m *mockDocker) InspectImage(ctx context.Context, id string) (models.ImageDetails, error) {
	return models.ImageDetails{}, nil
}
func (m *mockDocker) ListVolumes(ctx context.Context) ([]models.Volume, error) { return nil, nil }
func (m *mockDocker) InspectVolume(ctx context.Context, name string) (models.VolumeDetails, error) {
	return models.VolumeDetails{}, nil
}
func (m *mockDocker) ListNetworks(ctx context.Context) ([]models.Network, error) { return nil, nil }
func (m *mockDocker) InspectNetwork(ctx context.Context, id string) (models.NetworkDetails, error) {
	return models.NetworkDetails{}, nil
}
func (m *mockDocker) Events(ctx context.Context) (<-chan models.Event, <-chan error) {
	ch := make(chan models.Event)
	errCh := make(chan error)
	close(ch)
	close(errCh)
	return ch, errCh
}
func (m *mockDocker) CreateExec(ctx context.Context, containerID string, opts ports.ExecOptions) (string, error) {
	return "", errors.New("not implemented")
}
func (m *mockDocker) AttachExec(ctx context.Context, execID string) (ports.ExecSession, error) {
	return nil, errors.New("not implemented")
}

// Ensure mock still satisfies ports.DockerClient as the interface evolves.
var _ ports.DockerClient = (*mockDocker)(nil)
func (m *mockDocker) Close() error { return nil }

type stringsReader struct{ s string; i int }

func stringsNewReader(s string) *stringsReader { return &stringsReader{s: s} }
func (r *stringsReader) Read(p []byte) (int, error) {
	if r.i >= len(r.s) {
		return 0, io.EOF
	}
	n := copy(p, r.s[r.i:])
	r.i += n
	return n, nil
}

func TestListFiltersAndSorts(t *testing.T) {
	mock := &mockDocker{
		containers: []models.Container{
			{ID: "1", Name: "zebra", Status: models.StatusExited, Image: "alpine"},
			{ID: "2", Name: "api", Status: models.StatusRunning, Image: "nginx"},
			{ID: "3", Name: "worker", Status: models.StatusRunning, Image: "busybox"},
		},
	}
	svc := New(mock)

	items, err := svc.List(context.Background(), ListOptions{ShowStopped: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 running containers, got %d", len(items))
	}
	if items[0].Name != "api" || items[1].Name != "worker" {
		t.Fatalf("unexpected sort order: %#v", items)
	}

	items, err = svc.List(context.Background(), ListOptions{ShowStopped: true, Query: "api"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "api" {
		t.Fatalf("expected api only, got %#v", items)
	}
}

func TestRemoveNeverDeletesVolumes(t *testing.T) {
	mock := &mockDocker{}
	svc := New(mock)
	if err := svc.Remove(context.Background(), "abc", true); err != nil {
		t.Fatal(err)
	}
	if len(mock.removed) != 1 {
		t.Fatalf("expected remove call")
	}
}

func TestListPropagatesErrors(t *testing.T) {
	mock := &mockDocker{listErr: errors.New("boom")}
	svc := New(mock)
	_, err := svc.List(context.Background(), ListOptions{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMatchesByImageAndStatus(t *testing.T) {
	c := models.Container{
		ID:         "abcdef1234567890",
		Name:       "web",
		Image:      "nginx:latest",
		Status:     models.StatusRunning,
		StatusText: "Up 2 hours",
		CreatedAt:  time.Now(),
	}
	if !matches(c, "nginx") {
		t.Fatal("expected image match")
	}
	if !matches(c, "abcdef") {
		t.Fatal("expected id match")
	}
	if matches(c, "postgres") {
		t.Fatal("unexpected match")
	}
}
