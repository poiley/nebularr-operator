package downloadstack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
)

// NZBGetClientInterface defines the NZBGet JSON-RPC API operations.
// This interface allows for mock implementations in tests.
type NZBGetClientInterface interface {
	// TestConnection tests the connection to NZBGet
	TestConnection(ctx context.Context) error

	// GetVersion gets NZBGet version
	GetVersion(ctx context.Context) (string, error)

	// GetConfig gets NZBGet configuration
	GetConfig(ctx context.Context) ([]NZBGetConfigItem, error)

	// SetConfig updates NZBGet configuration
	SetConfig(ctx context.Context, name, value string) error

	// GetStatus gets current download status
	GetStatus(ctx context.Context) (*NZBGetStatus, error)

	// GetGroups gets the download queue
	GetGroups(ctx context.Context) ([]NZBGetGroup, error)

	// GetHistory gets download history
	GetHistory(ctx context.Context) ([]NZBGetHistoryItem, error)

	// PauseDownload pauses downloading
	PauseDownload(ctx context.Context) error

	// ResumeDownload resumes downloading
	ResumeDownload(ctx context.Context) error

	// SetDownloadRate sets download rate limit (KB/s, 0 = unlimited)
	SetDownloadRate(ctx context.Context, rate int) error

	// Reload reloads configuration
	Reload(ctx context.Context) error
}

// Ensure NZBGetClient implements the interface
var _ NZBGetClientInterface = (*NZBGetClient)(nil)

// NZBGetClient is a client for NZBGet JSON-RPC API
type NZBGetClient struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	requestID  int64
}

// NZBGetRPCRequest is the request structure for NZBGet JSON-RPC
type NZBGetRPCRequest struct {
	Method  string        `json:"method"`
	Params  []interface{} `json:"params,omitempty"`
	ID      int64         `json:"id"`
	Version string        `json:"jsonrpc"`
}

// NZBGetRPCResponse is the response structure for NZBGet JSON-RPC
type NZBGetRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *NZBGetRPCError `json:"error"`
	ID     int64           `json:"id"`
}

// NZBGetRPCError represents a JSON-RPC error
type NZBGetRPCError struct {
	Name    string `json:"name"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NZBGetConfigItem represents a configuration item
type NZBGetConfigItem struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// NZBGetStatus represents the current status
type NZBGetStatus struct {
	RemainingSizeLo     int64                `json:"RemainingSizeLo"`
	RemainingSizeHi     int64                `json:"RemainingSizeHi"`
	RemainingSizeMB     int64                `json:"RemainingSizeMB"`
	ForcedSizeLo        int64                `json:"ForcedSizeLo"`
	ForcedSizeHi        int64                `json:"ForcedSizeHi"`
	ForcedSizeMB        int64                `json:"ForcedSizeMB"`
	DownloadedSizeLo    int64                `json:"DownloadedSizeLo"`
	DownloadedSizeHi    int64                `json:"DownloadedSizeHi"`
	DownloadedSizeMB    int64                `json:"DownloadedSizeMB"`
	ArticleCacheLo      int64                `json:"ArticleCacheLo"`
	ArticleCacheHi      int64                `json:"ArticleCacheHi"`
	ArticleCacheMB      int64                `json:"ArticleCacheMB"`
	DownloadRate        int64                `json:"DownloadRate"`
	AverageDownloadRate int64                `json:"AverageDownloadRate"`
	DownloadLimit       int64                `json:"DownloadLimit"`
	ThreadCount         int                  `json:"ThreadCount"`
	PostJobCount        int                  `json:"PostJobCount"`
	UrlCount            int                  `json:"UrlCount"`
	UpTimeSec           int64                `json:"UpTimeSec"`
	DownloadTimeSec     int64                `json:"DownloadTimeSec"`
	ServerPaused        bool                 `json:"ServerPaused"`
	DownloadPaused      bool                 `json:"DownloadPaused"`
	Download2Paused     bool                 `json:"Download2Paused"`
	ServerStandBy       bool                 `json:"ServerStandBy"`
	PostPaused          bool                 `json:"PostPaused"`
	ScanPaused          bool                 `json:"ScanPaused"`
	QuotaReached        bool                 `json:"QuotaReached"`
	FreeDiskSpaceLo     int64                `json:"FreeDiskSpaceLo"`
	FreeDiskSpaceHi     int64                `json:"FreeDiskSpaceHi"`
	FreeDiskSpaceMB     int64                `json:"FreeDiskSpaceMB"`
	ServerTime          int64                `json:"ServerTime"`
	ResumeTime          int64                `json:"ResumeTime"`
	FeedActive          bool                 `json:"FeedActive"`
	QueueScriptCount    int                  `json:"QueueScriptCount"`
	NewsServers         []NZBGetServerStatus `json:"NewsServers"`
}

// NZBGetServerStatus represents a news server status
type NZBGetServerStatus struct {
	ID     int  `json:"ID"`
	Active bool `json:"Active"`
}

// NZBGetGroup represents a download group (queue item)
type NZBGetGroup struct {
	NZBID             int                  `json:"NZBID"`
	NZBName           string               `json:"NZBName"`
	NZBNicename       string               `json:"NZBNicename"`
	Kind              string               `json:"Kind"`
	URL               string               `json:"URL"`
	NZBFilename       string               `json:"NZBFilename"`
	DestDir           string               `json:"DestDir"`
	FinalDir          string               `json:"FinalDir"`
	Category          string               `json:"Category"`
	ParStatus         string               `json:"ParStatus"`
	UnpackStatus      string               `json:"UnpackStatus"`
	MoveStatus        string               `json:"MoveStatus"`
	ScriptStatus      string               `json:"ScriptStatus"`
	DeleteStatus      string               `json:"DeleteStatus"`
	MarkStatus        string               `json:"MarkStatus"`
	UrlStatus         string               `json:"UrlStatus"`
	FileSizeLo        int64                `json:"FileSizeLo"`
	FileSizeHi        int64                `json:"FileSizeHi"`
	FileSizeMB        int64                `json:"FileSizeMB"`
	FileCount         int                  `json:"FileCount"`
	MinPostTime       int64                `json:"MinPostTime"`
	MaxPostTime       int64                `json:"MaxPostTime"`
	TotalArticles     int                  `json:"TotalArticles"`
	SuccessArticles   int                  `json:"SuccessArticles"`
	FailedArticles    int                  `json:"FailedArticles"`
	Health            int                  `json:"Health"`
	CriticalHealth    int                  `json:"CriticalHealth"`
	DupeKey           string               `json:"DupeKey"`
	DupeScore         int                  `json:"DupeScore"`
	DupeMode          string               `json:"DupeMode"`
	Deleted           bool                 `json:"Deleted"`
	DownloadedSizeLo  int64                `json:"DownloadedSizeLo"`
	DownloadedSizeHi  int64                `json:"DownloadedSizeHi"`
	DownloadedSizeMB  int64                `json:"DownloadedSizeMB"`
	DownloadTimeSec   int                  `json:"DownloadTimeSec"`
	PostTotalTimeSec  int                  `json:"PostTotalTimeSec"`
	ParTimeSec        int                  `json:"ParTimeSec"`
	RepairTimeSec     int                  `json:"RepairTimeSec"`
	UnpackTimeSec     int                  `json:"UnpackTimeSec"`
	MessageCount      int                  `json:"MessageCount"`
	ExtraParBlocks    int                  `json:"ExtraParBlocks"`
	Parameters        []NZBGetParameter    `json:"Parameters"`
	ScriptStatuses    []NZBGetScriptStatus `json:"ScriptStatuses"`
	ServerStats       []NZBGetServerStat   `json:"ServerStats"`
	PostInfoText      string               `json:"PostInfoText"`
	PostStageProgress int                  `json:"PostStageProgress"`
	PostStageTimeSec  int                  `json:"PostStageTimeSec"`
	Log               []NZBGetLogEntry     `json:"Log"`
}

// NZBGetParameter represents a download parameter
type NZBGetParameter struct {
	Name  string `json:"Name"`
	Value string `json:"Value"`
}

// NZBGetScriptStatus represents a script status
type NZBGetScriptStatus struct {
	Name   string `json:"Name"`
	Status string `json:"Status"`
}

// NZBGetServerStat represents server statistics
type NZBGetServerStat struct {
	ServerID        int `json:"ServerID"`
	SuccessArticles int `json:"SuccessArticles"`
	FailedArticles  int `json:"FailedArticles"`
}

// NZBGetLogEntry represents a log entry
type NZBGetLogEntry struct {
	ID   int    `json:"ID"`
	Kind string `json:"Kind"`
	Time int64  `json:"Time"`
	Text string `json:"Text"`
}

// NZBGetHistoryItem represents a history item
type NZBGetHistoryItem struct {
	NZBID            int                  `json:"NZBID"`
	NZBName          string               `json:"NZBName"`
	NZBNicename      string               `json:"NZBNicename"`
	Kind             string               `json:"Kind"`
	URL              string               `json:"URL"`
	NZBFilename      string               `json:"NZBFilename"`
	DestDir          string               `json:"DestDir"`
	FinalDir         string               `json:"FinalDir"`
	Category         string               `json:"Category"`
	ParStatus        string               `json:"ParStatus"`
	UnpackStatus     string               `json:"UnpackStatus"`
	MoveStatus       string               `json:"MoveStatus"`
	ScriptStatus     string               `json:"ScriptStatus"`
	DeleteStatus     string               `json:"DeleteStatus"`
	MarkStatus       string               `json:"MarkStatus"`
	UrlStatus        string               `json:"UrlStatus"`
	FileSizeLo       int64                `json:"FileSizeLo"`
	FileSizeHi       int64                `json:"FileSizeHi"`
	FileSizeMB       int64                `json:"FileSizeMB"`
	FileCount        int                  `json:"FileCount"`
	TotalArticles    int                  `json:"TotalArticles"`
	SuccessArticles  int                  `json:"SuccessArticles"`
	FailedArticles   int                  `json:"FailedArticles"`
	Health           int                  `json:"Health"`
	CriticalHealth   int                  `json:"CriticalHealth"`
	DupeKey          string               `json:"DupeKey"`
	DupeScore        int                  `json:"DupeScore"`
	DupeMode         string               `json:"DupeMode"`
	Deleted          bool                 `json:"Deleted"`
	DownloadedSizeLo int64                `json:"DownloadedSizeLo"`
	DownloadedSizeHi int64                `json:"DownloadedSizeHi"`
	DownloadedSizeMB int64                `json:"DownloadedSizeMB"`
	DownloadTimeSec  int                  `json:"DownloadTimeSec"`
	PostTotalTimeSec int                  `json:"PostTotalTimeSec"`
	ParTimeSec       int                  `json:"ParTimeSec"`
	RepairTimeSec    int                  `json:"RepairTimeSec"`
	UnpackTimeSec    int                  `json:"UnpackTimeSec"`
	MessageCount     int                  `json:"MessageCount"`
	ExtraParBlocks   int                  `json:"ExtraParBlocks"`
	Parameters       []NZBGetParameter    `json:"Parameters"`
	ScriptStatuses   []NZBGetScriptStatus `json:"ScriptStatuses"`
	ServerStats      []NZBGetServerStat   `json:"ServerStats"`
	Status           string               `json:"Status"` // SUCCESS, FAILURE, etc.
	HistoryTime      int64                `json:"HistoryTime"`
}

// NewNZBGetClient creates a new NZBGet JSON-RPC client
func NewNZBGetClient(baseURL, username, password string) *NZBGetClient {
	return &NZBGetClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// nextID generates the next request ID
func (c *NZBGetClient) nextID() int64 {
	return atomic.AddInt64(&c.requestID, 1)
}

// request makes a JSON-RPC request to NZBGet
func (c *NZBGetClient) request(ctx context.Context, method string, params ...interface{}) (*NZBGetRPCResponse, error) {
	rpcReq := NZBGetRPCRequest{
		Method:  method,
		Params:  params,
		ID:      c.nextID(),
		Version: "2.0",
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// NZBGet uses /jsonrpc endpoint with basic auth
	reqURL := fmt.Sprintf("%s/jsonrpc", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed: invalid credentials")
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var rpcResp NZBGetRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s (code %d)", rpcResp.Error.Message, rpcResp.Error.Code)
	}

	return &rpcResp, nil
}

// TestConnection tests the connection to NZBGet
func (c *NZBGetClient) TestConnection(ctx context.Context) error {
	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion gets NZBGet version
func (c *NZBGetClient) GetVersion(ctx context.Context) (string, error) {
	resp, err := c.request(ctx, "version")
	if err != nil {
		return "", err
	}

	var version string
	if err := json.Unmarshal(resp.Result, &version); err != nil {
		return "", fmt.Errorf("failed to parse version: %w", err)
	}

	return version, nil
}

// GetConfig gets NZBGet configuration
func (c *NZBGetClient) GetConfig(ctx context.Context) ([]NZBGetConfigItem, error) {
	resp, err := c.request(ctx, "config")
	if err != nil {
		return nil, err
	}

	var config []NZBGetConfigItem
	if err := json.Unmarshal(resp.Result, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return config, nil
}

// SetConfig updates a NZBGet configuration value
func (c *NZBGetClient) SetConfig(ctx context.Context, name, value string) error {
	resp, err := c.request(ctx, "saveconfig",
		[]NZBGetConfigItem{{Name: name, Value: value}},
	)
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("saveconfig failed")
	}

	return nil
}

// GetStatus gets current download status
func (c *NZBGetClient) GetStatus(ctx context.Context) (*NZBGetStatus, error) {
	resp, err := c.request(ctx, "status")
	if err != nil {
		return nil, err
	}

	var status NZBGetStatus
	if err := json.Unmarshal(resp.Result, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status: %w", err)
	}

	return &status, nil
}

// GetGroups gets the download queue
func (c *NZBGetClient) GetGroups(ctx context.Context) ([]NZBGetGroup, error) {
	resp, err := c.request(ctx, "listgroups")
	if err != nil {
		return nil, err
	}

	var groups []NZBGetGroup
	if err := json.Unmarshal(resp.Result, &groups); err != nil {
		return nil, fmt.Errorf("failed to parse groups: %w", err)
	}

	return groups, nil
}

// GetHistory gets download history
func (c *NZBGetClient) GetHistory(ctx context.Context) ([]NZBGetHistoryItem, error) {
	// hidden parameter: true to include hidden items
	resp, err := c.request(ctx, "history", false)
	if err != nil {
		return nil, err
	}

	var history []NZBGetHistoryItem
	if err := json.Unmarshal(resp.Result, &history); err != nil {
		return nil, fmt.Errorf("failed to parse history: %w", err)
	}

	return history, nil
}

// PauseDownload pauses downloading
func (c *NZBGetClient) PauseDownload(ctx context.Context) error {
	resp, err := c.request(ctx, "pausedownload")
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("pausedownload failed")
	}

	return nil
}

// ResumeDownload resumes downloading
func (c *NZBGetClient) ResumeDownload(ctx context.Context) error {
	resp, err := c.request(ctx, "resumedownload")
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("resumedownload failed")
	}

	return nil
}

// SetDownloadRate sets download rate limit (KB/s, 0 = unlimited)
func (c *NZBGetClient) SetDownloadRate(ctx context.Context, rate int) error {
	resp, err := c.request(ctx, "rate", rate)
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("rate failed")
	}

	return nil
}

// Reload reloads configuration
func (c *NZBGetClient) Reload(ctx context.Context) error {
	resp, err := c.request(ctx, "reload")
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("reload failed")
	}

	return nil
}

// Scan scans incoming directory for new NZB files
func (c *NZBGetClient) Scan(ctx context.Context) error {
	resp, err := c.request(ctx, "scan")
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("scan failed")
	}

	return nil
}

// Shutdown shuts down NZBGet
func (c *NZBGetClient) Shutdown(ctx context.Context) error {
	resp, err := c.request(ctx, "shutdown")
	if err != nil {
		return err
	}

	var success bool
	if err := json.Unmarshal(resp.Result, &success); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if !success {
		return fmt.Errorf("shutdown failed")
	}

	return nil
}

// GetLog gets the server log
func (c *NZBGetClient) GetLog(ctx context.Context, start, count int) ([]NZBGetLogEntry, error) {
	resp, err := c.request(ctx, "log", start, count)
	if err != nil {
		return nil, err
	}

	var logs []NZBGetLogEntry
	if err := json.Unmarshal(resp.Result, &logs); err != nil {
		return nil, fmt.Errorf("failed to parse log: %w", err)
	}

	return logs, nil
}
