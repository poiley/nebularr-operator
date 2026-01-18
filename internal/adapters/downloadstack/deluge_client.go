package downloadstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync/atomic"
)

// DelugeClientInterface defines the Deluge JSON-RPC API operations.
// This interface allows for mock implementations in tests.
type DelugeClientInterface interface {
	// Login authenticates with Deluge Web UI
	Login(ctx context.Context) error

	// TestConnection tests the connection to Deluge
	TestConnection(ctx context.Context) error

	// GetVersion gets Deluge version
	GetVersion(ctx context.Context) (string, error)

	// GetConfig gets Deluge configuration
	GetConfig(ctx context.Context) (*DelugeConfig, error)

	// SetConfig updates Deluge configuration
	SetConfig(ctx context.Context, config map[string]interface{}) error

	// GetLabels gets all labels (if label plugin is enabled)
	GetLabels(ctx context.Context) ([]string, error)

	// AddLabel adds a new label
	AddLabel(ctx context.Context, label string) error
}

// Ensure DelugeClient implements the interface
var _ DelugeClientInterface = (*DelugeClient)(nil)

// DelugeClient is a client for Deluge JSON-RPC API (via Web UI)
type DelugeClient struct {
	baseURL    string
	httpClient *http.Client
	password   string
	loggedIn   bool
	requestID  int64
}

// DelugeRPCRequest is the request structure for Deluge JSON-RPC
type DelugeRPCRequest struct {
	ID     int64         `json:"id"`
	Method string        `json:"method"`
	Params []interface{} `json:"params"`
}

// DelugeRPCResponse is the response structure for Deluge JSON-RPC
type DelugeRPCResponse struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  *DelugeRPCError `json:"error"`
}

// DelugeRPCError represents a Deluge RPC error
type DelugeRPCError struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// DelugeConfig contains Deluge daemon configuration
type DelugeConfig struct {
	// Directories
	DownloadLocation     string `json:"download_location"`
	MoveCompleted        bool   `json:"move_completed"`
	MoveCompletedPath    string `json:"move_completed_path"`
	AutoManagedDefault   bool   `json:"auto_managed_default"`
	CopyTorrentFile      bool   `json:"copy_torrent_file"`
	TorrentfilesLocation string `json:"torrentfiles_location"`

	// Speed limits (bytes/sec, -1 = unlimited)
	MaxDownloadSpeed           float64 `json:"max_download_speed"`
	MaxUploadSpeed             float64 `json:"max_upload_speed"`
	MaxDownloadSpeedPerTorrent float64 `json:"max_download_speed_per_torrent"`
	MaxUploadSpeedPerTorrent   float64 `json:"max_upload_speed_per_torrent"`

	// Connections
	MaxConnections           int `json:"max_connections_global"`
	MaxConnectionsPerTorrent int `json:"max_connections_per_torrent"`
	MaxUploadSlots           int `json:"max_upload_slots_global"`
	MaxUploadSlotsPerTorrent int `json:"max_upload_slots_per_torrent"`

	// Seeding
	StopSeedAtRatio    bool    `json:"stop_seed_at_ratio"`
	StopSeedRatio      float64 `json:"stop_seed_ratio"`
	RemoveAtRatio      bool    `json:"remove_seed_at_ratio"`
	ShareRatioLimit    float64 `json:"share_ratio_limit"`
	SeedTimeRatioLimit float64 `json:"seed_time_ratio_limit"`
	SeedTimeLimit      int     `json:"seed_time_limit"`

	// Queue
	MaxActiveDownloading          int  `json:"max_active_downloading"`
	MaxActiveSeeding              int  `json:"max_active_seeding"`
	MaxActiveLimit                int  `json:"max_active_limit"`
	QueueNewToTop                 bool `json:"queue_new_to_top"`
	StopSeedWhenShareRatioReached bool `json:"stop_seed_when_share_ratio_reached"`

	// Network
	ListenPorts        []int  `json:"listen_ports"`
	RandomPort         bool   `json:"random_port"`
	ListenInterface    string `json:"listen_interface"`
	OutgoingInterface  string `json:"outgoing_interface"`
	ListenUsesSysPorts bool   `json:"listen_use_sys_port"`
	ListenReusePorts   bool   `json:"listen_reuse_port"`

	// Protocol
	DHTEnabled    bool `json:"dht"`
	UPnPEnabled   bool `json:"upnp"`
	NATPMPEnabled bool `json:"natpmp"`
	LSDEnabled    bool `json:"lsd"`
	PEEnabled     bool `json:"pe_enabled"` // Protocol Encryption
	PEEncLevel    int  `json:"enc_level"`  // 0=handshake, 1=full, 2=either

	// Proxy (if configured)
	ProxyEnabled  bool   `json:"proxy_type"` // 0=none, 1=socksv4, 2=socksv5, 3=socksv5_auth, 4=http, 5=http_auth
	ProxyHostname string `json:"proxy_hostname"`
	ProxyPort     int    `json:"proxy_port"`
	ProxyUsername string `json:"proxy_username"`
	ProxyPassword string `json:"proxy_password"`

	// Misc
	AddPaused                 bool `json:"add_paused"`
	PreAllocateStorage        bool `json:"pre_allocate_storage"`
	PrioritizeFirstLastPieces bool `json:"prioritize_first_last_pieces"`
	SequentialDownload        bool `json:"sequential_download"`
	SuperSeeding              bool `json:"super_seeding"`
}

// NewDelugeClient creates a new Deluge JSON-RPC client
func NewDelugeClient(baseURL, password string) *DelugeClient {
	jar, _ := cookiejar.New(nil)
	return &DelugeClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		password: password,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Jar:     jar,
		},
	}
}

// nextID generates the next request ID
func (c *DelugeClient) nextID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}

// request makes a JSON-RPC request to Deluge
func (c *DelugeClient) request(ctx context.Context, method string, params ...interface{}) (*DelugeRPCResponse, error) {
	if params == nil {
		params = []interface{}{}
	}

	rpcReq := DelugeRPCRequest{
		ID:     c.nextID(),
		Method: method,
		Params: params,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var rpcResp DelugeRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	return &rpcResp, nil
}

// Login authenticates with Deluge Web UI
func (c *DelugeClient) Login(ctx context.Context) error {
	if c.loggedIn {
		return nil
	}

	resp, err := c.request(ctx, "auth.login", c.password)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	if !success {
		return fmt.Errorf("login failed: invalid password")
	}

	// Connect to the daemon
	// First, get available hosts
	hostsResp, err := c.request(ctx, "web.get_hosts")
	if err != nil {
		return fmt.Errorf("failed to get hosts: %w", err)
	}

	var hosts [][]interface{}
	if err := json.Unmarshal(hostsResp.Result, &hosts); err != nil {
		return fmt.Errorf("failed to parse hosts: %w", err)
	}

	// Connect to the first available host
	if len(hosts) > 0 {
		hostID, ok := hosts[0][0].(string)
		if ok {
			_, err = c.request(ctx, "web.connect", hostID)
			if err != nil {
				// Connection might already be established, continue
			}
		}
	}

	c.loggedIn = true
	return nil
}

// TestConnection tests the connection to Deluge
func (c *DelugeClient) TestConnection(ctx context.Context) error {
	if err := c.Login(ctx); err != nil {
		return err
	}

	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion gets Deluge version
func (c *DelugeClient) GetVersion(ctx context.Context) (string, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return "", err
	}

	resp, err := c.request(ctx, "daemon.info")
	if err != nil {
		return "", err
	}

	var version string
	if err := json.Unmarshal(resp.Result, &version); err != nil {
		return "", fmt.Errorf("failed to parse version: %w", err)
	}

	return version, nil
}

// GetConfig gets Deluge configuration
func (c *DelugeClient) GetConfig(ctx context.Context) (*DelugeConfig, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}

	resp, err := c.request(ctx, "core.get_config")
	if err != nil {
		return nil, err
	}

	var config DelugeConfig
	if err := json.Unmarshal(resp.Result, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SetConfig updates Deluge configuration
func (c *DelugeClient) SetConfig(ctx context.Context, config map[string]interface{}) error {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return err
	}

	_, err := c.request(ctx, "core.set_config", config)
	return err
}

// GetLabels gets all labels (if label plugin is enabled)
func (c *DelugeClient) GetLabels(ctx context.Context) ([]string, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}

	resp, err := c.request(ctx, "label.get_labels")
	if err != nil {
		// Label plugin might not be enabled
		return nil, fmt.Errorf("label plugin might not be enabled: %w", err)
	}

	var labels []string
	if err := json.Unmarshal(resp.Result, &labels); err != nil {
		return nil, fmt.Errorf("failed to parse labels: %w", err)
	}

	return labels, nil
}

// AddLabel adds a new label
func (c *DelugeClient) AddLabel(ctx context.Context, label string) error {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return err
	}

	_, err := c.request(ctx, "label.add", label)
	return err
}

// ensureLoggedIn ensures the client is logged in
func (c *DelugeClient) ensureLoggedIn(ctx context.Context) error {
	if !c.loggedIn {
		return c.Login(ctx)
	}

	// Check if session is still valid
	resp, err := c.request(ctx, "auth.check_session")
	if err != nil {
		c.loggedIn = false
		return c.Login(ctx)
	}

	var valid bool
	if err := json.Unmarshal(resp.Result, &valid); err != nil || !valid {
		c.loggedIn = false
		return c.Login(ctx)
	}

	return nil
}

// GetSessionStatus gets current session status including transfer speeds
func (c *DelugeClient) GetSessionStatus(ctx context.Context) (map[string]interface{}, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}

	keys := []string{
		"upload_rate", "download_rate",
		"num_peers", "num_connections",
		"payload_upload_rate", "payload_download_rate",
		"total_upload", "total_download",
		"dht_nodes",
	}

	resp, err := c.request(ctx, "core.get_session_status", keys)
	if err != nil {
		return nil, err
	}

	var status map[string]interface{}
	if err := json.Unmarshal(resp.Result, &status); err != nil {
		return nil, fmt.Errorf("failed to parse session status: %w", err)
	}

	return status, nil
}

// PauseAllTorrents pauses all torrents
func (c *DelugeClient) PauseAllTorrents(ctx context.Context) error {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return err
	}

	_, err := c.request(ctx, "core.pause_session")
	return err
}

// ResumeAllTorrents resumes all torrents
func (c *DelugeClient) ResumeAllTorrents(ctx context.Context) error {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return err
	}

	_, err := c.request(ctx, "core.resume_session")
	return err
}

// GetEnabledPlugins gets list of enabled plugins
func (c *DelugeClient) GetEnabledPlugins(ctx context.Context) ([]string, error) {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}

	resp, err := c.request(ctx, "core.get_enabled_plugins")
	if err != nil {
		return nil, err
	}

	var plugins []string
	if err := json.Unmarshal(resp.Result, &plugins); err != nil {
		return nil, fmt.Errorf("failed to parse plugins: %w", err)
	}

	return plugins, nil
}

// EnablePlugin enables a plugin
func (c *DelugeClient) EnablePlugin(ctx context.Context, plugin string) error {
	if err := c.ensureLoggedIn(ctx); err != nil {
		return err
	}

	_, err := c.request(ctx, "core.enable_plugin", plugin)
	return err
}
