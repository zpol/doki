package models

// Severity classifies security and diagnostic findings.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
	SeverityInfo     Severity = "informational"
	SeverityUnknown  Severity = "unknown"
)

// Evidence supports a finding with an observed fact.
type Evidence struct {
	Field string
	Value string
}

// Finding is a security configuration finding.
type Finding struct {
	ID             string
	Title          string
	Severity       Severity
	ResourceID     string
	ResourceName   string
	Description    string
	Evidence       []Evidence
	Recommendation string
}

// DiagnosticFinding is an operational troubleshooting finding.
type DiagnosticFinding struct {
	ID                   string
	ResourceID           string
	ResourceName         string
	Severity             Severity
	Observation          string
	Evidence             []Evidence
	PossibleCauses       []string
	RecommendedVerify    []string
	SuggestedNextSteps   []string
	ConfidenceNote       string
}
