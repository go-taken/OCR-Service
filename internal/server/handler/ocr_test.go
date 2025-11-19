package handler

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"app/internal/ocr"

	"github.com/gin-gonic/gin"
)

const handlerExpectedText = "content in page 1"

type fakeService struct {
	pages []ocr.PageContent
	err   error
}

func (f *fakeService) Process(ctx context.Context, file multipart.File, header *multipart.FileHeader, lang string) ([]ocr.PageContent, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.pages, nil
}

func TestOCRHandler_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeService{pages: []ocr.PageContent{{Page: 1, Content: handlerExpectedText}}}
	handler := NewOCRHandler(svc)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/ocr", handler.HandleOCR)

	req := newMultipartRequest(t, map[string]string{"lang": "eng"})
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected json content type got %s", ct)
	}
	if body := w.Body.String(); !strings.Contains(body, handlerExpectedText) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestOCRHandler_MethodNotAllowed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOCRHandler(&fakeService{})

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	// Register only POST, try GET
	r.POST("/ocr", handler.HandleOCR)

	req := httptest.NewRequest(http.MethodGet, "/ocr", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	// Gin returns 404 for method not allowed on unregistered routes
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestOCRHandler_InvalidMultipart(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOCRHandler(&fakeService{})

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/ocr", handler.HandleOCR)

	req := httptest.NewRequest(http.MethodPost, "/ocr", bytes.NewBufferString("invalid"))
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestOCRHandler_MissingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOCRHandler(&fakeService{})

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.Close()

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/ocr", handler.HandleOCR)

	req := httptest.NewRequest(http.MethodPost, "/ocr", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Code)
	}
}

func TestOCRHandler_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewOCRHandler(&fakeService{err: errors.New("boom")})

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.POST("/ocr", handler.HandleOCR)

	req := newMultipartRequest(t, nil)
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502 got %d", w.Code)
	}
}

func newMultipartRequest(t *testing.T, fields map[string]string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}

	part, err := writer.CreateFormFile("file", filepath.Base(handlerSamplePDFPath(t)))
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	writeSamplePDF(t, part)
	writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/ocr", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func writeSamplePDF(t *testing.T, dst io.Writer) {
	t.Helper()
	data, err := os.ReadFile(handlerSamplePDFPath(t))
	if err != nil {
		t.Fatalf("read sample pdf: %v", err)
	}
	if _, err := dst.Write(data); err != nil {
		t.Fatalf("write sample pdf: %v", err)
	}
}

func handlerSamplePDFPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "pdf-example.pdf"))
}
