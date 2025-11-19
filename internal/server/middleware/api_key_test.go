package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWithAPIKey_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	// Setup middleware and test handler
	r.Use(WithAPIKey("secret"))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusTeapot)
	})

	// Create request with valid API key
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "secret")
	c.Request = req

	r.ServeHTTP(w, req)

	if w.Code != http.StatusTeapot {
		t.Fatalf("expected chained handler code, got %d", w.Code)
	}
}

func TestWithAPIKey_MissingOrInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Test missing API key
	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	r.Use(WithAPIKey("secret"))
	r.GET("/test", func(c *gin.Context) {
		t.Fatal("should not reach handler")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing key, got %d", w.Code)
	}

	// Test wrong API key
	w = httptest.NewRecorder()
	c, r = gin.CreateTestContext(w)

	r.Use(WithAPIKey("secret"))
	r.GET("/test", func(c *gin.Context) {
		t.Fatal("should not reach handler")
	})

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("x-api-key", "wrong")
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong key, got %d", w.Code)
	}
}

func TestWithAPIKey_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, r := gin.CreateTestContext(w)

	// Empty API key means disabled
	r.Use(WithAPIKey(""))
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusAccepted)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request = req
	r.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected downstream status when disabled, got %d", w.Code)
	}
}
