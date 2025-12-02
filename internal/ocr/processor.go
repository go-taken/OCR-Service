package ocr

import (
	"app/pkg"
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

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

const (
	// DefaultTextThreshold is the minimum character count to consider a page has text
	DefaultTextThreshold = 50
)

// PageContent represents OCR text for a single page.
type PageContent struct {
	Page    int    `json:"page"`
	Content string `json:"content"`
}

// Options controls the OCR command invocation.
type Options struct {
	Language        string
	TextThreshold   int  // Minimum characters to consider page has text (default: 50)
	ForceOCR        bool // Force OCR even if text exists
	RemoveWatermark bool // Remove watermark before processing (default: true)
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

// ExtractText runs smart OCR: splits PDF, removes watermarks, extracts existing text, and OCRs only pages without text.
func (p *Processor) ExtractText(ctx context.Context, pdfPath string, opts Options) ([]PageContent, error) {
	if pdfPath == "" {
		return nil, errors.New("pdf path is required")
	}

	// Set defaults
	if opts.TextThreshold <= 0 {
		opts.TextThreshold = 150
	}

	// Split PDF into individual pages
	pageFiles, tempDir, err := p.splitPDFPages(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("split pdf: %w", err)
	}
	defer os.RemoveAll(tempDir)

	var results []PageContent

	for i, pageFile := range pageFiles {
		pageNum := i + 1
		fmt.Println("pageFile", pageFile)

		// Remove watermark if enabled (default: true when not explicitly set)
		if opts.RemoveWatermark || (!opts.ForceOCR && opts.TextThreshold > 0) {
			if err := p.removeWatermark(pageFile); err != nil {
				// Log but continue - watermark removal is best effort
				_ = err
			}
		}

		var text string

		// Try to extract existing text first (unless ForceOCR is set)
		if !opts.ForceOCR {
			extractedText, err := p.extractTextFromPage(pageFile)
			if err == nil && p.hasSignificantText(extractedText, opts.TextThreshold) {
				text = strings.TrimSpace(extractedText)
			}
		}

		// If no significant text found, run OCR on this page
		if text == "" {
			ocrText, err := p.ocrSinglePage(ctx, pageFile, opts)
			if err != nil {
				return nil, fmt.Errorf("ocr page %d: %w", pageNum, err)
			}
			text = ocrText
		}

		// Only add pages with content
		if text != "" {
			results = append(results, PageContent{
				Page:    pageNum,
				Content: pkg.RemoveExtraSpaces(text),
			})
		}
	}

	return results, nil
}

// splitPDFPages splits a PDF into individual page files.
func (p *Processor) splitPDFPages(pdfPath string) ([]string, string, error) {
	// Create temp directory for split pages
	tempDir, err := os.MkdirTemp("", "ocr-pages-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}

	// Get page count
	pageCount, err := api.PageCountFile(pdfPath)
	if err != nil {
		os.RemoveAll(tempDir)
		return nil, "", fmt.Errorf("get page count: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	var pageFiles []string

	// Split each page
	for i := 1; i <= pageCount; i++ {
		outputPath := filepath.Join(tempDir, fmt.Sprintf("page_%04d.pdf", i))

		// Extract single page
		err := api.ExtractPagesFile(pdfPath, tempDir, []string{fmt.Sprintf("%d", i)}, conf)
		if err != nil {
			os.RemoveAll(tempDir)
			return nil, "", fmt.Errorf("extract page %d: %w", i, err)
		}

		// pdfcpu creates files with pattern: originalname_page_N.pdf
		// We need to find and rename it
		baseName := strings.TrimSuffix(filepath.Base(pdfPath), ".pdf")
		expectedName := filepath.Join(tempDir, fmt.Sprintf("%s_page_%d.pdf", baseName, i))

		if _, err := os.Stat(expectedName); err == nil {
			if err := os.Rename(expectedName, outputPath); err != nil {
				os.RemoveAll(tempDir)
				return nil, "", fmt.Errorf("rename page %d: %w", i, err)
			}
		} else {
			// Try alternate naming pattern
			altName := filepath.Join(tempDir, fmt.Sprintf("%s_%d.pdf", baseName, i))
			if _, err := os.Stat(altName); err == nil {
				if err := os.Rename(altName, outputPath); err != nil {
					os.RemoveAll(tempDir)
					return nil, "", fmt.Errorf("rename page %d: %w", i, err)
				}
			} else {
				os.RemoveAll(tempDir)
				return nil, "", fmt.Errorf("find extracted page %d file", i)
			}
		}

		pageFiles = append(pageFiles, outputPath)
	}

	return pageFiles, tempDir, nil
}

// removeWatermark attempts to remove watermarks from a PDF page.
func (p *Processor) removeWatermark(pagePath string) error {
	conf := model.NewDefaultConfiguration()

	// Create temp file for output
	tempFile, err := os.CreateTemp("", "ocr-nowm-*.pdf")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tempPath := tempFile.Name()
	tempFile.Close()

	// Try to remove watermarks
	if err := api.RemoveWatermarksFile(pagePath, tempPath, nil, conf); err != nil {
		os.Remove(tempPath)
		// Watermark removal failed, but this is not critical
		return err
	}

	// Replace original with watermark-removed version
	if err := os.Rename(tempPath, pagePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("replace original: %w", err)
	}

	return nil
}

// extractTextFromPage extracts text content from a PDF page using pdftotext (poppler-utils).
func (p *Processor) extractTextFromPage(pagePath string) (string, error) {
	// Use pdftotext from poppler-utils for reliable text extraction
	cmd := exec.Command("pdftotext", "-layout", pagePath, "-")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// pdftotext not available or failed, return empty string
		return "", nil
	}

	return normalizeNewlines(stdout.String()), nil
}

// hasSignificantText checks if the text has enough content to be considered valid.
func (p *Processor) hasSignificantText(text string, threshold int) bool {
	// Remove whitespace for character count
	cleaned := strings.ReplaceAll(text, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\t", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")

	return len(cleaned) >= threshold
}

// ocrSinglePage runs OCRmyPDF on a single page PDF.
func (p *Processor) ocrSinglePage(ctx context.Context, pagePath string, opts Options) (string, error) {
	binary := p.Binary
	if binary == "" {
		binary = "ocrmypdf"
	}
	timeout := p.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}

	// Create temp files
	sidecarFile, err := os.CreateTemp("", "ocr-sidecar-*.txt")
	if err != nil {
		return "", fmt.Errorf("create sidecar: %w", err)
	}
	defer os.Remove(sidecarFile.Name())
	defer sidecarFile.Close()

	outputPDF, err := os.CreateTemp("", "ocr-output-*.pdf")
	if err != nil {
		return "", fmt.Errorf("create temp output: %w", err)
	}
	defer os.Remove(outputPDF.Name())
	defer outputPDF.Close()

	// Build OCRmyPDF arguments
	args := []string{
		"--sidecar", sidecarFile.Name(),
		"--quiet",
		"--rotate-pages-threshold", "0.0",
		"--force-ocr",
	}
	if opts.Language != "" {
		args = append(args, "--language", opts.Language)
	}
	args = append(args, pagePath, outputPDF.Name())

	// Execute OCRmyPDF
	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ocrmypdf: %w - %s", err, stderr.String())
	}

	// Read sidecar output
	data, err := os.ReadFile(sidecarFile.Name())
	if err != nil {
		return "", fmt.Errorf("read sidecar: %w", err)
	}

	return strings.TrimSpace(normalizeNewlines(string(data))), nil
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
