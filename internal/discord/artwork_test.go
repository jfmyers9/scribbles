package discord

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
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

func TestArtworkLookup_CachesEmptyResult(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(itunesResponse{Results: nil})
	}))
	defer srv.Close()

	a := newArtworkLookup()
	a.endpoint = srv.URL

	a.Lookup("Unknown", "Album")
	a.Lookup("Unknown", "Album")

	if n := hits.Load(); n != 1 {
		t.Errorf("expected 1 HTTP request for cached empty result, got %d", n)
	}
}
