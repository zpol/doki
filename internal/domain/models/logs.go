package models

import "time"

// LogLine is a sanitized container log entry.
type LogLine struct {
	Time    time.Time
	Stream  string // stdout|stderr|unknown
	Message string
	Raw     string
}

// LogSeverity is a best-effort severity guess from log text.
type LogSeverity string

const (
	LogSeverityDebug   LogSeverity = "debug"
	LogSeverityInfo    LogSeverity = "info"
	LogSeverityWarn    LogSeverity = "warn"
	LogSeverityError   LogSeverity = "error"
	LogSeverityUnknown LogSeverity = "unknown"
)

// DetectSeverity performs a conservative severity heuristic.
func DetectSeverity(message string) LogSeverity {
	lower := message
	// cheap ASCII lowercasing without allocating when possible
	buf := make([]byte, len(lower))
	for i := 0; i < len(lower); i++ {
		c := lower[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		buf[i] = c
	}
	s := string(buf)
	switch {
	case containsAny(s, " error", "error:", " err ", "fatal", "panic"):
		return LogSeverityError
	case containsAny(s, " warn", "warning", "caution"):
		return LogSeverityWarn
	case containsAny(s, " debug", "trace"):
		return LogSeverityDebug
	case containsAny(s, " info", "inform"):
		return LogSeverityInfo
	default:
		return LogSeverityUnknown
	}
}

func containsAny(s string, parts ...string) bool {
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		for i := 0; i+len(p) <= len(s); i++ {
			if s[i:i+len(p)] == p {
				return true
			}
		}
	}
	return false
}
