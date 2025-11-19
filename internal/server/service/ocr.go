package service

import (
	"context"
	"fmt"
	"mime/multipart"

	"app/internal/ocr"
)

// Processor defines the OCR dependency.
type Processor interface {
	ExtractText(ctx context.Context, pdfPath string, opts ocr.Options) ([]ocr.PageContent, error)
}

// OCRService orchestrates OCR processing.
type OCRService struct {
	processor Processor
}

// NewOCRService creates OCRService.
func NewOCRService(proc Processor) *OCRService {
	return &OCRService{processor: proc}
}

// Process persists the uploaded file and runs OCR.
func (s *OCRService) Process(ctx context.Context, file multipart.File, header *multipart.FileHeader, lang string) ([]ocr.PageContent, error) {
	tempPath, cleanup, err := ocr.SaveUploadedFile(file)
	if err != nil {
		return nil, fmt.Errorf("persist upload (%s): %w", header.Filename, err)
	}
	defer cleanup()

	pages, err := s.processor.ExtractText(ctx, tempPath, ocr.Options{Language: lang})
	if err != nil {
		return nil, err
	}
	return pages, nil
}
