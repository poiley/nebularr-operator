package downloadstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// SABnzbdClientInterface defines the SABnzbd API operations.
// This interface allows for mock implementations in tests.
type SABnzbdClientInterface interface {
	// TestConnection tests the connection to SABnzbd
	TestConnection(ctx context.Context) error

	// GetVersion gets SABnzbd version
	GetVersion(ctx context.Context) (string, error)

	// GetConfig gets SABnzbd configuration
	GetConfig(ctx context.Context) (*SABnzbdConfig, error)

	// SetConfig updates SABnzbd configuration
	SetConfig(ctx context.Context, section, keyword, value string) error

	// GetQueue gets the current download queue
	GetQueue(ctx context.Context) (*SABnzbdQueue, error)

	// GetHistory gets download history
	GetHistory(ctx context.Context) (*SABnzbdHistory, error)

	// Pause pauses downloading
	Pause(ctx context.Context) error

	// Resume resumes downloading
	Resume(ctx context.Context) error

	// SetSpeedLimit sets download speed limit (KB/s, 0 = unlimited)
	SetSpeedLimit(ctx context.Context, limit int) error
}

// Ensure SABnzbdClient implements the interface
var _ SABnzbdClientInterface = (*SABnzbdClient)(nil)

// SABnzbdClient is a client for SABnzbd API
type SABnzbdClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// SABnzbdConfig contains SABnzbd configuration sections
type SABnzbdConfig struct {
	Misc       SABnzbdMiscConfig     `json:"misc"`
	Servers    []SABnzbdServerConfig `json:"servers"`
	Categories []SABnzbdCategory     `json:"categories"`
}

// SABnzbdMiscConfig contains misc configuration
type SABnzbdMiscConfig struct {
	// Directories
	DownloadDir   string `json:"download_dir"`
	CompleteDir   string `json:"complete_dir"`
	IncompleteDir string `json:"incomplete_dir,omitempty"`
	ScriptDir     string `json:"script_dir,omitempty"`
	NzbBackupDir  string `json:"nzb_backup_dir,omitempty"`
	AdminDir      string `json:"admin_dir,omitempty"`
	LogDir        string `json:"log_dir,omitempty"`

	// Speed
	Bandwidth     string `json:"bandwidth_max,omitempty"`  // Max bandwidth (e.g., "100M")
	BandwidthPerc int    `json:"bandwidth_perc,omitempty"` // Percentage of max

	// Queue
	QueueCompleteAction string `json:"queue_complete,omitempty"`
	PreCheck            bool   `json:"pre_check,omitempty"`

	// Processing
	Unpack        bool   `json:"unpack,omitempty"`
	UnpackReplace bool   `json:"unpack_replace,omitempty"`
	ScriptArgs    string `json:"script,omitempty"`
	DirectUnpack  bool   `json:"direct_unpack,omitempty"`

	// Sorting
	EnableMeta bool `json:"enable_meta,omitempty"`

	// Safety
	SafeMode   bool `json:"safe_mode,omitempty"`
	NoAssembly bool `json:"no_assembly,omitempty"`
	ParOption  int  `json:"par_option,omitempty"` // 0=normal, 1=force, 2=auto

	// Cleanup
	HistoryRetention string `json:"history_retention,omitempty"`

	// Notifications
	EmailServer string `json:"email_server,omitempty"`
	EmailTo     string `json:"email_to,omitempty"`
	EmailFrom   string `json:"email_from,omitempty"`
}

// SABnzbdServerConfig represents a news server configuration
type SABnzbdServerConfig struct {
	Name        string `json:"name"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Username    string `json:"username,omitempty"`
	Password    string `json:"password,omitempty"`
	Connections int    `json:"connections"`
	SSL         bool   `json:"ssl"`
	SSLVerify   int    `json:"ssl_verify"` // 0=none, 1=verify, 2=strict
	Optional    bool   `json:"optional"`
	Retention   int    `json:"retention,omitempty"` // Days
	Enable      bool   `json:"enable"`
	Priority    int    `json:"priority"`
}

// SABnzbdCategory represents a download category
type SABnzbdCategory struct {
	Name     string `json:"name"`
	Order    int    `json:"order"`
	Dir      string `json:"dir,omitempty"`
	Script   string `json:"script,omitempty"`
	Priority int    `json:"priority"`
	PostProc int    `json:"pp,omitempty"` // Post-processing: 0=skip, 1=repair, 2=repair+unpack, 3=repair+unpack+delete
}

// SABnzbdQueue represents the download queue
type SABnzbdQueue struct {
	Status         string        `json:"status"`
	Paused         bool          `json:"paused"`
	PausedAll      bool          `json:"paused_all"`
	Speed          string        `json:"speed"`
	SpeedLimit     string        `json:"speedlimit"`
	SpeedLimitAbs  string        `json:"speedlimit_abs"`
	SizeLeft       string        `json:"sizeleft"`
	TimeLeft       string        `json:"timeleft"`
	NoOfSlots      int           `json:"noofslots"`
	NoOfSlotsTotal int           `json:"noofslots_total"`
	Slots          []SABnzbdSlot `json:"slots"`
}

// SABnzbdSlot represents a queue slot (download item)
type SABnzbdSlot struct {
	ID         string `json:"nzo_id"`
	Filename   string `json:"filename"`
	Status     string `json:"status"`
	Size       string `json:"size"`
	SizeLeft   string `json:"sizeleft"`
	Percentage string `json:"percentage"`
	Category   string `json:"cat"`
	Priority   string `json:"priority"`
	TimeLeft   string `json:"timeleft"`
}

// SABnzbdHistory represents download history
type SABnzbdHistory struct {
	NoOfSlots int                  `json:"noofslots"`
	Slots     []SABnzbdHistorySlot `json:"slots"`
}

// SABnzbdHistorySlot represents a history item
type SABnzbdHistorySlot struct {
	ID          string `json:"nzo_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Size        string `json:"size"`
	Category    string `json:"category"`
	Completed   int64  `json:"completed"` // Unix timestamp
	FailMessage string `json:"fail_message,omitempty"`
}

// SABnzbdResponse is a generic API response
type SABnzbdResponse struct {
	Status bool   `json:"status"`
	Error  string `json:"error,omitempty"`
}

// NewSABnzbdClient creates a new SABnzbd API client
func NewSABnzbdClient(baseURL, apiKey string) *SABnzbdClient {
	return &SABnzbdClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// request makes an API request to SABnzbd
func (c *SABnzbdClient) request(ctx context.Context, mode string, params url.Values) ([]byte, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("apikey", c.apiKey)
	params.Set("mode", mode)
	params.Set("output", "json")

	reqURL := fmt.Sprintf("%s/api?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(body))
	}

	return io.ReadAll(resp.Body)
}

// TestConnection tests the connection to SABnzbd
func (c *SABnzbdClient) TestConnection(ctx context.Context) error {
	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion gets SABnzbd version
func (c *SABnzbdClient) GetVersion(ctx context.Context) (string, error) {
	body, err := c.request(ctx, "version", nil)
	if err != nil {
		return "", err
	}

	var result struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse version: %w", err)
	}

	return result.Version, nil
}

// GetConfig gets SABnzbd configuration
func (c *SABnzbdClient) GetConfig(ctx context.Context) (*SABnzbdConfig, error) {
	body, err := c.request(ctx, "get_config", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Config SABnzbdConfig `json:"config"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &result.Config, nil
}

// SetConfig updates a SABnzbd configuration value
func (c *SABnzbdClient) SetConfig(ctx context.Context, section, keyword, value string) error {
	params := url.Values{}
	params.Set("section", section)
	params.Set("keyword", keyword)
	params.Set("value", value)

	body, err := c.request(ctx, "set_config", params)
	if err != nil {
		return err
	}

	var result SABnzbdResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !result.Status {
		return fmt.Errorf("set_config failed: %s", result.Error)
	}

	return nil
}

// GetQueue gets the current download queue
func (c *SABnzbdClient) GetQueue(ctx context.Context) (*SABnzbdQueue, error) {
	body, err := c.request(ctx, "queue", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Queue SABnzbdQueue `json:"queue"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse queue: %w", err)
	}

	return &result.Queue, nil
}

// GetHistory gets download history
func (c *SABnzbdClient) GetHistory(ctx context.Context) (*SABnzbdHistory, error) {
	body, err := c.request(ctx, "history", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		History SABnzbdHistory `json:"history"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse history: %w", err)
	}

	return &result.History, nil
}

// Pause pauses downloading
func (c *SABnzbdClient) Pause(ctx context.Context) error {
	_, err := c.request(ctx, "pause", nil)
	return err
}

// Resume resumes downloading
func (c *SABnzbdClient) Resume(ctx context.Context) error {
	_, err := c.request(ctx, "resume", nil)
	return err
}

// SetSpeedLimit sets download speed limit (KB/s, 0 = unlimited)
func (c *SABnzbdClient) SetSpeedLimit(ctx context.Context, limit int) error {
	params := url.Values{}
	params.Set("value", fmt.Sprintf("%d", limit))

	_, err := c.request(ctx, "config", params)
	return err
}

// Restart restarts SABnzbd
func (c *SABnzbdClient) Restart(ctx context.Context) error {
	_, err := c.request(ctx, "restart", nil)
	return err
}

// GetServerStats gets server statistics
func (c *SABnzbdClient) GetServerStats(ctx context.Context) (map[string]interface{}, error) {
	body, err := c.request(ctx, "server_stats", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse server stats: %w", err)
	}

	return result, nil
}

// GetWarnings gets current warnings
func (c *SABnzbdClient) GetWarnings(ctx context.Context) ([]string, error) {
	body, err := c.request(ctx, "warnings", nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse warnings: %w", err)
	}

	return result.Warnings, nil
}
