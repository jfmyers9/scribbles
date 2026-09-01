package music

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	mprisPrefix          = "org.mpris.MediaPlayer2."
	mprisPath            = dbus.ObjectPath("/org/mpris/MediaPlayer2")
	mprisPlayerInterface = "org.mpris.MediaPlayer2.Player"
	propertiesInterface  = "org.freedesktop.DBus.Properties"
)

// MPRISClient queries Linux media players through the MPRIS D-Bus interface.
type MPRISClient struct {
	mu            sync.Mutex
	lastPlayer    string
	durationCache map[string]time.Duration
	httpClient    *http.Client
	searchURL     string
}

// NewMPRISClient creates a client for media players on the Linux desktop.
func NewMPRISClient() *MPRISClient {
	return &MPRISClient{
		durationCache: make(map[string]time.Duration),
		httpClient:    &http.Client{Timeout: 3 * time.Second},
		searchURL:     "https://itunes.apple.com/search",
	}
}

func sessionBus() (*dbus.Conn, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("connect to session D-Bus: %w", err)
	}
	return conn, nil
}

func listMPRISPlayers(ctx context.Context, conn *dbus.Conn) ([]string, error) {
	obj := conn.Object("org.freedesktop.DBus", "/org/freedesktop/DBus")
	var names []string
	if err := obj.CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return nil, fmt.Errorf("list MPRIS players: %w", err)
	}

	players := make([]string, 0, len(names))
	for _, name := range names {
		if strings.HasPrefix(name, mprisPrefix) {
			players = append(players, name)
		}
	}
	return players, nil
}

// IsRunning reports whether any MPRIS media player is available.
func (c *MPRISClient) IsRunning(ctx context.Context) (bool, error) {
	conn, err := sessionBus()
	if err != nil {
		return false, err
	}
	players, err := listMPRISPlayers(ctx, conn)
	return len(players) > 0, err
}

type mprisTrack struct {
	player string
	track  *Track
}

// GetCurrentTrack returns the playing MPRIS track, preferring a playing player
// over paused players when multiple browser tabs or applications are present.
func (c *MPRISClient) GetCurrentTrack(ctx context.Context) (*Track, error) {
	conn, err := sessionBus()
	if err != nil {
		return nil, err
	}
	players, err := listMPRISPlayers(ctx, conn)
	if err != nil {
		return nil, err
	}

	var paused *mprisTrack
	for _, player := range players {
		track, err := c.getMPRISTrack(ctx, conn, player)
		if err != nil || track == nil {
			// Players can disappear while their browser tab is closing.
			continue
		}
		candidate := &mprisTrack{player: player, track: track}
		if track.State == StatePlaying {
			c.rememberPlayer(player)
			return track, nil
		}
		if track.State == StatePaused && paused == nil {
			paused = candidate
		}
	}

	if paused != nil {
		c.rememberPlayer(paused.player)
		return paused.track, nil
	}
	return nil, nil
}

func (c *MPRISClient) getMPRISTrack(ctx context.Context, conn *dbus.Conn, player string) (*Track, error) {
	obj := conn.Object(player, mprisPath)
	props := make(map[string]dbus.Variant)
	if err := obj.CallWithContext(ctx, propertiesInterface+".GetAll", 0, mprisPlayerInterface).Store(&props); err != nil {
		return nil, fmt.Errorf("read MPRIS properties from %s: %w", player, err)
	}
	track, err := trackFromMPRISProperties(props)
	if err != nil || track == nil || track.Duration > 0 || !isAppleMusicMetadata(props) {
		return track, err
	}

	track.Duration = c.appleMusicDuration(ctx, track)
	return track, nil
}

func isAppleMusicMetadata(props map[string]dbus.Variant) bool {
	metadata, ok := props["Metadata"].Value().(map[string]dbus.Variant)
	if !ok {
		return false
	}
	source, _ := variantString(metadata["xesam:url"])
	parsed, err := url.Parse(source)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "music.apple.com" || strings.HasSuffix(host, ".music.apple.com")
}

type iTunesSearchResponse struct {
	Results []struct {
		TrackName       string `json:"trackName"`
		ArtistName      string `json:"artistName"`
		CollectionName  string `json:"collectionName"`
		TrackTimeMillis int64  `json:"trackTimeMillis"`
	} `json:"results"`
}

func (c *MPRISClient) appleMusicDuration(ctx context.Context, track *Track) time.Duration {
	key := strings.ToLower(track.Artist + "\x00" + track.Name + "\x00" + track.Album)
	c.mu.Lock()
	duration, cached := c.durationCache[key]
	c.mu.Unlock()
	if cached {
		return duration
	}

	duration = c.searchAppleMusicDuration(ctx, track)
	c.mu.Lock()
	c.durationCache[key] = duration
	c.mu.Unlock()
	return duration
}

func (c *MPRISClient) searchAppleMusicDuration(ctx context.Context, track *Track) time.Duration {
	query, err := url.Parse(c.searchURL)
	if err != nil {
		return 0
	}
	params := query.Query()
	params.Set("term", track.Artist+" "+track.Name)
	params.Set("media", "music")
	params.Set("entity", "song")
	params.Set("limit", "25")
	query.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, query.String(), nil)
	if err != nil {
		return 0
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0
	}

	var result iTunesSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0
	}

	name := normalizeCatalogText(track.Name)
	artist := normalizeCatalogText(track.Artist)
	album := normalizeCatalogText(track.Album)
	var fallback int64
	for _, candidate := range result.Results {
		if normalizeCatalogText(candidate.TrackName) != name || normalizeCatalogText(candidate.ArtistName) != artist {
			continue
		}
		if candidate.TrackTimeMillis <= 0 {
			continue
		}
		if normalizeCatalogText(candidate.CollectionName) == album {
			return time.Duration(candidate.TrackTimeMillis) * time.Millisecond
		}
		if fallback == 0 {
			fallback = candidate.TrackTimeMillis
		}
	}
	return time.Duration(fallback) * time.Millisecond
}

func normalizeCatalogText(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func trackFromMPRISProperties(props map[string]dbus.Variant) (*Track, error) {
	status, _ := variantString(props["PlaybackStatus"])
	var state PlayState
	switch status {
	case "Playing":
		state = StatePlaying
	case "Paused":
		state = StatePaused
	case "Stopped", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("unknown MPRIS playback status %q", status)
	}

	metadata, ok := props["Metadata"].Value().(map[string]dbus.Variant)
	if !ok {
		return nil, fmt.Errorf("MPRIS metadata has unexpected type %T", props["Metadata"].Value())
	}

	name, _ := variantString(metadata["xesam:title"])
	artists := variantStrings(metadata["xesam:artist"])
	if name == "" || len(artists) == 0 {
		return nil, nil
	}
	album, _ := variantString(metadata["xesam:album"])
	durationMicros := variantInt64(metadata["mpris:length"])
	positionMicros := variantInt64(props["Position"])

	return &Track{
		Name:     name,
		Artist:   strings.Join(artists, ", "),
		Album:    album,
		Duration: time.Duration(durationMicros) * time.Microsecond,
		Position: time.Duration(positionMicros) * time.Microsecond,
		State:    state,
	}, nil
}

func variantString(v dbus.Variant) (string, bool) {
	value, ok := v.Value().(string)
	return value, ok
}

func variantStrings(v dbus.Variant) []string {
	switch value := v.Value().(type) {
	case []string:
		return value
	case string:
		if value != "" {
			return []string{value}
		}
	}
	return nil
}

func variantInt64(v dbus.Variant) int64 {
	switch value := v.Value().(type) {
	case int64:
		return value
	case uint64:
		if value <= uint64(^uint64(0)>>1) {
			return int64(value)
		}
	case int32:
		return int64(value)
	case uint32:
		return int64(value)
	}
	return 0
}

func (c *MPRISClient) rememberPlayer(player string) {
	c.mu.Lock()
	c.lastPlayer = player
	c.mu.Unlock()
}

func (c *MPRISClient) control(ctx context.Context, method string) error {
	conn, err := sessionBus()
	if err != nil {
		return err
	}

	c.mu.Lock()
	player := c.lastPlayer
	c.mu.Unlock()
	if player == "" {
		players, err := listMPRISPlayers(ctx, conn)
		if err != nil {
			return err
		}
		if len(players) == 0 {
			return fmt.Errorf("no MPRIS media player is running")
		}
		player = players[0]
	}

	obj := conn.Object(player, mprisPath)
	if call := obj.CallWithContext(ctx, mprisPlayerInterface+"."+method, 0); call.Err != nil {
		return fmt.Errorf("MPRIS %s on %s: %w", method, player, call.Err)
	}
	return nil
}

func (c *MPRISClient) Play(ctx context.Context) error          { return c.control(ctx, "Play") }
func (c *MPRISClient) Pause(ctx context.Context) error         { return c.control(ctx, "Pause") }
func (c *MPRISClient) PlayPause(ctx context.Context) error     { return c.control(ctx, "PlayPause") }
func (c *MPRISClient) NextTrack(ctx context.Context) error     { return c.control(ctx, "Next") }
func (c *MPRISClient) PreviousTrack(ctx context.Context) error { return c.control(ctx, "Previous") }
