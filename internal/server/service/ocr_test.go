package service

import (
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"app/internal/ocr"
)

const expectedOCRText = "content in page 1"

type fakeProcessor struct {
	pages    []ocr.PageContent
	err      error
	lastPath string
	lastOpts ocr.Options
}

func (f *fakeProcessor) ExtractText(ctx context.Context, pdfPath string, opts ocr.Options) ([]ocr.PageContent, error) {
	f.lastPath = pdfPath
	f.lastOpts = opts
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

func TestOCRService_Process_Success(t *testing.T) {
	proc := &fakeProcessor{
		pages: []ocr.PageContent{{Page: 1, Content: expectedOCRText}},
	}
	svc := NewOCRService(proc)

	file, header := sampleUploadFile(t)

	pages, err := svc.Process(context.Background(), file, header, "eng")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pages) != 1 || pages[0].Content != expectedOCRText {
		t.Fatalf("unexpected pages: %+v", pages)
	}
	if proc.lastOpts.Language != "eng" {
		t.Fatalf("expected language to pass through, got %s", proc.lastOpts.Language)
	}
	if proc.lastPath == "" {
		t.Fatal("expected pdf path to be captured")
	}
	if _, err := os.Stat(proc.lastPath); !os.IsNotExist(err) {
		t.Fatalf("expected temp file cleanup, got err=%v", err)
	}
}

func TestOCRService_Process_PropagatesProcessorError(t *testing.T) {
	wantErr := errors.New("ocr failed")
	proc := &fakeProcessor{err: wantErr}
	svc := NewOCRService(proc)

	file, header := sampleUploadFile(t)

	_, err := svc.Process(context.Background(), file, header, "")
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}

func sampleUploadFile(t *testing.T) (*os.File, *multipart.FileHeader) {
	t.Helper()
	src, err := os.Open(samplePDFPath(t))
	if err != nil {
		t.Fatalf("open sample pdf: %v", err)
	}
	defer src.Close()

	tmp, err := os.CreateTemp("", "upload-*.pdf")
	if err != nil {
		t.Fatalf("create temp upload: %v", err)
	}
	if _, err := io.Copy(tmp, src); err != nil {
		t.Fatalf("copy sample pdf: %v", err)
	}
	if _, err := tmp.Seek(0, 0); err != nil {
		t.Fatalf("rewind temp pdf: %v", err)
	}
	t.Cleanup(func() {
		tmp.Close()
		os.Remove(tmp.Name())
	})

	header := &multipart.FileHeader{
		Filename: filepath.Base(samplePDFPath(t)),
		Size:     fileSize(tmp),
	}
	return tmp, header
}

func fileSize(f *os.File) int64 {
	info, err := f.Stat()
	if err != nil {
		return 0
	}
	return info.Size()
}

func samplePDFPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "pdf-example.pdf"))
}
