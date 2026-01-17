package v1

// AuthenticationIR represents authentication configuration
type AuthenticationIR struct {
	// Method: none, forms, external
	Method string `json:"method"`

	// Username for forms authentication
	Username string `json:"username,omitempty"`

	// Password for forms authentication (only set on initial setup)
	Password string `json:"password,omitempty"`

	// AuthenticationRequired: enabled, disabledForLocalAddresses
	AuthenticationRequired string `json:"authenticationRequired"`
}
