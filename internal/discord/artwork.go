package discord

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// artworkLookup fetches album artwork URLs from the iTunes Search API
// and caches results to avoid repeated lookups for the same album.
type artworkLookup struct {
	mu       sync.Mutex
	cache    map[string]string
	client   *http.Client
	endpoint string
}

func newArtworkLookup() *artworkLookup {
	return &artworkLookup{
		cache: make(map[string]string),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		endpoint: "https://itunes.apple.com/search",
	}
}

type itunesResponse struct {
	Results []itunesResult `json:"results"`
}

type itunesResult struct {
	ArtworkURL100 string `json:"artworkUrl100"`
}

// Lookup returns an artwork URL for the given artist and album.
// Returns empty string on any failure — callers should treat artwork
// as optional.
func (a *artworkLookup) Lookup(artist, album string) string {
	key := artist + "|" + album
	a.mu.Lock()
	if url, ok := a.cache[key]; ok {
		a.mu.Unlock()
		return url
	}
	a.mu.Unlock()

	artURL := a.fetch(artist, album)

	a.mu.Lock()
	a.cache[key] = artURL
	a.mu.Unlock()

	return artURL
}

func (a *artworkLookup) fetch(artist, album string) string {
	query := url.Values{
		"term":   {artist + " " + album},
		"entity": {"album"},
		"limit":  {"1"},
	}
	resp, err := a.client.Get(fmt.Sprintf("%s?%s", a.endpoint, query.Encode()))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result itunesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	if len(result.Results) == 0 || result.Results[0].ArtworkURL100 == "" {
		return ""
	}

	// Upscale from 100x100 to 600x600 for better quality
	return strings.Replace(result.Results[0].ArtworkURL100, "100x100bb", "600x600bb", 1)
}
