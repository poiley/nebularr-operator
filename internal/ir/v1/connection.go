package v1

// ConnectionIR holds resolved connection details
type ConnectionIR struct {
	URL                string `json:"url"`
	APIKey             string `json:"apiKey"` // Resolved from secret or auto-discovery
	InsecureSkipVerify bool   `json:"insecureSkipVerify,omitempty"`
}
