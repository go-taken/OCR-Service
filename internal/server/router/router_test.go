package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakeOCRHandler struct {
	called bool
}

func (f *fakeOCRHandler) HandleOCR(c *gin.Context) {
	f.called = true
	c.Status(http.StatusAccepted)
}

func TestNew_Healthz(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := New("", &fakeOCRHandler{})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if body := w.Body.String(); body != "ok" {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestNew_OCRHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeHandler := &fakeOCRHandler{}
	router := New("", fakeHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/pdf", nil)
	router.ServeHTTP(w, req)

	if !fakeHandler.called {
		t.Fatal("expected ocr handler to be invoked")
	}
	if w.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: %d", w.Code)
	}
}

func TestNew_WithAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fakeHandler := &fakeOCRHandler{}
	router := New("secret-key", fakeHandler)

	// Test without API key - should fail
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ocr/pdf", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without API key, got %d", w.Code)
	}
	if fakeHandler.called {
		t.Fatal("handler should not be called without valid API key")
	}

	// Test with correct API key - should succeed
	fakeHandler.called = false
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/v1/ocr/pdf", nil)
	req.Header.Set("x-api-key", "secret-key")
	router.ServeHTTP(w, req)

	if !fakeHandler.called {
		t.Fatal("handler should be called with valid API key")
	}
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with valid API key, got %d", w.Code)
	}
}
