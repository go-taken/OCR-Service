# OCR Service

Go-based HTTP service that accepts PDF uploads, runs [OCRmyPDF](https://ocrmypdf.org/), and returns page-by-page text.

## Prerequisites

- Go 1.25.3+
- OCRmyPDF CLI available on `PATH`
  - Ubuntu: `sudo apt-get install ocrmypdf`
  - Docker alternative: `docker run --rm -v $PWD:/data ghcr.io/ocrmypdf/ocrmypdf ocrmypdf ...`

## Running locally

```bash
export PORT=8080
export API_KEY=supersecret
go run ./cmd/server
```

## API

`POST /api/v1/ocr/pdf`

Headers:

- `x-api-key`: must match the `API_KEY` environment variable. Omit if `API_KEY` is unset.

Multipart form fields:

- `file` (required): PDF file upload.
- `lang` (optional): language hint passed to OCRmyPDF.

Response:

```json
[
  {
    "page": 1,
    "content": "content in page 1"
  }
]
```

Health check: `GET /healthz`

## Testing

```bash
go test ./...
```
