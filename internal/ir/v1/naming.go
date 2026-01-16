package v1

// NamingIR represents naming configuration (union type for app-specific)
type NamingIR struct {
	// Radarr naming
	Radarr *RadarrNamingIR `json:"radarr,omitempty"`

	// Sonarr naming
	Sonarr *SonarrNamingIR `json:"sonarr,omitempty"`

	// Lidarr naming
	Lidarr *LidarrNamingIR `json:"lidarr,omitempty"`
}

// RadarrNamingIR for Radarr naming config
type RadarrNamingIR struct {
	RenameMovies             bool   `json:"renameMovies"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	ColonReplacementFormat   int    `json:"colonReplacementFormat"` // 0=delete, 1=dash, 2=space, 4=smart
	StandardMovieFormat      string `json:"standardMovieFormat"`
	MovieFolderFormat        string `json:"movieFolderFormat"`
}

// SonarrNamingIR for Sonarr naming config
type SonarrNamingIR struct {
	RenameEpisodes           bool   `json:"renameEpisodes"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	ColonReplacementFormat   int    `json:"colonReplacementFormat"`
	StandardEpisodeFormat    string `json:"standardEpisodeFormat"`
	DailyEpisodeFormat       string `json:"dailyEpisodeFormat"`
	AnimeEpisodeFormat       string `json:"animeEpisodeFormat"`
	SeriesFolderFormat       string `json:"seriesFolderFormat"`
	SeasonFolderFormat       string `json:"seasonFolderFormat"`
	SpecialsFolderFormat     string `json:"specialsFolderFormat"`
	MultiEpisodeStyle        int    `json:"multiEpisodeStyle"`
}

// LidarrNamingIR for Lidarr naming config
type LidarrNamingIR struct {
	RenameTracks             bool   `json:"renameTracks"`
	ReplaceIllegalCharacters bool   `json:"replaceIllegalCharacters"`
	ColonReplacementFormat   int    `json:"colonReplacementFormat"`
	StandardTrackFormat      string `json:"standardTrackFormat"`
	MultiDiscTrackFormat     string `json:"multiDiscTrackFormat"`
	ArtistFolderFormat       string `json:"artistFolderFormat"`
	AlbumFolderFormat        string `json:"albumFolderFormat"`
}

// Colon replacement format constants
const (
	ColonReplacementDelete = 0
	ColonReplacementDash   = 1
	ColonReplacementSpace  = 2
	ColonReplacementSmart  = 4
)

// Sonarr multi-episode style constants
const (
	MultiEpisodeStyleExtend        = 0
	MultiEpisodeStyleDuplicate     = 1
	MultiEpisodeStyleRepeat        = 2
	MultiEpisodeStyleScene         = 3
	MultiEpisodeStyleRange         = 4
	MultiEpisodeStylePrefixedRange = 5
)
