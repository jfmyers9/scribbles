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

const negativeCacheTTL = 10 * time.Minute

type cacheEntry struct {
	url     string
	expires time.Time // zero value means never expires
}

// artworkLookup fetches album artwork URLs from the iTunes Search API
// and caches results to avoid repeated lookups for the same album.
type artworkLookup struct {
	mu       sync.Mutex
	cache    map[string]cacheEntry
	client   *http.Client
	endpoint string
	now      func() time.Time
}

func newArtworkLookup() *artworkLookup {
	return &artworkLookup{
		cache: make(map[string]cacheEntry),
		client: &http.Client{
			Timeout: 3 * time.Second,
		},
		endpoint: "https://itunes.apple.com/search",
		now:      time.Now,
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
	now := a.now()

	a.mu.Lock()
	if entry, ok := a.cache[key]; ok {
		if entry.expires.IsZero() || now.Before(entry.expires) {
			a.mu.Unlock()
			return entry.url
		}
		delete(a.cache, key)
	}
	a.mu.Unlock()

	artURL := a.fetch(artist, album)

	a.mu.Lock()
	entry := cacheEntry{url: artURL}
	if artURL == "" {
		entry.expires = now.Add(negativeCacheTTL)
	}
	a.cache[key] = entry
	a.mu.Unlock()

	return artURL
}

func (a *artworkLookup) fetch(artist, album string) string {
	// Try album entity first, fall back to song entity.
	// Some albums only appear in iTunes search as song results.
	for _, entity := range []string{"album", "song"} {
		if artURL := a.search(artist+" "+album, entity); artURL != "" {
			return artURL
		}
	}
	return ""
}

func (a *artworkLookup) search(term, entity string) string {
	query := url.Values{
		"term":   {term},
		"entity": {entity},
		"limit":  {"1"},
	}
	resp, err := a.client.Get(fmt.Sprintf("%s?%s", a.endpoint, query.Encode()))
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

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
