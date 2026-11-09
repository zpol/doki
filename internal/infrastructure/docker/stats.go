package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/container"

	"github.com/zpol/doki/internal/domain/models"
)

// ContainerStats fetches a single stats sample for a container.
func (c *Client) ContainerStats(ctx context.Context, id string) (models.ContainerStats, error) {
	resp, err := c.cli.ContainerStatsOneShot(ctx, id)
	if err != nil {
		return models.ContainerStats{}, fmt.Errorf("container stats %s: %w", id, err)
	}
	defer func() { _ = resp.Body.Close() }()

	var raw container.StatsResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return models.ContainerStats{}, fmt.Errorf("decode stats %s: %w", id, err)
	}
	return mapStats(id, raw), nil
}

func mapStats(id string, raw container.StatsResponse) models.ContainerStats {
	stats := models.ContainerStats{
		ContainerID: id,
		Name:        raw.Name,
		Timestamp:   time.Now().UTC(),
		Available:   true,
		MemoryUsage: raw.MemoryStats.Usage,
		MemoryLimit: raw.MemoryStats.Limit,
		PIDs:        raw.PidsStats.Current,
	}
	if stats.Name != "" && stats.Name[0] == '/' {
		stats.Name = stats.Name[1:]
	}
	stats.CPUPercent = calculateCPUPercent(raw)
	if stats.MemoryLimit > 0 {
		stats.MemoryPercent = float64(stats.MemoryUsage) / float64(stats.MemoryLimit) * 100.0
	}
	for _, n := range raw.Networks {
		stats.NetRx += n.RxBytes
		stats.NetTx += n.TxBytes
	}
	for _, e := range raw.BlkioStats.IoServiceBytesRecursive {
		switch e.Op {
		case "read", "Read":
			stats.BlockRead += e.Value
		case "write", "Write":
			stats.BlockWrite += e.Value
		}
	}
	return stats
}

func calculateCPUPercent(v container.StatsResponse) float64 {
	cpuDelta := float64(v.CPUStats.CPUUsage.TotalUsage) - float64(v.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(v.CPUStats.SystemUsage) - float64(v.PreCPUStats.SystemUsage)
	if cpuDelta <= 0 || systemDelta <= 0 {
		return 0
	}
	online := float64(v.CPUStats.OnlineCPUs)
	if online == 0 {
		online = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if online == 0 {
		online = 1
	}
	return (cpuDelta / systemDelta) * online * 100.0
}
