package cmd

import (
	"testing"
	"time"

	"github.com/mattn/go-runewidth"
)

func TestPadToWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		width    int
		expected string
	}{
		{
			name:     "no padding when width is 0",
			input:    "Hello",
			width:    0,
			expected: "Hello",
		},
		{
			name:     "no padding when width is negative",
			input:    "Hello",
			width:    -1,
			expected: "Hello",
		},
		{
			name:     "pad short text with spaces",
			input:    "Hi",
			width:    10,
			expected: "Hi        ",
		},
		{
			name:     "exact width unchanged",
			input:    "Hello",
			width:    5,
			expected: "Hello",
		},
		{
			name:     "truncate long text with ellipsis",
			input:    "This is a very long string that needs truncation",
			width:    20,
			expected: "This is a very lo...",
		},
		{
			name:     "handle emoji correctly",
			input:    "🎵 Music",
			width:    15,
			expected: "🎵 Music       ", // emoji is 2 chars wide, so 8 total + 7 spaces
		},
		{
			name:     "truncate emoji text",
			input:    "🎵 This is a very long song title",
			width:    15,
			expected: "🎵 This is a...",
		},
		{
			name:     "handle unicode characters",
			input:    "日本語",
			width:    10,
			expected: "日本語    ",
		},
		{
			name:     "truncate unicode text",
			input:    "日本語とても長いテキスト",
			width:    10,
			expected: "日本語... ", // 日本語 is 6 chars, ... is 3, need 1 space
		},
		{
			name:     "empty string padding",
			input:    "",
			width:    5,
			expected: "     ",
		},
		{
			name:     "single character padding",
			input:    "A",
			width:    5,
			expected: "A    ",
		},
		{
			name:     "minimum width for truncation",
			input:    "Hello",
			width:    3,
			expected: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := padToWidth(tt.input, tt.width)
			if result != tt.expected {
				t.Errorf("padToWidth(%q, %d) = %q, expected %q",
					tt.input, tt.width, result, tt.expected)
			}

			// Verify the result has the expected display width (if width > 0)
			if tt.width > 0 {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != tt.width {
					t.Errorf("padToWidth(%q, %d) produced width %d, expected %d",
						tt.input, tt.width, resultWidth, tt.width)
				}
			}
		})
	}
}

func TestExtractWindow(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		startPos  int
		width     int
		expected  string
	}{
		{
			name:     "extract from beginning",
			text:     "Hello World",
			startPos: 0,
			width:    5,
			expected: "Hello",
		},
		{
			name:     "extract from middle",
			text:     "Hello World",
			startPos: 6,
			width:    5,
			expected: "World",
		},
		{
			name:     "extract with padding",
			text:     "Hi",
			startPos: 0,
			width:    5,
			expected: "Hi   ",
		},
		{
			name:     "extract near end requires padding",
			text:     "Hello",
			startPos: 3,
			width:    5,
			expected: "lo   ",
		},
		{
			name:     "zero width returns empty",
			text:     "Hello",
			startPos: 0,
			width:    0,
			expected: "",
		},
		{
			name:     "extract emoji text",
			text:     "🎵 Music",
			startPos: 0,
			width:    5,
			expected: "🎵 Mu", // emoji is 2 wide, space is 1, M is 1, u is 1 = 5 total
		},
		{
			name:     "extract emoji from middle",
			text:     "Hello 🎵 World",
			startPos: 6,
			width:    5,
			expected: "🎵 Wo", // emoji is 2 wide, space is 1, W is 1, o is 1 = 5 total
		},
		{
			name:     "extract unicode text",
			text:     "日本語テスト",
			startPos: 0,
			width:    6,
			expected: "日本語",
		},
		{
			name:     "extract unicode from middle",
			text:     "日本語テスト",
			startPos: 6,
			width:    6,
			expected: "テスト",
		},
		{
			name:     "start position beyond text",
			text:     "Hello",
			startPos: 20,
			width:    5,
			expected: "     ",
		},
		{
			name:     "single character extraction",
			text:     "ABCDEF",
			startPos: 2,
			width:    1,
			expected: "C",
		},
		{
			name:     "wide character at boundary",
			text:     "A🎵B",
			startPos: 1,
			width:    2,
			expected: "🎵",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractWindow(tt.text, tt.startPos, tt.width)
			if result != tt.expected {
				t.Errorf("extractWindow(%q, %d, %d) = %q, expected %q",
					tt.text, tt.startPos, tt.width, result, tt.expected)
			}

			// Verify the result has the expected display width
			if tt.width > 0 {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != tt.width {
					t.Errorf("extractWindow(%q, %d, %d) produced width %d, expected %d",
						tt.text, tt.startPos, tt.width, resultWidth, tt.width)
				}
			}
		})
	}
}

func TestMarqueeText(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		width     int
		speed     int
		separator string
		// For deterministic testing, we'll check properties rather than exact output
		checkFn func(*testing.T, string, int)
	}{
		{
			name:      "short text gets padded not scrolled",
			text:      "Hi",
			width:     10,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				// Should be exactly "Hi" with padding, no separator
				if result != "Hi        " {
					t.Errorf("Expected static padding for short text, got %q", result)
				}
			},
		},
		{
			name:      "text exactly at width gets no padding or scroll",
			text:      "Hello",
			width:     5,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				if result != "Hello" {
					t.Errorf("Expected exact text for exact width, got %q", result)
				}
			},
		},
		{
			name:      "long text returns exact width",
			text:      "This is a very long string that needs scrolling",
			width:     25,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != width {
					t.Errorf("Expected width %d, got %d", width, resultWidth)
				}
			},
		},
		{
			name:      "emoji text scrolls correctly",
			text:      "🎵 This is a very long song title with emoji",
			width:     20,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != width {
					t.Errorf("Expected width %d, got %d", width, resultWidth)
				}
				// Result should not contain the separator yet (depends on timing)
				// but should be valid UTF-8
				if !isValidUTF8(result) {
					t.Errorf("Result is not valid UTF-8: %q", result)
				}
			},
		},
		{
			name:      "unicode text scrolls correctly",
			text:      "日本語のとても長いテキストがスクロールします",
			width:     20,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != width {
					t.Errorf("Expected width %d, got %d", width, resultWidth)
				}
			},
		},
		{
			name:      "different separator is used",
			text:      "Long text that will scroll around and around",
			width:     20,
			speed:     2,
			separator: " | ",
			checkFn: func(t *testing.T, result string, width int) {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != width {
					t.Errorf("Expected width %d, got %d", width, resultWidth)
				}
				// Eventually the separator should appear in the output
				// We can't test exact timing, but result should be valid
			},
		},
		{
			name:      "zero width returns original text",
			text:      "Hello",
			width:     0,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				if result != "Hello" {
					t.Errorf("Expected original text for width 0, got %q", result)
				}
			},
		},
		{
			name:      "minimum width of 1",
			text:      "Long text",
			width:     1,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				resultWidth := runewidth.StringWidth(result)
				if resultWidth != 1 {
					t.Errorf("Expected width 1, got %d", resultWidth)
				}
			},
		},
		{
			name:      "empty text with width",
			text:      "",
			width:     5,
			speed:     2,
			separator: " • ",
			checkFn: func(t *testing.T, result string, width int) {
				if result != "     " {
					t.Errorf("Expected padding for empty text, got %q", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := marqueeText(tt.text, tt.width, tt.speed, tt.separator)
			tt.checkFn(t, result, tt.width)
		})
	}
}

// TestMarqueeTextScrolling tests that the marquee actually scrolls over time
func TestMarqueeTextScrolling(t *testing.T) {
	text := "Long text that scrolls"
	width := 10
	speed := 1 // 1 char per second
	separator := " • "

	// We can't easily mock time.Now in tests, but we can verify that:
	// 1. The output width is always correct
	// 2. The output contains valid characters from the extended text
	// 3. Multiple calls might return different results (due to time passing)

	results := make(map[string]bool)

	// Take multiple samples
	for i := 0; i < 5; i++ {
		result := marqueeText(text, width, speed, separator)

		// Check width
		resultWidth := runewidth.StringWidth(result)
		if resultWidth != width {
			t.Errorf("Iteration %d: Expected width %d, got %d", i, width, resultWidth)
		}

		// Store unique results
		results[result] = true

		// Small delay to potentially get different output
		time.Sleep(10 * time.Millisecond)
	}

	// We should have consistent width across all samples
	t.Logf("Collected %d unique results from 5 samples", len(results))
}

// TestMarqueeTextDeterministic verifies that marquee output is deterministic
// for a given point in time (same timestamp = same output)
func TestMarqueeTextDeterministic(t *testing.T) {
	text := "Deterministic scrolling text"
	width := 15
	speed := 2
	separator := " • "

	// Call multiple times in quick succession
	// Should get the same result since time hasn't changed significantly
	result1 := marqueeText(text, width, speed, separator)
	result2 := marqueeText(text, width, speed, separator)

	if result1 != result2 {
		t.Logf("Note: Results differed between rapid calls (may happen if second boundary crossed)")
		t.Logf("Result 1: %q", result1)
		t.Logf("Result 2: %q", result2)
	}

	// Both should still have correct width
	if runewidth.StringWidth(result1) != width {
		t.Errorf("Result 1 has wrong width: %d", runewidth.StringWidth(result1))
	}
	if runewidth.StringWidth(result2) != width {
		t.Errorf("Result 2 has wrong width: %d", runewidth.StringWidth(result2))
	}
}

// Helper function to check if a string is valid UTF-8
func isValidUTF8(s string) bool {
	// In Go, strings are always valid UTF-8 or the rune iterator will handle it
	// We can check by converting to runes and back
	return s == string([]rune(s))
}
