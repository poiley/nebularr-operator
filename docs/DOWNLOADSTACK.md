# Nebularr - DownloadStack Configuration Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains DownloadStack adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md)

This document is a reference for implementing the DownloadStack controller. The DownloadStackConfig manages VPN (Gluetun) and multiple download clients as a unified stack.

---

## 1. Overview

The DownloadStackConfig CRD manages:
- **Gluetun VPN**: Secure VPN container configuration
- **Torrent clients**: Transmission, qBittorrent, Deluge, rTorrent
- **Usenet clients**: SABnzbd, NZBGet

All download clients run through the Gluetun VPN container for privacy.

---

## 2. Architecture

### 2.1 Pod Structure

```
+--------------------------------------------------+
|                  Download Stack Pod               |
|  +--------------------------------------------+  |
|  |              Gluetun (VPN)                 |  |
|  |  - Network namespace                       |  |
|  |  - Kill switch                             |  |
|  |  - Port forwarding                         |  |
|  +--------------------------------------------+  |
|              |                |                   |
|  +-----------v--+  +---------v--------+          |
|  | Transmission |  |    SABnzbd       |          |
|  | qBittorrent  |  |    NZBGet        |          |
|  | Deluge       |  +------------------+          |
|  | rTorrent     |                                |
|  +--------------+                                |
+--------------------------------------------------+
```

### 2.2 Configuration Flow

```
DownloadStackConfig CR
        |
        v
+-------+--------+
| Gluetun Secret |  (VPN credentials/config)
+-------+--------+
        |
        v
+-------+--------+
|  Deployment    |  (Restart if config changes)
+-------+--------+
        |
        v
+-------+--------+-------+--------+
|  Transmission  | SABnzbd | etc. |  (API configuration)
+----------------+--------+-------+
```

---

## 3. Gluetun VPN Configuration

### 3.1 Supported Providers

| Provider | VPN Types | Auth Method |
|----------|-----------|-------------|
| `mullvad` | OpenVPN, WireGuard | Account ID |
| `nordvpn` | OpenVPN, WireGuard | Username/Password |
| `expressvpn` | OpenVPN | Username/Password |
| `pia` | OpenVPN, WireGuard | Username/Password |
| `surfshark` | OpenVPN, WireGuard | Username/Password |
| `windscribe` | OpenVPN, WireGuard | Username/Password |
| `protonvpn` | OpenVPN | Username/Password |
| `ivpn` | OpenVPN, WireGuard | Username/Password |
| `airvpn` | OpenVPN, WireGuard | Username/Password |
| `custom` | OpenVPN, WireGuard | Config file |

### 3.2 Gluetun Environment Variables

The controller generates a Secret with Gluetun environment variables:

```go
// internal/adapters/downloadstack/gluetun.go

package downloadstack

import (
    arrv1alpha1 "github.com/your-org/nebularr/api/v1alpha1"
)

// GenerateGluetunEnv creates Gluetun environment variables
func GenerateGluetunEnv(spec *arrv1alpha1.GluetunSpec, secrets map[string]string) map[string]string {
    env := map[string]string{
        "VPN_SERVICE_PROVIDER": spec.Provider.Name,
        "VPN_TYPE":             spec.VPNType,
    }
    
    // Authentication
    if spec.Provider.CredentialsSecretRef != nil {
        env["OPENVPN_USER"] = secrets["username"]
        env["OPENVPN_PASSWORD"] = secrets["password"]
    }
    if spec.Provider.PrivateKeySecretRef != nil {
        env["WIREGUARD_PRIVATE_KEY"] = secrets["privateKey"]
    }
    
    // Server selection
    if spec.Server != nil {
        if len(spec.Server.Countries) > 0 {
            env["SERVER_COUNTRIES"] = strings.Join(spec.Server.Countries, ",")
        }
        if len(spec.Server.Regions) > 0 {
            env["SERVER_REGIONS"] = strings.Join(spec.Server.Regions, ",")
        }
        if len(spec.Server.Cities) > 0 {
            env["SERVER_CITIES"] = strings.Join(spec.Server.Cities, ",")
        }
        if len(spec.Server.Hostnames) > 0 {
            env["SERVER_HOSTNAMES"] = strings.Join(spec.Server.Hostnames, ",")
        }
    }
    
    // Firewall
    if spec.Firewall != nil {
        if len(spec.Firewall.VPNInputPorts) > 0 {
            ports := make([]string, len(spec.Firewall.VPNInputPorts))
            for i, p := range spec.Firewall.VPNInputPorts {
                ports[i] = strconv.Itoa(p)
            }
            env["VPN_PORT_FORWARDING_PORTS"] = strings.Join(ports, ",")
        }
        if len(spec.Firewall.OutboundSubnets) > 0 {
            env["FIREWALL_OUTBOUND_SUBNETS"] = strings.Join(spec.Firewall.OutboundSubnets, ",")
        }
    }
    
    // Kill switch
    if spec.KillSwitch != nil {
        if spec.KillSwitch.Enabled {
            env["FIREWALL"] = "on"
        } else {
            env["FIREWALL"] = "off"
        }
    }
    
    // DNS
    if spec.DNS != nil {
        if spec.DNS.OverTLS {
            env["DOT"] = "on"
        }
        if spec.DNS.PlaintextAddress != "" {
            env["DOT_PROVIDERS"] = spec.DNS.PlaintextAddress
        }
    }
    
    // Logging
    if spec.Logging != nil {
        env["LOG_LEVEL"] = spec.Logging.Level
    }
    
    return env
}
```

---

## 4. Torrent Clients

### 4.1 Transmission

**Connection:**
- Protocol: JSON-RPC over HTTP
- Default port: 9091
- Path: `/transmission/rpc`

**API Client:**
```go
// internal/adapters/downloadstack/transmission_client.go

package downloadstack

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type TransmissionClient struct {
    url       string
    username  string
    password  string
    sessionID string
    client    *http.Client
}

type transmissionRequest struct {
    Method    string      `json:"method"`
    Arguments interface{} `json:"arguments,omitempty"`
    Tag       int         `json:"tag,omitempty"`
}

type transmissionResponse struct {
    Result    string          `json:"result"`
    Arguments json.RawMessage `json:"arguments,omitempty"`
    Tag       int             `json:"tag,omitempty"`
}

func (c *TransmissionClient) call(ctx context.Context, method string, args interface{}, result interface{}) error {
    req := transmissionRequest{Method: method, Arguments: args}
    body, _ := json.Marshal(req)
    
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.url+"/transmission/rpc", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.Header.Set("X-Transmission-Session-Id", c.sessionID)
    
    if c.username != "" {
        httpReq.SetBasicAuth(c.username, c.password)
    }
    
    resp, err := c.client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    // Handle session ID refresh (409 conflict)
    if resp.StatusCode == 409 {
        c.sessionID = resp.Header.Get("X-Transmission-Session-Id")
        return c.call(ctx, method, args, result)
    }
    
    var response transmissionResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return err
    }
    
    if response.Result != "success" {
        return fmt.Errorf("transmission error: %s", response.Result)
    }
    
    if result != nil {
        return json.Unmarshal(response.Arguments, result)
    }
    return nil
}

func (c *TransmissionClient) GetSessionStats(ctx context.Context) (*TransmissionSession, error) {
    var result struct {
        Version string `json:"version"`
    }
    if err := c.call(ctx, "session-get", nil, &result); err != nil {
        return nil, err
    }
    return &TransmissionSession{Version: result.Version}, nil
}

func (c *TransmissionClient) SetSessionSettings(ctx context.Context, settings map[string]interface{}) error {
    return c.call(ctx, "session-set", settings, nil)
}
```

**Settings Mapping:**

| CRD Field | Transmission API Field |
|-----------|----------------------|
| `speed.downloadLimit` | `speed-limit-down` |
| `speed.uploadLimit` | `speed-limit-up` |
| `speed.downloadLimitEnabled` | `speed-limit-down-enabled` |
| `speed.uploadLimitEnabled` | `speed-limit-up-enabled` |
| `directories.download` | `download-dir` |
| `directories.incomplete` | `incomplete-dir` |
| `seeding.ratioLimit` | `seedRatioLimit` |
| `queue.downloadSize` | `download-queue-size` |
| `peers.limitGlobal` | `peer-limit-global` |
| `security.encryption` | `encryption` |
| `blocklist.url` | `blocklist-url` |

---

### 4.2 qBittorrent

**Connection:**
- Protocol: REST API over HTTP
- Default port: 8080
- Auth: Cookie-based session

**API Client:**
```go
// internal/adapters/downloadstack/qbittorrent_client.go

package downloadstack

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strings"
)

type QBittorrentClient struct {
    url      string
    username string
    password string
    client   *http.Client
    cookie   string
}

func (c *QBittorrentClient) Login(ctx context.Context) error {
    data := url.Values{}
    data.Set("username", c.username)
    data.Set("password", c.password)
    
    req, _ := http.NewRequestWithContext(ctx, "POST", c.url+"/api/v2/auth/login", 
        strings.NewReader(data.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    for _, cookie := range resp.Cookies() {
        if cookie.Name == "SID" {
            c.cookie = cookie.Value
            return nil
        }
    }
    return fmt.Errorf("login failed: no session cookie")
}

func (c *QBittorrentClient) GetVersion(ctx context.Context) (string, error) {
    req, _ := http.NewRequestWithContext(ctx, "GET", c.url+"/api/v2/app/version", nil)
    req.AddCookie(&http.Cookie{Name: "SID", Value: c.cookie})
    
    resp, err := c.client.Do(req)
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()
    
    var version string
    json.NewDecoder(resp.Body).Decode(&version)
    return version, nil
}

func (c *QBittorrentClient) SetPreferences(ctx context.Context, prefs map[string]interface{}) error {
    data, _ := json.Marshal(prefs)
    form := url.Values{}
    form.Set("json", string(data))
    
    req, _ := http.NewRequestWithContext(ctx, "POST", c.url+"/api/v2/app/setPreferences",
        strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{Name: "SID", Value: c.cookie})
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != 200 {
        return fmt.Errorf("failed to set preferences: %d", resp.StatusCode)
    }
    return nil
}
```

---

### 4.3 Deluge

**Connection:**
- Protocol: JSON-RPC over HTTP
- Default port: 8112
- Auth: Password only

---

### 4.4 rTorrent

**Connection:**
- Protocol: XML-RPC
- Default port: varies (often via SCGI)
- Auth: HTTP Basic (if behind reverse proxy)

---

## 5. Usenet Clients

### 5.1 SABnzbd

**Connection:**
- Protocol: REST API
- Default port: 8080
- Auth: API Key

**API Client:**
```go
// internal/adapters/downloadstack/sabnzbd_client.go

package downloadstack

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
)

type SABnzbdClient struct {
    url    string
    apiKey string
    client *http.Client
}

func (c *SABnzbdClient) call(ctx context.Context, mode string, params url.Values) (json.RawMessage, error) {
    if params == nil {
        params = url.Values{}
    }
    params.Set("apikey", c.apiKey)
    params.Set("mode", mode)
    params.Set("output", "json")
    
    reqURL := fmt.Sprintf("%s/api?%s", c.url, params.Encode())
    req, _ := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()
    
    var result json.RawMessage
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, err
    }
    return result, nil
}

func (c *SABnzbdClient) GetVersion(ctx context.Context) (string, error) {
    result, err := c.call(ctx, "version", nil)
    if err != nil {
        return "", err
    }
    
    var response struct {
        Version string `json:"version"`
    }
    if err := json.Unmarshal(result, &response); err != nil {
        return "", err
    }
    return response.Version, nil
}

func (c *SABnzbdClient) SetConfig(ctx context.Context, section, keyword, value string) error {
    params := url.Values{}
    params.Set("section", section)
    params.Set("keyword", keyword)
    params.Set("value", value)
    
    _, err := c.call(ctx, "set_config", params)
    return err
}
```

**API Modes:**

| Mode | Description |
|------|-------------|
| `version` | Get SABnzbd version |
| `get_config` | Get configuration |
| `set_config` | Set configuration |
| `queue` | Get download queue |
| `history` | Get download history |
| `addurl` | Add NZB by URL |
| `pause` | Pause downloads |
| `resume` | Resume downloads |

---

### 5.2 NZBGet

**Connection:**
- Protocol: JSON-RPC
- Default port: 6789
- Auth: Username/Password

**API Client:**
```go
// internal/adapters/downloadstack/nzbget_client.go

package downloadstack

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
)

type NZBGetClient struct {
    url      string
    username string
    password string
    client   *http.Client
}

type nzbgetRequest struct {
    Method string        `json:"method"`
    Params []interface{} `json:"params,omitempty"`
}

type nzbgetResponse struct {
    Result json.RawMessage `json:"result"`
    Error  *struct {
        Code    int    `json:"code"`
        Message string `json:"message"`
    } `json:"error,omitempty"`
}

func (c *NZBGetClient) call(ctx context.Context, method string, params []interface{}, result interface{}) error {
    req := nzbgetRequest{Method: method, Params: params}
    body, _ := json.Marshal(req)
    
    httpReq, _ := http.NewRequestWithContext(ctx, "POST", c.url+"/jsonrpc", bytes.NewReader(body))
    httpReq.Header.Set("Content-Type", "application/json")
    httpReq.SetBasicAuth(c.username, c.password)
    
    resp, err := c.client.Do(httpReq)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    var response nzbgetResponse
    if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
        return err
    }
    
    if response.Error != nil {
        return fmt.Errorf("nzbget error %d: %s", response.Error.Code, response.Error.Message)
    }
    
    if result != nil {
        return json.Unmarshal(response.Result, result)
    }
    return nil
}

func (c *NZBGetClient) GetVersion(ctx context.Context) (string, error) {
    var version string
    if err := c.call(ctx, "version", nil, &version); err != nil {
        return "", err
    }
    return version, nil
}

func (c *NZBGetClient) GetConfig(ctx context.Context) ([]NZBGetConfigItem, error) {
    var config []NZBGetConfigItem
    if err := c.call(ctx, "config", nil, &config); err != nil {
        return nil, err
    }
    return config, nil
}

func (c *NZBGetClient) SetConfig(ctx context.Context, name, value string) error {
    return c.call(ctx, "saveconfig", []interface{}{
        []map[string]string{{"Name": name, "Value": value}},
    }, nil)
}

type NZBGetConfigItem struct {
    Name  string `json:"Name"`
    Value string `json:"Value"`
}
```

---

## 6. CRD Example

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: DownloadStackConfig
metadata:
  name: download-stack
spec:
  deploymentRef:
    name: download-stack

  # Gluetun VPN
  gluetun:
    provider:
      name: mullvad
      credentialsSecretRef:
        name: vpn-credentials
        usernameKey: username
        passwordKey: password
    vpnType: wireguard
    server:
      countries: ["Netherlands", "Germany"]
    firewall:
      vpnInputPorts: [51413, 6881]
      outboundSubnets: ["10.0.0.0/8", "192.168.0.0/16"]
    killSwitch:
      enabled: true
    dns:
      overTls: true

  # Torrent client
  transmission:
    connection:
      url: http://localhost:9091
      credentialsSecretRef:
        name: transmission-credentials
        usernameKey: username
        passwordKey: password
    speed:
      uploadLimitEnabled: true
      uploadLimit: 1000
    directories:
      download: /downloads/complete
      incomplete: /downloads/incomplete
      incompleteEnabled: true
    seeding:
      ratioLimited: true
      ratioLimit: "2.0"
    security:
      encryption: required

  # Usenet client  
  sabnzbd:
    connection:
      url: http://localhost:8080
      apiKeySecretRef:
        name: sabnzbd-credentials
        key: apiKey
    directories:
      completeDir: /downloads/complete
      downloadDir: /downloads/incomplete
    categories:
      - name: movies
        dir: movies
      - name: tv
        dir: tv
    postProcessing:
      enabled: true
      unpackEnabled: true

  restartOnGluetunChange: true
  reconciliation:
    interval: 5m
```

---

## 7. Status Fields

The DownloadStackConfigStatus tracks all components:

| Field | Description |
|-------|-------------|
| `gluetunSecretGenerated` | VPN Secret created |
| `gluetunConfigHash` | Hash for change detection |
| `transmissionConnected` | Transmission reachable |
| `transmissionVersion` | Transmission version |
| `qbittorrentConnected` | qBittorrent reachable |
| `qbittorrentVersion` | qBittorrent version |
| `delugeConnected` | Deluge reachable |
| `delugeVersion` | Deluge version |
| `rtorrentConnected` | rTorrent reachable |
| `rtorrentVersion` | rTorrent version |
| `sabnzbdConnected` | SABnzbd reachable |
| `sabnzbdVersion` | SABnzbd version |
| `nzbgetConnected` | NZBGet reachable |
| `nzbgetVersion` | NZBGet version |

---

## 8. Deployment Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: download-stack
spec:
  template:
    spec:
      containers:
        - name: gluetun
          image: qmcgaw/gluetun:latest
          securityContext:
            capabilities:
              add: ["NET_ADMIN"]
          envFrom:
            - secretRef:
                name: download-stack-gluetun-env
          ports:
            - containerPort: 9091  # Transmission
            - containerPort: 8080  # SABnzbd

        - name: transmission
          image: lscr.io/linuxserver/transmission:latest
          # Uses Gluetun network namespace
          
        - name: sabnzbd
          image: lscr.io/linuxserver/sabnzbd:latest
          # Uses Gluetun network namespace
```

---

## 9. Troubleshooting

### 9.1 VPN Not Connecting

1. Check Gluetun logs: `kubectl logs <pod> -c gluetun`
2. Verify credentials in Secret
3. Check provider-specific requirements

### 9.2 Download Client Unreachable

1. Ensure Gluetun is healthy first
2. Check firewall allows local access
3. Verify `outboundSubnets` includes cluster network

### 9.3 Port Forwarding Not Working

1. Verify provider supports port forwarding
2. Check `vpnInputPorts` configuration
3. Some providers require specific servers
