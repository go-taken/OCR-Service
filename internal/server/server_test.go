package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"app/internal/ocr"
	"app/internal/server/handler"
	"app/internal/server/router"
	"app/internal/server/service"

	"github.com/gin-gonic/gin"
)

const samplePDFName = "pdf-example.pdf"
const sampleExpectedText = "content in page 1"

type testProcessor struct {
	pages []ocr.PageContent
	err   error
}

func (t *testProcessor) ExtractText(ctx context.Context, pdfPath string, opts ocr.Options) ([]ocr.PageContent, error) {
	if t.err != nil {
		return nil, t.err
	}
	return t.pages, nil
}

func TestServer_OCRFlowWithAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	proc := &testProcessor{pages: []ocr.PageContent{{Page: 1, Content: sampleExpectedText}}}
	svc := service.NewOCRService(proc)
	ocrHandler := handler.NewOCRHandler(svc)
	r := router.New("secret", ocrHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Missing key => 401
	req := newFileRequest(t, ts.URL+"/api/v1/ocr/pdf", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 without key, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Include key => 200
	req = newFileRequest(t, ts.URL+"/api/v1/ocr/pdf", map[string]string{"lang": "eng"})
	req.Header.Set("x-api-key", "secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	fmt.Println(string(bodyBytes))
	if !strings.Contains(string(bodyBytes), sampleExpectedText) {
		t.Fatalf("response missing expected text: %s", bodyBytes)
	}
}

func TestServer_OCRFlowWithoutAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	proc := &testProcessor{pages: []ocr.PageContent{{Page: 1, Content: sampleExpectedText}}}
	svc := service.NewOCRService(proc)
	ocrHandler := handler.NewOCRHandler(svc)
	r := router.New("", ocrHandler) // No API key

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Without API key requirement, request should succeed
	req := newFileRequest(t, ts.URL+"/api/v1/ocr/pdf", map[string]string{"lang": "eng"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", resp.StatusCode)
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(bodyBytes), sampleExpectedText) {
		t.Fatalf("response missing expected text: %s", bodyBytes)
	}
}

func newFileRequest(t *testing.T, url string, fields map[string]string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for k, v := range fields {
		if err := writer.WriteField(k, v); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	writer.CreateFormFile("file", filepath.Base(samplePDFName))
	writer.Close()

	req, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

