package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestArtworkLookup_ReturnsUpscaledURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(itunesResponse{
			Results: []itunesResult{
				{ArtworkURL100: "https://example.com/art/100x100bb.jpg"},
			},
		})
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	got := a.Lookup("Queen", "A Night at the Opera")
	want := "https://example.com/art/600x600bb.jpg"
	if got != want {
		t.Errorf("Lookup() = %q, want %q", got, want)
	}
}

func TestArtworkLookup_CachesResults(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(itunesResponse{
			Results: []itunesResult{
				{ArtworkURL100: "https://example.com/art/100x100bb.jpg"},
			},
		})
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	a.Lookup("Queen", "A Night at the Opera")
	a.Lookup("Queen", "A Night at the Opera")
	a.Lookup("Queen", "A Night at the Opera")

	if n := hits.Load(); n != 1 {
		t.Errorf("expected 1 HTTP request, got %d", n)
	}
}

func TestArtworkLookup_FallsBackToSongEntity(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		entity := r.URL.Query().Get("entity")
		if entity == "album" {
			_ = json.NewEncoder(w).Encode(itunesResponse{Results: nil})
			return
		}
		_ = json.NewEncoder(w).Encode(itunesResponse{
			Results: []itunesResult{
				{ArtworkURL100: "https://example.com/art/100x100bb.jpg"},
			},
		})
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	got := a.Lookup("Ninajirachi", "I Love My Computer")
	want := "https://example.com/art/600x600bb.jpg"
	if got != want {
		t.Errorf("Lookup() = %q, want %q", got, want)
	}
}

func TestArtworkLookup_EmptyOnNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(itunesResponse{Results: nil})
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	if got := a.Lookup("Unknown", "Album"); got != "" {
		t.Errorf("expected empty string for no results, got %q", got)
	}
}

func TestArtworkLookup_EmptyOnHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	if got := a.Lookup("Artist", "Album"); got != "" {
		t.Errorf("expected empty string on HTTP error, got %q", got)
	}
}

func TestArtworkLookup_EmptyOnUnreachable(t *testing.T) {
	a := newArtworkLookup()
	a.endpoint = "http://127.0.0.1:1" // nothing listening

	if got := a.Lookup("Artist", "Album"); got != "" {
		t.Errorf("expected empty string on connection error, got %q", got)
	}
}

func TestArtworkLookup_NegativeCacheWithTTL(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(itunesResponse{Results: nil})
	}))
	defer srv.Close()

	now := time.Now()
	a := newArtworkLookup()
	a.endpoint = srv.URL
	a.now = func() time.Time { return now }

	// First lookup misses (album + song fallback) — cached as negative
	if got := a.Lookup("Unknown", "Album"); got != "" {
		t.Errorf("first lookup: expected empty, got %q", got)
	}
	firstHits := hits.Load()

	// Second lookup within TTL — served from negative cache, no new requests
	if got := a.Lookup("Unknown", "Album"); got != "" {
		t.Errorf("within TTL: expected empty, got %q", got)
	}
	if n := hits.Load(); n != firstHits {
		t.Errorf("expected no new requests within TTL, got %d more", n-firstHits)
	}

	// Advance past TTL — negative cache expires, retries
	now = now.Add(negativeCacheTTL + time.Second)
	a.Lookup("Unknown", "Album")
	if n := hits.Load(); n == firstHits {
		t.Error("expected new requests after TTL expiry, got none")
	}
}
