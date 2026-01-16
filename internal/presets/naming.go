package presets

// NamingPreset defines a naming preset
type NamingPreset struct {
	Name        string
	Description string
}

// RadarrNamingExpansion expands a preset for Radarr
type RadarrNamingExpansion struct {
	RenameMovies             bool
	ReplaceIllegalCharacters bool
	ColonReplacement         int // 0=delete, 1=dash, 2=space, 4=smart
	StandardMovieFormat      string
	MovieFolderFormat        string
}

// SonarrNamingExpansion expands a preset for Sonarr
type SonarrNamingExpansion struct {
	RenameEpisodes           bool
	ReplaceIllegalCharacters bool
	ColonReplacement         int
	StandardEpisodeFormat    string
	DailyEpisodeFormat       string
	AnimeEpisodeFormat       string
	SeriesFolderFormat       string
	SeasonFolderFormat       string
	SpecialsFolderFormat     string
	MultiEpisodeStyle        int // 0=extend, 1=duplicate, 2=repeat, 3=scene, 4=range, 5=prefixed
}

// LidarrNamingExpansion expands a preset for Lidarr
type LidarrNamingExpansion struct {
	RenameTracks             bool
	ReplaceIllegalCharacters bool
	ColonReplacement         int
	StandardTrackFormat      string
	MultiDiscTrackFormat     string
	ArtistFolderFormat       string
	AlbumFolderFormat        string
}

// NamingPresets contains all built-in naming presets
var NamingPresets = map[string]NamingPreset{
	"plex-friendly":     {Name: "plex-friendly", Description: "Optimized for Plex metadata matching"},
	"jellyfin-friendly": {Name: "jellyfin-friendly", Description: "Optimized for Jellyfin"},
	"kodi-friendly":     {Name: "kodi-friendly", Description: "Optimized for Kodi"},
	"detailed":          {Name: "detailed", Description: "Maximum info in filename"},
	"minimal":           {Name: "minimal", Description: "Clean, simple names"},
	"scene":             {Name: "scene", Description: "Scene-style naming"},
}

// DefaultNamingPreset is used when no preset is specified
const DefaultNamingPreset = "plex-friendly"

// Colon replacement constants
const (
	ColonDelete = 0
	ColonDash   = 1
	ColonSpace  = 2
	ColonSmart  = 4
)

// GetRadarrNaming returns the Radarr expansion for a preset
func GetRadarrNaming(presetName string) RadarrNamingExpansion {
	switch presetName {
	case "plex-friendly", "jellyfin-friendly", "kodi-friendly":
		return RadarrNamingExpansion{
			RenameMovies:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardMovieFormat:      "{Movie CleanTitle} ({Release Year}) - {Quality Full}",
			MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
		}
	case "detailed":
		return RadarrNamingExpansion{
			RenameMovies:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardMovieFormat:      "{Movie CleanTitle} ({Release Year}) [{Quality Full}]{[MediaInfo AudioCodec]}{[MediaInfo AudioChannels]}{[MediaInfo VideoCodec]}{[MediaInfo VideoDynamicRange]}{-Release Group}",
			MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
		}
	case "minimal":
		return RadarrNamingExpansion{
			RenameMovies:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonDelete,
			StandardMovieFormat:      "{Movie CleanTitle} ({Release Year})",
			MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
		}
	case "scene":
		return RadarrNamingExpansion{
			RenameMovies:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonDelete,
			StandardMovieFormat:      "{Movie.CleanTitle}.{Release Year}.{Quality.Full}-{Release Group}",
			MovieFolderFormat:        "{Movie CleanTitle} ({Release Year})",
		}
	default:
		return GetRadarrNaming("plex-friendly")
	}
}

// GetSonarrNaming returns the Sonarr expansion for a preset
func GetSonarrNaming(presetName string) SonarrNamingExpansion {
	switch presetName {
	case "plex-friendly", "jellyfin-friendly", "kodi-friendly":
		return SonarrNamingExpansion{
			RenameEpisodes:           true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardEpisodeFormat:    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
			DailyEpisodeFormat:       "{Series Title} - {Air-Date} - {Episode Title} {Quality Full}",
			AnimeEpisodeFormat:       "{Series Title} - S{season:00}E{episode:00} - {Episode Title} {Quality Full}",
			SeriesFolderFormat:       "{Series Title}",
			SeasonFolderFormat:       "Season {season:00}",
			SpecialsFolderFormat:     "Specials",
			MultiEpisodeStyle:        5, // Prefixed Range
		}
	case "detailed":
		return SonarrNamingExpansion{
			RenameEpisodes:           true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardEpisodeFormat:    "{Series Title} - S{season:00}E{episode:00} - {Episode Title} [{Quality Full}]{[MediaInfo AudioCodec]}{[MediaInfo VideoCodec]}{-Release Group}",
			DailyEpisodeFormat:       "{Series Title} - {Air-Date} - {Episode Title} [{Quality Full}]{-Release Group}",
			AnimeEpisodeFormat:       "{Series Title} - S{season:00}E{episode:00} - {absolute:000} - {Episode Title} [{Quality Full}]{-Release Group}",
			SeriesFolderFormat:       "{Series Title}",
			SeasonFolderFormat:       "Season {season:00}",
			SpecialsFolderFormat:     "Specials",
			MultiEpisodeStyle:        5,
		}
	case "minimal":
		return SonarrNamingExpansion{
			RenameEpisodes:           true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonDelete,
			StandardEpisodeFormat:    "{Series Title} - S{season:00}E{episode:00} - {Episode Title}",
			DailyEpisodeFormat:       "{Series Title} - {Air-Date} - {Episode Title}",
			AnimeEpisodeFormat:       "{Series Title} - S{season:00}E{episode:00} - {Episode Title}",
			SeriesFolderFormat:       "{Series Title}",
			SeasonFolderFormat:       "Season {season:00}",
			SpecialsFolderFormat:     "Specials",
			MultiEpisodeStyle:        0, // Extend
		}
	case "scene":
		return SonarrNamingExpansion{
			RenameEpisodes:           true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonDelete,
			StandardEpisodeFormat:    "{Series.Title}.S{season:00}E{episode:00}.{Episode.Title}.{Quality.Full}-{Release Group}",
			DailyEpisodeFormat:       "{Series.Title}.{Air.Date}.{Episode.Title}.{Quality.Full}-{Release Group}",
			AnimeEpisodeFormat:       "{Series.Title}.S{season:00}E{episode:00}.{Episode.Title}.{Quality.Full}-{Release Group}",
			SeriesFolderFormat:       "{Series Title}",
			SeasonFolderFormat:       "Season {season:00}",
			SpecialsFolderFormat:     "Specials",
			MultiEpisodeStyle:        3, // Scene
		}
	default:
		return GetSonarrNaming("plex-friendly")
	}
}

// GetLidarrNaming returns the Lidarr expansion for a preset
func GetLidarrNaming(presetName string) LidarrNamingExpansion {
	switch presetName {
	case "plex-friendly", "jellyfin-friendly", "kodi-friendly":
		return LidarrNamingExpansion{
			RenameTracks:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardTrackFormat:      "{Album Artist} - {Album Title} - {track:00} - {Track Title}",
			MultiDiscTrackFormat:     "{Album Artist} - {Album Title} - {medium:0}{track:00} - {Track Title}",
			ArtistFolderFormat:       "{Artist Name}",
			AlbumFolderFormat:        "{Album Title} ({Release Year})",
		}
	case "detailed":
		return LidarrNamingExpansion{
			RenameTracks:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonSmart,
			StandardTrackFormat:      "{Album Artist} - {Album Title} - {track:00} - {Track Title} [{Quality Full}]",
			MultiDiscTrackFormat:     "{Album Artist} - {Album Title} - {medium:0}{track:00} - {Track Title} [{Quality Full}]",
			ArtistFolderFormat:       "{Artist Name}",
			AlbumFolderFormat:        "{Album Title} ({Release Year}) [{Quality Full}]",
		}
	case "minimal":
		return LidarrNamingExpansion{
			RenameTracks:             true,
			ReplaceIllegalCharacters: true,
			ColonReplacement:         ColonDelete,
			StandardTrackFormat:      "{track:00} - {Track Title}",
			MultiDiscTrackFormat:     "{medium:0}{track:00} - {Track Title}",
			ArtistFolderFormat:       "{Artist Name}",
			AlbumFolderFormat:        "{Album Title}",
		}
	default:
		return GetLidarrNaming("plex-friendly")
	}
}

// GetNamingPreset returns a naming preset by name
func GetNamingPreset(name string) (NamingPreset, bool) {
	preset, ok := NamingPresets[name]
	return preset, ok
}

// ListNamingPresets returns all available naming preset names
func ListNamingPresets() []string {
	names := make([]string, 0, len(NamingPresets))
	for name := range NamingPresets {
		names = append(names, name)
	}
	return names
}
