# Nebularr - Readarr Configuration Reference

> **For coding agents:** Start with [README.md](./README.md) for build order. This document contains Readarr adapter code to copy.
>
> **Related:** [README](./README.md) | [TYPES](./TYPES.md) | [CRDS](./CRDS.md)

This document is a reference for implementing the Readarr adapter. Readarr manages eBooks and audiobooks, with unique features like metadata profiles for Goodreads integration.

---

## 1. Overview

Readarr is unique among the *arr applications because it:
- Manages both **eBooks** (EPUB, MOBI, AZW3, PDF) and **audiobooks** (FLAC, MP3, M4B)
- Uses **metadata profiles** to control book discovery from Goodreads
- Organizes content by **author** rather than just title
- Has different quality concepts compared to video-focused apps

---

## 2. Quality/Format Mapping

### 2.1 Book Format Quality IDs

Readarr uses format-based quality IDs rather than resolution-based:

| Format | Readarr ID | Description |
|--------|------------|-------------|
| EPUB | 1 | Standard eBook format |
| MOBI | 2 | Amazon Kindle format |
| AZW3 | 3 | Amazon Kindle enhanced format |
| PDF | 4 | Portable Document Format |
| FLAC | 10 | Lossless audiobook |
| MP3 | 11 | Compressed audiobook |
| M4B | 12 | Apple audiobook format |
| Unknown | 0 | Unknown/other format |

### 2.2 Quality Presets

The `ReadarrQualitySpec.Preset` field supports these presets:

| Preset | Description | Allowed Formats |
|--------|-------------|-----------------|
| `standard` | General reading | EPUB, MOBI, AZW3, PDF |
| `high-quality` | Prefer lossless | EPUB, AZW3, FLAC, M4B |
| `audiobook-focus` | Audiobooks only | FLAC, MP3, M4B |

### 2.3 Go Implementation

```go
// internal/adapters/readarr/mapping.go

package readarr

// FormatMapping maps format names to Readarr quality IDs
var FormatMapping = map[string]int{
    "EPUB":    1,
    "MOBI":    2,
    "AZW3":    3,
    "PDF":     4,
    "FLAC":    10,
    "MP3":     11,
    "M4B":     12,
    "Unknown": 0,
}

// ReverseFormatMapping for converting Readarr state back to format names
var ReverseFormatMapping = func() map[int]string {
    m := make(map[int]string)
    for name, id := range FormatMapping {
        m[id] = name
    }
    return m
}()

// QualityPresets defines built-in quality configurations
var QualityPresets = map[string][]string{
    "standard":       {"EPUB", "MOBI", "AZW3", "PDF"},
    "high-quality":   {"EPUB", "AZW3", "FLAC", "M4B"},
    "audiobook-focus": {"FLAC", "MP3", "M4B"},
}
```

---

## 3. Metadata Profile Mapping

### 3.1 Metadata Profile API

Readarr metadata profiles control how books are discovered and matched from Goodreads.

**API Endpoints:**
- `GET /api/v1/metadataprofile` - List all metadata profiles
- `GET /api/v1/metadataprofile/{id}` - Get specific profile
- `POST /api/v1/metadataprofile` - Create new profile
- `PUT /api/v1/metadataprofile/{id}` - Update profile
- `DELETE /api/v1/metadataprofile/{id}` - Delete profile

### 3.2 Metadata Profile Fields

| CRD Field | API Field | Description |
|-----------|-----------|-------------|
| `name` | `name` | Profile name |
| `minPopularity` | `minPopularity` | Minimum Goodreads popularity |
| `skipMissingDate` | `skipMissingDate` | Skip books without release date |
| `skipMissingIsbn` | `skipMissingIsbn` | Skip books without ISBN |
| `skipPartsAndSets` | `skipPartsAndSets` | Skip parts and box sets |
| `skipSeriesSecondary` | `skipSeriesSecondary` | Skip secondary series entries |
| `allowedLanguages` | `allowedLanguages` | List of allowed language codes |

### 3.3 Go Implementation

```go
// internal/adapters/readarr/metadata_profile.go

package readarr

import (
    "context"
    "fmt"
)

// MetadataProfileAPI represents the Readarr metadata profile API response
type MetadataProfileAPI struct {
    ID                  int      `json:"id,omitempty"`
    Name                string   `json:"name"`
    MinPopularity       int      `json:"minPopularity"`
    SkipMissingDate     bool     `json:"skipMissingDate"`
    SkipMissingIsbn     bool     `json:"skipMissingIsbn"`
    SkipPartsAndSets    bool     `json:"skipPartsAndSets"`
    SkipSeriesSecondary bool     `json:"skipSeriesSecondary"`
    AllowedLanguages    []string `json:"allowedLanguages,omitempty"`
}

// GetMetadataProfiles fetches all metadata profiles
func (c *Client) GetMetadataProfiles(ctx context.Context) ([]MetadataProfileAPI, error) {
    var profiles []MetadataProfileAPI
    if err := c.get(ctx, "/api/v1/metadataprofile", &profiles); err != nil {
        return nil, fmt.Errorf("failed to get metadata profiles: %w", err)
    }
    return profiles, nil
}

// CreateMetadataProfile creates a new metadata profile
func (c *Client) CreateMetadataProfile(ctx context.Context, profile MetadataProfileAPI) (*MetadataProfileAPI, error) {
    var result MetadataProfileAPI
    if err := c.post(ctx, "/api/v1/metadataprofile", profile, &result); err != nil {
        return nil, fmt.Errorf("failed to create metadata profile: %w", err)
    }
    return &result, nil
}

// UpdateMetadataProfile updates an existing metadata profile
func (c *Client) UpdateMetadataProfile(ctx context.Context, profile MetadataProfileAPI) error {
    endpoint := fmt.Sprintf("/api/v1/metadataprofile/%d", profile.ID)
    if err := c.put(ctx, endpoint, profile, nil); err != nil {
        return fmt.Errorf("failed to update metadata profile: %w", err)
    }
    return nil
}
```

---

## 4. Naming Configuration

### 4.1 Readarr Naming Variables

Readarr uses different naming variables than video apps:

| Variable | Description | Example |
|----------|-------------|---------|
| `{Author Name}` | Author's name | `Stephen King` |
| `{Author CleanName}` | Clean author name | `Stephen King` |
| `{Author SortName}` | Sortable author name | `King, Stephen` |
| `{Book Title}` | Book title | `The Shining` |
| `{Book CleanTitle}` | Clean book title | `The Shining` |
| `{Release Year}` | Year published | `1977` |
| `{Edition Year}` | Edition year | `2012` |
| `{Quality Full}` | Full quality | `EPUB` |
| `{Quality Title}` | Quality title | `EPUB` |
| `{Series Title}` | Series name | `The Dark Tower` |
| `{Series Position}` | Position in series | `1` |

### 4.2 Naming Presets

| Preset | Author Folder | Book File |
|--------|---------------|-----------|
| `standard` | `{Author Name}` | `{Book Title} ({Release Year})` |
| `plex-friendly` | `{Author Name}` | `{Author Name} - {Book Title}` |
| `calibre-style` | `{Author SortName}` | `{Book Title} - {Author Name}` |

---

## 5. Import Lists

### 5.1 Goodreads Lists

Readarr supports Goodreads import lists:

| List Type | API Type | Description |
|-----------|----------|-------------|
| `goodreads-shelf` | `GoodreadsBookshelfImportList` | User's bookshelf |
| `goodreads-list` | `GoodreadsListImportList` | Curated lists |
| `goodreads-owned` | `GoodreadsOwnedBooksImportList` | Owned books |
| `goodreads-series` | `GoodreadsSeriesImportList` | Book series |

### 5.2 Other Import Lists

| List Type | API Type | Description |
|-----------|----------|-------------|
| `lazylibrarian` | `LazyLibrarianImportList` | LazyLibrarian integration |
| `readarr` | `ReadarrImportList` | Another Readarr instance |

---

## 6. CRD Example

```yaml
apiVersion: arr.rinzler.cloud/v1alpha1
kind: ReadarrConfig
metadata:
  name: readarr-main
spec:
  connection:
    url: http://readarr.media.svc.cluster.local:8787
    apiKeySecretRef:
      name: readarr-credentials
      key: apiKey

  # Quality profile for book formats
  quality:
    preset: standard
    # Or custom:
    # allowedFormats:
    #   - format: EPUB
    #     allowed: true
    #   - format: MOBI
    #     allowed: true
    #   - format: FLAC
    #     allowed: true
    upgradeAllowed: true
    cutoff: EPUB

  # Metadata profile for Goodreads
  metadataProfile:
    name: standard
    minPopularity: 0
    skipMissingDate: true
    skipMissingIsbn: false
    skipPartsAndSets: false
    skipSeriesSecondary: false
    allowedLanguages:
      - en
      - es

  # Root folders for books
  rootFolders:
    - /books
    - /audiobooks

  # Download clients
  downloadClients:
    - name: qbittorrent
      type: qbittorrent
      url: http://qbittorrent.media.svc.cluster.local:8080
      credentialsSecretRef:
        name: qbittorrent-credentials
        usernameKey: username
        passwordKey: password
      category: books
      priority: 1

  # Use Prowlarr for indexers
  indexers:
    prowlarrRef:
      name: prowlarrconfig-main
      autoRegister: true

  # Naming conventions
  naming:
    preset: plex-friendly
    # Or custom:
    # authorFolderFormat: "{Author Name}"
    # standardBookFormat: "{Author Name} - {Book Title} ({Release Year})"

  # Goodreads import
  importLists:
    - name: my-to-read
      type: goodreads-shelf
      listType: to-read
      enabled: true
      enableAutomaticAdd: true
      qualityProfileId: 1
      metadataProfileId: 1
      rootFolderPath: /books
      settingsSecretRef:
        name: goodreads-credentials

  # Media management
  mediaManagement:
    deleteEmptyFolders: true
    createEmptyFolders: false
    minimumFreeSpace: 100

  reconciliation:
    interval: 5m
    suspend: false
```

---

## 7. API Reference

### 7.1 Quality Profile API

```
GET /api/v1/qualityprofile
POST /api/v1/qualityprofile
PUT /api/v1/qualityprofile/{id}
DELETE /api/v1/qualityprofile/{id}
```

### 7.2 Metadata Profile API

```
GET /api/v1/metadataprofile
POST /api/v1/metadataprofile
PUT /api/v1/metadataprofile/{id}
DELETE /api/v1/metadataprofile/{id}
```

### 7.3 Root Folder API

```
GET /api/v1/rootfolder
POST /api/v1/rootfolder
DELETE /api/v1/rootfolder/{id}
```

### 7.4 Download Client API

```
GET /api/v1/downloadclient
POST /api/v1/downloadclient
PUT /api/v1/downloadclient/{id}
DELETE /api/v1/downloadclient/{id}
```

### 7.5 Naming Configuration API

```
GET /api/v1/config/naming
PUT /api/v1/config/naming
```

---

## 8. Differences from Other *arr Apps

| Feature | Readarr | Radarr/Sonarr/Lidarr |
|---------|---------|---------------------|
| Quality concept | File format | Resolution + source |
| Metadata source | Goodreads | TMDb/TVDb/MusicBrainz |
| Organization | By author | By title/artist |
| Metadata profiles | Yes (unique) | No |
| Content types | Books + Audiobooks | Video/Audio only |

---

## 9. Common Issues

### 9.1 Goodreads Rate Limiting

Goodreads has strict rate limits. Configure metadata profiles carefully:
- Set `minPopularity` to reduce API calls
- Use `skipMissingDate: true` to reduce matches

### 9.2 Mixed eBook/Audiobook Libraries

For libraries with both formats:
- Use separate root folders: `/books` and `/audiobooks`
- Create distinct quality profiles for each type
- Consider separate ReadarrConfig resources for cleaner separation
