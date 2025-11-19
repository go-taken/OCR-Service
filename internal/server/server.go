package server

import (
	"log"
	"net/http"
	"os"
	"time"

	"app/internal/ocr"
	"app/internal/server/handler"
	"app/internal/server/router"
	"app/internal/server/service"
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

	// Build dependency chain
	processor := ocr.NewProcessor()
	ocrService := service.NewOCRService(processor)
	ocrHandler := handler.NewOCRHandler(ocrService)

	// Setup router with all routes and middleware
	r := router.New(apiKey, ocrHandler)

	// Configure server with 10 minute timeout
	addr := ":" + port
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Minute,
		WriteTimeout: 10 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	// Start server
	log.Printf("listening on %s", addr)
	return srv.ListenAndServe()
}
