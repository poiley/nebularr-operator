// Package discovery provides auto-discovery capabilities for *arr applications.
// This includes parsing API keys from config.xml files and inferring download client types.
package discovery

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ArrConfig represents the parsed config.xml structure for *arr applications.
// All *arr apps use a similar config.xml format.
type ArrConfig struct {
	XMLName xml.Name `xml:"Config"`
	ApiKey  string   `xml:"ApiKey"`
	Port    int      `xml:"Port"`
	UrlBase string   `xml:"UrlBase"`
}

// ParseConfigXML parses an *arr config.xml file and extracts configuration.
func ParseConfigXML(r io.Reader) (*ArrConfig, error) {
	var config ArrConfig
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to parse config.xml: %w", err)
	}
	return &config, nil
}

// ParseConfigXMLFromFile parses an *arr config.xml file from a file path.
func ParseConfigXMLFromFile(path string) (*ArrConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open config.xml: %w", err)
	}
	defer func() { _ = f.Close() }()
	return ParseConfigXML(f)
}

// DiscoverAPIKeyFromPVC attempts to discover the API key from a config.xml
// in a PersistentVolumeClaim mounted into a pod.
//
// Parameters:
//   - ctx: context for the operation
//   - k8sClient: Kubernetes client
//   - namespace: namespace where the PVC exists
//   - pvcName: name of the PVC containing the config
//   - configPath: path within the PVC to config.xml (default: "config.xml")
//
// Returns the API key or an error if discovery fails.
func DiscoverAPIKeyFromPVC(ctx context.Context, k8sClient client.Client, namespace, pvcName, configPath string) (string, error) {
	// This is a placeholder - actual implementation would need to:
	// 1. Create a temporary pod that mounts the PVC
	// 2. Execute a command to read the config.xml
	// 3. Parse the API key
	// 4. Clean up the temporary pod
	//
	// For now, we'll return an error indicating this needs implementation
	return "", fmt.Errorf("PVC-based API key discovery not yet implemented")
}

// DiscoverAPIKeyFromConfigMap attempts to discover the API key from a ConfigMap.
// This is useful when the config.xml is stored in a ConfigMap (less common).
func DiscoverAPIKeyFromConfigMap(ctx context.Context, k8sClient client.Client, namespace, configMapName, key string) (string, error) {
	var cm corev1.ConfigMap
	if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: namespace, Name: configMapName}, &cm); err != nil {
		return "", fmt.Errorf("failed to get ConfigMap %s/%s: %w", namespace, configMapName, err)
	}

	data, ok := cm.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in ConfigMap %s/%s", key, namespace, configMapName)
	}

	config, err := ParseConfigXML(strings.NewReader(data))
	if err != nil {
		return "", err
	}

	if config.ApiKey == "" {
		return "", fmt.Errorf("ApiKey is empty in config.xml from ConfigMap %s/%s", namespace, configMapName)
	}

	return config.ApiKey, nil
}

// InferDownloadClientType attempts to infer the download client type from its name.
// This is used when the user doesn't explicitly specify the implementation type.
//
// Examples:
//   - "qbittorrent" -> "qbittorrent"
//   - "my-qbit-client" -> "qbittorrent"
//   - "transmission-vpn" -> "transmission"
//   - "sabnzbd-main" -> "sabnzbd"
func InferDownloadClientType(name string) string {
	name = strings.ToLower(name)

	// Torrent clients
	torrentClients := map[string]string{
		"qbittorrent":  "qbittorrent",
		"qbit":         "qbittorrent",
		"transmission": "transmission",
		"deluge":       "deluge",
		"rtorrent":     "rtorrent",
		"rutorrent":    "rtorrent",
		"vuze":         "vuze",
		"utorrent":     "utorrent",
		"aria2":        "aria2",
		"flood":        "flood",
	}

	// Usenet clients
	usenetClients := map[string]string{
		"sabnzbd":     "sabnzbd",
		"sab":         "sabnzbd",
		"nzbget":      "nzbget",
		"nzbvortex":   "nzbvortex",
		"pneumatic":   "pneumatic",
		"usenetblack": "usenetblackhole",
	}

	// Check for matches
	for key, clientType := range torrentClients {
		if strings.Contains(name, key) {
			return clientType
		}
	}

	for key, clientType := range usenetClients {
		if strings.Contains(name, key) {
			return clientType
		}
	}

	// Unable to infer
	return ""
}

// InferProtocolFromClientType returns the protocol (torrent/usenet) for a client type.
func InferProtocolFromClientType(clientType string) string {
	torrentClients := []string{
		"qbittorrent", "transmission", "deluge", "rtorrent",
		"vuze", "utorrent", "aria2", "flood",
	}

	usenetClients := []string{
		"sabnzbd", "nzbget", "nzbvortex", "pneumatic", "usenetblackhole",
	}

	clientType = strings.ToLower(clientType)

	for _, tc := range torrentClients {
		if clientType == tc {
			return "torrent"
		}
	}

	for _, uc := range usenetClients {
		if clientType == uc {
			return "usenet"
		}
	}

	return ""
}

// DefaultConfigPaths returns the default paths where *arr apps store their config.xml
// relative to the data directory.
func DefaultConfigPaths() []string {
	return []string{
		"config.xml",
		filepath.Join("config", "config.xml"),
	}
}
