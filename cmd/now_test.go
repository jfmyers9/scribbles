package cmd

import (
	"testing"

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
