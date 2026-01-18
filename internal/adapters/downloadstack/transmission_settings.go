package downloadstack

import (
	"context"
	"strconv"

	arrv1alpha1 "github.com/poiley/nebularr-operator/api/v1alpha1"
)

// TransmissionSettingsInput contains all values needed for settings sync
type TransmissionSettingsInput struct {
	Spec     *arrv1alpha1.TransmissionSpec
	Username string
	Password string
}

// SyncTransmissionSettings synchronizes the desired settings to Transmission
func SyncTransmissionSettings(ctx context.Context, client TransmissionClientInterface, input *TransmissionSettingsInput) error {
	settings := buildTransmissionSettings(input.Spec)
	if len(settings) == 0 {
		return nil
	}
	return client.SetSession(ctx, settings)
}

// buildTransmissionSettings converts CRD spec to Transmission settings map
func buildTransmissionSettings(spec *arrv1alpha1.TransmissionSpec) map[string]interface{} {
	settings := make(map[string]interface{})

	// Speed limits
	if spec.Speed != nil {
		settings["speed-limit-down"] = spec.Speed.DownloadLimit
		settings["speed-limit-down-enabled"] = spec.Speed.DownloadLimitEnabled
		settings["speed-limit-up"] = spec.Speed.UploadLimit
		settings["speed-limit-up-enabled"] = spec.Speed.UploadLimitEnabled
	}

	// Alt-speed (turtle mode)
	if spec.AltSpeed != nil {
		settings["alt-speed-enabled"] = spec.AltSpeed.Enabled
		settings["alt-speed-down"] = spec.AltSpeed.Down
		settings["alt-speed-up"] = spec.AltSpeed.Up
		settings["alt-speed-time-enabled"] = spec.AltSpeed.TimeEnabled
		settings["alt-speed-time-begin"] = spec.AltSpeed.TimeBegin
		settings["alt-speed-time-end"] = spec.AltSpeed.TimeEnd

		// Convert days array to bitmask (Transmission uses a bitmask)
		// Sunday = 1, Monday = 2, Tuesday = 4, ... Saturday = 64
		// Our input: 1=Mon, 2=Tue, ... 7=Sun
		if len(spec.AltSpeed.TimeDays) > 0 {
			dayMask := 0
			for _, day := range spec.AltSpeed.TimeDays {
				switch day {
				case 1: // Monday
					dayMask |= 2
				case 2: // Tuesday
					dayMask |= 4
				case 3: // Wednesday
					dayMask |= 8
				case 4: // Thursday
					dayMask |= 16
				case 5: // Friday
					dayMask |= 32
				case 6: // Saturday
					dayMask |= 64
				case 7: // Sunday
					dayMask |= 1
				}
			}
			settings["alt-speed-time-day"] = dayMask
		}
	}

	// Directories
	if spec.Directories != nil {
		if spec.Directories.Download != "" {
			settings["download-dir"] = spec.Directories.Download
		}
		if spec.Directories.Incomplete != "" {
			settings["incomplete-dir"] = spec.Directories.Incomplete
		}
		settings["incomplete-dir-enabled"] = spec.Directories.IncompleteEnabled
	}

	// Seeding
	if spec.Seeding != nil {
		if spec.Seeding.RatioLimit != "" {
			if ratio, err := strconv.ParseFloat(spec.Seeding.RatioLimit, 64); err == nil {
				settings["seedRatioLimit"] = ratio
			}
		}
		settings["seedRatioLimited"] = spec.Seeding.RatioLimited
		settings["idle-seeding-limit"] = spec.Seeding.IdleLimit
		settings["idle-seeding-limit-enabled"] = spec.Seeding.IdleLimitEnabled
	}

	// Queue
	if spec.Queue != nil {
		settings["download-queue-size"] = spec.Queue.DownloadSize
		settings["download-queue-enabled"] = spec.Queue.DownloadEnabled
		settings["seed-queue-size"] = spec.Queue.SeedSize
		settings["seed-queue-enabled"] = spec.Queue.SeedEnabled
		settings["queue-stalled-enabled"] = spec.Queue.StalledEnabled
		settings["queue-stalled-minutes"] = spec.Queue.StalledMinutes
	}

	// Peers
	if spec.Peers != nil {
		if spec.Peers.LimitGlobal > 0 {
			settings["peer-limit-global"] = spec.Peers.LimitGlobal
		}
		if spec.Peers.LimitPerTorrent > 0 {
			settings["peer-limit-per-torrent"] = spec.Peers.LimitPerTorrent
		}
		if spec.Peers.Port > 0 {
			settings["peer-port"] = spec.Peers.Port
		}
		settings["peer-port-random-on-start"] = spec.Peers.RandomPort
		settings["port-forwarding-enabled"] = spec.Peers.PortForwardingEnabled
	}

	// Security
	if spec.Security != nil {
		if spec.Security.Encryption != "" {
			settings["encryption"] = spec.Security.Encryption
		}
		if spec.Security.PEXEnabled != nil {
			settings["pex-enabled"] = *spec.Security.PEXEnabled
		}
		if spec.Security.DHTEnabled != nil {
			settings["dht-enabled"] = *spec.Security.DHTEnabled
		}
		if spec.Security.LPDEnabled != nil {
			settings["lpd-enabled"] = *spec.Security.LPDEnabled
		}
		if spec.Security.UTPEnabled != nil {
			settings["utp-enabled"] = *spec.Security.UTPEnabled
		}
	}

	// Blocklist
	if spec.Blocklist != nil {
		settings["blocklist-enabled"] = spec.Blocklist.Enabled
		if spec.Blocklist.URL != "" {
			settings["blocklist-url"] = spec.Blocklist.URL
		}
	}

	return settings
}

// GetTransmissionVersion retrieves the Transmission version string
func GetTransmissionVersion(ctx context.Context, client TransmissionClientInterface) (string, error) {
	session, err := client.GetSession(ctx)
	if err != nil {
		return "", err
	}
	return session.Version, nil
}
