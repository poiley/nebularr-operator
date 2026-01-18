//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package containers

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ArrType represents the type of *arr application
type ArrType string

const (
	ArrTypeRadarr   ArrType = "radarr"
	ArrTypeSonarr   ArrType = "sonarr"
	ArrTypeLidarr   ArrType = "lidarr"
	ArrTypeProwlarr ArrType = "prowlarr"
)

// ArrContainer wraps a testcontainer for an *arr application
type ArrContainer struct {
	Container testcontainers.Container
	Type      ArrType
	Host      string
	Port      string
	APIKey    string
}

// URL returns the base URL for the *arr API
func (c *ArrContainer) URL() string {
	return fmt.Sprintf("http://%s:%s", c.Host, c.Port)
}

// APIURL returns the full API URL
func (c *ArrContainer) APIURL() string {
	return fmt.Sprintf("%s/api/v3", c.URL())
}

// Terminate stops and removes the container
func (c *ArrContainer) Terminate(ctx context.Context) error {
	if c.Container != nil {
		return c.Container.Terminate(ctx)
	}
	return nil
}

// ArrContainerOptions configures the *arr container
type ArrContainerOptions struct {
	// ImageTag is the Docker image tag to use (default: "latest")
	ImageTag string
	// StartupTimeout is how long to wait for the container to be ready
	StartupTimeout time.Duration
}

// DefaultArrContainerOptions returns default options
func DefaultArrContainerOptions() ArrContainerOptions {
	return ArrContainerOptions{
		ImageTag:       "latest",
		StartupTimeout: 2 * time.Minute,
	}
}

// getImageName returns the Docker image name for the *arr type
func getImageName(arrType ArrType, tag string) string {
	// Use linuxserver images as they're well-maintained and widely used
	return fmt.Sprintf("lscr.io/linuxserver/%s:%s", arrType, tag)
}

// getDefaultPort returns the default port for each *arr type
func getDefaultPort(arrType ArrType) string {
	switch arrType {
	case ArrTypeRadarr:
		return "7878"
	case ArrTypeSonarr:
		return "8989"
	case ArrTypeLidarr:
		return "8686"
	case ArrTypeProwlarr:
		return "9696"
	default:
		return "7878"
	}
}

// StartArrContainer starts an *arr container and waits for it to be ready
func StartArrContainer(ctx context.Context, arrType ArrType, opts ArrContainerOptions) (*ArrContainer, error) {
	if opts.ImageTag == "" {
		opts.ImageTag = "latest"
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 2 * time.Minute
	}

	port := getDefaultPort(arrType)
	image := getImageName(arrType, opts.ImageTag)
	natPort := nat.Port(fmt.Sprintf("%s/tcp", port))

	req := testcontainers.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{string(natPort)},
		Env: map[string]string{
			"PUID": "1000",
			"PGID": "1000",
			"TZ":   "Etc/UTC",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(natPort),
			wait.ForHTTP("/").
				WithPort(natPort).
				WithStatusCodeMatcher(func(status int) bool {
					// *arr apps return 200 or redirect to setup wizard
					return status == http.StatusOK || status == http.StatusFound || status == http.StatusMovedPermanently
				}),
		).WithDeadline(opts.StartupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start %s container: %w", arrType, err)
	}

	// Get the mapped host and port
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedPort, err := container.MappedPort(ctx, natPort)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped port: %w", err)
	}

	arrContainer := &ArrContainer{
		Container: container,
		Type:      arrType,
		Host:      host,
		Port:      mappedPort.Port(),
	}

	// Wait for the app to fully initialize and extract API key
	apiKey, err := waitForAPIKeyAndExtract(ctx, container, arrType, opts.StartupTimeout)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to extract API key: %w", err)
	}
	arrContainer.APIKey = apiKey

	return arrContainer, nil
}

// waitForAPIKeyAndExtract waits for the config.xml to be created and extracts the API key
func waitForAPIKeyAndExtract(ctx context.Context, container testcontainers.Container, arrType ArrType, timeout time.Duration) (string, error) {
	configPath := fmt.Sprintf("/config/%s/config.xml", arrType)
	// Newer versions might use just /config/config.xml
	altConfigPath := "/config/config.xml"

	deadline := time.Now().Add(timeout)
	var lastErr error

	for time.Now().Before(deadline) {
		// Try primary path first
		apiKey, err := extractAPIKeyFromContainer(ctx, container, configPath)
		if err == nil && apiKey != "" {
			return apiKey, nil
		}
		lastErr = err

		// Try alternate path
		apiKey, err = extractAPIKeyFromContainer(ctx, container, altConfigPath)
		if err == nil && apiKey != "" {
			return apiKey, nil
		}
		if err != nil {
			lastErr = err
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(2 * time.Second):
			// Continue polling
		}
	}

	return "", fmt.Errorf("timed out waiting for API key: %v", lastErr)
}

// extractAPIKeyFromContainer reads config.xml from the container and extracts the API key
func extractAPIKeyFromContainer(ctx context.Context, container testcontainers.Container, configPath string) (string, error) {
	// Execute cat command to read the config file
	exitCode, reader, err := container.Exec(ctx, []string{"cat", configPath})
	if err != nil {
		return "", fmt.Errorf("failed to exec cat: %w", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("cat command failed with exit code %d", exitCode)
	}

	content, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read config content: %w", err)
	}

	return parseAPIKeyFromXML(string(content))
}

// parseAPIKeyFromXML extracts the API key from config.xml content
func parseAPIKeyFromXML(content string) (string, error) {
	// The config.xml structure varies slightly but ApiKey is always present
	type Config struct {
		XMLName xml.Name `xml:"Config"`
		ApiKey  string   `xml:"ApiKey"`
	}

	// Try to find just the config part (container exec might include other output)
	startIdx := strings.Index(content, "<Config>")
	if startIdx == -1 {
		startIdx = strings.Index(content, "<config>")
	}
	if startIdx == -1 {
		return "", fmt.Errorf("could not find Config element in XML")
	}

	endIdx := strings.Index(content, "</Config>")
	if endIdx == -1 {
		endIdx = strings.Index(content, "</config>")
	}
	if endIdx == -1 {
		return "", fmt.Errorf("could not find closing Config element in XML")
	}

	xmlContent := content[startIdx : endIdx+len("</Config>")]

	var config Config
	if err := xml.Unmarshal([]byte(xmlContent), &config); err != nil {
		// Try case-insensitive parsing
		xmlContent = strings.ReplaceAll(xmlContent, "<config>", "<Config>")
		xmlContent = strings.ReplaceAll(xmlContent, "</config>", "</Config>")
		xmlContent = strings.ReplaceAll(xmlContent, "<apikey>", "<ApiKey>")
		xmlContent = strings.ReplaceAll(xmlContent, "</apikey>", "</ApiKey>")
		if err := xml.Unmarshal([]byte(xmlContent), &config); err != nil {
			return "", fmt.Errorf("failed to parse config XML: %w", err)
		}
	}

	if config.ApiKey == "" {
		return "", fmt.Errorf("API key is empty in config")
	}

	return config.ApiKey, nil
}

// StartRadarr is a convenience function to start a Radarr container
func StartRadarr(ctx context.Context, opts ...ArrContainerOptions) (*ArrContainer, error) {
	opt := DefaultArrContainerOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	return StartArrContainer(ctx, ArrTypeRadarr, opt)
}

// StartSonarr is a convenience function to start a Sonarr container
func StartSonarr(ctx context.Context, opts ...ArrContainerOptions) (*ArrContainer, error) {
	opt := DefaultArrContainerOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	return StartArrContainer(ctx, ArrTypeSonarr, opt)
}

// StartLidarr is a convenience function to start a Lidarr container
func StartLidarr(ctx context.Context, opts ...ArrContainerOptions) (*ArrContainer, error) {
	opt := DefaultArrContainerOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	return StartArrContainer(ctx, ArrTypeLidarr, opt)
}

// StartProwlarr is a convenience function to start a Prowlarr container
func StartProwlarr(ctx context.Context, opts ...ArrContainerOptions) (*ArrContainer, error) {
	opt := DefaultArrContainerOptions()
	if len(opts) > 0 {
		opt = opts[0]
	}
	return StartArrContainer(ctx, ArrTypeProwlarr, opt)
}

// TransmissionContainer wraps a testcontainer for Transmission
type TransmissionContainer struct {
	Container testcontainers.Container
	Host      string
	Port      string
	RPCPort   string
	Username  string
	Password  string
}

// URL returns the Transmission RPC URL
func (c *TransmissionContainer) URL() string {
	return fmt.Sprintf("http://%s:%s/transmission/rpc", c.Host, c.RPCPort)
}

// Terminate stops and removes the container
func (c *TransmissionContainer) Terminate(ctx context.Context) error {
	if c.Container != nil {
		return c.Container.Terminate(ctx)
	}
	return nil
}

// TransmissionOptions configures the Transmission container
type TransmissionOptions struct {
	ImageTag       string
	Username       string
	Password       string
	StartupTimeout time.Duration
}

// DefaultTransmissionOptions returns default options
func DefaultTransmissionOptions() TransmissionOptions {
	return TransmissionOptions{
		ImageTag:       "latest",
		Username:       "transmission",
		Password:       "transmission",
		StartupTimeout: 1 * time.Minute,
	}
}

// StartTransmission starts a Transmission container
func StartTransmission(ctx context.Context, opts TransmissionOptions) (*TransmissionContainer, error) {
	if opts.ImageTag == "" {
		opts.ImageTag = "latest"
	}
	if opts.StartupTimeout == 0 {
		opts.StartupTimeout = 1 * time.Minute
	}
	if opts.Username == "" {
		opts.Username = "transmission"
	}
	if opts.Password == "" {
		opts.Password = "transmission"
	}

	rpcPort := nat.Port("9091/tcp")
	peerPortTCP := nat.Port("51413/tcp")
	peerPortUDP := nat.Port("51413/udp")

	req := testcontainers.ContainerRequest{
		Image:        fmt.Sprintf("lscr.io/linuxserver/transmission:%s", opts.ImageTag),
		ExposedPorts: []string{string(rpcPort), string(peerPortTCP), string(peerPortUDP)},
		Env: map[string]string{
			"PUID":           "1000",
			"PGID":           "1000",
			"TZ":             "Etc/UTC",
			"USER":           opts.Username,
			"PASS":           opts.Password,
			"PEERPORT":       "51413",
			"HOST_WHITELIST": "*",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort(rpcPort),
			wait.ForHTTP("/transmission/web/").
				WithPort(rpcPort).
				WithBasicAuth(opts.Username, opts.Password).
				WithStatusCodeMatcher(func(status int) bool {
					return status == http.StatusOK || status == http.StatusUnauthorized
				}),
		).WithDeadline(opts.StartupTimeout),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start transmission container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	mappedRPCPort, err := container.MappedPort(ctx, rpcPort)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get RPC port: %w", err)
	}

	mappedPeerPort, err := container.MappedPort(ctx, peerPortTCP)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get peer port: %w", err)
	}

	return &TransmissionContainer{
		Container: container,
		Host:      host,
		Port:      mappedPeerPort.Port(),
		RPCPort:   mappedRPCPort.Port(),
		Username:  opts.Username,
		Password:  opts.Password,
	}, nil
}
