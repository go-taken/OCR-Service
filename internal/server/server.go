package server

import (
	"log"
	"os"

	"app/internal/ocr"
	"app/internal/server/handler"
	"app/internal/server/router"
	"app/internal/server/service"

	"github.com/gin-gonic/gin"
)

// Run starts the HTTP server.
func Run() error {
	if err := ocr.EnsureBinary(""); err != nil {
		return err
	}

	// Get configuration from environment
	apiKey := os.Getenv("API_KEY")
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Set Gin mode based on environment
	mode := os.Getenv("MODE")
	if mode == "prod" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Build dependency chain
	processor := ocr.NewProcessor()
	ocrService := service.NewOCRService(processor)
	ocrHandler := handler.NewOCRHandler(ocrService)

	// Setup router with all routes and middleware
	r := router.New(apiKey, ocrHandler)

	// Start server
	addr := ":" + port
	log.Printf("listening on %s", addr)
	return r.Run(addr)
}
