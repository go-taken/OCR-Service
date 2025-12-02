include .env


dev:
	sh -c 'set -a; . ./.env; set +a; go run cmd/server/ocr.go'