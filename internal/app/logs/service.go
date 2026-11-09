package logs

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/docker/docker/pkg/stdcopy"

	"github.com/zpol/doki/internal/domain/models"
	"github.com/zpol/doki/internal/domain/ports"
)

// Service streams and sanitizes container logs.
type Service struct {
	docker ports.DockerClient
}

// New creates a logs service.
func New(docker ports.DockerClient) *Service {
	return &Service{docker: docker}
}

// Stream opens a follow/non-follow log stream and sends sanitized lines.
// Cancel ctx to stop. The returned cancel closes the Docker stream.
func (s *Service) Stream(ctx context.Context, id string, opts ports.LogOptions) (<-chan models.LogLine, <-chan error, context.CancelFunc, error) {
	streamCtx, cancel := context.WithCancel(ctx)
	rc, err := s.docker.ContainerLogs(streamCtx, id, opts)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}

	lines := make(chan models.LogLine, 256)
	errs := make(chan error, 1)

	go func() {
		defer close(lines)
		defer close(errs)
		defer rc.Close()
		defer cancel()

		stdoutR, stdoutW := io.Pipe()
		stderrR, stderrW := io.Pipe()

		go func() {
			_, copyErr := stdcopy.StdCopy(stdoutW, stderrW, rc)
			_ = stdoutW.CloseWithError(copyErr)
			_ = stderrW.CloseWithError(copyErr)
		}()

		done := make(chan struct{}, 2)
		go scanStream(streamCtx, stdoutR, "stdout", opts.Timestamps, lines, done)
		go scanStream(streamCtx, stderrR, "stderr", opts.Timestamps, lines, done)

		select {
		case <-streamCtx.Done():
		case <-done:
			<-done
		}
	}()

	return lines, errs, cancel, nil
}

func scanStream(ctx context.Context, r io.Reader, stream string, timestamps bool, out chan<- models.LogLine, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}
		raw := sc.Text()
		msg := Sanitize(raw)
		line := models.LogLine{
			Stream:  stream,
			Message: msg,
			Raw:     raw,
			Time:    time.Now().UTC(),
		}
		if timestamps {
			if ts, rest, ok := splitTimestamp(msg); ok {
				line.Time = ts
				line.Message = rest
			}
		}
		select {
		case <-ctx.Done():
			return
		case out <- line:
		}
	}
}

// Sanitize strips terminal escape sequences and non-printable controls.
func Sanitize(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if r == 0x1b { // ESC
			// skip CSI / OSC-ish sequences conservatively
			for i < len(s) {
				c := s[i]
				i++
				if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '@' {
					break
				}
			}
			continue
		}
		if r == '\t' || r == '\n' || r == '\r' {
			b.WriteRune(' ')
			continue
		}
		if r < 32 {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimRight(b.String(), " ")
}

func splitTimestamp(s string) (time.Time, string, bool) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return time.Time{}, s, false
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, s, false
	}
	return ts.UTC(), parts[1], true
}
