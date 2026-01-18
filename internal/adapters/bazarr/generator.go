// Package bazarr provides Bazarr configuration generation for Nebularr.
// Unlike other *arr apps, Bazarr uses file-based configuration (config.yaml)
// rather than an API for configuration management.
package bazarr

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"strconv"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
)

// Config represents the Bazarr config.yaml structure
type Config struct {
	Sonarr    SonarrConfig            `yaml:"sonarr"`
	Radarr    RadarrConfig            `yaml:"radarr"`
	General   GeneralConfig           `yaml:"general"`
	Languages []ConfigLanguageProfile `yaml:"languages,omitempty"`
	Auth      *AuthConfig             `yaml:"auth,omitempty"`
	Providers map[string]interface{}  `yaml:",inline"`
}

// SonarrConfig represents Sonarr connection settings in Bazarr
type SonarrConfig struct {
	IP      string `yaml:"ip"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
	SSL     bool   `yaml:"ssl"`
	APIKey  string `yaml:"apikey"`
}

// RadarrConfig represents Radarr connection settings in Bazarr
type RadarrConfig struct {
	IP      string `yaml:"ip"`
	Port    int    `yaml:"port"`
	BaseURL string `yaml:"base_url"`
	SSL     bool   `yaml:"ssl"`
	APIKey  string `yaml:"apikey"`
}

// GeneralConfig represents Bazarr general settings
type GeneralConfig struct {
	UseSonarr           bool     `yaml:"use_sonarr"`
	UseRadarr           bool     `yaml:"use_radarr"`
	EnabledProviders    []string `yaml:"enabled_providers"`
	SingleLanguage      bool     `yaml:"single_language"`
	SerieDefaultEnabled bool     `yaml:"serie_default_enabled"`
	SerieDefaultProfile int      `yaml:"serie_default_profile"`
	MovieDefaultEnabled bool     `yaml:"movie_default_enabled"`
	MovieDefaultProfile int      `yaml:"movie_default_profile"`
}

// ConfigLanguageProfile represents a Bazarr language profile for config.yaml generation
type ConfigLanguageProfile struct {
	Name      string               `yaml:"name"`
	ProfileID int                  `yaml:"profileId"`
	Cutoff    interface{}          `yaml:"cutoff"`
	Items     []ConfigLanguageItem `yaml:"items"`
}

// ConfigLanguageItem represents a language within a profile for config.yaml generation
type ConfigLanguageItem struct {
	ID           int    `yaml:"id"`
	Language     string `yaml:"language"`
	Forced       bool   `yaml:"forced"`
	HI           bool   `yaml:"hi"`
	AudioExclude bool   `yaml:"audio_exclude"`
}

// AuthConfig represents Bazarr authentication settings
type AuthConfig struct {
	Type     *string `yaml:"type"`
	Username string  `yaml:"username,omitempty"`
	Password string  `yaml:"password,omitempty"`
}

// ProviderConfig represents a subtitle provider configuration
type ProviderConfig struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	APIKey   string `yaml:"apikey,omitempty"`
}

// GeneratorInput contains all resolved values needed for config generation
type GeneratorInput struct {
	Spec         *arrv1alpha1.BazarrConfigSpec
	SonarrAPIKey string
	RadarrAPIKey string
	// ResolvedProviderSecrets maps provider name -> resolved credentials
	ResolvedProviderSecrets map[string]ProviderSecrets
	// ResolvedAuthPassword is the resolved authentication password
	ResolvedAuthPassword string
}

// ProviderSecrets contains resolved secrets for a provider
type ProviderSecrets struct {
	Password string
	APIKey   string
}

// Generate generates Bazarr config.yaml content from the input
func Generate(input *GeneratorInput) (*Config, error) {
	spec := input.Spec

	// Parse Sonarr URL
	sonarrHost, sonarrPort, sonarrBase, sonarrSSL, err := parseURL(spec.Sonarr.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid sonarr URL: %w", err)
	}

	// Parse Radarr URL
	radarrHost, radarrPort, radarrBase, radarrSSL, err := parseURL(spec.Radarr.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid radarr URL: %w", err)
	}

	config := &Config{
		Sonarr: SonarrConfig{
			IP:      sonarrHost,
			Port:    sonarrPort,
			BaseURL: sonarrBase,
			SSL:     sonarrSSL,
			APIKey:  input.SonarrAPIKey,
		},
		Radarr: RadarrConfig{
			IP:      radarrHost,
			Port:    radarrPort,
			BaseURL: radarrBase,
			SSL:     radarrSSL,
			APIKey:  input.RadarrAPIKey,
		},
		General:   buildGeneralConfig(spec),
		Providers: make(map[string]interface{}),
	}

	// Build language profiles
	if len(spec.LanguageProfiles) > 0 {
		config.Languages = buildLanguageProfiles(spec.LanguageProfiles)
	}

	// Build auth config
	if spec.Authentication != nil && spec.Authentication.Method != "none" {
		config.Auth = buildAuthConfig(spec.Authentication, input.ResolvedAuthPassword)
	}

	// Build provider configs
	for _, provider := range spec.Providers {
		secrets := input.ResolvedProviderSecrets[provider.Name]
		if providerConfig := buildProviderConfig(&provider, secrets); providerConfig != nil {
			config.Providers[provider.Name] = providerConfig
		}
	}

	return config, nil
}

// GenerateYAML generates YAML bytes from the config
func GenerateYAML(config *Config) ([]byte, error) {
	return yaml.Marshal(config)
}

// parseURL parses a URL into host, port, base path, and SSL flag
func parseURL(rawURL string) (host string, port int, basePath string, ssl bool, err error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", 0, "", false, err
	}

	host = parsed.Hostname()
	if host == "" {
		host = "localhost"
	}

	ssl = parsed.Scheme == "https"

	portStr := parsed.Port()
	if portStr == "" {
		if ssl {
			port = 443
		} else {
			port = 80
		}
	} else {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return "", 0, "", false, fmt.Errorf("invalid port: %w", err)
		}
	}

	basePath = parsed.Path
	if basePath == "" {
		basePath = "/"
	}

	return host, port, basePath, ssl, nil
}

// buildGeneralConfig builds the general settings section
func buildGeneralConfig(spec *arrv1alpha1.BazarrConfigSpec) GeneralConfig {
	// Find default profiles
	serieDefaultProfile := 1
	movieDefaultProfile := 1

	for idx, profile := range spec.LanguageProfiles {
		if profile.DefaultForSeries {
			serieDefaultProfile = idx + 1
		}
		if profile.DefaultForMovies {
			movieDefaultProfile = idx + 1
		}
	}

	// Collect enabled providers
	enabledProviders := make([]string, 0, len(spec.Providers))
	for _, p := range spec.Providers {
		enabledProviders = append(enabledProviders, p.Name)
	}

	return GeneralConfig{
		UseSonarr:           true,
		UseRadarr:           true,
		EnabledProviders:    enabledProviders,
		SingleLanguage:      false,
		SerieDefaultEnabled: true,
		SerieDefaultProfile: serieDefaultProfile,
		MovieDefaultEnabled: true,
		MovieDefaultProfile: movieDefaultProfile,
	}
}

// buildLanguageProfiles builds the language profiles section
func buildLanguageProfiles(profiles []arrv1alpha1.BazarrLanguageProfile) []ConfigLanguageProfile {
	result := make([]ConfigLanguageProfile, 0, len(profiles))

	for idx, profile := range profiles {
		items := make([]ConfigLanguageItem, 0, len(profile.Languages))
		for langIdx, lang := range profile.Languages {
			items = append(items, ConfigLanguageItem{
				ID:           langIdx + 1,
				Language:     lang.Code,
				Forced:       lang.Forced,
				HI:           lang.HearingImpaired,
				AudioExclude: false,
			})
		}

		result = append(result, ConfigLanguageProfile{
			Name:      profile.Name,
			ProfileID: idx + 1,
			Cutoff:    nil,
			Items:     items,
		})
	}

	return result
}

// buildAuthConfig builds the authentication section
func buildAuthConfig(auth *arrv1alpha1.AuthenticationSpec, resolvedPassword string) *AuthConfig {
	if auth == nil || auth.Method == "none" {
		return nil
	}

	var authType *string
	if auth.Method != "" && auth.Method != "none" {
		authType = &auth.Method
	}

	config := &AuthConfig{
		Type:     authType,
		Username: auth.Username,
	}

	// Bazarr stores passwords as MD5 hashes
	if resolvedPassword != "" {
		config.Password = md5Hash(resolvedPassword)
	}

	return config
}

// buildProviderConfig builds a provider configuration section
func buildProviderConfig(provider *arrv1alpha1.BazarrProvider, secrets ProviderSecrets) *ProviderConfig {
	config := &ProviderConfig{
		Username: provider.Username,
		Password: secrets.Password,
		APIKey:   secrets.APIKey,
	}

	// Only return if there are actual settings
	if config.Username == "" && config.Password == "" && config.APIKey == "" {
		return nil
	}

	return config
}

// md5Hash returns the MD5 hash of a string
func md5Hash(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
