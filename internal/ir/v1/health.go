// Package v1 contains the Intermediate Representation (IR) types for Nebularr.
package v1

// HealthIssue represents a health check issue from an *arr app
type HealthIssue struct {
	// Source identifies the check that produced this issue (e.g., "IndexerRssCheck")
	Source string `json:"source"`

	// Type is the severity: error, warning, notice
	Type HealthIssueType `json:"type"`

	// Message is the human-readable description
	Message string `json:"message"`

	// WikiURL is a link to documentation about this issue
	WikiURL string `json:"wikiUrl,omitempty"`
}

// HealthIssueType represents the severity of a health issue
type HealthIssueType string

const (
	HealthIssueTypeError   HealthIssueType = "error"
	HealthIssueTypeWarning HealthIssueType = "warning"
	HealthIssueTypeNotice  HealthIssueType = "notice"
)

// HealthStatus represents the overall health of an app
type HealthStatus struct {
	// Healthy is true when there are no error-level issues
	Healthy bool `json:"healthy"`

	// Issues is the list of health issues
	Issues []HealthIssue `json:"issues,omitempty"`
}

// HasErrors returns true if there are any error-level issues
func (h *HealthStatus) HasErrors() bool {
	for _, issue := range h.Issues {
		if issue.Type == HealthIssueTypeError {
			return true
		}
	}
	return false
}

// HasWarnings returns true if there are any warning-level issues
func (h *HealthStatus) HasWarnings() bool {
	for _, issue := range h.Issues {
		if issue.Type == HealthIssueTypeWarning {
			return true
		}
	}
	return false
}

// IssueKey returns a unique key for an issue (for deduplication)
func (h *HealthIssue) IssueKey() string {
	return h.Source + ":" + h.Message
}
