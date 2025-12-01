package router

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

// OCRHandler defines the interface for the OCR handler.
type OCRHandler interface {
	HandleOCR(c *gin.Context)
}

// New wires up handlers to the Gin engine.
func New(apiKey string, ocrHandler OCRHandler) *gin.Engine {
	r := gin.Default()

	
	// Health check endpoint (no middleware)
	r.GET("/healthz", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	// Proxy to port 11434 (Ollama)
	ollamaURL, _ := url.Parse("http://localhost:11434")
	proxy := httputil.NewSingleHostReverseProxy(ollamaURL)

	r.NoRoute(func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		// OCR endpoints group with API key middleware
		ocr := v1.Group("/ocr")

		// Apply API key middleware if configured
		if apiKey != "" {
			ocr.Use(func(c *gin.Context) {
				if c.GetHeader("x-api-key") != apiKey {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": "unauthorized",
					})
					return
				}
				c.Next()
			})
		}

		ocr.POST("/pdf", ocrHandler.HandleOCR)
	}

	return r
}
