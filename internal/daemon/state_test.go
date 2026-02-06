package daemon

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jfmyers9/scribbles/internal/music"
)

func newTestState(t *testing.T, interval time.Duration) *State {
	t.Helper()
	dir := t.TempDir()
	fp := filepath.Join(dir, "state.json")
	s, err := NewState(fp)
	if err != nil {
		t.Fatalf("NewState: %v", err)
	}
	s.persistInterval = interval
	return s
}

func TestThrottledPersist_SkipsWhenIntervalNotElapsed(t *testing.T) {
	s := newTestState(t, 1*time.Hour) // very long interval

	// Seed a track so persist creates the file initially
	track := &music.Track{
		Name:   "Song A",
		Artist: "Artist A",
		Album:  "Album A",
		State:  music.StatePlaying,
	}
	if err := s.SetTrack(track); err != nil {
		t.Fatalf("SetTrack: %v", err)
	}

	// Record mod time after initial persist
	info1, err := os.Stat(s.filePath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Call throttledPersist (via UpdatePosition with same track)
	// This should NOT write because the interval hasn't elapsed
	s.mu.Lock()
	err = s.throttledPersist()
	s.mu.Unlock()
	if err != nil {
		t.Fatalf("throttledPersist: %v", err)
	}

	info2, err := os.Stat(s.filePath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if info2.ModTime() != info1.ModTime() {
		t.Error("throttledPersist wrote to disk when interval had not elapsed")
	}

	// dirty flag should be set
	s.mu.Lock()
	dirty := s.dirty
	s.mu.Unlock()
	if !dirty {
		t.Error("expected dirty flag to be true after throttledPersist skip")
	}
}

func TestThrottledPersist_WritesWhenIntervalElapsed(t *testing.T) {
	s := newTestState(t, 10*time.Millisecond) // very short interval

	track := &music.Track{
		Name:   "Song B",
		Artist: "Artist B",
		Album:  "Album B",
		State:  music.StatePlaying,
	}
	if err := s.SetTrack(track); err != nil {
		t.Fatalf("SetTrack: %v", err)
	}

	// Wait for interval to elapse
	time.Sleep(20 * time.Millisecond)

	// throttledPersist should write now
	s.mu.Lock()
	s.dirty = true // ensure dirty
	err := s.throttledPersist()
	s.mu.Unlock()
	if err != nil {
		t.Fatalf("throttledPersist: %v", err)
	}

	// dirty flag should be cleared after successful persist
	s.mu.Lock()
	dirty := s.dirty
	s.mu.Unlock()
	if dirty {
		t.Error("expected dirty flag to be false after throttledPersist write")
	}
}

func TestFlush_WritesWhenDirty(t *testing.T) {
	s := newTestState(t, 1*time.Hour)

	track := &music.Track{
		Name:   "Song C",
		Artist: "Artist C",
		Album:  "Album C",
		State:  music.StatePlaying,
	}
	if err := s.SetTrack(track); err != nil {
		t.Fatalf("SetTrack: %v", err)
	}

	// Manually set dirty to simulate throttled skip
	s.mu.Lock()
	s.dirty = true
	s.mu.Unlock()

	// Record current file content
	before, err := os.ReadFile(s.filePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	// Modify state to produce different output
	s.mu.Lock()
	s.current.Scrobbled = true
	s.dirty = true
	s.mu.Unlock()

	// Flush should write
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	after, err := os.ReadFile(s.filePath)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(before) == string(after) {
		t.Error("Flush did not write updated state to disk")
	}

	// dirty flag should be cleared
	s.mu.Lock()
	dirty := s.dirty
	s.mu.Unlock()
	if dirty {
		t.Error("expected dirty flag to be false after Flush")
	}
}

func TestFlush_NoOpWhenClean(t *testing.T) {
	s := newTestState(t, 1*time.Hour)

	track := &music.Track{
		Name:   "Song D",
		Artist: "Artist D",
		Album:  "Album D",
		State:  music.StatePlaying,
	}
	if err := s.SetTrack(track); err != nil {
		t.Fatalf("SetTrack: %v", err)
	}

	// dirty should be false after SetTrack (it calls persist directly)
	s.mu.Lock()
	dirty := s.dirty
	s.mu.Unlock()
	if dirty {
		t.Fatal("expected dirty=false after SetTrack persist")
	}

	info1, err := os.Stat(s.filePath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	// Flush on clean state should be no-op
	if err := s.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	info2, err := os.Stat(s.filePath)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if info2.ModTime() != info1.ModTime() {
		t.Error("Flush wrote to disk when state was clean")
	}
}
