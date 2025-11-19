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
