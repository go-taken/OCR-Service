# Multi-stage Dockerfile for OCR Service
# Stage 1: Build the Go application
FROM golang:1.25.3-alpine AS builder

WORKDIR /build

# Copy go module files
COPY go.mod go.sum ./

# Download dependencies (if any)
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Stage 2: Runtime environment with OCRmyPDF
FROM python:3.11-slim

# Install OCRmyPDF and all dependencies via apt
RUN apt-get update && apt-get install -y \
    ocrmypdf \
    tesseract-ocr \
    tesseract-ocr-eng \
    tesseract-ocr-chi-sim \
    tesseract-ocr-ind \
    poppler-utils \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Set working directory
WORKDIR /app

# Copy the compiled binary from builder
COPY --from=builder /build/server /app/server

# Expose the application port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080


# Run the application
ENTRYPOINT ["/app/server"]
