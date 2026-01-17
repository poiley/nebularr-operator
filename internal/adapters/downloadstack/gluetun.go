// Package downloadstack provides configuration management for gluetun+transmission stack.
package downloadstack

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
)

// GluetunEnvInput contains all resolved values for generating Gluetun env vars
type GluetunEnvInput struct {
	Spec *arrv1alpha1.GluetunSpec

	// Resolved credentials
	Username   string
	Password   string
	PrivateKey string // For WireGuard
}

// GenerateGluetunEnv generates environment variables for Gluetun container
func GenerateGluetunEnv(input *GluetunEnvInput) map[string]string {
	env := make(map[string]string)
	spec := input.Spec

	// Provider
	env["VPN_SERVICE_PROVIDER"] = spec.Provider.Name
	env["VPN_TYPE"] = spec.VPNType

	// Credentials based on VPN type
	if spec.VPNType == "wireguard" {
		if input.PrivateKey != "" {
			env["WIREGUARD_PRIVATE_KEY"] = input.PrivateKey
		}
	} else {
		// OpenVPN
		if input.Username != "" {
			env["OPENVPN_USER"] = input.Username
		}
		if input.Password != "" {
			env["OPENVPN_PASSWORD"] = input.Password
		}
	}

	// Server selection
	if spec.Server != nil {
		if len(spec.Server.Regions) > 0 {
			env["SERVER_REGIONS"] = strings.Join(spec.Server.Regions, ",")
		}
		if len(spec.Server.Countries) > 0 {
			env["SERVER_COUNTRIES"] = strings.Join(spec.Server.Countries, ",")
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
			env["FIREWALL_VPN_INPUT_PORTS"] = joinInts(spec.Firewall.VPNInputPorts)
		}
		if len(spec.Firewall.InputPorts) > 0 {
			env["FIREWALL_INPUT_PORTS"] = joinInts(spec.Firewall.InputPorts)
		}
		if len(spec.Firewall.OutboundSubnets) > 0 {
			env["FIREWALL_OUTBOUND_SUBNETS"] = strings.Join(spec.Firewall.OutboundSubnets, ",")
		}
		if spec.Firewall.Debug {
			env["FIREWALL_DEBUG"] = "on"
		}
	}

	// Kill switch
	if spec.KillSwitch != nil {
		if spec.KillSwitch.Enabled {
			env["BLOCK_WITHOUT_VPN"] = "on"
		} else {
			env["BLOCK_WITHOUT_VPN"] = "off"
		}
	}

	// DNS
	if spec.DNS != nil {
		if spec.DNS.OverTLS {
			env["DOT"] = "on"
		} else {
			env["DOT"] = "off"
		}
		if spec.DNS.PlaintextAddress != "" {
			env["DNS_PLAINTEXT_ADDRESS"] = spec.DNS.PlaintextAddress
		}
		if spec.DNS.KeepNameserver {
			env["DNS_KEEP_NAMESERVER"] = "on"
		}
	}

	// IPv6
	if spec.IPv6 != nil {
		if !spec.IPv6.Enabled {
			env["OPENVPN_IPV6"] = "off"
		} else {
			env["OPENVPN_IPV6"] = "on"
		}
	}

	// Logging
	if spec.Logging != nil && spec.Logging.Level != "" {
		env["LOG_LEVEL"] = spec.Logging.Level
	}

	return env
}

// HashGluetunEnv computes a hash of the env map for change detection
func HashGluetunEnv(env map[string]string) string {
	// Sort keys for deterministic ordering
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build string to hash
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(env[k])
		sb.WriteString("\n")
	}

	hash := sha256.Sum256([]byte(sb.String()))
	return fmt.Sprintf("%x", hash[:8]) // First 8 bytes as hex
}

// joinInts joins a slice of ints with commas
func joinInts(ints []int) string {
	strs := make([]string, len(ints))
	for i, v := range ints {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(strs, ",")
}
