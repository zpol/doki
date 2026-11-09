package metrics

import (
	"context"
	"sync"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// historySize must cover Width*2 braille samples on wide terminals.
// Each braille column consumes two history points; a ~120-col graph needs 240.
const historySize = 256

// Aggregate is a snapshot of summed container metrics.
// Values are container-aggregated, not host CPU/memory unless labeled otherwise.
type Aggregate struct {
	CPUPercent  float64
	MemoryUsage uint64
	MemoryLimit uint64
	NetRx       uint64
	NetTx       uint64
	BlockRead   uint64
	BlockWrite  uint64
	PIDs        uint64
	Samples     int
}

// Service collects bounded container metrics histories.
type Service struct {
	docker ports.DockerClient

	mu      sync.Mutex
	history map[string]*ring
	agg     *ring
	lastAgg Aggregate
	// previous totals for rate-style sparklines (bytes)
	prevNetRx, prevNetTx     uint64
	prevBlkR, prevBlkW       uint64
	havePrevIO               bool
	netRxRate, netTxRate     []float64
	blkRRate, blkWRate       []float64
}

type ring struct {
	cpu []float64
	mem []float64
}

// New creates a metrics service.
func New(docker ports.DockerClient) *Service {
	return &Service{
		docker:  docker,
		history: make(map[string]*ring),
		agg:     &ring{},
	}
}

// Sample fetches stats for the provided container IDs with bounded concurrency
// and updates the dashboard aggregate totals.
func (s *Service) Sample(ctx context.Context, ids []string) ([]models.ContainerStats, error) {
	return s.sample(ctx, ids, true)
}

// SampleOne fetches stats for specific containers without rewriting aggregate totals.
func (s *Service) SampleOne(ctx context.Context, ids []string) ([]models.ContainerStats, error) {
	return s.sample(ctx, ids, false)
}

func (s *Service) sample(ctx context.Context, ids []string, updateAggregate bool) ([]models.ContainerStats, error) {
	if len(ids) == 0 {
		if updateAggregate {
			s.recordAggregate(Aggregate{})
		}
		return nil, nil
	}

	type result struct {
		stats models.ContainerStats
		err   error
	}

	workers := 4
	if len(ids) < workers {
		workers = len(ids)
	}
	jobs := make(chan string)
	out := make(chan result, len(ids))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				stats, err := s.docker.ContainerStats(ctx, id)
				out <- result{stats: stats, err: err}
			}
		}()
	}
	go func() {
		defer close(jobs)
		for _, id := range ids {
			select {
			case <-ctx.Done():
				return
			case jobs <- id:
			}
		}
	}()
	go func() {
		wg.Wait()
		close(out)
	}()

	items := make([]models.ContainerStats, 0, len(ids))
	var agg Aggregate
	for r := range out {
		if r.err != nil {
			continue
		}
		s.record(r.stats)
		items = append(items, r.stats)
		agg.CPUPercent += r.stats.CPUPercent
		agg.MemoryUsage += r.stats.MemoryUsage
		agg.MemoryLimit += r.stats.MemoryLimit
		agg.NetRx += r.stats.NetRx
		agg.NetTx += r.stats.NetTx
		agg.BlockRead += r.stats.BlockRead
		agg.BlockWrite += r.stats.BlockWrite
		agg.PIDs += r.stats.PIDs
		agg.Samples++
	}
	if updateAggregate {
		s.recordAggregate(agg)
	}
	return items, nil
}

// History returns CPU/memory sparkline histories for a container.
func (s *Service) History(id string) (cpu []float64, mem []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.history[id]
	if r == nil {
		return nil, nil
	}
	return append([]float64(nil), r.cpu...), append([]float64(nil), r.mem...)
}

// Aggregate returns the latest summed container metrics.
func (s *Service) Aggregate() Aggregate {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastAgg
}

// AggregateHistory returns sparkline data for aggregate CPU/mem and IO rates.
func (s *Service) AggregateHistory() (cpu, mem, netRx, netTx, blkR, blkW []float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.agg == nil {
		return nil, nil, nil, nil, nil, nil
	}
	return append([]float64(nil), s.agg.cpu...),
		append([]float64(nil), s.agg.mem...),
		append([]float64(nil), s.netRxRate...),
		append([]float64(nil), s.netTxRate...),
		append([]float64(nil), s.blkRRate...),
		append([]float64(nil), s.blkWRate...)
}

func (s *Service) record(stats models.ContainerStats) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.history[stats.ContainerID]
	if r == nil {
		r = &ring{}
		s.history[stats.ContainerID] = r
	}
	r.cpu = appendBounded(r.cpu, stats.CPUPercent, historySize)
	r.mem = appendBounded(r.mem, stats.MemoryPercentOrZero(), historySize)
}

func (s *Service) recordAggregate(agg Aggregate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastAgg = agg
	if s.agg == nil {
		s.agg = &ring{}
	}
	s.agg.cpu = appendBounded(s.agg.cpu, agg.CPUPercent, historySize)
	memPct := 0.0
	if agg.MemoryLimit > 0 {
		memPct = float64(agg.MemoryUsage) / float64(agg.MemoryLimit) * 100
	}
	s.agg.mem = appendBounded(s.agg.mem, memPct, historySize)

	if s.havePrevIO {
		s.netRxRate = appendBounded(s.netRxRate, float64(delta(agg.NetRx, s.prevNetRx)), historySize)
		s.netTxRate = appendBounded(s.netTxRate, float64(delta(agg.NetTx, s.prevNetTx)), historySize)
		s.blkRRate = appendBounded(s.blkRRate, float64(delta(agg.BlockRead, s.prevBlkR)), historySize)
		s.blkWRate = appendBounded(s.blkWRate, float64(delta(agg.BlockWrite, s.prevBlkW)), historySize)
	}
	s.prevNetRx, s.prevNetTx = agg.NetRx, agg.NetTx
	s.prevBlkR, s.prevBlkW = agg.BlockRead, agg.BlockWrite
	s.havePrevIO = true
}

func delta(cur, prev uint64) uint64 {
	if cur < prev {
		return 0
	}
	return cur - prev
}

func appendBounded(in []float64, v float64, max int) []float64 {
	in = append(in, v)
	if len(in) > max {
		in = in[len(in)-max:]
	}
	return in
}
