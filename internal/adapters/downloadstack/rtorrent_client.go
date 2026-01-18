package downloadstack

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RTorrentClientInterface defines the rTorrent XML-RPC API operations.
// This interface allows for mock implementations in tests.
type RTorrentClientInterface interface {
	// TestConnection tests the connection to rTorrent
	TestConnection(ctx context.Context) error

	// GetVersion gets rTorrent version
	GetVersion(ctx context.Context) (string, error)

	// GetSetting gets a single setting value
	GetSetting(ctx context.Context, name string) (string, error)

	// SetSetting sets a single setting value
	SetSetting(ctx context.Context, name string, value interface{}) error

	// GetDownloadRate gets the current download rate limit (bytes/sec)
	GetDownloadRate(ctx context.Context) (int64, error)

	// SetDownloadRate sets the download rate limit (bytes/sec, 0 = unlimited)
	SetDownloadRate(ctx context.Context, rate int64) error

	// GetUploadRate gets the current upload rate limit (bytes/sec)
	GetUploadRate(ctx context.Context) (int64, error)

	// SetUploadRate sets the upload rate limit (bytes/sec, 0 = unlimited)
	SetUploadRate(ctx context.Context, rate int64) error

	// GetDirectory gets the default download directory
	GetDirectory(ctx context.Context) (string, error)

	// SetDirectory sets the default download directory
	SetDirectory(ctx context.Context, dir string) error
}

// Ensure RTorrentClient implements the interface
var _ RTorrentClientInterface = (*RTorrentClient)(nil)

// RTorrentClient is a client for rTorrent XML-RPC API
type RTorrentClient struct {
	baseURL    string
	httpClient *http.Client
	username   string
	password   string
}

// XMLRPCRequest represents an XML-RPC method call
type XMLRPCRequest struct {
	XMLName    xml.Name `xml:"methodCall"`
	MethodName string   `xml:"methodName"`
	Params     []XMLRPCParam
}

// XMLRPCParam represents an XML-RPC parameter
type XMLRPCParam struct {
	XMLName xml.Name `xml:"param"`
	Value   XMLRPCValue
}

// XMLRPCValue represents an XML-RPC value
type XMLRPCValue struct {
	XMLName xml.Name `xml:"value"`
	String  string   `xml:"string,omitempty"`
	Int     *int64   `xml:"i8,omitempty"`
	Int4    *int     `xml:"i4,omitempty"`
	Double  *float64 `xml:"double,omitempty"`
	Boolean *int     `xml:"boolean,omitempty"`
}

// XMLRPCResponse represents an XML-RPC response
type XMLRPCResponse struct {
	XMLName xml.Name `xml:"methodResponse"`
	Params  []struct {
		Value struct {
			String  string  `xml:"string"`
			Int     int64   `xml:"i8"`
			Int4    int     `xml:"i4"`
			Double  float64 `xml:"double"`
			Boolean int     `xml:"boolean"`
			Array   *struct {
				Data struct {
					Values []struct {
						String  string  `xml:"string"`
						Int     int64   `xml:"i8"`
						Int4    int     `xml:"i4"`
						Double  float64 `xml:"double"`
						Boolean int     `xml:"boolean"`
					} `xml:"value"`
				} `xml:"data"`
			} `xml:"array"`
		} `xml:"value"`
	} `xml:"params>param"`
	Fault *struct {
		Value struct {
			Struct struct {
				Members []struct {
					Name  string `xml:"name"`
					Value struct {
						String string `xml:"string"`
						Int    int    `xml:"i4"`
					} `xml:"value"`
				} `xml:"member"`
			} `xml:"struct"`
		} `xml:"value"`
	} `xml:"fault"`
}

// NewRTorrentClient creates a new rTorrent XML-RPC client
func NewRTorrentClient(baseURL, username, password string) *RTorrentClient {
	return &RTorrentClient{
		baseURL:  strings.TrimSuffix(baseURL, "/"),
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
	}
}

// call makes an XML-RPC call to rTorrent
func (c *RTorrentClient) call(ctx context.Context, method string, params ...interface{}) (*XMLRPCResponse, error) {
	// Build XML-RPC request
	var xmlParams []XMLRPCParam
	for _, p := range params {
		var val XMLRPCValue
		switch v := p.(type) {
		case string:
			val.String = v
		case int:
			i := int64(v)
			val.Int = &i
		case int64:
			val.Int = &v
		case float64:
			val.Double = &v
		case bool:
			b := 0
			if v {
				b = 1
			}
			val.Boolean = &b
		default:
			return nil, fmt.Errorf("unsupported parameter type: %T", p)
		}
		xmlParams = append(xmlParams, XMLRPCParam{Value: val})
	}

	req := XMLRPCRequest{
		MethodName: method,
		Params:     xmlParams,
	}

	body, err := xml.MarshalIndent(req, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Add XML declaration
	body = append([]byte(xml.Header), body...)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "text/xml")
	if c.username != "" {
		httpReq.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var xmlResp XMLRPCResponse
	if err := xml.Unmarshal(responseBody, &xmlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for fault
	if xmlResp.Fault != nil {
		var faultCode int
		var faultString string
		for _, m := range xmlResp.Fault.Value.Struct.Members {
			switch m.Name {
			case "faultCode":
				faultCode = m.Value.Int
			case "faultString":
				faultString = m.Value.String
			}
		}
		return nil, fmt.Errorf("XML-RPC fault %d: %s", faultCode, faultString)
	}

	return &xmlResp, nil
}

// TestConnection tests the connection to rTorrent
func (c *RTorrentClient) TestConnection(ctx context.Context) error {
	_, err := c.GetVersion(ctx)
	return err
}

// GetVersion gets rTorrent version
func (c *RTorrentClient) GetVersion(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "system.client_version")
	if err != nil {
		return "", err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.String, nil
	}
	return "", fmt.Errorf("no version in response")
}

// GetLibTorrentVersion gets libtorrent version
func (c *RTorrentClient) GetLibTorrentVersion(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "system.library_version")
	if err != nil {
		return "", err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.String, nil
	}
	return "", fmt.Errorf("no version in response")
}

// GetSetting gets a single setting value
func (c *RTorrentClient) GetSetting(ctx context.Context, name string) (string, error) {
	resp, err := c.call(ctx, name)
	if err != nil {
		return "", err
	}

	if len(resp.Params) > 0 {
		// Handle different value types
		v := resp.Params[0].Value
		if v.String != "" {
			return v.String, nil
		}
		if v.Int != 0 {
			return fmt.Sprintf("%d", v.Int), nil
		}
		if v.Int4 != 0 {
			return fmt.Sprintf("%d", v.Int4), nil
		}
	}
	return "", nil
}

// SetSetting sets a single setting value
func (c *RTorrentClient) SetSetting(ctx context.Context, name string, value interface{}) error {
	_, err := c.call(ctx, name+".set", "", value)
	return err
}

// GetDownloadRate gets the current download rate limit (bytes/sec)
func (c *RTorrentClient) GetDownloadRate(ctx context.Context) (int64, error) {
	resp, err := c.call(ctx, "throttle.global_down.max_rate")
	if err != nil {
		return 0, err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.Int, nil
	}
	return 0, nil
}

// SetDownloadRate sets the download rate limit (bytes/sec, 0 = unlimited)
func (c *RTorrentClient) SetDownloadRate(ctx context.Context, rate int64) error {
	_, err := c.call(ctx, "throttle.global_down.max_rate.set", "", rate)
	return err
}

// GetUploadRate gets the current upload rate limit (bytes/sec)
func (c *RTorrentClient) GetUploadRate(ctx context.Context) (int64, error) {
	resp, err := c.call(ctx, "throttle.global_up.max_rate")
	if err != nil {
		return 0, err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.Int, nil
	}
	return 0, nil
}

// SetUploadRate sets the upload rate limit (bytes/sec, 0 = unlimited)
func (c *RTorrentClient) SetUploadRate(ctx context.Context, rate int64) error {
	_, err := c.call(ctx, "throttle.global_up.max_rate.set", "", rate)
	return err
}

// GetDirectory gets the default download directory
func (c *RTorrentClient) GetDirectory(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "directory.default")
	if err != nil {
		return "", err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.String, nil
	}
	return "", nil
}

// SetDirectory sets the default download directory
func (c *RTorrentClient) SetDirectory(ctx context.Context, dir string) error {
	_, err := c.call(ctx, "directory.default.set", "", dir)
	return err
}

// GetSessionDirectory gets the session directory
func (c *RTorrentClient) GetSessionDirectory(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "session.path")
	if err != nil {
		return "", err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.String, nil
	}
	return "", nil
}

// GetMaxPeers gets the global max peers
func (c *RTorrentClient) GetMaxPeers(ctx context.Context) (int64, error) {
	resp, err := c.call(ctx, "throttle.max_peers.normal")
	if err != nil {
		return 0, err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.Int, nil
	}
	return 0, nil
}

// SetMaxPeers sets the global max peers
func (c *RTorrentClient) SetMaxPeers(ctx context.Context, peers int64) error {
	_, err := c.call(ctx, "throttle.max_peers.normal.set", "", peers)
	return err
}

// GetMaxUploads gets the global max upload slots
func (c *RTorrentClient) GetMaxUploads(ctx context.Context) (int64, error) {
	resp, err := c.call(ctx, "throttle.max_uploads.global")
	if err != nil {
		return 0, err
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.Int, nil
	}
	return 0, nil
}

// SetMaxUploads sets the global max upload slots
func (c *RTorrentClient) SetMaxUploads(ctx context.Context, uploads int64) error {
	_, err := c.call(ctx, "throttle.max_uploads.global.set", "", uploads)
	return err
}

// IsDHTEnabled checks if DHT is enabled
func (c *RTorrentClient) IsDHTEnabled(ctx context.Context) (bool, error) {
	resp, err := c.call(ctx, "dht.mode")
	if err != nil {
		return false, err
	}

	if len(resp.Params) > 0 {
		// "auto", "on", or "off"
		return resp.Params[0].Value.String != "off", nil
	}
	return false, nil
}

// SetDHTMode sets DHT mode ("auto", "on", "off")
func (c *RTorrentClient) SetDHTMode(ctx context.Context, mode string) error {
	_, err := c.call(ctx, "dht.mode.set", mode)
	return err
}

// GetEncryptionMode gets the encryption mode
func (c *RTorrentClient) GetEncryptionMode(ctx context.Context) (string, error) {
	resp, err := c.call(ctx, "protocol.encryption.set")
	if err != nil {
		// This might be a get method, try alternative
		return "", nil
	}

	if len(resp.Params) > 0 {
		return resp.Params[0].Value.String, nil
	}
	return "", nil
}

// SetEncryptionMode sets the encryption mode
// Valid modes: none, allow_incoming, try_outgoing, require, require_RC4, require_RC4_strong
func (c *RTorrentClient) SetEncryptionMode(ctx context.Context, mode string) error {
	_, err := c.call(ctx, "protocol.encryption.set", mode)
	return err
}

// GetTransferStats gets current transfer statistics
func (c *RTorrentClient) GetTransferStats(ctx context.Context) (downloadRate, uploadRate int64, err error) {
	// Get download rate
	dlResp, err := c.call(ctx, "throttle.global_down.rate")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get download rate: %w", err)
	}
	if len(dlResp.Params) > 0 {
		downloadRate = dlResp.Params[0].Value.Int
	}

	// Get upload rate
	ulResp, err := c.call(ctx, "throttle.global_up.rate")
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get upload rate: %w", err)
	}
	if len(ulResp.Params) > 0 {
		uploadRate = ulResp.Params[0].Value.Int
	}

	return downloadRate, uploadRate, nil
}
