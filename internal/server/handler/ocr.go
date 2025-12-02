package handler

import (
	"context"
	"log"
	"mime/multipart"
	"net/http"

	"app/internal/ocr"

	"github.com/gin-gonic/gin"
)

// OCRService defines the behavior consumed by the handler.
type OCRService interface {
	Process(ctx context.Context, file multipart.File, header *multipart.FileHeader, lang string) ([]ocr.PageContent, error)
}

// OCRHandler manages OCR HTTP interactions.
type OCRHandler struct {
	service OCRService
}

// NewOCRHandler builds the handler.
func NewOCRHandler(svc OCRService) *OCRHandler {
	return &OCRHandler{service: svc}
}

// HandleOCR processes OCR requests for PDF files.
func (h *OCRHandler) HandleOCR(c *gin.Context) {
	// Parse multipart form (50MB limit)
	if err := c.Request.ParseMultipartForm(100 << 20); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "invalid multipart payload",
		})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "missing file",
		})
		return
	}
	defer file.Close()

	// Get language parameter (default: eng+chi_sim+ind)
	lang := c.Request.FormValue("lang")
	if lang == "" {
		lang = "eng+chi_sim+ind"
	}

	// Process the OCR request
	pages, err := h.service.Process(c.Request.Context(), file, header, lang)
	if err != nil {
		log.Printf("ocr error: %v", err)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
			"error": "ocr error",
		})
		return
	}

	c.JSON(http.StatusOK, pages)
}
