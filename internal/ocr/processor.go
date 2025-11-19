package ocr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PageContent represents OCR text for a single page.
type PageContent struct {
	Page    int    `json:"page"`
	Content string `json:"content"`
}

// Options controls the OCR command invocation.
type Options struct {
	Language string
}

// Processor wraps OCRmyPDF CLI invocation.
type Processor struct {
	Binary  string
	Timeout time.Duration
}

// NewProcessor returns a Processor with sane defaults.
func NewProcessor() *Processor {
	return &Processor{
		Binary:  "ocrmypdf",
		Timeout: 2 * time.Minute,
	}
}

// ExtractText runs OCRmyPDF against pdfPath and returns cleaned page contents.
func (p *Processor) ExtractText(ctx context.Context, pdfPath string, opts Options) ([]PageContent, error) {
	if pdfPath == "" {
		return nil, errors.New("pdf path is required")
	}
	binary := p.Binary
	if binary == "" {
		binary = "ocrmypdf"
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	sidecarFile, err := os.CreateTemp("", "ocr-sidecar-*.txt")
	if err != nil {
		return nil, fmt.Errorf("create sidecar: %w", err)
	}
	defer os.Remove(sidecarFile.Name())
	defer sidecarFile.Close()

	outputPDF, err := os.CreateTemp("", "ocr-output-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("create temp output: %w", err)
	}
	defer os.Remove(outputPDF.Name())
	defer outputPDF.Close()

	args := []string{
		"--sidecar", sidecarFile.Name(),
		"--quiet",
		"--force-ocr",
		"--rotate-pages-threshold", "0.0",
		"--jobs", "20",
	}
	if opts.Language != "" {
		args = append(args, "--language", opts.Language)
	}
	args = append(args, pdfPath, outputPDF.Name())

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ocrmypdf: %w - %s", err, stderr.String())
	}

	pages, err := parseSidecar(sidecarFile.Name())
	if err != nil {
		return nil, err
	}
	return pages, nil
}

func parseSidecar(path string) ([]PageContent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open sidecar: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read sidecar: %w", err)
	}
	return parseSidecarBytes(data), nil
}

func parseSidecarBytes(data []byte) []PageContent {
	raw := strings.Split(string(data), "\f")
	var pages []PageContent
	pageNum := 1
	for _, chunk := range raw {
		text := strings.TrimSpace(normalizeNewlines(chunk))
		if text == "" {
			pageNum++
			continue
		}
		pages = append(pages, PageContent{
			Page:    pageNum,
			Content: text,
		})
		pageNum++
	}
	return pages
}

func normalizeNewlines(in string) string {
	return strings.ReplaceAll(in, "\r\n", "\n")
}

// SaveUploadedFile copies the provided reader to a temporary PDF file.
func SaveUploadedFile(r io.Reader) (string, func(), error) {
	tmpFile, err := os.CreateTemp("", "ocr-input-*.pdf")
	if err != nil {
		return "", nil, fmt.Errorf("create temp pdf: %w", err)
	}

	cleanup := func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}

	if _, err := io.Copy(tmpFile, r); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("write temp pdf: %w", err)
	}

	return tmpFile.Name(), cleanup, nil
}

// EnsureBinary checks whether the OCR binary is available on PATH.
func EnsureBinary(binary string) error {
	if binary == "" {
		binary = "ocrmypdf"
	}
	_, err := exec.LookPath(binary)
	if err != nil {
		return fmt.Errorf("ocrmypdf binary not found (%s): %w", binary, err)
	}
	return nil
}

// ResolveBinary returns the absolute binary path if available on PATH.
func ResolveBinary(binary string) (string, error) {
	if binary == "" {
		binary = "ocrmypdf"
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", err
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs, nil
	}
	return path, nil
}
