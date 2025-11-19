# OCR Service API Documentation

This documentation provides details on how to interact with the OCR Service API.

## Base URL
The default base URL for local development is:
`http://localhost:8081`

## Authentication
The API is protected by an API Key. You must include the API Key in the request headers.

- **Header Name:** `x-api-key`
- **Value:** The API key configured in your environment variables (e.g., `API_KEY=supersecret`).

> **Note:** If the `API_KEY` environment variable is not set on the server, authentication is skipped.

## Endpoints

### 1. Extract Text from PDF
Upload a PDF file to extract text content from its pages using OCR.

- **Endpoint:** `/api/v1/ocr/pdf`
- **Method:** `POST`
- **Content-Type:** `multipart/form-data`

#### Request Parameters

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `file` | File | **Yes** | The PDF file to be processed. |
| `lang` | String | No | Language code(s) for OCR. Multiple languages can be joined by `+`. Default: `eng+chi_sim+ind`. |

#### Response Format
The response is a JSON array where each object represents a page in the PDF.

```json
[
  {
    "page": 1,
    "content": "Text extracted from the first page..."
  },
  {
    "page": 2,
    "content": "Text extracted from the second page..."
  }
]
```

#### Status Codes

| Code | Description |
|------|-------------|
| `200` | OK. The OCR process was successful. |
| `400` | Bad Request. Missing file or invalid multipart payload. |
| `401` | Unauthorized. Invalid or missing `x-api-key`. |
| `405` | Method Not Allowed. Only `POST` is supported. |
| `502` | Bad Gateway. An error occurred during the OCR processing (e.g., `ocrmypdf` failed). |

#### Example Request (cURL)

```bash
curl -X POST http://localhost:8081/api/v1/ocr/pdf \
  -H "x-api-key: supersecret" \
  -F "file=@/path/to/your/document.pdf" 
```

---

### 2. Health Check
Check the health status of the service.

- **Endpoint:** `/healthz`
- **Method:** `GET`

#### Response
Returns a plain text response.

- **Code:** `200 OK`
- **Body:** `ok`
