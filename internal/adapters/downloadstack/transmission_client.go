package downloadstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// SessionHeader is the header name for Transmission CSRF protection
	SessionHeader = "X-Transmission-Session-Id"

	// MaxRetries is the maximum number of retry attempts
	MaxRetries = 3

	// DefaultTimeout is the default HTTP timeout
	DefaultTimeout = 30 * time.Second
)

// TransmissionClient is a client for Transmission RPC API
type TransmissionClient struct {
	baseURL    string
	rpcURL     string
	httpClient *http.Client
	sessionID  string
	username   string
	password   string
}

// TransmissionRPCRequest is the request structure for Transmission RPC
type TransmissionRPCRequest struct {
	Method    string                 `json:"method"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Tag       int                    `json:"tag,omitempty"`
}

// TransmissionRPCResponse is the response structure for Transmission RPC
type TransmissionRPCResponse struct {
	Result    string                 `json:"result"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Tag       int                    `json:"tag,omitempty"`
}

// TransmissionSession contains Transmission session/settings information
type TransmissionSession struct {
	Version string `json:"version"`

	// Speed limits
	SpeedLimitDown        int  `json:"speed-limit-down"`
	SpeedLimitDownEnabled bool `json:"speed-limit-down-enabled"`
	SpeedLimitUp          int  `json:"speed-limit-up"`
	SpeedLimitUpEnabled   bool `json:"speed-limit-up-enabled"`

	// Alt-speed (turtle mode)
	AltSpeedEnabled     bool `json:"alt-speed-enabled"`
	AltSpeedDown        int  `json:"alt-speed-down"`
	AltSpeedUp          int  `json:"alt-speed-up"`
	AltSpeedTimeEnabled bool `json:"alt-speed-time-enabled"`
	AltSpeedTimeBegin   int  `json:"alt-speed-time-begin"`
	AltSpeedTimeEnd     int  `json:"alt-speed-time-end"`
	AltSpeedTimeDay     int  `json:"alt-speed-time-day"`

	// Directories
	DownloadDir          string `json:"download-dir"`
	IncompleteDirEnabled bool   `json:"incomplete-dir-enabled"`
	IncompleteDir        string `json:"incomplete-dir"`

	// Seeding
	SeedRatioLimit          float64 `json:"seedRatioLimit"`
	SeedRatioLimited        bool    `json:"seedRatioLimited"`
	IdleSeedingLimit        int     `json:"idle-seeding-limit"`
	IdleSeedingLimitEnabled bool    `json:"idle-seeding-limit-enabled"`

	// Queue
	DownloadQueueSize    int  `json:"download-queue-size"`
	DownloadQueueEnabled bool `json:"download-queue-enabled"`
	SeedQueueSize        int  `json:"seed-queue-size"`
	SeedQueueEnabled     bool `json:"seed-queue-enabled"`
	QueueStalledEnabled  bool `json:"queue-stalled-enabled"`
	QueueStalledMinutes  int  `json:"queue-stalled-minutes"`

	// Peers
	PeerLimitGlobal       int  `json:"peer-limit-global"`
	PeerLimitPerTorrent   int  `json:"peer-limit-per-torrent"`
	PeerPort              int  `json:"peer-port"`
	PeerPortRandomOnStart bool `json:"peer-port-random-on-start"`
	PortForwardingEnabled bool `json:"port-forwarding-enabled"`

	// Security
	Encryption string `json:"encryption"`
	PexEnabled bool   `json:"pex-enabled"`
	DhtEnabled bool   `json:"dht-enabled"`
	LpdEnabled bool   `json:"lpd-enabled"`
	UtpEnabled bool   `json:"utp-enabled"`

	// Blocklist
	BlocklistEnabled bool   `json:"blocklist-enabled"`
	BlocklistURL     string `json:"blocklist-url"`
	BlocklistSize    int    `json:"blocklist-size"`
}

// NewTransmissionClient creates a new Transmission RPC client
func NewTransmissionClient(baseURL, username, password string) *TransmissionClient {
	return &TransmissionClient{
		baseURL:  baseURL,
		rpcURL:   baseURL + "/transmission/rpc",
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// getSessionID gets a new session ID from Transmission
func (c *TransmissionClient) getSessionID(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer([]byte(`{"method":"session-get"}`)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 409 is expected - it provides the session ID
	if resp.StatusCode == 409 {
		c.sessionID = resp.Header.Get(SessionHeader)
		if c.sessionID == "" {
			return fmt.Errorf("no session ID in 409 response")
		}
		return nil
	}

	// 200 means we already have a valid session
	if resp.StatusCode == 200 {
		c.sessionID = resp.Header.Get(SessionHeader)
		if c.sessionID == "" {
			return fmt.Errorf("successful response but no session ID")
		}
		return nil
	}

	return fmt.Errorf("unexpected status code getting session: %d", resp.StatusCode)
}

// request makes an RPC request to Transmission with retry logic
func (c *TransmissionClient) request(ctx context.Context, method string, arguments map[string]interface{}) (*TransmissionRPCResponse, error) {
	rpcReq := TransmissionRPCRequest{
		Method:    method,
		Arguments: arguments,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		// Get session ID if we don't have one
		if c.sessionID == "" {
			if err := c.getSessionID(ctx); err != nil {
				return nil, fmt.Errorf("failed to get session ID: %w", err)
			}
		}

		req, err := http.NewRequestWithContext(ctx, "POST", c.rpcURL, bytes.NewBuffer(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(SessionHeader, c.sessionID)
		if c.username != "" {
			req.SetBasicAuth(c.username, c.password)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < MaxRetries {
				time.Sleep(time.Duration(1<<attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("request failed after retries: %w", err)
		}
		defer resp.Body.Close()

		// Session expired - get new one and retry
		if resp.StatusCode == 409 {
			c.sessionID = resp.Header.Get(SessionHeader)
			if c.sessionID == "" {
				return nil, fmt.Errorf("session expired but no new ID provided")
			}
			continue
		}

		if resp.StatusCode == 401 {
			return nil, fmt.Errorf("unauthorized - invalid credentials")
		}

		if resp.StatusCode != 200 {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(bodyBytes))
		}

		var rpcResp TransmissionRPCResponse
		if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}

		if rpcResp.Result != "success" {
			return nil, fmt.Errorf("RPC error: %s", rpcResp.Result)
		}

		return &rpcResp, nil
	}

	return nil, fmt.Errorf("request failed - max retries exceeded")
}

// TestConnection tests the connection to Transmission
func (c *TransmissionClient) TestConnection(ctx context.Context) error {
	_, err := c.GetSession(ctx)
	return err
}

// GetSession gets session/settings information
func (c *TransmissionClient) GetSession(ctx context.Context) (*TransmissionSession, error) {
	resp, err := c.request(ctx, "session-get", nil)
	if err != nil {
		return nil, err
	}

	// Convert arguments map to TransmissionSession struct
	argBytes, err := json.Marshal(resp.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal arguments: %w", err)
	}

	var session TransmissionSession
	if err := json.Unmarshal(argBytes, &session); err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// SetSession updates session settings
func (c *TransmissionClient) SetSession(ctx context.Context, settings map[string]interface{}) error {
	_, err := c.request(ctx, "session-set", settings)
	return err
}

// GetSessionStats gets session statistics
func (c *TransmissionClient) GetSessionStats(ctx context.Context) (map[string]interface{}, error) {
	resp, err := c.request(ctx, "session-stats", nil)
	if err != nil {
		return nil, err
	}
	return resp.Arguments, nil
}

// UpdateBlocklist updates the blocklist
func (c *TransmissionClient) UpdateBlocklist(ctx context.Context) error {
	_, err := c.request(ctx, "blocklist-update", nil)
	return err
}
