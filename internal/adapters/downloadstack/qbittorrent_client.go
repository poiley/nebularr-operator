package downloadstack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
)

// QBittorrentClientInterface defines the qBittorrent WebUI API operations.
// This interface allows for mock implementations in tests.
type QBittorrentClientInterface interface {
	// Login authenticates with qBittorrent WebUI
	Login(ctx context.Context) error

	// TestConnection tests the connection to qBittorrent
	TestConnection(ctx context.Context) error

	// GetVersion gets qBittorrent version
	GetVersion(ctx context.Context) (string, error)

	// GetPreferences gets application preferences
	GetPreferences(ctx context.Context) (*QBittorrentPreferences, error)

	// SetPreferences updates application preferences
	SetPreferences(ctx context.Context, prefs map[string]interface{}) error

	// GetTransferInfo gets transfer info (speeds, etc.)
	GetTransferInfo(ctx context.Context) (map[string]interface{}, error)
}

// Ensure QBittorrentClient implements the interface
var _ QBittorrentClientInterface = (*QBittorrentClient)(nil)

// QBittorrentClient is a client for qBittorrent WebUI API
type QBittorrentClient struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
	loggedIn   bool
}

// QBittorrentPreferences contains qBittorrent application preferences
type QBittorrentPreferences struct {
	// General
	Locale                 string `json:"locale,omitempty"`
	CreateSubfolderEnabled bool   `json:"create_subfolder_enabled,omitempty"`
	StartPausedEnabled     bool   `json:"start_paused_enabled,omitempty"`
	AutoDeleteMode         int    `json:"auto_delete_mode,omitempty"`
	PreallocateAll         bool   `json:"preallocate_all,omitempty"`
	IncompleteFilesExt     bool   `json:"incomplete_files_ext,omitempty"`

	// Downloads
	SavePath                    string      `json:"save_path,omitempty"`
	TempPathEnabled             bool        `json:"temp_path_enabled,omitempty"`
	TempPath                    string      `json:"temp_path,omitempty"`
	ExportDir                   string      `json:"export_dir,omitempty"`
	ExportDirFin                string      `json:"export_dir_fin,omitempty"`
	ScanDirs                    interface{} `json:"scan_dirs,omitempty"`
	MailNotificationEnabled     bool        `json:"mail_notification_enabled,omitempty"`
	MailNotificationSender      string      `json:"mail_notification_sender,omitempty"`
	MailNotificationEmail       string      `json:"mail_notification_email,omitempty"`
	MailNotificationSmtp        string      `json:"mail_notification_smtp,omitempty"`
	MailNotificationSslEnabled  bool        `json:"mail_notification_ssl_enabled,omitempty"`
	MailNotificationAuthEnabled bool        `json:"mail_notification_auth_enabled,omitempty"`
	MailNotificationUsername    string      `json:"mail_notification_username,omitempty"`
	MailNotificationPassword    string      `json:"mail_notification_password,omitempty"`

	// Connection
	ListenPort           int  `json:"listen_port,omitempty"`
	Upnp                 bool `json:"upnp,omitempty"`
	RandomPort           bool `json:"random_port,omitempty"`
	MaxConnec            int  `json:"max_connec,omitempty"`
	MaxConnecPerTorrent  int  `json:"max_connec_per_torrent,omitempty"`
	MaxUploads           int  `json:"max_uploads,omitempty"`
	MaxUploadsPerTorrent int  `json:"max_uploads_per_torrent,omitempty"`

	// Speed
	DlLimit            int  `json:"dl_limit,omitempty"`
	UpLimit            int  `json:"up_limit,omitempty"`
	AltDlLimit         int  `json:"alt_dl_limit,omitempty"`
	AltUpLimit         int  `json:"alt_up_limit,omitempty"`
	BittorrentProtocol int  `json:"bittorrent_protocol,omitempty"`
	LimitUtpRate       bool `json:"limit_utp_rate,omitempty"`
	LimitTcpOverhead   bool `json:"limit_tcp_overhead,omitempty"`
	LimitLanPeers      bool `json:"limit_lan_peers,omitempty"`

	// Alt-speed / Scheduler
	SchedulerEnabled bool `json:"scheduler_enabled,omitempty"`
	ScheduleFromHour int  `json:"schedule_from_hour,omitempty"`
	ScheduleFromMin  int  `json:"schedule_from_min,omitempty"`
	ScheduleToHour   int  `json:"schedule_to_hour,omitempty"`
	ScheduleToMin    int  `json:"schedule_to_min,omitempty"`
	SchedulerDays    int  `json:"scheduler_days,omitempty"`

	// BitTorrent
	Dht           bool `json:"dht,omitempty"`
	Pex           bool `json:"pex,omitempty"`
	Lsd           bool `json:"lsd,omitempty"`
	Encryption    int  `json:"encryption,omitempty"`
	AnonymousMode bool `json:"anonymous_mode,omitempty"`

	// Queueing
	QueueingEnabled            bool `json:"queueing_enabled,omitempty"`
	MaxActiveDownloads         int  `json:"max_active_downloads,omitempty"`
	MaxActiveTorrents          int  `json:"max_active_torrents,omitempty"`
	MaxActiveUploads           int  `json:"max_active_uploads,omitempty"`
	DontCountSlowTorrents      bool `json:"dont_count_slow_torrents,omitempty"`
	SlowTorrentDlRateThreshold int  `json:"slow_torrent_dl_rate_threshold,omitempty"`
	SlowTorrentUlRateThreshold int  `json:"slow_torrent_ul_rate_threshold,omitempty"`
	SlowTorrentInactiveTimer   int  `json:"slow_torrent_inactive_timer,omitempty"`

	// Seeding
	MaxRatio              float64 `json:"max_ratio,omitempty"`
	MaxRatioEnabled       bool    `json:"max_ratio_enabled,omitempty"`
	MaxSeedingTime        int     `json:"max_seeding_time,omitempty"`
	MaxSeedingTimeEnabled bool    `json:"max_seeding_time_enabled,omitempty"`
	MaxRatioAct           int     `json:"max_ratio_act,omitempty"`

	// Web UI
	WebUiDomainList                    string `json:"web_ui_domain_list,omitempty"`
	WebUiAddress                       string `json:"web_ui_address,omitempty"`
	WebUiPort                          int    `json:"web_ui_port,omitempty"`
	WebUiUpnp                          bool   `json:"web_ui_upnp,omitempty"`
	WebUiUsername                      string `json:"web_ui_username,omitempty"`
	WebUiPassword                      string `json:"web_ui_password,omitempty"`
	WebUiCsrfProtectionEnabled         bool   `json:"web_ui_csrf_protection_enabled,omitempty"`
	WebUiClickjackingProtectionEnabled bool   `json:"web_ui_clickjacking_protection_enabled,omitempty"`
	WebUiSecureCookieEnabled           bool   `json:"web_ui_secure_cookie_enabled,omitempty"`
	WebUiMaxAuthFailCount              int    `json:"web_ui_max_auth_fail_count,omitempty"`
	WebUiBanDuration                   int    `json:"web_ui_ban_duration,omitempty"`
	WebUiSessionTimeout                int    `json:"web_ui_session_timeout,omitempty"`
	BypassLocalAuth                    bool   `json:"bypass_local_auth,omitempty"`
	BypassAuthSubnetWhitelistEnabled   bool   `json:"bypass_auth_subnet_whitelist_enabled,omitempty"`
	BypassAuthSubnetWhitelist          string `json:"bypass_auth_subnet_whitelist,omitempty"`
	AlternativeWebuiEnabled            bool   `json:"alternative_webui_enabled,omitempty"`
	AlternativeWebuiPath               string `json:"alternative_webui_path,omitempty"`
	UseHttps                           bool   `json:"use_https,omitempty"`
}

// NewQBittorrentClient creates a new qBittorrent WebUI API client
func NewQBittorrentClient(baseURL, username, password string) *QBittorrentClient {
	jar, _ := cookiejar.New(nil)
	return &QBittorrentClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
			Jar:     jar,
		},
	}
}

// Login authenticates with qBittorrent WebUI
func (c *QBittorrentClient) Login(ctx context.Context) error {
	if c.loggedIn {
		return nil
	}

	data := url.Values{}
	data.Set("username", c.username)
	data.Set("password", c.password)

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/v2/auth/login", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	result := strings.TrimSpace(string(body))

	if resp.StatusCode != 200 {
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, result)
	}

	if result == "Fails." {
		return fmt.Errorf("login failed: invalid credentials")
	}

	if result != "Ok." {
		return fmt.Errorf("unexpected login response: %s", result)
	}

	c.loggedIn = true
	return nil
}

// request makes an authenticated request to qBittorrent API
func (c *QBittorrentClient) request(ctx context.Context, method, endpoint string, data url.Values) ([]byte, error) {
	// Ensure we're logged in
	if !c.loggedIn {
		if err := c.Login(ctx); err != nil {
			return nil, err
		}
	}

	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if data != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Session expired - re-login and retry
	if resp.StatusCode == 403 {
		c.loggedIn = false
		if err := c.Login(ctx); err != nil {
			return nil, fmt.Errorf("re-login failed: %w", err)
		}

		// Retry the request
		if data != nil {
			body = strings.NewReader(data.Encode())
		}
		req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}
		if data != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err = c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("retry request failed: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	return io.ReadAll(resp.Body)
}

// TestConnection tests the connection to qBittorrent
func (c *QBittorrentClient) TestConnection(ctx context.Context) error {
	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion gets qBittorrent version
func (c *QBittorrentClient) GetVersion(ctx context.Context) (string, error) {
	body, err := c.request(ctx, "GET", "/api/v2/app/version", nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// GetPreferences gets application preferences
func (c *QBittorrentClient) GetPreferences(ctx context.Context) (*QBittorrentPreferences, error) {
	body, err := c.request(ctx, "GET", "/api/v2/app/preferences", nil)
	if err != nil {
		return nil, err
	}

	var prefs QBittorrentPreferences
	if err := json.Unmarshal(body, &prefs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal preferences: %w", err)
	}

	return &prefs, nil
}

// SetPreferences updates application preferences
func (c *QBittorrentClient) SetPreferences(ctx context.Context, prefs map[string]interface{}) error {
	prefsJSON, err := json.Marshal(prefs)
	if err != nil {
		return fmt.Errorf("failed to marshal preferences: %w", err)
	}

	data := url.Values{}
	data.Set("json", string(prefsJSON))

	_, err = c.request(ctx, "POST", "/api/v2/app/setPreferences", data)
	return err
}

// GetTransferInfo gets transfer info (speeds, etc.)
func (c *QBittorrentClient) GetTransferInfo(ctx context.Context) (map[string]interface{}, error) {
	body, err := c.request(ctx, "GET", "/api/v2/transfer/info", nil)
	if err != nil {
		return nil, err
	}

	var info map[string]interface{}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transfer info: %w", err)
	}

	return info, nil
}

// GetMainData gets main data including torrents
func (c *QBittorrentClient) GetMainData(ctx context.Context, rid int) (map[string]interface{}, error) {
	data := url.Values{}
	if rid > 0 {
		data.Set("rid", fmt.Sprintf("%d", rid))
	}

	body, err := c.request(ctx, "GET", "/api/v2/sync/maindata?"+data.Encode(), nil)
	if err != nil {
		return nil, err
	}

	var mainData map[string]interface{}
	if err := json.Unmarshal(body, &mainData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal main data: %w", err)
	}

	return mainData, nil
}

// ToggleSpeedLimitsMode toggles alternative speed limits
func (c *QBittorrentClient) ToggleSpeedLimitsMode(ctx context.Context) error {
	_, err := c.request(ctx, "POST", "/api/v2/transfer/toggleSpeedLimitsMode", nil)
	return err
}

// SetSpeedLimits sets download/upload speed limits (0 = unlimited)
func (c *QBittorrentClient) SetSpeedLimits(ctx context.Context, downloadLimit, uploadLimit int) error {
	prefs := map[string]interface{}{
		"dl_limit": downloadLimit * 1024, // Convert to bytes
		"up_limit": uploadLimit * 1024,
	}
	return c.SetPreferences(ctx, prefs)
}
