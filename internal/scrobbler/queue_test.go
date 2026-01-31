package scrobbler

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"
)

// createTestQueue creates an in-memory SQLite queue for testing
func createTestQueue(t *testing.T) *Queue {
	t.Helper()

	// Use in-memory database for tests
	queue, err := NewQueue(":memory:")
	if err != nil {
		t.Fatalf("failed to create test queue: %v", err)
	}

	t.Cleanup(func() {
		_ = queue.Close()
	})

	return queue
}

func TestNewQueue(t *testing.T) {
	t.Run("in-memory database", func(t *testing.T) {
		queue, err := NewQueue(":memory:")
		if err != nil {
			t.Fatalf("failed to create in-memory queue: %v", err)
		}
		defer func() { _ = queue.Close() }()

		if queue.db == nil {
			t.Error("queue database is nil")
		}
	})

	t.Run("file-based database", func(t *testing.T) {
		tmpfile, err := os.CreateTemp("", "scrobbles-test-*.db")
		if err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}
		_ = tmpfile.Close()
		defer func() { _ = os.Remove(tmpfile.Name()) }()

		queue, err := NewQueue(tmpfile.Name())
		if err != nil {
			t.Fatalf("failed to create file-based queue: %v", err)
		}
		defer func() { _ = queue.Close() }()

		if queue.db == nil {
			t.Error("queue database is nil")
		}
	})
}

func TestQueueAdd(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	scrobble := Scrobble{
		Artist:    "Test Artist",
		Track:     "Test Track",
		Album:     "Test Album",
		Duration:  3 * time.Minute,
		Timestamp: time.Now(),
	}

	id, err := queue.Add(ctx, scrobble)
	if err != nil {
		t.Fatalf("failed to add scrobble: %v", err)
	}

	if id <= 0 {
		t.Errorf("expected positive id, got %d", id)
	}

	// Verify it was added
	count, err := queue.Count(ctx, false)
	if err != nil {
		t.Fatalf("failed to count scrobbles: %v", err)
	}

	if count != 1 {
		t.Errorf("expected 1 pending scrobble, got %d", count)
	}
}

func TestQueueMarkScrobbled(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	scrobble := Scrobble{
		Artist:    "Test Artist",
		Track:     "Test Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now(),
	}

	id, err := queue.Add(ctx, scrobble)
	if err != nil {
		t.Fatalf("failed to add scrobble: %v", err)
	}

	// Mark as scrobbled
	err = queue.MarkScrobbled(ctx, id)
	if err != nil {
		t.Fatalf("failed to mark scrobbled: %v", err)
	}

	// Verify pending count is now 0
	count, err := queue.Count(ctx, false)
	if err != nil {
		t.Fatalf("failed to count pending scrobbles: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 pending scrobbles, got %d", count)
	}

	// Verify total count is still 1
	totalCount, err := queue.Count(ctx, true)
	if err != nil {
		t.Fatalf("failed to count total scrobbles: %v", err)
	}

	if totalCount != 1 {
		t.Errorf("expected 1 total scrobble, got %d", totalCount)
	}
}

func TestQueueMarkScrobbledBatch(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add multiple scrobbles
	var ids []int64
	for i := 0; i < 5; i++ {
		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Duration:  3 * time.Minute,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}

		id, err := queue.Add(ctx, scrobble)
		if err != nil {
			t.Fatalf("failed to add scrobble: %v", err)
		}
		ids = append(ids, id)
	}

	// Mark first 3 as scrobbled
	err := queue.MarkScrobbledBatch(ctx, ids[:3])
	if err != nil {
		t.Fatalf("failed to mark batch scrobbled: %v", err)
	}

	// Verify pending count
	count, err := queue.Count(ctx, false)
	if err != nil {
		t.Fatalf("failed to count pending scrobbles: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 pending scrobbles, got %d", count)
	}
}

func TestQueueMarkError(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	scrobble := Scrobble{
		Artist:    "Test Artist",
		Track:     "Test Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now(),
	}

	id, err := queue.Add(ctx, scrobble)
	if err != nil {
		t.Fatalf("failed to add scrobble: %v", err)
	}

	// Mark with error
	errMsg := "network timeout"
	err = queue.MarkError(ctx, id, errMsg)
	if err != nil {
		t.Fatalf("failed to mark error: %v", err)
	}

	// Verify it's still pending
	pending, err := queue.GetPending(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get pending: %v", err)
	}

	if len(pending) != 1 {
		t.Fatalf("expected 1 pending scrobble, got %d", len(pending))
	}

	if pending[0].Error != errMsg {
		t.Errorf("expected error %q, got %q", errMsg, pending[0].Error)
	}
}

func TestQueueGetPending(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add scrobbles with different timestamps
	timestamps := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
	}

	for i, ts := range timestamps {
		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Album:     "Album",
			Duration:  3 * time.Minute,
			Timestamp: ts,
		}

		id, err := queue.Add(ctx, scrobble)
		if err != nil {
			t.Fatalf("failed to add scrobble %d: %v", i, err)
		}

		// Mark the second one as scrobbled
		if i == 1 {
			if err := queue.MarkScrobbled(ctx, id); err != nil {
				t.Fatalf("failed to mark scrobbled: %v", err)
			}
		}
	}

	// Get pending (should exclude the scrobbled one)
	pending, err := queue.GetPending(ctx, 0)
	if err != nil {
		t.Fatalf("failed to get pending: %v", err)
	}

	if len(pending) != 2 {
		t.Fatalf("expected 2 pending scrobbles, got %d", len(pending))
	}

	// Verify they're ordered by timestamp (oldest first)
	if !pending[0].Timestamp.Before(pending[1].Timestamp) {
		t.Error("scrobbles are not ordered by timestamp")
	}
}

func TestQueueGetPendingWithLimit(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add 5 scrobbles
	for i := 0; i < 5; i++ {
		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Duration:  3 * time.Minute,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}

		_, err := queue.Add(ctx, scrobble)
		if err != nil {
			t.Fatalf("failed to add scrobble: %v", err)
		}
	}

	// Get with limit of 3
	pending, err := queue.GetPending(ctx, 3)
	if err != nil {
		t.Fatalf("failed to get pending: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("expected 3 pending scrobbles, got %d", len(pending))
	}
}

func TestQueueCleanup(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add old scrobbled scrobble
	oldScrobble := Scrobble{
		Artist:    "Old Artist",
		Track:     "Old Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-10 * 24 * time.Hour), // 10 days ago
	}
	oldID, err := queue.Add(ctx, oldScrobble)
	if err != nil {
		t.Fatalf("failed to add old scrobble: %v", err)
	}
	if err := queue.MarkScrobbled(ctx, oldID); err != nil {
		t.Fatalf("failed to mark old scrobble: %v", err)
	}

	// Add recent scrobbled scrobble
	recentScrobble := Scrobble{
		Artist:    "Recent Artist",
		Track:     "Recent Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-1 * time.Hour),
	}
	recentID, err := queue.Add(ctx, recentScrobble)
	if err != nil {
		t.Fatalf("failed to add recent scrobble: %v", err)
	}
	if err := queue.MarkScrobbled(ctx, recentID); err != nil {
		t.Fatalf("failed to mark recent scrobble: %v", err)
	}

	// Add pending scrobble (should not be deleted)
	pendingScrobble := Scrobble{
		Artist:    "Pending Artist",
		Track:     "Pending Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-10 * 24 * time.Hour),
	}
	_, err = queue.Add(ctx, pendingScrobble)
	if err != nil {
		t.Fatalf("failed to add pending scrobble: %v", err)
	}

	// Cleanup scrobbles older than 7 days
	deleted, err := queue.Cleanup(ctx, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to cleanup: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted scrobble, got %d", deleted)
	}

	// Verify total count is 2 (recent scrobbled + old pending)
	count, err := queue.Count(ctx, true)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 remaining scrobbles, got %d", count)
	}
}

func TestQueueCleanupOldFailed(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add old failed scrobble (older than 2 weeks)
	oldScrobble := Scrobble{
		Artist:    "Old Artist",
		Track:     "Old Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-15 * 24 * time.Hour),
	}
	oldID, err := queue.Add(ctx, oldScrobble)
	if err != nil {
		t.Fatalf("failed to add old scrobble: %v", err)
	}
	if err := queue.MarkError(ctx, oldID, "network error"); err != nil {
		t.Fatalf("failed to mark old scrobble error: %v", err)
	}

	// Add recent failed scrobble
	recentScrobble := Scrobble{
		Artist:    "Recent Artist",
		Track:     "Recent Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-1 * time.Hour),
	}
	recentID, err := queue.Add(ctx, recentScrobble)
	if err != nil {
		t.Fatalf("failed to add recent scrobble: %v", err)
	}
	if err := queue.MarkError(ctx, recentID, "network error"); err != nil {
		t.Fatalf("failed to mark recent scrobble error: %v", err)
	}

	// Add old pending without error (should not be deleted)
	oldPendingScrobble := Scrobble{
		Artist:    "Old Pending",
		Track:     "Old Pending Track",
		Duration:  3 * time.Minute,
		Timestamp: time.Now().Add(-15 * 24 * time.Hour),
	}
	_, err = queue.Add(ctx, oldPendingScrobble)
	if err != nil {
		t.Fatalf("failed to add old pending scrobble: %v", err)
	}

	// Cleanup old failed scrobbles
	deleted, err := queue.CleanupOldFailed(ctx)
	if err != nil {
		t.Fatalf("failed to cleanup old failed: %v", err)
	}

	if deleted != 1 {
		t.Errorf("expected 1 deleted scrobble, got %d", deleted)
	}

	// Verify remaining count is 2
	count, err := queue.Count(ctx, true)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 remaining scrobbles, got %d", count)
	}
}

func TestQueueConcurrentAccess(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	var errMutex sync.Mutex
	var errors []error
	numGoroutines := 10
	numScrobblesPerGoroutine := 10

	// Concurrently add scrobbles
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numScrobblesPerGoroutine; j++ {
				scrobble := Scrobble{
					Artist:    "Artist",
					Track:     "Track",
					Duration:  3 * time.Minute,
					Timestamp: time.Now(),
				}

				_, err := queue.Add(ctx, scrobble)
				if err != nil {
					errMutex.Lock()
					errors = append(errors, err)
					errMutex.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	// Check for errors from goroutines
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("concurrent add error: %v", err)
		}
		t.FailNow()
	}

	// Verify all scrobbles were added
	expectedCount := numGoroutines * numScrobblesPerGoroutine
	count, err := queue.Count(ctx, false)
	if err != nil {
		t.Fatalf("failed to count: %v", err)
	}

	if count != expectedCount {
		t.Errorf("expected %d scrobbles, got %d", expectedCount, count)
	}
}

func TestQueueGetAll(t *testing.T) {
	queue := createTestQueue(t)
	ctx := context.Background()

	// Add multiple scrobbles
	for i := 0; i < 3; i++ {
		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Duration:  3 * time.Minute,
			Timestamp: time.Now().Add(time.Duration(i) * time.Second),
		}

		id, err := queue.Add(ctx, scrobble)
		if err != nil {
			t.Fatalf("failed to add scrobble: %v", err)
		}

		// Mark first one as scrobbled
		if i == 0 {
			if err := queue.MarkScrobbled(ctx, id); err != nil {
				t.Fatalf("failed to mark scrobbled: %v", err)
			}
		}
	}

	// Get all (should include both scrobbled and pending)
	all, err := queue.GetAll(ctx)
	if err != nil {
		t.Fatalf("failed to get all: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("expected 3 scrobbles, got %d", len(all))
	}

	// Verify they're ordered by timestamp DESC
	for i := 0; i < len(all)-1; i++ {
		if all[i].Timestamp.Before(all[i+1].Timestamp) {
			t.Error("scrobbles are not ordered by timestamp DESC")
		}
	}
}

func TestQueueEdgeCases(t *testing.T) {
	t.Run("mark non-existent scrobble", func(t *testing.T) {
		queue := createTestQueue(t)
		ctx := context.Background()

		err := queue.MarkScrobbled(ctx, 999)
		if err == nil {
			t.Error("expected error when marking non-existent scrobble")
		}
	})

	t.Run("empty album", func(t *testing.T) {
		queue := createTestQueue(t)
		ctx := context.Background()

		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Album:     "", // Empty album
			Duration:  3 * time.Minute,
			Timestamp: time.Now(),
		}

		id, err := queue.Add(ctx, scrobble)
		if err != nil {
			t.Fatalf("failed to add scrobble with empty album: %v", err)
		}

		pending, err := queue.GetPending(ctx, 0)
		if err != nil {
			t.Fatalf("failed to get pending: %v", err)
		}

		if len(pending) != 1 {
			t.Fatalf("expected 1 pending scrobble, got %d", len(pending))
		}

		if pending[0].ID != id {
			t.Errorf("expected id %d, got %d", id, pending[0].ID)
		}
	})

	t.Run("mark scrobbled batch with empty list", func(t *testing.T) {
		queue := createTestQueue(t)
		ctx := context.Background()

		err := queue.MarkScrobbledBatch(ctx, []int64{})
		if err != nil {
			t.Errorf("expected no error for empty batch, got: %v", err)
		}
	})
}

// Benchmark tests
func BenchmarkQueueAdd(b *testing.B) {
	queue, _ := NewQueue(":memory:")
	defer func() { _ = queue.Close() }()
	ctx := context.Background()

	scrobble := Scrobble{
		Artist:    "Artist",
		Track:     "Track",
		Album:     "Album",
		Duration:  3 * time.Minute,
		Timestamp: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = queue.Add(ctx, scrobble)
	}
}

func BenchmarkQueueGetPending(b *testing.B) {
	queue, _ := NewQueue(":memory:")
	defer func() { _ = queue.Close() }()
	ctx := context.Background()

	// Add some scrobbles
	for i := 0; i < 100; i++ {
		scrobble := Scrobble{
			Artist:    "Artist",
			Track:     "Track",
			Duration:  3 * time.Minute,
			Timestamp: time.Now(),
		}
		_, _ = queue.Add(ctx, scrobble)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = queue.GetPending(ctx, 50)
	}
}
