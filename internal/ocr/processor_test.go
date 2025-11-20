package ocr

import (
	"os"
	"strings"
	"testing"
)

func TestParseSidecarBytes(t *testing.T) {
	input := "Hello Page 1\nLine 2\f\fPage 3 Content\nLine B"
	result := parseSidecarBytes([]byte(input))

	if len(result) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(result))
	}
	if result[0].Page != 1 || !strings.Contains(result[0].Content, "Page 1") {
		t.Fatalf("unexpected first page: %+v", result[0])
	}
	if result[1].Page != 3 || !strings.Contains(result[1].Content, "Page 3") {
		t.Fatalf("unexpected third page: %+v", result[1])
	}
}

func TestParseSidecarBytes_NormalizesNewlinesAndSkipsEmpty(t *testing.T) {
	input := "Line 1\r\nLine 2\f\f\fLast Page"
	result := parseSidecarBytes([]byte(input))

	if len(result) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(result))
	}
	if result[0].Content != "Line 1\nLine 2" {
		t.Fatalf("unexpected newline normalization: %q", result[0].Content)
	}
	if result[1].Page != 4 {
		t.Fatalf("expected last page to be 4 due to skipped empties, got %d", result[1].Page)
	}
}

func TestParseSidecar_ErrorsWhenMissingFile(t *testing.T) {
	_, err := parseSidecar("missing-file.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseSidecar_ReadsFile(t *testing.T) {
	tmp, err := os.CreateTemp("", "sidecar-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmp.Name())
	content := "A page"
	if _, err := tmp.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmp.Close()

	result, err := parseSidecar(tmp.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 || result[0].Content != content {
		t.Fatalf("unexpected parsed content: %+v", result)
	}
}

func TestHasSignificantText(t *testing.T) {
	p := NewProcessor()

	tests := []struct {
		name      string
		text      string
		threshold int
		expected  bool
	}{
		{
			name:      "text exceeds threshold",
			text:      "Hello World, this is a test document with enough characters",
			threshold: 50,
			expected:  true,
		},
		{
			name:      "text below threshold",
			text:      "Short",
			threshold: 50,
			expected:  false,
		},
		{
			name:      "empty text",
			text:      "",
			threshold: 50,
			expected:  false,
		},
		{
			name:      "whitespace only",
			text:      "   \n\t\r   ",
			threshold: 1,
			expected:  false,
		},
		{
			name:      "exact threshold",
			text:      "12345678901234567890123456789012345678901234567890", // 50 chars
			threshold: 50,
			expected:  true,
		},
		{
			name:      "text with whitespace counted correctly",
			text:      "a b c d e f g h i j", // 10 letters, 9 spaces = 10 chars without spaces
			threshold: 10,
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.hasSignificantText(tt.text, tt.threshold)
			if result != tt.expected {
				t.Errorf("hasSignificantText(%q, %d) = %v, want %v", tt.text, tt.threshold, result, tt.expected)
			}
		})
	}
}

func TestNormalizeNewlines(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello\r\nWorld", "Hello\nWorld"},
		{"No change", "No change"},
		{"Multiple\r\nLine\r\nBreaks", "Multiple\nLine\nBreaks"},
		{"", ""},
	}

	for _, tt := range tests {
		result := normalizeNewlines(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeNewlines(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestNewProcessor(t *testing.T) {
	p := NewProcessor()
	if p.Binary != "ocrmypdf" {
		t.Errorf("expected binary 'ocrmypdf', got %q", p.Binary)
	}
	if p.Timeout != 2*60*1000000000 { // 2 minutes in nanoseconds
		t.Errorf("expected timeout 2 minutes, got %v", p.Timeout)
	}
}

func TestOptionsDefaults(t *testing.T) {
	opts := Options{}

	// TextThreshold should default to 0 (will be set to 50 in ExtractText)
	if opts.TextThreshold != 0 {
		t.Errorf("expected TextThreshold 0 by default, got %d", opts.TextThreshold)
	}

	// ForceOCR should default to false
	if opts.ForceOCR {
		t.Error("expected ForceOCR false by default")
	}

	// RemoveWatermark should default to false (but treated as true in logic)
	if opts.RemoveWatermark {
		t.Error("expected RemoveWatermark false by default")
	}
}
